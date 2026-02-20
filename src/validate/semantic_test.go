package validate

import (
	"strings"
	"testing"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

func minimalAPI() *model.APIDefinition {
	return &model.APIDefinition{
		API: model.APIMetadata{
			Name:     "test_api",
			Version:  "0.1.0",
			ImplLang: "c",
		},
		FlatBuffers: []string{"specs/common.fbs"},
		Handles: []model.HandleDef{
			{Name: "Engine"},
		},
		Interfaces: []model.InterfaceDef{
			{
				Name: "lifecycle",
				Methods: []model.MethodDef{
					{
						Name: "create_engine",
						Returns: &model.ReturnDef{
							Type: "handle:Engine",
						},
						Error: "Common.ErrorCode",
					},
				},
			},
		},
	}
}

func TestValidate_ValidMinimal(t *testing.T) {
	types := resolver.ResolvedTypes{
		"Common.ErrorCode": &resolver.TypeInfo{Kind: resolver.TypeKindEnum},
	}
	result := Validate(minimalAPI(), types)
	if !result.IsValid() {
		t.Errorf("expected valid, got errors:\n%s", result.Error())
	}
}

func TestValidate_UnresolvedHandle(t *testing.T) {
	api := minimalAPI()
	api.Interfaces[0].Methods = append(api.Interfaces[0].Methods, model.MethodDef{
		Name: "get_widget",
		Parameters: []model.ParameterDef{
			{Name: "w", Type: "handle:Widget"},
		},
	})

	result := Validate(api, nil)
	if result.IsValid() {
		t.Error("expected validation error for unresolved handle:Widget")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "Widget") && strings.Contains(e.Message, "not defined") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about Widget handle, got: %s", result.Error())
	}
}

func TestValidate_StringReturnType(t *testing.T) {
	api := minimalAPI()
	api.Interfaces[0].Methods = append(api.Interfaces[0].Methods, model.MethodDef{
		Name:    "get_name",
		Returns: &model.ReturnDef{Type: "string"},
	})

	result := Validate(api, nil)
	if result.IsValid() {
		t.Error("expected validation error for string return type")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "string cannot be used as a return type") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected string return type error, got: %s", result.Error())
	}
}

func TestValidate_BufferReturnType(t *testing.T) {
	api := minimalAPI()
	api.Interfaces[0].Methods = append(api.Interfaces[0].Methods, model.MethodDef{
		Name:    "get_data",
		Returns: &model.ReturnDef{Type: "buffer<uint8>"},
	})

	result := Validate(api, nil)
	if result.IsValid() {
		t.Error("expected validation error for buffer<T> return type")
	}
}

func TestValidate_ErrorTypeNotEnum(t *testing.T) {
	api := minimalAPI()
	types := resolver.ResolvedTypes{
		"Common.ErrorCode": &resolver.TypeInfo{Kind: resolver.TypeKindEnum},
		"Common.Result":    &resolver.TypeInfo{Kind: resolver.TypeKindTable},
	}
	api.Interfaces[0].Methods[0].Error = "Common.Result"

	result := Validate(api, types)
	if result.IsValid() {
		t.Error("expected validation error for non-enum error type")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "must be an enum") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected enum error, got: %s", result.Error())
	}
}

func TestValidate_DuplicateHandleName(t *testing.T) {
	api := minimalAPI()
	api.Handles = append(api.Handles, model.HandleDef{Name: "Engine"})

	result := Validate(api, nil)
	if result.IsValid() {
		t.Error("expected validation error for duplicate handle name")
	}
}

func TestValidate_DuplicateMethodName(t *testing.T) {
	api := minimalAPI()
	api.Interfaces[0].Methods = append(api.Interfaces[0].Methods, model.MethodDef{
		Name: "create_engine",
	})

	result := Validate(api, nil)
	if result.IsValid() {
		t.Error("expected validation error for duplicate method name")
	}
}

func TestValidate_HandleTransferNotValue(t *testing.T) {
	api := minimalAPI()
	api.Interfaces[0].Methods = append(api.Interfaces[0].Methods, model.MethodDef{
		Name: "use_engine",
		Parameters: []model.ParameterDef{
			{Name: "engine", Type: "handle:Engine", Transfer: "ref"},
		},
	})

	result := Validate(api, nil)
	if result.IsValid() {
		t.Error("expected validation error for handle with ref transfer")
	}
}

func TestValidate_BufferMissingTransfer(t *testing.T) {
	api := minimalAPI()
	api.Interfaces[0].Methods = append(api.Interfaces[0].Methods, model.MethodDef{
		Name: "send_data",
		Parameters: []model.ParameterDef{
			{Name: "data", Type: "buffer<uint8>"},
		},
	})

	result := Validate(api, nil)
	if result.IsValid() {
		t.Error("expected validation error for buffer without transfer")
	}
}

func TestValidate_UnresolvedFlatBufferType(t *testing.T) {
	api := minimalAPI()
	api.Interfaces[0].Methods = append(api.Interfaces[0].Methods, model.MethodDef{
		Name: "set_config",
		Parameters: []model.ParameterDef{
			{Name: "config", Type: "Nonexistent.Type", Transfer: "ref"},
		},
	})

	types := resolver.ResolvedTypes{
		"Common.ErrorCode": &resolver.TypeInfo{Kind: resolver.TypeKindEnum},
	}

	result := Validate(api, types)
	if result.IsValid() {
		t.Error("expected validation error for unresolved FlatBuffer type")
	}
}

func TestValidate_CollectsAllErrors(t *testing.T) {
	api := minimalAPI()
	// Add multiple errors: duplicate handle + string return + unresolved handle ref
	api.Handles = append(api.Handles, model.HandleDef{Name: "Engine"})
	api.Interfaces[0].Methods = append(api.Interfaces[0].Methods,
		model.MethodDef{
			Name:    "get_name",
			Returns: &model.ReturnDef{Type: "string"},
		},
		model.MethodDef{
			Name: "use_widget",
			Parameters: []model.ParameterDef{
				{Name: "w", Type: "handle:Widget"},
			},
		},
	)

	result := Validate(api, nil)
	if result.IsValid() {
		t.Fatal("expected multiple validation errors")
	}
	if len(result.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %s", len(result.Errors), result.Error())
	}
}
