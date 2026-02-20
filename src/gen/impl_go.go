package gen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

func init() {
	Register("impl_go", func() Generator { return &GoImplGenerator{} })
}

// GoImplGenerator produces Go implementation scaffolding:
// an interface file, a cgo shim file, a stub implementation file,
// type definitions, and a go.mod.
//
// The generated interface excludes lifecycle methods (create/destroy) and
// removes handle parameters — the shim manages handle↔impl mapping so
// the user implements business logic once without FFI concerns.
type GoImplGenerator struct{}

func (g *GoImplGenerator) Name() string { return "impl_go" }

func (g *GoImplGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name

	genHeader := GeneratedFileHeader(ctx, "//", false)
	scaffoldHeader := GeneratedFileHeader(ctx, "//", true)

	ifaceFile, err := g.generateInterface(api, apiName, ctx.ResolvedTypes)
	if err != nil {
		return nil, fmt.Errorf("generating interface: %w", err)
	}
	ifaceFile.Content = prependHeader(genHeader, ifaceFile.Content)

	cgoFile, err := g.generateCgo(api, apiName, ctx)
	if err != nil {
		return nil, fmt.Errorf("generating cgo shim: %w", err)
	}
	cgoFile.Content = prependHeader(genHeader, cgoFile.Content)

	implFile, err := g.generateImpl(api, apiName, ctx.ResolvedTypes)
	if err != nil {
		return nil, fmt.Errorf("generating stub impl: %w", err)
	}
	implFile.Content = prependHeader(scaffoldHeader, implFile.Content)

	files := []*OutputFile{ifaceFile, cgoFile, implFile}

	if len(ctx.ResolvedTypes) > 0 {
		typesFile := g.generateTypes(ctx.ResolvedTypes, apiName, api)
		typesFile.Content = prependHeader(genHeader, typesFile.Content)
		files = append(files, typesFile)
	}

	goModFile := g.generateGoMod(api)
	goModFile.Content = prependHeader(scaffoldHeader, goModFile.Content)
	files = append(files, goModFile)

	gitignoreFile := g.generateGitignore(apiName)
	files = append(files, gitignoreFile)

	return files, nil
}

// --- Lifecycle detection helpers ---

// isGoCreateMethod detects a create/factory method: returns a handle, is fallible,
// and has no handle input parameters.
func isGoCreateMethod(method *model.MethodDef) (handleName string, ok bool) {
	if method.Returns == nil || method.Error == "" {
		return "", false
	}
	hn, isHandle := model.IsHandle(method.Returns.Type)
	if !isHandle {
		return "", false
	}
	for _, p := range method.Parameters {
		if _, ok := model.IsHandle(p.Type); ok {
			return "", false
		}
	}
	return hn, true
}

// isGoDestroyMethod detects a destroy method: single handle parameter, void return, no error.
func isGoDestroyMethod(method *model.MethodDef) (handleParamName string, ok bool) {
	if method.Error != "" || method.Returns != nil {
		return "", false
	}
	if len(method.Parameters) != 1 {
		return "", false
	}
	p := method.Parameters[0]
	_, isHandle := model.IsHandle(p.Type)
	if !isHandle {
		return "", false
	}
	return p.Name, true
}

// isGoLifecycleMethod returns true if the method is a create or destroy lifecycle method.
func isGoLifecycleMethod(method *model.MethodDef) bool {
	_, isCreate := isGoCreateMethod(method)
	_, isDestroy := isGoDestroyMethod(method)
	return isCreate || isDestroy
}

// goReturnStructName returns the Go struct name for a FlatBuffer return type.
// e.g., "Hello.Greeting" -> "HelloGreeting"
func goReturnStructName(t string) string {
	return strings.ReplaceAll(t, ".", "")
}

// goReturnStructType returns the Go type name for a method return type,
// using generated Go structs for FlatBuffer types.
func goReturnStructType(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return "uintptr"
	}
	if model.IsPrimitive(t) {
		return primitiveGoType(t)
	}
	// FlatBuffer type — use generated Go struct
	return goReturnStructName(t)
}

