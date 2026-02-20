package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

func init() {
	Register("jswasm", func() Generator { return &JSWASMGenerator{} })
}

// JSWASMGenerator produces a JavaScript ES module that loads and wraps a WASM build
// of the C API. It provides idiomatic JS classes for handles, string/buffer marshalling
// via WASM linear memory, and error handling that throws on non-zero return codes.
type JSWASMGenerator struct{}

func (g *JSWASMGenerator) Name() string { return "jswasm" }

func (g *JSWASMGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name

	var b strings.Builder

	writeModuleHeader(&b, ctx, api)
	writeMemoryHelpers(&b)
	writeStringMarshalling(&b)
	writeBufferMarshalling(&b)
	writeHandleClasses(&b, api)
	writeWASIPolyfill(&b)
	writePlatformServiceImports(&b, apiName)
	writeWASMLoader(&b, apiName, api)
	writeInterfaceWrappers(&b, apiName, api, ctx.ResolvedTypes)
	writeModuleExports(&b, apiName, api)

	filename := apiName + ".js"
	return []*OutputFile{
		{Path: filename, Content: []byte(b.String())},
	}, nil
}

// writeModuleHeader writes the top-of-file comment and shared state.
func writeModuleHeader(b *strings.Builder, ctx *Context, api *model.APIDefinition) {
	b.WriteString(GeneratedFileHeader(ctx, "//", false))
	b.WriteString(`
const _encoder = new TextEncoder();
const _decoder = new TextDecoder();

let _wasm = null;

`)
}

// writeMemoryHelpers writes malloc/free wrappers for WASM linear memory.
func writeMemoryHelpers(b *strings.Builder) {
	b.WriteString(`// Memory management helpers
function _malloc(size) {
  return _wasm.exports.malloc(size);
}

function _free(ptr) {
  _wasm.exports.free(ptr);
}

function _memoryBuffer() {
  return _wasm.exports.memory.buffer;
}

`)
}

// writeStringMarshalling writes string encode/decode helpers.
func writeStringMarshalling(b *strings.Builder) {
	b.WriteString(`// String marshalling
function _encodeString(str) {
  const bytes = _encoder.encode(str);
  const ptr = _malloc(bytes.length + 1);
  const dest = new Uint8Array(_memoryBuffer(), ptr, bytes.length + 1);
  dest.set(bytes);
  dest[bytes.length] = 0;
  return ptr;
}

function _decodeString(ptr) {
  const mem = new Uint8Array(_memoryBuffer());
  let end = ptr;
  while (mem[end] !== 0) end++;
  return _decoder.decode(mem.subarray(ptr, end));
}

`)
}

// writeBufferMarshalling writes TypedArray → WASM linear memory helpers.
func writeBufferMarshalling(b *strings.Builder) {
	b.WriteString(`// Buffer marshalling
function _copyBufferToWasm(typedArray) {
  const bytes = new Uint8Array(typedArray.buffer, typedArray.byteOffset, typedArray.byteLength);
  const ptr = _malloc(bytes.length);
  const dest = new Uint8Array(_memoryBuffer(), ptr, bytes.length);
  dest.set(bytes);
  return [ptr, typedArray.length];
}

function _readBufferFromWasm(ptr, length, TypedArrayCtor) {
  const byteSize = length * TypedArrayCtor.BYTES_PER_ELEMENT;
  const src = new Uint8Array(_memoryBuffer(), ptr, byteSize);
  const result = new TypedArrayCtor(length);
  new Uint8Array(result.buffer).set(src);
  return result;
}

`)
}

// writeHandleClasses writes wrapper classes for each handle type.
func writeHandleClasses(b *strings.Builder, api *model.APIDefinition) {
	if len(api.Handles) == 0 {
		return
	}

	b.WriteString("// Handle wrapper classes\n")
	for _, h := range api.Handles {
		className := h.Name // Already PascalCase
		fmt.Fprintf(b, `class %[1]s {
  #ptr;

  /** @internal */
  constructor(ptr) {
    this.#ptr = ptr;
  }

  /** @internal */
  get _ptr() {
    if (this.#ptr === 0) {
      throw new Error('%[1]s has been disposed');
    }
    return this.#ptr;
  }

  dispose() {
    this.#ptr = 0;
  }

  close() {
    this.dispose();
  }

  [Symbol.dispose]() {
    this.dispose();
  }
}

`, className)
	}
}

