# TODO

* to obtain tasking context read ./ARCHITECTURE.md and ./README.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order. 
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes

## in the example apps the consumed impl package is not obvious at run time.
* I've added the string field apiImpl to the hello_types.fbs definition of the Greeting structure.
* Update each of the c, cpp, rust, go implementations to fill in that field with impl-[language]
* Update each of the app-[platform] examples to invoke say_hello() on startup with an empty string for the name parameter. it should display "Backing implementation: [apiImpl]" in a static text field above the existing controls (GUI) or just print that string to stdout before starting the interaction loop. During the interaction loop it will, as before, just print the greeting. Update the tests to account for the change. 

## example app-android project not debuggable from Android Studio like iOS xcode project is. find out why.

## impl-go is implementing twice - once for WASM and once for everyone else. see if there's a workaround.

## it looks like a packaging artifact for iOS build is in repo? investigate.

## idiomatic bindings for kotlin, javascript, swift should return the values fetched by API getters and raise exceptions. the pattern of return by writable reference is possible, but not smooth. The only reason we do it for the C API is to allow for the error code to be returned.
Architecture question: preferable or more efficient to return the primary value and have a writable error code pointer parameter (reversing the return behavior) instead? something else?

## DONE - the system fbs files should live in a schemas/ directory that's a sibling to the executable.
* ParseFBSFiles now accepts searchDirs []string — tries YAML dir first, then exe-sibling dir
* ResolveFBSPath searches directories in order for relative .fbs paths
* generate and validate commands construct search path: [yamlBaseDir, exeDir]
* `make build` creates bin/schemas symlink to ../schemas/
* `make dist` copies schemas into bin/schemas/ (no top-level schemas in distro)

