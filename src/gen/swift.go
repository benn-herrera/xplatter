package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

func init() {
	Register("swift", func() Generator { return &SwiftGenerator{} })
}

// SwiftGenerator produces the Swift/C bridge binding file.
type SwiftGenerator struct{}

func (g *SwiftGenerator) Name() string { return "swift" }

func (g *SwiftGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name
	pascalAPI := ToPascalCase(apiName)

	var b strings.Builder

	b.WriteString(GeneratedFileHeader(ctx, "//", false))
	b.WriteString("\nimport Foundation\n\n")

	// Collect all error types used across the API
	errorTypes := CollectErrorTypes(api)

	// Error enum
	if len(errorTypes) > 0 {
		for _, errType := range errorTypes {
			writeSwiftErrorEnum(&b, errType, ctx.ResolvedTypes)
		}
	}

	// Handle wrapper classes
	for _, h := range api.Handles {
		writeSwiftHandleClass(&b, h, api, ctx.ResolvedTypes)
	}

	// Free functions (methods on interfaces that don't take a handle as first param
	// and don't return a handle — these are rare but possible)
	// We group methods by the handle they operate on. Methods that create a handle
	// become static factory methods. Methods with a handle as first param become
	// instance methods. Remaining methods go into a namespace enum.
	writeSwiftFreeFunctions(&b, api, ctx.ResolvedTypes)

	filename := pascalAPI + ".swift"
	return []*OutputFile{
		{Path: filename, Content: []byte(b.String())},
	}, nil
}


// writeSwiftErrorEnum writes a Swift enum conforming to Error for a FlatBuffer error code type.
func writeSwiftErrorEnum(b *strings.Builder, errType string, resolved resolver.ResolvedTypes) {
	swiftName := swiftErrorEnumName(errType)
	b.WriteString(fmt.Sprintf("public enum %s: Int32, Error {\n", swiftName))
	// We generate a case for each possible error value. Since we don't know the
	// enum values from the FBS at codegen time (we only know it's an enum), we
	// generate a raw-value based pattern that the C code returns.
	b.WriteString("    case ok = 0\n")
	b.WriteString("    case invalidArgument = 1\n")
	b.WriteString("    case outOfMemory = 2\n")
	b.WriteString("    case notFound = 3\n")
	b.WriteString("    case internalError = 4\n")
	b.WriteString("}\n\n")
}

// swiftErrorEnumName converts a FlatBuffer error type like "Common.ErrorCode" to a Swift name.
func swiftErrorEnumName(errType string) string {
	return strings.ReplaceAll(errType, ".", "")
}

// writeSwiftHandleClass writes a Swift wrapper class for an opaque handle.
func writeSwiftHandleClass(b *strings.Builder, handle model.HandleDef, api *model.APIDefinition, resolved resolver.ResolvedTypes) {
	apiName := api.API.Name
	className := handle.Name
	handleSnake := model.HandleToSnake(handle.Name)
	handleCType := HandleTypedefName(handle.Name) // e.g. "engine_handle"

	if handle.Description != "" {
		fmt.Fprintf(b, "/// %s\n", handle.Description)
	}
	fmt.Fprintf(b, "public final class %s {\n", className)
	fmt.Fprintf(b, "    let handle: OpaquePointer\n\n")

	// Internal init from raw handle
	fmt.Fprintf(b, "    init(handle: OpaquePointer) {\n")
	fmt.Fprintf(b, "        self.handle = handle\n")
	fmt.Fprintf(b, "    }\n\n")

	// Find the destroy method for this handle
	if destroyIface, destroyMethodName, found := FindDestroyInfo(api, handle.Name); found {
		destroyFunc := CABIFunctionName(apiName, destroyIface, destroyMethodName)
		fmt.Fprintf(b, "    deinit {\n")
		fmt.Fprintf(b, "        %s(handle)\n", destroyFunc)
		fmt.Fprintf(b, "    }\n\n")
	}

	// Find factory methods that return this handle type
	for _, iface := range api.Interfaces {
		for _, method := range iface.Methods {
			if method.Returns != nil {
				if hName, ok := model.IsHandle(method.Returns.Type); ok && hName == handle.Name {
					// Skip destroy methods
					if isDestroyMethod(&method, handle.Name) {
						continue
					}
					writeSwiftFactoryMethod(b, apiName, iface.Name, &method, className, resolved)
				}
			}
		}
	}

	// Find instance methods (first param is this handle, not returning a new handle or returning non-handle)
	for _, iface := range api.Interfaces {
		for _, method := range iface.Methods {
			if isDestroyMethod(&method, handle.Name) {
				continue
			}
			// Skip factory methods (already written above)
			if method.Returns != nil {
				if hName, ok := model.IsHandle(method.Returns.Type); ok && hName == handle.Name {
					continue
				}
			}
			// Check if first param is this handle
			if len(method.Parameters) > 0 {
				if hName, ok := model.IsHandle(method.Parameters[0].Type); ok && hName == handle.Name {
					writeSwiftInstanceMethod(b, apiName, iface.Name, &method, handleCType, resolved)
				}
			}
		}
	}

	b.WriteString("}\n\n")

	_ = handleSnake
	_ = handleCType
}

