XPLATTERGY ?= ../../../xplattergy.sh
IMPL       ?= cpp
IMPL_DIR   := ../$(IMPL)
BUILD_DIR  := build
TARGET     := $(BUILD_DIR)/hello_app

# Use packaged files from impl's dist, or fall back to generated dir / local codegen
GEN_DIR      = $(if $(wildcard $(IMPL_DIR)/dist/desktop/include/hello_xplattergy.h),$(IMPL_DIR)/dist/desktop/include,$(if $(wildcard $(IMPL_DIR)/generated/hello_xplattergy.h),$(IMPL_DIR)/generated,$(BUILD_DIR)/generated))
BRIDGING_HDR = $(GEN_DIR)/hello_xplattergy.h

# Swift binding: prefer generated dir (has the .swift file), fall back to local codegen
SWIFT_GEN_DIR = $(if $(wildcard $(IMPL_DIR)/generated/HelloXplattergy.swift),$(IMPL_DIR)/generated,$(BUILD_DIR)/generated)
SWIFT_BINDING = $(SWIFT_GEN_DIR)/HelloXplattergy.swift

# Use packaged library from impl's dist, or fall back to build dir
LIB_DIR      = $(if $(wildcard $(IMPL_DIR)/dist/desktop/lib/libhello_xplattergy.*),$(IMPL_DIR)/dist/desktop/lib,$(IMPL_DIR)/build)

.PHONY: run test clean

run: $(TARGET)
	./$(TARGET)

test: $(TARGET)
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: Swift desktop app" || (echo "FAIL: Swift desktop app"; exit 1)

$(TARGET): main.swift $(SWIFT_BINDING) $(LIB_DIR)/libhello_xplattergy.dylib
	@mkdir -p $(BUILD_DIR)
	swiftc -import-objc-header $(BRIDGING_HDR) \
		-L$(LIB_DIR) -lhello_xplattergy \
		-Xlinker -rpath -Xlinker @executable_path/../../$(IMPL)/dist/desktop/lib \
		-Xlinker -rpath -Xlinker @executable_path/../../$(IMPL)/build \
		-o $@ main.swift $(SWIFT_BINDING)

$(BUILD_DIR)/generated/hello_xplattergy.h $(BUILD_DIR)/generated/HelloXplattergy.swift: ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)/generated
	$(XPLATTERGY) generate --impl-lang $(IMPL) -o $(BUILD_DIR)/generated $<

$(LIB_DIR)/libhello_xplattergy.dylib:
	$(MAKE) -C $(IMPL_DIR) shared-lib

clean:
	rm -rf $(BUILD_DIR)
