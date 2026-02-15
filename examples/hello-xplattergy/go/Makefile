XPLATTERGY ?= ../../../xplattergy.sh
BUILD_DIR  := build
STAMP      := $(BUILD_DIR)/.generated

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif
SHARED_LIB := $(BUILD_DIR)/libhello_xplattergy.$(DYLIB_EXT)

.PHONY: run shared-lib clean

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

clean:
	rm -rf generated $(BUILD_DIR) hello_xplattergy.h
