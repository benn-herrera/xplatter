# <img src="docs/logo.png" alt="drawing" width="32"/> xplattergy Architecture Overview

## What It Is

xplattergy is a code generation system that produces cross-platform API bindings from a single API definition. It targets six platforms — Android, iOS, Web, Windows, macOS, and Linux — and is agnostic to the language used to implement the underlying library.

## Core Principle

The Pure C ABI is the universal contract at the center of the system. Any implementation language that can export C-compatible functions and compile to WASM with C ABI exports is a valid choice. The code generation system neither knows nor cares what language is on the other side of that boundary.

## System Layers

### Layer 1 — Core (Language-Agnostic)

This is the product. Given an API definition and FlatBuffers schemas as input, the code gen system produces:

- **Pure C API header** — the contract any implementation must satisfy
- **Kotlin public API + JNI bridge** — calls the C API (Android)
- **Swift public API + C bridge** — calls the C API (iOS, macOS)
- **JavaScript public API + WASM bindings** — calls C ABI exports from the WASM module (Web, desktop via embedded browser/runtime)

All generated bindings route through the C ABI. The WASM/JS path uses C ABI exports from the WASM module rather than language-specific binding mechanisms, ensuring any implementation language that compiles to WASM works uniformly.

### Layer 2 — Implementation Interface & Scaffolding

Code gen that produces the complete implementation-side stack for the chosen language, as specified by the `impl_lang` field in the API definition. For each supported language, three things are generated:

1. **Abstract interface** — the API contract expressed in the implementation language's idioms
2. **C ABI shim** — generated bridge code that implements the exported C functions by delegating to the abstract interface, handling all marshalling between C types and language-native types
3. **Stub implementation** — a skeleton that satisfies the abstract interface, ready for the consumer to fill in

| `impl_lang` | Abstract Interface | C ABI Shim | Stubs |
|-------------|-------------------|------------|-------|
| `cpp` | Abstract class with pure virtual methods | `.cpp` implementing each C function via virtual dispatch on the handle | Concrete class with stub method bodies |
| `rust` | Trait definition | `extern "C"` functions delegating to the trait impl | Skeleton `impl` block |
| `go` | Interface type | `//export` cgo functions delegating to the interface impl | Stub functions |
| `c` | — | — | — |

With `c`, only the C API header is generated. This is the option for pure C implementations or any language not in the front-door path — the consumer implements the exported C functions directly.

For supported languages, the consumer never writes C. They implement the abstract interface in their language, build, and the generated shim handles all C ABI compliance. Adding a new implementation language target never touches Layer 1.

## Data Flow

```
┌──────────────────────┐     ┌──────────────────────┐
│  API Definition YAML │     │  FlatBuffers Schemas │
│  (authored)          │     │  (.fbs, authored)    │
└─────────┬────────────┘     └─────────┬────────────┘
          │                            │
          ▼                            ▼
┌─────────────────────────────────────────────────────┐
│              Code Gen Tool (Go binary)              │
│                                                     │
│  Reads API definition + FlatBuffers schemas.        │
│  Invokes FlatBuffers compiler for per-language      │
│  struct code. Generates C ABI header, platform      │
│  bindings, and impl interface + scaffolding.        │
└──┬──────────┬──────────┬──────────┬──────────┬──────┘
   │          │          │          │          │
   ▼          ▼          ▼          ▼          ▼
 C API     Kotlin/     Swift/    JS/WASM    Impl language
 Header    JNI         C bridge  bindings   output:
   │                                        ┌───────────┐
   │                                        │ Abstract  │
   │                                        │ interface │
   │                                        ├───────────┤
   │                                        │ C ABI     │
   │                                        │ shim      │
   │                                        ├───────────┤
   │                                        │ Stub      │
   │                                        │ impl      │
   │                                        └────┬──────┘
   │                                             │
   │   ┌─────────────────────────────────────┐   │
   │   │  User Implementation                │   │
   │   │                                     │   │
   │   │  Implements the abstract interface  │<──┘
   └──>│  in their chosen language. The      │
       │  generated C ABI shim handles all   │
       │  FFI compliance — no C required.    │
       └──────────────┬──────────────────────┘
                      │
                      │ builds to
                      ▼
       ┌──────────────────────────────────┐
       │  Native library    WASM module   │
       │  (.so/.dylib/.dll) (.wasm)       │
       └──────┬───────────────┬───────────┘
              │               │
              ▼               ▼
        Kotlin/JNI        JS/WASM
        Swift/C bridge    bindings
        link and load     load and call
        the native lib    the WASM module
```

## The C ABI Boundary

