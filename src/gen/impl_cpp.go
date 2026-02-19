package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
)

func init() {
	Register("impl_cpp", func() Generator { return &ImplCppGenerator{} })
}

// ImplCppGenerator produces C++ implementation scaffolding:
//   - An abstract interface header with pure virtual methods
//   - A C ABI shim that bridges extern "C" functions to the interface
//   - A stub implementation header and source file
type ImplCppGenerator struct{}

func (g *ImplCppGenerator) Name() string { return "impl_cpp" }

func (g *ImplCppGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name

	ifaceFile, err := g.generateInterface(api, apiName)
	if err != nil {
		return nil, fmt.Errorf("generating interface: %w", err)
	}

	shimFile, err := g.generateShim(api, apiName)
	if err != nil {
		return nil, fmt.Errorf("generating shim: %w", err)
	}

	implHeader, err := g.generateImplHeader(api, apiName)
	if err != nil {
		return nil, fmt.Errorf("generating impl header: %w", err)
	}

	implSource, err := g.generateImplSource(api, apiName)
	if err != nil {
		return nil, fmt.Errorf("generating impl source: %w", err)
	}

	cmakeFile := g.generateCMakeLists(api, apiName)

	return []*OutputFile{ifaceFile, shimFile, implHeader, implSource, cmakeFile}, nil
}

// generateInterface produces the abstract C++ interface header.
func (g *ImplCppGenerator) generateInterface(api *model.APIDefinition, apiName string) (*OutputFile, error) {
	className := ToPascalCase(apiName) + "Interface"
	guardName := UpperSnakeCase(apiName) + "_INTERFACE_H"

	var b strings.Builder

	// Include guard
	fmt.Fprintf(&b, "#ifndef %s\n", guardName)
	fmt.Fprintf(&b, "#define %s\n\n", guardName)

	// Standard includes
	b.WriteString("#include <stdint.h>\n")
	b.WriteString("#include <stdbool.h>\n")
	b.WriteString("#include <cstddef>\n")
	b.WriteString("#include <string_view>\n")
	b.WriteString("#include <span>\n")
	// Include the C header for FlatBuffer type definitions
	fmt.Fprintf(&b, "#include \"%s.h\"\n\n", apiName)

	// Abstract class
	fmt.Fprintf(&b, "class %s {\n", className)
	b.WriteString("public:\n")
	fmt.Fprintf(&b, "    virtual ~%s() = default;\n\n", className)

	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "    /* %s */\n", iface.Name)
		for _, method := range iface.Methods {
			g.writeInterfaceMethod(&b, &method)
		}
		b.WriteString("\n")
	}

	b.WriteString("};\n\n")

	// Factory function declaration
	fmt.Fprintf(&b, "// Factory function — implement this to return your concrete instance.\n")
	fmt.Fprintf(&b, "%s* create_%s_instance();\n\n", className, apiName)

	fmt.Fprintf(&b, "#endif\n")

	return &OutputFile{
		Path:    apiName + "_interface.h",
		Content: []byte(b.String()),
	}, nil
}

// writeInterfaceMethod writes a single pure virtual method declaration.
func (g *ImplCppGenerator) writeInterfaceMethod(b *strings.Builder, method *model.MethodDef) {
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Determine return type
	var returnType string
	switch {
	case hasError:
		returnType = "int32_t"
	case hasReturn:
		returnType = cppReturnType(method.Returns.Type)
	default:
		returnType = "void"
	}

	// Build parameter list
	var params []string
	for _, p := range method.Parameters {
		params = append(params, formatCppParam(&p)...)
	}

	// If fallible with return, add out-parameter
	if hasError && hasReturn {
		params = append(params, cppOutParamType(method.Returns.Type)+" out_result")
	}

	paramStr := strings.Join(params, ", ")

	fmt.Fprintf(b, "    virtual %s %s(%s) = 0;\n", returnType, method.Name, paramStr)
}

// generateShim produces the C ABI shim source file.
func (g *ImplCppGenerator) generateShim(api *model.APIDefinition, apiName string) (*OutputFile, error) {
	className := ToPascalCase(apiName) + "Interface"

	var b strings.Builder

	// Includes
	fmt.Fprintf(&b, "#include \"%s_interface.h\"\n", apiName)
	fmt.Fprintf(&b, "#include \"%s.h\"\n\n", apiName)

	b.WriteString("extern \"C\" {\n\n")

	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "/* %s */\n", iface.Name)
		for _, method := range iface.Methods {
			g.writeShimFunction(&b, api, apiName, iface.Name, className, &method)
			b.WriteString("\n")
		}
	}

	b.WriteString("} // extern \"C\"\n")

	return &OutputFile{
		Path:    apiName + "_shim.cpp",
		Content: []byte(b.String()),
	}, nil
}

