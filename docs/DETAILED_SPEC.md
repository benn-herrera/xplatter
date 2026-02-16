# xplattergy Code Generation Specification

This document is a self-contained specification for implementing the xplattergy code generation tool. It captures all design decisions, rules, and constraints needed to build the tool from scratch.

## 1. Project Overview

xplattergy ("splat-er-jee") is a code generation tool that produces cross-platform API bindings from a single YAML API definition. It targets six platforms: Android, iOS, Web, Windows, macOS, and Linux.

The tool is **implementation language agnostic** — any language that can export a Pure C ABI and compile to WASM with C ABI exports is a valid implementation choice. The generated bindings work the same regardless of what language is behind the C ABI boundary.

## 2. Tool Implementation

The code gen tool is written in **Go**. This produces a single static binary with trivial cross-compilation, eliminating runtime bootstrapping for end users.

### 2.1 Distribution

Prebuilt binaries for:
- x86_64 and arm64 Windows 10+
- arm64 macOS
- x86_64 Linux (statically linked, `CGO_ENABLED=0`)

Fallback: a `build_codegen.sh` script that handles Go detection/installation, plus a Makefile. Dependencies resolve automatically via `go build` (Go modules).

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

The tool consumes two types of input files:

### 3.1 API Definition YAML

Defines the API surface — handles, interfaces, and methods. Validated against a JSON Schema (see Section 13).

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

Array of file paths (relative to the API definition file). All must end in `.fbs`. At least one required.

Types from these files are referenced by fully-qualified FlatBuffers namespace (e.g., `Common.ErrorCode`).

#### `handles` — Opaque Handle Types

| Field | Required | Type | Constraint |
|-------|----------|------|------------|
| `name` | yes | string | `PascalCase`: `^[A-Z][a-zA-Z0-9]*$` |
| `description` | no | string | Human-readable description |

Referenced in method signatures as `handle:Name`.

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

All data types (structs, enums, unions, tables, constants) are defined here. The xplattergy YAML never defines data types — clean separation between API surface (YAML) and data types (FlatBuffers).

The code gen tool:
1. Parses `.fbs` files to resolve type references from the YAML
2. Invokes the FlatBuffers compiler (`flatc`) to generate per-language struct code

## 4. Type System

### 4.1 Primitive Types

Following FlatBuffers naming conventions:

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

Reference to an opaque handle defined in the `handles` section. C ABI: the handle typedef (e.g., `engine_handle`). Always passed by value (pointer copy). `transfer` field not applicable.

Valid as both parameter and return types.

### 4.5 FlatBuffer Types

Referenced by fully-qualified FlatBuffers namespace (e.g., `Common.ErrorCode`, `Geometry.Transform3D`). Must resolve to a type in one of the included `.fbs` files.

Valid as both parameter and return types. Should typically use `transfer: ref` as parameters.

**C type name mapping:** Dots become underscores. `Common.ErrorCode` → `Common_ErrorCode`, `Geometry.Transform3D` → `Geometry_Transform3D`. This mapping is used consistently across all generators.

**C header type emission:** The generated C header includes full `typedef enum`, `typedef struct` definitions for all FlatBuffer types referenced by the API. These are emitted after handle typedefs and before platform services. Enums are emitted first, then structs, then tables. Within each category, types are sorted alphabetically for deterministic output.

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

The side that allocates is the side that deallocates. No ownership transfer across the FFI. No release callbacks, no ref-counting.

### 5.2 Transfer Semantics

| Mode | C ABI | Meaning |
|------|-------|---------|
| `value` | Pass by value | Copied. Default for primitives and handles. |
| `ref` | `const T*` | Immutable borrow for call duration. |
| `ref_mut` | `T*` | Mutable borrow for call duration. |

### 5.3 No Callbacks

The C ABI is strictly unidirectional — the bound language calls into the implementation, never the reverse. No function pointers cross the boundary.

The implementation communicates back via a shared ring buffer with platform-native signaling (see Section 8).

### 5.4 No Singletons

All state is per-handle. Multiple engine instances can coexist. No architectural barriers to concurrent instances.

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

