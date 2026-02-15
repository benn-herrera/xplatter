# xplattergy Makefile

SRC_DIR     := src
BIN_DIR     := bin
BINARY      := $(BIN_DIR)/xplattergy
MODULE_PATH := github.com/benn-herrera/xplattergy

VERSION     ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X $(MODULE_PATH)/cmd.Version=$(VERSION)

# Cross-compilation targets (for future `make dist`)
DIST_DIR    := dist
PLATFORMS   := windows/amd64 windows/arm64 darwin/arm64 linux/amd64
HOST_OS     := $(shell uname -s)

# the API implementation to bind when building test app (c, cpp, rust, or go)
TEST_APP_BOUND_IMPL ?= cpp

.PHONY: build test clean validate fmt vet lint dist help \
       test-impl-c test-impl-cpp test-impl-rust test-impl-go test-impls \
       test-app-desktop-cpp test-app-desktop-swift test-app-ios test-app-android test-app-web test-apps

## build: Build for the current platform (default)
build: $(BINARY)

$(BINARY):
	@mkdir -p $(BIN_DIR)
	cd $(SRC_DIR) && go build -ldflags "$(LDFLAGS)" -o ../$(BINARY) .
	@echo "Built: $(BINARY)"

## test: Run all tests
test:
	cd $(SRC_DIR) && go test ./...

## test-v: Run all tests with verbose output
test-v:
	cd $(SRC_DIR) && go test -v ./...

## clean: Remove build artifacts
clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)

## fmt: Format all Go source files
fmt:
	cd $(SRC_DIR) && go fmt ./...

## vet: Run go vet
vet:
	cd $(SRC_DIR) && go vet ./...

## validate: Build and run the validate command against the example API definition
validate: $(BINARY)
	$(BINARY) validate docs/example_api_definition.yaml

## dist: Build a batteries-included SDK archive for distribution
dist: DIST_NAME := xplattergy-$(VERSION)
dist: DIST_PKG  := $(DIST_DIR)/$(DIST_NAME)
dist:
	@mkdir -p $(DIST_PKG)/bin
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output="$(DIST_PKG)/bin/xplattergy-$${GOOS}-$${GOARCH}"; \
		if [ "$$GOOS" = "windows" ]; then output="$${output}.exe"; fi; \
		echo "Building $$output ..."; \
		cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH \
			go build -ldflags "$(LDFLAGS)" -o "../$$output" . && cd ..; \
	done
	@cp -r $(SRC_DIR) $(DIST_PKG)/$(SRC_DIR)
	@cp build_codegen.sh $(DIST_PKG)/
	@cp xplattergy.sh $(DIST_PKG)/
	@rsync -a --exclude='build/' --exclude='generated/' --exclude='target/' --exclude='hello_xplattergy.h' --exclude='Cargo.lock' examples $(DIST_PKG)/
	@cp -r schemas $(DIST_PKG)/schemas
	@cp -r docs $(DIST_PKG)/docs
	@cp LICENSE.md $(DIST_PKG)/
	@cp USER_README.md $(DIST_PKG)/README.md
	@tar -czf $(DIST_PKG).tar.gz -C $(DIST_DIR) $(DIST_NAME)
	@echo "SDK archive ready: $(DIST_PKG).tar.gz"

## test-impl-c: Build and run the C example
test-impl-c: build
	$(MAKE) -C examples/hello-xplattergy/c run

## test-impl-cpp: Build and run the C++ example
test-impl-cpp: build
	$(MAKE) -C examples/hello-xplattergy/cpp run

## test-impl-rust: Build and run the Rust example
test-impl-rust: build
	cd examples/hello-xplattergy/rust && cargo test

## test-impl-go: Build and run the Go example
test-impl-go: build
	$(MAKE) -C examples/hello-xplattergy/go run

## test-impls: Run all examples
test-impls: test-impl-c test-impl-cpp test-impl-rust test-impl-go

## test-app-desktop-cpp: Build and test the C++ desktop app
test-app-desktop-cpp: build
	$(MAKE) -C examples/hello-xplattergy/app-desktop-cpp IMPL=$(TEST_APP_BOUND_IMPL) test

## test-app-desktop-swift: Build and test the Swift desktop app (macOS only)
test-app-desktop-swift: build
	[[ $(HOST_OS) == Darwin ]] && $(MAKE) -C examples/hello-xplattergy/app-desktop-swift IMPL=$(TEST_APP_BOUND_IMPL) test

## test-app-ios: Build and test the iOS app (simulator)
test-app-ios: build
	[[ $(HOST_OS) == Darwin ]] && $(MAKE) -C examples/hello-xplattergy/app-ios IMPL=$(TEST_APP_BOUND_IMPL) test

## test-app-android: Build and test the Android app
test-app-android: build
	$(MAKE) -C examples/hello-xplattergy/app-android IMPL=$(TEST_APP_BOUND_IMPL) test

## test-app-web: Build and test the Web/WASM app (requires emcc)
test-app-web: build
	$(MAKE) -C examples/hello-xplattergy/app-web IMPL=$(TEST_APP_BOUND_IMPL) test

## test-apps: Run all app tests
test-apps: test-app-desktop-cpp test-app-desktop-swift test-app-ios test-app-android test-app-web

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
