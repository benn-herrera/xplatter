# examples/Makefile — build and test example implementations and apps

ROOT_DIR := ..
HOST_OS  := $(shell uname -s)
BUILD_TARGET  := $(shell [[ -d $(ROOT_DIR)/.git ]] && echo build_for_dev || echo build_in_dist)

# the API implementation to bind when building test apps (c, cpp, rust, or go)
IMPL ?= cpp

.PHONY: build build_in_dist build_for_dev \
       test-hello-impl-c test-hello-impl-cpp test-hello-impl-rust test-hello-impl-go test-hello-impls \
       test-hello-app-desktop-cpp test-hello-app-desktop-swift test-hello-app-ios test-hello-app-android test-hello-app-web test-hello-apps \
       package-hello-cpp package-hello-all \
       help

## build: Ensure a valid xplattergy binary is available
build_in_dist:
	($(ROOT_DIR)/xplattergy.sh version 1>&2) 2> /dev/null || $(ROOT_DIR)/build_codegen.sh

build_for_dev:
	$(MAKE) -C $(ROOT_DIR) build

build: $(BUILD_TARGET)

## test-hello-impl-c: Build and run the hello example C implementation
test-hello-impl-c: build
	$(MAKE) -C hello-xplattergy/c run

## test-hello-impl-cpp: Build and run the hello example C++ implementation
test-hello-impl-cpp: build
	$(MAKE) -C hello-xplattergy/cpp run

## test-hello-impl-rust: Build and run the hello example Rust implementation
test-hello-impl-rust: build
	cd hello-xplattergy/rust && cargo test

## test-hello-impl-go: Build and run the hello example Go implementation
test-hello-impl-go: build
	$(MAKE) -C hello-xplattergy/go run

## test-hello-impls: Run all hello example implementations
test-hello-impls: test-hello-impl-c test-hello-impl-cpp test-hello-impl-rust test-hello-impl-go

## test-hello-app-desktop-cpp: Build and test the hello C++ desktop app
test-hello-app-desktop-cpp: build
	$(MAKE) -C hello-xplattergy/app-desktop-cpp IMPL=$(IMPL) test

## test-hello-app-desktop-swift: Build and test the hello Swift desktop app (macOS only)
test-hello-app-desktop-swift: build
	[[ $(HOST_OS) != Darwin ]] || $(MAKE) -C hello-xplattergy/app-desktop-swift IMPL=$(IMPL) test
	@[[ $(HOST_OS) == Darwin ]] || echo $@ skipped on $(HOST_OS)

## test-hello-app-ios: Build and test the hello iOS app (simulator, macOS only)
test-hello-app-ios: build
	@[[ $(HOST_OS) != Darwin ]] || $(MAKE) -C hello-xplattergy/app-ios IMPL=$(IMPL) test
	@[[ $(HOST_OS) == Darwin ]] || echo $@ skipped on $(HOST_OS)

## test-hello-app-android: Build and test the hello Android app
test-hello-app-android: build
	$(MAKE) -C hello-xplattergy/app-android IMPL=$(IMPL) test

## test-hello-app-web: Build and test the hello Web/WASM app (requires emcc)
test-hello-app-web: build
	$(MAKE) -C hello-xplattergy/app-web IMPL=$(IMPL) test

## test-hello-apps: Run all app tests
test-hello-apps: test-hello-app-desktop-cpp test-hello-app-desktop-swift test-hello-app-ios test-hello-app-android test-hello-app-web

## package-hello-cpp: Build cross-platform packages for C++ impl
package-hello-cpp: build
	$(MAKE) -C hello-xplattergy/cpp packages

## package-hello-all: Build packages for all implementations
package-hello-all: package-hello-cpp

## help: Show available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
