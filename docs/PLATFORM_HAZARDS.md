# Platform Hazards Reference

This document catalogs platform-specific hazards encountered in xplatter's example `impl` and `app` Makefiles, along with the workarounds that resolve each one. It is organized by platform and topic. This dense packet of lore might save someone a bit of frustration and misery.

---

## Windows / MSVC Bootstrapping

### cl.exe not on PATH outside Developer Command Prompt

GNU Make under MSYS2/bash on Windows does not inherit the Developer Command Prompt environment. `cl.exe` is not on PATH by default. The official mechanism is `vcvarsall.bat x64`, but calling it from bash cannot propagate the environment to the parent shell — bat files run in a subprocess and the parent shell sees none of their exports.

**Fix (impl-c, impl-cpp, app-desktop-cpp):** Write a temp `.cmd` file via `printf` that calls `vcvarsall.bat` then re-invokes `$(MAKE) $(MAKECMDGOALS)`. Run the `.cmd` with `.\\_msvc_setup.cmd`. The inner make finds `cl.exe` on PATH and skips the bootstrap block. The outer make's real targets are guarded with `ifdef _DO_BOOTSTRAP / @: / else / real recipe / endif` so they no-op in the outer context.

### backslash paths on windows can cause problems with gnu make

`vswhere.exe -property installationPath` returns Windows backslash paths. GNU Make and bash use forward slashes. A path like `C:\Program Files\Microsoft Visual Studio` causes Make to interpret `\n` as a line continuation and other sequences as escape codes.

**Fix:** Pipe vswhere output through `| tr -d '\r' | sed 's:\\\\:/:g'` before assigning to a Make variable.

### `ProgramFiles(x86)` env var has parentheses — invalid in Make

`$(ProgramFiles(x86))` is illegal Make syntax because `(` is not a valid variable name character. The env var cannot be accessed directly in a Make expression.

**Fix:** `$(shell cmd //C "echo %ProgramFiles(x86)%" 2>/dev/null | tr -d '\r')` — delegate expansion to `cmd.exe` then strip the CRLF.

### `cmd //C` not `/C` under MSYS2

MSYS2 translates arguments that look like POSIX paths. `/C` is treated as a path and converted to something like `C:\Program Files\Git\C`.

**Fix:** Use `//C` — MSYS2 recognizes the double-slash as an escaped single slash intended for Windows and passes it as `/C` to `cmd.exe`.

### cmd.exe output has CRLF line endings

Any output captured from `cmd //C "echo ..."` includes a trailing `\r\n`. The `\r` survives into Make variable values and causes spurious mismatches.

**Fix:** Pipe all cmd.exe output through `| tr -d '\r'` before assignment.

### Temp `.cmd` file must have CRLF line endings

`cmd.exe` expects CRLF line endings in `.cmd`/`.bat` files. A file written with only `\n` may execute unreliably.

**Fix:** Use `printf '@call "%s" x64 >nul 2>&1\r\n@"$(MAKE)" $(MAKECMDGOALS)\r\n'` to write the file. The literal `\r\n` in the printf format string produces the correct CRLF in the output file.

### Temp `.cmd` invocation requires backslash

Running `./foo.cmd` from MSYS2 bash hands the path to the OS with a forward slash. On Windows, a relative path like `./foo.cmd` may not be recognized as a batch file.

**Fix:** In the Make recipe, invoke as `@.\\$(_TMP_CMD)` — the double backslash in a Make recipe string produces one literal backslash in the shell command.

### Catch-all `%: _msvc_bootstrap ;` triggers on Makefile itself

The `%: _msvc_bootstrap ;` pattern adds `_msvc_bootstrap` as a prerequisite for every goal. Without special handling, Make will attempt to rebuild the Makefile itself through this pattern, causing infinite recursion.

**Fix:** Add `Makefile: ;` — an explicit empty rule for `Makefile` that short-circuits the catch-all.

### Timestamp-unique temp `.cmd` filename

If multiple make invocations run concurrently, they would clobber the same temp `.cmd` file and corrupt each other's bootstraps.

**Fix:** `_TMP_CMD := _msvc_setup_$(shell date '+%s').cmd` — epoch seconds appended to the filename to make it unique per invocation.

### GNU Make built-in `CC := cc`

GNU Make has a built-in default rule that sets `CC=cc`. Using `CC ?= cl.exe` is silently overridden by this built-in on Windows where `cc` does not exist, resulting in a failed compilation with no obvious error.

**Fix:** Use `CC := cl.exe` (`:=` immediate assignment) on the Windows branch to unconditionally override the built-in default.

