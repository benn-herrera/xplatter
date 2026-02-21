package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/benn-herrera/xplatter/loader"
	"github.com/benn-herrera/xplatter/resolver"
	"github.com/benn-herrera/xplatter/validate"
	"github.com/spf13/cobra"
)

var (
	valFlatc string
)

var validateCmd = &cobra.Command{
	Use:   "validate [api-definition.yaml]",
	Short: "Check API definition and FlatBuffers schemas without generating",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidate,
}

func init() {
	validateCmd.Flags().StringVarP(&valFlatc, "flatc", "f", "", "Path to FlatBuffers compiler")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	apiDefPath := args[0]

	if !quiet {
		fmt.Printf("Validating %s\n", apiDefPath)
	}

	// Load and schema-validate the API definition
	def, err := loader.LoadAPIDefinition(apiDefPath)
	if err != nil {
		return fmt.Errorf("loading API definition: %w", err)
	}

	if verbose {
		fmt.Printf("  API: %s v%s (%s)\n", def.API.Name, def.API.Version, def.API.ImplLang)
		fmt.Printf("  FlatBuffers schemas: %d\n", len(def.FlatBuffers))
		fmt.Printf("  Handles: %d\n", len(def.Handles))
		fmt.Printf("  Interfaces: %d\n", len(def.Interfaces))
	}

	// Resolve FlatBuffers types — search YAML dir first, then exe-sibling schemas dir
	baseDir := filepath.Dir(apiDefPath)
	searchDirs := schemaSearchDirs(baseDir)
	resolvedTypes, err := resolver.ParseFBSFiles(searchDirs, def.FlatBuffers)
	if err != nil {
		return fmt.Errorf("parsing FlatBuffers schemas: %w", err)
	}

	if verbose {
		fmt.Printf("  Resolved types: %d\n", len(resolvedTypes))
	}

	// Run semantic validation
	result := validate.Validate(def, resolvedTypes)
	if !result.IsValid() {
		return fmt.Errorf("semantic validation failed:\n%s", result.Error())
	}

	if !quiet {
		fmt.Println("Validation passed.")
	}
	return nil
}