// isDestroyMethod returns true if the method looks like a destroy/release method for the given handle.
func isDestroyMethod(method *model.MethodDef, handleName string) bool {
	if !strings.HasPrefix(method.Name, "destroy") {
		return false
	}
	if len(method.Parameters) != 1 {
		return false
	}
	if hName, ok := model.IsHandle(method.Parameters[0].Type); ok && hName == handleName {
		return true
	}
	return false
}


// writeSwiftFactoryMethod writes a static factory method that creates a handle.
func writeSwiftFactoryMethod(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef, className string, resolved resolver.ResolvedTypes) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	swiftMethodName := ToCamelCase(method.Name)
	hasError := method.Error != ""

	// Build Swift parameter list (factory methods are static, no self handle)
	var swiftParams []string
	var callArgs []string

	for _, p := range method.Parameters {
		sp, ca := swiftParamAndCallArg(&p, resolved)
		swiftParams = append(swiftParams, sp...)
		callArgs = append(callArgs, ca...)
	}

	paramStr := strings.Join(swiftParams, ", ")

	if hasError {
		errEnumName := swiftErrorEnumName(method.Error)
		if method.Description != "" {
			fmt.Fprintf(b, "    /// %s\n", method.Description)
		}
		fmt.Fprintf(b, "    public static func %s(%s) throws -> %s {\n", swiftMethodName, paramStr, className)
		fmt.Fprintf(b, "        var result: OpaquePointer?\n")

		// Build the C call with withCString wrappers
		writeSwiftCCall(b, funcName, callArgs, method.Parameters, "result", true, errEnumName, className, resolved)

		fmt.Fprintf(b, "    }\n\n")
	} else {
		if method.Description != "" {
			fmt.Fprintf(b, "    /// %s\n", method.Description)
		}
		fmt.Fprintf(b, "    public static func %s(%s) -> %s {\n", swiftMethodName, paramStr, className)
		fmt.Fprintf(b, "        var result: OpaquePointer?\n")

		writeSwiftCCall(b, funcName, callArgs, method.Parameters, "result", false, "", className, resolved)

		fmt.Fprintf(b, "    }\n\n")
	}
}

