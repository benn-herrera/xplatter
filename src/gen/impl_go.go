package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplattergy/model"
)

func init() {
	Register("impl_go", func() Generator { return &GoImplGenerator{} })
}

// GoImplGenerator produces Go implementation scaffolding:
// an interface file, a cgo shim file, and a stub implementation file.
type GoImplGenerator struct{}

func (g *GoImplGenerator) Name() string { return "impl_go" }

func (g *GoImplGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name

	ifaceFile, err := g.generateInterface(api, apiName)
	if err != nil {
		return nil, fmt.Errorf("generating interface: %w", err)
	}

	cgoFile, err := g.generateCgo(api, apiName)
	if err != nil {
		return nil, fmt.Errorf("generating cgo shim: %w", err)
	}

	implFile, err := g.generateImpl(api, apiName)
	if err != nil {
		return nil, fmt.Errorf("generating stub impl: %w", err)
	}

	return []*OutputFile{ifaceFile, cgoFile, implFile}, nil
}

// generateInterface produces the Go interface type file.
func (g *GoImplGenerator) generateInterface(api *model.APIDefinition, apiName string) (*OutputFile, error) {
	pkgName := goPackageName(apiName)
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n", pkgName)
	b.WriteString("\n")

	for _, iface := range api.Interfaces {
		ifaceName := ToPascalCase(iface.Name)
		if iface.Description != "" {
			fmt.Fprintf(&b, "// %s %s\n", ifaceName, iface.Description)
		}
		fmt.Fprintf(&b, "type %s interface {\n", ifaceName)
		for _, method := range iface.Methods {
			writeGoInterfaceMethod(&b, &method)
		}
		b.WriteString("}\n\n")
	}

	filename := apiName + "_interface.go"
	return &OutputFile{Path: filename, Content: []byte(b.String())}, nil
}

// generateCgo produces the cgo shim file with //export functions.
func (g *GoImplGenerator) generateCgo(api *model.APIDefinition, apiName string) (*OutputFile, error) {
	pkgName := goPackageName(apiName)
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n", pkgName)
	b.WriteString("\n")

	// cgo preamble
	fmt.Fprintf(&b, "/*\n")
	fmt.Fprintf(&b, "#include \"%s.h\"\n", apiName)
	fmt.Fprintf(&b, "*/\n")
	b.WriteString("import \"C\"\n")
	b.WriteString("\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"sync\"\n")
	b.WriteString("\t\"unsafe\"\n")
	b.WriteString(")\n")
	b.WriteString("\n")

	// Handle map for mapping integer keys to Go interface implementations.
	b.WriteString("// handles maps integer keys to Go interface implementations.\n")
	b.WriteString("var handles sync.Map\n")
	b.WriteString("\n")

	// Export functions for each interface method.
	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "/* %s */\n\n", iface.Name)
		for _, method := range iface.Methods {
			writeCgoExportFunc(&b, apiName, iface.Name, &method)
			b.WriteString("\n")
		}
	}

	filename := apiName + "_cgo.go"
	return &OutputFile{Path: filename, Content: []byte(b.String())}, nil
}

// generateImpl produces the stub implementation file.
func (g *GoImplGenerator) generateImpl(api *model.APIDefinition, apiName string) (*OutputFile, error) {
	pkgName := goPackageName(apiName)
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n", pkgName)
	b.WriteString("\n")

	for _, iface := range api.Interfaces {
		ifaceName := ToPascalCase(iface.Name)
		structName := ifaceName + "Impl"

		fmt.Fprintf(&b, "// %s is a stub implementation of %s.\n", structName, ifaceName)
		fmt.Fprintf(&b, "type %s struct{}\n\n", structName)

		// Verify interface satisfaction.
		fmt.Fprintf(&b, "var _ %s = (*%s)(nil)\n\n", ifaceName, structName)

		for _, method := range iface.Methods {
			writeGoStubMethod(&b, structName, &method)
			b.WriteString("\n")
		}
	}

	filename := apiName + "_impl.go"
	return &OutputFile{Path: filename, Content: []byte(b.String())}, nil
}

