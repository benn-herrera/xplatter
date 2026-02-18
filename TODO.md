# TODO

* to obtain tasking context read ./ARCHITECTURE.md and ./README.md
* items on the todo list are addressed one encapsulated (## headed) issue at a time in top down order. 
* when completed they are marked done (## DONE - [description]) and moved to the end of the file for archival purposes

## example app-[platform] projects should not be packaging the go api for their own consumption
  * the impl-go project should be handling packaging for all selected/supported consumer platform/language combos.
  * each app project should only consume prebuilt packages targeting their platform/language combo, not manipulating the internals of an impl package or doing its job for it.
  * the convenience make target in an app project for building the impl-[language] it is consuming (as defined by IMPL variable) should use $(MAKE) -C ../impl-$(IMPL) build only.
  
## example app-android project not debuggable from Android Studio like iOS xcode project is. find out why.

## impl-go is implementing twice - once for WASM and once for everyone else. see if there's a workaround.

## the system fbs files should live in a schemas/ directory that's a sibling to the executable. api yaml files should be able to reference those fbs files from the installed location. in the distro there should be no top level schemas, that should be in bin/. the project 'build' target should make a symlink in bin/ to schemas/

## it looks like a packaging artifact for iOS build is in repo? investigate.