### `BUILD_MACRO` must be in base CFLAGS/CXXFLAGS on MSVC

The API's export macro expands to `__declspec(dllexport)` when `BUILD_MACRO` is defined, and `__declspec(dllimport)` otherwise. If a `.c`/`.cpp` file that *defines* the exported functions is compiled without `BUILD_MACRO` set, MSVC sees `dllimport` on its own definitions and issues error C2491 ("definition of dllimport function not allowed").

**Fix:** Include `/D$(BUILD_MACRO)` in the base `CFLAGS`/`CXXFLAGS` on Windows, not only in `LIB_VISIBILITY_FLAGS`. On GCC/Clang, `-fvisibility=hidden` handles symbol visibility differently (it is an attribute, not a declaration class), so this separation does not apply there.

### MSVC flags differ entirely from GCC/Clang

MSVC uses `/W4 /std:c17 /I /Fe: /LD /D /EHsc /std:c++20`. GCC/Clang flags like `-fvisibility=hidden`, `-fPIC`, `-shared`, and `-Wall` are not recognized by `cl.exe` and will cause build failures if passed to it.

**Fix:** Maintain two separate flag sets: one for the Windows MSVC branch, one for GCC/Clang. Cross-compilation flags (NDK, Emscripten) always use the GCC/Clang set regardless of host.

---

## Windows / MSYS2 Argument Conversion

### MSYS2 converts `/FLAG` arguments to Windows paths

MSYS2 automatically converts arguments that look like POSIX absolute paths (starting with `/`) before passing them to processes. `/EXPORTS`, `/DEF:foo.def`, `/OUT:foo.lib`, and similar MSVC tool flags get mangled to paths like `C:\Program Files\Git\EXPORTS`.

**Fix:** Set `MSYS2_ARG_CONV_EXCL='*'` in the environment before calling any MSVC tool (`dumpbin`, `lib.exe`). This tells MSYS2 to pass all arguments through unmodified.

---

## Windows / DLL and Import Libraries

### Rust `cdylib` on Windows drops the `lib` prefix

Rust builds a DLL named `foo.dll` for a crate named `foo`, not `libfoo.dll`. All other platforms produce `libfoo.dylib` or `libfoo.so`.

**Fix:** On Windows, set `DESKTOP_LIB_NAME := $(API_NAME)` (without `lib` prefix). The Makefile already conditionally sets this per platform.

### Rust `cdylib` produces `.dll.lib` not `.lib`

Cargo produces `target/release/foo.dll.lib` as the MSVC-compatible import library — a double extension — not `foo.lib`.

**Fix:** Copy from `target/release/$(DESKTOP_LIB_NAME).$(DYLIB_EXT).lib` explicitly (note the `.dll.lib` suffix in the source path).

### Go `buildmode=c-shared` doesn't produce an MSVC import library

`go build -buildmode=c-shared` on Windows produces only the `.dll`. Unlike MSVC or Rust/cargo, it does not emit a `.lib` import library. Without a `.lib`, MSVC-based consumers cannot link at compile time.

**Fix (impl-go):** Parse the generated C header to extract exported symbol names (awk over `extern` declarations), write a `.def` file, then run `zig dlltool -d foo.def -D foo.dll -l foo.lib`. Alternatives include `dumpbin /EXPORTS` + `lib /DEF:` (requires Windows SDK). The header-parsing approach avoids needing MSVC SDK tools.

The awk pattern used: match lines starting with `extern ` but not `extern "C"`, strip the opening `(`, and print the third field (the function name).

---

## Windows / Paths in Make

### Backslashes in environment variables cause problems in gnu make

Windows environment variables set by Windows tools (`ANDROID_NDK`, `ANDROID_SDK`, `LOCALAPPDATA`, `EMSDK`, vcvarsall output) use backslash separators. GNU Make and bash use forward slashes, making these values unusable as path components without normalization.

**Fix:** Normalize via `$(shell echo "$(VAR)" | sed 's:\\\\:/:g')` at point of use. This is idempotent on Linux/macOS where the sed pattern matches nothing.

---

## Windows / Go + CGO

### Go CGO on Windows does not support MSVC

Go's CGO pipeline calls its C compiler for the cgo preamble. It requires a GCC-compatible compiler and does not support `cl.exe`.

**Fix:** Use `zig cc` as the C compiler for CGO: set `CGO_CC := zig cc` on Windows, then invoke `CC="$(CGO_CC)" go build`. Zig ships as a single portable binary — simpler to install than MinGW while providing a fully compliant GCC-compatible frontend.

---

