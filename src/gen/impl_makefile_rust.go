package gen

import (
	"fmt"
	"strings"
)

func init() {
	Register("impl_makefile_rust", func() Generator { return &RustMakefileGenerator{} })
}

// RustMakefileGenerator produces a scaffold Makefile for Rust implementations.
// Handles local build, cross-compilation for all targets, and packaging.
type RustMakefileGenerator struct{}

func (g *RustMakefileGenerator) Name() string { return "impl_makefile_rust" }

func (g *RustMakefileGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	apiName := ctx.API.API.Name

	var b strings.Builder

	MakefileHeader(&b, ctx, "rust")
	MakefileTargetConfig(&b)
	MakefileBindingVars(&b, apiName, "generated/")
	MakefileWASMExports(&b, apiName, ctx.API)

	b.WriteString("# Ensure codegen runs before any target needs generated files\n")
	b.WriteString("$(GEN_HEADER): $(STAMP)\n\n")

	b.WriteString("LIB_C_FLAGS := -std=c11 -Wall -Wextra -fvisibility=hidden -D$(BUILD_MACRO)\n\n")

	// Codegen stamp
	MakefileCodegenStamp(&b, "rust", "-o generated")

	// Phony declarations
	b.WriteString(".PHONY: run shared-lib clean\n\n")

	// Local build
	b.WriteString("run: $(STAMP)\n")
	b.WriteString("\tcargo test\n\n")

	// Shared library
	b.WriteString("shared-lib: $(SHARED_LIB)\n\n")
	b.WriteString("$(SHARED_LIB): $(STAMP)\n")
	b.WriteString("\tcargo build --release\n")
	b.WriteString("\t@mkdir -p $(BUILD_DIR)\n")
	b.WriteString("\tcp target/release/$(LIB_NAME).$(DYLIB_EXT) $(SHARED_LIB)\n")
	b.WriteString("ifeq ($(UNAME_S),Darwin)\n")
	b.WriteString("\tinstall_name_tool -id @rpath/$(LIB_NAME).$(DYLIB_EXT) $(SHARED_LIB)\n")
	b.WriteString("endif\n\n")

	// Clean
	b.WriteString("clean:\n")
	b.WriteString("\tcargo clean\n")
	b.WriteString("\trm -rf generated $(BUILD_DIR) $(DIST_DIR) flatbuffers\n\n")

	// iOS packaging
	MakefilePackageIOS(&b, func(b *strings.Builder) {
		g.writeIOSArchRules(b)
	})

	// Android packaging
	MakefilePackageAndroid(&b, func(b *strings.Builder) {
		g.writeAndroidABIRules(b)
	})

	// Web packaging
	MakefilePackageWeb(&b, func(b *strings.Builder) {
		g.writeWASMBuildRule(b)
	})

	// Desktop packaging
	MakefilePackageDesktop(&b)

	// Aggregate
	MakefileAggregateTargets(&b)

	return []*OutputFile{
		{Path: "Makefile", Content: []byte(b.String()), Scaffold: true, ProjectFile: true},
	}, nil
}

func (g *RustMakefileGenerator) writeIOSArchRules(b *strings.Builder) {
	// Macro for building one iOS arch via cargo
	b.WriteString("# $(1) = arch dir name, $(2) = Rust target triple\n")
	b.WriteString("define BUILD_IOS_ARCH\n\n")
	b.WriteString("$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(STAMP)\n")
	b.WriteString("\t@mkdir -p $$(dir $$@)\n")
	b.WriteString("\tcargo build --release --target $(2)\n")
	b.WriteString("\tcp target/$(2)/release/$(LIB_NAME).a $$@\n\n")
	b.WriteString("endef\n\n")

	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-arm64,aarch64-apple-ios))\n")
	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,aarch64-apple-ios-sim))\n")
	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,x86_64-apple-ios))\n\n")
}

func (g *RustMakefileGenerator) writeAndroidABIRules(b *strings.Builder) {
	// Macro for building one Android ABI via cargo + NDK link
	b.WriteString("# $(1) = ABI name, $(2) = Rust target triple, $(3) = NDK target prefix\n")
	b.WriteString("define BUILD_ANDROID_ABI\n\n")
	b.WriteString("$(DIST_DIR)/android/src/main/jniLibs/$(1)/$(LIB_NAME).so: $(STAMP)\n")
	b.WriteString("\t@mkdir -p $(DIST_DIR)/android/obj/$(1) $$(dir $$@)\n")
	b.WriteString("\tPATH=$(NDK_BIN):$$$$PATH cargo build --release --target $(2)\n")
	b.WriteString("\t$(NDK_BIN)/$(3)-clang $(LIB_C_FLAGS) -fPIC \\\n")
	b.WriteString("\t\t-Igenerated -c -o $(DIST_DIR)/android/obj/$(1)/jni.o $(GEN_JNI_SOURCE)\n")
	b.WriteString("\t$(NDK_BIN)/$(3)-clang -shared \\\n")
	b.WriteString("\t\t-Wl,--whole-archive target/$(2)/release/$(LIB_NAME).a -Wl,--no-whole-archive \\\n")
	b.WriteString("\t\t$(DIST_DIR)/android/obj/$(1)/jni.o \\\n")
	b.WriteString("\t\t-ldl -lm -llog \\\n")
	b.WriteString("\t\t-o $$@\n\n")
	b.WriteString("endef\n\n")

	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,aarch64-linux-android,aarch64-linux-android$(ANDROID_MIN_API)))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,armv7-linux-androideabi,armv7a-linux-androideabi$(ANDROID_MIN_API)))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,x86_64,x86_64-linux-android,x86_64-linux-android$(ANDROID_MIN_API)))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,x86,i686-linux-android,i686-linux-android$(ANDROID_MIN_API)))\n\n")
}

func (g *RustMakefileGenerator) writeWASMBuildRule(b *strings.Builder) {
	b.WriteString("$(DIST_DIR)/web/$(API_NAME).wasm: $(STAMP)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tcargo build --release --target wasm32-unknown-unknown\n")
	fmt.Fprintf(b, "\tcp target/wasm32-unknown-unknown/release/%s.wasm $@\n\n",
		"$(shell echo $(API_NAME) | tr '-' '_')")
}
