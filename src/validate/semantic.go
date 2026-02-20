package validate

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

// isValidConstructorName returns true if name is "create" or "create_<snake>".
func isValidConstructorName(name string) bool {
	if name == "create" {
		return true
	}
	return strings.HasPrefix(name, "create_") && len(name) > len("create_")
}

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

		// Collect all names within this interface to detect collisions
		allNames := make(map[string]bool)

		// Validate constructors
		var constructorHandleName string
		for j, ctor := range iface.Constructors {
			ctorPath := fmt.Sprintf("%s.constructors[%d]", ifacePath, j)
			if !isValidConstructorName(ctor.Name) {
				result.addError(ctorPath+".name", fmt.Sprintf("constructor name %q must be \"create\" or start with \"create_\"", ctor.Name))
			}
			if allNames[ctor.Name] {
				result.addError(ctorPath+".name", fmt.Sprintf("duplicate name %q in interface %q", ctor.Name, iface.Name))
			}
			allNames[ctor.Name] = true

			// Constructor must be fallible
			if ctor.Error == "" {
				result.addError(ctorPath+".error", fmt.Sprintf("constructor %q must declare an error type", ctor.Name))
			}
			// Constructor must return a handle
			if ctor.Returns == nil {
				result.addError(ctorPath+".returns", fmt.Sprintf("constructor %q must return a handle type", ctor.Name))
			} else if hn, ok := model.IsHandle(ctor.Returns.Type); !ok {
				result.addError(ctorPath+".returns.type", fmt.Sprintf("constructor %q must return a handle type, got %q", ctor.Name, ctor.Returns.Type))
			} else {
				// All constructors in an interface must return the same handle
				if constructorHandleName == "" {
					constructorHandleName = hn
				} else if hn != constructorHandleName {
					result.addError(ctorPath+".returns.type", fmt.Sprintf("constructor %q returns handle %q but interface already has constructors returning %q; all constructors must return the same handle type", ctor.Name, hn, constructorHandleName))
				}
				// Validate the handle is defined
				if !handleNames[hn] {
					result.addError(ctorPath+".returns.type", fmt.Sprintf("handle %q not defined in handles section", hn))
				}
			}
			// Constructor must not take handle-typed input parameters
			for k, param := range ctor.Parameters {
				paramPath := fmt.Sprintf("%s.parameters[%d]", ctorPath, k)
				if hn, ok := model.IsHandle(param.Type); ok {
					result.addError(paramPath+".type", fmt.Sprintf("constructor %q must not take handle parameter %q of type handle:%s", ctor.Name, param.Name, hn))
				}
				validateParamType(result, paramPath, &param, handleNames, resolvedTypes)
			}
			// Validate error type
			if ctor.Error != "" && resolvedTypes != nil {
				info, ok := resolvedTypes[ctor.Error]
				if !ok {
					result.addError(ctorPath+".error", fmt.Sprintf("error type %q not found in FlatBuffers schemas", ctor.Error))
				} else if info.Kind != resolver.TypeKindEnum {
					result.addError(ctorPath+".error", fmt.Sprintf("error type %q must be an enum, got %s", ctor.Error, info.Kind))
				}
			}
		}

		// Check for duplicate method names within interface (and collision with constructors)
		for j, method := range iface.Methods {
			methodPath := fmt.Sprintf("%s.methods[%d]", ifacePath, j)
			if allNames[method.Name] {
				result.addError(methodPath+".name", fmt.Sprintf("duplicate name %q in interface %q", method.Name, iface.Name))
			}
			allNames[method.Name] = true

			validateMethod(result, methodPath, &method, handleNames, resolvedTypes)
		}

		// Interface must have at least one constructor or at least one method
		if len(iface.Constructors) == 0 && len(iface.Methods) == 0 {
			result.addError(ifacePath, fmt.Sprintf("interface %q must have at least one constructor or method", iface.Name))
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
