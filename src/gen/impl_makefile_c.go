package gen

import (
	"strings"
)

func init() {
	Register("impl_makefile_c", func() Generator { return &CMakefileGenerator{} })
}

// CMakefileGenerator produces a scaffold Makefile for C implementations.
// Similar to C++ but uses clang, no shim source (C impl directly exports C ABI).
type CMakefileGenerator struct{}

func (g *CMakefileGenerator) Name() string { return "impl_makefile_c" }

func (g *CMakefileGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	apiName := ctx.API.API.Name

	var b strings.Builder

	MakefileHeader(&b, ctx, "c")
	MakefileTargetConfig(&b)
	MakefileBindingVars(&b, apiName, "generated/")
	MakefileWASMExports(&b, apiName, ctx.API)

	// C specific variables
	b.WriteString("# ── C build configuration ─────────────────────────────────────────────────────\n\n")
	b.WriteString("CC         ?= cc\n")
	b.WriteString("CFLAGS     := -Wall -Wextra -std=c11 -I. -Igenerated\n")
	b.WriteString("LIB_VISIBILITY_FLAGS := -fvisibility=hidden -D$(BUILD_MACRO)\n")
	b.WriteString("LIB_C_FLAGS := -std=c11 -Wall -Wextra $(LIB_VISIBILITY_FLAGS)\n\n")

	b.WriteString("PLATFORM_SERVICES := platform_services\n\n")

	b.WriteString("# Ensure codegen runs before any target needs generated files\n")
	b.WriteString("$(GEN_HEADER): $(STAMP)\n\n")

	// Codegen stamp
	MakefileCodegenStamp(&b, "c", "-o generated")

	// Phony declarations
	b.WriteString(".PHONY: run shared-lib clean\n\n")

	// Local build
	b.WriteString("# ── Local build ──────────────────────────────────────────────────────────────\n\n")
	b.WriteString("IMPL_SOURCES := $(API_NAME)_impl.c\n\n")

	b.WriteString("run: $(STAMP)\n")
	b.WriteString("\t@mkdir -p $(BUILD_DIR)\n")
	b.WriteString("\t$(CC) $(CFLAGS) -o $(BUILD_DIR)/$(API_NAME) \\\n")
	b.WriteString("\t\t$(IMPL_SOURCES) $(PLATFORM_SERVICES)/desktop.c main.c\n")
	b.WriteString("\t./$(BUILD_DIR)/$(API_NAME)\n\n")

	// Shared library
	b.WriteString("shared-lib: $(SHARED_LIB)\n\n")
	b.WriteString("$(SHARED_LIB): $(STAMP)\n")
	b.WriteString("\t@mkdir -p $(BUILD_DIR)\n")
	b.WriteString("\t$(CC) $(CFLAGS) $(LIB_VISIBILITY_FLAGS) -shared -fPIC \\\n")
	b.WriteString("\t\t-Wl,-install_name,@rpath/$(LIB_NAME).$(DYLIB_EXT) \\\n")
	b.WriteString("\t\t-o $@ $(IMPL_SOURCES) $(PLATFORM_SERVICES)/desktop.c\n\n")

	// Clean
	b.WriteString("clean:\n")
	b.WriteString("\trm -rf generated $(BUILD_DIR) $(DIST_DIR)\n\n")

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

func (g *CMakefileGenerator) writeIOSArchRules(b *strings.Builder) {
	b.WriteString("# $(1) = arch dir name, $(2) = clang target triple, $(3) = SDK name\n")
	b.WriteString("define BUILD_IOS_ARCH\n\n")

	b.WriteString("$(DIST_DIR)/ios/obj/$(1)/impl.o: $(IMPL_SOURCES) $(GEN_HEADER)\n")
	b.WriteString("\t@mkdir -p $$(dir $$@)\n")
	b.WriteString("\txcrun --sdk $(3) clang $(CFLAGS) $(LIB_VISIBILITY_FLAGS) \\\n")
	b.WriteString("\t\t-target $(2) -c -o $$@ $$<\n\n")

	b.WriteString("$(DIST_DIR)/ios/obj/$(1)/platform.o: $(PLATFORM_SERVICES)/ios.c $(GEN_HEADER)\n")
	b.WriteString("\t@mkdir -p $$(dir $$@)\n")
	b.WriteString("\txcrun --sdk $(3) clang $(LIB_C_FLAGS) \\\n")
	b.WriteString("\t\t-target $(2) -c -o $$@ $$<\n\n")

	b.WriteString("$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(DIST_DIR)/ios/obj/$(1)/impl.o $(DIST_DIR)/ios/obj/$(1)/platform.o\n")
	b.WriteString("\tar rcs $$@ $$^\n\n")

	b.WriteString("endef\n\n")

	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-arm64,arm64-apple-ios$(IOS_MIN),iphoneos))\n")
	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,arm64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))\n")
	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,x86_64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))\n\n")
}

