Examples
========
* Examples currently have an entry point in the implementation language, which is nonsense.

For each example category there needs to be one each of an Android, iOS, WASM, host desktop front end consumer for the back end lib.
This is authored and stands in for the consumer of the API that the xplattergy user produces. The WASM front end will require a local http server to serve up the ui html file. let them use a python 1 liner.
that's not version dependent.

* Each example should have build setups for a cross-platform project
* Android and WASM should build from any host OS
* iOS only from macOS, of course
* the example API will need to build both the backend impl and the system target language library (an APK for android, a Swift package for iOS, a WASM lib for web)

After the hello world examples are working we'll want to do an example that uses the touch events and demonstrates the thinness of the binding layer