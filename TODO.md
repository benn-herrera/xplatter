The hello-xplattergy projects currently have an entry point in the *implementation* language, which is nonsense. This doesn't prove out the system at all. We need examples that verify functionality in the *target* language to which the implementation is bound. The examples need to prove that everything works end to end and show users how to do it.

The current examples skip several major requirements:

* DONE - The API implementation code is compiled to a static library
  * DONE - in all cases (C, C++, Go, Rust) it needs to compile to a shared library (.so, .dylib, .dll)
  * DONE - in all cases the only symbols that should be exported from the shared library are those declared as part of the API. For clang/gcc compilers that means explicitly setting default symbol exporting off and explicitly specifying the API defined symbols for export. For msvc (cl) compilers that will mean explicitly labeling dllexport/dllimport symbols.
  * DONE - this will require changes to the generated C code to label the symbols for export appropriately. even though we're on macOS right now don't forget about windows. we won't be able to test that behavior, but set up for it and we'll continue work on a windows machine when ready.
* The target language bindings that hide the origin of the API implementation are not getting built
  * for iOS build a target platform swift package that consumes the APIimplementation and presents the idiomatic Swift interface.
    * must have configs that support both iOS device and simulator
  * DONE - for macOS also build a swift package that can be consumed by a macOS desktop swift app
  * for Android - build an AAR that consumes the API implementation and presents an idiomatic Kotlin interface
  * for linux and windows building the API shared library is sufficient
  * as we are currently working on macOS we can build for both mobile platforms, wasm, and macos. windows development will happen in a session on another machine.
* The target language bindings are not getting used - a minimal app is necessary
  * For mobile and web
    * project setup must depend on the built bound API package
    * presents a ui with a text input field, a 'greet' button, and text display field.
    * the input text field will have a default value of 'xplattergy'
    * the text display field will start empty
    * hitting the greet button will send the text input field value to the API, retrieve the boxed string from the return value and set the contents of the text display field to that string.
  * For desktop an interactive terminal app will be used
    * DONE - runs without command line arguments
    * DONE - runs an interaction loop
    * DONE - prompts for a name (exit or quit will end the session)
    * DONE - when a non-empty name is entered it calls the bound API function, retrieves the boxed string and prints the value to stdout.
    * DONE (macOS) - linux, windows, mac - implement in C++.
      * DONE - on mac also include an implementation in swift that targets the host (as opposed to iOS)
