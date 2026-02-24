SHELL      := /bin/bash
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
# If cl.exe is already on PATH (e.g. Developer Command Prompt), nothing to do.
# Otherwise, locate vcvarsall.bat via vswhere and re-execute make inside the
# MSVC environment it establishes. The second invocation finds cl.exe on PATH
# and skips this block entirely.

ifneq (,$(EXE)) # IS_WINDOWS
  ifeq (,$(shell which cl.exe 2>/dev/null))
    PROGRAMFILES_X86 ?= $(shell cmd //C "echo %ProgramFiles(x86)%" 2>/dev/null | tr -d '\r')
    _VSWHERE   := $(PROGRAMFILES_X86)/Microsoft Visual Studio/Installer/vswhere.exe
    _VS_PATH   := $(shell "$(_VSWHERE)" -latest -products '*' \
        -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 \
        -property installationPath -format value 2>/dev/null | tr -d '\r' | sed 's:\\\\:/:g')
    ifeq (,$(_VS_PATH))
      $(error Visual Studio with MSVC not found. Install VS with the "Desktop development with C++" workload.)
    endif
    _VCVARSALL := $(shell echo "$(_VS_PATH)/VC/Auxiliary/Build/vcvarsall.bat")
    _TMP_CMD := _msvc_setup_$(shell date '+%s').cmd
    _MSVC_BOOSTRAPPED := 1
    .PHONY: _msvc_bootstrap
    _msvc_bootstrap:
		@printf '@call "%s" x64 >nul 2>&1\r\n@"$(MAKE)" $(MAKECMDGOALS)\r\n' "$(_VCVARSALL)" > $(_TMP_CMD)
		@.\\$(_TMP_CMD)
		@rm -f $(_TMP_CMD)
    %: _msvc_bootstrap ;
  endif # cl.exe
endif # IS_WINDOWS

.PHONY: build ensure-package run test clean

ifdef _MSVC_BOOSTRAPPED

# this instance does not have correct env and kicked off a nested build - targets are no-ops
build: ;
ensure-package: ;
run: ;
test: ;
clean: ;

else # _MSVC_BOOSTRAPPED

# this instance has has correct env (may be a nested build call post-boostrap)
TARGET := $(BUILD_DIR)/hello_app$(EXE)

build: $(TARGET)

ensure-package:
	@test -f $(HEADER_DIR)/hello_xplatter.h || $(MAKE) -C $(IMPL_DIR) build

run: build
	./$(TARGET)

test: build
	@echo Handled by ready to go make
	@printf 'World\nexit\n' | ./$(TARGET) | grep -q 'Hello, World!' && echo "PASS: C++ desktop app" || (echo "FAIL: C++ desktop app"; exit 1)

$(TARGET): main.cpp ensure-package
	@mkdir -p $(BUILD_DIR)
ifeq ($(HOST_OS),Darwin)
	clang -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-o $@ main.cpp \
		-L$(LIB_DIR) -lhello_xplatter \
		-Wl,-rpath,@executable_path/../../impl-$(IMPL)/dist/desktop/lib
else ifneq (,$(EXE))
	cl.exe /nologo /std:c++20 /EHsc /W4 /I$(HEADER_DIR) /Fo:$(BUILD_DIR)/ \
		/Fe:$@ main.cpp \
		$(LIB_DIR)/hello_xplatter.lib
	cp $(LIB_DIR)/hello_xplatter.dll $(BUILD_DIR)/
else
	clang -std=c++20 -Wall -Wextra -I$(HEADER_DIR) \
		-o $@ main.cpp \
		-L$(LIB_DIR) -lhello_xplatter \
		-Wl,-rpath,'$$ORIGIN/../../impl-$(IMPL)/dist/desktop/lib'
endif

clean:
	rm -rf $(BUILD_DIR)

endif # _MSVC_BOOSTRAPPED
