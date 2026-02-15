XPLATTERGY ?= ../../../xplattergy.sh
IMPL       ?= cpp
IMPL_DIR   := ../$(IMPL)
BUILD_DIR  := build
TARGET     := $(BUILD_DIR)/hello_app

# The C header and Swift binding are impl-independent. Use from the impl's
# generated dir if available, otherwise generate our own copy.
GEN_DIR      = $(if $(wildcard $(IMPL_DIR)/generated/hello_xplattergy.h),$(IMPL_DIR)/generated,$(BUILD_DIR)/generated)
BRIDGING_HDR = $(GEN_DIR)/hello_xplattergy.h
SWIFT_BINDING = $(GEN_DIR)/HelloXplattergy.swift

.PHONY: run test clean

run: $(TARGET)
	./$(TARGET)

test: $(TARGET)
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: Swift desktop app" || (echo "FAIL: Swift desktop app"; exit 1)

$(TARGET): main.swift $(SWIFT_BINDING) $(IMPL_DIR)/build/libhello_xplattergy.dylib
	@mkdir -p $(BUILD_DIR)
	swiftc -import-objc-header $(BRIDGING_HDR) \
		-L$(IMPL_DIR)/build -lhello_xplattergy \
		-Xlinker -rpath -Xlinker @executable_path/../../$(IMPL)/build \
		-o $@ main.swift $(SWIFT_BINDING)

$(BUILD_DIR)/generated/hello_xplattergy.h $(BUILD_DIR)/generated/HelloXplattergy.swift: ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)/generated
	$(XPLATTERGY) generate --impl-lang $(IMPL) -o $(BUILD_DIR)/generated $<

$(IMPL_DIR)/build/libhello_xplattergy.dylib:
	$(MAKE) -C $(IMPL_DIR) shared-lib

clean:
	rm -rf $(BUILD_DIR)
