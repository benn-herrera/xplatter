XPLATTERGY ?= ../../../xplattergy.sh
IMPL       ?= cpp
IMPL_DIR   := ../$(IMPL)
BUILD_DIR  := build

# Use generated files from impl dir if available, otherwise generate our own.
GEN_DIR = $(if $(wildcard $(IMPL_DIR)/generated/hello_xplattergy.h),$(IMPL_DIR)/generated,$(BUILD_DIR)/generated)

# iOS deployment target
IOS_MIN := 15.0

# Architecture targets
ARCHS := ios-arm64 ios-sim-arm64 ios-sim-x86_64

# Source files
IMPL_CXX_SRC     := $(IMPL_DIR)/hello_xplattergy_impl.cpp
SHIM_CXX_SRC      = $(GEN_DIR)/hello_xplattergy_shim.cpp
PLATFORM_C_SRC    := platform_services_ios.c

# Common flags
COMMON_CFLAGS := -fvisibility=hidden -DHELLO_XPLATTERGY_BUILD
CXXFLAGS      := -std=c++20 -Wall -Wextra $(COMMON_CFLAGS)
CFLAGS        := -std=c11 -Wall -Wextra $(COMMON_CFLAGS)

.PHONY: xcframework package build-app test clean

# End-to-end: build everything needed for the iOS app
test: xcframework package build-app
	@echo "PASS: iOS app builds for simulator"

# --- Step 1: Cross-compile static libraries ---

define BUILD_ARCH
# $(1) = arch dir name, $(2) = clang target triple, $(3) = SDK name

$(BUILD_DIR)/$(1)/impl.o: $(IMPL_CXX_SRC) $(GEN_DIR)/hello_xplattergy.h
	@mkdir -p $(BUILD_DIR)/$(1)
	xcrun --sdk $(3) clang++ $(CXXFLAGS) -target $(2) \
		-I$(IMPL_DIR) -I$(GEN_DIR) -c -o $$@ $$<

$(BUILD_DIR)/$(1)/shim.o: $(GEN_DIR)/hello_xplattergy_shim.cpp $(GEN_DIR)/hello_xplattergy.h
	@mkdir -p $(BUILD_DIR)/$(1)
	xcrun --sdk $(3) clang++ $(CXXFLAGS) -target $(2) \
		-I$(GEN_DIR) -c -o $$@ $$<

$(BUILD_DIR)/$(1)/platform.o: $(PLATFORM_C_SRC) $(GEN_DIR)/hello_xplattergy.h
	@mkdir -p $(BUILD_DIR)/$(1)
	xcrun --sdk $(3) clang $(CFLAGS) -target $(2) \
		-I$(GEN_DIR) -c -o $$@ $$<

$(BUILD_DIR)/$(1)/libhello_xplattergy.a: $(BUILD_DIR)/$(1)/impl.o $(BUILD_DIR)/$(1)/shim.o $(BUILD_DIR)/$(1)/platform.o
	ar rcs $$@ $$^

endef

$(eval $(call BUILD_ARCH,ios-arm64,arm64-apple-ios$(IOS_MIN),iphoneos))
$(eval $(call BUILD_ARCH,ios-sim-arm64,arm64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))
$(eval $(call BUILD_ARCH,ios-sim-x86_64,x86_64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))

# Fat simulator library (arm64 + x86_64)
$(BUILD_DIR)/ios-sim-fat/libhello_xplattergy.a: $(BUILD_DIR)/ios-sim-arm64/libhello_xplattergy.a $(BUILD_DIR)/ios-sim-x86_64/libhello_xplattergy.a
	@mkdir -p $(BUILD_DIR)/ios-sim-fat
	lipo -create $^ -output $@

# --- Step 2: XCFramework ---

$(BUILD_DIR)/headers/module.modulemap: $(GEN_DIR)/hello_xplattergy.h
	@mkdir -p $(BUILD_DIR)/headers
	cp $(GEN_DIR)/hello_xplattergy.h $(BUILD_DIR)/headers/
	printf 'module CHelloXplattergy {\n    header "hello_xplattergy.h"\n    export *\n}\n' > $@

$(BUILD_DIR)/HelloXplattergy.xcframework: $(BUILD_DIR)/ios-arm64/libhello_xplattergy.a $(BUILD_DIR)/ios-sim-fat/libhello_xplattergy.a $(BUILD_DIR)/headers/module.modulemap
	rm -rf $@
	xcodebuild -create-xcframework \
		-library $(BUILD_DIR)/ios-arm64/libhello_xplattergy.a -headers $(BUILD_DIR)/headers \
		-library $(BUILD_DIR)/ios-sim-fat/libhello_xplattergy.a -headers $(BUILD_DIR)/headers \
		-output $@

xcframework: $(BUILD_DIR)/HelloXplattergy.xcframework

# --- Step 3: Stage Swift binding into SPM package ---

SWIFT_BINDING_SRC = $(GEN_DIR)/HelloXplattergy.swift
SWIFT_BINDING_DST := HelloXplattergyLib/Sources/HelloXplattergyBinding/HelloXplattergy.swift

$(SWIFT_BINDING_DST): $(SWIFT_BINDING_SRC)
	@mkdir -p $(dir $@)
	awk '{print} /^import Foundation$$/{print "import CHelloXplattergy"}' $(SWIFT_BINDING_SRC) > $@

package: $(SWIFT_BINDING_DST)

# --- Step 4: Build iOS app for simulator ---

build-app: xcframework package
	xcodebuild -project HelloApp/HelloApp.xcodeproj \
		-scheme HelloApp \
		-destination 'generic/platform=iOS Simulator' \
		build SYMROOT=$(CURDIR)/$(BUILD_DIR)/app

# --- Codegen fallback (if impl dir has no generated files) ---

$(BUILD_DIR)/generated/hello_xplattergy.h $(BUILD_DIR)/generated/HelloXplattergy.swift: ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)/generated
	$(XPLATTERGY) generate --impl-lang $(IMPL) --clean -o $(BUILD_DIR)/generated $<

# --- Clean ---

clean:
	rm -rf $(BUILD_DIR) $(SWIFT_BINDING_DST)
