# xplattergy Code Generation Specification

Self-contained specification for implementing the xplattergy code generation tool.

## 1. Project Overview

xplattergy generates cross-platform API bindings from a single YAML API definition, targeting Android, iOS, Web, Windows, macOS, and Linux. **Implementation language agnostic** — any language with C ABI + WASM exports works.

## 2. Tool Implementation

Written in **Go** — single static binary, trivial cross-compilation.

### 2.1 Distribution

Prebuilt binaries for:
- x86_64 and arm64 Windows 10+
- arm64 macOS
- x86_64 Linux (statically linked, `CGO_ENABLED=0`)

Fallback: `build_codegen.sh` script + Makefile. Dependencies resolve via Go modules.

### 2.2 CLI Interface

```
xplattergy <command> [flags]
```

**Commands:**

| Command | Description |
|---------|-------------|
| `generate` | Generate C ABI header, platform bindings, and impl scaffolding |
| `validate` | Check API definition and FlatBuffers schemas without generating |
| `init` | Scaffold a new project with starter API definition and FBS files |
| `version` | Print version and exit |

**Global flags** (apply to all commands):

| Flag | Description |
|------|-------------|
| `-v, --verbose` | Verbose output |
| `-q, --quiet` | Suppress all output except errors |

**`generate` flags:**

| Flag | Description |
|------|-------------|
| `-o, --output <dir>` | Output directory (default: `./generated`) |
| `-f, --flatc <path>` | Path to FlatBuffers compiler |
| `--impl-lang <lang>` | Override `impl_lang` from API definition |
| `--targets <list>` | Override targets (comma-separated) |
| `--dry-run` | Show what would be generated without writing |
| `--clean` | Remove previously generated files first |
| `--skip-flatc` | Skip flatc invocation even if flatc is available (generated bindings will be incomplete) |

**`validate` flags:**

| Flag | Description |
|------|-------------|
| `-f, --flatc <path>` | Path to FlatBuffers compiler |

**`init` flags:**

| Flag | Description |
|------|-------------|
| `-n, --name <name>` | API name (default: `my_api`) |
| `--impl-lang <lang>` | Implementation language (default: `cpp`) |
| `-o, --output <dir>` | Output directory (default: current directory) |

**FlatBuffers compiler resolution order:**
1. `--flatc` flag
2. `XPLATTERGY_FLATC_PATH` environment variable
3. `flatc` in `PATH`

## 3. Inputs

### 3.1 API Definition YAML

Defines the API surface — handles, interfaces, and methods. Validated against a JSON Schema (Section 13).

**Top-level keys:**

```yaml
api:            # Required. Metadata.
flatbuffers:    # Required. FlatBuffers schema file paths.
handles:        # Optional. Opaque handle type definitions.
interfaces:     # Required. Grouped method definitions.
```

No additional top-level keys are permitted.

#### `api` — Metadata

| Field | Required | Type | Constraint |
|-------|----------|------|------------|
| `name` | yes | string | `snake_case`: `^[a-z][a-z0-9_]*$`. Used as prefix in all C ABI function names. |
| `version` | yes | string | Semver: `^\d+\.\d+\.\d+$` |
| `description` | no | string | Human-readable description |
| `impl_lang` | yes | string | One of: `cpp`, `rust`, `go`, `c` |
| `targets` | no | array | Subset of: `android`, `ios`, `web`, `windows`, `macos`, `linux`. If omitted, all targets. |

#### `flatbuffers` — Schema Includes

Array of `.fbs` file paths (relative to YAML file). At least one required. Types referenced by fully-qualified namespace (e.g., `Common.ErrorCode`).

#### `handles` — Opaque Handle Types

| Field | Required | Type | Constraint |
|-------|----------|------|------------|
| `name` | yes | string | `PascalCase`: `^[A-Z][a-zA-Z0-9]*$` |
| `description` | no | string | |

Referenced as `handle:Name` in method signatures.

#### `interfaces` — Method Groups

| Field | Required | Type | Constraint |
|-------|----------|------|------------|
| `name` | yes | string | `snake_case` |
| `description` | no | string | Human-readable description |
| `methods` | yes | array | At least one method |

#### Methods

| Field | Required | Type | Constraint |
|-------|----------|------|------------|
| `name` | yes | string | `snake_case` |
| `description` | no | string | |
| `parameters` | no | array | Ordered list of parameters |
| `returns` | no | object | Has a `type` field and optional `description` |
| `error` | no | string | Must be a FlatBuffers enum type reference |

