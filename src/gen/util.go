package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplattergy/model"
)

// CABIFunctionName builds the C ABI function name: <api_name>_<interface>_<method>
func CABIFunctionName(apiName, ifaceName, methodName string) string {
	return fmt.Sprintf("%s_%s_%s", apiName, ifaceName, methodName)
}

// HandleTypedefName returns the C typedef name for a handle.
// e.g., "Engine" → "engine_handle"
func HandleTypedefName(handleName string) string {
	return model.HandleToSnake(handleName) + "_handle"
}

// HandleStructName returns the C struct tag for a handle.
// e.g., "Engine" → "engine_s"
func HandleStructName(handleName string) string {
	return model.HandleToSnake(handleName) + "_s"
}

// CParamType returns the C type string for a parameter, considering transfer semantics.
func CParamType(paramType string, transfer string) string {
	if model.IsString(paramType) {
		return "const char*"
	}

	if elemType, ok := model.IsBuffer(paramType); ok {
		cType := model.PrimitiveCType(elemType)
		if transfer == "ref_mut" {
			return cType + "*"
		}
		return "const " + cType + "*"
	}

	if handleName, ok := model.IsHandle(paramType); ok {
		return HandleTypedefName(handleName)
	}

	if model.IsPrimitive(paramType) {
		return model.PrimitiveCType(paramType)
	}

	// FlatBuffer type
	cType := model.FlatBufferCType(paramType)
	if transfer == "ref_mut" {
		return cType + "*"
	}
	if transfer == "ref" {
		return "const " + cType + "*"
	}
	return cType
}

// CReturnType returns the C type string for a return value.
func CReturnType(retType string) string {
	if handleName, ok := model.IsHandle(retType); ok {
		return HandleTypedefName(handleName)
	}
	if model.IsPrimitive(retType) {
		return model.PrimitiveCType(retType)
	}
	// FlatBuffer type
	return model.FlatBufferCType(retType)
}

// COutParamType returns the C type for an out-parameter (pointer to return type).
func COutParamType(retType string) string {
	return CReturnType(retType) + "*"
}

// UpperSnakeCase converts a snake_case string to UPPER_SNAKE_CASE.
func UpperSnakeCase(s string) string {
	return strings.ToUpper(s)
}

// ToPascalCase converts a snake_case string to PascalCase.
func ToPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			result.WriteString(part[1:])
		}
	}
	return result.String()
}

// ToCamelCase converts a snake_case string to camelCase.
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if len(pascal) == 0 {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

// ExportMacroName returns the export macro name for an API, e.g. "HELLO_XPLATTERGY_EXPORT".
func ExportMacroName(apiName string) string {
	return UpperSnakeCase(apiName) + "_EXPORT"
}

// BuildMacroName returns the build macro name for an API, e.g. "HELLO_XPLATTERGY_BUILD".
func BuildMacroName(apiName string) string {
	return UpperSnakeCase(apiName) + "_BUILD"
}

// CollectErrorTypes returns deduplicated error type names used across all methods.
func CollectErrorTypes(api *model.APIDefinition) []string {
	seen := map[string]bool{}
	var result []string
	for _, iface := range api.Interfaces {
		for _, method := range iface.Methods {
			if method.Error != "" && !seen[method.Error] {
				seen[method.Error] = true
				result = append(result, method.Error)
			}
		}
	}
	return result
}

// FindDestroyInfo looks for a destroy/release method for a handle type,
// returning the interface and method names.
func FindDestroyInfo(api *model.APIDefinition, handleName string) (ifaceName, methodName string, found bool) {
	snake := model.HandleToSnake(handleName)
	for _, iface := range api.Interfaces {
		for _, method := range iface.Methods {
			if (method.Name == "destroy_"+snake || method.Name == "release_"+snake) &&
				len(method.Parameters) == 1 {
				hName, ok := model.IsHandle(method.Parameters[0].Type)
				if ok && hName == handleName {
					return iface.Name, method.Name, true
				}
			}
		}
	}
	return "", "", false
}
