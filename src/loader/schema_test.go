package loader

import (
	"testing"
)

func TestValidateSchema_ValidMinimal(t *testing.T) {
	yaml := `
api:
  name: test_api
  version: "1.0.0"
  impl_lang: c
flatbuffers:
  - types.fbs
interfaces:
  - name: test
    methods:
      - name: do_thing
`
	if err := ValidateSchema([]byte(yaml)); err != nil {
		t.Errorf("expected valid schema, got error: %v", err)
	}
}

func TestValidateSchema_MissingAPI(t *testing.T) {
	yaml := `
flatbuffers:
  - types.fbs
interfaces:
  - name: test
    methods:
      - name: do_thing
`
	if err := ValidateSchema([]byte(yaml)); err == nil {
		t.Error("expected error for missing 'api' key")
	}
}

func TestValidateSchema_InvalidAPIName(t *testing.T) {
	yaml := `
api:
  name: BadName
  version: "1.0.0"
  impl_lang: c
flatbuffers:
  - types.fbs
interfaces:
  - name: test
    methods:
      - name: do_thing
`
	if err := ValidateSchema([]byte(yaml)); err == nil {
		t.Error("expected error for PascalCase API name (must be snake_case)")
	}
}

func TestValidateSchema_InvalidImplLang(t *testing.T) {
	yaml := `
api:
  name: test_api
  version: "1.0.0"
  impl_lang: python
flatbuffers:
  - types.fbs
interfaces:
  - name: test
    methods:
      - name: do_thing
`
	if err := ValidateSchema([]byte(yaml)); err == nil {
		t.Error("expected error for invalid impl_lang 'python'")
	}
}

func TestValidateSchema_InvalidTarget(t *testing.T) {
	yaml := `
api:
  name: test_api
  version: "1.0.0"
  impl_lang: c
  targets:
    - android
    - playstation
flatbuffers:
  - types.fbs
interfaces:
  - name: test
    methods:
      - name: do_thing
`
	if err := ValidateSchema([]byte(yaml)); err == nil {
		t.Error("expected error for invalid target 'playstation'")
	}
}

func TestValidateSchema_ReturnTypeStringBlocked(t *testing.T) {
	yaml := `
api:
  name: test_api
  version: "1.0.0"
  impl_lang: c
flatbuffers:
  - types.fbs
interfaces:
  - name: test
    methods:
      - name: get_name
        returns:
          type: string
`
	if err := ValidateSchema([]byte(yaml)); err == nil {
		t.Error("expected error for string as return type")
	}
}

func TestValidateSchema_BufferParamValid(t *testing.T) {
	yaml := `
api:
  name: test_api
  version: "1.0.0"
  impl_lang: c
flatbuffers:
  - types.fbs
interfaces:
  - name: test
    methods:
      - name: write_data
        parameters:
          - name: data
            type: "buffer<uint8>"
            transfer: ref
`
	if err := ValidateSchema([]byte(yaml)); err != nil {
		t.Errorf("expected valid buffer param, got error: %v", err)
	}
}

func TestValidateSchema_AdditionalTopLevelKey(t *testing.T) {
	yaml := `
api:
  name: test_api
  version: "1.0.0"
  impl_lang: c
flatbuffers:
  - types.fbs
interfaces:
  - name: test
    methods:
      - name: do_thing
extra_key: "should fail"
`
	if err := ValidateSchema([]byte(yaml)); err == nil {
		t.Error("expected error for additional top-level key")
	}
}

func TestValidateSchema_HandlePascalCase(t *testing.T) {
	yaml := `
api:
  name: test_api
  version: "1.0.0"
  impl_lang: c
flatbuffers:
  - types.fbs
handles:
  - name: my_handle
interfaces:
  - name: test
    methods:
      - name: do_thing
`
	if err := ValidateSchema([]byte(yaml)); err == nil {
		t.Error("expected error for snake_case handle name (must be PascalCase)")
	}
}