// writeWASIPolyfill emits _buildWasiImports(), a minimal WASI snapshot_preview1
// implementation required by GOOS=wasip1 binaries (e.g. Go WASM).
// Providing these imports is harmless for non-WASI WASM modules (they are never called).
func writeWASIPolyfill(b *strings.Builder) {
	b.WriteString(`// Minimal WASI snapshot_preview1 polyfill
// Required for GOOS=wasip1 WASM modules (e.g. Go). Harmless for others.
function _buildWasiImports() {
  const ERRNO_SUCCESS = 0;
  const ERRNO_BADF    = 8;
  const ERRNO_NOSYS   = 52;
  return {
    // fd_write — route fd1→console.log, fd2→console.error, others discarded
    fd_write(fd, iovsPtr, iovsLen, nwrittenPtr) {
      const mem  = _memoryBuffer();
      const view = new DataView(mem);
      let written = 0;
      for (let i = 0; i < iovsLen; i++) {
        const base = iovsPtr + i * 8;
        const ptr  = view.getUint32(base,     true);
        const len  = view.getUint32(base + 4, true);
        if (len > 0 && (fd === 1 || fd === 2)) {
          const text = new TextDecoder().decode(new Uint8Array(mem, ptr, len));
          (fd === 2 ? console.error : console.log)(text.replace(/\n$/, ''));
        }
        written += len;
      }
      view.setUint32(nwrittenPtr, written, true);
      return ERRNO_SUCCESS;
    },
    fd_read:            () => ERRNO_NOSYS,
    fd_seek:            () => ERRNO_NOSYS,
    fd_close:           () => ERRNO_SUCCESS,
    fd_fdstat_get:      () => ERRNO_NOSYS,
    fd_fdstat_set_flags:() => ERRNO_NOSYS,
    // fd_prestat_get: BADF signals "no preopened directories"
    fd_prestat_get:     () => ERRNO_BADF,
    fd_prestat_dir_name:() => ERRNO_BADF,
    path_open:          () => ERRNO_NOSYS,
    path_filestat_get:  () => ERRNO_NOSYS,
    // environ — empty environment
    environ_sizes_get(countPtr, bufSizePtr) {
      const v = new DataView(_memoryBuffer());
      v.setUint32(countPtr,   0, true);
      v.setUint32(bufSizePtr, 0, true);
      return ERRNO_SUCCESS;
    },
    environ_get(environPtr) {
      new DataView(_memoryBuffer()).setUint32(environPtr, 0, true);
      return ERRNO_SUCCESS;
    },
    // args — no command-line arguments
    args_sizes_get(argcPtr, bufSizePtr) {
      const v = new DataView(_memoryBuffer());
      v.setUint32(argcPtr,    0, true);
      v.setUint32(bufSizePtr, 0, true);
      return ERRNO_SUCCESS;
    },
    args_get(argvPtr) {
      new DataView(_memoryBuffer()).setUint32(argvPtr, 0, true);
      return ERRNO_SUCCESS;
    },
    // proc_exit — must throw, not no-op: Go places an unreachable instruction
    // immediately after every proc_exit call site, expecting it to be terminal.
    // Throwing unwinds the WASM stack cleanly; the loader catches exit code 0
    // as a successful initialization (main() returning normally).
    proc_exit: (code) => {
      const e = new Error('proc_exit:' + code);
      e.wasiExitCode = code;
      throw e;
    },
    // random_get — use Web Crypto
    random_get(bufPtr, bufLen) {
      crypto.getRandomValues(new Uint8Array(_memoryBuffer(), bufPtr, bufLen));
      return ERRNO_SUCCESS;
    },
    // clock_time_get — wall clock in nanoseconds (BigInt)
    clock_time_get(_clockId, _precision, timePtr) {
      const ns = BigInt(Date.now()) * 1_000_000n;
      new DataView(_memoryBuffer()).setBigUint64(timePtr, ns, true);
      return ERRNO_SUCCESS;
    },
    clock_res_get(_clockId, resPtr) {
      new DataView(_memoryBuffer()).setBigUint64(resPtr, 1_000_000n, true);
      return ERRNO_SUCCESS;
    },
    sched_yield:  () => ERRNO_SUCCESS,
    poll_oneoff:  () => ERRNO_NOSYS,
    sock_accept:  () => ERRNO_NOSYS,
    sock_recv:    () => ERRNO_NOSYS,
    sock_send:    () => ERRNO_NOSYS,
    sock_shutdown:() => ERRNO_NOSYS,
  };
}

`)
}

