# xplatgen API Definition Specification

This document specifies the YAML format for defining cross-platform APIs in xplatgen. The API definition file describes the **API surface** — handles, interfaces, and methods — that the code gen tool uses to produce the C ABI header, platform bindings, and optional implementation scaffolding.

All data types (structs, enums, unions, constants, typedefs) are defined in FlatBuffers schemas, not in this file. The API definition references FlatBuffers types by their fully-qualified namespace.

The API definition format is validated by a [JSON Schema](./api_definition_schema.json).

## File Structure

An API definition file has four top-level keys:

```yaml
api:            # Required. API metadata.
flatbuffers:    # Required. FlatBuffers schema file paths.
handles:        # Optional. Opaque handle type definitions.
interfaces:     # Required. Grouped method definitions.
```

No additional top-level keys are permitted.

## `api` — Metadata

```yaml
api:
  name: my_library
  version: 1.0.0
  description: "Optional human-readable description"
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | yes | string | API name. Must be `snake_case` (`^[a-z][a-z0-9_]*$`). Used as a prefix in all generated C ABI function names. |
| `version` | yes | string | Semantic version (`major.minor.patch`). |
| `description` | no | string | Human-readable description of the API. |

## `flatbuffers` — Schema Includes

```yaml
flatbuffers:
  - schemas/common.fbs
  - schemas/geometry.fbs
```

A list of FlatBuffers schema file paths (relative to the API definition file). At least one is required. All paths must end in `.fbs`.

Types defined in these schemas become available for use in method parameters, return types, and error types. They are referenced by their fully-qualified FlatBuffers namespace — for example, a type `ErrorCode` in a schema with `namespace Common;` is referenced as `Common.ErrorCode`.

The code gen tool parses these schemas to resolve type references and invokes the FlatBuffers compiler to generate per-language data structure code.

## `handles` — Opaque Handle Types

```yaml
handles:
  - name: Engine
    description: "Top-level application engine instance"

  - name: Renderer
    description: "Rendering context bound to a platform surface"
```

Handles represent implementation-managed objects that exist behind the C ABI as opaque pointers. At the C level, each handle generates:

```c
typedef struct engine_s* engine_handle;
```

Handles are not data types — they have no fields, no serialization, and no representation outside the FFI boundary. They follow create/destroy lifecycle semantics: the implementation allocates on create and deallocates on destroy.

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | yes | string | Handle type name. Must be `PascalCase` (`^[A-Z][a-zA-Z0-9]*$`). |
| `description` | no | string | Human-readable description of what this handle represents. |

Handles are referenced in method signatures as `handle:Name` (e.g., `handle:Engine`).

## `interfaces` — Method Groups

```yaml
interfaces:
  - name: lifecycle
    description: "Engine creation and teardown"
    methods:
      - name: create_engine
        # ...
```

Interfaces group related methods. They have no runtime representation — they exist purely for organization and to namespace the generated C ABI function names.

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | yes | string | Interface name. Must be `snake_case`. Used in C ABI function naming. |
| `description` | no | string | Human-readable description of this interface group. |
| `methods` | yes | array | List of method definitions. At least one required. |

### Methods

```yaml
methods:
  - name: create_renderer
    description: "Create a rendering context"
    parameters:
      - name: engine
        type: handle:Engine
      - name: config
        type: Rendering.RendererConfig
        transfer: ref
    returns:
      type: handle:Renderer
    error: Common.ErrorCode
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | yes | string | Method name. Must be `snake_case`. |
| `description` | no | string | Human-readable description. Use this to document performance characteristics (e.g., "hot path"), threading expectations, or usage notes. |
| `parameters` | no | array | Ordered list of input parameters. Omit for parameterless methods. |
| `returns` | no | object | Return value definition. Omit for void methods. |
| `error` | no | string | FlatBuffers enum type for error returns (e.g., `Common.ErrorCode`). |

### Parameters

```yaml
parameters:
  - name: events
    type: Input.TouchEventBatch
    transfer: ref
    description: "Batch of touch events to process"
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | yes | string | Parameter name. Must be `snake_case`. |
| `type` | yes | string | Parameter type. See [Type System](#type-system). |
| `transfer` | no | string | Transfer semantics: `value`, `ref`, or `ref_mut`. Defaults to `value`. See [Transfer Semantics](#transfer-semantics). |
| `description` | no | string | Human-readable description of this parameter. |

### Returns

```yaml
returns:
  type: handle:Renderer
  description: "The newly created renderer"
```

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `type` | yes | string | Return type. Restricted subset of the type system — see [Type System](#type-system). |
| `description` | no | string | Human-readable description of the return value. |

## Type System

### Primitive Types

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

Primitives are valid as both parameter and return types. Default transfer is `value` (pass by copy).

### `string`

```yaml
- name: path
  type: string
