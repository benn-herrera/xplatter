package gen

import (
	"strings"
	"testing"
)

func TestCMakefileGenerator_Registry(t *testing.T) {
	gen, ok := Get("impl_makefile_c")
	if !ok {
		t.Fatal("impl_makefile_c generator not found in registry")
	}
	if gen.Name() != "impl_makefile_c" {
		t.Errorf("expected name %q, got %q", "impl_makefile_c", gen.Name())
	}
}

func TestCMakefileGenerator_Generate(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &CMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "Makefile" {
		t.Errorf("expected path Makefile, got %q", files[0].Path)
	}
	if !files[0].Scaffold {
		t.Error("Makefile should be scaffold")
	}
	if !files[0].ProjectFile {
		t.Error("Makefile should be a project file")
	}
}

func TestCMakefileGenerator_Content(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &CMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// C specific: uses cc, not c++
	if !strings.Contains(content, "CC         ?= cc") {
		t.Error("missing CC variable")
	}
	if !strings.Contains(content, "IMPL_LANG := c") {
		t.Error("missing IMPL_LANG := c")
	}
	if !strings.Contains(content, "-std=c17") {
		t.Error("missing C11 standard flag")
	}

	// No SHIM_SOURCE — C impl directly exports C ABI
	if strings.Contains(content, "SHIM_SOURCE") {
		t.Error("C Makefile should not have SHIM_SOURCE")
	}

	// GEN_DIR variable drives include path and binding file paths
	if !strings.Contains(content, "GEN_DIR            := generated/") {
		t.Error("missing GEN_DIR variable")
	}
	if !strings.Contains(content, "GEN_HEADER         := $(GEN_DIR)$(API_NAME).h") {
		t.Error("missing GEN_HEADER using $(GEN_DIR)")
	}
	if !strings.Contains(content, "-I$(GEN_DIR)") {
		t.Error("missing -I$(GEN_DIR) in CFLAGS")
	}

	// Codegen stamp
	if !strings.Contains(content, "$(XPLATTER) generate --impl-lang c -o generated") {
		t.Error("missing codegen stamp with -o generated flag")
	}

	// Local build uses $(CC)
	if !strings.Contains(content, "$(CC) $(CFLAGS)") {
		t.Error("missing CC compilation command")
	}
	if !strings.Contains(content, "/Fo:$(BUILD_DIR)/") {
		t.Error("missing /Fo: flag to redirect MSVC obj files into build/")
	}
}

func TestCMakefileGenerator_NoShimInIOSRules(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &CMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// iOS rules: only impl.o + platform.o, no shim.o
	if strings.Contains(content, "shim.o") {
		t.Error("C Makefile iOS rules should not reference shim.o")
	}
}

func TestCMakefileGenerator_AndroidUsesClang(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &CMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Android should use clang (not c++/cc) via NDK
	if !strings.Contains(content, "$(NDK_BIN)/$(2)-clang") {
		t.Error("missing NDK clang compilation in Android rules")
	}
	// Should NOT reference $(CXX) in Android rules
	if strings.Contains(content, "$(NDK_BIN)/$(2)-$(CXX)") {
		t.Error("C Makefile Android rules should not use $(CXX)")
	}
}

func TestCMakefileGenerator_WASMNoShim(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &CMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// WASM rules should use cmake via the Emscripten toolchain
	if !strings.Contains(content, "cmake") {
		t.Error("missing cmake in WASM rules")
	}
	if !strings.Contains(content, "EMSCRIPTEN_TOOLCHAIN") {
		t.Error("missing EMSCRIPTEN_TOOLCHAIN in WASM rules")
	}
	// WASM rules should not have shim.o
	if strings.Contains(content, "shim.o") {
		t.Error("C Makefile WASM rules should not reference shim.o")
	}
}
