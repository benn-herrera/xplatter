XPLATTERGY ?= ../../../xplattergy.sh
CC         ?= cc
CFLAGS     := -Wall -Wextra -std=c11 -I. -Igenerated

SRCS      := greeter_impl.c ../platform_services/desktop.c main.c
LIB_SRCS  := greeter_impl.c ../platform_services/desktop.c
BUILD_DIR := build
STAMP     := $(BUILD_DIR)/.generated
TARGET    := $(BUILD_DIR)/hello_xplattergy

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  DYLIB_EXT := dylib
else
  DYLIB_EXT := so
endif
SHARED_LIB := $(BUILD_DIR)/libhello_xplattergy.$(DYLIB_EXT)

.PHONY: run shared-lib clean

run: $(TARGET)
	./$(TARGET)

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(LIB_SRCS) $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CC) $(CFLAGS) -shared -fPIC -fvisibility=hidden -DHELLO_XPLATTERGY_BUILD \
		-Wl,-install_name,@rpath/libhello_xplattergy.$(DYLIB_EXT) -o $@ $(LIB_SRCS)

$(TARGET): $(SRCS) $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CC) $(CFLAGS) -o $@ $(SRCS)

$(STAMP): ../shared_schemas/hello_xplattergy.yaml
	@mkdir -p $(BUILD_DIR)
	$(XPLATTERGY) generate --impl-lang c --clean -o generated $<
	@touch $@

clean:
	rm -rf generated $(BUILD_DIR) dist

# ══════════════════════════════════════════════════════════════════════════════
# Cross-platform packaging
# ══════════════════════════════════════════════════════════════════════════════

API_NAME          := hello_xplattergy
DIST_DIR          := dist
PLATFORM_SERVICES := ../platform_services

# Variables consumed by cross-compile-cc.mk
COMPILER       := clang
COMPILER_FLAGS := -std=c11 -Wall -Wextra
IMPL_SOURCES   := greeter_impl.c
SHIM_SOURCE    :=
JNI_SOURCE     := generated/hello_xplattergy_jni.c
IMPL_INCLUDES  := -I. -Igenerated
GEN_INCLUDES   := -Igenerated
GEN_HEADER     := generated/hello_xplattergy.h

# Generated binding files (produced by codegen)
GEN_SWIFT_BINDING  := generated/HelloXplattergy.swift
GEN_KOTLIN_BINDING := generated/HelloXplattergy.kt
GEN_JS_BINDING     := generated/hello_xplattergy.js

# Ensure codegen runs before any cross-compile target needs generated files
generated/hello_xplattergy.h: $(STAMP)

include ../cross-compile-cc.mk