**Formatting rules:**

- Function signatures exceeding **80 characters** wrap to multi-line format with 4-space indented parameters, one per line
- Single-line: `EXPORT int32_t api_iface_method(type param);`
- Multi-line:
  ```c
  EXPORT int32_t api_iface_method(
      type param1,
      type param2);
  ```
- The 80-character check is on the full signature including the export macro prefix

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
```

Example for handle `Renderer`:
```c
typedef struct renderer_s* renderer_handle;
```

### 6.3 Error Convention

Methods with an `error` field return the error enum from the C function. The error enum must be a FlatBuffers enum type. Success is typically value `0`.

**Four method signature patterns:**

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

A single `buffer<T>` parameter in the YAML becomes two C parameters:

```yaml
- name: data
  type: buffer<uint8>
  transfer: ref
```

Generates:
```c
const uint8_t* data, uint32_t data_len
```

The second parameter is named `<param_name>_len` and is element count.

### 6.5 `string` Expansion

```yaml
- name: path
  type: string
```

Generates:
```c
const char* path
```

Always `const char*`, always UTF-8, always null-terminated.

### 6.6 Symbol Visibility / Export Macro

Generated shared libraries must export only API-defined symbols. The C header emits a per-API export macro that handles platform differences:

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

Where `<UPPER_API_NAME>` is the API name in `UPPER_SNAKE_CASE` (e.g., `HELLO_XPLATTERGY` for API name `hello_xplattergy`).

**Rules:**

- The export macro block is emitted after standard includes and before the `extern "C"` block
- The `<UPPER_API_NAME>_BUILD` macro is defined by the build system when compiling the shared library; consumers do not define it (getting `dllimport` on Windows)
- **API method declarations** (C header) and **API method definitions** (C++ shim) are prefixed with `<UPPER_API_NAME>_EXPORT`
- **Platform services** are NOT prefixed — they are provided by the consumer at link time, not exported by the library
- The macro appears before the return type: `MACRO return_type function_name(params)`
- Rust (`#[no_mangle]` + `cdylib`) and Go (`//export` + `c-shared`) handle symbol export natively — no codegen changes needed for those

## 7. Platform Binding Generation (Layer 1)

The core output. Language-agnostic, always generated.

### 7.1 Targets

| Target | Output |
|--------|--------|
| `android` | Kotlin public API + JNI bridge calling C ABI functions |
| `ios` | Swift public API + C bridge calling C ABI functions |
| `macos` | Swift public API + C bridge calling C ABI functions |
| `web` | JavaScript public API + WASM bindings calling C ABI exports |
| `windows` | C API header (consumed directly or via language-specific FFI) |
| `linux` | C API header (consumed directly or via language-specific FFI) |

The C API header is always generated regardless of `targets`.

All platform bindings route through the C ABI. The WASM/JS path uses C ABI exports (not Emscripten embind or wasm-bindgen), ensuring any implementation language that compiles to WASM works.

### 7.2 Per-Platform String Marshalling

| Platform | Mechanism |
|----------|-----------|
| Android/Kotlin | JNI `GetStringUTFChars` / `ReleaseStringUTFChars` |
| iOS/Swift | `String.withCString` or automatic bridging |
| Web/JS | `TextEncoder` into WASM linear memory |

### 7.3 Per-Platform Handle Wrapping

Generated platform bindings wrap opaque handles in idiomatic types — Kotlin classes, Swift classes, JS objects — with create/destroy methods mapped to constructor/close or destructor patterns.

### 7.4 Kotlin/JNI Binding Details

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

### 7.5 Swift Binding Details

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

**Handle lifecycle:** Factory methods are static class methods. Destroy methods are called from `deinit`.

### 7.6 JavaScript/WASM Binding Details

**Output:** `{api_name}.js` (ES module)

**Naming:**

| Concept | Pattern | Example |
|---------|---------|---------|
| Handle class | `{handle.Name}` (PascalCase) | `Engine` |
| Loader function | `load{PascalCase(api_name)}` | `loadHelloWorld` |
| Method names | `{camelCase(method_name)}` | `beginFrame` |