// writeSwiftInstanceMethod writes an instance method on a handle class.
func writeSwiftInstanceMethod(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef, handleCType string, resolved resolver.ResolvedTypes) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	swiftMethodName := ToCamelCase(method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build Swift parameter list (skip the first param which is self handle)
	var swiftParams []string
	var callArgs []string

	// First arg is always the handle (self)
	callArgs = append(callArgs, "handle")

	for _, p := range method.Parameters[1:] {
		sp, ca := swiftParamAndCallArg(&p, resolved)
		swiftParams = append(swiftParams, sp...)
		callArgs = append(callArgs, ca...)
	}

	paramStr := strings.Join(swiftParams, ", ")

	// Determine return type
	var swiftReturnType string
	if hasReturn {
		swiftReturnType = swiftType(method.Returns.Type, resolved)
	}

	if method.Description != "" {
		fmt.Fprintf(b, "    /// %s\n", method.Description)
	}

	switch {
	case hasError && hasReturn:
		fmt.Fprintf(b, "    public func %s(%s) throws -> %s {\n", swiftMethodName, paramStr, swiftReturnType)
		if isHandleReturn(method.Returns.Type) {
			fmt.Fprintf(b, "        var result: OpaquePointer?\n")
			handleName, _ := model.IsHandle(method.Returns.Type)
			writeSwiftCCall(b, funcName, callArgs, method.Parameters[1:], "result", true, swiftErrorEnumName(method.Error), handleName, resolved)
		} else {
			fmt.Fprintf(b, "        var result: %s = %s\n", swiftCBridgeType(method.Returns.Type, resolved), swiftDefaultValue(method.Returns.Type))
			writeSwiftCCallPrimitive(b, funcName, callArgs, method.Parameters[1:], "result", true, swiftErrorEnumName(method.Error), resolved)
		}
	case hasError && !hasReturn:
		fmt.Fprintf(b, "    public func %s(%s) throws {\n", swiftMethodName, paramStr)
		writeSwiftCCallVoid(b, funcName, callArgs, method.Parameters[1:], true, swiftErrorEnumName(method.Error), resolved)
	case !hasError && hasReturn:
		fmt.Fprintf(b, "    public func %s(%s) -> %s {\n", swiftMethodName, paramStr, swiftReturnType)
		writeSwiftCCallDirect(b, funcName, callArgs, method.Parameters[1:], resolved)
	default:
		fmt.Fprintf(b, "    public func %s(%s) {\n", swiftMethodName, paramStr)
		writeSwiftCCallVoid(b, funcName, callArgs, method.Parameters[1:], false, "", resolved)
	}

	fmt.Fprintf(b, "    }\n\n")
}

// writeSwiftFreeFunctions writes free functions that don't belong to any handle class.
func writeSwiftFreeFunctions(b *strings.Builder, api *model.APIDefinition, resolved resolver.ResolvedTypes) {
	// Collect methods that are not associated with any handle
	// (no handle as first param, not returning a handle, not a destroy method)
	handleNames := map[string]bool{}
	for _, h := range api.Handles {
		handleNames[h.Name] = true
	}

	var freeMethods []struct {
		iface  string
		method model.MethodDef
	}

	for _, iface := range api.Interfaces {
		for _, method := range iface.Methods {
			isHandleRelated := false

			// Check if it returns a handle
			if method.Returns != nil {
				if hName, ok := model.IsHandle(method.Returns.Type); ok && handleNames[hName] {
					isHandleRelated = true
				}
			}

			// Check if first param is a handle
			if len(method.Parameters) > 0 {
				if hName, ok := model.IsHandle(method.Parameters[0].Type); ok && handleNames[hName] {
					isHandleRelated = true
				}
			}

			// Check if it's a destroy method
			for _, h := range api.Handles {
				if isDestroyMethod(&method, h.Name) {
					isHandleRelated = true
					break
				}
			}

			if !isHandleRelated {
				freeMethods = append(freeMethods, struct {
					iface  string
					method model.MethodDef
				}{iface.Name, method})
			}
		}
	}

	if len(freeMethods) == 0 {
		return
	}

	// Write free functions in a namespace enum
	pascalAPI := ToPascalCase(api.API.Name)
	fmt.Fprintf(b, "public enum %s {\n", pascalAPI)
	for _, fm := range freeMethods {
		writeSwiftFreeFunction(b, api.API.Name, fm.iface, &fm.method, resolved)
	}
	b.WriteString("}\n\n")
}

