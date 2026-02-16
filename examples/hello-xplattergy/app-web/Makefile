IMPL       ?= cpp
IMPL_DIR   := ../impl-$(IMPL)
BUILD_DIR  := build

.PHONY: test stage serve clean

# --- Ensure impl has built Web package ---

.PHONY: ensure-package
ensure-package:
	@test -f $(IMPL_DIR)/dist/web/hello_xplattergy.wasm || $(MAKE) -C $(IMPL_DIR) package-web

## stage: Copy WASM, JS binding, and HTML from package into build/
stage: ensure-package
	@mkdir -p $(BUILD_DIR)
	cp $(IMPL_DIR)/dist/web/hello_xplattergy.wasm $(BUILD_DIR)/hello_xplattergy.wasm
	cp $(IMPL_DIR)/dist/web/hello_xplattergy.js $(BUILD_DIR)/hello_xplattergy.js
	cp index.html $(BUILD_DIR)/index.html

## test: Build and verify all files exist
test: stage
	@test -f $(BUILD_DIR)/hello_xplattergy.wasm || (echo "FAIL: missing .wasm"; exit 1)
	@test -f $(BUILD_DIR)/hello_xplattergy.js   || (echo "FAIL: missing .js"; exit 1)
	@test -f $(BUILD_DIR)/index.html             || (echo "FAIL: missing index.html"; exit 1)
	@echo "PASS: Web app builds"

## serve: Build and start local HTTP server
serve: stage
	./serve.sh

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
