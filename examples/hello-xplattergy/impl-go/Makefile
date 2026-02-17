XPLATTERGY ?= ../../../xplattergy.sh
BUILD_DIR  := build
DIST_DIR   := dist
STAMP      := $(BUILD_DIR)/.generated

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif
SHARED_LIB := $(BUILD_DIR)/libhello_xplattergy.$(DYLIB_EXT)

API_NAME := hello_xplattergy
LIB_NAME := libhello_xplattergy

.PHONY: run shared-lib package-desktop package-web clean

run: $(STAMP)
	go build -o $(BUILD_DIR)/hello_xplattergy .
	./$(BUILD_DIR)/hello_xplattergy

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(STAMP)
	go build -buildmode=c-shared -ldflags='-extldflags "-Wl,-install_name,@rpath/libhello_xplattergy.$(DYLIB_EXT)"' -o $(SHARED_LIB) .

# Generate and copy C header locally (cgo needs headers in the package directory)
$(STAMP): ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTERGY) generate --impl-lang go --clean -o generated $<
	cp generated/hello_xplattergy.h .
	@touch $@

$(DIST_DIR)/desktop/include/$(API_NAME).h: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/$(API_NAME).h $@

$(DIST_DIR)/desktop/include/HelloXplattergy.swift: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/HelloXplattergy.swift $@

$(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT): $(SHARED_LIB)
	@mkdir -p $(dir $@)
	cp $(SHARED_LIB) $@

package-desktop: $(STAMP) $(DIST_DIR)/desktop/include/$(API_NAME).h $(DIST_DIR)/desktop/include/HelloXplattergy.swift $(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT)
	@echo "Packaged Desktop: $(DIST_DIR)/desktop/"

$(DIST_DIR)/web/hello_xplattergy.wasm: $(STAMP)
	@mkdir -p $(dir $@)
	GOOS=wasip1 GOARCH=wasm go build -o $@ .

$(DIST_DIR)/web/hello_xplattergy.js: $(STAMP)
	@mkdir -p $(dir $@)
	cp generated/hello_xplattergy.js $@

package-web: $(DIST_DIR)/web/hello_xplattergy.wasm $(DIST_DIR)/web/hello_xplattergy.js
	@echo "Packaged Web: $(DIST_DIR)/web/"

clean:
	rm -rf generated $(BUILD_DIR) $(DIST_DIR) hello_xplattergy.h
