package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

func init() {
	Register("impl_go_wasm", func() Generator { return &GoWASMImplGenerator{} })
}

// GoWASMImplGenerator produces Go WASM implementation scaffolding for GOOS=wasip1 builds.
// It generates a _wasm.go file with //go:wasmexport functions that delegate to the
// Go interface, complementing the cgo shim (_cgo.go) which is automatically excluded
// when cgo is disabled.
type GoWASMImplGenerator struct{}

func (g *GoWASMImplGenerator) Name() string { return "impl_go_wasm" }

func (g *GoWASMImplGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name
	pkgName := goPackageName(apiName)

	var b strings.Builder

	// Build tag, generated header, and package declaration
	b.WriteString("//go:build wasip1\n\n")
	b.WriteString(GeneratedFileHeader(ctx, "//", false))
	fmt.Fprintf(&b, "\npackage %s\n\n", pkgName)

	// Imports
	b.WriteString("import (\n")
	b.WriteString("\t\"sync\"\n")
	b.WriteString("\t\"sync/atomic\"\n")
	b.WriteString("\t\"unsafe\"\n")
	b.WriteString(")\n\n")

	writeWasmMemoryAllocator(&b)
	writeWasmHandleManagement(&b)
	writeWasmCStringHelper(&b)
	writeWasmStringCacheHelpers(&b)
	writeWasmPlatformImports(&b, apiName)

	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "/* %s */\n\n", iface.Name)
		for _, method := range iface.Methods {
			writeWasmExportFunc(&b, apiName, iface.Name, &method, ctx.ResolvedTypes)
			b.WriteString("\n")
		}
	}

	filename := apiName + "_wasm.go"
	return []*OutputFile{{Path: filename, Content: []byte(b.String())}}, nil
}

// writeWasmMemoryAllocator writes malloc/free exports for WASM linear memory.
// The sync.Map pins byte slices so the GC does not collect them.
func writeWasmMemoryAllocator(b *strings.Builder) {
	b.WriteString("// Memory allocation exports — called by JS binding via malloc/free.\n")
	b.WriteString("var _wasmAllocs sync.Map\n\n")
	b.WriteString("//go:wasmexport malloc\n")
	b.WriteString("func _wasmMalloc(size uint32) uintptr {\n")
	b.WriteString("\tbuf := make([]byte, size)\n")
	b.WriteString("\tptr := uintptr(unsafe.Pointer(&buf[0]))\n")
	b.WriteString("\t_wasmAllocs.Store(ptr, buf)\n")
	b.WriteString("\treturn ptr\n")
	b.WriteString("}\n\n")
	b.WriteString("//go:wasmexport free\n")
	b.WriteString("func _wasmFree(ptr uintptr) {\n")
	b.WriteString("\t_wasmAllocs.Delete(ptr)\n")
	b.WriteString("}\n\n")
}

