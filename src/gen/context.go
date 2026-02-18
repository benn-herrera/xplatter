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
	Verbose       bool
	DryRun        bool
}

// NewContext creates a new generation context.
func NewContext(api *model.APIDefinition, resolvedTypes resolver.ResolvedTypes, outputDir string) *Context {
	return &Context{
		API:           api,
		ResolvedTypes: resolvedTypes,
		OutputDir:     outputDir,
	}
}
