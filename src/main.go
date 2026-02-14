package main

import (
	"os"

	"github.com/benn-herrera/xplattergy/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
