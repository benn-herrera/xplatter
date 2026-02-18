package cmd

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "xplatter",
	Short: "Cross-platform API binding code generator",
	Long:  "xplatter generates C ABI headers, platform bindings, and implementation scaffolding from a single YAML API definition.",
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all output except errors")
}

func Execute() error {
	return rootCmd.Execute()
}
