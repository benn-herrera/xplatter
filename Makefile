# xplatter Makefile

SRC_DIR     := src
BIN_DIR     := bin
BINARY      := $(BIN_DIR)/xplatter
MODULE_PATH := github.com/benn-herrera/xplatter

VERSION     ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X $(MODULE_PATH)/cmd.Version=$(VERSION)

# Cross-compilation targets (for future `make dist`)
DIST_DIR    := dist
PLATFORMS   := windows/amd64 windows/arm64 darwin/arm64 linux/amd64
HOST_OS     := $(shell uname -s)
TEST_DIST   ?= false

.PHONY: build test clean validate fmt vet lint dist help

## build: Build for the current platform (default)
build: $(BINARY)

$(BINARY):
	@mkdir -p $(BIN_DIR)
	cd $(SRC_DIR) && go build -ldflags "$(LDFLAGS)" -o ../$(BINARY) .
	@ln -sfn ../schemas $(BIN_DIR)/schemas
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

## test-examples-hello-impl-app-matrix: build all hello example api impls and test all apps against all api impls
test-examples-hello-impl-app-matrix:
	$(MAKE) -C examples test-examples-hello-impl-app-matrix


## validate: Build and run the validate command against the example API definition
validate: $(BINARY)
	$(BINARY) validate docs/example_api_definition.yaml

## dist: Build a batteries-included SDK archive for distribution
dist: DIST_NAME := xplatter-$(VERSION)
dist: DIST_PKG  := $(DIST_DIR)/$(DIST_NAME)
dist:
	@mkdir -p $(DIST_PKG)/bin
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output="$(DIST_PKG)/bin/xplatter-$${GOOS}-$${GOARCH}"; \
		if [ "$$GOOS" = "windows" ]; then output="$${output}.exe"; fi; \
		echo "Building $$output ..."; \
		cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH \
			go build -ldflags "$(LDFLAGS)" -o "../$$output" . && cd ..; \
	done
	@cp -r $(SRC_DIR) $(DIST_PKG)/$(SRC_DIR)
	@cp build_codegen.sh $(DIST_PKG)/
	@cp xplatter.sh $(DIST_PKG)/
	@rsync -a --exclude='build/' --exclude='generated/' --exclude='dist/' --exclude='target/' --exclude='hello_xplatter.h' --exclude='Cargo.lock' examples $(DIST_PKG)/
	@cp -r schemas $(DIST_PKG)/bin/schemas
	@cp -r docs $(DIST_PKG)/docs
	@cp LICENSE.md $(DIST_PKG)/
	@cp USER_README.md $(DIST_PKG)/README.md
	@tar -czf $(DIST_PKG).tar.gz -C $(DIST_DIR) $(DIST_NAME)
	@echo "SDK archive ready: $(DIST_PKG).tar.gz"
	@if $(TEST_DIST); then \
		echo "Verifying distribution..."; \
		$(MAKE) -C $(DIST_PKG)/examples test-examples-hello-impl-app-matrix; \
	fi

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
