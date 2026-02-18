# TODO

* to obtain tasking context read ./ARCHITECTURE.md and ./README.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order. 
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes
  
## the system fbs files should live in a schemas/ directory that's a sibling to the executable. 
* the tool binary should look for a schemas/ directory next to itself via argv[0] (or the golang equivalent)
* api yaml files should be able to reference those fbs files from the installed location. 
* in the distro there should be no top level schemas, that should be in bin/. 
* the project 'build' target should make a symlink in bin/ to schemas/.
* architectural directive: 
  * while the distro package contains a lot of materials for examples and buiding from source, the minimum necessary parts of xplattergy are the executable and the core schemas.
  * the user should not have to copy or directly reference core files to be able to use them (not their problem)

## example app-android project not debuggable from Android Studio like iOS xcode project is. find out why.

## impl-go is implementing twice - once for WASM and once for everyone else. see if there's a workaround.

## it looks like a packaging artifact for iOS build is in repo? investigate.