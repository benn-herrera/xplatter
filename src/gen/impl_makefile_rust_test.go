package gen

import (
	"strings"
	"testing"
)

func TestRustMakefileGenerator_Registry(t *testing.T) {
	gen, ok := Get("impl_makefile_rust")
	if !ok {
		t.Fatal("impl_makefile_rust generator not found in registry")
	}
	if gen.Name() != "impl_makefile_rust" {
		t.Errorf("expected name %q, got %q", "impl_makefile_rust", gen.Name())
	}
}

func TestRustMakefileGenerator_Generate(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &RustMakefileGenerator{}

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

func TestRustMakefileGenerator_Content(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &RustMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	// Rust specific
	if !strings.Contains(content, "IMPL_LANG := rust") {
		t.Error("missing IMPL_LANG := rust")
	}
	if !strings.Contains(content, "cargo test") {
		t.Error("missing cargo test in run target")
	}
	if !strings.Contains(content, "cargo build --release") {
		t.Error("missing cargo build --release")
	}
	if !strings.Contains(content, "cargo clean") {
		t.Error("missing cargo clean in clean target")
	}

	// Codegen stamp with -o . (Rust outputs to project root)
	if !strings.Contains(content, "$(XPLATTER) generate --impl-lang rust -o .") {
		t.Error("missing codegen stamp with -o . flag")
	}

	// No generated/ prefix for bindings (Rust uses -o .)
	if !strings.Contains(content, "GEN_HEADER         := $(API_NAME).h") {
		t.Error("Rust should have GEN_HEADER without generated/ prefix")
	}
}

func TestRustMakefileGenerator_IOSUsesCargo(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &RustMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "cargo build --release --target $(2)") {
		t.Error("missing cargo build per target in iOS rules")
	}
	if !strings.Contains(content, "aarch64-apple-ios") {
		t.Error("missing iOS ARM64 Rust target")
	}
	if !strings.Contains(content, "aarch64-apple-ios-sim") {
		t.Error("missing iOS sim ARM64 Rust target")
	}
}

func TestRustMakefileGenerator_AndroidUsesCargoAndNDK(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &RustMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "cargo build --release --target $(2)") {
		t.Error("missing cargo build per target in Android rules")
	}
	if !strings.Contains(content, "aarch64-linux-android") {
		t.Error("missing Android ARM64 Rust target")
	}
	if !strings.Contains(content, "--whole-archive") {
		t.Error("missing --whole-archive for Rust static lib linking on Android")
	}
}

func TestRustMakefileGenerator_WASMUsesCargo(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &RustMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "wasm32-unknown-unknown") {
		t.Error("missing wasm32-unknown-unknown target")
	}
}

func TestRustMakefileGenerator_SharedLibInstallName(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &RustMakefileGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content := string(files[0].Content)

	if !strings.Contains(content, "install_name_tool") {
		t.Error("missing install_name_tool for macOS shared lib")
	}
}

func TestRustMakefileGenerator_Packaging(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &RustMakefileGenerator{}

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
