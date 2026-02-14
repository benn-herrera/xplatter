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

.PHONY: build test clean validate fmt vet lint dist help

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

## dist: Cross-compile for all distribution targets
dist:
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output="$(DIST_DIR)/xplattergy-$${GOOS}-$${GOARCH}"; \
		if [ "$$GOOS" = "windows" ]; then output="$${output}.exe"; fi; \
		echo "Building $$output ..."; \
		cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH \
			go build -ldflags "$(LDFLAGS)" -o "../$$output" . && cd ..; \
	done
	@echo "Distribution binaries in $(DIST_DIR)/"

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