// writeSwiftFreeFunction writes a single free function inside the namespace enum.
func writeSwiftFreeFunction(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef, resolved resolver.ResolvedTypes) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	swiftMethodName := ToCamelCase(method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	var swiftParams []string
	var callArgs []string

	for _, p := range method.Parameters {
		sp, ca := swiftParamAndCallArg(&p, resolved)
		swiftParams = append(swiftParams, sp...)
		callArgs = append(callArgs, ca...)
	}

	paramStr := strings.Join(swiftParams, ", ")

	var swiftReturnType string
	if hasReturn {
		swiftReturnType = swiftType(method.Returns.Type, resolved)
	}

	if method.Description != "" {
		fmt.Fprintf(b, "    /// %s\n", method.Description)
	}

	switch {
	case hasError && hasReturn:
		fmt.Fprintf(b, "    public static func %s(%s) throws -> %s {\n", swiftMethodName, paramStr, swiftReturnType)
		fmt.Fprintf(b, "        var result: %s = %s\n", swiftCBridgeType(method.Returns.Type, resolved), swiftDefaultValue(method.Returns.Type))
		writeSwiftCCallPrimitive(b, funcName, callArgs, method.Parameters, "result", true, swiftErrorEnumName(method.Error), resolved)
	case hasError && !hasReturn:
		fmt.Fprintf(b, "    public static func %s(%s) throws {\n", swiftMethodName, paramStr)
		writeSwiftCCallVoid(b, funcName, callArgs, method.Parameters, true, swiftErrorEnumName(method.Error), resolved)
	case !hasError && hasReturn:
		fmt.Fprintf(b, "    public static func %s(%s) -> %s {\n", swiftMethodName, paramStr, swiftReturnType)
		writeSwiftCCallDirect(b, funcName, callArgs, method.Parameters, resolved)
	default:
		fmt.Fprintf(b, "    public static func %s(%s) {\n", swiftMethodName, paramStr)
		writeSwiftCCallVoid(b, funcName, callArgs, method.Parameters, false, "", resolved)
	}

	fmt.Fprintf(b, "    }\n\n")
}

// swiftParamAndCallArg returns the Swift parameter declaration(s) and C call argument(s)
// for a given parameter definition.
func swiftParamAndCallArg(p *model.ParameterDef, resolved resolver.ResolvedTypes) (swiftParams []string, callArgs []string) {
	paramName := ToCamelCase(p.Name)

	if model.IsString(p.Type) {
		swiftParams = append(swiftParams, paramName+": String")
		// The call arg is handled specially via withCString
		callArgs = append(callArgs, paramName)
		return
	}

	if _, ok := model.IsBuffer(p.Type); ok {
		if p.Transfer == "ref_mut" {
			swiftParams = append(swiftParams, paramName+": UnsafeMutableBufferPointer<UInt8>")
		} else {
			swiftParams = append(swiftParams, paramName+": Data")
		}
		// Buffer expands to pointer + length in the C call
		callArgs = append(callArgs, paramName, paramName+"_len")
		return
	}

	if _, ok := model.IsHandle(p.Type); ok {
		handleName, _ := model.IsHandle(p.Type)
		swiftParams = append(swiftParams, paramName+": "+handleName)
		callArgs = append(callArgs, paramName+".handle")
		return
	}

	if model.IsPrimitive(p.Type) {
		swiftParams = append(swiftParams, paramName+": "+swiftPrimitiveType(p.Type))
		callArgs = append(callArgs, paramName)
		return
	}

	// FlatBuffer type — pass as opaque pointer
	fbType := swiftFlatBufferParamType(p.Type, p.Transfer)
	swiftParams = append(swiftParams, paramName+": "+fbType)
	callArgs = append(callArgs, paramName)
	return
}