// goReturnStructZeroValue returns the zero value for a Go return struct type.
func goReturnStructZeroValue(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return "0"
	}
	if model.IsPrimitive(t) {
		switch t {
		case "bool":
			return "false"
		default:
			return "0"
		}
	}
	// FlatBuffer struct — zero value is the struct name with empty fields
	return goReturnStructName(t) + "{}"
}

// --- Interface generation ---

// generateInterface produces the Go interface type file.
// Lifecycle methods (create/destroy) are excluded — the shim handles them.
// Handle parameters are removed — the shim resolves handles to impl instances.
func (g *GoImplGenerator) generateInterface(api *model.APIDefinition, apiName string, resolved resolver.ResolvedTypes) (*OutputFile, error) {
	pkgName := goPackageName(apiName)
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n", pkgName)
	b.WriteString("\n")

	for _, iface := range api.Interfaces {
		// Collect non-lifecycle methods for this interface
		var methods []model.MethodDef
		for _, method := range iface.Methods {
			if !isGoLifecycleMethod(&method) {
				methods = append(methods, method)
			}
		}
		if len(methods) == 0 {
			continue
		}

		ifaceName := ToPascalCase(iface.Name)
		if iface.Description != "" {
			fmt.Fprintf(&b, "// %s %s\n", ifaceName, iface.Description)
		}
		fmt.Fprintf(&b, "type %s interface {\n", ifaceName)
		for _, method := range methods {
			writeGoInterfaceMethod(&b, &method, resolved)
		}
		b.WriteString("}\n\n")
	}

	filename := apiName + "_interface.go"
	return &OutputFile{Path: filename, Content: []byte(b.String())}, nil
}

// writeGoInterfaceMethod writes a single method signature to the interface definition.
// Handle parameters are excluded (the shim resolves handles to impl instances).
func writeGoInterfaceMethod(b *strings.Builder, method *model.MethodDef, resolved resolver.ResolvedTypes) {
	methodName := ToPascalCase(method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build parameter list, excluding handle parameters
	var params []string
	for _, p := range method.Parameters {
		if _, ok := model.IsHandle(p.Type); ok {
			continue // shim handles handle→impl lookup
		}
		params = append(params, goInterfaceParamSignature(&p, resolved))
	}
	paramStr := strings.Join(params, ", ")

	// Determine return signature
	var retSig string
	switch {
	case hasError && hasReturn:
		retSig = fmt.Sprintf("(%s, error)", goReturnStructType(method.Returns.Type))
	case hasError && !hasReturn:
		retSig = "error"
	case !hasError && hasReturn:
		retSig = goReturnStructType(method.Returns.Type)
	default:
		retSig = ""
	}

	if retSig != "" {
		fmt.Fprintf(b, "\t%s(%s) %s\n", methodName, paramStr, retSig)
	} else {
		fmt.Fprintf(b, "\t%s(%s)\n", methodName, paramStr)
	}
}

// goInterfaceParamSignature returns a Go parameter as "name type" for an interface method.
func goInterfaceParamSignature(p *model.ParameterDef, resolved resolver.ResolvedTypes) string {
	name := ToCamelCase(p.Name)
	goType := goInterfaceParamType(p.Type, resolved)
	return name + " " + goType
}

// goInterfaceParamType returns the Go type for an interface parameter.
func goInterfaceParamType(t string, resolved resolver.ResolvedTypes) string {
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
	// FlatBuffer type — use generated Go struct
	return goReturnStructName(t)
}

// --- Cgo shim generation ---

// generateCgo produces the cgo shim file with //export functions.
// The cgo preamble emits local C type definitions instead of #include-ing
// the generated C header, because //export generates C prototypes that
// conflict with the header's declarations.
func (g *GoImplGenerator) generateCgo(api *model.APIDefinition, apiName string, ctx *Context) (*OutputFile, error) {
	pkgName := goPackageName(apiName)
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n", pkgName)
	b.WriteString("\n")

	// cgo preamble — local C typedefs instead of #include (avoids prototype conflicts)
	fmt.Fprintf(&b, "/*\n")
	b.WriteString("#include <stdint.h>\n")
	b.WriteString("#include <stdbool.h>\n")
	b.WriteString("#include <stdlib.h>\n")
	b.WriteString("\n")
	WriteCTypedefs(&b, api.Handles, ctx.ResolvedTypes)
	fmt.Fprintf(&b, "*/\n")
	b.WriteString("import \"C\"\n")
	b.WriteString("\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"sync\"\n")
	b.WriteString("\t\"sync/atomic\"\n")
	b.WriteString("\t\"unsafe\"\n")
	b.WriteString(")\n")
	b.WriteString("\n")

	// Handle management
	writeCgoHandleHelpers(&b)

	// Export functions for each interface method.
	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "/* %s */\n\n", iface.Name)
		for _, method := range iface.Methods {
			writeCgoExportFunc(&b, apiName, iface.Name, &method, ctx.ResolvedTypes)
			b.WriteString("\n")
		}
	}

	filename := apiName + "_cgo.go"
	return &OutputFile{Path: filename, Content: []byte(b.String())}, nil
}