#### Parameters

| Field | Required | Type | Constraint |
|-------|----------|------|------------|
| `name` | yes | string | `snake_case` |
| `type` | yes | string | See Section 4 (Type System) |
| `transfer` | no | string | `value` (default), `ref`, or `ref_mut` |
| `description` | no | string | |

### 3.2 FlatBuffers Schema Files (`.fbs`)

All data types (structs, enums, unions, tables, constants) are defined in `.fbs` files — the YAML never defines data types. The tool parses `.fbs` files to resolve type references and invokes `flatc` for per-language struct codegen.

## 4. Type System

### 4.1 Primitive Types (FlatBuffers naming)

| Type | C ABI | Size |
|------|-------|------|
| `int8` | `int8_t` | 1 byte |
| `int16` | `int16_t` | 2 bytes |
| `int32` | `int32_t` | 4 bytes |
| `int64` | `int64_t` | 8 bytes |
| `uint8` | `uint8_t` | 1 byte |
| `uint16` | `uint16_t` | 2 bytes |
| `uint32` | `uint32_t` | 4 bytes |
| `uint64` | `uint64_t` | 8 bytes |
| `float32` | `float` | 4 bytes |
| `float64` | `double` | 8 bytes |
| `bool` | `bool` | 1 byte |

Valid as both parameter and return types. Default transfer: `value`.

### 4.2 `string`

C ABI: `const char*`, null-terminated, UTF-8. Follows `ref` semantics implicitly.

**Parameter only.** Cannot be used as a return type. Return string data via FlatBuffer result types.

### 4.3 `buffer<T>`

Where T is any primitive type. Expands to two C parameters:

```c
const T* data, uint32_t data_len  // data_len = element count, NOT byte count
```

Transfer: `ref` produces `const T*`, `ref_mut` produces `T*`.

**Parameter only.** Cannot be used as a return type. Return buffer data via FlatBuffer result types.

### 4.4 `handle:Name`

Opaque handle from the `handles` section. C ABI: the handle typedef (e.g., `engine_handle`). Passed by value (pointer copy). `transfer` not applicable. Valid as both parameter and return types.

### 4.5 FlatBuffer Types

Fully-qualified namespace references (e.g., `Common.ErrorCode`). Must resolve in the included `.fbs` files. Valid as both parameter and return types; typically use `transfer: ref` as parameters.

**C type name mapping:** Dots → underscores (`Common.ErrorCode` → `Common_ErrorCode`). Used consistently across all generators.

**C header type emission:** Full `typedef enum`/`typedef struct` definitions for all referenced FlatBuffer types, emitted after handle typedefs and before platform services. Order: enums, then structs, then tables, alphabetically within each category.

### 4.6 Parameter vs Return Type Matrix

| Type | Parameter | Return |
|------|-----------|--------|
| Primitives | yes | yes |
| `string` | yes | **no** |
| `buffer<T>` | yes | **no** |
| `handle:Name` | yes | yes |
| FlatBuffer types | yes | yes |

## 5. C ABI Boundary Rules

### 5.1 Borrowing-Only Boundary

The side that allocates deallocates. No ownership transfer, release callbacks, or ref-counting across the FFI.

### 5.2 Transfer Semantics

| Mode | C ABI | Meaning |
|------|-------|---------|
| `value` | Pass by value | Copied. Default for primitives and handles. |
| `ref` | `const T*` | Immutable borrow for call duration. |
| `ref_mut` | `T*` | Mutable borrow for call duration. |

### 5.3 No Callbacks

Strictly unidirectional — bound language calls implementation, never the reverse. No function pointers cross the boundary. Reverse communication via shared ring buffer (Section 8).

### 5.4 No Singletons

All state is per-handle. Multiple instances can coexist.

## 6. C ABI Code Generation Rules

### 6.0 C Header Structure

The generated C header (`{api_name}.h`) has a fixed section ordering:

