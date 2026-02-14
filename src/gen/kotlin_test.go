package gen

import (
	"strings"
	"testing"
)

func TestKotlinGenerator_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 output files, got %d", len(files))
	}

	// Check filenames
	if files[0].Path != "TestApi.kt" {
		t.Errorf("expected Kotlin file TestApi.kt, got %q", files[0].Path)
	}
	if files[1].Path != "test_api_jni.c" {
		t.Errorf("expected JNI file test_api_jni.c, got %q", files[1].Path)
	}
}

func TestKotlinGenerator_KotlinPackage(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	kt := string(files[0].Content)
	if !strings.Contains(kt, "package test.api") {
		t.Error("missing package declaration")
	}
}

func TestKotlinGenerator_HandleClass(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	kt := string(files[0].Content)

	if !strings.Contains(kt, "class Engine internal constructor(internal val handle: Long) : AutoCloseable") {
		t.Error("missing Engine handle wrapper class")
	}

	if !strings.Contains(kt, "override fun close()") {
		t.Error("missing AutoCloseable close() method")
	}
}

func TestKotlinGenerator_ErrorException(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	kt := string(files[0].Content)

	if !strings.Contains(kt, "class CommonErrorCodeException") {
		t.Error("missing error exception class")
	}
	if !strings.Contains(kt, ": Exception(") {
		t.Error("exception class should extend Exception")
	}
}

func TestKotlinGenerator_NativeObject(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	kt := string(files[0].Content)

	if !strings.Contains(kt, "object TestApi") {
		t.Error("missing TestApi singleton object")
	}
	if !strings.Contains(kt, "System.loadLibrary(\"test_api\")") {
		t.Error("missing System.loadLibrary call")
	}
}

func TestKotlinGenerator_NativeMethodDeclarations(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	kt := string(files[0].Content)

	// create_engine: fallible + returns handle → LongArray
	if !strings.Contains(kt, "external fun nativeLifecycleCreateEngine(): LongArray") {
		t.Error("missing nativeLifecycleCreateEngine declaration")
	}

	// destroy_engine: infallible + void → Unit
	if !strings.Contains(kt, "external fun nativeLifecycleDestroyEngine(engine: Long): Unit") {
		t.Error("missing nativeLifecycleDestroyEngine declaration")
	}
}

func TestKotlinGenerator_FactoryMethod(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	kt := string(files[0].Content)

	// create_engine is a factory method (no handle first param)
	if !strings.Contains(kt, "fun createEngine(): Engine") {
		t.Error("missing createEngine factory method")
	}

	// Should throw on error
	if !strings.Contains(kt, "throw CommonErrorCodeException") {
		t.Error("missing error exception throw in factory method")
	}
}

func TestKotlinGenerator_CloseCallsDestroy(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	kt := string(files[0].Content)

	// close() should call the destroy native method
	if !strings.Contains(kt, "TestApi.nativeLifecycleDestroyEngine(handle)") {
		t.Error("close() should call nativeLifecycleDestroyEngine")
	}
}

func TestKotlinGenerator_JNIHeader(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	jni := string(files[1].Content)

	if !strings.Contains(jni, "#include <jni.h>") {
		t.Error("missing jni.h include")
	}
	if !strings.Contains(jni, "#include \"test_api.h\"") {
		t.Error("missing API C header include")
	}
}

func TestKotlinGenerator_JNIFunctions(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	jni := string(files[1].Content)

	// JNI function for create_engine
	if !strings.Contains(jni, "Java_test_api_TestApi_nativeLifecycleCreateEngine") {
		t.Error("missing JNI create_engine function")
	}

	// JNI function for destroy_engine
	if !strings.Contains(jni, "Java_test_api_TestApi_nativeLifecycleDestroyEngine") {
		t.Error("missing JNI destroy_engine function")
	}

	// Should call the C ABI function
	if !strings.Contains(jni, "test_api_lifecycle_create_engine") {
		t.Error("JNI should call C ABI function test_api_lifecycle_create_engine")
	}
	if !strings.Contains(jni, "test_api_lifecycle_destroy_engine") {
		t.Error("JNI should call C ABI function test_api_lifecycle_destroy_engine")
	}
}

func TestKotlinGenerator_JNIFallibleReturn(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	jni := string(files[1].Content)

	// create_engine is fallible with return → returns jlongArray with [error, result]
	if !strings.Contains(jni, "jlongArray") {
		t.Error("fallible+return JNI function should return jlongArray")
	}
	if !strings.Contains(jni, "NewLongArray") {
		t.Error("should create jlongArray for fallible+return")
	}
	if !strings.Contains(jni, "out_result") {
		t.Error("should use out_result for fallible+return")
	}
}

func TestKotlinGenerator_JNIThrowHelper(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	jni := string(files[1].Content)

	if !strings.Contains(jni, "throw_exception") {
		t.Error("missing throw_exception helper in JNI file")
	}
}

func TestKotlinGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &KotlinGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 output files, got %d", len(files))
	}

	if files[0].Path != "ExampleAppEngine.kt" {
		t.Errorf("expected ExampleAppEngine.kt, got %q", files[0].Path)
	}
	if files[1].Path != "example_app_engine_jni.c" {
		t.Errorf("expected example_app_engine_jni.c, got %q", files[1].Path)
	}

	kt := string(files[0].Content)
	jni := string(files[1].Content)

	// Handle classes
	for _, handle := range []string{"Engine", "Renderer", "Scene", "Texture"} {
		if !strings.Contains(kt, "class "+handle+" internal constructor") {
			t.Errorf("missing handle class %s", handle)
		}
	}

	// String parameter method
	if !strings.Contains(kt, "path: String") {
		t.Error("missing String parameter for load_texture_from_path")
	}

	// Buffer parameter
	if !strings.Contains(kt, "data: ByteArray") {
		t.Error("missing ByteArray parameter for load_texture_from_buffer")
	}

	// JNI string marshalling
	if !strings.Contains(jni, "GetStringUTFChars") {
		t.Error("missing GetStringUTFChars in JNI for string params")
	}
	if !strings.Contains(jni, "ReleaseStringUTFChars") {
		t.Error("missing ReleaseStringUTFChars in JNI for string params")
	}

	// Instance method on Renderer
	if !strings.Contains(kt, "fun beginFrame()") {
		t.Error("missing beginFrame instance method on Renderer")
	}

	// Input interface method on Engine
	if !strings.Contains(kt, "fun pushTouchEvents(") {
		t.Error("missing pushTouchEvents instance method")
	}
}

func TestKotlinGenerator_Registration(t *testing.T) {
	gen, ok := Get("kotlin")
	if !ok {
		t.Fatal("kotlin generator not registered")
	}
	if gen.Name() != "kotlin" {
		t.Errorf("expected name 'kotlin', got %q", gen.Name())
	}
}