// writePlatformServiceImports writes the env object builder for WASM imports
// that provide platform services (logging, resources).
func writePlatformServiceImports(b *strings.Builder, apiName string) {
	fmt.Fprintf(b, `// Platform service WASM imports
function _buildPlatformImports(services) {
  services = services || {};
  return {
    env: {
      %[1]s_log_sink: (level, tagPtr, msgPtr) => {
        if (services.logSink) {
          services.logSink(level, _decodeString(tagPtr), _decodeString(msgPtr));
        }
      },
      %[1]s_resource_count: () => {
        return services.resourceCount ? services.resourceCount() : 0;
      },
      %[1]s_resource_name: (index, bufferPtr, bufferSize) => {
        if (!services.resourceName) return -1;
        const name = services.resourceName(index);
        if (!name) return -1;
        const bytes = _encoder.encode(name);
        if (bytes.length + 1 > bufferSize) return -1;
        const dest = new Uint8Array(_memoryBuffer(), bufferPtr, bufferSize);
        dest.set(bytes);
        dest[bytes.length] = 0;
        return bytes.length;
      },
      %[1]s_resource_exists: (namePtr) => {
        if (!services.resourceExists) return 0;
        return services.resourceExists(_decodeString(namePtr)) ? 1 : 0;
      },
      %[1]s_resource_size: (namePtr) => {
        if (!services.resourceSize) return 0;
        return services.resourceSize(_decodeString(namePtr));
      },
      %[1]s_resource_read: (namePtr, bufferPtr, bufferSize) => {
        if (!services.resourceRead) return -1;
        const data = services.resourceRead(_decodeString(namePtr));
        if (!data || data.length > bufferSize) return -1;
        const dest = new Uint8Array(_memoryBuffer(), bufferPtr, bufferSize);
        dest.set(new Uint8Array(data.buffer || data));
        return data.length;
      },
    },
    wasi_snapshot_preview1: _buildWasiImports(),
  };
}

`, apiName)
}

// writeWASMLoader writes the async loader function that instantiates the WASM module.
func writeWASMLoader(b *strings.Builder, apiName string, api *model.APIDefinition) {
	loaderName := ToCamelCase("load_" + apiName)
	fmt.Fprintf(b, `// WASM module loader
async function %s(wasmSource, platformServices) {
  const imports = _buildPlatformImports(platformServices);
  let result;
  if (wasmSource instanceof WebAssembly.Module) {
    result = await WebAssembly.instantiate(wasmSource, imports);
    _wasm = { exports: result.exports };
  } else if (wasmSource instanceof Response || typeof wasmSource === 'string') {
    const response = typeof wasmSource === 'string' ? fetch(wasmSource) : wasmSource;
    result = await WebAssembly.instantiateStreaming(response, imports);
    _wasm = result.instance;
  } else if (wasmSource instanceof ArrayBuffer || ArrayBuffer.isView(wasmSource)) {
    const module = await WebAssembly.compile(wasmSource);
    result = await WebAssembly.instantiate(module, imports);
    _wasm = { exports: result.exports };
  } else {
    throw new Error('wasmSource must be a URL string, Response, WebAssembly.Module, or ArrayBuffer');
  }
  // Initialize WASM runtime.
  // Reactor mode (Rust, C/Emscripten): exports _initialize, returns normally.
  // Command mode (Go wasip1): exports _start, which runs main() then calls
  // proc_exit(0). proc_exit throws to unwind the stack; exit code 0 is success.
  if (_wasm.exports._initialize) {
    _wasm.exports._initialize();
  } else if (_wasm.exports._start) {
    try {
      _wasm.exports._start();
    } catch (e) {
      if (!e || e.wasiExitCode !== 0) throw e;
    }
  }
  return {
`, loaderName)

	for _, iface := range api.Interfaces {
		jsName := ToCamelCase(iface.Name)
		fmt.Fprintf(b, "    %s: _create%s(),\n", jsName, ToPascalCase(iface.Name))
	}

	b.WriteString(`  };
}

`)
}

// writeInterfaceWrappers writes a factory function for each interface that returns
// an object with all methods properly wrapped.
func writeInterfaceWrappers(b *strings.Builder, apiName string, api *model.APIDefinition, resolved resolver.ResolvedTypes) {
	for _, iface := range api.Interfaces {
		factoryName := "_create" + ToPascalCase(iface.Name)
		fmt.Fprintf(b, "// %s interface\nfunction %s() {\n  return {\n", iface.Name, factoryName)

		for i, method := range iface.Methods {
			writeMethodWrapper(b, apiName, iface.Name, &method, resolved)
			if i < len(iface.Methods)-1 {
				b.WriteString("\n")
			}
		}

		b.WriteString("  };\n}\n\n")
	}
}

