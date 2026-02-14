package gen

import (
	"strings"
	"testing"
)

func TestSwiftGenerator_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(files))
	}

	if files[0].Path != "TestApi.swift" {
		t.Errorf("expected filename TestApi.swift, got %q", files[0].Path)
	}

	content := string(files[0].Content)

	// Should have Foundation import
	if !strings.Contains(content, "import Foundation") {
		t.Error("missing import Foundation")
	}

	// Should have error enum
	if !strings.Contains(content, "public enum CommonErrorCode: Int32, Error {") {
		t.Error("missing CommonErrorCode error enum")
	}

	// Should have Engine class
	if !strings.Contains(content, "public final class Engine {") {
		t.Error("missing Engine class declaration")
	}

	// Should have handle property
	if !strings.Contains(content, "let handle: OpaquePointer") {
		t.Error("missing handle property")
	}

	// Should have deinit calling destroy
	if !strings.Contains(content, "deinit {") {
		t.Error("missing deinit")
	}
	if !strings.Contains(content, "test_api_lifecycle_destroy_engine(handle)") {
		t.Error("deinit should call test_api_lifecycle_destroy_engine")
	}

	// Should have factory method for create_engine
	if !strings.Contains(content, "public static func createEngine() throws -> Engine {") {
		t.Error("missing createEngine factory method")
	}

	// Factory method should call C function
	if !strings.Contains(content, "test_api_lifecycle_create_engine(&result)") {
		t.Error("createEngine should call test_api_lifecycle_create_engine with out_result")
	}

	// Should have error handling in factory
	if !strings.Contains(content, "throw CommonErrorCode") {
		t.Error("missing error throw in factory method")
	}
}

func TestSwiftGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(files))
	}

	if files[0].Path != "ExampleAppEngine.swift" {
		t.Errorf("expected filename ExampleAppEngine.swift, got %q", files[0].Path)
	}

	content := string(files[0].Content)

	// Handle classes
	expectedClasses := []string{
		"public final class Engine {",
		"public final class Renderer {",
		"public final class Texture {",
	}
	for _, c := range expectedClasses {
		if !strings.Contains(content, c) {
			t.Errorf("missing class declaration: %s", c)
		}
	}

	// Factory methods
	if !strings.Contains(content, "public static func createEngine() throws -> Engine {") {
		t.Error("missing createEngine factory method")
	}
	if !strings.Contains(content, "public static func createRenderer(") {
		t.Error("missing createRenderer factory method")
	}

	// Instance methods on Renderer
	if !strings.Contains(content, "public func beginFrame() throws {") {
		t.Error("missing beginFrame instance method")
	}
	if !strings.Contains(content, "public func endFrame() throws {") {
		t.Error("missing endFrame instance method")
	}

	// Deinit methods
	if !strings.Contains(content, "example_app_engine_lifecycle_destroy_engine(handle)") {
		t.Error("missing Engine deinit calling example_app_engine_lifecycle_destroy_engine")
	}
	if !strings.Contains(content, "example_app_engine_renderer_destroy_renderer(handle)") {
		t.Error("missing Renderer deinit calling example_app_engine_renderer_destroy_renderer")
	}
	if !strings.Contains(content, "example_app_engine_texture_destroy_texture(handle)") {
		t.Error("missing Texture deinit calling example_app_engine_texture_destroy_texture")
	}
}

func TestSwiftGenerator_StringBridging(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// load_texture_from_path takes a string param — should use withCString
	if !strings.Contains(content, "path: String") {
		t.Error("missing String parameter type for path")
	}
	if !strings.Contains(content, "path.withCString { pathPtr in") {
		t.Error("missing withCString bridging for path parameter")
	}
	// The C call should use pathPtr
	if !strings.Contains(content, "pathPtr") {
		t.Error("missing pathPtr usage in C call")
	}
}

func TestSwiftGenerator_BufferBridging(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// load_texture_from_buffer takes a buffer<uint8> param
	if !strings.Contains(content, "data: Data") {
		t.Error("missing Data parameter type for buffer")
	}
	if !strings.Contains(content, "data.withUnsafeBytes { dataPtr in") {
		t.Error("missing withUnsafeBytes bridging for buffer parameter")
	}
}

func TestSwiftGenerator_FlatBufferParams(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// create_renderer takes a FlatBuffer config param with transfer: ref
	if !strings.Contains(content, "config: UnsafePointer<Rendering_RendererConfig>") {
		t.Error("missing FlatBuffer ref parameter type for config")
	}

	// push_touch_events takes FlatBuffer events param with transfer: ref
	if !strings.Contains(content, "events: UnsafePointer<Input_TouchEventBatch>") {
		t.Error("missing FlatBuffer ref parameter type for events")
	}

	// poll_events takes FlatBuffer events param with transfer: ref_mut
	if !strings.Contains(content, "events: UnsafeMutablePointer<Common_EventQueue>") {
		t.Error("missing FlatBuffer ref_mut parameter type for events")
	}
}

func TestSwiftGenerator_ErrorHandling(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Error enum should conform to Error protocol
	if !strings.Contains(content, "Int32, Error {") {
		t.Error("error enum should conform to Error protocol")
	}

	// Error enum should have cases
	expectedCases := []string{
		"case ok = 0",
		"case invalidArgument = 1",
		"case outOfMemory = 2",
		"case notFound = 3",
		"case internalError = 4",
	}
	for _, c := range expectedCases {
		if !strings.Contains(content, c) {
			t.Errorf("missing error case: %s", c)
		}
	}
}

func TestSwiftGenerator_HandleInit(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Internal init from raw handle
	if !strings.Contains(content, "init(handle: OpaquePointer)") {
		t.Error("missing internal init(handle:)")
	}
	if !strings.Contains(content, "self.handle = handle") {
		t.Error("missing self.handle assignment in init")
	}
}

func TestSwiftGenerator_GeneratedComment(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "Generated by xplattergy") {
		t.Error("missing generated comment")
	}
}

func TestSwiftGenerator_FallibleVoidMethod(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// begin_frame is fallible with no return — should be throws with no return type
	if !strings.Contains(content, "public func beginFrame() throws {") {
		t.Error("missing fallible void method signature for beginFrame")
	}

	// Should call the C function and check error
	if !strings.Contains(content, "example_app_engine_renderer_begin_frame(handle)") {
		t.Error("missing C function call for beginFrame")
	}
}

func TestSwiftGenerator_InfallibleVoidMethod(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &SwiftGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// destroy_engine is infallible void — handled by deinit, not exposed as public method
	// But the deinit should exist and call the C function
	if !strings.Contains(content, "deinit {") {
		t.Error("missing deinit for Engine")
	}
}

func TestSwiftGenerator_Name(t *testing.T) {
	gen := &SwiftGenerator{}
	if gen.Name() != "swift" {
		t.Errorf("expected name 'swift', got %q", gen.Name())
	}
}

func TestSwiftGenerator_Registry(t *testing.T) {
	g, ok := Get("swift")
	if !ok {
		t.Fatal("swift generator not found in registry")
	}
	if g.Name() != "swift" {
		t.Errorf("expected name 'swift', got %q", g.Name())
	}
}
