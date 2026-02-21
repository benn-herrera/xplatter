package gen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/benn-herrera/xplatter/model"
)

// ComputeWASMExports returns the WASM export function names as a JSON array string
// for Emscripten's -s EXPORTED_FUNCTIONS. Includes _malloc, _free, and all C ABI function names.
func ComputeWASMExports(apiName string, api *model.APIDefinition) string {
	exports := []string{`"_malloc"`, `"_free"`}
	for _, iface := range api.Interfaces {
		for _, ctor := range iface.Constructors {
			fn := "_" + CABIFunctionName(apiName, iface.Name, ctor.Name)
			exports = append(exports, `"`+fn+`"`)
		}
		if handleName, ok := iface.ConstructorHandleName(); ok {
			fn := "_" + CABIFunctionName(apiName, iface.Name, DestructorMethodName(handleName))
			exports = append(exports, `"`+fn+`"`)
		}
		for _, method := range iface.Methods {
			fn := "_" + CABIFunctionName(apiName, iface.Name, method.Name)
			exports = append(exports, `"`+fn+`"`)
		}
	}
	return "[" + strings.Join(exports, ",") + "]"
}

// APIDefRelPath computes the relative path from the project root to the API definition file.
// The Makefile is a ProjectFile, so it lives at filepath.Dir(outputDir), not in the output dir itself.
func APIDefRelPath(ctx *Context) string {
	base := filepath.Dir(ctx.OutputDir)
	rel, err := filepath.Rel(base, ctx.APIDefPath)
	if err != nil {
		return ctx.APIDefPath
	}
	return rel
}

// MakefileHeader emits the auto-generated banner and core variables.
func MakefileHeader(b *strings.Builder, ctx *Context, implLang string) {
	apiName := ctx.API.API.Name
	pascalName := ToPascalCase(apiName)
	buildMacro := BuildMacroName(apiName)

	b.WriteString(GeneratedFileHeader(ctx, "#", true))
	b.WriteString("\n")

	fmt.Fprintf(b, "SHELL := /bin/bash\n")
	fmt.Fprintf(b, "XPLATTER ?= xplatter\n")
	fmt.Fprintf(b, "API_DEF  := %s\n", APIDefRelPath(ctx))
	fmt.Fprintf(b, "IMPL_LANG := %s\n\n", implLang)

	fmt.Fprintf(b, "API_NAME  := %s\n", apiName)
	fmt.Fprintf(b, "LIB_NAME  := lib%s\n", apiName)
	fmt.Fprintf(b, "PASCAL_NAME := %s\n", pascalName)
	fmt.Fprintf(b, "BUILD_MACRO := %s\n", buildMacro)
	fmt.Fprintf(b, "BUILD_DIR := build\n")
	fmt.Fprintf(b, "DIST_DIR  := dist\n")
	fmt.Fprintf(b, "STAMP     := $(BUILD_DIR)/.generated\n\n")
}