// writeGoInterfaceMethod writes a single method signature to the interface definition.
func writeGoInterfaceMethod(b *strings.Builder, method *model.MethodDef) {
	methodName := ToPascalCase(method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build parameter list
	var params []string
	for _, p := range method.Parameters {
		params = append(params, goParamSignature(&p))
	}
	paramStr := strings.Join(params, ", ")

	// Determine return signature
	var retSig string
	switch {
	case hasError && hasReturn:
		retSig = fmt.Sprintf("(%s, error)", goReturnType(method.Returns.Type))
	case hasError && !hasReturn:
		retSig = "error"
	case !hasError && hasReturn:
		retSig = goReturnType(method.Returns.Type)
	default:
		retSig = ""
	}

	if retSig != "" {
		fmt.Fprintf(b, "\t%s(%s) %s\n", methodName, paramStr, retSig)
	} else {
		fmt.Fprintf(b, "\t%s(%s)\n", methodName, paramStr)
	}
}

// writeCgoExportFunc writes an //export annotated cgo function.
func writeCgoExportFunc(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build C parameter list
	var cParams []string
	for _, p := range method.Parameters {
		cParams = append(cParams, goCgoParam(&p)...)
	}

	// Determine C return type and out-parameter
	var cReturnType string
	switch {
	case hasError && hasReturn:
		cReturnType = "C.int32_t"
		cParams = append(cParams, "out_result *C."+cgoType(method.Returns.Type))
	case hasError && !hasReturn:
		cReturnType = "C.int32_t"
	case !hasError && hasReturn:
		cReturnType = "C." + cgoType(method.Returns.Type)
	default:
		cReturnType = ""
	}

	// Write //export annotation
	fmt.Fprintf(b, "//export %s\n", funcName)

	// Write function signature
	paramStr := strings.Join(cParams, ", ")
	if cReturnType != "" {
		fmt.Fprintf(b, "func %s(%s) %s {\n", funcName, paramStr, cReturnType)
	} else {
		fmt.Fprintf(b, "func %s(%s) {\n", funcName, paramStr)
	}

	// Body: parameter conversions and handle lookup
	writeCgoFuncBody(b, ifaceName, method)

	b.WriteString("}\n")
}

// writeCgoFuncBody writes the body of a cgo export function.
func writeCgoFuncBody(b *strings.Builder, ifaceName string, method *model.MethodDef) {
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Find handle parameter for interface lookup (first handle param, if any)
	var handleParam *model.ParameterDef
	for i := range method.Parameters {
		if _, ok := model.IsHandle(method.Parameters[i].Type); ok {
			handleParam = &method.Parameters[i]
			break
		}
	}

	// Convert parameters
	for _, p := range method.Parameters {
		if model.IsString(p.Type) {
			fmt.Fprintf(b, "\t%sGo := C.GoString(%s)\n", ToCamelCase(p.Name), p.Name)
		}
		if elemType, ok := model.IsBuffer(p.Type); ok {
			goElemType := primitiveGoType(elemType)
			fmt.Fprintf(b, "\t%sSlice := unsafe.Slice((*%s)(unsafe.Pointer(%s)), %s_len)\n",
				ToCamelCase(p.Name), goElemType, p.Name, p.Name)
		}
	}

	// Handle lookup
	goIfaceName := ToPascalCase(ifaceName)
	if handleParam != nil {
		fmt.Fprintf(b, "\timpl, ok := handles.Load(uintptr(%s))\n", handleParam.Name)
		b.WriteString("\tif !ok {\n")
		if hasError {
			b.WriteString("\t\treturn -1\n")
		} else {
			b.WriteString("\t\treturn\n")
		}
		b.WriteString("\t}\n")
		fmt.Fprintf(b, "\t_ = impl.(%s)\n", goIfaceName)
	}

	// Call placeholder
	b.WriteString("\t// TODO: implement\n")

	// Return
	switch {
	case hasError && hasReturn:
		b.WriteString("\treturn 0\n")
	case hasError && !hasReturn:
		b.WriteString("\treturn 0\n")
	case !hasError && hasReturn:
		fmt.Fprintf(b, "\treturn 0\n")
	default:
		// void — no return
	}
}

// writeGoStubMethod writes a stub method on the impl struct.
func writeGoStubMethod(b *strings.Builder, structName string, method *model.MethodDef) {
	methodName := ToPascalCase(method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build parameter list
	var params []string
	for _, p := range method.Parameters {
		params = append(params, goParamSignature(&p))
	}
	paramStr := strings.Join(params, ", ")

	// Determine return signature
	var retSig string
	switch {
	case hasError && hasReturn:
		retSig = fmt.Sprintf("(%s, error)", goReturnType(method.Returns.Type))
	case hasError && !hasReturn:
		retSig = "error"
	case !hasError && hasReturn:
		retSig = goReturnType(method.Returns.Type)
	default:
		retSig = ""
	}

	if retSig != "" {
		fmt.Fprintf(b, "func (s *%s) %s(%s) %s {\n", structName, methodName, paramStr, retSig)
	} else {
		fmt.Fprintf(b, "func (s *%s) %s(%s) {\n", structName, methodName, paramStr)
	}

	b.WriteString("\t// TODO: implement\n")

	// Return zero values
	switch {
	case hasError && hasReturn:
		zeroVal := goZeroValue(method.Returns.Type)
		fmt.Fprintf(b, "\treturn %s, nil\n", zeroVal)
	case hasError && !hasReturn:
		b.WriteString("\treturn nil\n")
	case !hasError && hasReturn:
		zeroVal := goZeroValue(method.Returns.Type)
		fmt.Fprintf(b, "\treturn %s\n", zeroVal)
	default:
		// void — no return
	}

	b.WriteString("}\n")
}

// goPackageName converts an api_name to a valid Go package name (lowercase, no underscores).
func goPackageName(apiName string) string {
	return strings.ReplaceAll(apiName, "_", "")
}

// goParamSignature returns the Go parameter as "name type" for an interface method signature.
func goParamSignature(p *model.ParameterDef) string {
	name := ToCamelCase(p.Name)
	goType := goParamType(p.Type)
	return name + " " + goType
}

// goParamType returns the Go type for a parameter type.
func goParamType(t string) string {
	if model.IsString(t) {
		return "string"
	}
	if elemType, ok := model.IsBuffer(t); ok {
		return "[]" + primitiveGoType(elemType)
	}
	if _, ok := model.IsHandle(t); ok {
		return "uintptr"
	}
	if model.IsPrimitive(t) {
		return primitiveGoType(t)
	}
	// FlatBuffer type — pass as byte slice
	return "[]byte"
}

// goReturnType returns the Go type for a return value.
func goReturnType(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return "uintptr"
	}
	if model.IsPrimitive(t) {
		return primitiveGoType(t)
	}
	// FlatBuffer type — return as byte slice
	return "[]byte"
}

// goZeroValue returns the Go zero value for a return type.
func goZeroValue(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return "0"
	}
	if model.IsPrimitive(t) {
		switch t {
		case "bool":
			return "false"
		case "float32", "float64":
			return "0"
		default:
			return "0"
		}
	}
	// FlatBuffer type ([]byte)
	return "nil"
}

