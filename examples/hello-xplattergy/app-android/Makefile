XPLATTERGY ?= ../../../bin/xplattergy
IMPL       ?= cpp
IMPL_DIR   := ../$(IMPL)
BUILD_DIR  := build

# Use generated files from impl dir if available, otherwise generate our own.
GEN_DIR = $(if $(wildcard $(IMPL_DIR)/generated/hello_xplattergy.h),$(IMPL_DIR)/generated,$(BUILD_DIR)/generated)

# NDK configuration
NDK_VERSION := 29.0.14206865
NDK         := $(HOME)/Library/Android/sdk/ndk/$(NDK_VERSION)
NDK_BIN     := $(NDK)/toolchains/llvm/prebuilt/darwin-x86_64/bin

# Source files
IMPL_CXX_SRC  := $(IMPL_DIR)/hello_xplattergy_impl.cpp
SHIM_CXX_SRC   = $(GEN_DIR)/hello_xplattergy_shim.cpp
JNI_C_SRC       = $(GEN_DIR)/hello_xplattergy_jni.c
PLATFORM_C_SRC := platform_services_android.c

# Common flags
COMMON_CFLAGS := -fPIC -fvisibility=hidden -DHELLO_XPLATTERGY_BUILD
CXXFLAGS      := -std=c++20 $(COMMON_CFLAGS)
CFLAGS        := -std=c11 $(COMMON_CFLAGS)
LDFLAGS       := -shared -llog

# Gradle wrapper source
GRADLE_WRAPPER_SRC := /Users/benn/projects/letterboxed_android

.PHONY: native-libs package build-app test clean gradle-wrapper

# End-to-end: build everything needed for the Android app
test: native-libs package build-app
	@echo "PASS: Android app builds (assembleDebug)"

# --- Architecture definitions ---
# $(1) = ABI name, $(2) = NDK target triple

define BUILD_ABI

$(BUILD_DIR)/jniLibs/$(1)/libhello_xplattergy.so: $(IMPL_CXX_SRC) $(SHIM_CXX_SRC) $(JNI_C_SRC) $(PLATFORM_C_SRC) $(GEN_DIR)/hello_xplattergy.h
	@mkdir -p $(BUILD_DIR)/obj/$(1) $(BUILD_DIR)/jniLibs/$(1)
	$(NDK_BIN)/$(2)-clang++ $(CXXFLAGS) \
		-I$(IMPL_DIR) -I$(GEN_DIR) \
		-c -o $(BUILD_DIR)/obj/$(1)/impl.o $(IMPL_CXX_SRC)
	$(NDK_BIN)/$(2)-clang++ $(CXXFLAGS) \
		-I$(GEN_DIR) \
		-c -o $(BUILD_DIR)/obj/$(1)/shim.o $(SHIM_CXX_SRC)
	$(NDK_BIN)/$(2)-clang $(CFLAGS) \
		-I$(GEN_DIR) \
		-c -o $(BUILD_DIR)/obj/$(1)/jni.o $(JNI_C_SRC)
	$(NDK_BIN)/$(2)-clang $(CFLAGS) \
		-I$(GEN_DIR) \
		-c -o $(BUILD_DIR)/obj/$(1)/platform.o $(PLATFORM_C_SRC)
	$(NDK_BIN)/$(2)-clang++ $(LDFLAGS) \
		$(BUILD_DIR)/obj/$(1)/impl.o \
		$(BUILD_DIR)/obj/$(1)/shim.o \
		$(BUILD_DIR)/obj/$(1)/jni.o \
		$(BUILD_DIR)/obj/$(1)/platform.o \
		-o $$@

endef

$(eval $(call BUILD_ABI,arm64-v8a,aarch64-linux-android21))
$(eval $(call BUILD_ABI,armeabi-v7a,armv7a-linux-androideabi21))
$(eval $(call BUILD_ABI,x86_64,x86_64-linux-android21))
$(eval $(call BUILD_ABI,x86,i686-linux-android21))

NATIVE_LIBS := \
	$(BUILD_DIR)/jniLibs/arm64-v8a/libhello_xplattergy.so \
	$(BUILD_DIR)/jniLibs/armeabi-v7a/libhello_xplattergy.so \
	$(BUILD_DIR)/jniLibs/x86_64/libhello_xplattergy.so \
	$(BUILD_DIR)/jniLibs/x86/libhello_xplattergy.so

native-libs: $(NATIVE_LIBS)

# --- Package: copy generated Kotlin + .so files into Gradle project ---

package: native-libs
	@mkdir -p lib/src/main/kotlin/hello/xplattergy
	cp $(GEN_DIR)/HelloXplattergy.kt lib/src/main/kotlin/hello/xplattergy/
	@for abi in arm64-v8a armeabi-v7a x86_64 x86; do \
		mkdir -p lib/src/main/jniLibs/$$abi; \
		cp $(BUILD_DIR)/jniLibs/$$abi/libhello_xplattergy.so lib/src/main/jniLibs/$$abi/; \
	done
	@echo "Packaged Kotlin + native libs into Gradle project"

# --- Build the Android app ---

build-app: gradle-wrapper package
	./gradlew :app:assembleDebug

# --- Gradle wrapper ---

gradle-wrapper:
	@if [ ! -f gradlew ]; then \
		echo "Copying Gradle wrapper from $(GRADLE_WRAPPER_SRC)..."; \
		cp $(GRADLE_WRAPPER_SRC)/gradlew .; \
		chmod +x gradlew; \
		cp $(GRADLE_WRAPPER_SRC)/gradlew.bat .; \
		cp $(GRADLE_WRAPPER_SRC)/gradle/wrapper/gradle-wrapper.jar gradle/wrapper/; \
	fi

# --- Codegen fallback (if impl dir has no generated files) ---

$(BUILD_DIR)/generated/hello_xplattergy.h $(BUILD_DIR)/generated/HelloXplattergy.kt: $(IMPL_DIR)/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)/generated
	$(XPLATTERGY) generate --clean -o $(BUILD_DIR)/generated $<

# --- Clean ---

clean:
	rm -rf $(BUILD_DIR)
	rm -rf lib/src/main/kotlin/hello/xplattergy/HelloXplattergy.kt
	rm -rf lib/src/main/jniLibs/*/libhello_xplattergy.so