// writeCgoHandleHelpers writes the handle allocation, lookup, and free helpers for the cgo shim.
func writeCgoHandleHelpers(b *strings.Builder) {
	b.WriteString("// Handle management: maps integer keys to Go interface implementations.\n")
	b.WriteString("var (\n")
	b.WriteString("\t_handles    sync.Map\n")
	b.WriteString("\t_nextHandle atomic.Uintptr\n")
	b.WriteString("\t_cstrCache  sync.Map // handle key → []*C.char for borrowing-only lifetime\n")
	b.WriteString(")\n\n")

	b.WriteString("func _allocHandle(impl interface{}) uintptr {\n")
	b.WriteString("\tkey := _nextHandle.Add(1)\n")
	b.WriteString("\t_handles.Store(key, impl)\n")
	b.WriteString("\treturn key\n")
	b.WriteString("}\n\n")

	b.WriteString("func _freeHandle(key uintptr) {\n")
	b.WriteString("\t_handles.Delete(key)\n")
	b.WriteString("\t// Free any cached C strings for this handle\n")
	b.WriteString("\tif val, ok := _cstrCache.LoadAndDelete(key); ok {\n")
	b.WriteString("\t\tfor _, cs := range val.([]*C.char) {\n")
	b.WriteString("\t\t\tC.free(unsafe.Pointer(cs))\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	b.WriteString("// _cacheStrings allocates C strings and caches them for the handle's lifetime.\n")
	b.WriteString("// Previous cache for this handle is freed first (borrowing-only semantics).\n")
	b.WriteString("func _cacheStrings(handleKey uintptr, ss ...string) []*C.char {\n")
	b.WriteString("\t// Free previous cache\n")
	b.WriteString("\tif val, ok := _cstrCache.LoadAndDelete(handleKey); ok {\n")
	b.WriteString("\t\tfor _, cs := range val.([]*C.char) {\n")
	b.WriteString("\t\t\tC.free(unsafe.Pointer(cs))\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tresult := make([]*C.char, len(ss))\n")
	b.WriteString("\tfor i, s := range ss {\n")
	b.WriteString("\t\tresult[i] = C.CString(s)\n")
	b.WriteString("\t}\n")
	b.WriteString("\t_cstrCache.Store(handleKey, result)\n")
	b.WriteString("\treturn result\n")
	b.WriteString("}\n\n")
}

// writeCgoExportFunc writes an //export annotated cgo function that delegates to the Go interface.
func writeCgoExportFunc(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef, resolved resolver.ResolvedTypes) {
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

	// Body: detect lifecycle vs regular and delegate appropriately
	if handleName, ok := isGoCreateMethod(method); ok {
		writeCgoCreateBody(b, handleName)
	} else if handleParamName, ok := isGoDestroyMethod(method); ok {
		writeCgoDestroyBody(b, handleParamName)
	} else {
		writeCgoRegularBody(b, ifaceName, method, resolved)
	}

	b.WriteString("}\n")
}

// writeCgoCreateBody writes the body of a create method in the cgo shim.
func writeCgoCreateBody(b *strings.Builder, handleName string) {
	implStruct := ToPascalCase(handleName) + "Impl"
	handleTypedef := HandleTypedefName(handleName)
	fmt.Fprintf(b, "\timpl := &%s{}\n", implStruct)
	b.WriteString("\tkey := _allocHandle(impl)\n")
	fmt.Fprintf(b, "\t*out_result = (C.%s)(unsafe.Pointer(key))\n", handleTypedef)
	b.WriteString("\treturn 0\n")
}

// writeCgoDestroyBody writes the body of a destroy method in the cgo shim.
func writeCgoDestroyBody(b *strings.Builder, handleParamName string) {
	fmt.Fprintf(b, "\t_freeHandle(uintptr(unsafe.Pointer(%s)))\n", handleParamName)
}

// writeCgoRegularBody writes the body of a regular method in the cgo shim,
// delegating to the Go interface.
func writeCgoRegularBody(b *strings.Builder, ifaceName string, method *model.MethodDef, resolved resolver.ResolvedTypes) {
	hasError := method.Error != ""
	hasReturn := method.Returns != nil
	goIfaceName := ToPascalCase(ifaceName)
	methodName := ToPascalCase(method.Name)

	// Find handle parameter for interface lookup
	var handleParam *model.ParameterDef
	for i := range method.Parameters {
		if _, ok := model.IsHandle(method.Parameters[i].Type); ok {
			handleParam = &method.Parameters[i]
			break
		}
	}

	if handleParam == nil {
		b.WriteString("\t// TODO: no handle parameter found — implement manually\n")
		if hasError {
			b.WriteString("\treturn 0\n")
		}
		return
	}

	// Look up impl from handle map
	fmt.Fprintf(b, "\thandle := uintptr(unsafe.Pointer(%s))\n", handleParam.Name)
	b.WriteString("\tval, ok := _handles.Load(handle)\n")
	b.WriteString("\tif !ok {\n")
	if hasError {
		b.WriteString("\t\treturn -1\n")
	} else {
		b.WriteString("\t\treturn\n")
	}
	b.WriteString("\t}\n")
	fmt.Fprintf(b, "\timpl := val.(%s)\n", goIfaceName)

	// Convert non-handle parameters
	var callArgs []string
	for _, p := range method.Parameters {
		if _, ok := model.IsHandle(p.Type); ok {
			continue // handle is resolved to impl above
		}
		if model.IsString(p.Type) {
			goVar := ToCamelCase(p.Name) + "Go"
			fmt.Fprintf(b, "\t%s := C.GoString(%s)\n", goVar, p.Name)
			callArgs = append(callArgs, goVar)
		} else if elemType, ok := model.IsBuffer(p.Type); ok {
			goVar := ToCamelCase(p.Name) + "Slice"
			goElemType := primitiveGoType(elemType)
			fmt.Fprintf(b, "\t%s := unsafe.Slice((*%s)(unsafe.Pointer(%s)), %s_len)\n",
				goVar, goElemType, p.Name, p.Name)
			callArgs = append(callArgs, goVar)
		} else if model.IsPrimitive(p.Type) {
			goVar := ToCamelCase(p.Name) + "Val"
			goType := primitiveGoType(p.Type)
			fmt.Fprintf(b, "\t%s := %s(%s)\n", goVar, goType, p.Name)
			callArgs = append(callArgs, goVar)
		} else {
			// FlatBuffer type — pass as pointer (TODO: proper marshalling)
			callArgs = append(callArgs, p.Name)
		}
	}

	// Call interface method
	argStr := strings.Join(callArgs, ", ")

	switch {
	case hasError && hasReturn:
		fmt.Fprintf(b, "\tresult, err := impl.%s(%s)\n", methodName, argStr)
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn -1\n")
		b.WriteString("\t}\n")
		// Marshal return value to C out_result
		writeCgoReturnMarshal(b, method.Returns.Type, resolved)
		b.WriteString("\treturn 0\n")
	case hasError && !hasReturn:
		fmt.Fprintf(b, "\terr := impl.%s(%s)\n", methodName, argStr)
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn -1\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn 0\n")
	case !hasError && hasReturn:
		fmt.Fprintf(b, "\tresult := impl.%s(%s)\n", methodName, argStr)
		writeCgoReturnMarshalDirect(b, method.Returns.Type)
	default:
		fmt.Fprintf(b, "\timpl.%s(%s)\n", methodName, argStr)
	}
}

// writeCgoReturnMarshal writes code to marshal a Go return value into a C out_result pointer.
func writeCgoReturnMarshal(b *strings.Builder, retType string, resolved resolver.ResolvedTypes) {
	if _, ok := model.IsHandle(retType); ok {
		handleName, _ := model.IsHandle(retType)
		handleTypedef := HandleTypedefName(handleName)
		fmt.Fprintf(b, "\t*out_result = (C.%s)(unsafe.Pointer(result))\n", handleTypedef)
		return
	}
	if model.IsPrimitive(retType) {
		cType := model.PrimitiveCType(retType)
		fmt.Fprintf(b, "\t*out_result = C.%s(result)\n", cType)
		return
	}

	// FlatBuffer struct — marshal fields from Go struct to C struct
	info, ok := resolved[retType]
	if !ok {
		b.WriteString("\t_ = result // TODO: marshal FlatBuffer return type\n")
		return
	}

	// Collect string fields and non-string fields
	var stringGoExprs []string
	for _, f := range info.Fields {
		if f.Type == "string" {
			stringGoExprs = append(stringGoExprs, "result."+ToPascalCase(f.Name))
		}
	}

	if len(stringGoExprs) > 0 {
		fmt.Fprintf(b, "\tcStrs := _cacheStrings(handle, %s)\n", strings.Join(stringGoExprs, ", "))
		idx := 0
		for _, f := range info.Fields {
			if f.Type == "string" {
				fmt.Fprintf(b, "\tout_result.%s = cStrs[%d]\n", f.Name, idx)
				idx++
			} else {
				goFieldName := ToPascalCase(f.Name)
				cType := fbsFieldToCgoType(f.Type)
				fmt.Fprintf(b, "\tout_result.%s = %s(result.%s)\n", f.Name, cType, goFieldName)
			}
		}
	} else {
		for _, f := range info.Fields {
			goFieldName := ToPascalCase(f.Name)
			cType := fbsFieldToCgoType(f.Type)
			fmt.Fprintf(b, "\tout_result.%s = %s(result.%s)\n", f.Name, cType, goFieldName)
		}
	}
}

// writeCgoReturnMarshalDirect writes code for infallible non-void methods that return directly.
func writeCgoReturnMarshalDirect(b *strings.Builder, retType string) {
	if _, ok := model.IsHandle(retType); ok {
		handleName, _ := model.IsHandle(retType)
		handleTypedef := HandleTypedefName(handleName)
		fmt.Fprintf(b, "\treturn (C.%s)(unsafe.Pointer(result))\n", handleTypedef)
		return
	}
	if model.IsPrimitive(retType) {
		cType := model.PrimitiveCType(retType)
		fmt.Fprintf(b, "\treturn C.%s(result)\n", cType)
		return
	}
	b.WriteString("\t_ = result // TODO: marshal FlatBuffer direct return\n")
	b.WriteString("\treturn 0\n")
}

// fbsFieldToCgoType maps a FlatBuffer field type to a cgo cast expression prefix.
func fbsFieldToCgoType(t string) string {
	switch t {
	case "bool":
		return "C.bool"
	case "int8":
		return "C.int8_t"
	case "uint8":
		return "C.uint8_t"
	case "int16":
		return "C.int16_t"
	case "uint16":
		return "C.uint16_t"
	case "int32":
		return "C.int32_t"
	case "uint32":
		return "C.uint32_t"
	case "int64":
		return "C.int64_t"
	case "uint64":
		return "C.uint64_t"
	case "float32":
		return "C.float"
	case "float64":
		return "C.double"
	}
	return "C." + model.FlatBufferCType(t)
}

// --- Impl stub generation ---

// generateImpl produces the stub implementation file.
// Only includes interfaces that have non-lifecycle methods.
func (g *GoImplGenerator) generateImpl(api *model.APIDefinition, apiName string, resolved resolver.ResolvedTypes) (*OutputFile, error) {
	pkgName := goPackageName(apiName)
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n", pkgName)
	b.WriteString("\n")

	for _, iface := range api.Interfaces {
		// Collect non-lifecycle methods
		var methods []model.MethodDef
		for _, method := range iface.Methods {
			if !isGoLifecycleMethod(&method) {
				methods = append(methods, method)
			}
		}
		if len(methods) == 0 {
			continue
		}

		ifaceName := ToPascalCase(iface.Name)
		structName := ifaceName + "Impl"

		fmt.Fprintf(&b, "// %s is a stub implementation of %s.\n", structName, ifaceName)
		fmt.Fprintf(&b, "type %s struct{}\n\n", structName)

		// Verify interface satisfaction.
		fmt.Fprintf(&b, "var _ %s = (*%s)(nil)\n\n", ifaceName, structName)

		for _, method := range methods {
			writeGoStubMethod(&b, structName, &method, resolved)
			b.WriteString("\n")
		}
	}

	filename := apiName + "_impl.go"
	return &OutputFile{Path: filename, Content: []byte(b.String()), Scaffold: true, ProjectFile: true}, nil
}

// writeGoStubMethod writes a stub method on the impl struct.
func writeGoStubMethod(b *strings.Builder, structName string, method *model.MethodDef, resolved resolver.ResolvedTypes) {
	methodName := ToPascalCase(method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build parameter list, excluding handle parameters
	var params []string
	for _, p := range method.Parameters {
		if _, ok := model.IsHandle(p.Type); ok {
			continue
		}
		params = append(params, goInterfaceParamSignature(&p, resolved))
	}
	paramStr := strings.Join(params, ", ")

	// Determine return signature
	var retSig string
	switch {
	case hasError && hasReturn:
		retSig = fmt.Sprintf("(%s, error)", goReturnStructType(method.Returns.Type))
	case hasError && !hasReturn:
		retSig = "error"
	case !hasError && hasReturn:
		retSig = goReturnStructType(method.Returns.Type)
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
		zeroVal := goReturnStructZeroValue(method.Returns.Type)
		fmt.Fprintf(b, "\treturn %s, nil\n", zeroVal)
	case hasError && !hasReturn:
		b.WriteString("\treturn nil\n")
	case !hasError && hasReturn:
		zeroVal := goReturnStructZeroValue(method.Returns.Type)
		fmt.Fprintf(b, "\treturn %s\n", zeroVal)
	default:
		// void — no return
	}

	b.WriteString("}\n")
}

// --- Types generation ---

// generateTypes produces the Go type definitions file from FBS schemas.
// Emits enum constants and Go structs for FlatBuffer types used as return values.
func (g *GoImplGenerator) generateTypes(resolved resolver.ResolvedTypes, apiName string, api *model.APIDefinition) *OutputFile {
	pkgName := goPackageName(apiName)
	var b strings.Builder

	fmt.Fprintf(&b, "package %s\n\n", pkgName)

	// Collect and sort enum names
	var enumNames []string
	for name, info := range resolved {
		if info.Kind == resolver.TypeKindEnum {
			enumNames = append(enumNames, name)
		}
	}
	sort.Strings(enumNames)

	// Emit enum constants
	if len(enumNames) > 0 {
		b.WriteString("const (\n")
		for _, name := range enumNames {
			info := resolved[name]
			goPrefix := goEnumPrefix(name)
			for _, val := range info.EnumValues {
				fmt.Fprintf(&b, "\t%s%s = %d\n", goPrefix, val.Name, val.Value)
			}
			b.WriteString("\n")
		}
		b.WriteString(")\n\n")
	}

	// Collect FlatBuffer types used as return values in the API
	returnTypes := collectReturnTypes(api)

	// Emit Go structs for return types (tables and structs only, not enums)
	var structNames []string
	for name := range returnTypes {
		info, ok := resolved[name]
		if !ok {
			continue
		}
		if info.Kind == resolver.TypeKindStruct || info.Kind == resolver.TypeKindTable {
			structNames = append(structNames, name)
		}
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		info := resolved[name]
		goName := goReturnStructName(name)
		fmt.Fprintf(&b, "// %s is the Go representation of the %s FlatBuffer type.\n", goName, name)
		fmt.Fprintf(&b, "type %s struct {\n", goName)
		for _, f := range info.Fields {
			goFieldName := ToPascalCase(f.Name)
			goFieldType := fbsFieldToGoType(f.Type)
			fmt.Fprintf(&b, "\t%s %s\n", goFieldName, goFieldType)
		}
		b.WriteString("}\n\n")
	}

	filename := apiName + "_types.go"
	return &OutputFile{Path: filename, Content: []byte(b.String())}
}

// collectReturnTypes collects all FlatBuffer types used as method return values.
func collectReturnTypes(api *model.APIDefinition) map[string]bool {
	types := make(map[string]bool)
	for _, iface := range api.Interfaces {
		for _, method := range iface.Methods {
			if method.Returns != nil && model.IsFlatBufferType(method.Returns.Type) {
				types[method.Returns.Type] = true
			}
		}
	}
	return types
}

// fbsFieldToGoType maps a FlatBuffer field type to a Go type.
func fbsFieldToGoType(t string) string {
	switch t {
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "int8":
		return "int8"
	case "uint8":
		return "uint8"
	case "int16":
		return "int16"
	case "uint16":
		return "uint16"
	case "int32":
		return "int32"
	case "uint32":
		return "uint32"
	case "int64":
		return "int64"
	case "uint64":
		return "uint64"
	case "float32":
		return "float32"
	case "float64":
		return "float64"
	}
	// FlatBuffer type reference
	return goReturnStructName(t)
}

// --- Go module generation ---

// generateGoMod produces a scaffold go.mod for the implementation package.
func (g *GoImplGenerator) generateGoMod(api *model.APIDefinition) *OutputFile {
	moduleName := strings.ReplaceAll(api.API.Name, "_", "-")
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n\n", moduleName)
	b.WriteString("go 1.24\n")
	return &OutputFile{Path: "go.mod", Content: []byte(b.String()), Scaffold: true, ProjectFile: true}
}

// generateGitignore produces a .gitignore that lists the generated Go source files
// copied from generated/ into the package root by the Makefile.
func (g *GoImplGenerator) generateGitignore(apiName string) *OutputFile {
	var b strings.Builder
	b.WriteString("# Generated Go sources — copied from generated/ by Makefile; do not edit.\n")
	fmt.Fprintf(&b, "%s_interface.go\n", apiName)
	fmt.Fprintf(&b, "%s_cgo.go\n", apiName)
	fmt.Fprintf(&b, "%s_types.go\n", apiName)
	fmt.Fprintf(&b, "%s_wasm.go\n", apiName)
	return &OutputFile{Path: ".gitignore", Content: []byte(b.String()), Scaffold: true, ProjectFile: true}
}

// --- Package-level utilities ---

// goPackageName returns "main" — Go impl packages are always built as
// c-shared or wasip1 binaries, both of which require package main.
func goPackageName(apiName string) string {
	return "main"
}

// goEnumPrefix converts a FlatBuffer enum name to a Go constant prefix.
// e.g., "Common.ErrorCode" -> "CommonErrorCode"
func goEnumPrefix(name string) string {
	return strings.ReplaceAll(name, ".", "")
}

// primitiveGoType maps xplatter primitive types to Go types.
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
	if handleName, ok := model.IsHandle(t); ok {
		return HandleTypedefName(handleName)
	}
	if model.IsPrimitive(t) {
		return model.PrimitiveCType(t)
	}
	return model.FlatBufferCType(t)
}