// writeShimFunction writes a single extern "C" function that delegates to the interface.
func (g *ImplCppGenerator) writeShimFunction(b *strings.Builder, api *model.APIDefinition, apiName, ifaceName, className string, method *model.MethodDef) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Check if this is a create method: returns a handle, is fallible, and has
	// no handle input parameters (pure factory — no existing handle to delegate through).
	isCreate := false
	isDestroy := false
	var handleName string
	if hasReturn {
		if hn, ok := model.IsHandle(method.Returns.Type); ok {
			if hasError {
				hasHandleInput := false
				for _, p := range method.Parameters {
					if _, ok := model.IsHandle(p.Type); ok {
						hasHandleInput = true
						break
					}
				}
				if !hasHandleInput {
					isCreate = true
					handleName = hn
				}
			}
		}
	}

	// Check if this is a destroy method: takes a handle param, void return, no error
	if !hasError && !hasReturn && len(method.Parameters) >= 1 {
		if hn, ok := model.IsHandle(method.Parameters[0].Type); ok {
			if strings.HasPrefix(method.Name, "destroy") {
				isDestroy = true
				handleName = hn
			}
		}
	}

	// Build C parameter list
	var cParams []string
	for _, p := range method.Parameters {
		cParams = append(cParams, formatCParam(&p)...)
	}

	// Determine C return type and out-param
	var returnType string
	switch {
	case hasError && hasReturn:
		returnType = "int32_t"
		cParams = append(cParams, COutParamType(method.Returns.Type)+" out_result")
	case hasError && !hasReturn:
		returnType = "int32_t"
	case !hasError && hasReturn:
		returnType = CReturnType(method.Returns.Type)
	default:
		returnType = "void"
	}

	cParamStr := strings.Join(cParams, ", ")
	if cParamStr == "" {
		cParamStr = "void"
	}

	exportMacro := ExportMacroName(apiName)
	fmt.Fprintf(b, "%s %s %s(%s) {\n", exportMacro, returnType, funcName, cParamStr)

	if isCreate {
		// Create method: call factory, cast to handle, store in out_result
		_ = handleName
		fmt.Fprintf(b, "    %s* instance = create_%s_instance();\n", className, apiName)
		fmt.Fprintf(b, "    if (!instance) {\n")
		fmt.Fprintf(b, "        return -1;\n")
		fmt.Fprintf(b, "    }\n")
		fmt.Fprintf(b, "    *out_result = reinterpret_cast<%s>(instance);\n", HandleTypedefName(handleName))
		fmt.Fprintf(b, "    return 0;\n")
	} else if isDestroy {
		// Destroy method: cast handle back to interface, delete
		paramName := method.Parameters[0].Name
		fmt.Fprintf(b, "    %s* instance = reinterpret_cast<%s*>(%s);\n", className, className, paramName)
		fmt.Fprintf(b, "    delete instance;\n")
	} else {
		// Regular method: find the handle parameter, cast it, and call the method
		g.writeShimDelegation(b, className, method)
	}

	b.WriteString("}\n")
}

// writeShimDelegation writes the body of a regular shim function that delegates to the interface.
func (g *ImplCppGenerator) writeShimDelegation(b *strings.Builder, className string, method *model.MethodDef) {
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Find the handle parameter (first handle param) to cast to the interface
	var handleParam *model.ParameterDef
	for i := range method.Parameters {
		if _, ok := model.IsHandle(method.Parameters[i].Type); ok {
			handleParam = &method.Parameters[i]
			break
		}
	}

	if handleParam == nil {
		// No handle parameter; this shouldn't normally happen for non-create methods,
		// but generate a placeholder comment.
		fmt.Fprintf(b, "    // TODO: no handle parameter found — implement manually\n")
		if hasError {
			fmt.Fprintf(b, "    return 0;\n")
		}
		return
	}

	// Cast handle to interface pointer
	fmt.Fprintf(b, "    %s* self = reinterpret_cast<%s*>(%s);\n", className, className, handleParam.Name)

	// Build the call arguments (handle param passes through as void*)
	var callArgs []string
	for _, p := range method.Parameters {
		if model.IsString(p.Type) {
			callArgs = append(callArgs, fmt.Sprintf("std::string_view(%s)", p.Name))
		} else if _, ok := model.IsBuffer(p.Type); ok {
			callArgs = append(callArgs, fmt.Sprintf("std::span(%s, %s_len)", p.Name, p.Name))
		} else {
			callArgs = append(callArgs, p.Name)
		}
	}

	// Add out_result if fallible with return
	if hasError && hasReturn {
		callArgs = append(callArgs, "out_result")
	}

	argStr := strings.Join(callArgs, ", ")

	switch {
	case hasError:
		fmt.Fprintf(b, "    return self->%s(%s);\n", method.Name, argStr)
	case hasReturn:
		fmt.Fprintf(b, "    return self->%s(%s);\n", method.Name, argStr)
	default:
		fmt.Fprintf(b, "    self->%s(%s);\n", method.Name, argStr)
	}
}

