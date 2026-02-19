# TODO

* to obtain tasking context read ./AGENTS.md and ./ARCHITECTURE.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order.
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes

## impl-go is implementing twice - once for WASM and once for everyone else. see if there's a workaround.

## example app-android project not debuggable from Android Studio like iOS xcode project is. find out why.

## DONE - impl projects are producing package elements, but not complete packages.
* Each impl language (C, C++, Rust, Go) now generates a complete Makefile (scaffold) with:
  - Local build (`run`, `shared-lib`)
  - iOS cross-compilation (xcrun per arch -> lipo -> xcframework -> SPM Package.swift)
  - Android cross-compilation (NDK per ABI -> .so -> Gradle module with build.gradle.kts + AndroidManifest.xml)
  - Web/WASM packaging (.wasm + JS binding + package.json)
  - Desktop packaging (header + shared lib + Swift binding)
* Platform services stubs generated per target (desktop, ios, android, web) with platform-appropriate logging.
* WASM exports computed from API definition (replaces hardcoded lists).
* Shared Makefile helpers in makefile.go: header, target config, binding vars, packaging rules for iOS/Android/Web/Desktop.
* ProjectFile flag routes Makefile and platform_services to project root when using `-o generated`.
* Examples updated: hand-written Makefiles and cross-compile-cc.mk deleted; examples/Makefile bootstraps codegen before builds.
* Source packages (Cargo.toml, go.mod, CMakeLists.txt) and scaffold stubs preserved across regeneration.
