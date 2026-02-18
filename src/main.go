package main

import (
	"os"

	"github.com/benn-herrera/xplatter/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
