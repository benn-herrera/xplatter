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
	b.WriteString("# ── Target filtering ──────────────────────────────────────────────────────────\n\n")
	b.WriteString("TARGETS ?= ios android web desktop\n\n")
	b.WriteString("target_enabled = $(filter $(1),$(TARGETS))\n\n")

	b.WriteString("# ── Platform detection ────────────────────────────────────────────────────────\n\n")
	b.WriteString("UNAME_S := $(shell uname -s)\n")
	b.WriteString("ifeq ($(UNAME_S),Darwin)\n")
	b.WriteString("  DYLIB_EXT := dylib\n")
	b.WriteString("else\n")
	b.WriteString("  DYLIB_EXT := so\n")
	b.WriteString("endif\n")
	b.WriteString("SHARED_LIB := $(BUILD_DIR)/$(LIB_NAME).$(DYLIB_EXT)\n\n")

	b.WriteString("# ── NDK configuration ─────────────────────────────────────────────────────────\n\n")
	b.WriteString("NDK_VERSION     ?= 29.0.14206865\n")
	b.WriteString("NDK             ?= $(HOME)/Library/Android/sdk/ndk/$(NDK_VERSION)\n")
	b.WriteString("NDK_BIN         := $(NDK)/toolchains/llvm/prebuilt/darwin-x86_64/bin\n")
	b.WriteString("ANDROID_MIN_API := 21\n\n")

	b.WriteString("# ── iOS ───────────────────────────────────────────────────────────────────────\n\n")
	b.WriteString("IOS_MIN := 15.0\n\n")

	b.WriteString("# ── Emscripten ────────────────────────────────────────────────────────────────\n\n")
	b.WriteString("EMCC ?= emcc\n\n")
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
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n")
	b.WriteString("# iOS: static libs per arch → lipo → xcframework + SPM package\n")
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n\n")
	b.WriteString("ifneq ($(call target_enabled,ios),)\n\n")

	// Emit arch-specific build rules (provided by caller)
	buildArchRule(b)

	// lipo simulator fat lib
	b.WriteString("$(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a: $(DIST_DIR)/ios/obj/ios-sim-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-x86_64/$(LIB_NAME).a\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tlipo -create $^ -output $@\n\n")

	// module.modulemap + header
	b.WriteString("$(DIST_DIR)/ios/headers/module.modulemap: $(GEN_HEADER)\n")
	b.WriteString("\t@mkdir -p $(DIST_DIR)/ios/headers\n")
	b.WriteString("\tcp $(GEN_HEADER) $(DIST_DIR)/ios/headers/\n")
	b.WriteString("\tprintf 'module C$(PASCAL_NAME) {\\n    header \"$(API_NAME).h\"\\n    export *\\n}\\n' > $@\n\n")

	// xcframework
	b.WriteString("$(DIST_DIR)/ios/$(PASCAL_NAME).xcframework: $(DIST_DIR)/ios/obj/ios-arm64/$(LIB_NAME).a $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a $(DIST_DIR)/ios/headers/module.modulemap\n")
	b.WriteString("\trm -rf $@\n")
	b.WriteString("\txcodebuild -create-xcframework \\\n")
	b.WriteString("\t\t-library $(DIST_DIR)/ios/obj/ios-arm64/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \\\n")
	b.WriteString("\t\t-library $(DIST_DIR)/ios/obj/ios-sim-fat/$(LIB_NAME).a -headers $(DIST_DIR)/ios/headers \\\n")
	b.WriteString("\t\t-output $@\n\n")

	// Swift binding with import
	b.WriteString("$(DIST_DIR)/ios/$(PASCAL_NAME)Lib/Sources/$(PASCAL_NAME)Binding/$(PASCAL_NAME).swift: $(GEN_SWIFT_BINDING)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tawk '{print} /^import Foundation$$/{print \"import C$(PASCAL_NAME)\"}' $(GEN_SWIFT_BINDING) > $@\n\n")

	// Package.swift
	b.WriteString("$(DIST_DIR)/ios/$(PASCAL_NAME)Lib/Package.swift: $(DIST_DIR)/ios/$(PASCAL_NAME).xcframework\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tprintf '// swift-tools-version: 5.9\\nimport PackageDescription\\n\\nlet package = Package(\\n    name: \"$(PASCAL_NAME)Lib\",\\n    platforms: [.iOS(.v15)],\\n    products: [\\n        .library(name: \"$(PASCAL_NAME)Lib\", targets: [\"$(PASCAL_NAME)Binding\"]),\\n    ],\\n    targets: [\\n        .binaryTarget(name: \"C$(PASCAL_NAME)\", path: \"../$(PASCAL_NAME).xcframework\"),\\n        .target(\\n            name: \"$(PASCAL_NAME)Binding\",\\n            dependencies: [\"C$(PASCAL_NAME)\"],\\n            path: \"Sources/$(PASCAL_NAME)Binding\"\\n        ),\\n    ]\\n)\\n' > $@\n\n")

	// package-ios target
	b.WriteString(".PHONY: package-ios\n")
	b.WriteString("package-ios: $(DIST_DIR)/ios/$(PASCAL_NAME).xcframework $(DIST_DIR)/ios/$(PASCAL_NAME)Lib/Package.swift $(DIST_DIR)/ios/$(PASCAL_NAME)Lib/Sources/$(PASCAL_NAME)Binding/$(PASCAL_NAME).swift\n")
	b.WriteString("\t@echo \"Packaged iOS: $(DIST_DIR)/ios/\"\n\n")

	b.WriteString("endif\n\n")
}