// generateImplHeader produces the stub implementation header.
func (g *ImplCppGenerator) generateImplHeader(api *model.APIDefinition, apiName string) (*OutputFile, error) {
	ifaceClassName := ToPascalCase(apiName) + "Interface"
	implClassName := ToPascalCase(apiName) + "Impl"
	guardName := UpperSnakeCase(apiName) + "_IMPL_H"

	var b strings.Builder

	// Include guard
	fmt.Fprintf(&b, "#ifndef %s\n", guardName)
	fmt.Fprintf(&b, "#define %s\n\n", guardName)

	fmt.Fprintf(&b, "#include \"%s_interface.h\"\n\n", apiName)

	// Concrete class
	fmt.Fprintf(&b, "class %s : public %s {\n", implClassName, ifaceClassName)
	b.WriteString("public:\n")
	fmt.Fprintf(&b, "    %s();\n", implClassName)
	fmt.Fprintf(&b, "    ~%s() override;\n\n", implClassName)

	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "    /* %s */\n", iface.Name)
		for _, method := range iface.Methods {
			g.writeImplMethodDecl(&b, &method)
		}
		b.WriteString("\n")
	}

	b.WriteString("};\n\n")
	fmt.Fprintf(&b, "#endif\n")

	return &OutputFile{
		Path:     apiName + "_impl.h",
		Content:  []byte(b.String()),
		Scaffold: true,
	}, nil
}

// writeImplMethodDecl writes a method declaration with override.
func (g *ImplCppGenerator) writeImplMethodDecl(b *strings.Builder, method *model.MethodDef) {
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	var returnType string
	switch {
	case hasError:
		returnType = "int32_t"
	case hasReturn:
		returnType = cppReturnType(method.Returns.Type)
	default:
		returnType = "void"
	}

	var params []string
	for _, p := range method.Parameters {
		params = append(params, formatCppParam(&p)...)
	}

	if hasError && hasReturn {
		params = append(params, cppOutParamType(method.Returns.Type)+" out_result")
	}

	paramStr := strings.Join(params, ", ")

	fmt.Fprintf(b, "    %s %s(%s) override;\n", returnType, method.Name, paramStr)
}

