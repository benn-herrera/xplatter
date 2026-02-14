package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initName     string
	initImplLang string
	initOutput   string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new project with starter API definition and FBS files",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVarP(&initName, "name", "n", "my_api", "API name")
	initCmd.Flags().StringVar(&initImplLang, "impl-lang", "cpp", "Implementation language (cpp, rust, go, c)")
	initCmd.Flags().StringVarP(&initOutput, "output", "o", ".", "Output directory")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if !quiet {
		fmt.Printf("Initializing project %s in %s\n", initName, initOutput)
	}

	// Create directories
	schemasDir := filepath.Join(initOutput, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		return fmt.Errorf("creating schemas directory: %w", err)
	}

	// Write starter API definition
	apiDefPath := filepath.Join(initOutput, initName+".yaml")
	apiDef := fmt.Sprintf(`api:
  name: %s
  version: 0.1.0
  description: "TODO: describe your API"
  impl_lang: %s

flatbuffers:
  - schemas/types.fbs

handles:
  - name: Instance
    description: "Main instance handle"

interfaces:
  - name: lifecycle
    methods:
      - name: create_instance
        returns:
          type: handle:Instance
        error: Common.ErrorCode
      - name: destroy_instance
        parameters:
          - name: instance
            type: handle:Instance
`, initName, initImplLang)

	if err := os.WriteFile(apiDefPath, []byte(apiDef), 0644); err != nil {
		return fmt.Errorf("writing API definition: %w", err)
	}

	// Write starter FBS schema
	fbsPath := filepath.Join(schemasDir, "types.fbs")
	fbs := `namespace Common;

enum ErrorCode : int32 {
    Ok = 0,
    InvalidArgument = 1,
    OutOfMemory = 2,
    NotFound = 3,
    InternalError = 4
}
`
	if err := os.WriteFile(fbsPath, []byte(fbs), 0644); err != nil {
		return fmt.Errorf("writing FBS schema: %w", err)
	}

	if !quiet {
		fmt.Printf("Created:\n")
		fmt.Printf("  %s\n", apiDefPath)
		fmt.Printf("  %s\n", fbsPath)
		fmt.Printf("\nNext: xplattergy validate %s\n", apiDefPath)
	}
	return nil
}
