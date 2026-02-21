IMPL       ?= cpp
IMPL_DIR   := ../impl-$(IMPL)
BUILD_DIR  := build
TARGET     := $(BUILD_DIR)/hello_app
CXX        ?= c++

HEADER_DIR := $(IMPL_DIR)/dist/desktop/include
LIB_DIR    := $(IMPL_DIR)/dist/desktop/lib

.PHONY: build run test clean ensure-package

build: $(TARGET)

ensure-package:
	@test -f $(HEADER_DIR)/hello_xplatter.h || $(MAKE) -C $(IMPL_DIR) build

run: build
	./$(TARGET)

test: build
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: C++ desktop app" || (echo "FAIL: C++ desktop app"; exit 1)

HOST_OS := $(shell uname -s)

$(TARGET): main.cpp ensure-package
	@mkdir -p $(BUILD_DIR)
ifeq ($(HOST_OS),Darwin)
	$(CXX) -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-o $@ main.cpp \
		-L$(LIB_DIR) -lhello_xplatter \
		-Wl,-rpath,@executable_path/../../impl-$(IMPL)/dist/desktop/lib
else
	$(CXX) -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-o $@ main.cpp \
		-L$(LIB_DIR) -lhello_xplatter \
		-Wl,-rpath,'$$ORIGIN/../../impl-$(IMPL)/dist/desktop/lib'
endif

clean:
	rm -rf $(BUILD_DIR)
