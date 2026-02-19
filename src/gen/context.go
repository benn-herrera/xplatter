package gen

import (
	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

// Context holds everything a generator needs to produce output.
type Context struct {
	API           *model.APIDefinition
	ResolvedTypes resolver.ResolvedTypes
	OutputDir     string
	APIDefPath    string // Path to the API definition YAML (for Makefile codegen step)
	Version       string // xplatter version (e.g. "v0.1.1-6-g27008c1")
	Verbose       bool
	DryRun        bool
}

// NewContext creates a new generation context.
func NewContext(api *model.APIDefinition, resolvedTypes resolver.ResolvedTypes, outputDir string, apiDefPath string) *Context {
	return &Context{
		API:           api,
		ResolvedTypes: resolvedTypes,
		OutputDir:     outputDir,
		APIDefPath:    apiDefPath,
	}
}
