package resolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveFlatc_ExplicitPath(t *testing.T) {
	// Create a temp file to simulate flatc binary
	tmp := t.TempDir()
	fakeExe := filepath.Join(tmp, "flatc")
	os.WriteFile(fakeExe, []byte("#!/bin/sh\n"), 0755)

	path, err := ResolveFlatc(fakeExe)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != fakeExe {
		t.Errorf("expected %q, got %q", fakeExe, path)
	}
}

func TestResolveFlatc_ExplicitPathNotFound(t *testing.T) {
	_, err := ResolveFlatc("/nonexistent/flatc")
	if err == nil {
		t.Error("expected error for nonexistent explicit path")
	}
}

func TestResolveFlatc_EnvVar(t *testing.T) {
	tmp := t.TempDir()
	fakeExe := filepath.Join(tmp, "flatc")
	os.WriteFile(fakeExe, []byte("#!/bin/sh\n"), 0755)

	t.Setenv("XPLATTER_FLATC_PATH", fakeExe)

	path, err := ResolveFlatc("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != fakeExe {
		t.Errorf("expected %q, got %q", fakeExe, path)
	}
}

func TestResolveFlatc_EnvVarNotFound(t *testing.T) {
	t.Setenv("XPLATTER_FLATC_PATH", "/nonexistent/flatc")
	_, err := ResolveFlatc("")
	if err == nil {
		t.Error("expected error for nonexistent env path")
	}
}

func TestResolveFlatc_FlagTakesPrecedence(t *testing.T) {
	tmp := t.TempDir()
	flagExe := filepath.Join(tmp, "flatc_flag")
	envExe := filepath.Join(tmp, "flatc_env")
	os.WriteFile(flagExe, []byte("#!/bin/sh\n"), 0755)
	os.WriteFile(envExe, []byte("#!/bin/sh\n"), 0755)

	t.Setenv("XPLATTER_FLATC_PATH", envExe)

	path, err := ResolveFlatc(flagExe)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != flagExe {
		t.Errorf("expected flag path %q to take precedence, got %q", flagExe, path)
	}
}