// MakefileTargetConfig emits target filtering, NDK, iOS, and platform detection.
func MakefileTargetConfig(b *strings.Builder) {
	b.WriteString(`# ── Target filtering ──────────────────────────────────────────────────────────

TARGETS ?= ios android web desktop

target_enabled = $(filter $(1),$(TARGETS))

# ── Host platform detection ────────────────────────────────────────────────────

HOST_OS   := $(shell uname -s)
HOST_ARCH := $(shell uname -m)
EXE       :=
ifeq ($(HOST_OS),Darwin)
  DYLIB_EXT     := dylib
  NDK_HOST_OS   := darwin
  NDK_HOST_ARCH := x86_64
else ifneq (,$(findstring MINGW,$(HOST_OS))$(findstring MSYS,$(HOST_OS)))
  DYLIB_EXT     := dll
  NDK_HOST_OS   := windows
  NDK_HOST_ARCH := x86_64
  EXE           := .exe
else
  DYLIB_EXT    := so
  NDK_HOST_OS  := linux
  ifeq ($(HOST_ARCH),aarch64)
    NDK_HOST_ARCH := aarch64
  else
    NDK_HOST_ARCH := x86_64
  endif
endif
SHARED_LIB := $(BUILD_DIR)/$(LIB_NAME).$(DYLIB_EXT)

# ── MSVC discovery (Windows only) ─────────────────────────────────────────────
# If cl.exe is already on PATH (e.g. Developer Command Prompt), this is skipped.
# Otherwise, uses vswhere.exe to locate Visual Studio and sets up paths.
# Override MSVC_DIR to point to a custom MSVC toolset directory.

ifneq (,$(EXE))
  PROGRAMFILES_X86 ?= $(shell cmd //C "echo %ProgramFiles(x86)%" 2>/dev/null | tr -d '\r')
  VSWHERE := $(PROGRAMFILES_X86)/Microsoft Visual Studio/Installer/vswhere.exe
  ifeq (,$(shell which cl.exe 2>/dev/null))
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
endif

# ── NDK configuration ─────────────────────────────────────────────────────────

NDK_VERSION     ?= 29.0.14206865
ifdef ANDROID_NDK
  NDK           ?= $(ANDROID_NDK)
else ifdef ANDROID_SDK
  NDK           ?= $(ANDROID_SDK)/ndk/$(NDK_VERSION)
else ifeq ($(HOST_OS),Darwin)
  NDK           ?= $(HOME)/Library/Android/sdk/ndk/$(NDK_VERSION)
else ifneq (,$(findstring windows,$(NDK_HOST_OS)))
  NDK           ?= $(LOCALAPPDATA)/Android/Sdk/ndk/$(NDK_VERSION)
else
  NDK           ?= $(HOME)/Android/Sdk/ndk/$(NDK_VERSION)
endif
NDK_BIN         := $(NDK)/toolchains/llvm/prebuilt/$(NDK_HOST_OS)-$(NDK_HOST_ARCH)/bin
ANDROID_MIN_API := 21

# ── iOS ───────────────────────────────────────────────────────────────────────

IOS_MIN := 15.0

# ── Emscripten ────────────────────────────────────────────────────────────────

EMCC ?= emcc

`)
}

// MakefileBindingVars emits variables for generated binding file paths.
func MakefileBindingVars(b *strings.Builder, apiName, genPrefix string) {
	pascalName := ToPascalCase(apiName)
	b.WriteString("# ── Generated binding files ───────────────────────────────────────────────────\n\n")
	fmt.Fprintf(b, "GEN_DIR            := %s\n", genPrefix)
	fmt.Fprintf(b, "GEN_HEADER         := $(GEN_DIR)$(API_NAME).h\n")
	fmt.Fprintf(b, "GEN_SWIFT_BINDING  := $(GEN_DIR)%s.swift\n", pascalName)
	fmt.Fprintf(b, "GEN_KOTLIN_BINDING := $(GEN_DIR)%s.kt\n", pascalName)
	fmt.Fprintf(b, "GEN_JS_BINDING     := $(GEN_DIR)$(API_NAME).js\n")
	fmt.Fprintf(b, "GEN_JNI_SOURCE     := $(GEN_DIR)$(API_NAME)_jni.c\n\n")
}

// MakefileWASMExports emits the WASM_EXPORTS variable.
func MakefileWASMExports(b *strings.Builder, apiName string, api *model.APIDefinition) {
	b.WriteString("# ── WASM exports (computed from API definition) ──────────────────────────────\n\n")
	fmt.Fprintf(b, "WASM_EXPORTS := %s\n\n", ComputeWASMExports(apiName, api))
}

// MakefileCodegenStamp emits the STAMP rule that reruns xplatter generate.
func MakefileCodegenStamp(b *strings.Builder, implLang, outputFlag string) {
	b.WriteString("# ── Codegen ──────────────────────────────────────────────────────────────────\n\n")
	fmt.Fprintf(b, "$(STAMP): $(API_DEF)\n")
	fmt.Fprintf(b, "\t@mkdir -p $(BUILD_DIR)\n")
	fmt.Fprintf(b, "\t$(XPLATTER) generate --impl-lang %s %s $(API_DEF)\n", implLang, outputFlag)
	fmt.Fprintf(b, "\t@touch $@\n\n")
}