The C ABI is a **borrowing boundary**, not an ownership transfer boundary. The side that allocates is the side that deallocates. This eliminates the need for release callbacks or ref-counting across the FFI.

Transfer semantics on parameters:

| Mode | Meaning | C ABI |
|------|---------|-------|
| `value` | Copied across the boundary. Default for primitives and handles. | Pass by value |
| `ref` | Caller owns, callee borrows immutably for the call duration. | `const T*` |
| `ref_mut` | Caller owns, callee borrows mutably for the call duration. | `T*` |

If the callee needs data to outlive the call, it copies explicitly.

### Strings

`string` is a first-class parameter type: `const char*`, null-terminated, UTF-8 at the C ABI level. It follows `ref` semantics — the caller owns the string memory, the callee borrows it for the call duration. Code gen handles per-platform marshalling (JNI `GetStringUTFChars`, Swift `withCString`, JS `TextEncoder` into WASM linear memory).

`string` is **parameter-only** — it cannot be used as a return type. Methods that need to return string data do so via a FlatBuffer result type. This preserves the borrowing boundary: no ambiguity about who owns returned string memory.

### Buffers

`buffer<T>` is a first-class parameter type for passing contiguous arrays of primitive data (raw bytes, audio samples, vertex data, video frames). T must be a primitive type. At the C ABI level it expands to two parameters:

```c
const T* data, uint32_t data_len  // data_len is element count, not byte count
```

Transfer semantics apply: `ref` produces `const T*`, `ref_mut` produces `T*`. Like `string`, `buffer<T>` is **parameter-only** — methods that need to return buffer data do so via a FlatBuffer result type. Element count (not byte count) eliminates a class of sizing errors when T is wider than one byte.

### Opaque Handles

Implementation-managed objects are represented as opaque handles — typed `void*` at the C level. They follow create/destroy lifecycle pairs. The implementation allocates on create and deallocates on destroy.

### Error Convention

Methods that can fail declare an error enum type. The C ABI function returns the error code. If the method also produces a return value, that value is delivered through a final out-parameter pointer.

### No Callbacks

The C ABI boundary is strictly unidirectional — the bound language calls into the implementation, never the reverse. Callbacks (function pointers crossing the FFI from implementation back to caller) are intentionally excluded because:

- They introduce threading hazards: the implementation may fire callbacks from background threads, requiring JNI thread attachment, main-thread dispatch on iOS/Swift, and conflicting with JS single-threaded execution.
- On the web, WASM and JS on the main thread cannot run concurrently. A "callback" from WASM into JS is really just a synchronous call within an already-initiated request — no different from returning data as an out-parameter.
- They roughly double the code gen surface area (function pointer wrapping, user_data management, platform-specific idioms).

Instead, the implementation communicates back to the bound language via a **shared ring buffer with platform-native signaling**: `eventfd`/`pipe()` on Android/Linux, dispatch sources on iOS/macOS, `Event` objects on Windows, and `SharedArrayBuffer` + `Atomics.notify` (via Web Workers) or `requestAnimationFrame` polling on the web. This pattern integrates with each platform's native event loop without crossing the FFI in the reverse direction.

### Platform Services Layer

While the API methods flow from the bound language into the implementation, a small set of **link-time platform service functions** flow in the reverse direction — the implementation calls into platform-provided functionality. These are not callbacks: they are plain C functions with fixed signatures, resolved at link time (or via WASM imports on web). The code gen always produces them as part of the platform bindings.

#### Logging

```c
void <api_name>_log_sink(int32_t level, const char* tag, const char* message);
```

The generated platform bindings provide the implementation: `android.util.Log` on Android, `os_log` on iOS/macOS, `console.log/warn/error` on web (via WASM import). Zero-latency, crash-safe, platform-native log output.

#### Resource Access

```c
uint32_t <api_name>_resource_count(void);
int32_t  <api_name>_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);
int32_t  <api_name>_resource_exists(const char* name);
uint32_t <api_name>_resource_size(const char* name);
int32_t  <api_name>_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);
```

Provides the implementation with uniform access to read-only resources bundled with the application. The generated platform bindings map to the native mechanism:

| Platform | Implementation |
|----------|---------------|
| Android | `AssetManager` via JNI |
| iOS/macOS | `NSBundle.main` path resolution + file read |
| Desktop | Filesystem read relative to app/executable directory |
| Web | Synchronous lookup in a pre-loaded in-memory store |

On web, the JS layer populates an in-memory resource store. Resources do not need to be fully loaded before WASM initialization — `resource_exists` returns false and `resource_read` returns an error for not-yet-available resources, allowing the implementation to handle deferred availability gracefully. This avoids blocking time-to-first-pixel on a full resource pre-load; web developers control the loading strategy (pre-load critical resources, lazy-load the rest) to suit their performance requirements.

