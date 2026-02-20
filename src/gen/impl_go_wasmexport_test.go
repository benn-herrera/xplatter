package gen

import (
	"strings"
	"testing"
)

func TestGoWASMImplGenerator_Basic(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(files))
	}

	if files[0].Path != "test_api_wasm.go" {
		t.Errorf("expected test_api_wasm.go, got %q", files[0].Path)
	}

	// No longer a scaffold — fully generated
	if files[0].Scaffold {
		t.Error("WASM file should not be scaffold (fully generated)")
	}

	// Generated Go files go to generated/ (not ProjectFile)
	if files[0].ProjectFile {
		t.Error("WASM file should not be ProjectFile (goes to generated/)")
	}
}

func TestGoWASMImplGenerator_BuildTagAndPackage(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "//go:build wasip1") {
		t.Error("missing //go:build wasip1 build tag")
	}
	if !strings.Contains(content, "package main") {
		t.Error("missing package main declaration")
	}
}

func TestGoWASMImplGenerator_MallocFreeExports(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "//go:wasmexport malloc") {
		t.Error("missing //go:wasmexport malloc")
	}
	if !strings.Contains(content, "//go:wasmexport free") {
		t.Error("missing //go:wasmexport free")
	}
	if !strings.Contains(content, "var _wasmAllocs sync.Map") {
		t.Error("missing _wasmAllocs sync.Map")
	}
}

func TestGoWASMImplGenerator_HandleManagement(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "func _allocHandle(") {
		t.Error("missing _allocHandle")
	}
	if !strings.Contains(content, "func _freeHandle(") {
		t.Error("missing _freeHandle")
	}
	if !strings.Contains(content, "_wasmHandles sync.Map") {
		t.Error("missing _wasmHandles sync.Map")
	}
	if !strings.Contains(content, "_nextHandle  atomic.Uintptr") {
		t.Error("missing _nextHandle atomic.Uintptr")
	}
}

func TestGoWASMImplGenerator_CStringHelper(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "func _cstring(ptr uintptr) string") {
		t.Error("missing _cstring helper")
	}
	if !strings.Contains(content, "unsafe.Slice(") {
		t.Error("_cstring missing unsafe.Slice call")
	}
}

func TestGoWASMImplGenerator_PlatformServiceImports(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	expectedImports := []string{
		"//go:wasmimport env test_api_log_sink",
		"//go:wasmimport env test_api_resource_count",
		"//go:wasmimport env test_api_resource_name",
		"//go:wasmimport env test_api_resource_exists",
		"//go:wasmimport env test_api_resource_size",
		"//go:wasmimport env test_api_resource_read",
	}
	for _, imp := range expectedImports {
		if !strings.Contains(content, imp) {
			t.Errorf("missing platform import: %q", imp)
		}
	}
}

func TestGoWASMImplGenerator_InterfaceExports(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "//go:wasmexport test_api_lifecycle_create_engine") {
		t.Error("missing //go:wasmexport for create_engine")
	}
	if !strings.Contains(content, "//go:wasmexport test_api_lifecycle_destroy_engine") {
		t.Error("missing //go:wasmexport for destroy_engine")
	}
}

func TestGoWASMImplGenerator_CreateDelegation(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Create method should instantiate EngineImpl
	if !strings.Contains(content, "&EngineImpl{}") {
		t.Error("create_engine should instantiate EngineImpl")
	}
	if !strings.Contains(content, "_allocHandle(impl)") {
		t.Error("create_engine should call _allocHandle")
	}
	// Write handle to out_result in WASM32 format
	if !strings.Contains(content, "*(*uint32)(unsafe.Pointer(out_result))") {
		t.Error("create_engine should write handle to out_result via unsafe.Pointer")
	}
}

func TestGoWASMImplGenerator_DestroyDelegation(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// destroy_engine auto-implemented with _freeHandle
	if !strings.Contains(content, "_freeHandle(uintptr(engine))") {
		t.Error("destroy_engine not auto-implemented with _freeHandle(uintptr(engine))")
	}
}

func TestGoWASMImplGenerator_FallibleNoReturnReturnsInt32(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// create_engine is fallible with handle return → (out_result uintptr) int32
	if !strings.Contains(content, "out_result uintptr) int32") {
		t.Error("fallible-with-return method missing int32 return type and out_result param")
	}
}

func TestGoWASMImplGenerator_Registry(t *testing.T) {
	g, ok := Get("impl_go_wasm")
	if !ok {
		t.Fatal("impl_go_wasm generator not found in registry")
	}
	if g.Name() != "impl_go_wasm" {
		t.Errorf("expected name impl_go_wasm, got %q", g.Name())
	}
}

func TestGoWASMImplGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Create methods auto-implemented
	if !strings.Contains(content, "&EngineImpl{}") {
		t.Error("create_engine should instantiate EngineImpl")
	}

	// Destroy auto-implemented for lifecycle interface (has constructors:)
	if !strings.Contains(content, "_freeHandle(uintptr(engine))") {
		t.Error("destroy_engine not auto-implemented")
	}
	// renderer and texture use old-style create methods (with handle params) so no auto-destructor

	// String param → uintptr
	if !strings.Contains(content, "path uintptr") {
		t.Error("string parameter not mapped to uintptr")
	}

	// Buffer param → uintptr + uint32
	if !strings.Contains(content, "data uintptr") {
		t.Error("buffer parameter not mapped to uintptr")
	}
	if !strings.Contains(content, "data_len uint32") {
		t.Error("buffer length not mapped to uint32")
	}

	// Platform service imports use the correct api name
	if !strings.Contains(content, "//go:wasmimport env example_app_engine_log_sink") {
		t.Error("platform import uses wrong api name")
	}

	// Regular methods delegate to interface
	if !strings.Contains(content, "_cstring(") {
		t.Error("missing _cstring call for string parameter conversion")
	}
}

func TestGoWASMImplGenerator_FallibleVoidStub(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// begin_frame: fallible + no return → returns int32
	if !strings.Contains(content, "func example_app_engine_renderer_begin_frame(renderer uintptr) int32") {
		t.Error("begin_frame missing correct signature (fallible void → int32)")
	}
}

func TestGoWASMImplGenerator_StringCacheHelpers(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoWASMImplGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "_wasmStrCache sync.Map") {
		t.Error("missing _wasmStrCache sync.Map")
	}
	if !strings.Contains(content, "func _wasmCacheStrings(") {
		t.Error("missing _wasmCacheStrings function")
	}
}
