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

	// minimal.yaml has only lifecycle (create_engine/destroy_engine) which are
	// both excluded from the interface. So we get: interface (empty body),
	// cgo, impl (no struct), types, go.mod = 5 files.
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

	// Verify ProjectFile flags — Go generated files go alongside user code
	projectFiles := map[string]bool{
		"test_api_interface.go": true,
		"test_api_cgo.go":      true,
		"test_api_impl.go":     true,
		"test_api_types.go":    true,
		"go.mod":               true,
	}
	for _, f := range files {
		if projectFiles[f.Path] && !f.ProjectFile {
			t.Errorf("%s should be ProjectFile", f.Path)
		}
	}
}

func TestGoImplGenerator_InterfaceFile_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Package declaration
	if !strings.Contains(content, "package main") {
		t.Error("missing package declaration")
	}

	// minimal.yaml lifecycle has only create_engine and destroy_engine,
	// both lifecycle methods — interface should be empty (no Lifecycle interface).
	if strings.Contains(content, "type Lifecycle interface {") {
		t.Error("Lifecycle interface should be excluded (all methods are lifecycle)")
	}
	if strings.Contains(content, "CreateEngine") {
		t.Error("CreateEngine should be excluded from interface (it's a create method)")
	}
	if strings.Contains(content, "DestroyEngine") {
		t.Error("DestroyEngine should be excluded from interface (it's a destroy method)")
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
	if !strings.Contains(content, "package main") {
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

	// Handle management helpers
	if !strings.Contains(content, "_handles    sync.Map") {
		t.Error("missing _handles sync.Map")
	}
	if !strings.Contains(content, "_nextHandle atomic.Uintptr") {
		t.Error("missing _nextHandle atomic.Uintptr")
	}
	if !strings.Contains(content, "func _allocHandle(") {
		t.Error("missing _allocHandle function")
	}
	if !strings.Contains(content, "func _freeHandle(") {
		t.Error("missing _freeHandle function")
	}

	// //export annotations
	if !strings.Contains(content, "//export test_api_lifecycle_create_engine") {
		t.Error("missing //export for create_engine")
	}
	if !strings.Contains(content, "//export test_api_lifecycle_destroy_engine") {
		t.Error("missing //export for destroy_engine")
	}

	// Create method delegates via handle map
	if !strings.Contains(content, "&EngineImpl{}") {
		t.Error("create_engine should instantiate EngineImpl")
	}
	if !strings.Contains(content, "_allocHandle(impl)") {
		t.Error("create_engine should call _allocHandle")
	}

	// Destroy method frees handle
	if !strings.Contains(content, "_freeHandle(uintptr(unsafe.Pointer(engine)))") {
		t.Error("destroy_engine should call _freeHandle")
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

func TestGoImplGenerator_ImplFile_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[2].Content)

	// Package declaration
	if !strings.Contains(content, "package main") {
		t.Error("missing package declaration")
	}

	// minimal.yaml has only lifecycle methods — no impl struct should be generated
	if strings.Contains(content, "LifecycleImpl") {
		t.Error("LifecycleImpl should not exist (all lifecycle methods excluded)")
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

	// Lifecycle interface is excluded (only has create/destroy)
	if strings.Contains(ifaceContent, "type Lifecycle interface {") {
		t.Error("Lifecycle interface should be excluded (all methods are lifecycle)")
	}

	// Renderer interface: create_renderer has handle param so not auto-create,
	// destroy_renderer is auto-destroy (excluded), begin_frame/end_frame are regular
	if !strings.Contains(ifaceContent, "type Renderer interface {") {
		t.Error("missing Renderer interface")
	}
	// begin_frame: handle param excluded, error return → error
	if !strings.Contains(ifaceContent, "BeginFrame() error") {
		t.Error("missing or incorrect BeginFrame signature (handle excluded, fallible void)")
	}

	// Texture interface: load_texture_from_path, load_texture_from_buffer (both have handle param),
	// destroy_texture is excluded
	if !strings.Contains(ifaceContent, "type Texture interface {") {
		t.Error("missing Texture interface")
	}

	// Input interface: push_touch_events has handle param
	if !strings.Contains(ifaceContent, "type Input interface {") {
		t.Error("missing Input interface")
	}

	// Events interface
	if !strings.Contains(ifaceContent, "type Events interface {") {
		t.Error("missing Events interface")
	}

	// String parameter in interface (handle excluded)
	if !strings.Contains(ifaceContent, "path string") {
		t.Error("missing string parameter in interface")
	}

	// Buffer parameter in interface
	if !strings.Contains(ifaceContent, "data []uint8") {
		t.Error("missing buffer parameter in interface")
	}

	// String parameter handling in cgo
	if !strings.Contains(cgoContent, "C.GoString(") {
		t.Error("missing C.GoString conversion for string params")
	}

	// Buffer parameter handling in cgo
	if !strings.Contains(cgoContent, "unsafe.Slice(") {
		t.Error("missing unsafe.Slice for buffer params")
	}

	// Impl structs — Lifecycle excluded (all lifecycle), others present
	if strings.Contains(implContent, "type LifecycleImpl struct{}") {
		t.Error("LifecycleImpl should not exist (all lifecycle methods excluded)")
	}
	for _, name := range []string{"RendererImpl", "TextureImpl", "InputImpl", "EventsImpl"} {
		decl := "type " + name + " struct{}"
		if !strings.Contains(implContent, decl) {
			t.Errorf("missing impl struct: %s", decl)
		}
	}

	// Cgo shim auto-implements create_engine and destroy_engine
	if !strings.Contains(cgoContent, "&EngineImpl{}") {
		t.Error("create_engine should instantiate EngineImpl in cgo shim")
	}
	if !strings.Contains(cgoContent, "_freeHandle(uintptr(unsafe.Pointer(engine)))") {
		t.Error("destroy_engine should call _freeHandle in cgo shim")
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

	// begin_frame: fallible (error) + no return, handle excluded → error
	if !strings.Contains(ifaceContent, "BeginFrame() error") {
		t.Error("missing or incorrect BeginFrame signature (fallible, no return, handle excluded)")
	}
}

func TestGoImplGenerator_TypesFile(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	typesContent := string(files[3].Content)

	// Enum constants
	if !strings.Contains(typesContent, "CommonErrorCode") {
		t.Error("missing CommonErrorCode enum constants")
	}

	// minimal.yaml has no FlatBuffer return types, so no Go structs
	if strings.Contains(typesContent, "type ") {
		t.Error("should not have struct definitions for minimal.yaml (no FlatBuffer return types)")
	}
}

func TestGoImplGenerator_CgoCreateHandleTypedef(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	cgoContent := string(files[1].Content)

	// Verify create_engine uses the correct handle typedef
	if !strings.Contains(cgoContent, "C.engine_handle") {
		t.Error("create_engine should use C.engine_handle typedef")
	}
}
