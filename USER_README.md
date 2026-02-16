# <img src="docs/logo.png" alt="drawing" width="32"/> xplattergy

xplattergy *(splat-er-jee)* generates cross-platform API bindings from a single YAML definition. 

Define your API and implement it once in the cross-platform system language of your choice and get a set of ready-to-use, idiomatic target language API packages for mobile, web, and desktop.

## Quick Start

### Prerequisites

- A prebuilt `xplattergy` binary (included in the SDK) or Go 1.25+ to build from source
- FlatBuffers compiler (`flatc`) for per-language struct codegen (required)
- make (GNU Make)

Additional tools are required depending on which target platforms you select — see [Platform Tooling Requirements](#platform-tooling-requirements) below.

### Build from Source (if no prebuilt binary for your platform)

```bash
./build_codegen.sh
```

### Try the Examples

Working examples with API implementations in C, C++, Rust, and Go with front end consumer apps targeting mobile, desktop, and web live under `examples/`. hello-xplattergy defines a simple greeter API. All examples generate bindings, implement them, and run tests.

```bash
# Run all implementation examples
cd examples
make test-hello-examples

# Run individually
make test-hello-impl-c
make test-hello-impl-cpp
make test-hello-impl-rust
make test-hello-impl-go

# Run app examples (consumer-side binding usage)
make test-hello-app-desktop-cpp
make test-hello-app-desktop-swift     # macOS only
make test-hello-app-ios               # macOS only (builds for simulator)
make test-hello-app-android           # requires Android SDK + NDK
```

### Usage

```bash
# Generate all bindings
xplattergy generate my_api.yaml -o generated

# Generate for specific platforms only
xplattergy generate my_api.yaml -o generated --targets android,ios,web

# Validate without generating
xplattergy validate my_api.yaml

# Scaffold a new project
xplattergy init --name my_api --impl-lang cpp

# Preview what would be generated
xplattergy generate my_api.yaml --dry-run
```

### Workflow

1. Define your API in YAML
2. Define your data types in FlatBuffers schemas (`.fbs` files)
3. Run `xplattergy generate your_api.yaml -o generated`
4. Implement the generated abstract interface in your language (C++, Rust, Go, or plain C)
5. Build your implementation against the generated C header and shim
  * xplattergy should be integrated into your project build system as a code gen dependency

## CLI Reference

```
xplattergy <command> [flags]
```

### Commands

| Command | Description |
|---------|-------------|
| `generate` | Generate C ABI header, platform bindings, and impl scaffolding |
| `validate` | Check API definition and FlatBuffers schemas without generating |
| `init` | Scaffold a new project with starter API definition and FBS files |
| `version` | Print version and exit |

### `generate` Flags

| Flag | Description |
|------|-------------|
| `-o, --output <dir>` | Output directory (default: `./generated`) |
| `-f, --flatc <path>` | Path to FlatBuffers compiler |
| `--impl-lang <lang>` | Override `impl_lang` from API definition |
| `--targets <list>` | Override targets (comma-separated) |
| `--dry-run` | Show what would be generated without writing |
| `--clean` | Remove previously generated files first |
| `-v, --verbose` | Verbose output |
| `-q, --quiet` | Suppress all output except errors |

### `validate` Flags

| Flag | Description |
|------|-------------|
| `-f, --flatc <path>` | Path to FlatBuffers compiler |
| `-v, --verbose` | Show detailed validation results |

### `init` Flags

| Flag | Description |
|------|-------------|
| `-n, --name <name>` | API name (default: `my_api`) |
| `--impl-lang <lang>` | Implementation language (default: `cpp`) |
| `-o, --output <dir>` | Output directory (default: current directory) |

**FlatBuffers compiler resolution order:**
1. `--flatc` flag
2. `XPLATTERGY_FLATC_PATH` environment variable
3. `flatc` in `PATH`

## API Definition Format

API definitions are YAML files with four top-level keys:

```yaml
api:            # Required. Metadata.
flatbuffers:    # Required. FlatBuffers schema file paths.
handles:        # Optional. Opaque handle type definitions.
interfaces:     # Required. Grouped method definitions.
```

No additional top-level keys are permitted.

### `api` — Metadata

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | `snake_case` name, used as prefix in all C ABI function names |
| `version` | yes | Semver (`1.0.0`) |
| `description` | no | Human-readable description |
| `impl_lang` | yes | One of: `cpp`, `rust`, `go`, `c` |
| `targets` | no | Subset of: `android`, `ios`, `web`, `windows`, `macos`, `linux`. If omitted, all targets. |

### `flatbuffers` — Schema Includes

Array of `.fbs` file paths relative to the API definition file:

```yaml
flatbuffers:
  - schemas/common.fbs
  - schemas/rendering.fbs
```

### `handles` — Opaque Handle Types

```yaml
handles:
  - name: Engine
    description: "Top-level application engine instance"
  - name: Renderer
    description: "Rendering context bound to a platform surface"
```

Handle names are PascalCase. Referenced in methods as `handle:Engine`.

### `interfaces` — Method Groups

```yaml
interfaces:
  - name: lifecycle
    description: "Engine creation and teardown"
    methods:
      - name: create_engine
        description: "Create and initialize the engine"
        returns:
          type: handle:Engine
        error: Common.ErrorCode
      - name: destroy_engine
        parameters:
          - name: engine
            type: handle:Engine
```

### Method Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | `snake_case` method name |
| `parameters` | no | Ordered list of parameters |
| `returns` | no | Return type (`type` field + optional `description`) |
| `error` | no | FlatBuffers enum type for error codes |

### Parameter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | `snake_case` parameter name |
| `type` | yes | Type (see Type System below) |
| `transfer` | no | `value` (default), `ref`, or `ref_mut` |

### Error Convention

Methods with an `error` field return the error enum as an `int32_t` from the C function. Success is value `0`. If the method also has a return value, that value becomes a final out-parameter:

```c
// Fallible, no return value
int32_t my_api_renderer_begin_frame(renderer_handle renderer);

// Fallible, with return value (return becomes out-parameter)
int32_t my_api_lifecycle_create_engine(engine_handle* out_result);

// Infallible, with return value
uint64_t my_api_scene_get_entity_count(scene_handle scene);

// Infallible, no return value
void my_api_lifecycle_destroy_engine(engine_handle engine);
```

## Type System

### Primitive Types

| Type | C ABI | Size |
|------|-------|------|
| `int8` / `int16` / `int32` / `int64` | `int8_t` / `int16_t` / `int32_t` / `int64_t` | 1 / 2 / 4 / 8 bytes |
| `uint8` / `uint16` / `uint32` / `uint64` | `uint8_t` / `uint16_t` / `uint32_t` / `uint64_t` | 1 / 2 / 4 / 8 bytes |
| `float32` / `float64` | `float` / `double` | 4 / 8 bytes |
| `bool` | `bool` | 1 byte |

Valid as both parameter and return types. Default transfer: `value`.

### `string`

C ABI: `const char*`, null-terminated, UTF-8. **Parameter only** — cannot be used as a return type. Return string data via FlatBuffer result types.

### `buffer<T>`

Where T is any primitive type. Expands to two C parameters:

```c
const uint8_t* data, uint32_t data_len  // data_len = element count, NOT byte count
```

`ref` produces `const T*`, `ref_mut` produces `T*`. **Parameter only.**

### `handle:Name`

Reference to an opaque handle. C ABI: `typedef struct name_s* name_handle`. Always passed by value (pointer copy).

### FlatBuffer Types

Referenced by fully-qualified FlatBuffers namespace: `Common.ErrorCode`, `Geometry.Transform3D`. Must resolve to a type in one of the included `.fbs` files. In generated C code, dots become underscores: `Common.ErrorCode` becomes `Common_ErrorCode`.

### Parameter vs Return Rules

| Type | Parameter | Return |
|------|-----------|--------|
| Primitives | yes | yes |
| `string` | yes | **no** |
| `buffer<T>` | yes | **no** |
| `handle:Name` | yes | yes |
| FlatBuffer types | yes | yes |

### Transfer Semantics

| Mode | C ABI | Meaning |
|------|-------|---------|
| `value` | Pass by value | Copied. Default for primitives and handles. |
| `ref` | `const T*` | Immutable borrow for call duration. |
| `ref_mut` | `T*` | Mutable borrow for call duration. |

The FFI boundary is borrowing-only. The side that allocates is the side that deallocates. No ownership transfer, no release callbacks.

## What Gets Generated

### C Header

Always generated. Named `{api_name}.h`. Contains:

1. Include guard and standard includes
2. Symbol visibility export macro (see below)
3. `extern "C"` guards for C++ compatibility
4. Handle typedefs (`typedef struct engine_s* engine_handle;`)
5. FlatBuffer type definitions (C enums, structs, tables from `.fbs` schemas)
6. Platform service declarations (no export macro — link-time provided)
7. API function declarations (prefixed with export macro)

### Platform Bindings

| File | Platform |
|------|----------|
| `{PascalCase(api_name)}.kt` + `{api_name}_jni.c` | Android (Kotlin + JNI bridge) |
| `{PascalCase(api_name)}.swift` | iOS / macOS (Swift + C interop) |
| `{api_name}.js` | Web (JS/WASM ES module) |

### Symbol Visibility / Export Macro

The C header emits a per-API export macro:

```c
/* Symbol visibility */
#if defined(_WIN32) || defined(_WIN64)
  #ifdef MY_API_BUILD
    #define MY_API_EXPORT __declspec(dllexport)
  #else
    #define MY_API_EXPORT __declspec(dllimport)
  #endif
#elif defined(__GNUC__) || defined(__clang__)
  #define MY_API_EXPORT __attribute__((visibility("default")))
#else
  #define MY_API_EXPORT
#endif
```

**When building your shared library**, define `{UPPER_API_NAME}_BUILD` so API symbols are exported. Consumers of the library leave it undefined (getting `dllimport` on Windows).

```bash
# GCC/Clang: compile with hidden visibility, export only annotated symbols
cc -fvisibility=hidden -DMY_API_BUILD -shared -o libmy_api.so ...

# MSVC
cl /DMY_API_BUILD /LD my_api.c ...
```

API functions are annotated with the export macro. Platform services are **not** — they are provided by the consumer at link time, not exported by the library.

### Platform Services

Every generated API includes platform service functions that the implementation can call. These are declared in the C header without the export macro — the platform binding layer provides the implementation at link time.

**Logging:**
```c
void <api_name>_log_sink(int32_t level, const char* tag, const char* message);
```

**Resource access:**
```c
uint32_t <api_name>_resource_count(void);
int32_t  <api_name>_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  <api_name>_resource_exists(const char* name);
uint32_t <api_name>_resource_size(const char* name);
int32_t  <api_name>_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);
```

| Platform | Logging | Resources |
|----------|---------|-----------|
| Android | `android.util.Log` | `AssetManager` via JNI |
| iOS/macOS | `os_log` | `NSBundle.main` |
| Web | `console.log/warn/error` | Pre-loaded in-memory store |
| Desktop | stderr | Filesystem relative to executable |

## Implementation Languages

The `impl_lang` field controls what implementation scaffolding is generated. With any option, the C header and platform bindings are always produced.

### C (`impl_lang: c`)

No scaffolding. You implement the C ABI functions declared in the header directly.

### C++ (`impl_lang: cpp`)

Generates 4 files: abstract interface, C ABI shim, impl header, impl source.

- Interface class: `{PascalCase}Interface` with pure virtual methods
- Strings become `std::string_view`, buffers become `std::span<const T>`, handles are `void*`
- Factory function: `create_{api_name}_instance()` returns your concrete class
- The shim handles all C↔C++ marshalling — you implement the virtual methods

### Rust (`impl_lang: rust`)

Generates 3-4 files: trait definitions, FFI shim, stub impl, optional types.

- Each interface becomes a trait with `&self` methods
- FFI layer uses `#[no_mangle] pub unsafe extern "C" fn`
- Zero-sized type `Impl` dispatches via UFCS for zero runtime overhead
- Fallible trait methods return `Result<T, ErrorType>`

### Go (`impl_lang: go`)

Generates 3-4 files: interface definitions, cgo shim, stub impl, optional types.

- Each interface becomes a Go interface type
- Handles use an integer handle map (`sync.Map`) since cgo prohibits passing Go pointers to C
- Package name is the API name with underscores removed

## FlatBuffers Integration

All data types (structs, enums, tables) are defined in FlatBuffers `.fbs` schemas — the YAML API definition only describes the API surface. This gives you:

- **Per-language struct codegen** via `flatc` for every target language
- **Zero-copy serialization** across the JNI boundary
- **Binary-compatible data** across platforms (save files, network messages)
- **Schema evolution** (adding/deprecating fields) with forward/backward compatibility

Types are referenced in the YAML by fully-qualified namespace: `Common.ErrorCode`, `Geometry.Transform3D`. The generator parses `.fbs` files to resolve these references and emits C type definitions in the header.

## Complete Example

### API Definition

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
  - schemas/rendering.fbs
  - schemas/common.fbs

handles:
  - name: Engine
    description: "Top-level application engine instance"
  - name: Renderer
    description: "Rendering context bound to a platform surface"

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
```

### Generated C Header (excerpt)

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

/* Platform services — implement these per platform */
void example_app_engine_log_sink(int32_t level, const char* tag, const char* message);
uint32_t example_app_engine_resource_count(void);
/* ... */

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

#ifdef __cplusplus
}
#endif

#endif
```

## Platform Tooling Requirements

Only the tools for your selected target platforms are required. The `targets` field in the API definition controls which bindings are generated. Not all targets can be built on every host OS.

### Implementation Language

The `impl_lang` field determines what implementation scaffolding is generated. Each language requires its own compiler/toolchain:

| `impl_lang` | Required Tools |
|-------------|---------------|
| `c` | C11 compiler (cc/gcc/clang) |
| `cpp` | C++20 compiler (c++/g++/clang++), C11 compiler |
| `rust` | Rust toolchain (rustc + cargo) |
| `go` | Go 1.25+, cgo-compatible C compiler |

### Target Platforms

| Target | Required Tools | Host OS |
|--------|---------------|---------|
| `android` | Android SDK, NDK r29+, JDK 17+ | macOS, Linux, Windows |
| `ios` | Xcode (provides `xcrun`, `xcodebuild`, `lipo`, `ar`, `swiftc`) | macOS only |
| `macos` | Swift compiler (`swiftc`), C++20 compiler | macOS only |
| `web` | Emscripten (emcc) or wasm-compatible toolchain | macOS, Linux, Windows |
| `windows` | MSVC or MinGW (cl/gcc) | Windows (cross-compile possible with MinGW) |
| `linux` | GCC or Clang | Linux (cross-compile possible) |

### Android-Specific Setup

Building for Android requires:

1. **Android SDK** with platform API 35 (or your `compileSdk` target)
2. **Android NDK r29+** — provides cross-compilation toolchains for ARM64, ARMv7, x86_64, x86
3. **JDK 17+** — for Gradle and the Kotlin compiler (JDK 21 recommended for Android Studio compatibility)
4. **Gradle** — a wrapper (`gradlew`) is typically included in the project

The NDK is expected at `$HOME/Library/Android/sdk/ndk/<version>` (macOS) or `$ANDROID_HOME/ndk/<version>`. Install it via Android Studio's SDK Manager or `sdkmanager --install "ndk;<version>"`.

### iOS-Specific Setup

Building for iOS requires macOS with Xcode installed. The Xcode command-line tools provide all necessary compilers and utilities:

- `xcrun` — SDK-aware tool dispatch
- `xcodebuild` — project/workspace builds and XCFramework creation
- `lipo` — universal binary creation
- `ar` — static library archival
- `swiftc` — Swift compiler

Install Xcode from the Mac App Store, then run `xcode-select --install` to ensure command-line tools are available.

### Host OS / Target Availability

| Host OS | Buildable Targets |
|---------|-------------------|
| macOS | android, ios, macos, web |
| Linux | android, linux, web |
| Windows | android, windows, web |

## Design Principles

**Borrowing-only boundary** — the side that allocates is the side that deallocates. No ownership transfer across the FFI. No release callbacks, no ref-counting.

**No callbacks** — the C ABI is strictly unidirectional (bound language calls implementation). The implementation communicates back via a shared ring buffer with platform-native signaling.

**No singletons** — all state is per-handle. Multiple instances can coexist.

**Implementation language agnostic** — any language that can export a Pure C ABI and compile to WASM with C ABI exports is a valid choice.
