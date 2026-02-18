XPLATTER ?= ../../../xplatter.sh
BUILD_DIR  := build
STAMP      := $(BUILD_DIR)/.generated

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif
SHARED_LIB := $(BUILD_DIR)/libhello_xplatter.$(DYLIB_EXT)

.PHONY: run shared-lib clean

run: shared-lib
	cargo test

shared-lib: $(SHARED_LIB)

$(SHARED_LIB):
	cargo build --release
	@mkdir -p $(BUILD_DIR)
	cp target/release/libhello_xplatter.$(DYLIB_EXT) $(SHARED_LIB)
ifeq ($(UNAME_S),Darwin)
	install_name_tool -id @rpath/libhello_xplatter.$(DYLIB_EXT) $(SHARED_LIB)
endif

$(STAMP): ../shared_schemas/hello_xplatter.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTER) generate --impl-lang rust --clean -o generated $<
	@touch $@

clean:
	cargo clean
	rm -rf $(BUILD_DIR) generated dist

# ══════════════════════════════════════════════════════════════════════════════
# Cross-platform packaging
# ══════════════════════════════════════════════════════════════════════════════

API_NAME := hello_xplatter
LIB_NAME := lib$(API_NAME)
DIST_DIR := dist

# ── Target filtering ──────────────────────────────────────────────────────────

TARGETS ?= ios android web desktop

target_enabled = $(filter $(1),$(TARGETS))

# ── NDK configuration ─────────────────────────────────────────────────────────

NDK_VERSION ?= 29.0.14206865
NDK ?= $(HOME)/Library/Android/sdk/ndk/$(NDK_VERSION)
NDK_BIN := $(NDK)/toolchains/llvm/prebuilt/darwin-x86_64/bin
ANDROID_MIN_API := 21

LIB_C_FLAGS := -std=c11 -Wall -Wextra -fvisibility=hidden -DHELLO_XPLATTER_BUILD

# ── iOS ───────────────────────────────────────────────────────────────────────

IOS_MIN := 15.0

# ── Generated binding files (produced by codegen) ────────────────────────────

GEN_HEADER         := generated/$(API_NAME).h
GEN_SWIFT_BINDING  := generated/HelloXplatter.swift
GEN_KOTLIN_BINDING := generated/HelloXplatter.kt
GEN_JS_BINDING     := generated/$(API_NAME).js
GEN_JNI_SOURCE     := generated/$(API_NAME)_jni.c

# Ensure codegen runs before any packaging target needs generated files
generated/$(API_NAME).h: $(STAMP)

# ══════════════════════════════════════════════════════════════════════════════
# Desktop: C header + Swift binding + shared library
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,desktop),)

$(DIST_DIR)/desktop/include/$(API_NAME).h: $(GEN_HEADER)
	@mkdir -p $(dir $@)
	cp $(GEN_HEADER) $@

$(DIST_DIR)/desktop/include/HelloXplatter.swift: $(GEN_SWIFT_BINDING)
	@mkdir -p $(dir $@)
	cp $(GEN_SWIFT_BINDING) $@

$(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT): $(SHARED_LIB)
	@mkdir -p $(dir $@)
	cp $(SHARED_LIB) $@

.PHONY: package-desktop
package-desktop: $(STAMP) $(DIST_DIR)/desktop/include/$(API_NAME).h $(DIST_DIR)/desktop/include/HelloXplatter.swift $(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT)
	@echo "Packaged Desktop: $(DIST_DIR)/desktop/"

endif

# ══════════════════════════════════════════════════════════════════════════════
# iOS: static libs per arch → lipo → xcframework + SPM package
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,ios),)

# $(1) = arch dir name, $(2) = Rust target triple
define BUILD_IOS_ARCH

$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(STAMP)
	@mkdir -p $$(dir $$@)
	cargo build --release --target $(2)
	cp target/$(2)/release/$(LIB_NAME).a $$@

endef

$(eval $(call BUILD_IOS_ARCH,ios-arm64,aarch64-apple-ios))
$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,aarch64-apple-ios-sim))
$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,x86_64-apple-ios))

$(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a: $(DIST_DIR)/ios/obj/ios-sim-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-x86_64/$(LIB_NAME).a
	@mkdir -p $(dir $@)
	lipo -create $^ -output $@

$(DIST_DIR)/ios/headers/module.modulemap: $(GEN_HEADER)
	@mkdir -p $(DIST_DIR)/ios/headers
	cp $(GEN_HEADER) $(DIST_DIR)/ios/headers/
	printf 'module CHelloXplatter {\n    header "$(API_NAME).h"\n    export *\n}\n' > $@

$(DIST_DIR)/ios/HelloXplatter.xcframework: $(DIST_DIR)/ios/obj/ios-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a $(DIST_DIR)/ios/headers/module.modulemap
	rm -rf $@
	xcodebuild -create-xcframework \
		-library $(DIST_DIR)/ios/obj/ios-arm64/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
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
# Android: Rust staticlib + NDK link → .so per ABI
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,android),)

# $(1) = ABI name (e.g. arm64-v8a)
# $(2) = Rust target triple (e.g. aarch64-linux-android)
# $(3) = NDK target prefix (e.g. aarch64-linux-android21)
define BUILD_ANDROID_ABI

$(DIST_DIR)/android/jniLibs/$(1)/$(LIB_NAME).so: $(STAMP)
	@mkdir -p $(DIST_DIR)/android/obj/$(1) $$(dir $$@)
	PATH=$(NDK_BIN):$$$$PATH cargo build --release --target $(2)
	$(NDK_BIN)/$(3)-clang $(LIB_C_FLAGS) -fPIC \
		-Igenerated -c -o $(DIST_DIR)/android/obj/$(1)/jni.o $(GEN_JNI_SOURCE)
	$(NDK_BIN)/$(3)-clang -shared \
		-Wl,--whole-archive target/$(2)/release/$(LIB_NAME).a -Wl,--no-whole-archive \
		$(DIST_DIR)/android/obj/$(1)/jni.o \
		-ldl -lm -llog \
		-o $$@

endef

$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,aarch64-linux-android,aarch64-linux-android$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,armv7-linux-androideabi,armv7a-linux-androideabi$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,x86_64,x86_64-linux-android,x86_64-linux-android$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,x86,i686-linux-android,i686-linux-android$(ANDROID_MIN_API)))

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
# Web: Rust → wasm32-unknown-unknown + JS binding
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,web),)

$(DIST_DIR)/web/$(API_NAME).wasm: $(STAMP)
	@mkdir -p $(dir $@)
	cargo build --release --target wasm32-unknown-unknown
	cp target/wasm32-unknown-unknown/release/hello_xplatter.wasm $@

$(DIST_DIR)/web/$(API_NAME).js: $(GEN_JS_BINDING)
	@mkdir -p $(dir $@)
	cp $(GEN_JS_BINDING) $@

.PHONY: package-web
package-web: $(DIST_DIR)/web/$(API_NAME).wasm $(DIST_DIR)/web/$(API_NAME).js
	@echo "Packaged Web: $(DIST_DIR)/web/"

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
