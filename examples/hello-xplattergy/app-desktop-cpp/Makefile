XPLATTERGY ?= ../../../xplattergy.sh
CXX        ?= c++
IMPL       ?= cpp
IMPL_DIR   := ../$(IMPL)
BUILD_DIR  := build
TARGET     := $(BUILD_DIR)/hello_app

# Use packaged header from impl's dist, or fall back to generated dir / local codegen
HEADER_DIR  = $(if $(wildcard $(IMPL_DIR)/dist/desktop/include/hello_xplattergy.h),$(IMPL_DIR)/dist/desktop/include,$(if $(wildcard $(IMPL_DIR)/generated/hello_xplattergy.h),$(IMPL_DIR)/generated,$(BUILD_DIR)/generated))
HEADER      = $(HEADER_DIR)/hello_xplattergy.h

# Use packaged library from impl's dist, or fall back to build dir
LIB_DIR     = $(if $(wildcard $(IMPL_DIR)/dist/desktop/lib/libhello_xplattergy.*),$(IMPL_DIR)/dist/desktop/lib,$(IMPL_DIR)/build)

.PHONY: run test clean

run: $(TARGET)
	./$(TARGET)

test: $(TARGET)
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: C++ desktop app" || (echo "FAIL: C++ desktop app"; exit 1)

$(TARGET): main.cpp $(HEADER) $(LIB_DIR)/libhello_xplattergy.dylib
	@mkdir -p $(BUILD_DIR)
	$(CXX) -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-L$(LIB_DIR) -lhello_xplattergy \
		-Wl,-rpath,@executable_path/../../$(IMPL)/dist/desktop/lib \
		-Wl,-rpath,@executable_path/../../$(IMPL)/build \
		-o $@ main.cpp

$(BUILD_DIR)/generated/hello_xplattergy.h: ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)/generated
	$(XPLATTERGY) generate --impl-lang $(IMPL) -o $(BUILD_DIR)/generated $<

$(LIB_DIR)/libhello_xplattergy.dylib:
	$(MAKE) -C $(IMPL_DIR) shared-lib

clean:
	rm -rf $(BUILD_DIR)
