package gen

import (
	"strings"
	"testing"
)

func TestImplCppGenerator_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &ImplCppGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 4 {
		t.Fatalf("expected 4 output files, got %d", len(files))
	}

	// Verify filenames
	expectedNames := []string{
		"test_api_interface.h",
		"test_api_shim.cpp",
		"test_api_impl.h",
		"test_api_impl.cpp",
	}
	for i, expected := range expectedNames {
		if files[i].Path != expected {
			t.Errorf("file[%d]: expected %q, got %q", i, expected, files[i].Path)
		}
	}
}

func TestImplCppGenerator_InterfaceHeader(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &ImplCppGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Include guard
	if !strings.Contains(content, "#ifndef TEST_API_INTERFACE_H") {
		t.Error("missing include guard #ifndef")
	}
	if !strings.Contains(content, "#define TEST_API_INTERFACE_H") {
		t.Error("missing include guard #define")
	}

	// Standard includes
	if !strings.Contains(content, "#include <stdint.h>") {
		t.Error("missing stdint.h include")
	}
	if !strings.Contains(content, "#include <cstddef>") {
		t.Error("missing cstddef include")
	}
	if !strings.Contains(content, "#include <string_view>") {
		t.Error("missing string_view include")
	}
	if !strings.Contains(content, "#include <span>") {
		t.Error("missing span include")
	}

	// Abstract class declaration
	if !strings.Contains(content, "class TestApiInterface {") {
		t.Error("missing abstract class declaration")
	}

	// Virtual destructor
	if !strings.Contains(content, "virtual ~TestApiInterface() = default;") {
		t.Error("missing virtual destructor")
	}

	// Pure virtual methods
	if !strings.Contains(content, "virtual int32_t create_engine(void** out_result) = 0;") {
		t.Error("missing pure virtual create_engine method")
	}
	if !strings.Contains(content, "virtual void destroy_engine(void* engine) = 0;") {
		t.Error("missing pure virtual destroy_engine method")
	}

	// Factory function declaration
	if !strings.Contains(content, "TestApiInterface* create_test_api_instance();") {
		t.Error("missing factory function declaration")
	}

	// Closing guard
	if !strings.HasSuffix(strings.TrimSpace(content), "#endif") {
		t.Error("missing #endif at end")
	}
}

func TestImplCppGenerator_ShimFile(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &ImplCppGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[1].Content)

	// Includes
	if !strings.Contains(content, "#include \"test_api_interface.h\"") {
		t.Error("missing interface header include")
	}
	if !strings.Contains(content, "#include \"test_api.h\"") {
		t.Error("missing C header include")
	}

	// extern "C" block
	if !strings.Contains(content, "extern \"C\" {") {
		t.Error("missing extern \"C\" block")
	}

	// Create method: factory + reinterpret_cast
	if !strings.Contains(content, "create_test_api_instance()") {
		t.Error("missing factory call in create shim")
	}
	if !strings.Contains(content, "reinterpret_cast<engine_handle>(instance)") {
		t.Error("missing reinterpret_cast to handle in create shim")
	}

	// Destroy method: reinterpret_cast + delete
	if !strings.Contains(content, "reinterpret_cast<TestApiInterface*>(engine)") {
		t.Error("missing reinterpret_cast from handle in destroy shim")
	}
	if !strings.Contains(content, "delete instance") {
		t.Error("missing delete in destroy shim")
	}
}

func TestImplCppGenerator_ImplHeader(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &ImplCppGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[2].Content)

	// Include guard
	if !strings.Contains(content, "#ifndef TEST_API_IMPL_H") {
		t.Error("missing include guard")
	}

	// Includes interface header
	if !strings.Contains(content, "#include \"test_api_interface.h\"") {
		t.Error("missing interface header include")
	}

	// Inherits from interface
	if !strings.Contains(content, "class TestApiImpl : public TestApiInterface {") {
		t.Error("missing class declaration with inheritance")
	}

	// Constructor and destructor
	if !strings.Contains(content, "TestApiImpl();") {
		t.Error("missing constructor declaration")
	}
	if !strings.Contains(content, "~TestApiImpl() override;") {
		t.Error("missing destructor declaration")
	}

	// Override methods
	if !strings.Contains(content, "override;") {
		t.Error("missing override specifier on methods")
	}
}

func TestImplCppGenerator_ImplSource(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &ImplCppGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[3].Content)

	// Includes impl header
	if !strings.Contains(content, "#include \"test_api_impl.h\"") {
		t.Error("missing impl header include")
	}

	// TODO stubs
	if !strings.Contains(content, "// TODO: implement") {
		t.Error("missing TODO comments in stubs")
	}

	// Constructor / destructor
	if !strings.Contains(content, "TestApiImpl::TestApiImpl()") {
		t.Error("missing constructor definition")
	}
	if !strings.Contains(content, "TestApiImpl::~TestApiImpl()") {
		t.Error("missing destructor definition")
	}

	// Method stubs
	if !strings.Contains(content, "TestApiImpl::create_engine(") {
		t.Error("missing create_engine stub")
	}
	if !strings.Contains(content, "TestApiImpl::destroy_engine(") {
		t.Error("missing destroy_engine stub")
	}

	// Factory function
	if !strings.Contains(content, "create_test_api_instance()") {
		t.Error("missing factory function definition")
	}
	if !strings.Contains(content, "return new TestApiImpl()") {
		t.Error("missing new in factory function")
	}
}

func TestImplCppGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &ImplCppGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 4 {
		t.Fatalf("expected 4 output files, got %d", len(files))
	}

	// Verify filenames use correct API name
	expectedNames := []string{
		"example_app_engine_interface.h",
		"example_app_engine_shim.cpp",
		"example_app_engine_impl.h",
		"example_app_engine_impl.cpp",
	}
	for i, expected := range expectedNames {
		if files[i].Path != expected {
			t.Errorf("file[%d]: expected %q, got %q", i, expected, files[i].Path)
		}
	}

	ifaceContent := string(files[0].Content)
	shimContent := string(files[1].Content)

	// String param → std::string_view in interface
	if !strings.Contains(ifaceContent, "std::string_view path") {
		t.Error("missing std::string_view for string parameter in interface")
	}

	// Buffer param → std::span in interface
	if !strings.Contains(ifaceContent, "std::span<const uint8_t> data") {
		t.Error("missing std::span for buffer parameter in interface")
	}

	// Shim wraps string as string_view
	if !strings.Contains(shimContent, "std::string_view(path)") {
		t.Error("missing std::string_view wrapping in shim")
	}

	// Shim wraps buffer as span
	if !strings.Contains(shimContent, "std::span(data, data_len)") {
		t.Error("missing std::span wrapping in shim")
	}

	// Handle-based delegation in shim
	if !strings.Contains(shimContent, "reinterpret_cast<ExampleAppEngineInterface*>") {
		t.Error("missing reinterpret_cast delegation in shim for handle methods")
	}
}

func TestImplCppGenerator_Registration(t *testing.T) {
	gen, ok := Get("impl_cpp")
	if !ok {
		t.Fatal("impl_cpp generator not registered")
	}
	if gen.Name() != "impl_cpp" {
		t.Errorf("expected name impl_cpp, got %q", gen.Name())
	}
}
