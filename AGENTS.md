# Agent Guide

Go code-gen tool that reads YAML API definitions + FlatBuffers schemas and outputs a C ABI header, platform bindings (Kotlin/JNI, Swift, JS/WASM), and implementation scaffolding (C++, Rust, Go).

The key value proposition is single source of truth definition + single implementation -> deployability across all commercially valuable general purpose platforms, freeing human and AI attention budgets from scaffolding maintenance.

## Build and Test

```bash
# Build the tool
make build                    # → bin/xplatter

# Run all Go unit tests
make test                     # quiet
make test-v                   # verbose

# Code quality
make fmt                      # gofmt
make vet                      # go vet

# Validate example API definition against JSON Schema
make validate

# Per-package tests (from src/)
cd src && go test ./gen/...       # generators only
cd src && go test ./loader/...    # YAML loader
cd src && go test ./resolver/...  # FBS type resolution
cd src && go test ./validate/...  # semantic validation

# Example integration tests (from examples/)
cd examples && make test-hello-impls              # all impl languages
cd examples && make test-hello-impl-c
cd examples && make test-hello-impl-cpp
cd examples && make test-hello-impl-rust
cd examples && make test-hello-impl-go

# App tests — consumer-side binding usage (from examples/)
cd examples && make test-hello-apps               # all apps
cd examples && make test-hello-app-desktop-cpp
cd examples && make test-hello-app-desktop-swift   # macOS only
cd examples && make test-hello-app-ios             # macOS only
cd examples && make test-hello-app-android         # requires Android SDK + NDK
cd examples && make test-hello-app-web             # requires Emscripten

# on-shot test of entire hello example impl vs app matrix
make test-examples-hello-impl-app-matrix
```

## Source Layout

```
src/
  cmd/          CLI commands (generate, validate, init, version)
  gen/          All code generators — the core of the tool
    generator.go   Generator interface, registry, target/impl-lang mappings
    context.go     Generation context (API def + resolved types + output dir)
    util.go        Naming utilities (PascalCase, camelCase, C type helpers)
    flatc.go       FlatBuffers compiler invocation
    cheader.go     C API header generator (always runs)
    kotlin.go      Kotlin + JNI bridge (Android)
    swift.go       Swift + C bridge (iOS, macOS)
    jswasm.go      JavaScript + WASM bindings (Web)
    impl_cpp.go    C++ impl interface + shim + stubs
    impl_rust.go   Rust impl trait + FFI + stubs
    impl_go.go     Go impl interface + cgo shim + stubs
    impl_go_wasmexport.go  Go WASM impl (//go:wasmexport scaffolding for wasip1)
    *_test.go      Golden-file tests for each generator
  model/        API model types (APIDefinition, InterfaceDef, MethodDef, etc.)
    api.go         Type classification: IsPrimitive, IsString, IsBuffer, IsHandle, IsFlatBufferType
  loader/       YAML loading + JSON Schema validation
  resolver/     FlatBuffers schema parsing and type resolution
  validate/     Semantic validation of API definitions
  testdata/     Test YAML definitions and golden/ output files
schemas/        Top-level FlatBuffers schemas (core_types.fbs, input_events.fbs)
```

## Code Conventions

- **Formatting**: `gofmt` (enforced by `make fmt`)
- **Generator registration**: each generator file has an `init()` that calls `gen.Register(name, factory)`:
  ```go
  func init() {
      Register("cheader", func() Generator { return &CHeaderGenerator{} })
  }
  ```
- **Generator interface**: two methods
  ```go
  type Generator interface {
      Name() string
      Generate(ctx *Context) ([]*OutputFile, error)
  }
  ```
- **Output**: generators return `[]*OutputFile` where `OutputFile` has `Path` (relative) and `Content` ([]byte)
- **Golden-file testing**: `loadTestAPI(t, "minimal.yaml")` loads a test API + FBS types into a `*Context`, then compares `Generate()` output against files in `testdata/golden/`
- **Naming utilities** in `gen/util.go`: `ToPascalCase`, `ToCamelCase`, `UpperSnakeCase`, `CABIFunctionName`, `HandleTypedefName`, `HandleStructName`, `ExportMacroName`, `BuildMacroName`, `CParamType`, `CReturnType`, `COutParamType`, `CollectErrorTypes`, `FindDestroyInfo`
- **Type classification** in `model/api.go`: `IsPrimitive(t)`, `IsString(t)`, `IsBuffer(t)`, `IsHandle(t)`, `IsFlatBufferType(t)`, `PrimitiveCType(t)`, `FlatBufferCType(t)`, `HandleToSnake(name)`, `EffectiveTargets()`, `HandleByName(name)`

