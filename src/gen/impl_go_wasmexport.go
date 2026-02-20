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

	b.WriteString("//go:build wasip1\n\n")
	b.WriteString(GeneratedFileHeader(ctx, "//", false))
	fmt.Fprintf(&b, `
package %s

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

`, pkgName)

	writeWasmMemoryAllocator(&b)
	writeWasmHandleManagement(&b)
	writeWasmCStringHelper(&b)
	writeWasmStringCacheHelpers(&b)
	writeWasmPlatformImports(&b, apiName)

	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "/* %s */\n\n", iface.Name)
		for _, ctor := range iface.Constructors {
			writeWasmConstructorFunc(&b, apiName, iface.Name, &ctor)
			b.WriteString("\n")
		}
		if handleName, ok := iface.ConstructorHandleName(); ok {
			writeWasmDestructorFunc(&b, apiName, iface.Name, handleName)
			b.WriteString("\n")
		}
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
	b.WriteString(`// Memory allocation exports — called by JS binding via malloc/free.
var _wasmAllocs sync.Map

//go:wasmexport malloc
func _wasmMalloc(size uint32) uintptr {
	buf := make([]byte, size)
	ptr := uintptr(unsafe.Pointer(&buf[0]))
	_wasmAllocs.Store(ptr, buf)
	return ptr
}

//go:wasmexport free
func _wasmFree(ptr uintptr) {
	_wasmAllocs.Delete(ptr)
}

`)
}

// writeWasmHandleManagement writes handle map helpers mirroring the cgo version.
func writeWasmHandleManagement(b *strings.Builder) {
	b.WriteString(`// Handle management — maps integer keys to Go interface implementations.
var (
	_wasmHandles sync.Map
	_nextHandle  atomic.Uintptr
)

func _allocHandle(impl interface{}) uintptr {
	key := _nextHandle.Add(1)
	_wasmHandles.Store(key, impl)
	return key
}

func _freeHandle(key uintptr) {
	_wasmHandles.Delete(key)
	// Free any cached WASM strings for this handle
	if val, ok := _wasmStrCache.LoadAndDelete(key); ok {
		for _, ptr := range val.([]uintptr) {
			_wasmFree(ptr)
		}
	}
}

`)
}

// writeWasmCStringHelper writes a helper that reads null-terminated strings from linear memory.
func writeWasmCStringHelper(b *strings.Builder) {
	b.WriteString(`// _cstring reads a null-terminated C string from WASM linear memory.
func _cstring(ptr uintptr) string {
	var n int
	for *(*byte)(unsafe.Pointer(ptr + uintptr(n))) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n))
}

`)
}

// writeWasmStringCacheHelpers writes helpers for caching WASM string allocations per-handle.
func writeWasmStringCacheHelpers(b *strings.Builder) {
	b.WriteString(`// WASM string cache — maps handle key → []uintptr of allocated strings.
var _wasmStrCache sync.Map

// _wasmCacheStrings allocates null-terminated strings in WASM linear memory
// and caches them for the handle's lifetime (borrowing-only semantics).
func _wasmCacheStrings(handleKey uintptr, ss ...string) []uintptr {
	// Free previous cache
	if val, ok := _wasmStrCache.LoadAndDelete(handleKey); ok {
		for _, ptr := range val.([]uintptr) {
			_wasmFree(ptr)
		}
	}
	result := make([]uintptr, len(ss))
	for i, s := range ss {
		data := []byte(s)
		ptr := _wasmMalloc(uint32(len(data) + 1))
		buf := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(data)+1)
		copy(buf, data)
		buf[len(data)] = 0
		result[i] = ptr
	}
	_wasmStrCache.Store(handleKey, result)
	return result
}

`)
}

