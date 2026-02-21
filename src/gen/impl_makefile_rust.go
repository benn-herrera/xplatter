package gen

import (
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

	b.WriteString(`# Ensure codegen runs before any target needs generated files
$(GEN_HEADER) $(GEN_SWIFT_BINDING) $(GEN_KOTLIN_BINDING) $(GEN_JS_BINDING) $(GEN_JNI_SOURCE): $(STAMP)

CROSS_LIB_C_FLAGS := -std=c17 -Wall -Wextra -fvisibility=hidden -D$(BUILD_MACRO)

`)

	// Codegen stamp
	MakefileCodegenStamp(&b, "rust", "-o generated")

	b.WriteString(`.PHONY: run shared-lib clean

run: $(STAMP)
	cargo test

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(STAMP)
	cargo build --release
	@mkdir -p $(BUILD_DIR)
ifneq (,$(EXE))
	cp target/release/$(API_NAME).dll $(SHARED_LIB)
else ifeq ($(HOST_OS),Darwin)
	cp target/release/$(LIB_NAME).$(DYLIB_EXT) $(SHARED_LIB)
	install_name_tool -id @rpath/$(LIB_NAME).$(DYLIB_EXT) $(SHARED_LIB)
else
	cp target/release/$(LIB_NAME).$(DYLIB_EXT) $(SHARED_LIB)
endif

clean:
	cargo clean
	rm -rf generated $(BUILD_DIR) $(DIST_DIR) flatbuffers

`)

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
	b.WriteString(`# $(1) = arch dir name, $(2) = Rust target triple
define BUILD_IOS_ARCH

$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(STAMP)
	@mkdir -p $$(dir $$@)
	cargo build --release --target $(2)
	cp target/$(2)/release/$(LIB_NAME).a $$@

endef

$(eval $(call BUILD_IOS_ARCH,ios-arm64,aarch64-apple-ios))
$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,aarch64-apple-ios-sim))
$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,x86_64-apple-ios))

`)
}

func (g *RustMakefileGenerator) writeAndroidABIRules(b *strings.Builder) {
	b.WriteString(`# $(1) = ABI name, $(2) = Rust target triple, $(3) = NDK target prefix, $(4) = uppercase Cargo target
define BUILD_ANDROID_ABI

$(DIST_DIR)/android/src/main/jniLibs/$(1)/$(LIB_NAME).so: $(STAMP)
	@mkdir -p $(DIST_DIR)/android/obj/$(1) $$(dir $$@)
	CARGO_TARGET_$(4)_LINKER="$(NDK_BIN)/$(3)-clang$(NDK_CMD)" \
		PATH=$(NDK_BIN):$$$$PATH cargo build --release --target $(2)
	"$(NDK_BIN)/$(3)-clang" $(CROSS_LIB_C_FLAGS) -fPIC \
		-Igenerated -c -o $(DIST_DIR)/android/obj/$(1)/jni.o $(GEN_JNI_SOURCE)
	"$(NDK_BIN)/$(3)-clang" -shared \
		-Wl,--whole-archive target/$(2)/release/$(LIB_NAME).a -Wl,--no-whole-archive \
		$(DIST_DIR)/android/obj/$(1)/jni.o \
		-ldl -lm -llog \
		-o $$@

endef

$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,aarch64-linux-android,aarch64-linux-android$(ANDROID_MIN_API),AARCH64_LINUX_ANDROID))
$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,armv7-linux-androideabi,armv7a-linux-androideabi$(ANDROID_MIN_API),ARMV7_LINUX_ANDROIDEABI))
$(eval $(call BUILD_ANDROID_ABI,x86_64,x86_64-linux-android,x86_64-linux-android$(ANDROID_MIN_API),X86_64_LINUX_ANDROID))
$(eval $(call BUILD_ANDROID_ABI,x86,i686-linux-android,i686-linux-android$(ANDROID_MIN_API),I686_LINUX_ANDROID))

`)
}

func (g *RustMakefileGenerator) writeWASMBuildRule(b *strings.Builder) {
	b.WriteString(`$(DIST_DIR)/web/$(API_NAME).wasm: $(STAMP)
	@mkdir -p $(dir $@)
	cargo build --release --target wasm32-unknown-unknown
	cp target/wasm32-unknown-unknown/release/$(shell echo $(API_NAME) | tr '-' '_').wasm $@

`)
}
