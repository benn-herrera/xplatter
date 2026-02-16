IMPL       ?= cpp
IMPL_DIR   := ../impl-$(IMPL)
BUILD_DIR  := build
TARGET     := $(BUILD_DIR)/hello_app
CXX        ?= c++

HEADER_DIR := $(IMPL_DIR)/dist/desktop/include
LIB_DIR    := $(IMPL_DIR)/dist/desktop/lib

.PHONY: run test clean ensure-package

ensure-package:
	@test -f $(HEADER_DIR)/hello_xplattergy.h || $(MAKE) -C $(IMPL_DIR) package-desktop

run: $(TARGET)
	./$(TARGET)

test: $(TARGET)
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: C++ desktop app" || (echo "FAIL: C++ desktop app"; exit 1)

$(TARGET): main.cpp ensure-package
	@mkdir -p $(BUILD_DIR)
	$(CXX) -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-L$(LIB_DIR) -lhello_xplattergy \
		-Wl,-rpath,@executable_path/../../impl-$(IMPL)/dist/desktop/lib \
		-o $@ main.cpp

clean:
	rm -rf $(BUILD_DIR)
