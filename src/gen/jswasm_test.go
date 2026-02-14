package gen

import (
	"strings"
	"testing"
)

func TestJSWASMGenerator_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(files))
	}

	if files[0].Path != "test_api.js" {
		t.Errorf("expected filename test_api.js, got %q", files[0].Path)
	}
}

func TestJSWASMGenerator_ESModuleExports(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Should export the loader function
	if !strings.Contains(content, "export async function loadTestApi(") {
		t.Error("missing exported async loader function")
	}

	// Should export handle classes
	if !strings.Contains(content, "export { Engine }") {
		t.Error("missing Engine class export")
	}

	// Should export the loader by name
	if !strings.Contains(content, "export { loadTestApi }") {
		t.Error("missing named export of loader")
	}
}

func TestJSWASMGenerator_HandleClass(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Handle class declaration
	if !strings.Contains(content, "class Engine {") {
		t.Error("missing Engine class declaration")
	}

	// Private pointer field
	if !strings.Contains(content, "#ptr") {
		t.Error("missing private #ptr field")
	}

	// dispose method
	if !strings.Contains(content, "dispose()") {
		t.Error("missing dispose method")
	}

	// close method (alias)
	if !strings.Contains(content, "close()") {
		t.Error("missing close method")
	}

	// Symbol.dispose support
	if !strings.Contains(content, "[Symbol.dispose]()") {
		t.Error("missing Symbol.dispose method")
	}
}

func TestJSWASMGenerator_WASMCalls(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Should call WASM exports with C ABI function names
	if !strings.Contains(content, "_wasm.exports.test_api_lifecycle_create_engine") {
		t.Error("missing WASM call for create_engine")
	}
	if !strings.Contains(content, "_wasm.exports.test_api_lifecycle_destroy_engine") {
		t.Error("missing WASM call for destroy_engine")
	}
}

func TestJSWASMGenerator_ErrorHandling(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// create_engine is fallible — should check return code and throw
	if !strings.Contains(content, "throw new Error(") {
		t.Error("missing error throw for fallible method")
	}
	if !strings.Contains(content, "_rc !== 0") {
		t.Error("missing return code check")
	}
}

func TestJSWASMGenerator_MemoryHelpers(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "function _malloc(size)") {
		t.Error("missing _malloc helper")
	}
	if !strings.Contains(content, "function _free(ptr)") {
		t.Error("missing _free helper")
	}
	if !strings.Contains(content, "TextEncoder") {
		t.Error("missing TextEncoder usage")
	}
	if !strings.Contains(content, "TextDecoder") {
		t.Error("missing TextDecoder usage")
	}
}

func TestJSWASMGenerator_StringMarshalling(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "function _encodeString(str)") {
		t.Error("missing _encodeString helper")
	}
	if !strings.Contains(content, "function _decodeString(ptr)") {
		t.Error("missing _decodeString helper")
	}
}

func TestJSWASMGenerator_BufferMarshalling(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "function _copyBufferToWasm(typedArray)") {
		t.Error("missing _copyBufferToWasm helper")
	}
	if !strings.Contains(content, "function _readBufferFromWasm(") {
		t.Error("missing _readBufferFromWasm helper")
	}
}

func TestJSWASMGenerator_PlatformServiceImports(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	expectedImports := []string{
		"test_api_log_sink",
		"test_api_resource_count",
		"test_api_resource_name",
		"test_api_resource_exists",
		"test_api_resource_size",
		"test_api_resource_read",
	}
	for _, imp := range expectedImports {
		if !strings.Contains(content, imp) {
			t.Errorf("missing platform service import: %s", imp)
		}
	}
}

func TestJSWASMGenerator_HandleReturnWrapping(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// create_engine returns handle:Engine — should wrap in Engine class
	if !strings.Contains(content, "new Engine(") {
		t.Error("missing Engine handle construction from return value")
	}
}

func TestJSWASMGenerator_HandleParamPassthrough(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// destroy_engine takes handle:Engine — should pass engine._ptr
	if !strings.Contains(content, "engine._ptr") {
		t.Error("missing handle._ptr access for handle parameter")
	}
}

func TestJSWASMGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(files))
	}

	if files[0].Path != "example_app_engine.js" {
		t.Errorf("expected filename example_app_engine.js, got %q", files[0].Path)
	}

	content := string(files[0].Content)

	// All handle classes should exist
	expectedClasses := []string{
		"class Engine {",
		"class Renderer {",
		"class Scene {",
		"class Texture {",
	}
	for _, cls := range expectedClasses {
		if !strings.Contains(content, cls) {
			t.Errorf("missing handle class: %s", cls)
		}
	}

	// String parameter marshalling (load_texture_from_path has a string param)
	if !strings.Contains(content, "_encodeString(path)") {
		t.Error("missing string marshalling for path parameter")
	}

	// Buffer parameter marshalling (load_texture_from_buffer has buffer<uint8>)
	if !strings.Contains(content, "_copyBufferToWasm(data)") {
		t.Error("missing buffer marshalling for data parameter")
	}

	// Multiple interfaces should be present in loader return
	if !strings.Contains(content, "_createLifecycle()") {
		t.Error("missing lifecycle interface factory in loader")
	}
	if !strings.Contains(content, "_createRenderer()") {
		t.Error("missing renderer interface factory in loader")
	}
	if !strings.Contains(content, "_createTexture()") {
		t.Error("missing texture interface factory in loader")
	}
	if !strings.Contains(content, "_createInput()") {
		t.Error("missing input interface factory in loader")
	}
	if !strings.Contains(content, "_createEvents()") {
		t.Error("missing events interface factory in loader")
	}
}

func TestJSWASMGenerator_FullCleanupOnError(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Methods with string/buffer params should have try/finally with _free
	if !strings.Contains(content, "} finally {") {
		t.Error("missing try/finally cleanup block")
	}
	if !strings.Contains(content, "_free(") {
		t.Error("missing _free call for cleanup")
	}
}

func TestJSWASMGenerator_WASMLoaderAcceptsMultipleSources(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Loader should handle WebAssembly.Module, Response, string URL, and ArrayBuffer
	if !strings.Contains(content, "WebAssembly.Module") {
		t.Error("missing WebAssembly.Module support in loader")
	}
	if !strings.Contains(content, "WebAssembly.instantiateStreaming") {
		t.Error("missing WebAssembly.instantiateStreaming support in loader")
	}
	if !strings.Contains(content, "ArrayBuffer") {
		t.Error("missing ArrayBuffer support in loader")
	}
}

func TestJSWASMGenerator_GeneratedHeader(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &JSWASMGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "Generated by xplattergy") {
		t.Error("missing generated-by header comment")
	}
	if !strings.Contains(content, "Do not edit") {
		t.Error("missing do-not-edit warning")
	}
}
