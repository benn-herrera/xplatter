IMPL       ?= cpp
IMPL_DIR   := ../impl-$(IMPL)
BUILD_DIR  := build

# Gradle wrapper source
GRADLE_WRAPPER_SRC := /Users/benn/projects/letterboxed_android

.PHONY: build test package build-app clean gradle-wrapper

build: build-app

# End-to-end: build the Android app using packages from the impl
test: package build-app
	@echo "PASS: Android app builds (assembleDebug)"

# --- Ensure impl has built Android package ---

.PHONY: ensure-package
ensure-package:
	@test -d $(IMPL_DIR)/dist/android/src/main/jniLibs || $(MAKE) -C $(IMPL_DIR) build

# --- Copy packaged native libs + Kotlin binding into Gradle project ---

package: ensure-package
	@mkdir -p lib/src/main/kotlin/hello/xplatter
	cp $(IMPL_DIR)/dist/android/src/main/kotlin/hello/xplatter/HelloXplatter.kt lib/src/main/kotlin/hello/xplatter/
	@for abi in arm64-v8a armeabi-v7a x86_64 x86; do \
		mkdir -p lib/src/main/jniLibs/$$abi; \
		cp $(IMPL_DIR)/dist/android/src/main/jniLibs/$$abi/libhello_xplatter.so lib/src/main/jniLibs/$$abi/; \
	done
	@echo "Staged Android package from $(IMPL_DIR)/dist/android/"

# --- Build the Android app ---

HOST_OS := $(shell uname -s)

ifdef ANDROID_HOME
  ANDROID_SDK_DIR := $(ANDROID_HOME)
else ifdef ANDROID_SDK
  ANDROID_SDK_DIR := $(ANDROID_SDK)
else ifeq ($(HOST_OS),Darwin)
  ANDROID_SDK_DIR := $(HOME)/Library/Android/sdk
else
  ANDROID_SDK_DIR := $(HOME)/Android/Sdk
endif

local.properties:
	@echo "sdk.dir=$(ANDROID_SDK_DIR)" > $@

build-app: gradle-wrapper package local.properties
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

# --- Clean ---

clean:
	rm -rf $(BUILD_DIR)
	rm -rf lib/src/main/kotlin/hello/xplatter/HelloXplatter.kt
	rm -rf lib/src/main/jniLibs/*/libhello_xplatter.so
	rm -f local.properties
