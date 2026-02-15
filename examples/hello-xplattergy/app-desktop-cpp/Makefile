XPLATTERGY ?= ../../../bin/xplattergy
CXX        ?= c++
IMPL       ?= cpp
IMPL_DIR   := ../$(IMPL)
BUILD_DIR  := build
TARGET     := $(BUILD_DIR)/hello_app

# The C header is impl-independent. Use it from the impl's generated dir if
# available, otherwise generate our own copy from the shared YAML.
HEADER_DIR  = $(if $(wildcard $(IMPL_DIR)/generated/hello_xplattergy.h),$(IMPL_DIR)/generated,$(BUILD_DIR)/generated)
HEADER      = $(HEADER_DIR)/hello_xplattergy.h

.PHONY: run test clean

run: $(TARGET)
	./$(TARGET)

test: $(TARGET)
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: C++ desktop app" || (echo "FAIL: C++ desktop app"; exit 1)

$(TARGET): main.cpp $(HEADER) $(IMPL_DIR)/build/libhello_xplattergy.dylib
	@mkdir -p $(BUILD_DIR)
	$(CXX) -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-L$(IMPL_DIR)/build -lhello_xplattergy \
		-Wl,-rpath,@executable_path/../../$(IMPL)/build \
		-o $@ main.cpp

$(BUILD_DIR)/generated/hello_xplattergy.h: ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)/generated
	$(XPLATTERGY) generate --impl-lang $(IMPL) -o $(BUILD_DIR)/generated $<

$(IMPL_DIR)/build/libhello_xplattergy.dylib:
	$(MAKE) -C $(IMPL_DIR) shared-lib

clean:
	rm -rf $(BUILD_DIR)
