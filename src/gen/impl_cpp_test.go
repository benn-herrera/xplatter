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

	if len(files) != 5 {
		t.Fatalf("expected 5 output files, got %d", len(files))
	}

	// Verify filenames
	expectedNames := []string{
		"test_api_interface.h",
		"test_api_shim.cpp",
		"test_api_impl.h",
		"test_api_impl.cpp",
		"CMakeLists.txt",
	}
	for i, expected := range expectedNames {
		if files[i].Path != expected {
			t.Errorf("file[%d]: expected %q, got %q", i, expected, files[i].Path)
		}
	}

	// Verify scaffold and ProjectFile flags
	type fileExpect struct {
		scaffold    bool
		projectFile bool
	}
	expectedFlags := map[string]fileExpect{
		"test_api_interface.h": {false, false},
		"test_api_shim.cpp":   {false, false},
		"test_api_impl.h":     {true, true},
		"test_api_impl.cpp":   {true, true},
		"CMakeLists.txt":      {true, true},
	}
	for _, f := range files {
		expect := expectedFlags[f.Path]
		if expect.scaffold && !f.Scaffold {
			t.Errorf("%s should be scaffold", f.Path)
		}
		if !expect.scaffold && f.Scaffold {
			t.Errorf("%s should not be scaffold", f.Path)
		}
		if expect.projectFile && !f.ProjectFile {
			t.Errorf("%s should be ProjectFile", f.Path)
		}
		if !expect.projectFile && f.ProjectFile {
			t.Errorf("%s should not be ProjectFile", f.Path)
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

	// Lifecycle methods must NOT appear as pure virtual methods in the interface
	if strings.Contains(content, "virtual int32_t create_engine(") {
		t.Error("create_engine (lifecycle) must not appear as a virtual method in the interface")
	}
	if strings.Contains(content, "virtual void destroy_engine(") {
		t.Error("destroy_engine (lifecycle) must not appear as a virtual method in the interface")
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

	// Lifecycle methods must NOT appear as override methods in the impl header
	if strings.Contains(content, "create_engine(") {
		t.Error("create_engine (lifecycle) must not appear as override method in impl header")
	}
	if strings.Contains(content, "destroy_engine(") {
		t.Error("destroy_engine (lifecycle) must not appear as override method in impl header")
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

	// Lifecycle methods must NOT appear as method stubs
	if strings.Contains(content, "TestApiImpl::create_engine(") {
		t.Error("create_engine (lifecycle) must not appear as a method stub in impl source")
	}
	if strings.Contains(content, "TestApiImpl::destroy_engine(") {
		t.Error("destroy_engine (lifecycle) must not appear as a method stub in impl source")
	}

	// Factory function
	if !strings.Contains(content, "create_test_api_instance()") {
		t.Error("missing factory function definition")
	}
	if !strings.Contains(content, "return new TestApiImpl()") {
		t.Error("missing new in factory function")
	}
}

func TestImplCppGenerator_CMakeLists(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &ImplCppGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// CMakeLists.txt is the last file
	cmake := string(files[len(files)-1].Content)

	if !strings.Contains(cmake, "cmake_minimum_required(VERSION 3.15)") {
		t.Error("missing cmake_minimum_required in CMakeLists.txt")
	}
	if !strings.Contains(cmake, "project(test-api") {
		t.Error("missing project name in CMakeLists.txt")
	}
	if !strings.Contains(cmake, "test_api_shim.cpp") {
		t.Error("missing shim source in CMakeLists.txt")
	}
	if !strings.Contains(cmake, "test_api_impl.cpp") {
		t.Error("missing impl source in CMakeLists.txt")
	}
}

func TestImplCppGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &ImplCppGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 5 {
		t.Fatalf("expected 5 output files, got %d", len(files))
	}

	// Verify filenames use correct API name
	expectedNames := []string{
		"example_app_engine_interface.h",
		"example_app_engine_shim.cpp",
		"example_app_engine_impl.h",
		"example_app_engine_impl.cpp",
		"CMakeLists.txt",
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
