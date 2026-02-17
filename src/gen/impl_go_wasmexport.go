package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplattergy/model"
)

func init() {
	Register("impl_go_wasm", func() Generator { return &GoWASMImplGenerator{} })
}

// GoWASMImplGenerator produces Go WASM implementation scaffolding for GOOS=wasip1 builds.
// It generates a _wasm.go file with //go:wasmexport functions that complement the
// cgo shim (_cgo.go), which is automatically excluded when cgo is disabled.
type GoWASMImplGenerator struct{}

func (g *GoWASMImplGenerator) Name() string { return "impl_go_wasm" }

func (g *GoWASMImplGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name
	pkgName := goPackageName(apiName)

	var b strings.Builder

	// Build tag and package declaration
	b.WriteString("//go:build wasip1\n")
	fmt.Fprintf(&b, "package %s\n\n", pkgName)

	// Imports
	b.WriteString("import (\n")
	b.WriteString("\t\"sync\"\n")
	b.WriteString("\t\"sync/atomic\"\n")
	b.WriteString("\t\"unsafe\"\n")
	b.WriteString(")\n\n")

	writeWasmMemoryAllocator(&b)
	writeWasmHandleManagement(&b)
	writeWasmCStringHelper(&b)
	writeWasmPlatformImports(&b, apiName)

	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "/* %s */\n\n", iface.Name)
		for _, method := range iface.Methods {
			writeWasmExportFunc(&b, apiName, iface.Name, &method)
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
	b.WriteString("func _lookupHandle(key uintptr) (interface{}, bool) {\n")
	b.WriteString("\treturn _wasmHandles.Load(key)\n")
	b.WriteString("}\n\n")
	b.WriteString("func _freeHandle(key uintptr) {\n")
	b.WriteString("\t_wasmHandles.Delete(key)\n")
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

// writeWasmExportFunc writes a single //go:wasmexport annotated function.
func writeWasmExportFunc(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef) {
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

	writeWasmExportBody(b, method)
	b.WriteString("}\n")
}

// writeWasmExportBody writes the body of a //go:wasmexport function.
// Destroy methods are auto-implemented; all others emit a TODO stub.
func writeWasmExportBody(b *strings.Builder, method *model.MethodDef) {
	hasError := method.Error != ""

	// Destroy methods: auto-implement with _freeHandle
	if handleParamName, ok := isWasmDestroyMethod(method); ok {
		fmt.Fprintf(b, "\t_freeHandle(%s)\n", handleParamName)
		return
	}

	// Look up handle for methods that operate on an existing object
	var handleParam *model.ParameterDef
	for i := range method.Parameters {
		if _, ok := model.IsHandle(method.Parameters[i].Type); ok {
			handleParam = &method.Parameters[i]
			break
		}
	}

	if handleParam != nil {
		fmt.Fprintf(b, "\t_, ok := _lookupHandle(%s)\n", handleParam.Name)
		b.WriteString("\tif !ok {\n")
		if hasError {
			b.WriteString("\t\treturn -1\n")
		} else {
			b.WriteString("\t\treturn\n")
		}
		b.WriteString("\t}\n")
	}

	b.WriteString("\t// TODO: implement\n")

	if hasError {
		b.WriteString("\treturn 0\n")
	} else if method.Returns != nil {
		fmt.Fprintf(b, "\treturn %s\n", goWasmZeroValue(method.Returns.Type))
	}
}

// isWasmDestroyMethod returns the handle parameter name if this is a destroy/release method.
func isWasmDestroyMethod(method *model.MethodDef) (handleParamName string, ok bool) {
	if len(method.Parameters) != 1 {
		return "", false
	}
	p := method.Parameters[0]
	handleName, isHandle := model.IsHandle(p.Type)
	if !isHandle {
		return "", false
	}
	snake := model.HandleToSnake(handleName)
	if method.Name == "destroy_"+snake || method.Name == "release_"+snake {
		return p.Name, true
	}
	return "", false
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
