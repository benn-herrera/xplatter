# TODO

* to obtain tasking context read ./AGENTS.md and ./ARCHITECTURE.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order. The task is to be accomplished starting in planning mode and then moving on to execution.
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes

## impl-c and impl-cpp package-web fails on Windows
* directly invoking emcc in the Makefile does not extend well to Windows.
* impl-cpp already uses cmake for at least on compilation. 
  * WASM compilation should be handled in the CMakeLists.txt file.
    * emscripten provides a CMake toolchain file
    * on windows the EMSDK_PATH envar can be used to find the toolchain file
  * the same cmake approach should be used for impl-c.

## app-desktop-cpp not linking
* there are mismatches between the library produced and the one expected in the Makefile
  * the produced library is dynamic and a static library is expected
  * the produced library uses a 'lib' name prefix convention and the expected one does not

## _IGNORE THIS LINE AND EVERYTHING BELOW IT IN THIS FILE - STAGING AREA FOR FUTURE WORK_

## impl-go - cgo not working properly
  * fails with error ```-buildmode=c-shared requires external (cgo) linking, but cgo is not enabled``` 

## in GC'd languages (Swift, Kotlin, JavaScript)   
  * the generated bindings should map constructors (create functions) to setup functions that replace the string 'create' in the name with 'setup' 
  * a destructor should map a bound to a function called 'teardown'. 
    * teardown should clear the cached handle after invoking the destructor function.
    * post-teardown state should be equivalent to pre-setup state (i.e. safe to call setup again after)
  * calling any combination of setup functions twice without having called teardown should raise an exception
  * calling 'teardown' multiple times should be safe
  * method binding wrappers should verify a non-null (zero) handle and raise an exception if verification fails.

## users are going to have their own preferences for build systems in their projects. the extent to which we generate Makefiles to help them may be too opinionated. investigate just how much of an obstruction the current behavior is toward the user's preferences.

## ensure method and constructor names are not allowed to collide. name collisions must be fatal errors.

## error messages for violations of api definition constraints in the yaml file that can't be caught by the schema validator must produce error message that include file path and line number.

## architecture question: do we want to allow for interface inheritance?
* complexity cost vs. flexibility benefit?

## returning strings only as flatbuffer BoxedString that requires implementer to hold an allocation indefinitely is unacceptably burdensome on xplatter users (API implementers) - think about this problem more carefully.