## Generator Architecture

All generators follow the same pattern: iterate API interfaces, iterate methods, emit language-specific output.

- **C header** (`cheader`) — always generated regardless of targets. Produces the C ABI contract.
- **Target mapping** via `GeneratorsForTarget(target) []string`:
  - `"android"` → `["kotlin"]`
  - `"ios"`, `"macos"` → `["swift"]`
  - `"web"` → `["jswasm"]`
  - `"windows"`, `"linux"` → `nil` (C header only)
- **Impl-lang mapping** via `GeneratorsForImplLang(lang) string`:
  - `"cpp"` → `"impl_cpp"`, `"rust"` → `"impl_rust"`, `"go"` → `"impl_go"`, `"c"` → `""` (no scaffolding)
- **Impl-lang + targets mapping** via `GeneratorsForImplLangAndTargets(lang, targets) []string`:
  - `"go"` + `"web"` target → `["impl_go_wasm"]` (adds `//go:wasmexport` scaffolding alongside cgo shim)
- **Create/destroy detection**: `FindDestroyInfo()` locates destroy methods for handles. Create methods are detected by heuristic (returns handle + fallible + no handle input). Generators use these to emit factory/teardown bodies in shims.

## Adding a New Generator

1. Create `src/gen/<name>.go` implementing the `Generator` interface
2. Add `init()` calling `Register("<name>", factory)`
3. Create `src/gen/<name>_test.go` using `loadTestAPI` + golden-file comparison
4. Add expected output files to `src/testdata/golden/`
5. Wire into `GeneratorsForTarget()` or `GeneratorsForImplLang()` in `generator.go`

## Critical Pitfalls

- **Go cgo**: never `#include` the generated C header in files containing `//export` functions (conflicting prototypes). Use local C type definitions in the cgo preamble instead.
- **`string` and `buffer<T>` are parameter-only** — cannot be return types. Methods returning string/buffer data use FlatBuffer result types.
- **C header `extern "C"` guards** must wrap typedefs through function declarations.
- **FlatBuffer C type names**: dots become underscores (`Common.ErrorCode` → `Common_ErrorCode`).
- **Export macro**: annotate API methods only, never platform service functions.
- **Rust trait methods** take `&self` for UFCS dispatch via ZST `Impl` struct.
- **C++ shim** passes the handle parameter through to the interface (doesn't skip it).
- **Swift FlatBuffer returns** use C struct names (not OpaquePointer).
- **JS/WASM loader** must call `_initialize()` after WASM instantiation (WASI reactor init for static constructors).
- **`flatc` is required** — the FlatBuffers compiler must be available. Use `--skip-flatc` for incomplete output without it.
- **Go generated package name**: API name with underscores removed (e.g., `hello_xplatter` → `helloxplatter`).

## C ABI Boundary Summary

- **Borrowing-only** — the side that allocates deallocates. No ownership transfer across FFI.
- **Transfer modes**: `value` (copied), `ref` (`const T*`, immutable borrow), `ref_mut` (`T*`, mutable borrow).
- **Strings**: `const char*`, UTF-8, null-terminated. Parameter-only. Caller owns, callee borrows.
- **Buffers**: `const T* data, uint32_t data_len` (element count). Parameter-only. `ref`/`ref_mut` controls const.
- **Opaque handles**: typed `void*` with create/destroy lifecycle pairs.
- **Error convention**: fallible methods return error enum code; return value delivered via final out-parameter pointer.
- **No callbacks**: strictly unidirectional (bound language → implementation). Reverse communication via shared ring buffer with platform-native signaling.
- **Symbol visibility**: per-API export macro (`<UPPER_API_NAME>_EXPORT`). Only API methods are annotated. Platform services are link-time provided.

## Key References

- [ARCHITECTURE.md](./ARCHITECTURE.md) — system layers, design rationale, data flow
- [docs/DETAILED_SPEC.md](./docs/DETAILED_SPEC.md) — complete codegen specification (all generators, type mappings, output files, naming rules)
- [docs/api_definition_spec.md](./docs/api_definition_spec.md) — YAML API definition format
- [docs/api_definition_schema.json](./docs/api_definition_schema.json) — JSON Schema for validation
- [docs/example_api_definition.yaml](./docs/example_api_definition.yaml) — working example API definition
