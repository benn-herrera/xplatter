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
		info, ok := types[name]
		if !ok {
			t.Errorf("expected type %q not found", name)
			continue
		}
		if info.Kind != TypeKindEnum {
			t.Errorf("expected %q to be enum, got %s", name, info.Kind)
		}
	}

	expectedTables := []string{"Common.EntityId", "Common.EventQueue", "Rendering.RendererConfig",
		"Input.TouchEvent", "Input.TouchEventBatch", "Scene.EntityDefinition"}
	for _, name := range expectedTables {
		info, ok := types[name]
		if !ok {
			t.Errorf("expected type %q not found", name)
			continue
		}
		if info.Kind != TypeKindTable {
			t.Errorf("expected %q to be table, got %s", name, info.Kind)
		}
	}

	expectedStructs := []string{"Geometry.Transform3D"}
	for _, name := range expectedStructs {
		info, ok := types[name]
		if !ok {
			t.Errorf("expected type %q not found", name)
			continue
		}
		if info.Kind != TypeKindStruct {
			t.Errorf("expected %q to be struct, got %s", name, info.Kind)
		}
	}

	// Verify enum values are parsed
	ec := types["Common.ErrorCode"]
	if len(ec.EnumValues) != 5 {
		t.Errorf("expected 5 ErrorCode values, got %d", len(ec.EnumValues))
	}
	if ec.BaseType != "int32" {
		t.Errorf("expected ErrorCode base type int32, got %s", ec.BaseType)
	}

	// Verify table fields are parsed
	rc := types["Rendering.RendererConfig"]
	if len(rc.Fields) != 3 {
		t.Errorf("expected 3 RendererConfig fields, got %d", len(rc.Fields))
	}

	// Verify struct fields are parsed
	t3d := types["Geometry.Transform3D"]
	if len(t3d.Fields) != 16 {
		t.Errorf("expected 16 Transform3D fields, got %d", len(t3d.Fields))
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

	types, err := ParseFBSFiles([]string{tmp}, []string{"a.fbs", "b.fbs"})
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

func TestParseFBSFiles_FallbackSearchDir(t *testing.T) {
	primary := t.TempDir()
	fallback := t.TempDir()

	// Put a.fbs in primary, b.fbs only in fallback
	os.WriteFile(filepath.Join(primary, "a.fbs"), []byte("namespace A;\nenum Color : byte { Red = 0 }\n"), 0644)
	os.WriteFile(filepath.Join(fallback, "b.fbs"), []byte("namespace B;\ntable Point { x: float; }\n"), 0644)

	types, err := ParseFBSFiles([]string{primary, fallback}, []string{"a.fbs", "b.fbs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := types["A.Color"]; !ok {
		t.Error("expected A.Color from primary dir")
	}
	if _, ok := types["B.Point"]; !ok {
		t.Error("expected B.Point from fallback dir")
	}
}

func TestResolveFBSPath_NotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := ResolveFBSPath("nonexistent.fbs", []string{tmp})
	if err == nil {
		t.Error("expected error for nonexistent file")
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
