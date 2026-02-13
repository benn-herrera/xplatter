# xplatgen

xplatgen *(splat-jen)* is a code generation tool that produces cross-platform API bindings from a single API definition. Define your API once in YAML, define your data types in FlatBuffers, and xplatgen generates the C ABI contract and idiomatic bindings for every target platform.

## Target Platforms

Android, iOS, Web, Windows, macOS, and Linux.

## What It Generates

- **Pure C API header** — the universal contract your implementation fulfills
- **Kotlin + JNI bridge** — idiomatic Kotlin API for Android
- **Swift + C bridge** — idiomatic Swift API for iOS and macOS
- **JavaScript + WASM bindings** — idiomatic JS API for web and desktop
- **Implementation scaffolding** (optional) — starter code for C++, Rust, or Go

## Implementation Language Agnostic

xplatgen does not dictate what language you write your library in. Any language that can export a Pure C ABI and compile to WASM with C ABI exports is a valid choice. The generated bindings work the same regardless of whether the implementation behind them is C++, Rust, Go, or anything else.

## Performant

The binding layer is designed to stay out of the way of performance-critical code paths. FlatBuffers provides zero-copy data access across the JNI boundary — no serialization or deserialization on either side. Parameters use explicit transfer semantics (`value`, `ref`, `ref_mut`) so authors control exactly when data is copied and when it's borrowed. Raw `buffer<T>` parameters pass pointers directly with no wrapper overhead, supporting use cases like touch input at 120+ FPS and real-time audio/video streaming. The borrowing-only boundary eliminates ref-counting and release callbacks, keeping the FFI layer thin and predictable.

## Cross-Platform Data Compatibility

Because all data types are defined in FlatBuffers schemas shared across every target, serialized data is binary-compatible across platforms out of the box. A save file written on iOS can be loaded on Android or Windows. A message sent from a web app can be read by a peer running on Linux. Schema evolution rules (adding fields, deprecating fields) provide forward and backward compatibility as your data definitions change over time.

## Built On

- **FlatBuffers** — all data types are defined in `.fbs` schemas. FlatBuffers provides the type system, per-language struct codegen, and zero-copy serialization. xplatgen does not reinvent any of this.
- **YAML + JSON Schema** — API definitions are authored in YAML and validated by a JSON Schema, giving you editor autocompletion, inline validation, and a format that's friendly to both humans and AI coding agents.

## Core Platform Services

Every generated API includes platform service functions that the implementation can call without platform-specific code:

- **Logging** — zero-latency, crash-safe output routed to the native logging system (logcat, os_log, console)
- **Resource access** — uniform read access to bundled application resources across all platforms

## Documentation

- [Architecture Overview](./ARCHITECTURE.md) — design rationale, system layers, C ABI boundary rules, and technical decisions
- [API Definition Specification](./docs/api_definition_spec.md) — full reference for the YAML format
- [API Definition JSON Schema](./docs/api_definition_schema.json) — machine-readable schema for validation and editor support
