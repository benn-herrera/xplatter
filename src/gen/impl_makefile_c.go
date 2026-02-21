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

	b.WriteString(`# ── C build configuration ─────────────────────────────────────────────────────

CC         ?= cc
CFLAGS     := -Wall -Wextra -std=c11 -I. -I$(GEN_DIR)
LIB_VISIBILITY_FLAGS := -fvisibility=hidden -D$(BUILD_MACRO)
LIB_C_FLAGS := -std=c11 -Wall -Wextra $(LIB_VISIBILITY_FLAGS)

PLATFORM_SERVICES := platform_services

# Ensure codegen runs before any target needs generated files
$(GEN_HEADER) $(GEN_SWIFT_BINDING) $(GEN_KOTLIN_BINDING) $(GEN_JS_BINDING) $(GEN_JNI_SOURCE): $(STAMP)

`)

	// Codegen stamp
	MakefileCodegenStamp(&b, "c", "-o generated")

	b.WriteString(`.PHONY: run shared-lib clean

# ── Local build ──────────────────────────────────────────────────────────────

IMPL_SOURCES := $(API_NAME)_impl.c

run: $(STAMP)
	@mkdir -p $(BUILD_DIR)
	$(CC) $(CFLAGS) -o $(BUILD_DIR)/$(API_NAME) \
		$(IMPL_SOURCES) $(PLATFORM_SERVICES)/desktop.c main.c
	./$(BUILD_DIR)/$(API_NAME)

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(STAMP)
	@mkdir -p $(BUILD_DIR)
ifeq ($(HOST_OS),Darwin)
	$(CC) $(CFLAGS) $(LIB_VISIBILITY_FLAGS) -shared -fPIC \
		-Wl,-install_name,@rpath/$(LIB_NAME).$(DYLIB_EXT) \
		-o $@ $(IMPL_SOURCES) $(PLATFORM_SERVICES)/desktop.c
else
	$(CC) $(CFLAGS) $(LIB_VISIBILITY_FLAGS) -shared -fPIC \
		-o $@ $(IMPL_SOURCES) $(PLATFORM_SERVICES)/desktop.c
endif

clean:
	rm -rf generated $(BUILD_DIR) $(DIST_DIR)

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

func (g *CMakefileGenerator) writeIOSArchRules(b *strings.Builder) {
	b.WriteString(`# $(1) = arch dir name, $(2) = clang target triple, $(3) = SDK name
define BUILD_IOS_ARCH

$(DIST_DIR)/ios/obj/$(1)/impl.o: $(IMPL_SOURCES) $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	xcrun --sdk $(3) clang $(CFLAGS) $(LIB_VISIBILITY_FLAGS) \
		-target $(2) -c -o $$@ $$<

$(DIST_DIR)/ios/obj/$(1)/platform.o: $(PLATFORM_SERVICES)/ios.c $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	xcrun --sdk $(3) clang $(LIB_C_FLAGS) \
		-target $(2) -c -o $$@ $$<

$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(DIST_DIR)/ios/obj/$(1)/impl.o $(DIST_DIR)/ios/obj/$(1)/platform.o
	ar rcs $$@ $$^

endef

$(eval $(call BUILD_IOS_ARCH,ios-arm64,arm64-apple-ios$(IOS_MIN),iphoneos))
$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,arm64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))
$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,x86_64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))

`)
}

func (g *CMakefileGenerator) writeAndroidABIRules(b *strings.Builder) {
	b.WriteString(`# $(1) = ABI name, $(2) = NDK target triple
define BUILD_ANDROID_ABI

$(DIST_DIR)/android/src/main/jniLibs/$(1)/$(LIB_NAME).so: $(IMPL_SOURCES) $(GEN_JNI_SOURCE) $(PLATFORM_SERVICES)/android.c $(GEN_HEADER)
	@mkdir -p $(DIST_DIR)/android/obj/$(1) $$(dir $$@)
	$(NDK_BIN)/$(2)-clang $(CFLAGS) -fPIC $(LIB_VISIBILITY_FLAGS) \
		-c -o $(DIST_DIR)/android/obj/$(1)/impl.o $(IMPL_SOURCES)
	$(NDK_BIN)/$(2)-clang $(LIB_C_FLAGS) -fPIC \
		-c -o $(DIST_DIR)/android/obj/$(1)/jni.o $(GEN_JNI_SOURCE)
	$(NDK_BIN)/$(2)-clang $(LIB_C_FLAGS) -fPIC \
		-c -o $(DIST_DIR)/android/obj/$(1)/platform.o $(PLATFORM_SERVICES)/android.c
	$(NDK_BIN)/$(2)-clang -shared -llog \
		$(DIST_DIR)/android/obj/$(1)/impl.o \
		$(DIST_DIR)/android/obj/$(1)/jni.o \
		$(DIST_DIR)/android/obj/$(1)/platform.o \
		-o $$@

endef

$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,aarch64-linux-android$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,armv7a-linux-androideabi$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,x86_64,x86_64-linux-android$(ANDROID_MIN_API)))
$(eval $(call BUILD_ANDROID_ABI,x86,i686-linux-android$(ANDROID_MIN_API)))

`)
}

func (g *CMakefileGenerator) writeWASMBuildRule(b *strings.Builder) {
	b.WriteString(`$(DIST_DIR)/web/obj/impl.o: $(IMPL_SOURCES) $(GEN_HEADER)
	@mkdir -p $(dir $@)
	$(EMCC) $(CFLAGS) -O2 $(LIB_VISIBILITY_FLAGS) -c -o $@ $<

$(DIST_DIR)/web/obj/platform.o: $(PLATFORM_SERVICES)/web.c
	@mkdir -p $(dir $@)
	$(EMCC) $(LIB_C_FLAGS) -O2 -c -o $@ $<

$(DIST_DIR)/web/$(API_NAME).wasm: $(DIST_DIR)/web/obj/impl.o $(DIST_DIR)/web/obj/platform.o
	$(EMCC) -o $@ $^ \
		--no-entry \
		-s 'EXPORTED_FUNCTIONS=$(WASM_EXPORTS)' \
		-s STANDALONE_WASM \
		-O2

`)
}
