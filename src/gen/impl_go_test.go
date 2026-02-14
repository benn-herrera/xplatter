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

	if len(files) != 3 {
		t.Fatalf("expected 3 output files, got %d", len(files))
	}

	// Verify filenames
	expectedNames := []string{
		"test_api_interface.go",
		"test_api_cgo.go",
		"test_api_impl.go",
	}
	for i, name := range expectedNames {
		if files[i].Path != name {
			t.Errorf("file[%d]: expected %q, got %q", i, name, files[i].Path)
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

	// cgo header include
	if !strings.Contains(content, `#include "test_api.h"`) {
		t.Error("missing cgo header include")
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

func TestGoImplGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &GoImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 output files, got %d", len(files))
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