#### Metrics

Metrics are decoupled from both logging and resource access. Metrics are **structured FlatBuffer payloads** (names, values, dimensions, timestamps) delivered through the event queue polling mechanism. This allows the app layer to aggregate, batch, and route metrics to whatever reporting system they choose, independent of the logging pipeline.

## FlatBuffers

FlatBuffers serves three roles in the system:

1. **Data structure IDL** — `.fbs` schemas define all data types (structs, enums, unions, tables, constants) used in the API. All type definitions live in FlatBuffers — the YAML API definition exclusively describes the API surface (interfaces, methods, handles). This avoids inventing a type definition language and gives users a well-documented, mature format they may already know.

2. **Per-language struct codegen** — the FlatBuffers compiler generates idiomatic data structure code for every target language. The xplattergy code gen tool does not need to replicate this.

3. **Serialization** — zero-copy marshalling across the JNI boundary, binary-compatible save files across platforms, and a wire format for cross-device communication.

Types defined in FlatBuffers schemas are referenced in the API definition by their fully-qualified FlatBuffers namespace (e.g., `Geometry.Transform3D`).

## API Definition Format

The API is defined in YAML, validated by a JSON Schema.

For the full specification, see:

- [API Definition YAML Specification](./docs/api_definition_spec.md)
- [API Definition JSON Schema](./docs/api_definition_schema.json)

## Distribution

The code gen tool is distributed as prebuilt binaries:

- **x86_64 and arm64 Windows 10+**
- **arm64 macOS**
- **x86_64 Linux** (statically linked, `CGO_ENABLED=0`)

For platforms not covered by prebuilt binaries (e.g. uncommon Linux configurations), a `build_codegen.sh` script handles Go detection/installation and builds the binary from source via a Makefile.

## First-Party Contrib

Shipped alongside the core tool but not baked into the code gen:

- **Input event FlatBuffers schema** (`input_events.fbs`) — standardized touch and keyboard event structures
- **Platform event translation utilities** — Kotlin (Android), Swift (iOS/macOS), JavaScript (DOM) functions that convert platform-native input events to/from the FlatBuffer representation
- **Reference examples** — C++ and Rust input event translation as documented examples

These serve as both practical utilities and end-to-end integration test cases for validating marshalling throughput (targeting 120+ FPS input processing).

- **Build integration examples** — minimal project skeletons showing how to wire xplattergy code gen into standard build systems:
  - C++: CMake
  - Rust: `build.rs`
  - Go: `go generate` + Makefile

  Each example demonstrates invoking the code gen tool as a build step, placing generated files, and compiling the result end-to-end.

## Technical Decisions

| Decision | Rationale |
|----------|-----------|
| Pure C ABI as universal contract | Only ABI that every language and platform can produce and consume without special tooling. |
| FlatBuffers for data types | Mature IDL with per-language codegen, zero-copy serialization, and schema evolution — three capabilities we'd otherwise have to build. |
| YAML for API definitions | Human-authored format with comments, minimal syntactic noise; paired with JSON Schema for validation and editor tooling. |
| Go for code gen tool | Compiles to a single static binary with trivial cross-compilation — eliminates runtime/environment bootstrapping for end users. |
| Borrowing-only FFI boundary | Ownership transfer across FFI requires release callbacks and inverted control flow; borrowing keeps both sides simple and auditable. |
| WASM via C ABI exports (not language-specific) | Ensures any implementation language that compiles to WASM works, rather than coupling to Emscripten/embind or wasm-bindgen. |
| C++, Rust, Go impl interface + C ABI shim + stubs | The three viable languages with mature support for all six target platforms, C ABI export, and linear-memory WASM compilation. Generated shim means the consumer never writes C. |
| Prebuilt binaries + build-from-source fallback | Covers the common case with zero friction while providing a reliable escape hatch for uncommon platforms. |
| Input events as first-party contrib | Near-universal need, exercises the hot path, provides a concrete performance benchmark for the generated bindings. |
| Link-time platform services (logging, resources) | Fixed, narrow interfaces that the code gen always produces; link-time resolution avoids callback machinery while giving the implementation access to platform-native capabilities. |
| Metrics via event queue, not logging | Metrics are structured data suited to batching and aggregation; routing them through a text log sink would mean unnecessary serialize/parse overhead. |
| `string` as parameter-only type | Input strings are straightforward (`const char*`, UTF-8, borrowed). Returning strings creates ownership ambiguity; FlatBuffer result types handle that cleanly. |