**Patterns:**
- Handle classes use `#ptr` private field, zeroed on `dispose()`
- Strings: `TextEncoder`/`TextDecoder` for WASM linear memory marshalling
- Buffers: copied into/out of WASM memory via `TypedArray`
- Fallible methods with return: allocate out-param space in WASM memory, read result via `DataView`
- Cleanup via `finally { _free(ptr) }` for temporaries

**Platform service imports:** Passed as a services object to the loader. Functions: `logSink`, `resourceCount`, `resourceName`, `resourceExists`, `resourceSize`, `resourceRead`.

## 8. Platform Services Layer

A small set of **link-time C functions** with fixed signatures that the platform binding layer implements. The implementation calls these as plain C functions. They are not callbacks.

On web, these are WASM imports.

### 8.1 Logging

```c
void <api_name>_log_sink(int32_t level, const char* tag, const char* message);
```

| Platform | Implementation |
|----------|---------------|
| Android | `android.util.Log` |
| iOS/macOS | `os_log` |
| Web | `console.log` / `console.warn` / `console.error` (via WASM import) |
| Desktop | Platform logging or stderr |

Zero-latency, crash-safe. Log sink is global (not per-handle) — appropriate for logging.

### 8.2 Resource Access

```c
uint32_t <api_name>_resource_count(void);
int32_t  <api_name>_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  <api_name>_resource_exists(const char* name);
uint32_t <api_name>_resource_size(const char* name);
int32_t  <api_name>_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);
```

| Platform | Implementation |
|----------|---------------|
| Android | `AssetManager` via JNI |
| iOS/macOS | `NSBundle.main` path resolution + file read |
| Desktop | Filesystem read relative to app/executable directory |
| Web | Synchronous lookup in pre-loaded in-memory store |

On web, resources do **not** need to be fully loaded before WASM initialization. `resource_exists` returns false and `resource_read` returns an error for not-yet-available resources. This avoids blocking time-to-first-pixel; web developers control the loading strategy.

### 8.3 Metrics

Structured FlatBuffer payloads delivered through the event queue polling mechanism. Decoupled from logging — different routing, batching, and aggregation needs. The app layer polls accumulated metrics and routes them to whatever reporting system they choose.

### 8.4 Event Communication (Implementation → Bound Language)

Since there are no callbacks, the implementation communicates back via a **shared ring buffer with platform-native signaling**:

| Platform | Signal Mechanism |
|----------|-----------------|
| Android/Linux | `eventfd` or `pipe()` integrated with `Looper` |
| iOS/macOS | Dispatch source integrated with run loop |
| Windows | `Event` object integrated with message loop |
| Web (main thread) | Poll in `requestAnimationFrame` |
| Web (Worker) | `SharedArrayBuffer` + `Atomics.notify` |

## 9. Implementation Interface & Scaffolding (Layer 2)

Controlled by the `impl_lang` field. For each supported language, an abstract interface, a C ABI shim, and a stub implementation are generated.

### 9.1 Generation Matrix

| `impl_lang` | Abstract Interface | C ABI Shim | Stub Implementation |
|-------------|-------------------|------------|---------------------|
| `cpp` | Abstract class with pure virtual methods | `.cpp` implementing each C function via virtual dispatch on the handle | Concrete class with stub method bodies |
| `rust` | Trait definition | `extern "C"` functions delegating to the trait impl | Skeleton `impl` block |
| `go` | Interface type | `//export` cgo functions delegating to the interface impl | Stub functions |
| `c` | — | — | — |

With `c`, only the C API header is generated. Use for pure C implementations or any language not in the front-door path.

### 9.2 Output File Manifest

Generation produces three categories of output: the C header (always), implementation scaffolding (per `impl_lang`), and platform bindings (per `targets`). Additionally, the tool invokes `flatc` to generate per-language data structure code.

#### C header (always generated)

| File | Purpose |
|------|---------|
| `{api_name}.h` | C ABI header with types, platform services, and function declarations |

#### Implementation scaffolding (per `impl_lang`)

**`impl_lang: cpp`** — 4 files:

