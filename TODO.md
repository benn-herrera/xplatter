# TODO

## example app-[platform] projects should not be packaging the go api for their own consumption
  * each impl language project should be building packages for all available/selected target platform/language combos
  * each app project should only consume prebuilt packages targeting their platform/language combo.
  * the convenience make target for building the impl-[language] it is consuming (as defined by IMPL variable) should use $(MAKE) -C ../impl-$(IMPL) build only.
  * if any of the app-[platform] projects are building 
  
## example app-android project not debuggable from Android Studio like iOS xcode project is. find out why.

## impl-go is implementing twice - once for WASM and once for everyone else. see if there's a workaround.

## the system fbs files should live in a schemas/ directory that's a sibling to the executable. api yaml files should be able to reference those fbs files from the installed location. in the distro there should be no top level schemas, that should be in bin/. the project 'build' target should make a symlink in bin/ to schemas/

## it looks like a packaging artifact for iOS build is in repo? investigate.