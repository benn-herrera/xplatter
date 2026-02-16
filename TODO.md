# TODO

## c, cpp, go libraries
* currently these libraries are built only for the host platform.
* the shared libraries/wasm for each supported target need to be built
  * update the Makefile for each impl language to build not only for the host but also for the selected set of wasm, android/arm64, android/amd64, and if host is macOS iOS, iOS simulator
  * update the build process to produce ready to consume target language packages in each target language.
    * after the build there should be artifacts ready for direct consumption by a target language project that neither knows nor cares about the underlying API implementation language. it will only ever see the API surface presented in Swift (swift pagackage), Kotlin (AAR), JavaScript (JS file with auto loading/binding of WASM file + WASM file), C (header + dynamic library)

## app-android, app-ios, app-web
the Makefiles directly reference the cpp implementation files for the bound implementation.
* this is wrong.
* the app example project (Makefile and IDE project) should depend only on the complete, opaquely implemented package presenting the API in their own language 