func (g *CMakefileGenerator) writeAndroidABIRules(b *strings.Builder) {
	b.WriteString("# $(1) = ABI name, $(2) = NDK target triple\n")
	b.WriteString("define BUILD_ANDROID_ABI\n\n")

	b.WriteString("$(DIST_DIR)/android/src/main/jniLibs/$(1)/$(LIB_NAME).so: $(IMPL_SOURCES) $(GEN_JNI_SOURCE) $(PLATFORM_SERVICES)/android.c $(GEN_HEADER)\n")
	b.WriteString("\t@mkdir -p $(DIST_DIR)/android/obj/$(1) $$(dir $$@)\n")
	b.WriteString("\t$(NDK_BIN)/$(2)-clang $(CFLAGS) -fPIC $(LIB_VISIBILITY_FLAGS) \\\n")
	b.WriteString("\t\t-c -o $(DIST_DIR)/android/obj/$(1)/impl.o $(IMPL_SOURCES)\n")
	b.WriteString("\t$(NDK_BIN)/$(2)-clang $(LIB_C_FLAGS) -fPIC \\\n")
	b.WriteString("\t\t-c -o $(DIST_DIR)/android/obj/$(1)/jni.o $(GEN_JNI_SOURCE)\n")
	b.WriteString("\t$(NDK_BIN)/$(2)-clang $(LIB_C_FLAGS) -fPIC \\\n")
	b.WriteString("\t\t-c -o $(DIST_DIR)/android/obj/$(1)/platform.o $(PLATFORM_SERVICES)/android.c\n")
	b.WriteString("\t$(NDK_BIN)/$(2)-clang -shared -llog \\\n")
	b.WriteString("\t\t$(DIST_DIR)/android/obj/$(1)/impl.o \\\n")
	b.WriteString("\t\t$(DIST_DIR)/android/obj/$(1)/jni.o \\\n")
	b.WriteString("\t\t$(DIST_DIR)/android/obj/$(1)/platform.o \\\n")
	b.WriteString("\t\t-o $$@\n\n")

	b.WriteString("endef\n\n")

	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,aarch64-linux-android$(ANDROID_MIN_API)))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,armv7a-linux-androideabi$(ANDROID_MIN_API)))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,x86_64,x86_64-linux-android$(ANDROID_MIN_API)))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,x86,i686-linux-android$(ANDROID_MIN_API)))\n\n")
}

func (g *CMakefileGenerator) writeWASMBuildRule(b *strings.Builder) {
	b.WriteString("$(DIST_DIR)/web/obj/impl.o: $(IMPL_SOURCES) $(GEN_HEADER)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\t$(EMCC) $(CFLAGS) -O2 $(LIB_VISIBILITY_FLAGS) -c -o $@ $<\n\n")

	b.WriteString("$(DIST_DIR)/web/obj/platform.o: $(PLATFORM_SERVICES)/web.c\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\t$(EMCC) $(LIB_C_FLAGS) -O2 -c -o $@ $<\n\n")

	b.WriteString("$(DIST_DIR)/web/$(API_NAME).wasm: $(DIST_DIR)/web/obj/impl.o $(DIST_DIR)/web/obj/platform.o\n")
	b.WriteString("\t$(EMCC) -o $@ $^ \\\n")
	b.WriteString("\t\t--no-entry \\\n")
	b.WriteString("\t\t-s 'EXPORTED_FUNCTIONS=$(WASM_EXPORTS)' \\\n")
	b.WriteString("\t\t-s STANDALONE_WASM \\\n")
	b.WriteString("\t\t-O2\n\n")
}
