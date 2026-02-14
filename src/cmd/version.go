package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time:
//
//	go build -ldflags "-X github.com/benn-herrera/xplattergy/cmd.Version=1.0.0"
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("xplattergy %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
