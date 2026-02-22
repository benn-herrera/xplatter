package gen

import (
	"strings"
	"testing"
)

func TestImplCGenerator_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &ImplCGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 output files, got %d", len(files))
	}

	f := files[0]
	if f.Path != "test_api_impl.c" {
		t.Errorf("expected path %q, got %q", "test_api_impl.c", f.Path)
	}
	if !f.Scaffold {
		t.Error("impl .c file should be scaffold")
	}
	if !f.ProjectFile {
		t.Error("impl .c file should be a project file")
	}

	cmake := files[1]
	if cmake.Path != "CMakeLists.txt" {
		t.Errorf("expected path %q, got %q", "CMakeLists.txt", cmake.Path)
	}
	if !cmake.Scaffold {
		t.Error("CMakeLists.txt should be scaffold")
	}
	if !cmake.ProjectFile {
		t.Error("CMakeLists.txt should be a project file")
	}
}

func TestImplCGenerator_Content(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &ImplCGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Scaffold header comment
	if !strings.Contains(content, "scaffold") {
		t.Error("missing scaffold advisory in header comment")
	}

	// Includes generated header via include path (not hardcoded subdir)
	if !strings.Contains(content, "#include \"test_api.h\"") {
		t.Error("missing generated header include")
	}

	// Standard C includes
	if !strings.Contains(content, "#include <stdlib.h>") {
		t.Error("missing stdlib.h include")
	}
	if !strings.Contains(content, "#include <string.h>") {
		t.Error("missing string.h include")
	}

	// Export macro present
	if !strings.Contains(content, "TEST_API_EXPORT") {
		t.Error("missing export macro")
	}

	// TODO stubs
	if !strings.Contains(content, "// TODO: implement") {
		t.Error("missing TODO comments")
	}

	// Function stubs for minimal.yaml (create_engine, destroy_engine)
	if !strings.Contains(content, "test_api_lifecycle_create_engine(") {
		t.Error("missing create_engine stub")
	}
	if !strings.Contains(content, "test_api_lifecycle_destroy_engine(") {
		t.Error("missing destroy_engine stub")
	}

	// Return 0 for fallible methods
	if !strings.Contains(content, "return 0;") {
		t.Error("missing return 0 for fallible/returning stubs")
	}
}

func TestImplCGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &ImplCGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 output files, got %d", len(files))
	}

	if files[0].Path != "example_app_engine_impl.c" {
		t.Errorf("expected path %q, got %q", "example_app_engine_impl.c", files[0].Path)
	}
	if files[1].Path != "CMakeLists.txt" {
		t.Errorf("expected path %q, got %q", "CMakeLists.txt", files[1].Path)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "#include \"example_app_engine.h\"") {
		t.Error("missing generated header include for full API")
	}
}

func TestImplCGenerator_Registration(t *testing.T) {
	gen, ok := Get("impl_c")
	if !ok {
		t.Fatal("impl_c generator not registered")
	}
	if gen.Name() != "impl_c" {
		t.Errorf("expected name impl_c, got %q", gen.Name())
	}
}