## Android / NDK

### NDK cross-compiler on Windows is a `.cmd` wrapper, not an ELF binary

On Windows, the NDK's clang binaries are wrapped as `.cmd` batch files (e.g., `aarch64-linux-android28-clang.cmd`). Using the bare name without the `.cmd` extension fails to find the file.

**Fix (impl-rust):** Set `NDK_CMD := .cmd` on Windows, then append `$(NDK_CMD)` to the linker path: `CARGO_TARGET_..._LINKER="$(NDK_BIN)/$(3)-clang$(NDK_CMD)"`.

### Android NDK default path varies by OS

The NDK is installed in different default locations: `~/Library/Android/sdk` on macOS, `~/Android/Sdk` on Linux, and `$LOCALAPPDATA/Android/Sdk` on Windows.

**Fix:** Use an `ifeq`/`else ifeq` chain on `NDK_HOST_OS`. For Windows, `$(LOCALAPPDATA)` contains backslashes which must be normalized via `sed`.

### Android NDK version auto-selection

Multiple NDK versions may be installed. A hardcoded version string breaks when the user has a different version installed.

**Fix:** `ls -d $(ASDK)/ndk/* | sort -V | tail -1` selects the highest-versioned NDK directory. `-V` is GNU sort's version-aware sort; it is not available on stock macOS `sort`, but the NDK host toolchain environment includes GNU sort.

### Android C++ impl requires `-static-libstdc++`

When linking a `.so` for Android, the NDK's `libc++_shared.so` may not be available on the device at the expected version. Dynamically linking `libc++` risks runtime failures on older devices or stripped system images.

**Fix (impl-cpp):** Pass `-static-libstdc++` at link time to bundle `libc++` into the `.so`. C impls are unaffected since they have no C++ runtime dependency.

### CGO JNI build: `.c` files must be in the package root

Go's CGO compiles all `.c` files in the same directory as the `import "C"` source. The generated JNI bridge lives in `generated/`, which is not the package root.

**Fix (impl-go):** Temporarily copy the generated JNI `.c` file into the package root before `go build -buildmode=c-shared`, then remove it after. ABI targets using this pattern must not run in parallel — they write and remove the same filename.

---

## macOS / Dylib Install Name

### macOS dylibs must have `@rpath`-relative install names

The dynamic linker on macOS uses the install name embedded in the `.dylib` to find it at runtime. A dylib built without `@rpath/` in its install name will only load from its original build path.

**Fix:** At build time: `-Wl,-install_name,@rpath/libfoo.dylib` (C/C++), or `install_name_tool -id @rpath/libfoo.dylib` post-build (Rust, where `cargo` does not set this). At link time for the consumer app: `-Wl,-rpath,@executable_path/path/to/libs` (or `-Xlinker -rpath -Xlinker ...` for swiftc).

---

## Linux / rpath

### Linux `$ORIGIN` rpath must be shell-escaped in Make

Linux's dynamic linker supports `$ORIGIN` as a placeholder for the directory containing the executable. In Make recipes, `$` must be escaped as `$$`. Inside a shell string, the `$` may also need single-quoting to prevent shell expansion.

**Fix:** `-Wl,-rpath,'$$ORIGIN/relative/path'` — the `$$` produces a literal `$` in the shell command, and single quotes prevent further shell expansion of `$ORIGIN`.

---

## Emscripten

### Emscripten root location differs between install methods

Package manager installs (Homebrew, apt): `em-config EMSCRIPTEN_ROOT` returns the root. EMSDK installer installs: `$EMSDK/upstream/emscripten`. On Windows, the EMSDK path may contain backslashes.

**Fix:** Check `EMSDK_PATH` first, fall back to `em-config`, then apply `sed 's:\\\\:/:g'` unconditionally (a no-op on Unix). The CMake toolchain file lives at `$(EMSCRIPTEN_ROOT)/cmake/Modules/Platform/Emscripten.cmake`.

---

## Gradle / Android App

### `gradlew` shebang fails under Git Bash on Windows

Gradle's `gradlew` script uses `#!/usr/bin/env sh`, which Git Bash does not reliably handle when the script is invoked as `./gradlew`.

**Fix:** Invoke explicitly as `bash ./gradlew :app:assembleDebug` rather than `./gradlew`.

### Android SDK path written to `local.properties`

Gradle requires `local.properties` with `sdk.dir=...` pointing to the Android SDK. The path must use forward slashes on all platforms, including Windows.

**Fix:** Generate `local.properties` from Make using the normalized SDK path (already forward-slash-converted via `sed` at the point of variable assignment).
