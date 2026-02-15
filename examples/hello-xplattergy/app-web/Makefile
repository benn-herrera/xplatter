XPLATTERGY ?= ../../../xplattergy.sh
IMPL       ?= cpp
IMPL_DIR   := ../$(IMPL)
BUILD_DIR  := build

# Use generated files from impl dir if available, otherwise generate our own.
GEN_DIR = $(if $(wildcard $(IMPL_DIR)/generated/hello_xplattergy.h),$(IMPL_DIR)/generated,$(BUILD_DIR)/generated)

# Source files
IMPL_CXX_SRC   := $(IMPL_DIR)/hello_xplattergy_impl.cpp
SHIM_CXX_SRC    = $(GEN_DIR)/hello_xplattergy_shim.cpp
PLATFORM_C_SRC  := platform_services_web.c
JS_BINDING       = $(GEN_DIR)/hello_xplattergy.js

# Emscripten compiler (must be on PATH via emsdk)
EMCC ?= emcc

# Compiler flags
CXXFLAGS := -std=c++20 -O2 -fvisibility=hidden -DHELLO_XPLATTERGY_BUILD
CFLAGS   := -std=c11 -O2 -fvisibility=hidden -DHELLO_XPLATTERGY_BUILD

.PHONY: wasm stage test serve clean

## wasm: Compile C++ implementation to WASM via Emscripten
wasm: $(BUILD_DIR)/hello_xplattergy.wasm

$(BUILD_DIR)/impl.o: $(IMPL_CXX_SRC) $(GEN_DIR)/hello_xplattergy.h
	@mkdir -p $(BUILD_DIR)
	$(EMCC) $(CXXFLAGS) -I$(IMPL_DIR) -I$(GEN_DIR) -c -o $@ $<

$(BUILD_DIR)/shim.o: $(SHIM_CXX_SRC) $(GEN_DIR)/hello_xplattergy.h
	@mkdir -p $(BUILD_DIR)
	$(EMCC) $(CXXFLAGS) -I$(GEN_DIR) -c -o $@ $<

$(BUILD_DIR)/platform.o: $(PLATFORM_C_SRC)
	@mkdir -p $(BUILD_DIR)
	$(EMCC) $(CFLAGS) -c -o $@ $<

$(BUILD_DIR)/hello_xplattergy.wasm: $(BUILD_DIR)/impl.o $(BUILD_DIR)/shim.o $(BUILD_DIR)/platform.o
	$(EMCC) -o $@ $^ \
		--no-entry \
		-s 'EXPORTED_FUNCTIONS=["_malloc","_free","_hello_xplattergy_lifecycle_create_greeter","_hello_xplattergy_lifecycle_destroy_greeter","_hello_xplattergy_greeter_say_hello"]' \
		-s STANDALONE_WASM \
		-O2

## stage: Copy WASM, JS binding, and HTML into build/
stage: wasm $(JS_BINDING)
	cp $(JS_BINDING) $(BUILD_DIR)/hello_xplattergy.js
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

# Codegen fallback (if impl dir has no generated files)
$(BUILD_DIR)/generated/hello_xplattergy.h: ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)/generated
	$(XPLATTERGY) generate --impl-lang $(IMPL) --clean -o $(BUILD_DIR)/generated $<

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
