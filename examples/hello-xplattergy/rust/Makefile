BUILD_DIR := build

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif
SHARED_LIB := $(BUILD_DIR)/libhello_xplattergy.$(DYLIB_EXT)

.PHONY: shared-lib clean

shared-lib: $(SHARED_LIB)

$(SHARED_LIB):
	cargo build --release
	@mkdir -p $(BUILD_DIR)
	cp target/release/libhello_xplattergy.$(DYLIB_EXT) $(SHARED_LIB)
ifeq ($(UNAME_S),Darwin)
	install_name_tool -id @rpath/libhello_xplattergy.$(DYLIB_EXT) $(SHARED_LIB)
endif

clean:
	cargo clean
	rm -rf $(BUILD_DIR)