1. Include guard: `#ifndef {UPPER_SNAKE_CASE(api_name)}_H` / `#define ...`
2. Standard includes: `#include <stdint.h>`, `#include <stdbool.h>`
3. Symbol visibility export macro (see Section 6.6)
4. C++ compatibility: `#ifdef __cplusplus` / `extern "C" {` / `#endif`
5. Handle typedefs (if any handles defined)
6. FlatBuffer type definitions (enums, then structs, then tables — sorted alphabetically within each category)
7. Platform service declarations (no export macro — these are link-time provided)
8. Interface method declarations (grouped by interface, prefixed with export macro)
9. Closing C++ guard: `#ifdef __cplusplus` / `}` / `#endif`
10. Closing include guard: `#endif`

**Line wrapping:** Signatures exceeding 80 characters (including export macro) wrap to multi-line with 4-space indented parameters, one per line.

### 6.1 Function Naming

```
<api_name>_<interface_name>_<method_name>
```

Example: API `my_engine`, interface `renderer`, method `begin_frame`:
```c
int32_t my_engine_renderer_begin_frame(renderer_handle renderer);
```

### 6.2 Handle Typedefs

```c
typedef struct <lowercase_name>_s* <lowercase_name>_handle;
// e.g., typedef struct renderer_s* renderer_handle;
```

### 6.3 Error Convention

Methods with `error` return the error enum (FlatBuffers enum, success = `0`). Four patterns:

Fallible, no return value:
```c
int32_t myapi_renderer_begin_frame(renderer_handle renderer);
```

Fallible, with return value (return becomes out-parameter):
```c
int32_t myapi_lifecycle_create_engine(engine_handle* out_result);
```

Infallible, with return value:
```c
uint64_t myapi_scene_get_entity_count(scene_handle scene);
```

Infallible, no return value:
```c
void myapi_lifecycle_destroy_engine(engine_handle engine);
```

### 6.4 `buffer<T>` Expansion

A `buffer<T>` parameter becomes two C parameters: `const T* <name>, uint32_t <name>_len` (element count). `ref_mut` produces `T*` instead of `const T*`.

### 6.5 `string` Expansion

`string` becomes `const char* <name>` — always UTF-8, null-terminated.

### 6.6 Symbol Visibility / Export Macro

The C header emits a per-API export macro:

```c
/* Symbol visibility */
#if defined(_WIN32) || defined(_WIN64)
  #ifdef <UPPER_API_NAME>_BUILD
    #define <UPPER_API_NAME>_EXPORT __declspec(dllexport)
  #else
    #define <UPPER_API_NAME>_EXPORT __declspec(dllimport)
  #endif
#elif defined(__GNUC__) || defined(__clang__)
  #define <UPPER_API_NAME>_EXPORT __attribute__((visibility("default")))
#else
  #define <UPPER_API_NAME>_EXPORT
#endif
```

`<UPPER_API_NAME>` = API name in `UPPER_SNAKE_CASE` (e.g., `HELLO_XPLATTERGY`).

**Rules:**

- Emitted after standard includes, before `extern "C"` block
- `_BUILD` macro defined by build system when compiling the shared library; consumers get `dllimport` on Windows
- API methods prefixed with `_EXPORT`; platform services are NOT (link-time provided)
- Macro appears before return type: `MACRO return_type function_name(params)`
- Rust (`#[no_mangle]` + `cdylib`) and Go (`//export` + `c-shared`) handle export natively

## 7. Platform Binding Generation (Layer 1)

### 7.1 Targets

| Target | Output |
|--------|--------|
| `android` | Kotlin public API + JNI bridge calling C ABI functions |
| `ios` | Swift public API + C bridge calling C ABI functions |
| `macos` | Swift public API + C bridge calling C ABI functions |
| `web` | JavaScript public API + WASM bindings calling C ABI exports |
| `windows` | C API header (consumed directly or via language-specific FFI) |
| `linux` | C API header (consumed directly or via language-specific FFI) |

The C API header is always generated regardless of `targets`. All bindings route through the C ABI — WASM/JS uses C ABI exports (not embind/wasm-bindgen).

### 7.2 Kotlin/JNI Binding Details

Strings use JNI `GetStringUTFChars`/`ReleaseStringUTFChars`. Handles are wrapped in Kotlin classes with create/destroy mapped to constructor/`close()`.

**Output:** `{PascalCase(api_name)}.kt` + `{api_name}_jni.c`

**Naming:**

| Concept | Pattern | Example (`api_name: hello_world`) |
|---------|---------|-----------------------------------|
| Kotlin package | `{api_name}` with `_` → `.` | `hello.world` |
| Handle class | `{handle.Name}` (PascalCase from YAML) | `Engine` |
| Singleton object | `{PascalCase(api_name)}` | `HelloWorld` |
| JNI function | `Java_{package_path}_{Class}_{method}` | `Java_hello_world_Engine_start` |
| Error exception | `{FlatBufferCType}Exception` | `CommonErrorCodeException` |

