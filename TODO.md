# TODO

## example app-[platform] projects should not be packaging the go api for their own consumption
  * each impl language project should be building packages for all available/selected target platform/language combos
  * each app project should only consume prebuilt packages targeting their platform/language combo.
  * the convenience make target for building the impl-[language] it is consuming (as defined by IMPL variable) should use $(MAKE) -C ../impl-$(IMPL) build only.
  * if any of the app-[platform] projects are building 