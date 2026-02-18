XPLATTER ?= ../../../xplatter.sh
IMPL       ?= cpp
IMPL_DIR   := ../impl-$(IMPL)
BUILD_DIR  := build

.PHONY: test build clean

# End-to-end: build the iOS app using packages from the impl
test: build
	@echo "PASS: iOS app builds for simulator"

# --- Ensure impl has built iOS package ---

.PHONY: ensure-package
ensure-package:
	@test -d $(IMPL_DIR)/dist/ios/HelloXplatter.xcframework || $(MAKE) -C $(IMPL_DIR) build

# --- Stage SPM package locally (symlinks to impl's dist/ios/) ---

HelloXplatterLib/Package.swift: ensure-package
	@rm -rf HelloXplatterLib
	@mkdir -p HelloXplatterLib/Sources/HelloXplatterBinding
	cp $(IMPL_DIR)/dist/ios/HelloXplatterLib/Package.swift HelloXplatterLib/Package.swift
	cp $(IMPL_DIR)/dist/ios/HelloXplatterLib/Sources/HelloXplatterBinding/HelloXplatter.swift \
		HelloXplatterLib/Sources/HelloXplatterBinding/HelloXplatter.swift
	@# Rewrite the binaryTarget path to point at the impl's dist
	@sed -i '' 's|path: "../HelloXplatter.xcframework"|path: "../../impl-$(IMPL)/dist/ios/HelloXplatter.xcframework"|' HelloXplatterLib/Package.swift

# --- Build iOS app for simulator ---

build: HelloXplatterLib/Package.swift
	xcodebuild -project HelloApp/HelloApp.xcodeproj \
		-scheme HelloApp \
		-destination 'generic/platform=iOS Simulator' \
		build SYMROOT=$(CURDIR)/$(BUILD_DIR)/app

# --- Clean ---

clean:
	rm -rf $(BUILD_DIR)
	rm -rf HelloXplatterLib/Sources/HelloXplatterBinding/HelloXplatter.swift