// MakefilePackageIOS emits iOS packaging rules: static libs → lipo → xcframework → SPM.
func MakefilePackageIOS(b *strings.Builder, buildArchRule func(b *strings.Builder)) {
	b.WriteString(`# ══════════════════════════════════════════════════════════════════════════════
# iOS: static libs per arch → lipo → xcframework + SPM package
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,ios),)
ifeq ($(HOST_OS),Darwin)

`)
	buildArchRule(b)

	b.WriteString(`$(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a: $(DIST_DIR)/ios/obj/ios-sim-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-x86_64/$(LIB_NAME).a
	@mkdir -p $(dir $@)
	lipo -create $^ -output $@

$(DIST_DIR)/ios/headers/module.modulemap: $(GEN_HEADER)
	@mkdir -p $(DIST_DIR)/ios/headers
	cp $(GEN_HEADER) $(DIST_DIR)/ios/headers/
	printf 'module C$(PASCAL_NAME) {\n    header "$(API_NAME).h"\n    export *\n}\n' > $@

$(DIST_DIR)/ios/$(PASCAL_NAME).xcframework: $(DIST_DIR)/ios/obj/ios-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a $(DIST_DIR)/ios/headers/module.modulemap
	rm -rf $@
	xcodebuild -create-xcframework \
		-library $(DIST_DIR)/ios/obj/ios-arm64/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
		-library $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \
		-output $@

$(DIST_DIR)/ios/$(PASCAL_NAME)Lib/Sources/$(PASCAL_NAME)Binding/$(PASCAL_NAME).swift: $(GEN_SWIFT_BINDING)
	@mkdir -p $(dir $@)
	awk '{print} /^import Foundation$$/{print "import C$(PASCAL_NAME)"}' $(GEN_SWIFT_BINDING) > $@

$(DIST_DIR)/ios/$(PASCAL_NAME)Lib/Package.swift: $(DIST_DIR)/ios/$(PASCAL_NAME).xcframework
	@mkdir -p $(dir $@)
	printf '// swift-tools-version: 5.9\nimport PackageDescription\n\nlet package = Package(\n    name: "$(PASCAL_NAME)Lib",\n    platforms: [.iOS(.v15)],\n    products: [\n        .library(name: "$(PASCAL_NAME)Lib", targets: ["$(PASCAL_NAME)Binding"]),\n    ],\n    targets: [\n        .binaryTarget(name: "C$(PASCAL_NAME)", path: "../$(PASCAL_NAME).xcframework"),\n        .target(\n            name: "$(PASCAL_NAME)Binding",\n            dependencies: ["C$(PASCAL_NAME)"],\n            path: "Sources/$(PASCAL_NAME)Binding"\n        ),\n    ]\n)\n' > $@

.PHONY: package-ios
package-ios: $(DIST_DIR)/ios/$(PASCAL_NAME).xcframework $(DIST_DIR)/ios/$(PASCAL_NAME)Lib/Package.swift $(DIST_DIR)/ios/$(PASCAL_NAME)Lib/Sources/$(PASCAL_NAME)Binding/$(PASCAL_NAME).swift
	@echo "Packaged iOS: $(DIST_DIR)/ios/"

else

.PHONY: package-ios
package-ios:
	@echo "skipping iOS packaging on $(HOST_OS)"

endif
endif

`)
}

// MakefilePackageAndroid emits Android packaging rules with Gradle module structure.
func MakefilePackageAndroid(b *strings.Builder, buildABIRule func(b *strings.Builder)) {
	b.WriteString(`# ══════════════════════════════════════════════════════════════════════════════
# Android: native libs per ABI + Kotlin binding + Gradle module
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,android),)

`)
	buildABIRule(b)

	b.WriteString(`ANDROID_NATIVE_LIBS := \
	$(DIST_DIR)/android/src/main/jniLibs/arm64-v8a/$(LIB_NAME).so \
	$(DIST_DIR)/android/src/main/jniLibs/armeabi-v7a/$(LIB_NAME).so \
	$(DIST_DIR)/android/src/main/jniLibs/x86_64/$(LIB_NAME).so \
	$(DIST_DIR)/android/src/main/jniLibs/x86/$(LIB_NAME).so

ANDROID_KOTLIN_PKG := $(subst _,/,$(API_NAME))

$(DIST_DIR)/android/src/main/kotlin/$(ANDROID_KOTLIN_PKG)/$(PASCAL_NAME).kt: $(GEN_KOTLIN_BINDING)
	@mkdir -p $(dir $@)
	cp $(GEN_KOTLIN_BINDING) $@

$(DIST_DIR)/android/build.gradle.kts:
	@mkdir -p $(dir $@)
	printf 'plugins {\n    id("com.android.library")\n    id("org.jetbrains.kotlin.android")\n}\n\nandroid {\n    namespace = "$(subst _,.,$(API_NAME))"\n    compileSdk = 34\n    defaultConfig {\n        minSdk = $(ANDROID_MIN_API)\n    }\n}\n' > $@

$(DIST_DIR)/android/src/main/AndroidManifest.xml:
	@mkdir -p $(dir $@)
	printf '<?xml version="1.0" encoding="utf-8"?>\n<manifest />\n' > $@

.PHONY: package-android
package-android: $(ANDROID_NATIVE_LIBS) $(DIST_DIR)/android/src/main/kotlin/$(ANDROID_KOTLIN_PKG)/$(PASCAL_NAME).kt $(DIST_DIR)/android/build.gradle.kts $(DIST_DIR)/android/src/main/AndroidManifest.xml
	@echo "Packaged Android: $(DIST_DIR)/android/"

endif

`)
}

