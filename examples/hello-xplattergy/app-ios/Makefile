XPLATTERGY ?= ../../../xplattergy.sh
IMPL       ?= cpp
IMPL_DIR   := ../$(IMPL)
BUILD_DIR  := build

.PHONY: test build-app clean

# End-to-end: build the iOS app using packages from the impl
test: build-app
	@echo "PASS: iOS app builds for simulator"

# --- Ensure impl has built iOS package ---

.PHONY: ensure-package
ensure-package:
	@test -d $(IMPL_DIR)/dist/ios/HelloXplattergy.xcframework || $(MAKE) -C $(IMPL_DIR) package-ios

# --- Stage SPM package locally (symlinks to impl's dist/ios/) ---

HelloXplattergyLib/Package.swift: ensure-package
	@rm -rf HelloXplattergyLib
	@mkdir -p HelloXplattergyLib/Sources/HelloXplattergyBinding
	cp $(IMPL_DIR)/dist/ios/HelloXplattergyLib/Package.swift HelloXplattergyLib/Package.swift
	cp $(IMPL_DIR)/dist/ios/HelloXplattergyLib/Sources/HelloXplattergyBinding/HelloXplattergy.swift \
		HelloXplattergyLib/Sources/HelloXplattergyBinding/HelloXplattergy.swift
	@# Rewrite the binaryTarget path to point at the impl's dist
	@sed -i '' 's|path: "../HelloXplattergy.xcframework"|path: "../../$(IMPL)/dist/ios/HelloXplattergy.xcframework"|' HelloXplattergyLib/Package.swift

# --- Build iOS app for simulator ---

build-app: HelloXplattergyLib/Package.swift
	xcodebuild -project HelloApp/HelloApp.xcodeproj \
		-scheme HelloApp \
		-destination 'generic/platform=iOS Simulator' \
		build SYMROOT=$(CURDIR)/$(BUILD_DIR)/app

# --- Clean ---

clean:
	rm -rf $(BUILD_DIR)
	rm -rf HelloXplattergyLib/Sources/HelloXplattergyBinding/HelloXplattergy.swift