**Type mappings:**

| xplattergy | Kotlin | JNI C |
|------------|--------|-------|
| `string` | `String` | `jstring` |
| `buffer<uint8>` | `ByteArray` | `jbyteArray` |
| `handle:X` | Handle class | `jlong` |
| Primitives | `Int`, `Long`, `Float`, `Boolean` | `jint`, `jlong`, `jfloat`, `jboolean` |
| FlatBuffer | `ByteArray` | `jbyteArray` |

**Method patterns:**
- Factory methods (create): return the handle class
- Instance methods: called on handle class, skip the handle parameter (it's `this`)
- Destroy: mapped to `close()` or similar teardown

### 7.3 Swift Binding Details

Strings use `withCString` or automatic bridging. Handles are wrapped in Swift classes with create/destroy mapped to static factory/`deinit`.

**Output:** `{PascalCase(api_name)}.swift`

**Naming:**

| Concept | Pattern | Example |
|---------|---------|---------|
| Handle class | `{handle.Name}` (PascalCase) | `Engine` |
| Error enum | `{FlatBufferCType}Error: Error` | `CommonErrorCodeError` |

**Type mappings:**

| xplattergy | Swift |
|------------|-------|
| `string` | `String` (marshalled via `withCString`) |
| `buffer<uint8>` | `Data` / `UnsafeMutableBufferPointer<UInt8>` |
| `handle:X` | Handle class |
| Primitives | `Int32`, `UInt64`, `Bool`, `Float`, `Double` |
| FlatBuffer | `UnsafePointer<Type>` / `UnsafeMutablePointer<Type>` |

### 7.4 JavaScript/WASM Binding Details

Strings use `TextEncoder`/`TextDecoder` for WASM linear memory marshalling. Handles are wrapped in JS objects with create/destroy mapped to constructor/`dispose()`.

**Output:** `{api_name}.js` (ES module)

**Naming:**

| Concept | Pattern | Example |
|---------|---------|---------|
| Handle class | `{handle.Name}` (PascalCase) | `Engine` |
| Loader function | `load{PascalCase(api_name)}` | `loadHelloWorld` |
| Method names | `{camelCase(method_name)}` | `beginFrame` |

**Patterns:**
- Handle classes use `#ptr` private field, zeroed on `dispose()`
- Buffers: copied into/out of WASM memory via `TypedArray`
- Fallible methods with return: allocate out-param space in WASM memory, read result via `DataView`
- Cleanup via `finally { _free(ptr) }` for temporaries
- Platform services passed as a services object to the loader: `logSink`, `resourceCount`, `resourceName`, `resourceExists`, `resourceSize`, `resourceRead`

## 8. Platform Services Layer

Link-time C functions with fixed signatures, implemented by the platform binding layer. The implementation calls these as plain C functions (WASM imports on web). Not callbacks.

### 8.1 Logging

```c
void <api_name>_log_sink(int32_t level, const char* tag, const char* message);
```

Global (not per-handle). Per-platform: Android → `android.util.Log`, iOS/macOS → `os_log`, Web → `console.*` via WASM import, Desktop → stderr.

### 8.2 Resource Access

```c
uint32_t <api_name>_resource_count(void);
int32_t  <api_name>_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  <api_name>_resource_exists(const char* name);
uint32_t <api_name>_resource_size(const char* name);
int32_t  <api_name>_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);
```

Per-platform: Android → `AssetManager` via JNI, iOS/macOS → `NSBundle.main`, Desktop → filesystem relative to executable, Web → synchronous lookup in pre-loaded in-memory store.

On web, resources need not be fully loaded before WASM initialization — `resource_exists` returns false and `resource_read` returns an error for unavailable resources.

### 8.3 Metrics

Structured FlatBuffer payloads delivered through the event queue polling mechanism, decoupled from logging. The app layer polls and routes to its reporting system.

### 8.4 Event Communication (Implementation → Bound Language)

The implementation communicates back via a **shared ring buffer** with platform-native signaling: `eventfd`/`pipe()` + `Looper` on Android/Linux, dispatch source on iOS/macOS, `Event` object on Windows, `requestAnimationFrame` polling on web main thread, `SharedArrayBuffer` + `Atomics.notify` on web Workers.

## 9. Implementation Interface & Scaffolding (Layer 2)

Controlled by `impl_lang`. Generates an abstract interface, C ABI shim, and stub implementation per language.

### 9.1 Generation Matrix

| `impl_lang` | Abstract Interface | C ABI Shim | Stub Implementation |
|-------------|-------------------|------------|---------------------|
| `cpp` | Abstract class (pure virtual) | `.cpp` with virtual dispatch shims | Concrete class with stubs |
| `rust` | Trait definition | `extern "C"` functions → trait impl | Skeleton `impl` block |
| `go` | Interface type | `//export` cgo → interface impl | Stub functions |
| `c` | — | — | — |

With `c`, only the C API header is generated.

### 9.2 Output File Manifest

**Always:** `{api_name}.h` (C ABI header)

**`impl_lang: cpp`** — `_interface.h` (abstract class), `_shim.cpp` (extern "C" shim), `_impl.h` + `_impl.cpp` (concrete stubs + factory)

**`impl_lang: rust`** — `_trait.rs` (traits), `_ffi.rs` (extern "C" shims), `_impl.rs` (stubs), `_types.rs` (if types exist)

**`impl_lang: go`** — `_interface.go` (interfaces), `_cgo.go` (//export shims), `_impl.go` (stubs), `_types.go` (if enums exist), and `_wasm.go` (if `web` is in targets; //go:wasmexport stubs for GOOS=wasip1 builds)

**Platform bindings:** `android` → `{PascalCase}.kt` + `_jni.c` | `ios`/`macos` → `{PascalCase}.swift` | `web` → `{api_name}.js` | `windows`/`linux` → C header only

#### FlatBuffer-generated files (via `flatc`)

`flatc` is invoked once per required language into `flatbuffers/<lang>/` subdirectories, determined by `targets` and `impl_lang`:

| Trigger | flatc flag | Output subdirectory | Example output |
|---------|-----------|---------------------|----------------|
| `targets` includes `android` | `--kotlin` | `flatbuffers/kotlin/` | `{Namespace}/{Type}.kt` per type |
| `targets` includes `ios` or `macos` | `--swift` | `flatbuffers/swift/` | `{schema}_generated.swift` |
| `targets` includes `web` | `--ts` | `flatbuffers/ts/` | `{schema}.ts` + per-type files |
| `impl_lang: cpp` | `--cpp` | `flatbuffers/cpp/` | `{schema}_generated.h` |
| `impl_lang: rust` | `--rust` | `flatbuffers/rust/` | `{schema}_generated.rs` |
| `impl_lang: go` | `--go` | `flatbuffers/go/` | Go package files |

Duplicates are deduplicated (e.g., `ios` + `macos` → single `--swift`). Use `--skip-flatc` to suppress.

### 9.3 Create/Destroy Method Detection

The **C++ shim generator** uses heuristics to detect factory and teardown methods, which get special shim bodies:

**Create method** — all of these must be true:
- Method returns a handle type (`returns.type` is `handle:X`)
- Method is fallible (has `error` field)
- Method has **no** handle input parameters (pure factory — no existing handle to delegate through)

**Destroy method** — all of these must be true:
- Method name starts with `"destroy"` or `"release"`
- First parameter is a handle type
- Method is infallible (no `error` field)
- Method has no return value

All other methods are "regular" — they find the first handle parameter, cast it to the implementation object, and delegate the call.

**Rust and Go** do not special-case create/destroy. Rust delegates all methods uniformly through trait dispatch (`TraitName::method(&Impl, ...)`); Go delegates through interface lookup and a handle map. Neither needs distinct factory/teardown bodies.

**Swift and Kotlin bindings** use `FindDestroyInfo()` (which matches `destroy_` or `release_` prefix with a single handle parameter) to wire `deinit`/`close()` to the appropriate C function, but do not apply the full infallible/void check.

### 9.4 C++ Generator Details

**Naming conventions:**

| Concept | Pattern | Example (`api_name: hello_world`) |
|---------|---------|-----------------------------------|
| Interface class | `{PascalCase(api_name)}Interface` | `HelloWorldInterface` |
| Impl class | `{PascalCase(api_name)}Impl` | `HelloWorldImpl` |
| Factory function | `create_{api_name}_instance()` | `create_hello_world_instance()` |
| Interface guard | `{UPPER_SNAKE_CASE(api_name)}_INTERFACE_H` | `HELLO_WORLD_INTERFACE_H` |
| Impl guard | `{UPPER_SNAKE_CASE(api_name)}_IMPL_H` | `HELLO_WORLD_IMPL_H` |

**Interface type mappings** (idiomatic C++, not raw C):

| xplattergy Type | C++ Interface Type |
|-----------------|--------------------|
| `string` | `std::string_view` |
| `buffer<T>` (ref) | `std::span<const T>` |
| `buffer<T>` (ref_mut) | `std::span<T>` |
| `handle:X` | `void*` (opaque in interface; shim does the cast) |
| Primitives | stdint types (`int32_t`, `float`, etc.) |
| FlatBuffer (ref) | `const Type*` |
| FlatBuffer (ref_mut) | `Type*` |

**Includes:** `<stdint.h>`, `<stdbool.h>`, `<cstddef>`, `<string_view>`, `<span>`, and `"{api_name}.h"` for FlatBuffer types.

**Shim behavior:** Includes C header + interface header. All functions in `extern "C" { }`, prefixed with `_EXPORT`.

- **Create:** Calls `create_{api_name}_instance()`, checks null, `reinterpret_cast` to handle, stores in `*out_result`, returns 0
- **Destroy:** `reinterpret_cast` handle → interface pointer, `delete`
- **Regular:** Cast handle → `{InterfaceClass}*`, wrap strings in `string_view()`, buffers in `span()`, call `self->{method}(args)`. Handle params pass through as `void*`.

### 9.5 Rust Generator Details

**Naming conventions:**

| Concept | Pattern | Example |
|---------|---------|---------|
| Trait name | `{PascalCase(interface_name)}` | `Lifecycle`, `Renderer` |
| ZST struct | `pub struct Impl;` | (always `Impl`) |
| Trait impl | `impl {TraitName} for Impl` | `impl Lifecycle for Impl` |

**ZST dispatch:** All trait methods take `&self`. A ZST `Impl` implements all traits. FFI calls via UFCS: `Lifecycle::create_greeter(&Impl, ...)` — compile-time dispatch, zero overhead.

**Trait type mappings:**

| xplattergy Type | Rust Trait Type | Rust FFI Type |
|-----------------|-----------------|---------------|
| `string` | `&str` | `*const c_char` |
| `buffer<T>` (ref) | `&[T]` | `*const T`, `u32` |
| `buffer<T>` (ref_mut) | `&mut [T]` | `*mut T`, `u32` |
| `handle:X` | `*mut c_void` | `*mut c_void` |
| Primitives | Rust types (`i32`, `u64`, `bool`, `f32`) | Same |
| FlatBuffer (ref) | `&Type` | `*const Type` |
| FlatBuffer (ref_mut) | `&mut Type` | `*mut Type` |

**FFI function pattern:**

```rust
#[no_mangle]
pub unsafe extern "C" fn {cabi_function_name}(params) -> i32 {
    // Convert: CStr::from_ptr(s).to_str().expect("invalid UTF-8")
    // Convert: std::slice::from_raw_parts(ptr, len as usize)
    // Convert: &*ptr (for FlatBuffer const ref)
    match {TraitName}::{method}(&Impl, converted_args) {
        Ok(val) => { *out_result = val; 0 }
        Err(e) => e as i32,
    }
}
```

Trait methods with `error` return `Result<T, ErrorType>`. The FFI shim matches Ok/Err → integer error code.

### 9.6 Go Generator Details

**Naming conventions:**

| Concept | Pattern | Example (`api_name: hello_world`) |
|---------|---------|-----------------------------------|
| Package name | `{api_name}` with underscores removed | `helloworld` |
| Interface name | `{PascalCase(interface_name)}` | `Lifecycle`, `Renderer` |

**Critical cgo rule:** Do NOT `#include` the generated C header in `//export` files — conflicting prototypes. Use local typedefs in the cgo preamble:

```go
/*
typedef struct greeter_s* greeter_handle;
typedef struct { const char* message; } Hello_Greeting;
*/
import "C"
```

**Handle map:** Go cannot pass Go pointers to C; uses an integer handle map instead:

```go
var (
    handles   sync.Map
    nextHandle uintptr
)

func allocHandle(impl interface{}) C.{handle_type} {
    h := atomic.AddUintptr(&nextHandle, 1)
    handles.Store(h, impl)
    return C.{handle_type}(unsafe.Pointer(h))
}

func lookupHandle(h C.{handle_type}) (impl, bool) {
    val, ok := handles.Load(uintptr(unsafe.Pointer(h)))
    ...
}

func freeHandle(h C.{handle_type}) {
    handles.Delete(uintptr(unsafe.Pointer(h)))
}
```

**`//export` function pattern:**

```go
//export {cabi_function_name}
func {cabi_function_name}(params) C.int32_t {
    impl, ok := lookupHandle(handle)
    goStr := C.GoString(cStr)
    // delegate to impl...
}
```

**Go type mappings:**

| xplattergy Type | Go Interface Type | cgo Type |
|-----------------|-------------------|----------|
| `string` | `string` | `*C.char` |
| `buffer<T>` | `[]T` | `*C.{ctype}`, `C.uint32_t` |
| `handle:X` | `uintptr` | `C.{handle_typedef}` |
| Primitives | Go types (`int32`, `uint64`, `bool`) | `C.int32_t`, `C.uint64_t`, `C._Bool` |
| FlatBuffer | `*C.{Type}` | `*C.{Type}` |

All shim code is mechanically derivable from the API definition — each method produces one shim function determined entirely by parameter types, transfer semantics, return type, and error convention.

## 10. Packaging and Distribution

The code gen tool produces source files only. Packaging those into deliverable platform artifacts is a build-system concern handled by Makefiles, Gradle, Xcode projects, etc. See ARCHITECTURE.md for the full provider/consumer model and per-platform package contents.

The **provider** (library author) runs codegen, implements the interface, and builds platform packages. The **consumer** (app developer) receives pre-built packages and calls the idiomatic binding — no dependency on `xplattergy` or `flatc`.

Consumer app projects typically use an `ensure-package` pattern: check for the pre-built package, and if absent, trigger the provider's package build.

## 11. Validation Rules

**Structural (JSON Schema):**
- YAML structure matches schema
- All names follow conventions (snake_case, PascalCase)
- FlatBuffer paths end in `.fbs`
- Version is semver
- `impl_lang` is valid enum
- `targets` values are valid

**Semantic (requires `.fbs` parsing):**
- All `handle:Name` references resolve to handles defined in the `handles` section
- All FlatBuffer type references (e.g., `Common.ErrorCode`) resolve to types in the included `.fbs` files
- `error` types are FlatBuffer enums
- `string` and `buffer<T>` are not used as return types
- `transfer` is not specified on handle parameters

## 12. Complete Example

### API Definition (`api_definition.yaml`)

```yaml
api:
  name: example_app_engine
  version: 0.1.0
  description: "Example interactive application engine API"
  impl_lang: cpp
  targets:
    - android
    - ios
    - web

flatbuffers:
  - schemas/geometry.fbs
  - schemas/input_events.fbs
  - schemas/rendering.fbs
  - schemas/scene.fbs
  - schemas/common.fbs

handles:
  - name: Engine
  - name: Renderer
  - name: Scene
  - name: Texture

interfaces:
  - name: lifecycle
    methods:
      - name: create_engine
        returns:
          type: handle:Engine
        error: Common.ErrorCode
      - name: destroy_engine
        parameters:
          - name: engine
            type: handle:Engine

  - name: renderer
    methods:
      - name: create_renderer
        parameters:
          - name: engine
            type: handle:Engine
          - name: config
            type: Rendering.RendererConfig
            transfer: ref
        returns:
          type: handle:Renderer
        error: Common.ErrorCode
      - name: destroy_renderer
        parameters:
          - name: renderer
            type: handle:Renderer
      - name: begin_frame
        parameters:
          - name: renderer
            type: handle:Renderer
        error: Common.ErrorCode
      - name: end_frame
        parameters:
          - name: renderer
            type: handle:Renderer
        error: Common.ErrorCode

  - name: texture
    methods:
      - name: load_texture_from_path
        parameters:
          - name: renderer
            type: handle:Renderer
          - name: path
            type: string
        returns:
          type: handle:Texture
        error: Common.ErrorCode
      - name: load_texture_from_buffer
        parameters:
          - name: renderer
            type: handle:Renderer
          - name: data
            type: buffer<uint8>
          - name: format
            type: Rendering.TextureFormat
        returns:
          type: handle:Texture
        error: Common.ErrorCode
      - name: destroy_texture
        parameters:
          - name: texture
            type: handle:Texture

  - name: input
    methods:
      - name: push_touch_events
        parameters:
          - name: engine
            type: handle:Engine
          - name: events
            type: Input.TouchEventBatch
            transfer: ref
        error: Common.ErrorCode

  - name: events
    methods:
      - name: poll_events
        parameters:
          - name: engine
            type: handle:Engine
          - name: events
            type: Common.EventQueue
            transfer: ref_mut
        error: Common.ErrorCode
```

### Expected Generated C Header (excerpt)

```c
#ifndef EXAMPLE_APP_ENGINE_H
#define EXAMPLE_APP_ENGINE_H

#include <stdint.h>
#include <stdbool.h>

/* Symbol visibility */
#if defined(_WIN32) || defined(_WIN64)
  #ifdef EXAMPLE_APP_ENGINE_BUILD
    #define EXAMPLE_APP_ENGINE_EXPORT __declspec(dllexport)
  #else
    #define EXAMPLE_APP_ENGINE_EXPORT __declspec(dllimport)
  #endif
#elif defined(__GNUC__) || defined(__clang__)
  #define EXAMPLE_APP_ENGINE_EXPORT __attribute__((visibility("default")))
#else
  #define EXAMPLE_APP_ENGINE_EXPORT
#endif

#ifdef __cplusplus
extern "C" {
#endif

typedef struct engine_s* engine_handle;
typedef struct renderer_s* renderer_handle;
typedef struct scene_s* scene_handle;
typedef struct texture_s* texture_handle;

/* FlatBuffer type definitions (enums, structs, tables) — see Section 6.0 */

/* Platform services — implement these per platform */
void example_app_engine_log_sink(int32_t level, const char* tag, const char* message);
uint32_t example_app_engine_resource_count(void);
int32_t  example_app_engine_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  example_app_engine_resource_exists(const char* name);
uint32_t example_app_engine_resource_size(const char* name);
int32_t  example_app_engine_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);

/* lifecycle */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_lifecycle_create_engine(
    engine_handle* out_result);
EXAMPLE_APP_ENGINE_EXPORT void example_app_engine_lifecycle_destroy_engine(
    engine_handle engine);

/* renderer */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_renderer_create_renderer(
    engine_handle engine,
    const Rendering_RendererConfig* config,
    renderer_handle* out_result);
EXAMPLE_APP_ENGINE_EXPORT void example_app_engine_renderer_destroy_renderer(
    renderer_handle renderer);
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_renderer_begin_frame(
    renderer_handle renderer);
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_renderer_end_frame(
    renderer_handle renderer);

/* texture */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_texture_load_texture_from_path(
    renderer_handle renderer,
    const char* path,
    texture_handle* out_result);
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_texture_load_texture_from_buffer(
    renderer_handle renderer,
    const uint8_t* data,
    uint32_t data_len,
    Rendering_TextureFormat format,
    texture_handle* out_result);
EXAMPLE_APP_ENGINE_EXPORT void example_app_engine_texture_destroy_texture(
    texture_handle texture);

/* input */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_input_push_touch_events(
    engine_handle engine,
    const Input_TouchEventBatch* events);

/* events */
EXAMPLE_APP_ENGINE_EXPORT int32_t example_app_engine_events_poll_events(
    engine_handle engine,
    Common_EventQueue* events);

#ifdef __cplusplus
}
#endif

#endif
```

## 13. JSON Schema

Full schema at `docs/api_definition_schema.json`. Key rules:

- `api`, `flatbuffers`, `interfaces` are required top-level keys
- `handles` is optional
- No additional properties at any level
- Parameter types match: `^(int8|...|bool|string|buffer<primitive>|handle:[A-Z]...|[A-Z]Namespace.Type...)$`
- Return types exclude `string` and `buffer<T>`
- `error` must be a FlatBuffer type reference
- `impl_lang` is one of: `cpp`, `rust`, `go`, `c`
- `targets` values are from: `android`, `ios`, `web`, `windows`, `macos`, `linux`

## 14. Future Considerations

Not to be implemented now, but do not foreclose:

- **Platform + language pairs** — `targets` may evolve to `linux:python`, `windows:lua`, etc. The C ABI foundation supports this; platform services would need load-time registration for dynamically loaded languages.
