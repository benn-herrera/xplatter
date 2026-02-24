# Code State Review — Action Items

Architecture review of `src/` against: duplicate code, death-by-flags, abstraction layering violations, abstraction inversions, and overexposure of data.

Rating scale: 1 (possible concern, needs more information) → 10 (unambiguous problem, must be addressed).

---

## Findings

### #1 — Duplicate Swift closure-wrapping logic | Rating: 8

**File:** `src/gen/swift.go` lines 411–635

`writeSwiftCCall`, `writeSwiftCCallPrimitive`, `writeSwiftCCallVoid`, and `writeSwiftCCallDirect` each independently implement the same withCString/withUnsafeBytes closure-wrapping pattern (~100 lines duplicated 4×). Every one of them:
- Declares `indent` and `closingBraces`
- Defines an inner `closurePrefix()` closure (with minor variations)
- Loops over string and buffer params identically
- Closes with `b.WriteString(closingBraces)`

The only differences are the terminal emit block and the `closurePrefix` content. A bug in the withCString wrapping needs to be fixed in four places.

**Recommendation:** Extract a helper that accepts the body as a function parameter. Something like `writeSwiftCCallWrapped(b, params, hasError, body func(b, indent string))`. The four callers reduce to one helper call with a trailing body.

---

### #2 — Parallel `writeCgoRegularBody` / `writeWasmRegularBody` | Rating: 7

**Files:**
- `src/gen/impl_go.go` lines 380–460
- `src/gen/impl_go_wasmexport.go` lines 262–336

Structurally identical dispatch logic: find the handle parameter, look it up, convert non-handle parameters, call the interface method, marshal the return. The cgo version uses `_handles.Load(...)` with a `uintptr(unsafe.Pointer(...))` cast; the WASM version uses `_wasmHandles.Load(handleParam.Name)` directly. String conversion differs (`GoString` vs. `_cstring`). Both have identical `// TODO: FlatBuffer input` stubs. There is also an unexplained inconsistency: the cgo version uses an explicit `int32Val := int32(param)` suffix for primitives; the WASM version passes them through as-is — intentional or a bug?

**Recommendation:** Discuss before acting. A shared helper would need a strategy object or function parameters for the handle-lookup and string-conversion expressions, which may add more indirection than it saves. The pragmatic option may be to document the intended divergence clearly, add cross-reference comments, and accept the duplication. Only merge when the FlatBuffer input TODO forces parallel maintenance to become undeniable.

---

### #3 — Kotlin instance/factory method error-return dispatch duplicated | Rating: 6

**File:** `src/gen/kotlin.go` lines 134–327

