# ![xplatter logo](docs/logo_small.png)<br/> xplatter
xplatter takes an API spec and generates performant bindings for every major user-facing platform's natural app language.

Define a performance-critical API and implement it once in the cross-platform system language of your choice and get a set of ready-to-use, idiomatic API packages for mobile, web, and desktop.

## Why
AI makes exploring solution space much faster and cheaper, but it still takes human effort and inference expenditure to keep it on track. Bindings and glue code are highly rote and require perfect predictability & repeatability. Tradional mechanistic code gen is faster and cheaper for this application. It allows token budgets and human attention to be spent on solving the problem instead of maintaining the scaffold.

## Who
This tool benefits projects that need to deliver performance critical logic behind platform-natural U/X across multiple (or all) user-facing platforms - Android, iOS, mobile web, desktop web, Linux, macOS, Windows.
Examples would be projects implementing on-device inference, media processing, advanced rendering, or any other custom cpu/gpu intensive work. 

## Quick Start

### Prerequisites

- **Go 1.25+**
- **flatc** (FlatBuffers compiler — required for per-language struct codegen)
- **make** (GNU Make)
  - On Windows
    - ```winget install -e --id GnuWin32.Make```
    - OR if Android NDK installed, add ${ANDROID_NDK}/prebuilt/windows-x86_64/bin to PATH
- npm or python3
  - just used for running/testing WASM example
  - examples/app-web/serve.sh will use either to start a CLI one-liner local http server to serve up JS and WASM files
  - no virtual env or language-specific project setup needed

