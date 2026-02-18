# cross-compile-cc.mk — Shared cross-compilation infrastructure for C/C++ impls
#
# Each impl Makefile sets these variables then includes this file:
#   API_NAME          — e.g. hello_xplatter
#   IMPL_SOURCES      — impl .cpp/.c files (not shim/platform)
#   SHIM_SOURCE       — generated shim .cpp file (empty if no shim, e.g. C impl)
#   JNI_SOURCE        — generated JNI .c file (set by includer or empty)
#   IMPL_INCLUDES     — -I flags for impl sources
#   GEN_INCLUDES      — -I flags for generated headers
#   COMPILER          — c++ or cc
#   COMPILER_FLAGS    — e.g. -std=c++20
#   DIST_DIR          — output directory (e.g. dist)
#   PLATFORM_SERVICES — path to platform_services/ directory

# ── Host / toolchain detection ────────────────────────────────────────────────

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif

# ── Target filtering ──────────────────────────────────────────────────────────

TARGETS ?= ios android web desktop

# ── NDK configuration ─────────────────────────────────────────────────────────

NDK_VERSION ?= 29.0.14206865
NDK ?= $(HOME)/Library/Android/sdk/ndk/$(NDK_VERSION)
NDK_BIN := $(NDK)/toolchains/llvm/prebuilt/darwin-x86_64/bin
ANDROID_MIN_API := 21

# ── Emscripten ────────────────────────────────────────────────────────────────

EMCC ?= emcc

# ── iOS ───────────────────────────────────────────────────────────────────────

IOS_MIN := 15.0

# ── Common flags ──────────────────────────────────────────────────────────────

LIB_VISIBILITY_FLAGS := -fvisibility=hidden -DHELLO_XPLATTER_BUILD
LIB_C_FLAGS := -std=c11 -Wall -Wextra $(LIB_VISIBILITY_FLAGS)

# ── WASM exports ──────────────────────────────────────────────────────────────
# TODO: have the codegen tool output wasm_exports.txt
WASM_EXPORTS := ["_malloc","_free","_$(API_NAME)_lifecycle_create_greeter","_$(API_NAME)_lifecycle_destroy_greeter","_$(API_NAME)_greeter_say_hello"]

# ── Derived names ─────────────────────────────────────────────────────────────

LIB_NAME := lib$(API_NAME)

# ── Helper: is target enabled? ────────────────────────────────────────────────

target_enabled = $(filter $(1),$(TARGETS))

# ══════════════════════════════════════════════════════════════════════════════
# iOS static library — one per architecture
# ══════════════════════════════════════════════════════════════════════════════

# $(1) = arch dir name (e.g. ios-arm64)
# $(2) = clang target triple (e.g. arm64-apple-ios15.0)
# $(3) = SDK name (e.g. iphoneos)
define BUILD_IOS_ARCH

$(DIST_DIR)/ios/obj/$(1)/impl.o: $(IMPL_SOURCES) $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	xcrun --sdk $(3) $(COMPILER) $(COMPILER_FLAGS) $(LIB_VISIBILITY_FLAGS) \
		-target $(2) $(IMPL_INCLUDES) $(GEN_INCLUDES) -c -o $$@ $$<

$(if $(SHIM_SOURCE),
$(DIST_DIR)/ios/obj/$(1)/shim.o: $(SHIM_SOURCE) $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	xcrun --sdk $(3) $(COMPILER) $(COMPILER_FLAGS) $(LIB_VISIBILITY_FLAGS) \
		-target $(2) $(GEN_INCLUDES) -c -o $$@ $$<
)

$(DIST_DIR)/ios/obj/$(1)/platform.o: $(PLATFORM_SERVICES)/ios.c $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	xcrun --sdk $(3) clang $(LIB_C_FLAGS) \
		-target $(2) $(GEN_INCLUDES) -c -o $$@ $$<

$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(DIST_DIR)/ios/obj/$(1)/impl.o $(if $(SHIM_SOURCE),$(DIST_DIR)/ios/obj/$(1)/shim.o) $(DIST_DIR)/ios/obj/$(1)/platform.o
	ar rcs $$@ $$^

endef

# ══════════════════════════════════════════════════════════════════════════════
# Android shared library — one per ABI
# ══════════════════════════════════════════════════════════════════════════════

# $(1) = ABI name (e.g. arm64-v8a)
# $(2) = NDK target triple (e.g. aarch64-linux-android21)
define BUILD_ANDROID_ABI

