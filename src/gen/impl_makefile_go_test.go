package gen

import (
	"strings"
	"testing"
)

func TestGoMakefileGenerator_Registry(t *testing.T) {
	gen, ok := Get("impl_makefile_go")
	if !ok {
		t.Fatal("impl_makefile_go generator not found in registry")
	}
	if gen.Name() != "impl_makefile_go" {
		t.Errorf("expected name %q, got %q", "impl_makefile_go", gen.Name())
	}
}

func TestGoMakefileGenerator_Generate(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoMakefileGenerator{}

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
}

func TestGoMakefileGenerator_Content(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Go specific
	if !strings.Contains(content, "IMPL_LANG := go") {
		t.Error("missing IMPL_LANG := go")
	}
	if !strings.Contains(content, `CC="$(CGO_CC)" go build -o $(BUILD_DIR)/$(API_NAME)$(EXE) .`) {
		t.Error("missing CC=$(CGO_CC) go build in run target")
	}
	if !strings.Contains(content, `CC="$(CGO_CC)" go build -buildmode=c-shared`) {
		t.Error("missing CC=$(CGO_CC) go build -buildmode=c-shared")
	}
	// CGO_CC variable
	if !strings.Contains(content, "CGO_CC := clang") {
		t.Error("missing CGO_CC := clang default")
	}
	if !strings.Contains(content, "CGO_CC := zig cc -target x86_64-windows-gnu") {
		t.Error("missing CGO_CC Windows override")
	}

	// Codegen stamp with -o generated
	if !strings.Contains(content, "$(XPLATTER) generate --impl-lang go -o generated") {
		t.Error("missing codegen stamp with -o generated flag")
	}

	// GEN_DIR drives binding file paths
	if !strings.Contains(content, "GEN_DIR            := generated/") {
		t.Error("missing GEN_DIR variable")
	}
	if !strings.Contains(content, "GEN_HEADER         := $(GEN_DIR)$(API_NAME).h") {
		t.Error("missing GEN_HEADER using $(GEN_DIR)")
	}
}

func TestGoMakefileGenerator_IOSUsesCGO(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "CGO_ENABLED=1 GOOS=ios") {
		t.Error("missing CGO_ENABLED=1 GOOS=ios in iOS rules")
	}
	if !strings.Contains(content, "-buildmode=c-archive") {
		t.Error("missing c-archive buildmode in iOS rules")
	}
	if !strings.Contains(content, "IOS_CC") {
		t.Error("missing IOS_CC variable")
	}
	if !strings.Contains(content, "SIM_CC") {
		t.Error("missing SIM_CC variable")
	}
}

func TestGoMakefileGenerator_AndroidUsesCGOAndNDK(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "CGO_ENABLED=1 GOOS=android") {
		t.Error("missing CGO_ENABLED=1 GOOS=android")
	}
	if !strings.Contains(content, "CC=$(NDK_BIN)/$(3)-clang") {
		t.Error("missing NDK clang as CC for Go CGO")
	}
	if !strings.Contains(content, "-buildmode=c-shared") {
		t.Error("missing c-shared buildmode in Android rules")
	}
	if !strings.Contains(content, "GEN_JNI_SOURCE_LOCAL") {
		t.Error("missing GEN_JNI_SOURCE_LOCAL for JNI copy strategy")
	}
	if !strings.Contains(content, "GOARM=7") {
		t.Error("missing GOARM=7 for armeabi-v7a")
	}
}

func TestGoMakefileGenerator_WASMUsesWasip1(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "GOOS=wasip1 GOARCH=wasm") {
		t.Error("missing GOOS=wasip1 GOARCH=wasm in WASM rules")
	}
}

func TestGoMakefileGenerator_Packaging(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &GoMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "package-ios") {
		t.Error("missing package-ios target")
	}
	if !strings.Contains(content, "package-android") {
		t.Error("missing package-android target")
	}
	if !strings.Contains(content, "package-web") {
		t.Error("missing package-web target")
	}
	if !strings.Contains(content, "package-desktop") {
		t.Error("missing package-desktop target")
	}
	if !strings.Contains(content, "build.gradle.kts") {
		t.Error("missing build.gradle.kts in Android packaging")
	}
	if !strings.Contains(content, "package.json") {
		t.Error("missing package.json in Web packaging")
	}
	if !strings.Contains(content, "Package.swift") {
		t.Error("missing Package.swift in iOS packaging")
	}
}
