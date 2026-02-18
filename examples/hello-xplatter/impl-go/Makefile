XPLATTER ?= ../../../xplatter.sh
BUILD_DIR  := build
DIST_DIR   := dist
STAMP      := $(BUILD_DIR)/.generated

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif
SHARED_LIB := $(BUILD_DIR)/libhello_xplatter.$(DYLIB_EXT)

API_NAME := hello_xplatter
LIB_NAME := libhello_xplatter

# ── Target filtering ──────────────────────────────────────────────────────────

TARGETS ?= ios android web desktop

target_enabled = $(filter $(1),$(TARGETS))

# ──────────────────────────────────────────────────────────────────────────────

.PHONY: run shared-lib clean

run: $(STAMP)
	go build -o $(BUILD_DIR)/hello_xplatter .
	./$(BUILD_DIR)/hello_xplatter

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(STAMP)
	go build -buildmode=c-shared -ldflags='-extldflags "-Wl,-install_name,@rpath/libhello_xplatter.$(DYLIB_EXT)"' -o $(SHARED_LIB) .

# Generate and copy C header locally (cgo needs headers in the package directory)
$(STAMP): ../shared_schemas/hello_xplatter.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTER) generate --impl-lang go --clean -o generated $<
	cp generated/hello_xplatter.h .
	@touch $@

# ══════════════════════════════════════════════════════════════════════════════
# Desktop: C header + Swift binding + shared library
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,desktop),)

$(DIST_DIR)/desktop/include/$(API_NAME).h: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/$(API_NAME).h $@

$(DIST_DIR)/desktop/include/HelloXplatter.swift: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/HelloXplatter.swift $@

$(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT): $(SHARED_LIB)
	@mkdir -p $(dir $@)
	cp $(SHARED_LIB) $@

.PHONY: package-desktop
package-desktop: $(STAMP) $(DIST_DIR)/desktop/include/$(API_NAME).h $(DIST_DIR)/desktop/include/HelloXplatter.swift $(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT)
	@echo "Packaged Desktop: $(DIST_DIR)/desktop/"

endif

# ══════════════════════════════════════════════════════════════════════════════
# Web: Go WASM (wasip1) + JS binding
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,web),)

$(DIST_DIR)/web/hello_xplatter.wasm: $(STAMP)
	@mkdir -p $(dir $@)
	GOOS=wasip1 GOARCH=wasm go build -o $@ .

$(DIST_DIR)/web/hello_xplatter.js: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/hello_xplatter.js $@

.PHONY: package-web
package-web: $(DIST_DIR)/web/hello_xplatter.wasm $(DIST_DIR)/web/hello_xplatter.js
	@echo "Packaged Web: $(DIST_DIR)/web/"

endif

# ══════════════════════════════════════════════════════════════════════════════
# iOS: Go c-archive (CGO) per arch → lipo → xcframework + SPM package
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,ios),)

IOS_MIN := 15.0
IOS_SDK := $(shell xcrun --sdk iphoneos --show-sdk-path 2>/dev/null)
SIM_SDK := $(shell xcrun --sdk iphonesimulator --show-sdk-path 2>/dev/null)
IOS_CC  := $(shell xcrun --sdk iphoneos --find clang 2>/dev/null)
SIM_CC  := $(shell xcrun --sdk iphonesimulator --find clang 2>/dev/null)

GEN_SWIFT_BINDING := generated/HelloXplatter.swift

# $(1) = arch dir  (e.g. ios-arm64)
# $(2) = GOARCH    (e.g. arm64)
# $(3) = clang -arch flag  (e.g. arm64)
# $(4) = clang -target triple  (e.g. arm64-apple-ios15.0)
# $(5) = CC (IOS_CC or SIM_CC)
# $(6) = sysroot (IOS_SDK or SIM_SDK)
define BUILD_IOS_ARCH_GO

$(BUILD_DIR)/ios/$(1)/$(LIB_NAME).a: $(STAMP)
	@mkdir -p $$(dir $$@)
	CGO_ENABLED=1 GOOS=ios GOARCH=$(2) \
	  CC="$(5)" \
	  CGO_CFLAGS="-arch $(3) -target $(4) -isysroot $(6)" \
	  CGO_LDFLAGS="-arch $(3) -target $(4) -isysroot $(6)" \
	  go build -buildmode=c-archive -o $$@ .

endef

$(eval $(call BUILD_IOS_ARCH_GO,ios-arm64,arm64,arm64,arm64-apple-ios$(IOS_MIN),$(IOS_CC),$(IOS_SDK)))
$(eval $(call BUILD_IOS_ARCH_GO,ios-sim-arm64,arm64,arm64,arm64-apple-ios$(IOS_MIN)-simulator,$(SIM_CC),$(SIM_SDK)))
$(eval $(call BUILD_IOS_ARCH_GO,ios-sim-x86_64,amd64,x86_64,x86_64-apple-ios$(IOS_MIN)-simulator,$(SIM_CC),$(SIM_SDK)))

