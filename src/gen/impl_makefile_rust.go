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
$(GEN_HEADER): $(STAMP)

LIB_C_FLAGS := -std=c11 -Wall -Wextra -fvisibility=hidden -D$(BUILD_MACRO)

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
	cp target/release/$(LIB_NAME).$(DYLIB_EXT) $(SHARED_LIB)
ifeq ($(HOST_OS),Darwin)
	install_name_tool -id @rpath/$(LIB_NAME).$(DYLIB_EXT) $(SHARED_LIB)
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
	b.WriteString(`# $(1) = ABI name, $(2) = Rust target triple, $(3) = NDK target prefix
define BUILD_ANDROID_ABI

$(DIST_DIR)/android/src/main/jniLibs/$(1)/$(LIB_NAME).so: $(STAMP)
	@mkdir -p $(DIST_DIR)/android/obj/$(1) $$(dir $$@)
	PATH=$(NDK_BIN):$$$$PATH cargo build --release --target $(2)
	$(NDK_BIN)/$(3)-clang $(LIB_C_FLAGS) -fPIC \
		-Igenerated -c -o $(DIST_DIR)/android/obj/$(1)/jni.o $(GEN_JNI_SOURCE)
	$(NDK_BIN)/$(3)-clang -shared \
		-Wl,--whole-archive target/$(2)/release/$(LIB_NAME).a -Wl,--no-whole-archive \
		$(DIST_DIR)/android/obj/$(1)/jni.o \
		-ldl -lm -llog \
		-o $$@

endef

$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,aarch64-linux-android,aarch64-linux-android$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,armv7-linux-androideabi,armv7a-linux-androideabi$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,x86_64,x86_64-linux-android,x86_64-linux-android$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,x86,i686-linux-android,i686-linux-android$(ANDROID_MIN_API)))

`)
}

func (g *RustMakefileGenerator) writeWASMBuildRule(b *strings.Builder) {
	b.WriteString(`$(DIST_DIR)/web/$(API_NAME).wasm: $(STAMP)
	@mkdir -p $(dir $@)
	cargo build --release --target wasm32-unknown-unknown
	cp target/wasm32-unknown-unknown/release/$(shell echo $(API_NAME) | tr '-' '_').wasm $@

`)
}