// writeWasmPlatformImports writes //go:wasmimport declarations for the 6 platform services.
func writeWasmPlatformImports(b *strings.Builder, apiName string) {
	fmt.Fprintf(b, `// Platform service imports — provided by the JS binding as WASM imports.
//go:wasmimport env %[1]s_log_sink
func _platform_log_sink(level int32, tag uintptr, message uintptr)

//go:wasmimport env %[1]s_resource_count
func _platform_resource_count() uint32

//go:wasmimport env %[1]s_resource_name
func _platform_resource_name(index uint32, buffer uintptr, bufferSize uint32) int32

//go:wasmimport env %[1]s_resource_exists
func _platform_resource_exists(name uintptr) int32

//go:wasmimport env %[1]s_resource_size
func _platform_resource_size(name uintptr) uint32

//go:wasmimport env %[1]s_resource_read
func _platform_resource_read(name uintptr, buffer uintptr, bufferSize uint32) int32

`, apiName)
}

// writeWasmConstructorFunc writes a //go:wasmexport constructor that allocates a handle.
func writeWasmConstructorFunc(b *strings.Builder, apiName, ifaceName string, ctor *model.MethodDef) {
	funcName := CABIFunctionName(apiName, ifaceName, ctor.Name)
	handleName, _ := model.IsHandle(ctor.Returns.Type)
	implStruct := ToPascalCase(handleName) + "Impl"

	var wasmParams []string
	for _, p := range ctor.Parameters {
		wasmParams = append(wasmParams, goWasmExportParams(&p)...)
	}
	wasmParams = append(wasmParams, "out_result uintptr")

	fmt.Fprintf(b, "//go:wasmexport %s\n", funcName)
	paramStr := strings.Join(wasmParams, ", ")
	fmt.Fprintf(b, "func %s(%s) int32 {\n", funcName, paramStr)
	fmt.Fprintf(b, `	impl := &%s{}
	key := _allocHandle(impl)
	*(*uint32)(unsafe.Pointer(out_result)) = uint32(key)
	return 0
`, implStruct)
	b.WriteString("}\n")
}

// writeWasmDestructorFunc writes a //go:wasmexport destructor that frees a handle.
func writeWasmDestructorFunc(b *strings.Builder, apiName, ifaceName, handleName string) {
	destructor := SyntheticDestructor(handleName)
	funcName := CABIFunctionName(apiName, ifaceName, destructor.Name)
	paramName := destructor.Parameters[0].Name

	fmt.Fprintf(b, "//go:wasmexport %s\n", funcName)
	fmt.Fprintf(b, "func %s(%s uint32) {\n", funcName, paramName)
	fmt.Fprintf(b, "\t_freeHandle(uintptr(%s))\n", paramName)
	b.WriteString("}\n")
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

	writeWasmRegularBody(b, ifaceName, method, resolved)

	b.WriteString("}\n")
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
	if hasError {
		b.WriteString("\tif !ok {\n\t\treturn -1\n\t}\n")
	} else {
		b.WriteString("\tif !ok {\n\t\treturn\n\t}\n")
	}
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
			callArgs = append(callArgs, p.Name)
		} else {
			callArgs = append(callArgs, p.Name)
		}
	}

	// Call interface method
	argStr := strings.Join(callArgs, ", ")

	switch {
	case hasError && hasReturn:
		fmt.Fprintf(b, "\tresult, err := impl.%s(%s)\n", methodName, argStr)
		b.WriteString("\tif err != nil {\n\t\treturn -1\n\t}\n")
		writeWasmReturnMarshal(b, method.Returns.Type, handleParam.Name, resolved)
		b.WriteString("\treturn 0\n")
	case hasError && !hasReturn:
		fmt.Fprintf(b, "\terr := impl.%s(%s)\n", methodName, argStr)
		b.WriteString("\tif err != nil {\n\t\treturn -1\n\t}\n")
		b.WriteString("\treturn 0\n")
	case !hasError && hasReturn:
		fmt.Fprintf(b, "\tresult := impl.%s(%s)\n", methodName, argStr)
		b.WriteString("\t_ = result // TODO: marshal WASM direct return\n\treturn 0\n")
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
		fmt.Fprintf(b, "\t*(*%s)(unsafe.Pointer(out_result)) = result\n", primitiveGoType(retType))
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
