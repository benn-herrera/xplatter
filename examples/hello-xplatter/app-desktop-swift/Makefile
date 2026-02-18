IMPL       ?= cpp
IMPL_DIR   := ../impl-$(IMPL)
BUILD_DIR  := build
TARGET     := $(BUILD_DIR)/hello_app

BRIDGING_HDR  := $(IMPL_DIR)/dist/desktop/include/hello_xplatter.h
SWIFT_BINDING := $(IMPL_DIR)/dist/desktop/include/HelloXplatter.swift
LIB_DIR       := $(IMPL_DIR)/dist/desktop/lib

.PHONY: build run test clean ensure-package

build: $(TARGET)

ensure-package:
	@test -f $(BRIDGING_HDR) || $(MAKE) -C $(IMPL_DIR) build

run: build
	./$(TARGET)

test: build
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: Swift desktop app" || (echo "FAIL: Swift desktop app"; exit 1)

$(TARGET): main.swift ensure-package
	@mkdir -p $(BUILD_DIR)
	swiftc -import-objc-header $(BRIDGING_HDR) \
		-L$(LIB_DIR) -lhello_xplatter \
		-Xlinker -rpath -Xlinker @executable_path/../../impl-$(IMPL)/dist/desktop/lib \
		-o $@ main.swift $(SWIFT_BINDING)

clean:
	rm -rf $(BUILD_DIR)