// writeMethodWrapper writes a single method wrapper inside an interface object.
func writeMethodWrapper(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef, resolved resolver.ResolvedTypes) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	jsMethodName := ToCamelCase(method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil
	isFBReturn := hasReturn && model.IsFlatBufferType(method.Returns.Type)

	// Build JS parameter list
	var jsParams []string
	for _, p := range method.Parameters {
		jsParams = append(jsParams, ToCamelCase(p.Name))
	}

	paramStr := strings.Join(jsParams, ", ")
	fmt.Fprintf(b, "    %s(%s) {\n", jsMethodName, paramStr)

	// Marshalling prologue — allocate temporaries we'll need to free.
	// Track allocated pointers for cleanup.
	var marshalledParams []marshalledParam
	var cleanupPtrs []string

	for _, p := range method.Parameters {
		mp := marshalParam(p)
		marshalledParams = append(marshalledParams, mp)
		if mp.needsMarshal {
			for _, line := range mp.marshalLines {
				fmt.Fprintf(b, "      %s\n", line)
			}
			cleanupPtrs = append(cleanupPtrs, mp.cleanupPtrs...)
		}
	}

	// For fallible + return, allocate out-parameter space
	if hasError && hasReturn {
		outSize := wasmOutParamSize(method.Returns.Type, resolved)
		fmt.Fprintf(b, "      const _outPtr = _malloc(%d);\n", outSize)
		cleanupPtrs = append(cleanupPtrs, "_outPtr")
	}

	// For infallible + FlatBuffer return (sret convention on WASM32)
	if !hasError && isFBReturn {
		outSize := wasmOutParamSize(method.Returns.Type, resolved)
		fmt.Fprintf(b, "      const _outPtr = _malloc(%d);\n", outSize)
		cleanupPtrs = append(cleanupPtrs, "_outPtr")
	}

	// Build WASM call arguments
	var wasmArgs []string
	// Sret: prepend _outPtr as first argument
	if !hasError && isFBReturn {
		wasmArgs = append(wasmArgs, "_outPtr")
	}
	for _, mp := range marshalledParams {
		wasmArgs = append(wasmArgs, mp.wasmArgs...)
	}
	// Fallible out-parameter: append _outPtr as last argument
	if hasError && hasReturn {
		wasmArgs = append(wasmArgs, "_outPtr")
	}

	wasmArgStr := strings.Join(wasmArgs, ", ")

	// Wrap in try/finally if we need cleanup
	needsCleanup := len(cleanupPtrs) > 0
	if needsCleanup {
		b.WriteString("      try {\n")
	}

	indent := "      "
	if needsCleanup {
		indent = "        "
	}

	// The actual WASM call
	switch {
	case hasError && hasReturn:
		fmt.Fprintf(b, "%sconst _rc = _wasm.exports.%s(%s);\n", indent, funcName, wasmArgStr)
		fmt.Fprintf(b, "%sif (_rc !== 0) {\n", indent)
		fmt.Fprintf(b, "%s  throw new Error(`%s failed with error code ${_rc}`);\n", indent, jsMethodName)
		fmt.Fprintf(b, "%s}\n", indent)
		writeReturnRead(b, indent, method.Returns.Type, resolved)

	case hasError && !hasReturn:
		fmt.Fprintf(b, "%sconst _rc = _wasm.exports.%s(%s);\n", indent, funcName, wasmArgStr)
		fmt.Fprintf(b, "%sif (_rc !== 0) {\n", indent)
		fmt.Fprintf(b, "%s  throw new Error(`%s failed with error code ${_rc}`);\n", indent, jsMethodName)
		fmt.Fprintf(b, "%s}\n", indent)

	case !hasError && hasReturn:
		if isFBReturn {
			// Sret: call as void, read struct fields from _outPtr
			fmt.Fprintf(b, "%s_wasm.exports.%s(%s);\n", indent, funcName, wasmArgStr)
			writeJSFBSObjectReturn(b, indent, method.Returns.Type, resolved)
		} else {
			fmt.Fprintf(b, "%sconst _result = _wasm.exports.%s(%s);\n", indent, funcName, wasmArgStr)
			writeDirectReturn(b, indent, method.Returns.Type)
		}

	default:
		fmt.Fprintf(b, "%s_wasm.exports.%s(%s);\n", indent, funcName, wasmArgStr)
	}

	if needsCleanup {
		b.WriteString("      } finally {\n")
		for _, ptr := range cleanupPtrs {
			fmt.Fprintf(b, "        _free(%s);\n", ptr)
		}
		b.WriteString("      }\n")
	}

	b.WriteString("    },\n")
}

