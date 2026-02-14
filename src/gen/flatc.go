package gen

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// FlatcConfig holds configuration for running the flatc compiler.
type FlatcConfig struct {
	FlatcPath string   // resolved flatc binary path
	FBSFiles  []string // absolute paths to .fbs files
	OutputDir string   // base output directory
	Targets   []string // effective target list
	ImplLang  string   // impl_lang value
	DryRun    bool
	Verbose   bool
	Quiet     bool
}

// flatcLang pairs a flatc flag with its output subdirectory.
type flatcLang struct {
	Flag   string // e.g., "--kotlin"
	Subdir string // e.g., "flatbuffers/kotlin"
}

// FlatcLangsForTarget returns the flatc languages needed for a target platform.
func FlatcLangsForTarget(target string) []flatcLang {
	switch target {
	case "android":
		return []flatcLang{{"--kotlin", "flatbuffers/kotlin"}}
	case "ios", "macos":
		return []flatcLang{{"--swift", "flatbuffers/swift"}}
	case "web":
		return []flatcLang{{"--ts", "flatbuffers/ts"}}
	case "windows", "linux":
		return nil
	default:
		return nil
	}
}

// FlatcLangForImplLang returns the flatc language for an implementation language.
// Returns zero value if no flatc invocation is needed.
func FlatcLangForImplLang(implLang string) (flatcLang, bool) {
	switch implLang {
	case "cpp":
		return flatcLang{"--cpp", "flatbuffers/cpp"}, true
	case "rust":
		return flatcLang{"--rust", "flatbuffers/rust"}, true
	case "go":
		return flatcLang{"--go", "flatbuffers/go"}, true
	case "c":
		return flatcLang{}, false
	default:
		return flatcLang{}, false
	}
}

// collectFlatcLangs gathers the deduplicated set of flatc languages from targets and impl_lang.
func collectFlatcLangs(targets []string, implLang string) []flatcLang {
	seen := map[string]bool{}
	var langs []flatcLang

	for _, target := range targets {
		for _, lang := range FlatcLangsForTarget(target) {
			if !seen[lang.Flag] {
				seen[lang.Flag] = true
				langs = append(langs, lang)
			}
		}
	}

	if lang, ok := FlatcLangForImplLang(implLang); ok {
		if !seen[lang.Flag] {
			langs = append(langs, lang)
		}
	}

	return langs
}

// RunFlatc invokes flatc once per required language, writing output into
// <outputDir>/flatbuffers/<subdir>/. Returns the number of flatc invocations run.
func RunFlatc(cfg *FlatcConfig) (int, error) {
	langs := collectFlatcLangs(cfg.Targets, cfg.ImplLang)
	if len(langs) == 0 {
		return 0, nil
	}

	for _, lang := range langs {
		outDir := filepath.Join(cfg.OutputDir, lang.Subdir)

		args := []string{lang.Flag, "-o", outDir}
		args = append(args, cfg.FBSFiles...)

		if cfg.DryRun {
			fmt.Printf("  Would run: %s %s\n", cfg.FlatcPath, strings.Join(args, " "))
			continue
		}

		if cfg.Verbose {
			fmt.Printf("  Running: %s %s\n", cfg.FlatcPath, strings.Join(args, " "))
		}

		cmd := exec.Command(cfg.FlatcPath, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return 0, fmt.Errorf("flatc %s failed: %w\n%s", lang.Flag, err, string(output))
		}

		if !cfg.Quiet && len(output) > 0 {
			fmt.Print(string(output))
		}
	}

	if cfg.DryRun {
		return 0, nil
	}
	return len(langs), nil
}