$(DIST_DIR)/android/jniLibs/$(1)/$(LIB_NAME).so: $(IMPL_SOURCES) $(SHIM_SOURCE) $(JNI_SOURCE) $(PLATFORM_SERVICES)/android.c $(GEN_HEADER)
	@mkdir -p $(DIST_DIR)/android/obj/$(1) $$(dir $$@)
	$(NDK_BIN)/$(2)-$(COMPILER) $(COMPILER_FLAGS) -fPIC $(LIB_VISIBILITY_FLAGS) \
		$(IMPL_INCLUDES) $(GEN_INCLUDES) \
		-c -o $(DIST_DIR)/android/obj/$(1)/impl.o $(IMPL_SOURCES)
	$(if $(SHIM_SOURCE),$(NDK_BIN)/$(2)-$(COMPILER) $(COMPILER_FLAGS) -fPIC $(LIB_VISIBILITY_FLAGS) \
		$(GEN_INCLUDES) \
		-c -o $(DIST_DIR)/android/obj/$(1)/shim.o $(SHIM_SOURCE))
	$(NDK_BIN)/$(2)-clang $(LIB_C_FLAGS) -fPIC \
		$(GEN_INCLUDES) \
		-c -o $(DIST_DIR)/android/obj/$(1)/jni.o $(JNI_SOURCE)
	$(NDK_BIN)/$(2)-clang $(LIB_C_FLAGS) -fPIC \
		$(GEN_INCLUDES) \
		-c -o $(DIST_DIR)/android/obj/$(1)/platform.o $(PLATFORM_SERVICES)/android.c
	$(NDK_BIN)/$(2)-$(COMPILER) -shared -llog \
		$(DIST_DIR)/android/obj/$(1)/impl.o \
		$(if $(SHIM_SOURCE),$(DIST_DIR)/android/obj/$(1)/shim.o) \
		$(DIST_DIR)/android/obj/$(1)/jni.o \
		$(DIST_DIR)/android/obj/$(1)/platform.o \
		-o $$@

endef

# ══════════════════════════════════════════════════════════════════════════════
# WASM via Emscripten
# ══════════════════════════════════════════════════════════════════════════════

define BUILD_WASM

$(DIST_DIR)/web/obj/impl.o: $(IMPL_SOURCES) $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	$(EMCC) $(COMPILER_FLAGS) -O2 $(LIB_VISIBILITY_FLAGS) \
		$(IMPL_INCLUDES) $(GEN_INCLUDES) -c -o $$@ $$<

$(if $(SHIM_SOURCE),
$(DIST_DIR)/web/obj/shim.o: $(SHIM_SOURCE) $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	$(EMCC) $(COMPILER_FLAGS) -O2 $(LIB_VISIBILITY_FLAGS) \
		$(GEN_INCLUDES) -c -o $$@ $$<
)

$(DIST_DIR)/web/obj/platform.o: $(PLATFORM_SERVICES)/web.c
	@mkdir -p $$(dir $$@)
	$(EMCC) $(LIB_C_FLAGS) -O2 -c -o $$@ $$<

$(DIST_DIR)/web/$(API_NAME).wasm: $(DIST_DIR)/web/obj/impl.o $(if $(SHIM_SOURCE),$(DIST_DIR)/web/obj/shim.o) $(DIST_DIR)/web/obj/platform.o
	$(EMCC) -o $$@ $$^ \
		--no-entry \
		-s 'EXPORTED_FUNCTIONS=$(WASM_EXPORTS)' \
		-s STANDALONE_WASM \
		-O2

endef

# ══════════════════════════════════════════════════════════════════════════════
# Packaging targets
# ══════════════════════════════════════════════════════════════════════════════

# ── iOS: lipo + xcframework + SPM package ─────────────────────────────────────

define PACKAGE_IOS