// generateImplSource produces the stub implementation source file.
func (g *ImplCppGenerator) generateImplSource(api *model.APIDefinition, apiName string) (*OutputFile, error) {
	ifaceClassName := ToPascalCase(apiName) + "Interface"
	implClassName := ToPascalCase(apiName) + "Impl"

	var b strings.Builder

	fmt.Fprintf(&b, "#include \"%s_impl.h\"\n\n", apiName)

	// Constructor / destructor
	fmt.Fprintf(&b, "%s::%s() {\n", implClassName, implClassName)
	b.WriteString("    // TODO: implement\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "%s::~%s() {\n", implClassName, implClassName)
	b.WriteString("    // TODO: implement\n")
	b.WriteString("}\n\n")

	// Method stubs
	for _, iface := range api.Interfaces {
		for _, method := range iface.Methods {
			g.writeImplMethodStub(&b, implClassName, &method)
			b.WriteString("\n")
		}
	}

	// Factory function
	fmt.Fprintf(&b, "// Factory function — returns a new instance of the implementation.\n")
	fmt.Fprintf(&b, "%s* create_%s_instance() {\n", ifaceClassName, apiName)
	fmt.Fprintf(&b, "    return new %s();\n", implClassName)
	b.WriteString("}\n")

	return &OutputFile{
		Path:     apiName + "_impl.cpp",
		Content:  []byte(b.String()),
		Scaffold: true,
	}, nil
}

// writeImplMethodStub writes a stub method body.
func (g *ImplCppGenerator) writeImplMethodStub(b *strings.Builder, implClassName string, method *model.MethodDef) {
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	var returnType string
	switch {
	case hasError:
		returnType = "int32_t"
	case hasReturn:
		returnType = cppReturnType(method.Returns.Type)
	default:
		returnType = "void"
	}

	var params []string
	for _, p := range method.Parameters {
		params = append(params, formatCppParam(&p)...)
	}

	if hasError && hasReturn {
		params = append(params, cppOutParamType(method.Returns.Type)+" out_result")
	}

	paramStr := strings.Join(params, ", ")

	fmt.Fprintf(b, "%s %s::%s(%s) {\n", returnType, implClassName, method.Name, paramStr)
	b.WriteString("    // TODO: implement\n")

	switch {
	case hasError:
		fmt.Fprintf(b, "    return 0;\n")
	case hasReturn:
		fmt.Fprintf(b, "    return {};\n")
	}

	b.WriteString("}\n")
}

// generateCMakeLists produces a scaffold CMakeLists.txt for the C++ implementation.
func (g *ImplCppGenerator) generateCMakeLists(api *model.APIDefinition, apiName string) *OutputFile {
	projectName := strings.ReplaceAll(apiName, "_", "-")
	var b strings.Builder

	b.WriteString("cmake_minimum_required(VERSION 3.15)\n")
	fmt.Fprintf(&b, "project(%s VERSION %s LANGUAGES CXX)\n\n", projectName, api.API.Version)
	b.WriteString("set(CMAKE_CXX_STANDARD 20)\n")
	b.WriteString("set(CMAKE_CXX_STANDARD_REQUIRED ON)\n\n")

	// Collect source files
	fmt.Fprintf(&b, "add_library(%s SHARED\n", projectName)
	fmt.Fprintf(&b, "    %s_shim.cpp\n", apiName)
	fmt.Fprintf(&b, "    %s_impl.cpp\n", apiName)
	b.WriteString(")\n\n")

	fmt.Fprintf(&b, "target_include_directories(%s PRIVATE ${CMAKE_CURRENT_SOURCE_DIR})\n", projectName)

	return &OutputFile{
		Path:     "CMakeLists.txt",
		Content:  []byte(b.String()),
		Scaffold: true,
	}
}

// --- C++ type helpers ---

// formatCppParam formats a parameter for C++ interface methods.
// Strings become std::string_view, buffers become std::span<const T>.
func formatCppParam(p *model.ParameterDef) []string {
	if model.IsString(p.Type) {
		return []string{"std::string_view " + p.Name}
	}

	if elemType, ok := model.IsBuffer(p.Type); ok {
		cppType := cppPrimitiveType(elemType)
		if p.Transfer == "ref_mut" {
			return []string{fmt.Sprintf("std::span<%s> %s", cppType, p.Name)}
		}
		return []string{fmt.Sprintf("std::span<const %s> %s", cppType, p.Name)}
	}

	if handleName, ok := model.IsHandle(p.Type); ok {
		// Handles in the C++ interface stay as opaque pointers (void*)
		// The shim layer handles the cast.
		_ = handleName
		return []string{"void* " + p.Name}
	}

	if model.IsPrimitive(p.Type) {
		return []string{cppPrimitiveType(p.Type) + " " + p.Name}
	}

	// FlatBuffer type — pass as const pointer
	cType := model.FlatBufferCType(p.Type)
	if p.Transfer == "ref_mut" {
		return []string{cType + "* " + p.Name}
	}
	return []string{"const " + cType + "* " + p.Name}
}

// cppReturnType returns the C++ type for a return value.
func cppReturnType(retType string) string {
	if handleName, ok := model.IsHandle(retType); ok {
		_ = handleName
		return "void*"
	}
	if model.IsPrimitive(retType) {
		return cppPrimitiveType(retType)
	}
	return model.FlatBufferCType(retType)
}

// cppOutParamType returns the C++ out-parameter type (pointer to the return type).
func cppOutParamType(retType string) string {
	return cppReturnType(retType) + "*"
}

// cppPrimitiveType maps xplatter primitive types to C++ fixed-width types.
func cppPrimitiveType(t string) string {
	// Same as C — stdint.h types
	return model.PrimitiveCType(t)
}