// marshalledParam tracks how a parameter is marshalled from JS to WASM.
type marshalledParam struct {
	needsMarshal bool
	marshalLines []string // Lines to emit in the prologue
	wasmArgs     []string // Argument expressions for the WASM call
	cleanupPtrs  []string // Pointers that need _free in finally block
}

// marshalParam determines how to pass a JS parameter to WASM.
func marshalParam(p model.ParameterDef) marshalledParam {
	jsName := ToCamelCase(p.Name)

	if model.IsString(p.Type) {
		ptrVar := "_" + jsName + "Ptr"
		return marshalledParam{
			needsMarshal: true,
			marshalLines: []string{
				fmt.Sprintf("const %s = _encodeString(%s);", ptrVar, jsName),
			},
			wasmArgs:    []string{ptrVar},
			cleanupPtrs: []string{ptrVar},
		}
	}

	if _, ok := model.IsBuffer(p.Type); ok {
		ptrVar := "_" + jsName + "Ptr"
		lenVar := "_" + jsName + "Len"
		return marshalledParam{
			needsMarshal: true,
			marshalLines: []string{
				fmt.Sprintf("const [%s, %s] = _copyBufferToWasm(%s);", ptrVar, lenVar, jsName),
			},
			wasmArgs:    []string{ptrVar, lenVar},
			cleanupPtrs: []string{ptrVar},
		}
	}

	if _, ok := model.IsHandle(p.Type); ok {
		return marshalledParam{
			wasmArgs: []string{jsName + "._ptr"},
		}
	}

	// Primitives and FlatBuffer types pass through directly
	return marshalledParam{
		wasmArgs: []string{jsName},
	}
}

// writeReturnRead writes code to read the out-parameter result after a successful fallible call.
func writeReturnRead(b *strings.Builder, indent string, retType string, resolved resolver.ResolvedTypes) {
	if handleName, ok := model.IsHandle(retType); ok {
		fmt.Fprintf(b, "%sconst _view = new DataView(_memoryBuffer());\n", indent)
		fmt.Fprintf(b, "%sconst _handleVal = _view.getUint32(_outPtr, true);\n", indent)
		fmt.Fprintf(b, "%sreturn new %s(_handleVal);\n", indent, handleName)
		return
	}

	if model.IsPrimitive(retType) {
		getter := wasmDataViewGetter(retType)
		fmt.Fprintf(b, "%sconst _view = new DataView(_memoryBuffer());\n", indent)
		fmt.Fprintf(b, "%sreturn _view.%s(_outPtr, true);\n", indent, getter)
		return
	}

	// FlatBuffer types: decode struct fields and return JS object
	writeJSFBSObjectReturn(b, indent, retType, resolved)
}

// writeDirectReturn writes code to return a direct (non-out-param) return value.
func writeDirectReturn(b *strings.Builder, indent string, retType string) {
	if handleName, ok := model.IsHandle(retType); ok {
		fmt.Fprintf(b, "%sreturn new %s(_result);\n", indent, handleName)
		return
	}

	// Primitives return directly from WASM — no wrapping needed.
	fmt.Fprintf(b, "%sreturn _result;\n", indent)
}

// writeModuleExports writes the default export and named exports.
func writeModuleExports(b *strings.Builder, apiName string, api *model.APIDefinition) {
	loaderName := ToCamelCase("load_" + apiName)

	// Export handle classes
	b.WriteString("// Exports\n")
	fmt.Fprintf(b, "export { %s };\n", loaderName)
	for _, h := range api.Handles {
		fmt.Fprintf(b, "export { %s };\n", h.Name)
	}
}

// wasmOutParamSize returns the byte size needed for an out-parameter of the given type.
func wasmOutParamSize(retType string, resolved resolver.ResolvedTypes) int {
	if _, ok := model.IsHandle(retType); ok {
		return 4 // handle is a 32-bit pointer
	}
	switch retType {
	case "int8", "uint8", "bool":
		return 1
	case "int16", "uint16":
		return 2
	case "int32", "uint32", "float32":
		return 4
	case "int64", "uint64", "float64":
		return 8
	default:
		// FlatBuffer types: compute actual struct size
		totalSize, _ := wasmStructLayout(retType, resolved)
		return totalSize
	}
}

