# examples/Makefile — build and test example implementations and apps

SHELL := /bin/bash
ROOT_DIR := ..
HOST_OS  := $(shell uname -s)
BUILD_XPLATTER_TARGET  := $(shell [[ -d $(ROOT_DIR)/.git ]] && echo build-xplatter-for-dev || echo build-xplatter-in-dist)

# the API implementation to bind when building test apps (c, cpp, rust, or go)
IMPL ?= cpp

# Absolute paths for codegen bootstrap (so cd into impl dirs works)
XPLATTER_ABS := $(abspath $(ROOT_DIR)/xplatter.sh)
HELLO_API_ABS := $(abspath hello-xplatter/shared_specs/hello_xplatter.yaml)

# Bootstrap codegen: generates Makefile + platform_services + bindings.
# Scaffold files (Makefile, platform_services, impl stubs) are only written on first run.
# $(1) = impl dir, $(2) = impl lang, $(3) = output flag (e.g. "-o generated" or "-o .")
define gen-impl
	cd $(1) && $(XPLATTER_ABS) generate --impl-lang $(2) $(3) $(HELLO_API_ABS)
endef

.PHONY: default build-xplatter build-xplatter-in-dist build-xplatter-for-dev \
       test-hello-impl-c test-hello-impl-cpp test-hello-impl-rust test-hello-impl-go test-hello-impls \
       test-hello-app-desktop-cpp test-hello-app-desktop-swift test-hello-app-ios test-hello-app-android test-hello-app-web test-hello-apps \
       package-hello-c package-hello-cpp package-hello-go package-hello-rust package-hello-all \
       help

default: test-hello-apps

## build: Ensure a valid xplatter binary is available
build-xplatter-in-dist:
	($(ROOT_DIR)/xplatter.sh version 1>&2) 2> /dev/null || $(ROOT_DIR)/build_codegen.sh

build-xplatter-for-dev:
	$(MAKE) -C $(ROOT_DIR) build

build-xplatter: $(BUILD_XPLATTER_TARGET)

## test-hello-impl-c: Build and run the hello example C implementation
test-hello-impl-c: build-xplatter
	$(call gen-impl,hello-xplatter/impl-c,c,-o generated)
	$(MAKE) -C hello-xplatter/impl-c XPLATTER=$(XPLATTER_ABS) test

## test-hello-impl-cpp: Build and run the hello example C++ implementation
test-hello-impl-cpp: build-xplatter
	$(call gen-impl,hello-xplatter/impl-cpp,cpp,-o generated)
	$(MAKE) -C hello-xplatter/impl-cpp XPLATTER=$(XPLATTER_ABS) test

## test-hello-impl-rust: Build and run the hello example Rust implementation
test-hello-impl-rust: build-xplatter
	$(call gen-impl,hello-xplatter/impl-rust,rust,-o generated)
	$(MAKE) -C hello-xplatter/impl-rust XPLATTER=$(XPLATTER_ABS) test

## test-hello-impl-go: Build and run the hello example Go implementation
test-hello-impl-go: build-xplatter
	$(call gen-impl,hello-xplatter/impl-go,go,-o generated)
	$(MAKE) -C hello-xplatter/impl-go XPLATTER=$(XPLATTER_ABS) test

## test-hello-impls: Run all hello example implementations
test-hello-impls: test-hello-impl-c test-hello-impl-cpp test-hello-impl-rust test-hello-impl-go

## test-hello-app-desktop-cpp: Build and test the hello C++ desktop app
test-hello-app-desktop-cpp: build-xplatter
	$(MAKE) -C hello-xplatter/app-desktop-cpp IMPL=$(IMPL) test

## test-hello-app-desktop-swift: Build and test the hello Swift desktop app (macOS only)
test-hello-app-desktop-swift: build-xplatter
	[[ $(HOST_OS) != Darwin ]] || $(MAKE) -C hello-xplatter/app-desktop-swift IMPL=$(IMPL) test
	@[[ $(HOST_OS) == Darwin ]] || echo $@ skipped on $(HOST_OS)

## test-hello-app-ios: Build and test the hello iOS app (simulator, macOS only)
test-hello-app-ios: build-xplatter
	@[[ $(HOST_OS) != Darwin ]] || $(MAKE) -C hello-xplatter/app-ios IMPL=$(IMPL) test
	@[[ $(HOST_OS) == Darwin ]] || echo $@ skipped on $(HOST_OS)

## test-hello-app-android: Build and test the hello Android app
test-hello-app-android: build-xplatter
	$(MAKE) -C hello-xplatter/app-android IMPL=$(IMPL) test

## test-hello-app-web: Build and test the hello Web/WASM app (requires emcc)
test-hello-app-web: build-xplatter
	$(MAKE) -C hello-xplatter/app-web IMPL=$(IMPL) test

## test-hello-apps: Run all app tests
test-hello-apps: test-hello-app-desktop-cpp test-hello-app-desktop-swift test-hello-app-ios test-hello-app-android test-hello-app-web
clean-hello-apps:
	$(MAKE) -C hello-xplatter/app-desktop-cpp clean
	$(MAKE) -C hello-xplatter/app-desktop-swift clean
	$(MAKE) -C hello-xplatter/app-ios clean
	$(MAKE) -C hello-xplatter/app-android clean
	$(MAKE) -C hello-xplatter/app-web clean

## package-hello-c: Build cross-platform packages for C impl
package-hello-c: build-xplatter
	$(call gen-impl,hello-xplatter/impl-c,c,-o generated)
	$(MAKE) -C hello-xplatter/impl-c XPLATTER=$(XPLATTER_ABS) packages

## package-hello-cpp: Build cross-platform packages for C++ impl
package-hello-cpp: build-xplatter
	$(call gen-impl,hello-xplatter/impl-cpp,cpp,-o generated)
	$(MAKE) -C hello-xplatter/impl-cpp XPLATTER=$(XPLATTER_ABS) packages

## package-hello-go: Build cross-platform packages for Go impl
package-hello-go: build-xplatter
	$(call gen-impl,hello-xplatter/impl-go,go,-o generated)
	$(MAKE) -C hello-xplatter/impl-go XPLATTER=$(XPLATTER_ABS) packages

## package-hello-rust: Build cross-platform packages for Rust impl
package-hello-rust: build-xplatter
	$(call gen-impl,hello-xplatter/impl-rust,rust,-o generated)
	$(MAKE) -C hello-xplatter/impl-rust XPLATTER=$(XPLATTER_ABS) packages

## package-hello-all: Build packages for all implementations
package-hello-all: package-hello-c package-hello-cpp package-hello-rust package-hello-go

## test-examples-hello-impl-app-matrix: build all hello example api impls and test all apps against all api impls
test-hello-impl-app-matrix: test-hello-impls
	$(MAKE) IMPL=c    clean-hello-apps test-hello-apps
	$(MAKE) IMPL=cpp  clean-hello-apps test-hello-apps
	$(MAKE) IMPL=go   clean-hello-apps test-hello-apps
	$(MAKE) IMPL=rust clean-hello-apps test-hello-apps

## help: Show available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