// writeWasmHandleManagement writes handle map helpers mirroring the cgo version.
func writeWasmHandleManagement(b *strings.Builder) {
	b.WriteString("// Handle management — maps integer keys to Go interface implementations.\n")
	b.WriteString("var (\n")
	b.WriteString("\t_wasmHandles sync.Map\n")
	b.WriteString("\t_nextHandle  atomic.Uintptr\n")
	b.WriteString(")\n\n")
	b.WriteString("func _allocHandle(impl interface{}) uintptr {\n")
	b.WriteString("\tkey := _nextHandle.Add(1)\n")
	b.WriteString("\t_wasmHandles.Store(key, impl)\n")
	b.WriteString("\treturn key\n")
	b.WriteString("}\n\n")
	b.WriteString("func _freeHandle(key uintptr) {\n")
	b.WriteString("\t_wasmHandles.Delete(key)\n")
	b.WriteString("\t// Free any cached WASM strings for this handle\n")
	b.WriteString("\tif val, ok := _wasmStrCache.LoadAndDelete(key); ok {\n")
	b.WriteString("\t\tfor _, ptr := range val.([]uintptr) {\n")
	b.WriteString("\t\t\t_wasmFree(ptr)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
}

// writeWasmCStringHelper writes a helper that reads null-terminated strings from linear memory.
func writeWasmCStringHelper(b *strings.Builder) {
	b.WriteString("// _cstring reads a null-terminated C string from WASM linear memory.\n")
	b.WriteString("func _cstring(ptr uintptr) string {\n")
	b.WriteString("\tvar n int\n")
	b.WriteString("\tfor *(*byte)(unsafe.Pointer(ptr + uintptr(n))) != 0 {\n")
	b.WriteString("\t\tn++\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n))\n")
	b.WriteString("}\n\n")
}

// writeWasmStringCacheHelpers writes helpers for caching WASM string allocations per-handle.
func writeWasmStringCacheHelpers(b *strings.Builder) {
	b.WriteString("// WASM string cache — maps handle key → []uintptr of allocated strings.\n")
	b.WriteString("var _wasmStrCache sync.Map\n\n")
	b.WriteString("// _wasmCacheStrings allocates null-terminated strings in WASM linear memory\n")
	b.WriteString("// and caches them for the handle's lifetime (borrowing-only semantics).\n")
	b.WriteString("func _wasmCacheStrings(handleKey uintptr, ss ...string) []uintptr {\n")
	b.WriteString("\t// Free previous cache\n")
	b.WriteString("\tif val, ok := _wasmStrCache.LoadAndDelete(handleKey); ok {\n")
	b.WriteString("\t\tfor _, ptr := range val.([]uintptr) {\n")
	b.WriteString("\t\t\t_wasmFree(ptr)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tresult := make([]uintptr, len(ss))\n")
	b.WriteString("\tfor i, s := range ss {\n")
	b.WriteString("\t\tdata := []byte(s)\n")
	b.WriteString("\t\tptr := _wasmMalloc(uint32(len(data) + 1))\n")
	b.WriteString("\t\tbuf := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(data)+1)\n")
	b.WriteString("\t\tcopy(buf, data)\n")
	b.WriteString("\t\tbuf[len(data)] = 0\n")
	b.WriteString("\t\tresult[i] = ptr\n")
	b.WriteString("\t}\n")
	b.WriteString("\t_wasmStrCache.Store(handleKey, result)\n")
	b.WriteString("\treturn result\n")
	b.WriteString("}\n\n")
}

// writeWasmPlatformImports writes //go:wasmimport declarations for the 6 platform services.
func writeWasmPlatformImports(b *strings.Builder, apiName string) {
	b.WriteString("// Platform service imports — provided by the JS binding as WASM imports.\n")
	fmt.Fprintf(b, "//go:wasmimport env %s_log_sink\n", apiName)
	fmt.Fprintf(b, "func _platform_log_sink(level int32, tag uintptr, message uintptr)\n\n")
	fmt.Fprintf(b, "//go:wasmimport env %s_resource_count\n", apiName)
	fmt.Fprintf(b, "func _platform_resource_count() uint32\n\n")
	fmt.Fprintf(b, "//go:wasmimport env %s_resource_name\n", apiName)
	fmt.Fprintf(b, "func _platform_resource_name(index uint32, buffer uintptr, bufferSize uint32) int32\n\n")
	fmt.Fprintf(b, "//go:wasmimport env %s_resource_exists\n", apiName)
	fmt.Fprintf(b, "func _platform_resource_exists(name uintptr) int32\n\n")
	fmt.Fprintf(b, "//go:wasmimport env %s_resource_size\n", apiName)
	fmt.Fprintf(b, "func _platform_resource_size(name uintptr) uint32\n\n")
	fmt.Fprintf(b, "//go:wasmimport env %s_resource_read\n", apiName)
	fmt.Fprintf(b, "func _platform_resource_read(name uintptr, buffer uintptr, bufferSize uint32) int32\n\n")
}

// writeWasmExportFunc writes a single //go:wasmexport annotated function that
// delegates to the Go interface.
func writeWasmExportFunc(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef, resolved resolver.ResolvedTypes) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build WASM parameter list
	var wasmParams []string
	for _, p := range method.Parameters {
		wasmParams = append(wasmParams, goWasmExportParams(&p)...)
	}

	// Determine return type and out-parameter
	var wasmReturnType string
	switch {
	case hasError && hasReturn:
		wasmReturnType = "int32"
		wasmParams = append(wasmParams, "out_result uintptr")
	case hasError && !hasReturn:
		wasmReturnType = "int32"
	case !hasError && hasReturn:
		wasmReturnType = goWasmReturnType(method.Returns.Type)
	}

	fmt.Fprintf(b, "//go:wasmexport %s\n", funcName)

	paramStr := strings.Join(wasmParams, ", ")
	if wasmReturnType != "" {
		fmt.Fprintf(b, "func %s(%s) %s {\n", funcName, paramStr, wasmReturnType)
	} else {
		fmt.Fprintf(b, "func %s(%s) {\n", funcName, paramStr)
	}

	// Body: detect lifecycle vs regular and delegate appropriately
	if handleName, ok := isGoCreateMethod(method); ok {
		writeWasmCreateBody(b, handleName)
	} else if handleParamName, ok := isGoDestroyMethod(method); ok {
		writeWasmDestroyBody(b, handleParamName)
	} else {
		writeWasmRegularBody(b, ifaceName, method, resolved)
	}

	b.WriteString("}\n")
}

