package gen

import (
	"testing"
)

func TestCABIFunctionName(t *testing.T) {
	tests := []struct {
		apiName, ifaceName, methodName string
		want                           string
	}{
		{"my_engine", "lifecycle", "create_engine", "my_engine_lifecycle_create_engine"},
		{"test_api", "renderer", "begin_frame", "test_api_renderer_begin_frame"},
	}
	for _, tt := range tests {
		got := CABIFunctionName(tt.apiName, tt.ifaceName, tt.methodName)
		if got != tt.want {
			t.Errorf("CABIFunctionName(%q, %q, %q) = %q, want %q",
				tt.apiName, tt.ifaceName, tt.methodName, got, tt.want)
		}
	}
}

func TestHandleTypedefName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Engine", "engine_handle"},
		{"Renderer", "renderer_handle"},
		{"MyWidget", "my_widget_handle"},
	}
	for _, tt := range tests {
		got := HandleTypedefName(tt.input)
		if got != tt.want {
			t.Errorf("HandleTypedefName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"create_engine", "CreateEngine"},
		{"begin_frame", "BeginFrame"},
		{"a", "A"},
		{"hello_world_test", "HelloWorldTest"},
	}
	for _, tt := range tests {
		got := ToPascalCase(tt.input)
		if got != tt.want {
			t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"create_engine", "createEngine"},
		{"begin_frame", "beginFrame"},
		{"a", "a"},
	}
	for _, tt := range tests {
		got := ToCamelCase(tt.input)
		if got != tt.want {
			t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCParamType(t *testing.T) {
	tests := []struct {
		paramType string
		transfer  string
		want      string
	}{
		{"int32", "value", "int32_t"},
		{"float64", "value", "double"},
		{"bool", "value", "bool"},
		{"string", "", "const char*"},
		{"handle:Engine", "", "engine_handle"},
		{"Common.ErrorCode", "ref", "const Common_ErrorCode*"},
		{"Common.EventQueue", "ref_mut", "Common_EventQueue*"},
	}
	for _, tt := range tests {
		got := CParamType(tt.paramType, tt.transfer)
		if got != tt.want {
			t.Errorf("CParamType(%q, %q) = %q, want %q", tt.paramType, tt.transfer, got, tt.want)
		}
	}
}

func TestCReturnType(t *testing.T) {
	tests := []struct {
		retType string
		want    string
	}{
		{"int32", "int32_t"},
		{"bool", "bool"},
		{"handle:Engine", "engine_handle"},
		{"Common.EntityId", "Common_EntityId"},
	}
	for _, tt := range tests {
		got := CReturnType(tt.retType)
		if got != tt.want {
			t.Errorf("CReturnType(%q) = %q, want %q", tt.retType, got, tt.want)
		}
	}
}

func TestUpperSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test_api", "TEST_API"},
		{"example_app_engine", "EXAMPLE_APP_ENGINE"},
	}
	for _, tt := range tests {
		got := UpperSnakeCase(tt.input)
		if got != tt.want {
			t.Errorf("UpperSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