| File | Purpose |
|------|---------|
| `{api_name}_interface.h` | Abstract class with pure virtual methods |
| `{api_name}_shim.cpp` | `extern "C"` shim delegating to interface |
| `{api_name}_impl.h` | Concrete class declaration |
| `{api_name}_impl.cpp` | Stub method bodies + factory function |

**`impl_lang: rust`** — 3–4 files:

| File | Purpose |
|------|---------|
| `{api_name}_trait.rs` | Trait definitions (one per interface) |
| `{api_name}_ffi.rs` | `#[no_mangle] extern "C"` shim functions |
| `{api_name}_impl.rs` | Stub `impl` blocks |
| `{api_name}_types.rs` | FlatBuffer type definitions (emitted if types exist) |

**`impl_lang: go`** — 3–4 files:

| File | Purpose |
|------|---------|
| `{api_name}_interface.go` | Go interface type definitions |
| `{api_name}_cgo.go` | `//export` cgo shim functions |
| `{api_name}_impl.go` | Stub interface implementation |
| `{api_name}_types.go` | Go enum constants (emitted if enums exist) |

#### Platform bindings (per `targets`)

| Target | File | Purpose |
|--------|------|---------|
| `android` | `{PascalCase(api_name)}.kt` | Kotlin public API |
| `android` | `{api_name}_jni.c` | JNI C bridge |
| `ios`, `macos` | `{PascalCase(api_name)}.swift` | Swift public API + C bridge |
| `web` | `{api_name}.js` | JavaScript/WASM ES module |
| `windows`, `linux` | — | C header consumed directly |

#### FlatBuffer-generated files (via `flatc`)

The tool invokes `flatc` once per required language, writing output into `flatbuffers/<lang>/` subdirectories. Which languages are invoked depends on both `targets` and `impl_lang`:

| Trigger | flatc flag | Output subdirectory | Example output |
|---------|-----------|---------------------|----------------|
| `targets` includes `android` | `--kotlin` | `flatbuffers/kotlin/` | `{Namespace}/{Type}.kt` per type |
| `targets` includes `ios` or `macos` | `--swift` | `flatbuffers/swift/` | `{schema}_generated.swift` |
| `targets` includes `web` | `--ts` | `flatbuffers/ts/` | `{schema}.ts` + per-type files |
| `impl_lang: cpp` | `--cpp` | `flatbuffers/cpp/` | `{schema}_generated.h` |
| `impl_lang: rust` | `--rust` | `flatbuffers/rust/` | `{schema}_generated.rs` |
| `impl_lang: go` | `--go` | `flatbuffers/go/` | Go package files |

Duplicate invocations are deduplicated (e.g., if `ios` target already triggered `--swift`, `macos` does not invoke it again). Use `--skip-flatc` to suppress all flatc invocations.

### 9.3 Create/Destroy Method Detection

The shim generators use heuristics to detect factory and teardown methods, which get special shim bodies:

**Create method** — all of these must be true:
- Method returns a handle type (`returns.type` is `handle:X`)
- Method is fallible (has `error` field)
- Method has **no** handle input parameters (pure factory — no existing handle to delegate through)

**Destroy method** — all of these must be true:
- Method name starts with `"destroy"`
- First parameter is a handle type
- Method is infallible (no `error` field)
- Method has no return value

All other methods are "regular" — they find the first handle parameter, cast it to the implementation object, and delegate the call.

### 9.4 C++ Generator Details

**Naming conventions:**

| Concept | Pattern | Example (`api_name: hello_world`) |
|---------|---------|-----------------------------------|
| Interface class | `{PascalCase(api_name)}Interface` | `HelloWorldInterface` |
| Impl class | `{PascalCase(api_name)}Impl` | `HelloWorldImpl` |
| Factory function | `create_{api_name}_instance()` | `create_hello_world_instance()` |
| Interface guard | `{UPPER_SNAKE_CASE(api_name)}_INTERFACE_H` | `HELLO_WORLD_INTERFACE_H` |
| Impl guard | `{UPPER_SNAKE_CASE(api_name)}_IMPL_H` | `HELLO_WORLD_IMPL_H` |

