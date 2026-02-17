XPLATTERGY ?= ../../../xplattergy.sh
BUILD_DIR  := build
DIST_DIR   := dist
STAMP      := $(BUILD_DIR)/.generated

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif
SHARED_LIB := $(BUILD_DIR)/libhello_xplattergy.$(DYLIB_EXT)

API_NAME := hello_xplattergy
LIB_NAME := libhello_xplattergy

.PHONY: run shared-lib package-desktop package-web package-android package-ios clean

run: $(STAMP)
	go build -o $(BUILD_DIR)/hello_xplattergy .
	./$(BUILD_DIR)/hello_xplattergy

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(STAMP)
	go build -buildmode=c-shared -ldflags='-extldflags "-Wl,-install_name,@rpath/libhello_xplattergy.$(DYLIB_EXT)"' -o $(SHARED_LIB) .

# Generate and copy C header locally (cgo needs headers in the package directory)
$(STAMP): ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTERGY) generate --impl-lang go --clean -o generated $<
	cp generated/hello_xplattergy.h .
	@touch $@

$(DIST_DIR)/desktop/include/$(API_NAME).h: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/$(API_NAME).h $@

$(DIST_DIR)/desktop/include/HelloXplattergy.swift: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/HelloXplattergy.swift $@

$(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT): $(SHARED_LIB)
	@mkdir -p $(dir $@)
	cp $(SHARED_LIB) $@

package-desktop: $(STAMP) $(DIST_DIR)/desktop/include/$(API_NAME).h $(DIST_DIR)/desktop/include/HelloXplattergy.swift $(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT)
	@echo "Packaged Desktop: $(DIST_DIR)/desktop/"

$(DIST_DIR)/web/hello_xplattergy.wasm: $(STAMP)
	@mkdir -p $(dir $@)
	GOOS=wasip1 GOARCH=wasm go build -o $@ .

$(DIST_DIR)/web/hello_xplattergy.js: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/hello_xplattergy.js $@

package-web: $(DIST_DIR)/web/hello_xplattergy.wasm $(DIST_DIR)/web/hello_xplattergy.js
	@echo "Packaged Web: $(DIST_DIR)/web/"

# ══════════════════════════════════════════════════════════════════════════════
# iOS: Go c-archive (CGO) per arch → lipo → xcframework + SPM package
# ══════════════════════════════════════════════════════════════════════════════

IOS_MIN := 15.0
IOS_SDK := $(shell xcrun --sdk iphoneos --show-sdk-path 2>/dev/null)
SIM_SDK := $(shell xcrun --sdk iphonesimulator --show-sdk-path 2>/dev/null)
IOS_CC  := $(shell xcrun --sdk iphoneos --find clang 2>/dev/null)
SIM_CC  := $(shell xcrun --sdk iphonesimulator --find clang 2>/dev/null)

GEN_SWIFT_BINDING := generated/HelloXplattergy.swift

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
	printf 'module CHelloXplattergy {\n    header "$(API_NAME).h"\n    export *\n}\n' > $@

$(DIST_DIR)/ios/HelloXplattergy.xcframework: $(BUILD_DIR)/ios/ios-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a $(DIST_DIR)/ios/headers/module.modulemap
	rm -rf $@
	xcodebuild -create-xcframework \
		-library $(BUILD_DIR)/ios/ios-arm64/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
		-library $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
		-output $@

$(DIST_DIR)/ios/HelloXplattergyLib/Sources/HelloXplattergyBinding/HelloXplattergy.swift: $(GEN_SWIFT_BINDING)
	@mkdir -p $(dir $@)
	awk '{print} /^import Foundation$$/{print "import CHelloXplattergy"}' $(GEN_SWIFT_BINDING) > $@

$(DIST_DIR)/ios/HelloXplattergyLib/Package.swift: $(DIST_DIR)/ios/HelloXplattergy.xcframework
	@mkdir -p $(dir $@)
	printf '// swift-tools-version: 5.9\nimport PackageDescription\n\nlet package = Package(\n    name: "HelloXplattergyLib",\n    platforms: [.iOS(.v15)],\n    products: [\n        .library(name: "HelloXplattergyLib", targets: ["HelloXplattergyBinding"]),\n    ],\n    targets: [\n        .binaryTarget(name: "CHelloXplattergy", path: "../HelloXplattergy.xcframework"),\n        .target(\n            name: "HelloXplattergyBinding",\n            dependencies: ["CHelloXplattergy"],\n            path: "Sources/HelloXplattergyBinding"\n        ),\n    ]\n)\n' > $@

package-ios: $(DIST_DIR)/ios/HelloXplattergy.xcframework $(DIST_DIR)/ios/HelloXplattergyLib/Package.swift $(DIST_DIR)/ios/HelloXplattergyLib/Sources/HelloXplattergyBinding/HelloXplattergy.swift
	@echo "Packaged iOS: $(DIST_DIR)/ios/"

# ══════════════════════════════════════════════════════════════════════════════
# Android: Go c-shared (CGO) with JNI bridge → .so per ABI
# ══════════════════════════════════════════════════════════════════════════════

NDK_VERSION     ?= 29.0.14206865
NDK             ?= $(HOME)/Library/Android/sdk/ndk/$(NDK_VERSION)
NDK_BIN         := $(NDK)/toolchains/llvm/prebuilt/darwin-x86_64/bin
ANDROID_MIN_API := 21

GEN_JNI_SOURCE     := generated/$(API_NAME)_jni.c
GEN_KOTLIN_BINDING := generated/HelloXplattergy.kt

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
	cp $(GEN_JNI_SOURCE) hello_xplattergy_jni.c
	CGO_ENABLED=1 GOOS=android GOARCH=$(2) $(4) \
	  CC=$(NDK_BIN)/$(3)-clang \
	  go build -buildmode=c-shared -o $$@ . || (rm -f hello_xplattergy_jni.c; exit 1)
	rm -f hello_xplattergy_jni.c

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

$(DIST_DIR)/android/HelloXplattergy.kt: $(GEN_KOTLIN_BINDING)
	@mkdir -p $(dir $@)
	cp $(GEN_KOTLIN_BINDING) $@

package-android: $(ANDROID_NATIVE_LIBS) $(DIST_DIR)/android/HelloXplattergy.kt
	@echo "Packaged Android: $(DIST_DIR)/android/"

clean:
	rm -rf generated $(BUILD_DIR) $(DIST_DIR) hello_xplattergy.h