// MakefilePackageAndroid emits Android packaging rules with Gradle module structure.
func MakefilePackageAndroid(b *strings.Builder, buildABIRule func(b *strings.Builder)) {
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n")
	b.WriteString("# Android: native libs per ABI + Kotlin binding + Gradle module\n")
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n\n")
	b.WriteString("ifneq ($(call target_enabled,android),)\n\n")

	// Emit ABI-specific build rules (provided by caller)
	buildABIRule(b)

	b.WriteString("ANDROID_NATIVE_LIBS := \\\n")
	b.WriteString("\t$(DIST_DIR)/android/src/main/jniLibs/arm64-v8a/$(LIB_NAME).so \\\n")
	b.WriteString("\t$(DIST_DIR)/android/src/main/jniLibs/armeabi-v7a/$(LIB_NAME).so \\\n")
	b.WriteString("\t$(DIST_DIR)/android/src/main/jniLibs/x86_64/$(LIB_NAME).so \\\n")
	b.WriteString("\t$(DIST_DIR)/android/src/main/jniLibs/x86/$(LIB_NAME).so\n\n")

	// Kotlin binding into Gradle source set
	b.WriteString("ANDROID_KOTLIN_PKG := $(subst _,/,$(API_NAME))\n\n")

	b.WriteString("$(DIST_DIR)/android/src/main/kotlin/$(ANDROID_KOTLIN_PKG)/$(PASCAL_NAME).kt: $(GEN_KOTLIN_BINDING)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tcp $(GEN_KOTLIN_BINDING) $@\n\n")

	// build.gradle.kts
	b.WriteString("$(DIST_DIR)/android/build.gradle.kts:\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tprintf 'plugins {\\n    id(\"com.android.library\")\\n    id(\"org.jetbrains.kotlin.android\")\\n}\\n\\nandroid {\\n    namespace = \"$(subst _,.,$(API_NAME))\"\\n    compileSdk = 34\\n    defaultConfig {\\n        minSdk = $(ANDROID_MIN_API)\\n    }\\n}\\n' > $@\n\n")

	// AndroidManifest.xml
	b.WriteString("$(DIST_DIR)/android/src/main/AndroidManifest.xml:\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tprintf '<?xml version=\"1.0\" encoding=\"utf-8\"?>\\n<manifest />\\n' > $@\n\n")

	// package-android target
	b.WriteString(".PHONY: package-android\n")
	b.WriteString("package-android: $(ANDROID_NATIVE_LIBS) $(DIST_DIR)/android/src/main/kotlin/$(ANDROID_KOTLIN_PKG)/$(PASCAL_NAME).kt $(DIST_DIR)/android/build.gradle.kts $(DIST_DIR)/android/src/main/AndroidManifest.xml\n")
	b.WriteString("\t@echo \"Packaged Android: $(DIST_DIR)/android/\"\n\n")

	b.WriteString("endif\n\n")
}