// MakefilePackageWeb emits Web/WASM packaging rules with package.json.
func MakefilePackageWeb(b *strings.Builder, buildWASMRule func(b *strings.Builder)) {
	b.WriteString(`# ══════════════════════════════════════════════════════════════════════════════
# Web: WASM + JS binding + package.json
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,web),)

`)
	buildWASMRule(b)

	b.WriteString(`$(DIST_DIR)/web/$(API_NAME).js: $(GEN_JS_BINDING)
	@mkdir -p $(dir $@)
	cp $(GEN_JS_BINDING) $@

$(DIST_DIR)/web/package.json:
	@mkdir -p $(dir $@)
	printf '{\n  "name": "$(API_NAME)",\n  "version": "0.1.0",\n  "type": "module",\n  "main": "$(API_NAME).js"\n}\n' > $@

.PHONY: package-web
package-web: $(DIST_DIR)/web/$(API_NAME).wasm $(DIST_DIR)/web/$(API_NAME).js $(DIST_DIR)/web/package.json
	@echo "Packaged Web: $(DIST_DIR)/web/"

endif

`)
}

// MakefilePackageDesktop emits Desktop packaging rules.
func MakefilePackageDesktop(b *strings.Builder) {
	b.WriteString(`# ══════════════════════════════════════════════════════════════════════════════
# Desktop: C header + Swift binding + shared library
# ══════════════════════════════════════════════════════════════════════════════

ifneq ($(call target_enabled,desktop),)

$(DIST_DIR)/desktop/include/$(API_NAME).h: $(GEN_HEADER)
	@mkdir -p $(dir $@)
	cp $(GEN_HEADER) $@

$(DIST_DIR)/desktop/include/$(PASCAL_NAME).swift: $(GEN_SWIFT_BINDING)
	@mkdir -p $(dir $@)
	cp $(GEN_SWIFT_BINDING) $@

$(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT): $(SHARED_LIB)
	@mkdir -p $(dir $@)
	cp $(SHARED_LIB) $@

.PHONY: package-desktop
package-desktop: $(STAMP) $(DIST_DIR)/desktop/include/$(API_NAME).h $(DIST_DIR)/desktop/include/$(PASCAL_NAME).swift $(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT)
	@echo "Packaged Desktop: $(DIST_DIR)/desktop/"

endif

`)
}

// MakefileAggregateTargets emits the aggregate packages target and clean.
func MakefileAggregateTargets(b *strings.Builder) {
	b.WriteString(`# ══════════════════════════════════════════════════════════════════════════════
# Aggregate targets
# ══════════════════════════════════════════════════════════════════════════════

PACKAGE_TARGETS :=
ifneq ($(call target_enabled,ios),)
PACKAGE_TARGETS += package-ios
endif
ifneq ($(call target_enabled,android),)
PACKAGE_TARGETS += package-android
endif
ifneq ($(call target_enabled,web),)
PACKAGE_TARGETS += package-web
endif
ifneq ($(call target_enabled,desktop),)
PACKAGE_TARGETS += package-desktop
endif

.PHONY: packages build
packages: $(PACKAGE_TARGETS)
build: packages
`)
}
