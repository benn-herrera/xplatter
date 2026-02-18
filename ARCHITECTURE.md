# <img src="docs/logo.png" alt="drawing" width="32"/> xplatter Architecture Overview

## What It Is

xplatter is a code generation system that produces complete, ready-to-use API packages for a set of target platforms from a single API definition and implementation. It targets six platforms — Android, iOS, Web, Windows, macOS, and Linux — and is agnostic to the language used to implement the underlying library.

## Core Principles

* The Pure C ABI is the universal contract at the center of the system. Any implementation language that can export C-compatible functions and compile to WASM with C ABI exports is a valid choice. The code generation system neither knows nor cares what language is on the other side of that boundary.

* The app project consuming the API does not have to know or care about the implementation language. They get an idiomatic API in the natural language for the app without compromises to performance.

## System Layers

### Layer 1 — Core (Language-Agnostic)

This is the the first half of the delivered value. Given an API definition and FlatBuffers schemas as input, the code gen system produces:

- **Pure C API header** — the contract any implementation must satisfy. Includes handle typedefs, full C type definitions (enums, structs, tables) resolved from the FlatBuffers schemas using dot-to-underscore naming (`Common.ErrorCode` → `Common_ErrorCode`), platform service declarations, and export-annotated API function declarations.
- **Kotlin public API + JNI bridge** — calls the C API (Android)
- **Swift public API + C bridge** — calls the C API (iOS, macOS)
- **JavaScript public API + WASM bindings** — calls C ABI exports from the WASM module (Web, desktop via embedded browser/runtime)

All generated bindings route through the C ABI. The WASM/JS path uses C ABI exports from the WASM module rather than language-specific binding mechanisms, ensuring any implementation language that compiles to WASM works uniformly.

### Layer 2 — Implementation Interface & Scaffolding

This is the second half of the delivered value. The combination of these layers provides the "one implementation -> mulitply consumable API" that addresses the multiplatform performance-critical application pain point.
Code gen produces the complete implementation-side stack for the chosen language, as specified by the `impl_lang` field in the API definition. For each supported language, three things are generated:

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

## Provider and Consumer Roles

The system enforces a clean separation between two roles: the **API provider** (library author) and the **API consumer** (application developer).

### Provider — builds and packages

The provider authors the API definition and FlatBuffers schemas, runs `xplatter generate`, implements the generated abstract interface, and builds platform-specific packages. Each package bundles the compiled library with the idiomatic language binding for that platform:

| Platform | Package contents |
|----------|-----------------|
| iOS | XCFramework (static lib + headers) + SPM package with Swift binding |
| Android | `.so` per ABI (arm64-v8a, armeabi-v7a, x86_64, x86) + Kotlin binding |
| Web | `.wasm` module + JavaScript binding |
| Desktop (C/C++) | Shared library (`.dylib`/`.so`/`.dll`) + C header |
| Desktop (Swift) | Shared library + C header + Swift binding |

The provider owns the code gen tool, the build infrastructure, and the implementation source. None of these are visible to the consumer.

### Consumer — imports and calls

The consumer receives an opaque, pre-built package and imports it using the platform's standard mechanism — SPM dependency for iOS, Gradle JNI libs for Android, ES module or `<script>` tag for web, header + shared library for desktop. The consumer:

- Never runs `xplatter generate`
- Never sees the implementation source, the generated shim, or the C header (on platforms with higher-level bindings)
- Has no dependency on the code gen tool or any of its prerequisites
- Calls the API through the idiomatic binding in their platform's language

This mirrors the standard library distribution model: the provider is the shared library or framework author; the consumer is the application developer who links against it.

### In the examples

The `examples/hello-xplatter/` directory demonstrates both roles. The impl directories (`impl-c/`, `impl-cpp/`, `impl-rust/`, `impl-go/`) are provider-side — they run code gen, compile the implementation, and produce platform packages via `make packages`. The app directories (`app-ios/`, `app-android/`, `app-web/`, `app-desktop-cpp/`, `app-desktop-swift/`) are consumer-side — each has an `ensure-package` target that checks for the pre-built package and, if absent, triggers the provider's package build. But the app project itself only references the packaged artifacts. No app project runs code gen or reaches into implementation internals.

