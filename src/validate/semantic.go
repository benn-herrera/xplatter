package validate

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

// ValidationError represents a single semantic validation error.
type ValidationError struct {
	Path    string // e.g., "interfaces[0].methods[1].returns.type"
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationResult holds all validation errors.
type ValidationResult struct {
	Errors []ValidationError
}

func (r *ValidationResult) addError(path, message string) {
	r.Errors = append(r.Errors, ValidationError{Path: path, Message: message})
}

func (r *ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

func (r *ValidationResult) Error() string {
	if r.IsValid() {
		return ""
	}
	var msgs []string
	for _, e := range r.Errors {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "\n")
}

// Validate performs semantic validation on a parsed API definition.
// resolvedTypes may be nil if FBS files are not available (skips type resolution checks).
func Validate(def *model.APIDefinition, resolvedTypes resolver.ResolvedTypes) *ValidationResult {
	result := &ValidationResult{}

	handleNames := make(map[string]bool)
	for _, h := range def.Handles {
		handleNames[h.Name] = true
	}

	// Check for duplicate handle names
	seen := make(map[string]bool)
	for i, h := range def.Handles {
		path := fmt.Sprintf("handles[%d].name", i)
		if seen[h.Name] {
			result.addError(path, fmt.Sprintf("duplicate handle name %q", h.Name))
		}
		seen[h.Name] = true
	}

	// Check for duplicate interface names
	ifaceSeen := make(map[string]bool)
	for i, iface := range def.Interfaces {
		ifacePath := fmt.Sprintf("interfaces[%d]", i)
		if ifaceSeen[iface.Name] {
			result.addError(ifacePath+".name", fmt.Sprintf("duplicate interface name %q", iface.Name))
		}
		ifaceSeen[iface.Name] = true

		// Check for duplicate method names within interface
		methodSeen := make(map[string]bool)
		for j, method := range iface.Methods {
			methodPath := fmt.Sprintf("%s.methods[%d]", ifacePath, j)
			if methodSeen[method.Name] {
				result.addError(methodPath+".name", fmt.Sprintf("duplicate method name %q in interface %q", method.Name, iface.Name))
			}
			methodSeen[method.Name] = true

			validateMethod(result, methodPath, &method, handleNames, resolvedTypes)
		}
	}

	return result
}

func validateMethod(result *ValidationResult, path string, method *model.MethodDef, handleNames map[string]bool, resolvedTypes resolver.ResolvedTypes) {
	// Validate parameters
	for k, param := range method.Parameters {
		paramPath := fmt.Sprintf("%s.parameters[%d]", path, k)
		validateParamType(result, paramPath, &param, handleNames, resolvedTypes)
	}

	// Validate return type
	if method.Returns != nil {
		retPath := path + ".returns"
		validateReturnType(result, retPath, method.Returns.Type, handleNames, resolvedTypes)
	}

	// Validate error type
	if method.Error != "" {
		errPath := path + ".error"
		if resolvedTypes != nil {
			info, ok := resolvedTypes[method.Error]
			if !ok {
				result.addError(errPath, fmt.Sprintf("error type %q not found in FlatBuffers schemas", method.Error))
			} else if info.Kind != resolver.TypeKindEnum {
				result.addError(errPath, fmt.Sprintf("error type %q must be an enum, got %s", method.Error, info.Kind))
			}
		}
	}
}

func validateParamType(result *ValidationResult, path string, param *model.ParameterDef, handleNames map[string]bool, resolvedTypes resolver.ResolvedTypes) {
	typePath := path + ".type"
	t := param.Type

	if model.IsPrimitive(t) {
		// Primitives are always valid as parameters
		return
	}

	if model.IsString(t) {
		// String transfer is always ref (implicit), warn if specified differently
		if param.Transfer != "" && param.Transfer != "ref" {
			result.addError(path+".transfer", "string parameters always use ref transfer semantics")
		}
		return
	}

	if elemType, ok := model.IsBuffer(t); ok {
		if !model.IsPrimitive(elemType) {
			result.addError(typePath, fmt.Sprintf("buffer element type %q must be a primitive type", elemType))
		}
		if param.Transfer == "" || param.Transfer == "value" {
			result.addError(path+".transfer", "buffer<T> parameters must specify ref or ref_mut transfer")
		}
		return
	}

	if handleName, ok := model.IsHandle(t); ok {
		if !handleNames[handleName] {
			result.addError(typePath, fmt.Sprintf("handle %q not defined in handles section", handleName))
		}
		if param.Transfer != "" && param.Transfer != "value" {
			result.addError(path+".transfer", "handle parameters always use value transfer (pointer copy)")
		}
		return
	}

	if model.IsFlatBufferType(t) {
		if resolvedTypes != nil {
			if _, ok := resolvedTypes[t]; !ok {
				result.addError(typePath, fmt.Sprintf("FlatBuffer type %q not found in schemas", t))
			}
		}
		return
	}

	result.addError(typePath, fmt.Sprintf("unknown type %q", t))
}

func validateReturnType(result *ValidationResult, path string, t string, handleNames map[string]bool, resolvedTypes resolver.ResolvedTypes) {
	typePath := path + ".type"

	// string and buffer<T> are not allowed as return types
	if model.IsString(t) {
		result.addError(typePath, "string cannot be used as a return type; use a FlatBuffer result type")
		return
	}
	if _, ok := model.IsBuffer(t); ok {
		result.addError(typePath, "buffer<T> cannot be used as a return type; use a FlatBuffer result type")
		return
	}

	if model.IsPrimitive(t) {
		return
	}

	if handleName, ok := model.IsHandle(t); ok {
		if !handleNames[handleName] {
			result.addError(typePath, fmt.Sprintf("handle %q not defined in handles section", handleName))
		}
		return
	}

	if model.IsFlatBufferType(t) {
		if resolvedTypes != nil {
			if _, ok := resolvedTypes[t]; !ok {
				result.addError(typePath, fmt.Sprintf("FlatBuffer type %q not found in schemas", t))
			}
		}
		return
	}

	result.addError(typePath, fmt.Sprintf("unknown return type %q", t))
}
