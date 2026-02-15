# TODO

## examples make targets
create a Makefile in examples/
* move the example-related relay targets to the new  Makefile from the main Makefile.
* Eliminate example-related relay targets in the main Makefile.

## c, cpp, go libraries
* currently these libraries are built only for the host platform.
* the shared libraries/wasm for each supported target need to be built
  * update the Makefile for each impl language to build not only for the host but also for wasm, android/arm64, android/amd64, and if host is macOS iOS, iOS simulator

## app-android, app-ios, app-web
the Makefiles directly reference the cpp implementation files for the bound implementation.
* this is wrong.
* the app example project (Makefile and IDE project) should depend only on
  * the impl dylib built for the app platform platform
  * the pure C header from that language's impl directory