// writeWasmCreateBody writes the body of a create method in the WASM shim.
func writeWasmCreateBody(b *strings.Builder, handleName string) {
	implStruct := ToPascalCase(handleName) + "Impl"
	fmt.Fprintf(b, "\timpl := &%s{}\n", implStruct)
	b.WriteString("\tkey := _allocHandle(impl)\n")
	// In WASM32, handles are uint32 in linear memory
	b.WriteString("\t*(*uint32)(unsafe.Pointer(out_result)) = uint32(key)\n")
	b.WriteString("\treturn 0\n")
}

// writeWasmDestroyBody writes the body of a destroy method in the WASM shim.
func writeWasmDestroyBody(b *strings.Builder, handleParamName string) {
	fmt.Fprintf(b, "\t_freeHandle(%s)\n", handleParamName)
}

// writeWasmRegularBody writes the body of a regular method in the WASM shim,
// delegating to the Go interface.
func writeWasmRegularBody(b *strings.Builder, ifaceName string, method *model.MethodDef, resolved resolver.ResolvedTypes) {
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
	fmt.Fprintf(b, "\tval, ok := _wasmHandles.Load(%s)\n", handleParam.Name)
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
			continue // handle resolved to impl above
		}
		if model.IsString(p.Type) {
			goVar := ToCamelCase(p.Name) + "Go"
			fmt.Fprintf(b, "\t%s := _cstring(%s)\n", goVar, p.Name)
			callArgs = append(callArgs, goVar)
		} else if elemType, ok := model.IsBuffer(p.Type); ok {
			goVar := ToCamelCase(p.Name) + "Slice"
			goElemType := primitiveGoType(elemType)
			fmt.Fprintf(b, "\t%s := unsafe.Slice((*%s)(unsafe.Pointer(%s)), %s_len)\n",
				goVar, goElemType, p.Name, p.Name)
			callArgs = append(callArgs, goVar)
		} else if model.IsPrimitive(p.Type) {
			// Primitive — use directly (WASM types match Go types)
			callArgs = append(callArgs, p.Name)
		} else {
			// FlatBuffer type — pointer into linear memory (TODO: proper marshalling)
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
		writeWasmReturnMarshal(b, method.Returns.Type, handleParam.Name, resolved)
		b.WriteString("\treturn 0\n")
	case hasError && !hasReturn:
		fmt.Fprintf(b, "\terr := impl.%s(%s)\n", methodName, argStr)
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn -1\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn 0\n")
	case !hasError && hasReturn:
		fmt.Fprintf(b, "\tresult := impl.%s(%s)\n", methodName, argStr)
		b.WriteString("\t_ = result // TODO: marshal WASM direct return\n")
		b.WriteString("\treturn 0\n")
	default:
		fmt.Fprintf(b, "\timpl.%s(%s)\n", methodName, argStr)
	}
}