**Interface type mappings** (C++ interface uses idiomatic types, not raw C):

| xplattergy Type | C++ Interface Type |
|-----------------|--------------------|
| `string` | `std::string_view` |
| `buffer<T>` (ref) | `std::span<const T>` |
| `buffer<T>` (ref_mut) | `std::span<T>` |
| `handle:X` | `void*` (opaque in interface; shim does the cast) |
| Primitives | stdint types (`int32_t`, `float`, etc.) |
| FlatBuffer (ref) | `const Type*` |
| FlatBuffer (ref_mut) | `Type*` |

**Interface header includes:** `<stdint.h>`, `<stdbool.h>`, `<cstddef>`, `<string_view>`, `<span>`, and the C header (`"{api_name}.h"`) for FlatBuffer type definitions.

**Shim behavior:**

The shim `#include`s the C header and the interface header. All functions are inside `extern "C" { }`. Each function definition is prefixed with the export macro (`<UPPER_API_NAME>_EXPORT`).

- **Create shim:** Calls `create_{api_name}_instance()`, checks for null, casts to handle via `reinterpret_cast`, stores in `*out_result`, returns 0
- **Destroy shim:** Casts handle back to interface pointer via `reinterpret_cast`, calls `delete`
- **Regular shim:** Casts handle param to `{InterfaceClass}*`, wraps string params in `std::string_view()`, wraps buffer params in `std::span()`, calls `self->{method}(args)`. Handle params pass through (the interface takes `void*`).

### 9.5 Rust Generator Details

**Naming conventions:**

| Concept | Pattern | Example |
|---------|---------|---------|
| Trait name | `{PascalCase(interface_name)}` | `Lifecycle`, `Renderer` |
| ZST struct | `pub struct Impl;` | (always `Impl`) |
| Trait impl | `impl {TraitName} for Impl` | `impl Lifecycle for Impl` |

**ZST dispatch pattern:**

All trait methods take `&self`. A zero-sized type `Impl` implements all traits. FFI functions call trait methods via UFCS: `Lifecycle::create_greeter(&Impl, ...)`. This provides compile-time dispatch with zero runtime overhead.

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

**Error handling:** Trait methods with `error` return `Result<T, ErrorType>`. The FFI shim matches on Ok/Err and converts to integer error code.

### 9.6 Go Generator Details

**Naming conventions:**

| Concept | Pattern | Example (`api_name: hello_world`) |
|---------|---------|-----------------------------------|
| Package name | `{api_name}` with underscores removed | `helloworld` |
| Interface name | `{PascalCase(interface_name)}` | `Lifecycle`, `Renderer` |

**Critical cgo rule:** Do NOT `#include` the generated C header in files that contain `//export` functions. This causes conflicting prototype errors between the C header declarations and cgo's auto-generated wrappers. Instead, use local C type definitions in the cgo preamble comment:

```go
/*
typedef struct greeter_s* greeter_handle;
typedef struct { const char* message; } Hello_Greeting;
*/
import "C"
```

**Handle map pattern:**

Go cannot pass Go pointers to C. Instead, use an integer handle map:

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

### 9.7 Design Rationale

All shim code is mechanically derivable from the API definition — no analysis or domain knowledge required. Each method produces one shim function determined entirely by parameter types, transfer semantics, return type, and error convention. This:

- Eliminates formulaic code from the consumer's responsibility
- Reduces expensive AI inference on mechanical tasks during agentic coding
- Keeps the consumer focused solely on implementing business logic in the abstract interface

## 10. Packaging and Distribution

The code generation tool produces source files. Turning those into deliverable packages that application developers can consume is a build-system concern, not a codegen concern — but the system is designed with a clear separation between the two roles.

### 10.1 Provider and Consumer

The **API provider** (library author) runs the code gen tool, implements the generated abstract interface, and builds platform-specific packages. The **API consumer** (application developer) receives an opaque, pre-built package and calls the API through the idiomatic binding for their platform.

The consumer never runs `xplattergy generate`, never sees implementation source or generated shims, and has no dependency on the code gen tool or flatc.

### 10.2 Platform Package Contents

Each platform package bundles the compiled library with the generated language binding:

