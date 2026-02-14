Examples
========
* Examples currently have an entry point in the implementation language, which is nonsense.

For each example category there needs to be one each of an Android, iOS, WASM, host desktop front end consumer for the back end lib.
This is authored and stands in for the consumer of the API that the xplattergy user produces. The WASM front end will require a local http server to serve up the ui html file. let them use a python 1 liner.
that's not version dependent.

* the xplattergy binary referenced from makefiles is hard coded to ../../../bin. we'll want a wrapper script that looks for the bin/xplattergy-OS-ARCH that matches the host and falls back to ../../../bin/
The makefiles need to work for the case of a downloaded/unpacked sdk.
right now only works for xplattergy developer.

* Each example should have build setups for a cross-platform project
* Android and WASM should build from any host OS
* iOS only from macOS, of course
* the example API will need to build both the backend impl and the system target language library (an APK for android, a Swift package for iOS, a WASM lib for web) - right now the source for those bindings is generated but it is not built or tested.

After the hello world examples are working we'll want to do an example that uses the touch events and demonstrates the speed of the binding layer.
also demonstrate using core.fbs and input_events.fbs

add a test-dist makefile target that verifies examples build and work with only the built sdk distro for context.