// primitiveGoType maps xplattergy primitive types to Go types.
func primitiveGoType(t string) string {
	switch t {
	case "int8":
		return "int8"
	case "int16":
		return "int16"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "uint8":
		return "uint8"
	case "uint16":
		return "uint16"
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "float32":
		return "float32"
	case "float64":
		return "float64"
	case "bool":
		return "bool"
	default:
		return t
	}
}

// goCgoParam returns one or more cgo parameter strings for a C ABI parameter.
func goCgoParam(p *model.ParameterDef) []string {
	if model.IsString(p.Type) {
		return []string{p.Name + " *C.char"}
	}
	if elemType, ok := model.IsBuffer(p.Type); ok {
		cType := model.PrimitiveCType(elemType)
		return []string{
			p.Name + " *C." + cType,
			p.Name + "_len C.uint32_t",
		}
	}
	if handleName, ok := model.IsHandle(p.Type); ok {
		return []string{p.Name + " C." + HandleTypedefName(handleName)}
	}
	if model.IsPrimitive(p.Type) {
		return []string{p.Name + " C." + model.PrimitiveCType(p.Type)}
	}
	// FlatBuffer type
	cType := model.FlatBufferCType(p.Type)
	if p.Transfer == "ref_mut" {
		return []string{p.Name + " *C." + cType}
	}
	if p.Transfer == "ref" {
		return []string{p.Name + " *C." + cType}
	}
	return []string{p.Name + " C." + cType}
}

// cgoType returns the cgo type name for a return type.
func cgoType(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return HandleTypedefName(t[len("handle:"):])
	}
	if model.IsPrimitive(t) {
		return model.PrimitiveCType(t)
	}
	return model.FlatBufferCType(t)
}