// writeWasmReturnMarshal writes code to marshal a Go return value into WASM linear memory.
func writeWasmReturnMarshal(b *strings.Builder, retType string, handleParamName string, resolved resolver.ResolvedTypes) {
	if _, ok := model.IsHandle(retType); ok {
		b.WriteString("\t*(*uint32)(unsafe.Pointer(out_result)) = uint32(result)\n")
		return
	}
	if model.IsPrimitive(retType) {
		// Write primitive value at out_result
		goType := primitiveGoType(retType)
		fmt.Fprintf(b, "\t*(*%s)(unsafe.Pointer(out_result)) = result\n", goType)
		return
	}

	// FlatBuffer struct — marshal fields into WASM linear memory
	info, ok := resolved[retType]
	if !ok {
		b.WriteString("\t_ = result // TODO: marshal FlatBuffer return type to WASM\n")
		return
	}

	// Collect string fields for caching
	var stringGoExprs []string
	for _, f := range info.Fields {
		if f.Type == "string" {
			stringGoExprs = append(stringGoExprs, "result."+ToPascalCase(f.Name))
		}
	}

	if len(stringGoExprs) > 0 {
		fmt.Fprintf(b, "\tstrPtrs := _wasmCacheStrings(%s, %s)\n",
			handleParamName, strings.Join(stringGoExprs, ", "))
	}

	// Write fields at sequential offsets in the struct.
	// In WASM32: pointers and handles are uint32 (4 bytes), strings are uint32 pointers.
	offset := 0
	strIdx := 0
	for _, f := range info.Fields {
		if f.Type == "string" {
			// String: write the WASM pointer (uint32)
			fmt.Fprintf(b, "\t*(*uint32)(unsafe.Pointer(out_result + %d)) = uint32(strPtrs[%d])\n", offset, strIdx)
			strIdx++
			offset += 4
		} else {
			goFieldName := ToPascalCase(f.Name)
			wasmSize := goWasmFieldSize(f.Type)
			wasmGoType := goWasmFieldGoType(f.Type)
			fmt.Fprintf(b, "\t*(*%s)(unsafe.Pointer(out_result + %d)) = %s(result.%s)\n",
				wasmGoType, offset, wasmGoType, goFieldName)
			offset += wasmSize
		}
	}
}

// goWasmFieldSize returns the byte size of a field in WASM32 linear memory.
func goWasmFieldSize(t string) int {
	switch t {
	case "bool", "int8", "uint8":
		return 1
	case "int16", "uint16":
		return 2
	case "int32", "uint32", "float32":
		return 4
	case "int64", "uint64", "float64":
		return 8
	}
	// Pointer/handle types in WASM32
	return 4
}

// goWasmFieldGoType returns the Go type to use for writing a field into WASM memory.
func goWasmFieldGoType(t string) string {
	switch t {
	case "string":
		return "uint32" // pointer in WASM32
	case "bool":
		return "byte"
	}
	return primitiveGoType(t)
}

// goWasmExportParams returns WASM-typed parameter strings for an export function.
func goWasmExportParams(p *model.ParameterDef) []string {
	if model.IsString(p.Type) {
		return []string{p.Name + " uintptr"}
	}
	if _, ok := model.IsBuffer(p.Type); ok {
		return []string{p.Name + " uintptr", p.Name + "_len uint32"}
	}
	if _, ok := model.IsHandle(p.Type); ok {
		return []string{p.Name + " uintptr"}
	}
	if model.IsPrimitive(p.Type) {
		return []string{p.Name + " " + primitiveGoType(p.Type)}
	}
	// FlatBuffer type — pointer into WASM linear memory
	return []string{p.Name + " uintptr"}
}

// goWasmReturnType returns the WASM-compatible Go return type for an infallible method.
func goWasmReturnType(t string) string {
	if _, ok := model.IsHandle(t); ok {
		return "uintptr"
	}
	if model.IsPrimitive(t) {
		return primitiveGoType(t)
	}
	// FlatBuffer type — pointer
	return "uintptr"
}

// goWasmZeroValue returns the zero value for a WASM return type.
func goWasmZeroValue(t string) string {
	if model.IsPrimitive(t) && t == "bool" {
		return "false"
	}
	return "0"
}
