package gen

import (
	"strings"
	"testing"
)

func TestGoImplGenerator_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 5 {
		t.Fatalf("expected 5 output files, got %d", len(files))
	}

	// Verify filenames
	expectedNames := []string{
		"test_api_interface.go",
		"test_api_cgo.go",
		"test_api_impl.go",
		"test_api_types.go",
		"go.mod",
	}
	for i, name := range expectedNames {
		if files[i].Path != name {
			t.Errorf("file[%d]: expected %q, got %q", i, name, files[i].Path)
		}
	}

	// Verify scaffold flags
	scaffoldFiles := map[string]bool{
		"test_api_impl.go": true,
		"go.mod":           true,
	}
	for _, f := range files {
		if scaffoldFiles[f.Path] && !f.Scaffold {
			t.Errorf("%s should be scaffold", f.Path)
		}
		if !scaffoldFiles[f.Path] && f.Scaffold {
			t.Errorf("%s should not be scaffold", f.Path)
		}
	}
}

func TestGoImplGenerator_InterfaceFile(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Package declaration
	if !strings.Contains(content, "package testapi") {
		t.Error("missing package declaration")
	}

	// Interface definition
	if !strings.Contains(content, "type Lifecycle interface {") {
		t.Error("missing Lifecycle interface definition")
	}

	// Method signatures — CreateEngine is fallible with handle return
	if !strings.Contains(content, "CreateEngine() (uintptr, error)") {
		t.Error("missing or incorrect CreateEngine signature")
	}

	// DestroyEngine is infallible void with handle param
	if !strings.Contains(content, "DestroyEngine(engine uintptr)") {
		t.Error("missing or incorrect DestroyEngine signature")
	}
}

func TestGoImplGenerator_CgoFile(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[1].Content)

	// Package declaration
	if !strings.Contains(content, "package testapi") {
		t.Error("missing package declaration")
	}

	// cgo preamble should NOT include the header (causes prototype conflicts with //export)
	if strings.Contains(content, `#include "test_api.h"`) {
		t.Error("cgo preamble must not #include the C header (conflicts with //export prototypes)")
	}

	// cgo preamble should have local C type definitions
	if !strings.Contains(content, "#include <stdint.h>") {
		t.Error("missing #include <stdint.h> in cgo preamble")
	}
	if !strings.Contains(content, "#include <stdbool.h>") {
		t.Error("missing #include <stdbool.h> in cgo preamble")
	}

	// Handle typedef in cgo preamble
	if !strings.Contains(content, "typedef struct engine_s* engine_handle;") {
		t.Error("missing engine_handle typedef in cgo preamble")
	}

	// import "C"
	if !strings.Contains(content, `import "C"`) {
		t.Error(`missing import "C"`)
	}

	// sync.Map for handles
	if !strings.Contains(content, "var handles sync.Map") {
		t.Error("missing handles sync.Map")
	}

	// //export annotations
	if !strings.Contains(content, "//export test_api_lifecycle_create_engine") {
		t.Error("missing //export for create_engine")
	}
	if !strings.Contains(content, "//export test_api_lifecycle_destroy_engine") {
		t.Error("missing //export for destroy_engine")
	}

	// unsafe import
	if !strings.Contains(content, `"unsafe"`) {
		t.Error("missing unsafe import")
	}

	// sync import
	if !strings.Contains(content, `"sync"`) {
		t.Error("missing sync import")
	}
}

func TestGoImplGenerator_ImplFile(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[2].Content)

	// Package declaration
	if !strings.Contains(content, "package testapi") {
		t.Error("missing package declaration")
	}

	// Struct definition
	if !strings.Contains(content, "type LifecycleImpl struct{}") {
		t.Error("missing LifecycleImpl struct")
	}

	// Interface satisfaction check
	if !strings.Contains(content, "var _ Lifecycle = (*LifecycleImpl)(nil)") {
		t.Error("missing interface satisfaction check")
	}

	// TODO comments
	if !strings.Contains(content, "// TODO: implement") {
		t.Error("missing TODO comment in stub")
	}

	// Stub method for CreateEngine (fallible with return)
	if !strings.Contains(content, "func (s *LifecycleImpl) CreateEngine() (uintptr, error)") {
		t.Error("missing or incorrect CreateEngine stub method")
	}

	// Stub method for DestroyEngine (infallible void)
	if !strings.Contains(content, "func (s *LifecycleImpl) DestroyEngine(engine uintptr)") {
		t.Error("missing or incorrect DestroyEngine stub method")
	}

	// Return values
	if !strings.Contains(content, "return 0, nil") {
		t.Error("missing zero return for fallible handle method")
	}
}

func TestGoImplGenerator_GoMod(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// go.mod is the last file
	gomod := string(files[len(files)-1].Content)

	if !strings.Contains(gomod, "module test-api") {
		t.Error("missing module name in go.mod")
	}
	if !strings.Contains(gomod, "go 1.24") {
		t.Error("missing go version in go.mod")
	}
}

func TestGoImplGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 5 {
		t.Fatalf("expected 5 output files, got %d", len(files))
	}

	ifaceContent := string(files[0].Content)
	cgoContent := string(files[1].Content)
	implContent := string(files[2].Content)

	// Verify multiple interfaces are generated
	for _, name := range []string{"Lifecycle", "Renderer", "Texture", "Input", "Events"} {
		iface := "type " + name + " interface {"
		if !strings.Contains(ifaceContent, iface) {
			t.Errorf("missing interface: %s", iface)
		}
	}

	// String parameter handling in cgo
	if !strings.Contains(cgoContent, "C.GoString(") {
		t.Error("missing C.GoString conversion for string params")
	}

	// Buffer parameter handling in cgo
	if !strings.Contains(cgoContent, "unsafe.Slice(") {
		t.Error("missing unsafe.Slice for buffer params")
	}

	// String param in Go interface
	if !strings.Contains(ifaceContent, "path string") {
		t.Error("missing string parameter in interface")
	}

	// Buffer param in Go interface
	if !strings.Contains(ifaceContent, "data []uint8") {
		t.Error("missing buffer parameter in interface")
	}

	// Multiple impl structs
	for _, name := range []string{"LifecycleImpl", "RendererImpl", "TextureImpl", "InputImpl", "EventsImpl"} {
		decl := "type " + name + " struct{}"
		if !strings.Contains(implContent, decl) {
			t.Errorf("missing impl struct: %s", decl)
		}
	}
}

func TestGoImplGenerator_Registry(t *testing.T) {
	gen, ok := Get("impl_go")
	if !ok {
		t.Fatal("impl_go generator not found in registry")
	}
	if gen.Name() != "impl_go" {
		t.Errorf("expected name impl_go, got %q", gen.Name())
	}
}

func TestGoImplGenerator_FallibleNoReturn(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	ifaceContent := string(files[0].Content)

	// begin_frame: fallible (error) + no return → error
	if !strings.Contains(ifaceContent, "BeginFrame(renderer uintptr) error") {
		t.Error("missing or incorrect BeginFrame signature (fallible, no return)")
	}
}