```

A null-terminated UTF-8 string. At the C ABI level: `const char*`.

Strings follow `ref` semantics — the caller owns the string memory, the callee borrows it for the duration of the call.

**Parameter only.** `string` cannot be used as a return type. Methods that need to return string data should use a FlatBuffer result type containing a string field.

### `buffer<T>`

```yaml
- name: pixel_data
  type: buffer<uint8>
  transfer: ref
```

A contiguous array of primitive values. `T` must be a primitive type. At the C ABI level, a single `buffer<T>` parameter expands to two C parameters:

```c
const T* data, uint32_t data_len
```

`data_len` is the **element count**, not byte count. This eliminates sizing errors when `T` is wider than one byte.

Transfer semantics apply to the pointer: `ref` produces `const T*`, `ref_mut` produces `T*`.

**Parameter only.** `buffer<T>` cannot be used as a return type. Methods that need to return buffer data should use a FlatBuffer result type.

Valid `T` values: `int8`, `int16`, `int32`, `int64`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`.

### `handle:Name`

```yaml
- name: engine
  type: handle:Engine
```

A reference to an opaque handle type defined in the `handles` section. At the C ABI level: the corresponding handle typedef (e.g., `engine_handle`).

Handles are passed by value (the pointer itself is copied). The `transfer` field is not applicable to handles.

Valid as both parameter and return types.

### FlatBuffer Types

```yaml
- name: config
  type: Rendering.RendererConfig
  transfer: ref
```

A reference to a type defined in one of the included FlatBuffers schemas. Referenced by fully-qualified FlatBuffers namespace (e.g., `Common.ErrorCode`, `Geometry.Transform3D`).

Valid as both parameter and return types. FlatBuffer types used as parameters should typically specify `transfer: ref` to avoid copying the entire structure.

## Transfer Semantics

Transfer semantics control how data crosses the C ABI boundary.

| Mode | C ABI | Meaning |
|------|-------|---------|
| `value` | Pass by value | Data is copied across the boundary. Default for primitives and handles. |
| `ref` | `const T*` | Caller owns the data. Callee borrows it immutably for the call duration. |
| `ref_mut` | `T*` | Caller owns the data. Callee borrows it mutably for the call duration. |

**The C ABI is a borrowing boundary.** The side that allocates is the side that deallocates. If the callee needs data to outlive the call, it must copy it explicitly.

Transfer defaults:
- Primitives: `value`
- Handles: always `value` (the pointer itself is copied)
- `string`: always `ref` (implicit, does not need to be specified)
- `buffer<T>`: must specify `ref` or `ref_mut`
- FlatBuffer types: should typically specify `ref` or `ref_mut`

## Error Convention

Methods that can fail declare an error type using the `error` field:

```yaml
- name: create_renderer
  parameters:
    - name: engine
      type: handle:Engine
  returns:
    type: handle:Renderer
  error: Common.ErrorCode
```

The `error` value must be a FlatBuffers enum type. This changes the C ABI signature:

**Fallible method with no return value:**

```c
// error only
int32_t myapi_renderer_begin_frame(renderer_handle renderer);
```

The function returns the error enum value. Success is typically represented by value `0`.

**Fallible method with a return value:**

```c
// error + return value → out-parameter
int32_t myapi_lifecycle_create_engine(engine_handle* out_result);
```

The function returns the error enum value. The return value is delivered through a final out-parameter pointer. On error, the out-parameter is not modified.

**Infallible method with a return value:**

```c
// no error → direct return
uint64_t myapi_scene_get_entity_count(scene_handle scene);
```

The function directly returns the value.

**Infallible method with no return value:**

```c
// void
void myapi_lifecycle_destroy_engine(engine_handle engine);
```

## C ABI Naming

Generated C functions follow the pattern:

```
<api_name>_<interface_name>_<method_name>
```

For an API named `my_engine` with an interface `renderer` and method `begin_frame`:

```c
int32_t my_engine_renderer_begin_frame(renderer_handle renderer);
```

Handle typedefs follow the pattern:

```c
typedef struct <lowercase_name>_s* <lowercase_name>_handle;
```

For a handle named `Renderer`:

```c
typedef struct renderer_s* renderer_handle;
```

## Parameter-Only Types Summary

Two built-in types are restricted to parameters only. This preserves the borrowing boundary by avoiding ambiguous ownership of returned data.

| Type | Parameter | Return | Rationale |
|------|-----------|--------|-----------|
| `string` | yes | **no** | Returned string memory ownership would be ambiguous. Use a FlatBuffer result type. |
| `buffer<T>` | yes | **no** | Same ownership concern. Use a FlatBuffer result type. |

## Complete Example

See [example_api_definition.yaml](../example_api_definition.yaml) for a full working example demonstrating handles, interfaces, methods with parameters and return values, error handling, transfer semantics, and all supported type references.