// MakefilePackageWeb emits Web/WASM packaging rules with package.json.
func MakefilePackageWeb(b *strings.Builder, buildWASMRule func(b *strings.Builder)) {
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n")
	b.WriteString("# Web: WASM + JS binding + package.json\n")
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n\n")
	b.WriteString("ifneq ($(call target_enabled,web),)\n\n")

	// Emit WASM build rule (provided by caller)
	buildWASMRule(b)

	// JS binding
	b.WriteString("$(DIST_DIR)/web/$(API_NAME).js: $(GEN_JS_BINDING)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tcp $(GEN_JS_BINDING) $@\n\n")

	// package.json
	b.WriteString("$(DIST_DIR)/web/package.json:\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tprintf '{\\n  \"name\": \"$(API_NAME)\",\\n  \"version\": \"0.1.0\",\\n  \"type\": \"module\",\\n  \"main\": \"$(API_NAME).js\"\\n}\\n' > $@\n\n")

	// package-web target
	b.WriteString(".PHONY: package-web\n")
	b.WriteString("package-web: $(DIST_DIR)/web/$(API_NAME).wasm $(DIST_DIR)/web/$(API_NAME).js $(DIST_DIR)/web/package.json\n")
	b.WriteString("\t@echo \"Packaged Web: $(DIST_DIR)/web/\"\n\n")

	b.WriteString("endif\n\n")
}

// MakefilePackageDesktop emits Desktop packaging rules.
func MakefilePackageDesktop(b *strings.Builder) {
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n")
	b.WriteString("# Desktop: C header + Swift binding + shared library\n")
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n\n")
	b.WriteString("ifneq ($(call target_enabled,desktop),)\n\n")

	b.WriteString("$(DIST_DIR)/desktop/include/$(API_NAME).h: $(GEN_HEADER)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tcp $(GEN_HEADER) $@\n\n")

	b.WriteString("$(DIST_DIR)/desktop/include/$(PASCAL_NAME).swift: $(GEN_SWIFT_BINDING)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tcp $(GEN_SWIFT_BINDING) $@\n\n")

	b.WriteString("$(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT): $(SHARED_LIB)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tcp $(SHARED_LIB) $@\n\n")

	b.WriteString(".PHONY: package-desktop\n")
	b.WriteString("package-desktop: $(STAMP) $(DIST_DIR)/desktop/include/$(API_NAME).h $(DIST_DIR)/desktop/include/$(PASCAL_NAME).swift $(DIST_DIR)/desktop/lib/$(LIB_NAME).$(DYLIB_EXT)\n")
	b.WriteString("\t@echo \"Packaged Desktop: $(DIST_DIR)/desktop/\"\n\n")

	b.WriteString("endif\n\n")
}

// MakefileAggregateTargets emits the aggregate packages target and clean.
func MakefileAggregateTargets(b *strings.Builder) {
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n")
	b.WriteString("# Aggregate targets\n")
	b.WriteString("# ══════════════════════════════════════════════════════════════════════════════\n\n")

	b.WriteString("PACKAGE_TARGETS :=\n")
	b.WriteString("ifneq ($(call target_enabled,ios),)\n")
	b.WriteString("PACKAGE_TARGETS += package-ios\n")
	b.WriteString("endif\n")
	b.WriteString("ifneq ($(call target_enabled,android),)\n")
	b.WriteString("PACKAGE_TARGETS += package-android\n")
	b.WriteString("endif\n")
	b.WriteString("ifneq ($(call target_enabled,web),)\n")
	b.WriteString("PACKAGE_TARGETS += package-web\n")
	b.WriteString("endif\n")
	b.WriteString("ifneq ($(call target_enabled,desktop),)\n")
	b.WriteString("PACKAGE_TARGETS += package-desktop\n")
	b.WriteString("endif\n\n")

	b.WriteString(".PHONY: packages build\n")
	b.WriteString("packages: $(PACKAGE_TARGETS)\n")
	b.WriteString("build: packages\n")
}