// writeSwiftCCall writes the C function call for a factory method (returns handle, with out-param).
func writeSwiftCCall(b *strings.Builder, funcName string, callArgs []string, params []model.ParameterDef, outVar string, hasError bool, errEnumName string, handleClass string, resolved resolver.ResolvedTypes) {
	// Check if we need withCString wrappers
	stringParams := collectStringParams(params)
	bufferParams := collectBufferParams(params)
	hasClosures := len(stringParams)+len(bufferParams) > 0

	indent := "        "
	closingBraces := ""
	firstClosure := true

	// closurePrefix returns "return try " or "return " for the first closure line,
	// empty string for subsequent closures.
	closurePrefix := func() string {
		if !firstClosure {
			return ""
		}
		firstClosure = false
		if hasError {
			return "return try "
		}
		return "return "
	}

	// Wrap string params in withCString
	for _, sp := range stringParams {
		paramName := ToCamelCase(sp.Name)
		fmt.Fprintf(b, "%s%s%s.withCString { %sPtr in\n", indent, closurePrefix(), paramName, paramName)
		indent += "    "
		closingBraces += indent[:len(indent)-4] + "}\n"
	}

	// Wrap buffer params
	for _, bp := range bufferParams {
		paramName := ToCamelCase(bp.Name)
		prefix := closurePrefix()
		if bp.Transfer == "ref_mut" {
			fmt.Fprintf(b, "%s%s%s.withUnsafeMutableBufferPointer { %sPtr in\n", indent, prefix, paramName, paramName)
		} else {
			fmt.Fprintf(b, "%s%s%s.withUnsafeBytes { %sPtr in\n", indent, prefix, paramName, paramName)
		}
		indent += "    "
		closingBraces += indent[:len(indent)-4] + "}\n"
	}

	// Build the actual C call arguments
	actualArgs := buildActualCallArgs(callArgs, params)
	actualArgs = append(actualArgs, "&"+outVar)
	callStr := fmt.Sprintf("%s(%s)", funcName, strings.Join(actualArgs, ", "))

	if hasError {
		fmt.Fprintf(b, "%slet code = %s\n", indent, callStr)
		fmt.Fprintf(b, "%sguard code == 0, let ptr = %s else {\n", indent, outVar)
		fmt.Fprintf(b, "%s    throw %s(rawValue: code) ?? %s.internalError\n", indent, errEnumName, errEnumName)
		fmt.Fprintf(b, "%s}\n", indent)
		if hasClosures {
			fmt.Fprintf(b, "%sreturn %s(handle: ptr)\n", indent, handleClass)
		} else {
			fmt.Fprintf(b, "%sreturn %s(handle: ptr)\n", indent, handleClass)
		}
	} else {
		fmt.Fprintf(b, "%s_ = %s\n", indent, callStr)
		if hasClosures {
			fmt.Fprintf(b, "%sreturn %s(handle: %s!)\n", indent, handleClass, outVar)
		} else {
			fmt.Fprintf(b, "%sreturn %s(handle: %s!)\n", indent, handleClass, outVar)
		}
	}

	b.WriteString(closingBraces)
}

// writeSwiftCCallPrimitive writes a C call with a primitive or FlatBuffer out-parameter.
func writeSwiftCCallPrimitive(b *strings.Builder, funcName string, callArgs []string, params []model.ParameterDef, outVar string, hasError bool, errEnumName string, resolved resolver.ResolvedTypes) {
	stringParams := collectStringParams(params)
	bufferParams := collectBufferParams(params)

	indent := "        "
	closingBraces := ""
	firstClosure := true

	closurePrefix := func() string {
		if !firstClosure {
			return ""
		}
		firstClosure = false
		if hasError {
			return "return try "
		}
		return "return "
	}

	for _, sp := range stringParams {
		paramName := ToCamelCase(sp.Name)
		fmt.Fprintf(b, "%s%s%s.withCString { %sPtr in\n", indent, closurePrefix(), paramName, paramName)
		indent += "    "
		closingBraces += indent[:len(indent)-4] + "}\n"
	}

	for _, bp := range bufferParams {
		paramName := ToCamelCase(bp.Name)
		prefix := closurePrefix()
		if bp.Transfer == "ref_mut" {
			fmt.Fprintf(b, "%s%s%s.withUnsafeMutableBufferPointer { %sPtr in\n", indent, prefix, paramName, paramName)
		} else {
			fmt.Fprintf(b, "%s%s%s.withUnsafeBytes { %sPtr in\n", indent, prefix, paramName, paramName)
		}
		indent += "    "
		closingBraces += indent[:len(indent)-4] + "}\n"
	}

	actualArgs := buildActualCallArgs(callArgs, params)
	actualArgs = append(actualArgs, "&"+outVar)
	callStr := fmt.Sprintf("%s(%s)", funcName, strings.Join(actualArgs, ", "))

	if hasError {
		fmt.Fprintf(b, "%slet code = %s\n", indent, callStr)
		fmt.Fprintf(b, "%sguard code == 0 else {\n", indent)
		fmt.Fprintf(b, "%s    throw %s(rawValue: code) ?? %s.internalError\n", indent, errEnumName, errEnumName)
		fmt.Fprintf(b, "%s}\n", indent)
		fmt.Fprintf(b, "%sreturn %s\n", indent, outVar)
	} else {
		fmt.Fprintf(b, "%s_ = %s\n", indent, callStr)
		fmt.Fprintf(b, "%sreturn %s\n", indent, outVar)
	}

	b.WriteString(closingBraces)
}

