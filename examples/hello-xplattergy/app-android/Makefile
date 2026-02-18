XPLATTERGY ?= ../../../xplattergy.sh
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
	@test -d $(IMPL_DIR)/dist/android/jniLibs || $(MAKE) -C $(IMPL_DIR) build

# --- Copy packaged native libs + Kotlin binding into Gradle project ---

package: ensure-package
	@mkdir -p lib/src/main/kotlin/hello/xplattergy
	cp $(IMPL_DIR)/dist/android/HelloXplattergy.kt lib/src/main/kotlin/hello/xplattergy/
	@for abi in arm64-v8a armeabi-v7a x86_64 x86; do \
		mkdir -p lib/src/main/jniLibs/$$abi; \
		cp $(IMPL_DIR)/dist/android/jniLibs/$$abi/libhello_xplattergy.so lib/src/main/jniLibs/$$abi/; \
	done
	@echo "Staged Android package from $(IMPL_DIR)/dist/android/"

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

# --- Clean ---

clean:
	rm -rf $(BUILD_DIR)
	rm -rf lib/src/main/kotlin/hello/xplattergy/HelloXplattergy.kt
	rm -rf lib/src/main/jniLibs/*/libhello_xplattergy.so