// wasmDataViewGetter returns the DataView getter method for a primitive type.
func wasmDataViewGetter(t string) string {
	switch t {
	case "int8":
		return "getInt8"
	case "uint8":
		return "getUint8"
	case "int16":
		return "getInt16"
	case "uint16":
		return "getUint16"
	case "int32":
		return "getInt32"
	case "uint32":
		return "getUint32"
	case "float32":
		return "getFloat32"
	case "float64":
		return "getFloat64"
	case "int64":
		return "getBigInt64"
	case "uint64":
		return "getBigUint64"
	case "bool":
		return "getUint8"
	default:
		return "getUint32"
	}
}

// wasmFieldInfo describes a single field in a WASM32 struct layout.
type wasmFieldInfo struct {
	Name   string
	Type   string
	Offset int
	Size   int
}

// wasmFieldSize returns the byte size and alignment for a field type on WASM32.
func wasmFieldSize(fieldType string) (size, align int) {
	switch fieldType {
	case "string":
		return 4, 4 // const char* on wasm32
	case "bool":
		return 1, 1
	case "int8", "uint8":
		return 1, 1
	case "int16", "uint16":
		return 2, 2
	case "int32", "uint32", "float32":
		return 4, 4
	case "int64", "uint64", "float64":
		return 8, 8
	default:
		return 4, 4 // unknown — assume pointer-sized
	}
}

// wasmStructLayout computes the WASM32 struct layout with C alignment rules.
func wasmStructLayout(typeName string, resolved resolver.ResolvedTypes) (totalSize int, fields []wasmFieldInfo) {
	typeInfo, ok := resolved[typeName]
	if !ok {
		return 4, nil // unresolved — fallback to pointer size
	}

	offset := 0
	maxAlign := 1
	for _, f := range typeInfo.Fields {
		size, align := wasmFieldSize(f.Type)
		if align > maxAlign {
			maxAlign = align
		}
		// Align offset to field's natural alignment
		if rem := offset % align; rem != 0 {
			offset += align - rem
		}
		fields = append(fields, wasmFieldInfo{
			Name:   f.Name,
			Type:   f.Type,
			Offset: offset,
			Size:   size,
		})
		offset += size
	}

	// Pad total size to maximum alignment
	if rem := offset % maxAlign; rem != 0 {
		offset += maxAlign - rem
	}
	if offset == 0 {
		return 4, fields // empty struct — use pointer size
	}
	return offset, fields
}

// writeJSFBSObjectReturn emits JS code to read struct fields from _outPtr
// and return a plain JS object.
func writeJSFBSObjectReturn(b *strings.Builder, indent, retType string, resolved resolver.ResolvedTypes) {
	_, fields := wasmStructLayout(retType, resolved)
	if len(fields) == 0 {
		// Unresolved type — fallback to raw pointer
		fmt.Fprintf(b, "%sconst _view = new DataView(_memoryBuffer());\n", indent)
		fmt.Fprintf(b, "%sreturn _view.getUint32(_outPtr, true);\n", indent)
		return
	}

	fmt.Fprintf(b, "%sconst _view = new DataView(_memoryBuffer());\n", indent)
	var fieldExprs []string
	for _, f := range fields {
		jsFieldName := ToCamelCase(f.Name)
		var expr string
		switch f.Type {
		case "string":
			expr = fmt.Sprintf("_decodeString(_view.getUint32(_outPtr + %d, true))", f.Offset)
		case "bool":
			expr = fmt.Sprintf("_view.getUint8(_outPtr + %d) !== 0", f.Offset)
		case "int64":
			expr = fmt.Sprintf("_view.getBigInt64(_outPtr + %d, true)", f.Offset)
		case "uint64":
			expr = fmt.Sprintf("_view.getBigUint64(_outPtr + %d, true)", f.Offset)
		default:
			getter := wasmDataViewGetter(f.Type)
			expr = fmt.Sprintf("_view.%s(_outPtr + %d, true)", getter, f.Offset)
		}
		fieldExprs = append(fieldExprs, fmt.Sprintf("%s: %s", jsFieldName, expr))
	}

	fmt.Fprintf(b, "%sreturn { %s };\n", indent, strings.Join(fieldExprs, ", "))
}