| Platform | Package contents | Consumer imports via |
|----------|-----------------|---------------------|
| iOS | XCFramework (static lib + headers) + SPM package with Swift binding | SPM dependency |
| Android | `.so` per ABI (arm64-v8a, armeabi-v7a, x86_64, x86) + Kotlin binding | Gradle JNI libs |
| Web | `.wasm` module + JavaScript ES module | `<script>` or ES import |
| Desktop (C/C++) | Shared library (`.dylib`/`.so`/`.dll`) + C header | `-I`/`-L` flags |
| Desktop (Swift) | Shared library + C header + Swift binding | `swiftc -import-objc-header` |

On platforms with higher-level bindings (iOS, Android, web), the consumer never sees the C header — they only interact with the Kotlin, Swift, or JavaScript API.

### 10.3 Build System Integration

The packaging step is implemented in build system rules (Makefiles, Gradle, Xcode projects), not in the code gen tool. A typical provider build pipeline:

1. `xplattergy generate` — produce source files into `generated/`
2. Compile the implementation + generated shim into a library (static or shared)
3. Copy the library + the appropriate language binding into a distribution directory

The hello-xplattergy examples demonstrate this with per-platform `package-*` Make targets (`package-ios`, `package-android`, `package-web`, `package-desktop`) that produce self-contained distribution directories under `dist/`.

Consumer app projects use an `ensure-package` pattern: check for the pre-built package, and if absent, trigger the provider's package build. The app itself only references packaged artifacts — never generated source or build intermediates.

## 11. Validation Rules

The `validate` command checks:

**Structural (via JSON Schema):**
- YAML structure matches schema
- All names follow conventions (snake_case, PascalCase)
- FlatBuffer paths end in `.fbs`
- Version is semver
- `impl_lang` is valid enum
- `targets` values are valid

**Semantic (requires parsing `.fbs` files):**
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
    description: "Top-level application engine instance"
  - name: Renderer
    description: "Rendering context bound to a platform surface"
  - name: Scene
    description: "Scene graph container"
  - name: Texture
    description: "GPU texture resource"

interfaces:
  - name: lifecycle
    description: "Engine creation, configuration, and teardown"
    methods:
      - name: create_engine
        description: "Create and initialize the engine instance"
        returns:
          type: handle:Engine
        error: Common.ErrorCode
      - name: destroy_engine
        description: "Shut down and release all engine resources"
        parameters:
          - name: engine
            type: handle:Engine

  - name: renderer
    description: "Rendering context and frame management"
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
    description: "Texture resource loading and management"
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
    description: "Input event processing"
    methods:
      - name: push_touch_events
        description: "Hot path - minimal marshalling overhead"
        parameters:
          - name: engine
            type: handle:Engine
          - name: events
            type: Input.TouchEventBatch
            transfer: ref
        error: Common.ErrorCode

  - name: events
    description: "Poll for events from the implementation"
    methods:
      - name: poll_events
        description: "Drain pending events. Call once per frame."
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

The full JSON Schema for validating API definition YAML files is maintained at `docs/api_definition_schema.json`. Key validation rules:

- `api`, `flatbuffers`, `interfaces` are required top-level keys
- `handles` is optional
- No additional properties at any level
- Parameter types match: `^(int8|...|bool|string|buffer<primitive>|handle:[A-Z]...|[A-Z]Namespace.Type...)$`
- Return types exclude `string` and `buffer<T>`
- `error` must be a FlatBuffer type reference
- `impl_lang` is one of: `cpp`, `rust`, `go`, `c`
- `targets` values are from: `android`, `ios`, `web`, `windows`, `macos`, `linux`

## 14. Future Considerations

These are acknowledged but **not** to be implemented now. Do not design for them, but do not make choices that close the door:

- **Platform + language pairs for targets** — `targets` may evolve to support pairs like `linux:python` or `windows:lua` for binding to additional languages. Current single platform names would remain valid as shorthand. The C ABI foundation already supports this; the main adaptation needed is that platform services (log sink, resources) would use load-time registration rather than link-time resolution for dynamically loaded languages.
