package model

import (
	"regexp"
	"strings"
)

// APIDefinition is the top-level structure of an xplatter API definition YAML file.
type APIDefinition struct {
	API         APIMetadata    `yaml:"api"`
	FlatBuffers []string       `yaml:"flatbuffers"`
	Handles     []HandleDef    `yaml:"handles,omitempty"`
	Interfaces  []InterfaceDef `yaml:"interfaces"`
}

// APIMetadata holds API-level metadata.
type APIMetadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description,omitempty"`
	ImplLang    string   `yaml:"impl_lang"`
	Targets     []string `yaml:"targets,omitempty"`
}

// HandleDef defines an opaque handle type.
type HandleDef struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// InterfaceDef groups related methods.
type InterfaceDef struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Methods     []MethodDef `yaml:"methods"`
}

// MethodDef defines a single API method.
type MethodDef struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description,omitempty"`
	Parameters  []ParameterDef `yaml:"parameters,omitempty"`
	Returns     *ReturnDef     `yaml:"returns,omitempty"`
	Error       string         `yaml:"error,omitempty"`
}

// ParameterDef defines a method parameter.
type ParameterDef struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Transfer    string `yaml:"transfer,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// ReturnDef defines a method return value.
type ReturnDef struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description,omitempty"`
}

var primitiveTypes = map[string]bool{
	"int8": true, "int16": true, "int32": true, "int64": true,
	"uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true, "bool": true,
}

var bufferPattern = regexp.MustCompile(`^buffer<(\w+)>$`)
var handlePattern = regexp.MustCompile(`^handle:([A-Z][a-zA-Z0-9]*)$`)
var flatBufferPattern = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*(\.[A-Z][a-zA-Z0-9]*)*$`)

// IsPrimitive returns true if the type is a primitive type.
func IsPrimitive(t string) bool {
	return primitiveTypes[t]
}

// IsString returns true if the type is the string type.
func IsString(t string) bool {
	return t == "string"
}

// IsBuffer returns the element type and true if the type is a buffer<T> type.
func IsBuffer(t string) (elementType string, ok bool) {
	m := bufferPattern.FindStringSubmatch(t)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// IsHandle returns the handle name and true if the type is a handle:Name type.
func IsHandle(t string) (name string, ok bool) {
	m := handlePattern.FindStringSubmatch(t)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// IsFlatBufferType returns true if the type looks like a FlatBuffers fully-qualified type reference.
func IsFlatBufferType(t string) bool {
	if IsPrimitive(t) || IsString(t) || t == "bool" {
		return false
	}
	if _, ok := IsBuffer(t); ok {
		return false
	}
	if _, ok := IsHandle(t); ok {
		return false
	}
	return flatBufferPattern.MatchString(t)
}

// AllTargets is the complete list of valid target platforms.
var AllTargets = []string{"android", "ios", "web", "windows", "macos", "linux"}

// ValidImplLangs is the complete list of valid implementation languages.
var ValidImplLangs = []string{"cpp", "rust", "go", "c"}

// EffectiveTargets returns the targets to generate for, defaulting to all targets.
func (a *APIDefinition) EffectiveTargets() []string {
	if len(a.API.Targets) > 0 {
		return a.API.Targets
	}
	return AllTargets
}

// HandleByName looks up a handle definition by name.
func (a *APIDefinition) HandleByName(name string) *HandleDef {
	for i := range a.Handles {
		if a.Handles[i].Name == name {
			return &a.Handles[i]
		}
	}
	return nil
}

// PrimitiveCType returns the C type for a primitive type name.
func PrimitiveCType(t string) string {
	switch t {
	case "int8":
		return "int8_t"
	case "int16":
		return "int16_t"
	case "int32":
		return "int32_t"
	case "int64":
		return "int64_t"
	case "uint8":
		return "uint8_t"
	case "uint16":
		return "uint16_t"
	case "uint32":
		return "uint32_t"
	case "uint64":
		return "uint64_t"
	case "float32":
		return "float"
	case "float64":
		return "double"
	case "bool":
		return "bool"
	default:
		return t
	}
}

// HandleToSnake converts a PascalCase handle name to snake_case.
func HandleToSnake(name string) string {
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// FlatBufferCType converts a FlatBuffers type reference to a C type name.
// e.g., "Common.ErrorCode" → "Common_ErrorCode"
func FlatBufferCType(t string) string {
	return strings.ReplaceAll(t, ".", "_")
}
