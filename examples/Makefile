# examples/Makefile — build and test example implementations and apps

ROOT_DIR := ..
HOST_OS  := $(shell uname -s)

# the API implementation to bind when building test apps (c, cpp, rust, or go)
IMPL ?= cpp

.PHONY: build \
       test-impl-c test-impl-cpp test-impl-rust test-impl-go test-impls \
       test-app-desktop-cpp test-app-desktop-swift test-app-ios test-app-android test-app-web test-apps \
       help

## build: Ensure the xplattergy binary is built
build:
	$(MAKE) -C $(ROOT_DIR) build

## test-impl-c: Build and run the C example
test-impl-c: build
	$(MAKE) -C hello-xplattergy/c run

## test-impl-cpp: Build and run the C++ example
test-impl-cpp: build
	$(MAKE) -C hello-xplattergy/cpp run

## test-impl-rust: Build and run the Rust example
test-impl-rust: build
	cd hello-xplattergy/rust && cargo test

## test-impl-go: Build and run the Go example
test-impl-go: build
	$(MAKE) -C hello-xplattergy/go run

## test-impls: Run all implementation examples
test-impls: test-impl-c test-impl-cpp test-impl-rust test-impl-go

## test-app-desktop-cpp: Build and test the C++ desktop app
test-app-desktop-cpp: build
	$(MAKE) -C hello-xplattergy/app-desktop-cpp IMPL=$(IMPL) test

## test-app-desktop-swift: Build and test the Swift desktop app (macOS only)
test-app-desktop-swift: build
	[[ $(HOST_OS) != Darwin ]] || $(MAKE) -C hello-xplattergy/app-desktop-swift IMPL=$(IMPL) test
	@[[ $(HOST_OS) == Darwin ]] || echo $@ skipped on $(HOST_OS)

## test-app-ios: Build and test the iOS app (simulator, macOS only)
test-app-ios: build
	@[[ $(HOST_OS) != Darwin ]] || $(MAKE) -C hello-xplattergy/app-ios IMPL=$(IMPL) test
	@[[ $(HOST_OS) == Darwin ]] || echo $@ skipped on $(HOST_OS)

## test-app-android: Build and test the Android app
test-app-android: build
	$(MAKE) -C hello-xplattergy/app-android IMPL=$(IMPL) test

## test-app-web: Build and test the Web/WASM app (requires emcc)
test-app-web: build
	$(MAKE) -C hello-xplattergy/app-web IMPL=$(IMPL) test

## test-apps: Run all app tests
test-apps: test-app-desktop-cpp test-app-desktop-swift test-app-ios test-app-android test-app-web

## help: Show available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