## The C ABI Boundary

The C ABI is a **borrowing boundary**, not an ownership transfer boundary. The side that allocates is the side that deallocates. This eliminates the need for release callbacks or ref-counting across the FFI.

Transfer semantics on parameters:

| Mode | Meaning | C ABI |
|------|---------|-------|
| `value` | Copied across the boundary. Default for primitives and handles. | Pass by value |
| `ref` | Caller owns, callee borrows immutably for the call duration. | `const T*` |
| `ref_mut` | Caller owns, callee borrows mutably for the call duration. | `T*` |

If the callee needs data to outlive the call, it copies explicitly.

**Strings** — `const char*`, null-terminated, UTF-8. Follows `ref` semantics (caller owns, callee borrows). Parameter-only — methods returning string data use a FlatBuffer result type.

**Buffers** — `buffer<T>` expands to two C parameters: `const T* data, uint32_t data_len` (element count, not byte count). Transfer semantics control const qualification. Parameter-only like strings.

**Opaque Handles** — typed `void*` with create/destroy lifecycle pairs. The implementation allocates on create and deallocates on destroy.

**Error Convention** — fallible methods return an error enum code. If the method also produces a return value, it's delivered through a final out-parameter pointer.

**Symbol Visibility** — a per-API export macro (`<UPPER_API_NAME>_EXPORT`) annotates API method declarations and definitions. Platform service functions are not annotated — they are link-time provided, not exported. When building with `-fvisibility=hidden`, only API functions are exported.

**No Callbacks** — the boundary is strictly unidirectional (bound language → implementation). The implementation communicates back via a shared ring buffer with platform-native signaling (`eventfd`/`pipe()` on Linux/Android, dispatch sources on iOS/macOS, `Event` objects on Windows, `SharedArrayBuffer` + `Atomics.notify` on web).

### Platform Services

While API methods flow from the bound language into the implementation, a small set of **link-time platform service functions** flow in the reverse direction — the implementation calls into platform-provided functionality. These are plain C functions with fixed signatures, resolved at link time (or via WASM imports on web). They are not callbacks — they are always available, not dynamically registered.

- **Logging** — routes to the platform-native log sink (`android.util.Log`, `os_log`, `console.log`). Zero-latency, crash-safe.
- **Resource Access** — uniform read-only access to bundled application resources. Maps to `AssetManager` (Android), `NSBundle` (iOS/macOS), filesystem (desktop), or a pre-loaded in-memory store (web).
- **Metrics** — structured FlatBuffer payloads delivered through the event queue, decoupled from logging. The app layer aggregates and routes metrics independently.

## FlatBuffers

FlatBuffers serves three roles in the system:

1. **Data structure IDL** — `.fbs` schemas define all data types (structs, enums, unions, tables, constants) used in the API. All type definitions live in FlatBuffers — the YAML API definition exclusively describes the API surface (interfaces, methods, handles). This avoids inventing a type definition language and gives users a well-documented, mature format they may already know.

2. **Per-language struct codegen** — the FlatBuffers compiler generates idiomatic data structure code for every target language. The xplatter code gen tool does not need to replicate this.

3. **Serialization** — zero-copy marshalling across the JNI boundary, binary-compatible save files across platforms, and a wire format for cross-device communication.

Types defined in FlatBuffers schemas are referenced in the API definition by their fully-qualified FlatBuffers namespace (e.g., `Geometry.Transform3D`).

## API Definition Format

The API is defined in YAML, validated by a [JSON Schema](./docs/api_definition_schema.json). See the [API Definition Specification](./docs/api_definition_spec.md) for the full reference and the [example definition](./docs/example_api_definition.yaml) for a working sample.

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
| Per-API export macro for symbol visibility | Ensures shared libraries export only API-defined symbols; platform services remain link-time provided. Windows `dllexport`/`dllimport`, gcc/clang visibility attributes, with graceful fallback. |
| Opaque platform packages for consumers | Consumers depend on a pre-built package per platform, not on codegen output or implementation internals. Standard distribution model — the provider builds and packages, the consumer imports and calls. |