// writeSwiftCCallVoid writes a C call with no return value.
func writeSwiftCCallVoid(b *strings.Builder, funcName string, callArgs []string, params []model.ParameterDef, hasError bool, errEnumName string, resolved resolver.ResolvedTypes) {
	stringParams := collectStringParams(params)
	bufferParams := collectBufferParams(params)

	indent := "        "
	closingBraces := ""
	firstClosure := true

	// Void methods don't return, but throwing closures need `try`
	closurePrefix := func() string {
		if !firstClosure {
			return ""
		}
		firstClosure = false
		if hasError {
			return "try "
		}
		return ""
	}

	for _, sp := range stringParams {
		paramName := ToCamelCase(sp.Name)
		fmt.Fprintf(b, "%s%s%s.withCString { %sPtr in\n", indent, closurePrefix(), paramName, paramName)
		indent += "    "
		closingBraces += indent[:len(indent)-4] + "}\n"
	}

	for _, bp := range bufferParams {
		paramName := ToCamelCase(bp.Name)
		prefix := closurePrefix()
		if bp.Transfer == "ref_mut" {
			fmt.Fprintf(b, "%s%s%s.withUnsafeMutableBufferPointer { %sPtr in\n", indent, prefix, paramName, paramName)
		} else {
			fmt.Fprintf(b, "%s%s%s.withUnsafeBytes { %sPtr in\n", indent, prefix, paramName, paramName)
		}
		indent += "    "
		closingBraces += indent[:len(indent)-4] + "}\n"
	}

	actualArgs := buildActualCallArgs(callArgs, params)
	callStr := fmt.Sprintf("%s(%s)", funcName, strings.Join(actualArgs, ", "))

	if hasError {
		fmt.Fprintf(b, "%slet code = %s\n", indent, callStr)
		fmt.Fprintf(b, "%sguard code == 0 else {\n", indent)
		fmt.Fprintf(b, "%s    throw %s(rawValue: code) ?? %s.internalError\n", indent, errEnumName, errEnumName)
		fmt.Fprintf(b, "%s}\n", indent)
	} else {
		fmt.Fprintf(b, "%s%s\n", indent, callStr)
	}

	b.WriteString(closingBraces)
}

// writeSwiftCCallDirect writes a C call that directly returns its value.
func writeSwiftCCallDirect(b *strings.Builder, funcName string, callArgs []string, params []model.ParameterDef, resolved resolver.ResolvedTypes) {
	stringParams := collectStringParams(params)
	bufferParams := collectBufferParams(params)

	indent := "        "
	closingBraces := ""
	firstClosure := true

	closurePrefix := func() string {
		if !firstClosure {
			return ""
		}
		firstClosure = false
		return "return "
	}

	for _, sp := range stringParams {
		paramName := ToCamelCase(sp.Name)
		fmt.Fprintf(b, "%s%s%s.withCString { %sPtr in\n", indent, closurePrefix(), paramName, paramName)
		indent += "    "
		closingBraces += indent[:len(indent)-4] + "}\n"
	}

	for _, bp := range bufferParams {
		paramName := ToCamelCase(bp.Name)
		prefix := closurePrefix()
		if bp.Transfer == "ref_mut" {
			fmt.Fprintf(b, "%s%s%s.withUnsafeMutableBufferPointer { %sPtr in\n", indent, prefix, paramName, paramName)
		} else {
			fmt.Fprintf(b, "%s%s%s.withUnsafeBytes { %sPtr in\n", indent, prefix, paramName, paramName)
		}
		indent += "    "
		closingBraces += indent[:len(indent)-4] + "}\n"
	}

	actualArgs := buildActualCallArgs(callArgs, params)
	callStr := fmt.Sprintf("%s(%s)", funcName, strings.Join(actualArgs, ", "))

	fmt.Fprintf(b, "%sreturn %s\n", indent, callStr)

	b.WriteString(closingBraces)
}

// collectStringParams returns parameters that are string type.
func collectStringParams(params []model.ParameterDef) []model.ParameterDef {
	var result []model.ParameterDef
	for _, p := range params {
		if model.IsString(p.Type) {
			result = append(result, p)
		}
	}
	return result
}

