IMPL       ?= cpp
IMPL_DIR   := ../impl-$(IMPL)
BUILD_DIR  := build

HEADER_DIR := $(IMPL_DIR)/dist/desktop/include
LIB_DIR    := $(IMPL_DIR)/dist/desktop/lib

HOST_OS := $(shell uname -s)
EXE     :=
ifneq (,$(findstring MINGW,$(HOST_OS))$(findstring MSYS,$(HOST_OS)))
  EXE := .exe
endif

# ── MSVC discovery (Windows only) ─────────────────────────────────────────────

ifneq (,$(EXE))
  ifeq (,$(shell which cl.exe 2>/dev/null))
    PROGRAMFILES_X86 ?= $(shell cmd //C "echo %ProgramFiles(x86)%" 2>/dev/null | tr -d '\r')
    VSWHERE := $(PROGRAMFILES_X86)/Microsoft Visual Studio/Installer/vswhere.exe
    ifndef MSVC_DIR
      VS_PATH := $(shell "$(VSWHERE)" -latest -products '*' \
          -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 \
          -property installationPath -format value 2>/dev/null | tr -d '\r')
      MSVC_VER := $(shell cat "$(VS_PATH)/VC/Auxiliary/Build/Microsoft.VCToolsVersion.default.txt" 2>/dev/null | tr -d '\r')
      MSVC_DIR := $(VS_PATH)/VC/Tools/MSVC/$(MSVC_VER)
    endif
    MSVC_BIN  := $(MSVC_DIR)/bin/Hostx64/x64
    export PATH := $(MSVC_BIN);$(PATH)

    WIN_SDK_ROOT ?= $(PROGRAMFILES_X86)/Windows Kits/10
    WIN_SDK_VER  ?= $(shell ls "$(WIN_SDK_ROOT)/Include" 2>/dev/null | sort -V | tail -1)

    export INCLUDE := $(MSVC_DIR)/include;$(WIN_SDK_ROOT)/Include/$(WIN_SDK_VER)/ucrt;$(WIN_SDK_ROOT)/Include/$(WIN_SDK_VER)/shared;$(WIN_SDK_ROOT)/Include/$(WIN_SDK_VER)/um
    export LIB := $(MSVC_DIR)/lib/x64;$(WIN_SDK_ROOT)/Lib/$(WIN_SDK_VER)/ucrt/x64;$(WIN_SDK_ROOT)/Lib/$(WIN_SDK_VER)/um/x64
  endif
  CXX := cl
else
  CXX ?= c++
endif

TARGET := $(BUILD_DIR)/hello_app$(EXE)

.PHONY: build run test clean ensure-package

build: $(TARGET)

ensure-package:
	@test -f $(HEADER_DIR)/hello_xplatter.h || $(MAKE) -C $(IMPL_DIR) build

run: build
	./$(TARGET)

test: build
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: C++ desktop app" || (echo "FAIL: C++ desktop app"; exit 1)

$(TARGET): main.cpp ensure-package
	@mkdir -p $(BUILD_DIR)
ifeq ($(HOST_OS),Darwin)
	$(CXX) -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-o $@ main.cpp \
		-L$(LIB_DIR) -lhello_xplatter \
		-Wl,-rpath,@executable_path/../../impl-$(IMPL)/dist/desktop/lib
else ifneq (,$(EXE))
	$(CXX) /std:c++20 /EHsc /W4 /I$(HEADER_DIR) \
		/Fe:$@ main.cpp \
		$(LIB_DIR)/hello_xplatter.lib
	cp $(LIB_DIR)/libhello_xplatter.dll $(BUILD_DIR)/
else
	$(CXX) -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-o $@ main.cpp \
		-L$(LIB_DIR) -lhello_xplatter \
		-Wl,-rpath,'$$ORIGIN/../../impl-$(IMPL)/dist/desktop/lib'
endif

clean:
	rm -rf $(BUILD_DIR)
