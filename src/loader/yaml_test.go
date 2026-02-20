package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAPIDefinition_Minimal(t *testing.T) {
	path := filepath.Join("..", "testdata", "minimal.yaml")
	def, err := LoadAPIDefinition(path)
	if err != nil {
		t.Fatalf("unexpected error loading minimal.yaml: %v", err)
	}

	if def.API.Name != "test_api" {
		t.Errorf("expected api name 'test_api', got %q", def.API.Name)
	}
	if def.API.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got %q", def.API.Version)
	}
	if def.API.ImplLang != "c" {
		t.Errorf("expected impl_lang 'c', got %q", def.API.ImplLang)
	}
	if len(def.FlatBuffers) != 1 {
		t.Errorf("expected 1 flatbuffers path, got %d", len(def.FlatBuffers))
	}
	if len(def.Handles) != 1 {
		t.Errorf("expected 1 handle, got %d", len(def.Handles))
	}
	if def.Handles[0].Name != "Engine" {
		t.Errorf("expected handle name 'Engine', got %q", def.Handles[0].Name)
	}
	if len(def.Interfaces) != 1 {
		t.Errorf("expected 1 interface, got %d", len(def.Interfaces))
	}
	if len(def.Interfaces[0].Constructors) != 1 {
		t.Errorf("expected 1 constructor, got %d", len(def.Interfaces[0].Constructors))
	}
	if def.Interfaces[0].Constructors[0].Name != "create_engine" {
		t.Errorf("expected constructor name 'create_engine', got %q", def.Interfaces[0].Constructors[0].Name)
	}
}

func TestLoadAPIDefinition_Full(t *testing.T) {
	path := filepath.Join("..", "testdata", "full.yaml")
	def, err := LoadAPIDefinition(path)
	if err != nil {
		t.Fatalf("unexpected error loading full.yaml: %v", err)
	}

	if def.API.Name != "example_app_engine" {
		t.Errorf("expected api name 'example_app_engine', got %q", def.API.Name)
	}
	if len(def.Handles) != 4 {
		t.Errorf("expected 4 handles, got %d", len(def.Handles))
	}
	if len(def.Interfaces) != 5 {
		t.Errorf("expected 5 interfaces, got %d", len(def.Interfaces))
	}
	if len(def.API.Targets) != 3 {
		t.Errorf("expected 3 targets, got %d", len(def.API.Targets))
	}
}

func TestLoadAPIDefinition_FileNotFound(t *testing.T) {
	_, err := LoadAPIDefinition("nonexistent.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadAPIDefinition_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yaml")
	os.WriteFile(path, []byte("not: valid: yaml: {{{}}}"), 0644)
	_, err := LoadAPIDefinition(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
