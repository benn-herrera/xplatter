package cmd

import (
	"fmt"
	"os"

	"github.com/benn-herrera/xplatter/loader"
	"github.com/spf13/cobra"
)

var dumpSchemaOutput string

var dumpSchemaCmd = &cobra.Command{
	Use:   "dump_schema",
	Short: "Print the built-in API definition JSON Schema",
	Long:  "Prints the JSON Schema used to validate xplatter API definition YAML files. Use -o to write to a file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		schema := loader.SchemaJSON()
		if dumpSchemaOutput == "" {
			fmt.Println(schema)
			return nil
		}
		if err := os.WriteFile(dumpSchemaOutput, []byte(schema+"\n"), 0644); err != nil {
			return fmt.Errorf("writing schema to %s: %w", dumpSchemaOutput, err)
		}
		if !quiet {
			fmt.Fprintf(os.Stderr, "Schema written to %s\n", dumpSchemaOutput)
		}
		return nil
	},
}

func init() {
	dumpSchemaCmd.Flags().StringVarP(&dumpSchemaOutput, "output", "o", "", "Write schema to file instead of stdout")
	rootCmd.AddCommand(dumpSchemaCmd)
}