// collectBufferParams returns parameters that are buffer<T> type.
func collectBufferParams(params []model.ParameterDef) []model.ParameterDef {
	var result []model.ParameterDef
	for _, p := range params {
		if _, ok := model.IsBuffer(p.Type); ok {
			result = append(result, p)
		}
	}
	return result
}

// buildActualCallArgs translates Swift parameter names to actual C call arguments,
// replacing string params with their withCString closure variable and buffer params
// with pointer + count.
func buildActualCallArgs(callArgs []string, params []model.ParameterDef) []string {
	// Build a map of param names to their types
	paramTypes := map[string]model.ParameterDef{}
	for _, p := range params {
		paramTypes[ToCamelCase(p.Name)] = p
	}

	var result []string
	i := 0
	for i < len(callArgs) {
		arg := callArgs[i]

		// Check if this is a string param (not handle.handle)
		if p, ok := paramTypes[arg]; ok && model.IsString(p.Type) {
			result = append(result, arg+"Ptr")
			i++
			continue
		}

		// Check if this is a buffer param
		if p, ok := paramTypes[arg]; ok {
			if _, isBuf := model.IsBuffer(p.Type); isBuf {
				paramName := arg
				if p.Transfer == "ref_mut" {
					result = append(result, paramName+"Ptr.baseAddress")
				} else {
					result = append(result, paramName+"Ptr.baseAddress!.assumingMemoryBound(to: UInt8.self)")
				}
				// Skip the _len arg and replace with count
				i++
				result = append(result, "UInt32("+paramName+".count)")
				i++
				continue
			}
		}

		result = append(result, arg)
		i++
	}
	return result
}

// isHandleReturn returns true if the return type is a handle type.
func isHandleReturn(retType string) bool {
	_, ok := model.IsHandle(retType)
	return ok
}

// swiftType returns the Swift type for an API type.
func swiftType(t string, resolved resolver.ResolvedTypes) string {
	if model.IsString(t) {
		return "String"
	}
	if _, ok := model.IsBuffer(t); ok {
		return "Data"
	}
	if handleName, ok := model.IsHandle(t); ok {
		return handleName
	}
	if model.IsPrimitive(t) {
		return swiftPrimitiveType(t)
	}
	// FlatBuffer type — use the C struct name
	return model.FlatBufferCType(t)
}

// swiftPrimitiveType converts an API primitive to its Swift equivalent.
func swiftPrimitiveType(t string) string {
	switch t {
	case "int8":
		return "Int8"
	case "int16":
		return "Int16"
	case "int32":
		return "Int32"
	case "int64":
		return "Int64"
	case "uint8":
		return "UInt8"
	case "uint16":
		return "UInt16"
	case "uint32":
		return "UInt32"
	case "uint64":
		return "UInt64"
	case "float32":
		return "Float"
	case "float64":
		return "Double"
	case "bool":
		return "Bool"
	default:
		return t
	}
}

// swiftCBridgeType returns the C bridge type for use in variable declarations.
func swiftCBridgeType(t string, resolved resolver.ResolvedTypes) string {
	if _, ok := model.IsHandle(t); ok {
		return "OpaquePointer?"
	}
	if model.IsPrimitive(t) {
		return swiftPrimitiveType(t)
	}
	// FlatBuffer type — use the C struct name
	return model.FlatBufferCType(t)
}

// swiftDefaultValue returns the default zero value for a type.
func swiftDefaultValue(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return "nil"
	}
	if model.IsPrimitive(t) {
		switch t {
		case "bool":
			return "false"
		case "float32", "float64":
			return "0.0"
		default:
			return "0"
		}
	}
	// FlatBuffer type — zero-initialize the C struct
	return model.FlatBufferCType(t) + "()"
}

// swiftFlatBufferParamType returns the Swift parameter type for a FlatBuffer parameter.
func swiftFlatBufferParamType(t string, transfer string) string {
	switch transfer {
	case "ref_mut":
		return "UnsafeMutablePointer<" + model.FlatBufferCType(t) + ">"
	case "ref":
		return "UnsafePointer<" + model.FlatBufferCType(t) + ">"
	default:
		return model.FlatBufferCType(t)
	}
}