$(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a: $(DIST_DIR)/ios/obj/ios-sim-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-x86_64/$(LIB_NAME).a
	@mkdir -p $$(dir $$@)
	lipo -create $$^ -output $$@

$(DIST_DIR)/ios/headers/module.modulemap: $(GEN_HEADER)
	@mkdir -p $(DIST_DIR)/ios/headers
	cp $(GEN_HEADER) $(DIST_DIR)/ios/headers/
	printf 'module CHelloXplatter {\n    header "$(API_NAME).h"\n    export *\n}\n' > $$@

$(DIST_DIR)/ios/HelloXplatter.xcframework: $(DIST_DIR)/ios/obj/ios-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a $(DIST_DIR)/ios/headers/module.modulemap
	rm -rf $$@
	xcodebuild -create-xcframework \
		-library $(DIST_DIR)/ios/obj/ios-arm64/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
		-library $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
		-output $$@

$(DIST_DIR)/ios/HelloXplatterLib/Sources/HelloXplatterBinding/HelloXplatter.swift: $(GEN_SWIFT_BINDING)
	@mkdir -p $$(dir $$@)
	awk '{print} /^import Foundation$$$$/{print "import CHelloXplatter"}' $(GEN_SWIFT_BINDING) > $$@

$(DIST_DIR)/ios/HelloXplatterLib/Package.swift: $(DIST_DIR)/ios/HelloXplatter.xcframework
	@mkdir -p $$(dir $$@)
	printf '// swift-tools-version: 5.9\nimport PackageDescription\n\nlet package = Package(\n    name: "HelloXplatterLib",\n    platforms: [.iOS(.v15)],\n    products: [\n        .library(name: "HelloXplatterLib", targets: ["HelloXplatterBinding"]),\n    ],\n    targets: [\n        .binaryTarget(name: "CHelloXplatter", path: "../HelloXplatter.xcframework"),\n        .target(\n            name: "HelloXplatterBinding",\n            dependencies: ["CHelloXplatter"],\n            path: "Sources/HelloXplatterBinding",\n            linkerSettings: [.linkedLibrary("c++")]\n        ),\n    ]\n)\n' > $$@

.PHONY: package-ios
package-ios: $(DIST_DIR)/ios/HelloXplatter.xcframework $(DIST_DIR)/ios/HelloXplatterLib/Package.swift $(DIST_DIR)/ios/HelloXplatterLib/Sources/HelloXplatterBinding/HelloXplatter.swift
	@echo "Packaged iOS: $(DIST_DIR)/ios/"

endef

# ── Android: .so per ABI + Kotlin binding ─────────────────────────────────────

define PACKAGE_ANDROID

ANDROID_NATIVE_LIBS := \
	$(DIST_DIR)/android/jniLibs/arm64-v8a/$(LIB_NAME).so \
	$(DIST_DIR)/android/jniLibs/armeabi-v7a/$(LIB_NAME).so \
	$(DIST_DIR)/android/jniLibs/x86_64/$(LIB_NAME).so \
	$(DIST_DIR)/android/jniLibs/x86/$(LIB_NAME).so

$(DIST_DIR)/android/HelloXplatter.kt: $(GEN_KOTLIN_BINDING)
	@mkdir -p $$(dir $$@)
	cp $(GEN_KOTLIN_BINDING) $$@

.PHONY: package-android
package-android: $$(ANDROID_NATIVE_LIBS) $(DIST_DIR)/android/HelloXplatter.kt
	@echo "Packaged Android: $(DIST_DIR)/android/"

endef

# ── Web: .wasm + .js binding ──────────────────────────────────────────────────

define PACKAGE_WEB

$(DIST_DIR)/web/$(API_NAME).js: $(GEN_JS_BINDING)
	@mkdir -p $$(dir $$@)
	cp $(GEN_JS_BINDING) $$@

.PHONY: package-web
package-web: $(DIST_DIR)/web/$(API_NAME).wasm $(DIST_DIR)/web/$(API_NAME).js
	@echo "Packaged Web: $(DIST_DIR)/web/"

endef

# ── Desktop: C header + shared library ────────────────────────────────────────

define PACKAGE_DESKTOP

$(DIST_DIR)/desktop/include/$(API_NAME).h: $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	cp $(GEN_HEADER) $$@

$(DIST_DIR)/desktop/include/HelloXplatter.swift: $(GEN_SWIFT_BINDING)
	@mkdir -p $$(dir $$@)
	cp $(GEN_SWIFT_BINDING) $$@

$(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT): $(SHARED_LIB)
	@mkdir -p $$(dir $$@)
	cp $(SHARED_LIB) $$@

.PHONY: package-desktop
package-desktop: $(DIST_DIR)/desktop/include/$(API_NAME).h $(DIST_DIR)/desktop/include/HelloXplatter.swift $(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT)
	@echo "Packaged Desktop: $(DIST_DIR)/desktop/"

endef

# ══════════════════════════════════════════════════════════════════════════════
# Evaluate all templates (caller must have set all required variables)
# ══════════════════════════════════════════════════════════════════════════════

# iOS architectures
ifneq ($(call target_enabled,ios),)
$(eval $(call BUILD_IOS_ARCH,ios-arm64,arm64-apple-ios$(IOS_MIN),iphoneos))
$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,arm64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))
$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,x86_64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))
$(eval $(call PACKAGE_IOS))
endif

# Android ABIs
ifneq ($(call target_enabled,android),)
$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,aarch64-linux-android$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,armv7a-linux-androideabi$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,x86_64,x86_64-linux-android$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,x86,i686-linux-android$(ANDROID_MIN_API)))
$(eval $(call PACKAGE_ANDROID))
endif

# WASM
ifneq ($(call target_enabled,web),)
$(eval $(call BUILD_WASM))
$(eval $(call PACKAGE_WEB))
endif

# Desktop
ifneq ($(call target_enabled,desktop),)
$(eval $(call PACKAGE_DESKTOP))
endif

# ── Aggregate target ──────────────────────────────────────────────────────────

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