$(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a: $(BUILD_DIR)/ios/ios-sim-arm64/$(LIB_NAME).a $(BUILD_DIR)/ios/ios-sim-x86_64/$(LIB_NAME).a
	@mkdir -p $(dir $@)
	lipo -create $^ -output $@

$(DIST_DIR)/ios/headers/module.modulemap: $(STAMP)
	@mkdir -p $(DIST_DIR)/ios/headers
	cp generated/$(API_NAME).h $(DIST_DIR)/ios/headers/
	printf 'module CHelloXplatter {\n    header "$(API_NAME).h"\n    export *\n}\n' > $@

$(DIST_DIR)/ios/HelloXplatter.xcframework: $(BUILD_DIR)/ios/ios-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a $(DIST_DIR)/ios/headers/module.modulemap
	rm -rf $@
	xcodebuild -create-xcframework \
		-library $(BUILD_DIR)/ios/ios-arm64/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
		-library $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
		-output $@

$(DIST_DIR)/ios/HelloXplatterLib/Sources/HelloXplatterBinding/HelloXplatter.swift: $(GEN_SWIFT_BINDING)
	@mkdir -p $(dir $@)
	awk '{print} /^import Foundation$$/{print "import CHelloXplatter"}' $(GEN_SWIFT_BINDING) > $@

$(DIST_DIR)/ios/HelloXplatterLib/Package.swift: $(DIST_DIR)/ios/HelloXplatter.xcframework
	@mkdir -p $(dir $@)
	printf '// swift-tools-version: 5.9\nimport PackageDescription\n\nlet package = Package(\n    name: "HelloXplatterLib",\n    platforms: [.iOS(.v15)],\n    products: [\n        .library(name: "HelloXplatterLib", targets: ["HelloXplatterBinding"]),\n    ],\n    targets: [\n        .binaryTarget(name: "CHelloXplatter", path: "../HelloXplatter.xcframework"),\n        .target(\n            name: "HelloXplatterBinding",\n            dependencies: ["CHelloXplatter"],\n            path: "Sources/HelloXplatterBinding"\n        ),\n    ]\n)\n' > $@

.PHONY: package-ios
package-ios: $(DIST_DIR)/ios/HelloXplatter.xcframework $(DIST_DIR)/ios/HelloXplatterLib/Package.swift $(DIST_DIR)/ios/HelloXplatterLib/Sources/HelloXplatterBinding/HelloXplatter.swift
	@echo "Packaged iOS: $(DIST_DIR)/ios/"

endif

# ══════════════════════════════════════════════════════════════════════════════
# Android: Go c-shared (CGO) with JNI bridge → .so per ABI
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,android),)

NDK_VERSION     ?= 29.0.14206865
NDK             ?= $(HOME)/Library/Android/sdk/ndk/$(NDK_VERSION)
NDK_BIN         := $(NDK)/toolchains/llvm/prebuilt/darwin-x86_64/bin
ANDROID_MIN_API := 21

GEN_JNI_SOURCE     := generated/$(API_NAME)_jni.c
GEN_KOTLIN_BINDING := generated/HelloXplatter.kt

# $(1) = ABI name  (e.g. arm64-v8a)
# $(2) = GOARCH    (e.g. arm64)
# $(3) = NDK clang prefix  (e.g. aarch64-linux-android21)
# $(4) = extra env (e.g. GOARM=7, or empty)
#
# Strategy: CGO auto-compiles all .c files in the package directory alongside
# import "C" files. We temporarily copy the JNI bridge into the package root so
# CGO includes it when building the c-shared .so. (Do not run ABI targets in
# parallel — each target writes/removes the same jni.c copy in the package dir.)
define BUILD_ANDROID_ABI_GO

$(DIST_DIR)/android/jniLibs/$(1)/$(LIB_NAME).so: $(STAMP)
	@mkdir -p $$(dir $$@)
	cp $(GEN_JNI_SOURCE) hello_xplatter_jni.c
	CGO_ENABLED=1 GOOS=android GOARCH=$(2) $(4) \
	  CC=$(NDK_BIN)/$(3)-clang \
	  go build -buildmode=c-shared -o $$@ . || (rm -f hello_xplatter_jni.c; exit 1)
	rm -f hello_xplatter_jni.c

endef

$(eval $(call BUILD_ANDROID_ABI_GO,arm64-v8a,arm64,aarch64-linux-android$(ANDROID_MIN_API),))
$(eval $(call BUILD_ANDROID_ABI_GO,armeabi-v7a,arm,armv7a-linux-androideabi$(ANDROID_MIN_API),GOARM=7))
$(eval $(call BUILD_ANDROID_ABI_GO,x86_64,amd64,x86_64-linux-android$(ANDROID_MIN_API),))
$(eval $(call BUILD_ANDROID_ABI_GO,x86,386,i686-linux-android$(ANDROID_MIN_API),))

ANDROID_NATIVE_LIBS := \
	$(DIST_DIR)/android/jniLibs/arm64-v8a/$(LIB_NAME).so \
	$(DIST_DIR)/android/jniLibs/armeabi-v7a/$(LIB_NAME).so \
	$(DIST_DIR)/android/jniLibs/x86_64/$(LIB_NAME).so \
	$(DIST_DIR)/android/jniLibs/x86/$(LIB_NAME).so

$(DIST_DIR)/android/HelloXplatter.kt: $(GEN_KOTLIN_BINDING)
	@mkdir -p $(dir $@)
	cp $(GEN_KOTLIN_BINDING) $@

.PHONY: package-android
package-android: $(ANDROID_NATIVE_LIBS) $(DIST_DIR)/android/HelloXplatter.kt
	@echo "Packaged Android: $(DIST_DIR)/android/"

endif

# ══════════════════════════════════════════════════════════════════════════════
# Aggregate target
# ══════════════════════════════════════════════════════════════════════════════

PACKAGE_TARGETS :=
ifneq ($(call target_enabled,ios),)
PACKAGE_TARGETS += package-ios
endif
ifneq ($(call target_enabled,android),)
PACKAGE_TARGETS += package-android
endif
ifneq ($(call target_enabled,web),)
PACKAGE_TARGETS += package-web
endif
ifneq ($(call target_enabled,desktop),)
PACKAGE_TARGETS += package-desktop
endif

.PHONY: packages build
packages: $(PACKAGE_TARGETS)
build: packages

clean:
	rm -rf generated $(BUILD_DIR) $(DIST_DIR) hello_xplatter.h
