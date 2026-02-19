package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/benn-herrera/xplatter/gen"
	"github.com/benn-herrera/xplatter/loader"
	"github.com/benn-herrera/xplatter/resolver"
	"github.com/benn-herrera/xplatter/validate"
)

var (
	genOutput    string
	genFlatc     string
	genImplLang  string
	genTargets   []string
	genDryRun    bool
	genClean     bool
	genSkipFlatc bool
)

var generateCmd = &cobra.Command{
	Use:   "generate [api-definition.yaml]",
	Short: "Generate C ABI header, platform bindings, and impl scaffolding",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&genOutput, "output", "o", "./generated", "Output directory")
	generateCmd.Flags().StringVarP(&genFlatc, "flatc", "f", "", "Path to FlatBuffers compiler")
	generateCmd.Flags().StringVar(&genImplLang, "impl-lang", "", "Override impl_lang from API definition")
	generateCmd.Flags().StringSliceVar(&genTargets, "targets", nil, "Override targets (comma-separated)")
	generateCmd.Flags().BoolVar(&genDryRun, "dry-run", false, "Show what would be generated without writing")
	generateCmd.Flags().BoolVar(&genClean, "clean", false, "Remove previously generated files first")
	generateCmd.Flags().BoolVar(&genSkipFlatc, "skip-flatc", false, "Skip flatc invocation even if flatc is available")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	apiDefPath := args[0]

	if !quiet {
		fmt.Printf("Generating from %s\n", apiDefPath)
	}

	// Load and schema-validate
	def, err := loader.LoadAPIDefinition(apiDefPath)
	if err != nil {
		return fmt.Errorf("loading API definition: %w", err)
	}

	// Apply CLI overrides
	if genImplLang != "" {
		def.API.ImplLang = genImplLang
	}
	if len(genTargets) > 0 {
		def.API.Targets = genTargets
	}

	// Resolve FlatBuffers types — search YAML dir first, then exe-sibling schemas dir
	baseDir := filepath.Dir(apiDefPath)
	searchDirs := schemaSearchDirs(baseDir)
	resolvedTypes, err := resolver.ParseFBSFiles(searchDirs, def.FlatBuffers)
	if err != nil {
		return fmt.Errorf("parsing FlatBuffers schemas: %w", err)
	}

	// Semantic validation
	result := validate.Validate(def, resolvedTypes)
	if !result.IsValid() {
		return fmt.Errorf("validation failed:\n%s", result.Error())
	}

	// Clean output directory if requested
	if genClean {
		if !quiet {
			fmt.Printf("Cleaning %s\n", genOutput)
		}
		if !genDryRun {
			os.RemoveAll(genOutput)
		}
	}

	// Run flatc for FlatBuffers codegen
	var flatcCount int
	if !genSkipFlatc && len(def.FlatBuffers) > 0 {
		flatcPath, err := resolver.ResolveFlatc(genFlatc)
		if err != nil {
			return fmt.Errorf("flatc is required but not found: %w\n\nProvide flatc via --flatc flag, XPLATTER_FLATC_PATH env var, or ensure it is in PATH.\nUse --skip-flatc to skip FlatBuffers codegen (generated bindings will be incomplete).", err)
		}

		// Resolve absolute paths for .fbs files using same search dirs
		fbsFiles := make([]string, len(def.FlatBuffers))
		for i, p := range def.FlatBuffers {
			resolved, err := resolver.ResolveFBSPath(p, searchDirs)
			if err != nil {
				return fmt.Errorf("resolving %s for flatc: %w", p, err)
			}
			fbsFiles[i] = resolved
		}

		flatcCount, err = gen.RunFlatc(&gen.FlatcConfig{
			FlatcPath: flatcPath,
			FBSFiles:  fbsFiles,
			OutputDir: genOutput,
			Targets:   def.EffectiveTargets(),
			ImplLang:  def.API.ImplLang,
			DryRun:    genDryRun,
			Verbose:   verbose,
			Quiet:     quiet,
		})
		if err != nil {
			return fmt.Errorf("flatc: %w", err)
		}
	}

	// Create generation context
	ctx := gen.NewContext(def, resolvedTypes, genOutput, apiDefPath)
	ctx.Verbose = verbose
	ctx.DryRun = genDryRun

	// Determine which generators to run
	generatorNames := []string{"cheader"} // Always generate C header

	for _, target := range def.EffectiveTargets() {
		for _, name := range gen.GeneratorsForTarget(target) {
			generatorNames = appendUnique(generatorNames, name)
		}
	}

	for _, name := range gen.GeneratorsForImplLang(def.API.ImplLang) {
		generatorNames = appendUnique(generatorNames, name)
	}

	for _, name := range gen.GeneratorsForImplLangAndTargets(def.API.ImplLang, def.EffectiveTargets()) {
		generatorNames = appendUnique(generatorNames, name)
	}

	// Run generators and collect output
	var allFiles []*gen.OutputFile
	for _, name := range generatorNames {
		g, ok := gen.Get(name)
		if !ok {
			if verbose {
				fmt.Printf("  Skipping unavailable generator: %s\n", name)
			}
			continue
		}

		if verbose {
			fmt.Printf("  Running generator: %s\n", g.Name())
		}

		files, err := g.Generate(ctx)
		if err != nil {
			return fmt.Errorf("generator %s failed: %w", name, err)
		}
		allFiles = append(allFiles, files...)
	}

	// Write output files
	var written, skipped int
	for _, f := range allFiles {
		base := genOutput
		if f.ProjectFile {
			base = filepath.Dir(genOutput)
		}
		outPath := filepath.Join(base, f.Path)

		// Scaffold files are only written when they don't already exist.
		if f.Scaffold {
			if _, err := os.Stat(outPath); err == nil {
				skipped++
				if verbose {
					fmt.Printf("  Scaffold exists, skipped: %s\n", outPath)
				}
				continue
			}
		}

		if genDryRun {
			fmt.Printf("  Would write: %s\n", outPath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", outPath, err)
		}
		if err := os.WriteFile(outPath, f.Content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}

		written++
		if verbose {
			fmt.Printf("  Wrote: %s\n", outPath)
		}
	}

	if !quiet {
		skippedMsg := ""
		if skipped > 0 {
			skippedMsg = fmt.Sprintf(", %d scaffold file(s) preserved", skipped)
		}
		if flatcCount > 0 {
			fmt.Printf("Generated %d files in %s (flatc ran %d invocation(s)%s)\n", written, genOutput, flatcCount, skippedMsg)
		} else {
			fmt.Printf("Generated %d files in %s%s\n", written, genOutput, skippedMsg)
		}
	}
	return nil
}

// schemaSearchDirs returns directories to search for .fbs files:
// 1. The YAML file's directory (user schemas)
// 2. The directory containing the running executable (system schemas)
func schemaSearchDirs(yamlBaseDir string) []string {
	dirs := []string{yamlBaseDir}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	return dirs
}

func appendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}
