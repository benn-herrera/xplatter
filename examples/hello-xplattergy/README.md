# xplattergy Examples

Hello-world examples demonstrating the full xplattergy workflow:
**YAML API definition -> `xplattergy generate` -> hand-written implementation -> build -> run**

Each example implements the same simple `Greeter` API in a different implementation language.

## Shared Schemas

`shared_schemas/hello_types.fbs` contains the canonical FlatBuffer schema defining:
- `Hello.ErrorCode` enum (Ok, InvalidArgument, InternalError)
- `Hello.Greeting` table (with a `message` field)

The xplattergy codegen tool parses these `.fbs` schemas and generates C type definitions
(enums, structs) directly into the C header, plus equivalent Rust and Go type files.
No separate flatc step is required.

## Examples

| Directory | impl_lang | Description |
|-----------|-----------|-------------|
| `impl-c/`      | c         | Direct C ABI implementation |
| `impl-cpp/`    | cpp       | C++ interface/shim pattern |
| `impl-rust/`   | rust      | Rust trait/FFI pattern |
| `impl-go/`     | go        | Go interface/cgo pattern |

## Prerequisites

- xplattergy binary built (run `make` from the project root)
- C compiler (cc)
- C++ compiler with C++20 support (c++)
- Rust toolchain (cargo)
- Go toolchain (go)

## Running

From the project root:

```bash
# Run individual examples
make test-example-c
make test-example-cpp
make test-example-rust
make test-example-go

# Run all examples
make test-examples
```

Or from each example directory:

```bash
# C
cd impl-c && make run

# C++
cd impl-cpp && make run

# Rust
cd impl-rust && cargo test

# Go
cd impl-go && make run
```

## What Gets Generated vs. Hand-Written

Each example runs `xplattergy generate` to produce scaffolding in a `generated/`
subdirectory, then compiles hand-written implementation files alongside the generated code.

| impl_lang | Generated (in `generated/`)                                          | Hand-written                                           |
|-----------|----------------------------------------------------------------------|--------------------------------------------------------|
| c         | `.h` (header with types)                                             | `greeter_impl.c`, `platform_services.c`, `main.c`     |
| cpp       | `.h`, `_interface.h`, `_shim.cpp`, `_impl.h/cpp` (stubs)            | `_impl.h/cpp` (filled-in), `platform_services.c`, `main.cpp` |
| rust      | `.h`, `_trait.rs`, `_ffi.rs`, `_impl.rs` (stubs), `_types.rs`       | `_impl.rs` (filled-in), `platform_services.rs`, `tests/integration.rs` |
| go        | `.h`, `_interface.go`, `_cgo.go`, `_impl.go` (stubs), `_types.go`   | `_impl.go` (filled-in), `platform_services.go`, `main.go` |