`writeKotlinInstanceMethod` and `writeKotlinFactoryMethod` both implement an identical `hasError × hasReturn` four-case switch, differing only in how `nativeCallArgs` is assembled (instance methods prepend `"handle"`; factory methods don't) and the call-site prefix. The error-handling bodies — LongArray pattern, FlatBuffer exception path, fallible no-return path — are line-for-line duplicated.

**Recommendation:** Extract the four-case emit body into a helper that accepts a pre-built call expression string. Callers build the expression differently; the dispatch body is shared.

---

### #4 — `ComputeWASMExports` / `ComputeWASMExportsCSV` duplicate enumeration | Rating: 4

**File:** `src/gen/makefile.go` lines 11–48

Both functions enumerate interfaces/constructors/destructor/methods identically, differing only in output format (JSON string array vs. comma-separated). The body is ~20 lines duplicated nearly verbatim.

**Recommendation:** Extract `computeWASMExportNames(apiName string, api *model.APIDefinition) []string`. Both callers reduce to a join. Very low risk.

---

### #5 — Swift error enum emits hardcoded placeholder cases | Rating: 6

**File:** `src/gen/swift.go` lines 59–71

`writeSwiftErrorEnum` hardcodes `ok=0, invalidArgument=1, outOfMemory=2, notFound=3, internalError=4` despite `ctx.ResolvedTypes` being available and containing the real enum values. The `resolved` parameter is accepted and silently discarded. Rust and Go generators both iterate `info.EnumValues` correctly.

**This is a correctness bug.** Any API whose error enum has different values or case names gets a Swift binding with broken `rawValue:` initialization in the generated `guard ... throw ErrorEnum(rawValue: code)` calls.

**Recommendation:** Use `resolved[errType].EnumValues` to emit the real cases, matching what the Rust and Go generators already do.

---

### #6 — `jniToCArg` strips `"handle:"` prefix with a hardcoded slice index | Rating: 5

**File:** `src/gen/kotlin.go` line 694

```go
return []string{"(" + HandleTypedefName(p.Type[7:]) + ")" + name}
```

`p.Type[7:]` hardcodes the length of `"handle:"`. This is the only place in the codebase that does not use `model.IsHandle()`. If the handle prefix ever changed, this would silently produce wrong code.

**Recommendation:** Replace with `handleName, _ := model.IsHandle(p.Type)` and use `handleName`. One-line fix.

---

### #7 — `goPackageName` accepts and ignores its parameter | Rating: 3

**File:** `src/gen/impl_go.go` lines 790–794

```go
func goPackageName(apiName string) string {
    return "main"
}
```

`apiName` is accepted and unused. The comment explains why it is always `"main"`. Minor readability note only.

**Recommendation:** Remove the unused parameter, or leave it — the function is small and the comment is adequate. Not an action item.

---

### #8 — Instance-method classification heuristics differ between Swift and Kotlin without explanation | Rating: 4

**Files:**
- `src/gen/swift.go` ~line 266 (`writeSwiftFreeFunctions`)
- `src/gen/kotlin.go` `isAnyInstanceMethod`

Swift classifies a method as belonging to a handle class if it takes a handle as its first parameter OR returns a handle. Kotlin's `isAnyInstanceMethod` only checks the first-parameter handle. The asymmetry is intentional (the two languages map the pattern differently) but is undocumented. An agent adding a new classification rule has no signal about which generators need to stay in sync.

**Recommendation:** Add cross-reference comments in each function noting the deliberate asymmetry and which other generators are affected.

---

### #9 — WASM struct layout alignment mismatch between JS and Go sides | Rating: 5

**Files:**
- `src/gen/jswasm.go` lines 686–721 (`wasmStructLayout`)
- `src/gen/impl_go_wasmexport.go` lines 370–387 (`writeWasmReturnMarshal`)

`wasmStructLayout` correctly applies natural alignment padding for each field. `writeWasmReturnMarshal` tracks offsets sequentially with no padding. Any API returning a FlatBuffer struct with mixed-width fields (e.g., `uint8` followed by `int32`) will produce a layout mismatch: the JS side reads at offset 4; the Go side wrote at offset 1.

**Recommendation:** `writeWasmReturnMarshal` should use `wasmStructLayout` (already in the same package) instead of its own offset tracking.

---

### #10 — JS handle class `dispose()` never calls the WASM destructor | Rating: 5

**File:** `src/gen/jswasm.go` lines 120–159

The generated `dispose()` / `close()` / `[Symbol.dispose]()` methods set `this.#ptr = 0` but never invoke the C ABI destroy function through WASM. The WASM export exists (the auto-destructor is included in the interface wrapper), but handle class disposal silently leaks the implementation-side handle. Kotlin and Swift both correctly wire `close()`/`deinit` to the destroy function.

The current state is the worst of both worlds: it looks safe (`AutoCloseable`, `[Symbol.dispose]`) but silently leaks.

**Recommendation:** Either wire `dispose()` to call `_wasm.exports.<api_destroy_func>(this.#ptr)` — which requires threading the destructor name through to handle class generation — or remove the `AutoCloseable` pattern and document that consumers must call destroy explicitly through the interface wrapper.

---

## Action Item Checklist

Priority ordered. Items requiring discussion before action are marked.

- [x] **#5** Fix Swift error enum codegen to emit real enum cases from `ResolvedTypes` instead of hardcoded placeholders — `swift.go` `writeSwiftErrorEnum` *(correctness bug)*
- [x] **#10** Fix JS handle `dispose()` to call the WASM destructor, or remove the `AutoCloseable` pattern — `jswasm.go` *(correctness bug / misleading API)*
- [x] **#9** Fix WASM return-struct marshal to use `wasmStructLayout` alignment — `impl_go_wasmexport.go` `writeWasmReturnMarshal` *(correctness bug)*
- [ ] **#1** Refactor Swift closure-wrapping duplication into a shared helper — `swift.go` lines 411–635
- [ ] **#3** Extract Kotlin instance/factory method error-return dispatch into a shared helper — `kotlin.go` lines 134–327
- [ ] **#6** Replace `p.Type[7:]` with `model.IsHandle()` in `jniToCArg` — `kotlin.go` line 694 *(one-line fix)*
- [ ] **#4** Extract `computeWASMExportNames` helper from `ComputeWASMExports` / `ComputeWASMExportsCSV` — `makefile.go`
- [ ] **#8** Add cross-reference comments documenting deliberate Swift/Kotlin instance-method classification asymmetry — `swift.go`, `kotlin.go`
- [ ] **#2** *(discuss first)* Decide: merge `writeCgoRegularBody` / `writeWasmRegularBody` into a shared helper, or document divergence and add cross-reference comments — `impl_go.go`, `impl_go_wasmexport.go`
