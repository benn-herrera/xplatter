IMPL       ?= cpp
IMPL_DIR   := ../impl-$(IMPL)
BUILD_DIR  := build

.PHONY: build ensure_package test stage serve clean

# --- Ensure impl has built Web package ---

build: stage
ensure-package:
	@test -f $(IMPL_DIR)/dist/web/hello_xplatter.wasm || $(MAKE) -C $(IMPL_DIR) build

## stage: Copy WASM, JS binding, and HTML from package into build/
stage: ensure-package
	@mkdir -p $(BUILD_DIR)
	cp $(IMPL_DIR)/dist/web/hello_xplatter.wasm $(BUILD_DIR)/hello_xplatter.wasm
	cp $(IMPL_DIR)/dist/web/hello_xplatter.js $(BUILD_DIR)/hello_xplatter.js
	cp index.html $(BUILD_DIR)/index.html

## test: Build and verify all files exist
test: build
	@test -f $(BUILD_DIR)/hello_xplatter.wasm || (echo "FAIL: missing .wasm"; exit 1)
	@test -f $(BUILD_DIR)/hello_xplatter.js   || (echo "FAIL: missing .js"; exit 1)
	@test -f $(BUILD_DIR)/index.html            || (echo "FAIL: missing index.html"; exit 1)
	@echo "PASS: Web app builds"

## serve: Build and start local HTTP server
serve: build
	./serve.sh

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