Only the tools for your selected target platforms are required — see [Platform Tooling](#platform-tooling) below.

### Build

```bash
make build
```

This produces `bin/xplatter`.

### Run

```bash
# Generate bindings from an API definition
bin/xplatter generate docs/example_api_definition.yaml -o generated

# Validate an API definition without generating
bin/xplatter validate docs/example_api_definition.yaml

# Scaffold a new project
bin/xplatter init --name my_api --impl-lang cpp
```

### Run the Examples

Working examples with API implementations in C, C++, Rust, and Go with front end consumer apps targeting mobile, desktop, and web live under `examples/`. hello-xplatter defines a simple greeter API. All examples generate bindings, implement them, and run tests.

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

### Run the Tests

```bash
make test          # all Go unit tests
make test-v        # verbose
make validate      # validate the example API definition
```

## Developer Workflow

1. **Define your API** in YAML (see `docs/example_api_definition.yaml`)
2. **Define data types** in FlatBuffers schemas (`.fbs` files)
3. **Generate bindings:** `bin/xplatter generate your_api.yaml`
4. **Implement** the generated abstract interface in your language (C++, Rust, Go, or plain C)
5. **Build** your implementation — the generated C ABI shim handles all FFI compliance

The `examples/hello-xplatter/` directory shows this workflow end-to-end for each supported language.

## What It Generates

- **Pure C API header** — the universal contract, including handle typedefs, FlatBuffer type definitions, platform service declarations, and export-annotated API functions
- **Kotlin + JNI bridge** — idiomatic Kotlin API for Android
- **Swift + C bridge** — idiomatic Swift API for iOS and macOS
- **JavaScript + WASM bindings** — idiomatic JS API for web
- **Implementation interface + C ABI shim + stub implementation** — for C++, Rust, or Go (controlled by `impl_lang` in the API definition)

## Project Structure

```
src/                    Go source for the code gen tool
  gen/                  All code generators (cheader, impl_cpp, impl_rust, impl_go, kotlin, swift, jswasm)
  cmd/                  CLI commands (generate, validate, init, version)
  model/                API model types and type system
  loader/               YAML loading
  resolver/             FlatBuffers schema parsing and type resolution
  validate/             Semantic validation
  testdata/             Test fixtures and golden files
examples/               Working hello-world examples in C, C++, Rust, Go
docs/                   Specifications, schemas, and example definitions
specs/                  FlatBuffers spec files
```

## Documentation

- [Agent Guide](./AGENTS.md) — operational guide for AI coding agents
- [Architecture Overview](./ARCHITECTURE.md) — system layers, C ABI boundary rules, design rationale
- [Code Generation Specification](./docs/DETAILED_SPEC.md) — complete reference for all generators, type mappings, output files, naming conventions, symbol visibility
- [API Definition Specification](./docs/api_definition_spec.md) — full reference for the YAML format
- [Example API Definition](./docs/example_api_definition.yaml) — working example demonstrating the YAML format
- [API Definition JSON Schema](./docs/api_definition_schema.json) — machine-readable schema for validation and editor support

## Platform Tooling

Not all targets can be built on every host OS. Only the tools for your selected `targets` are required.

### Implementation Examples

These build and test the API implementation in each supported language. They run on any host OS with the appropriate compiler.

| Example | Required Tools |
|---------|---------------|
| C (`test-example-c`) | C11 compiler (cc/gcc/clang) |
| C++ (`test-example-cpp`) | C++20 compiler (c++/g++/clang++), C11 compiler |
| Rust (`test-example-rust`) | Rust toolchain (rustc + cargo) |
| Go (`test-example-go`) | Go 1.25+, cgo-compatible C compiler |

### App Examples

These build consumer-facing apps that use the generated bindings. Platform availability depends on the host OS.

| App | Host OS | Required Tools |
|-----|---------|---------------|
| Desktop C++ (`test-app-desktop-cpp`) | macOS, Linux | C++20 compiler, shared library from any impl backend |
| Desktop Swift (`test-app-desktop-swift`) | macOS | Swift compiler (`swiftc`), shared library from any impl backend |
| iOS (`test-app-ios`) | macOS | Xcode (provides `xcrun`, `xcodebuild`, `lipo`, `ar`, `swiftc`) |
| Android (`test-app-android`) | macOS, Linux, Windows | Android SDK, NDK r29+, JDK 17+ |
| Web/WASM (`test-app-web`) | macOS, Linux, Windows | [Emscripten](https://emscripten.org/docs/getting_started/downloads.html) (`emcc`) |

### Host OS / Target Matrix

| Host OS | Buildable Targets |
|---------|-------------------|
| macOS | Desktop (C++ and Swift), iOS, Android, Web |
| Linux | Desktop (C++ only), Android, Web, Linux native |
| Windows | Android, Web, Windows native |

## Make Targets

| Target | Description |
|--------|-------------|
| `build` | Build `bin/xplatter` for the current platform |
| `test` | Run all Go unit tests |
| `test-v` | Run tests with verbose output |
| `test-examples` | Build and run all implementation examples (C, C++, Rust, Go) |
| `test-example-c` | Run the C example only |
| `test-example-cpp` | Run the C++ example only |
| `test-example-rust` | Run the Rust example only |
| `test-example-go` | Run the Go example only |
| `test-apps` | Build and test all app examples |
| `test-app-desktop-cpp` | Build and test the C++ desktop app |
| `test-app-desktop-swift` | Build and test the Swift desktop app (macOS only) |
| `test-app-ios` | Build the iOS app for simulator (macOS only) |
| `test-app-android` | Build the Android app (requires Android SDK + NDK) |
| `test-app-web` | Build the Web/WASM app (requires Emscripten) |
| `validate` | Validate the example API definition |
| `dist` | Build cross-platform SDK archive |
| `fmt` | Format all Go source |
| `vet` | Run `go vet` |
| `clean` | Remove build artifacts |

## Design Overview

**Implementation language agnostic** — any language that can export a Pure C ABI and compile to WASM with C ABI exports is a valid implementation choice. The generated bindings work the same regardless of what's behind the C ABI boundary.

**Borrowing-only FFI boundary** — the side that allocates is the side that deallocates. No ownership transfer, no release callbacks, no ref-counting across the boundary.

**No callbacks** — the C ABI is strictly unidirectional (bound language calls implementation). The implementation communicates back via a shared ring buffer with platform-native signaling.

**Symbol visibility** — generated shared libraries export only API-defined symbols via a per-API export macro. Platform services are link-time provided, not exported.

**FlatBuffers for all data types** — provides the type system, per-language struct codegen, zero-copy serialization, and binary-compatible data across platforms.

**YAML + JSON Schema** — API definitions are human-authored YAML validated by a JSON Schema, with editor autocompletion and inline validation support.

## License

[MIT](./LICENSE.md)
