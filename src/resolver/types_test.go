package resolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFBSFile_CommonSchema(t *testing.T) {
	path := filepath.Join("..", "testdata", "schemas", "common.fbs")
	types, err := ParseFBSFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check expected types exist
	expectedEnums := []string{"Common.ErrorCode", "Common.LogLevel", "Rendering.TextureFormat"}
	for _, name := range expectedEnums {
		kind, ok := types[name]
		if !ok {
			t.Errorf("expected type %q not found", name)
			continue
		}
		if kind != TypeKindEnum {
			t.Errorf("expected %q to be enum, got %s", name, kind)
		}
	}

	expectedTables := []string{"Common.EntityId", "Common.EventQueue", "Rendering.RendererConfig",
		"Input.TouchEvent", "Input.TouchEventBatch", "Scene.EntityDefinition"}
	for _, name := range expectedTables {
		kind, ok := types[name]
		if !ok {
			t.Errorf("expected type %q not found", name)
			continue
		}
		if kind != TypeKindTable {
			t.Errorf("expected %q to be table, got %s", name, kind)
		}
	}

	expectedStructs := []string{"Geometry.Transform3D"}
	for _, name := range expectedStructs {
		kind, ok := types[name]
		if !ok {
			t.Errorf("expected type %q not found", name)
			continue
		}
		if kind != TypeKindStruct {
			t.Errorf("expected %q to be struct, got %s", name, kind)
		}
	}
}

func TestParseFBSFile_NotFound(t *testing.T) {
	_, err := ParseFBSFile("/nonexistent/file.fbs")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseFBSFiles_MultipleFiles(t *testing.T) {
	tmp := t.TempDir()

	// Write two .fbs files
	fbs1 := filepath.Join(tmp, "a.fbs")
	os.WriteFile(fbs1, []byte("namespace A;\nenum Color : byte { Red = 0, Green, Blue }\n"), 0644)

	fbs2 := filepath.Join(tmp, "b.fbs")
	os.WriteFile(fbs2, []byte("namespace B;\ntable Point { x: float; y: float; }\n"), 0644)

	types, err := ParseFBSFiles(tmp, []string{"a.fbs", "b.fbs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := types["A.Color"]; !ok {
		t.Error("expected A.Color")
	}
	if _, ok := types["B.Point"]; !ok {
		t.Error("expected B.Point")
	}
}

func TestParseFBSFile_NoNamespace(t *testing.T) {
	tmp := t.TempDir()
	fbs := filepath.Join(tmp, "bare.fbs")
	os.WriteFile(fbs, []byte("table Foo { x: int32; }\n"), 0644)

	types, err := ParseFBSFile(fbs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := types["Foo"]; !ok {
		t.Error("expected Foo (no namespace)")
	}
}

func TestParseFBSFile_Comments(t *testing.T) {
	tmp := t.TempDir()
	fbs := filepath.Join(tmp, "commented.fbs")
	content := `namespace Test;
// This is a comment
enum Status : byte { Ok = 0 } // inline comment
// table NotReal { x: int32; }
table Real { y: int32; }
`
	os.WriteFile(fbs, []byte(content), 0644)

	types, err := ParseFBSFile(fbs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := types["Test.Status"]; !ok {
		t.Error("expected Test.Status")
	}
	if _, ok := types["Test.Real"]; !ok {
		t.Error("expected Test.Real")
	}
	if _, ok := types["Test.NotReal"]; ok {
		t.Error("did not expect Test.NotReal (commented out)")
	}
}
