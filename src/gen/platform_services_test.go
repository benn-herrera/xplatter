package gen

import (
	"strings"
	"testing"
)

func TestPlatformServicesGenerator_Registry(t *testing.T) {
	gen, ok := Get("impl_platform_services")
	if !ok {
		t.Fatal("impl_platform_services generator not found in registry")
	}
	if gen.Name() != "impl_platform_services" {
		t.Errorf("expected name %q, got %q", "impl_platform_services", gen.Name())
	}
}

func TestPlatformServicesGenerator_Minimal(t *testing.T) {
	ctx := loadTestAPI(t, "minimal.yaml")
	gen := &PlatformServicesGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// minimal.yaml has impl_lang: c, no explicit targets → defaults should apply
	// Expect: desktop.c, web.c (from default targets)
	for _, f := range files {
		if !f.Scaffold {
			t.Errorf("expected %s to be scaffold", f.Path)
		}
		if !f.ProjectFile {
			t.Errorf("expected %s to be project file", f.Path)
		}
	}

	// Find desktop.c
	var desktop *OutputFile
	for _, f := range files {
		if f.Path == "platform_services/desktop.c" {
			desktop = f
		}
	}
	if desktop == nil {
		t.Fatal("missing platform_services/desktop.c")
	}

	content := string(desktop.Content)
	if !strings.Contains(content, "test_api_log_sink") {
		t.Error("missing test_api_log_sink in desktop.c")
	}
	if !strings.Contains(content, "test_api_resource_count") {
		t.Error("missing test_api_resource_count in desktop.c")
	}
	if !strings.Contains(content, "test_api_resource_read") {
		t.Error("missing test_api_resource_read in desktop.c")
	}
	if !strings.Contains(content, "(void)level") {
		t.Error("desktop.c should have no-op log_sink with void casts")
	}
}

func TestPlatformServicesGenerator_Full(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &PlatformServicesGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// full.yaml has targets: [android, ios, web] — no desktop targets
	pathSet := make(map[string]bool)
	for _, f := range files {
		pathSet[f.Path] = true
	}

	expectedPaths := []string{
		"platform_services/ios.c",
		"platform_services/android.c",
		"platform_services/web.c",
	}
	for _, p := range expectedPaths {
		if !pathSet[p] {
			t.Errorf("missing file %s", p)
		}
	}

	// desktop.c should NOT be present (no windows/linux/macos in targets)
	if pathSet["platform_services/desktop.c"] {
		t.Error("desktop.c should not be generated when no desktop targets specified")
	}
}

func TestPlatformServicesGenerator_IOSLogging(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &PlatformServicesGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	var ios *OutputFile
	for _, f := range files {
		if f.Path == "platform_services/ios.c" {
			ios = f
		}
	}
	if ios == nil {
		t.Fatal("missing platform_services/ios.c")
	}

	content := string(ios.Content)
	if !strings.Contains(content, "os/log.h") {
		t.Error("iOS stub should include os/log.h")
	}
	if !strings.Contains(content, "os_log_with_type") {
		t.Error("iOS stub should use os_log_with_type for logging")
	}
}

func TestPlatformServicesGenerator_AndroidLogging(t *testing.T) {
	ctx := loadTestAPI(t, "full.yaml")
	gen := &PlatformServicesGenerator{}

	files, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	var android *OutputFile
	for _, f := range files {
		if f.Path == "platform_services/android.c" {
			android = f
		}
	}
	if android == nil {
		t.Fatal("missing platform_services/android.c")
	}

	content := string(android.Content)
	if !strings.Contains(content, "android/log.h") {
		t.Error("Android stub should include android/log.h")
	}
	if !strings.Contains(content, "__android_log_print") {
		t.Error("Android stub should use __android_log_print for logging")
	}
}
