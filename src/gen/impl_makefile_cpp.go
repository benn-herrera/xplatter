package gen

import (
	"strings"
)

func init() {
	Register("impl_makefile_cpp", func() Generator { return &CppMakefileGenerator{} })
}

// CppMakefileGenerator produces a scaffold Makefile for C++ implementations.
type CppMakefileGenerator struct{}

func (g *CppMakefileGenerator) Name() string { return "impl_makefile_cpp" }

func (g *CppMakefileGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	apiName := ctx.API.API.Name

	var b strings.Builder

	MakefileHeader(&b, ctx, "cpp")
	MakefileTargetConfig(&b)
	MakefileBindingVars(&b, apiName, "generated/")
	MakefileWASMExports(&b, apiName, ctx.API)

	b.WriteString(`# ── C++ build configuration ───────────────────────────────────────────────────

PLATFORM_SERVICES := platform_services

ifneq (,$(EXE))
CXX        := cl
CC         := cl
CXXFLAGS   := /W4 /std:c++20 /EHsc /I$(GEN_DIR) /D$(BUILD_MACRO)
CFLAGS     := /W4 /std:c17 /I$(GEN_DIR) /D$(BUILD_MACRO)
LIB_VISIBILITY_FLAGS := /D$(BUILD_MACRO)
LIB_C_FLAGS := /std:c17 /W4 $(LIB_VISIBILITY_FLAGS)
else
CXX        ?= c++
CC         ?= cc
CXXFLAGS   := -Wall -Wextra -std=c++20 -I$(GEN_DIR)
CFLAGS     := -Wall -Wextra -std=c17 -I$(GEN_DIR)
LIB_VISIBILITY_FLAGS := -fvisibility=hidden -D$(BUILD_MACRO)
LIB_C_FLAGS := -std=c17 -Wall -Wextra $(LIB_VISIBILITY_FLAGS)
endif

# Cross-compilation flags (always GCC/Clang-style, for NDK + Emscripten)
CROSS_CXXFLAGS       := -Wall -Wextra -std=c++20 -I$(GEN_DIR)
CROSS_VISIBILITY     := -fvisibility=hidden -D$(BUILD_MACRO)
CROSS_LIB_C_FLAGS    := -std=c17 -Wall -Wextra $(CROSS_VISIBILITY)

`)

	// Codegen stamp
	MakefileCodegenStamp(&b, "cpp", "-o generated")

	b.WriteString(`.PHONY: run shared-lib clean

# ── Local build ──────────────────────────────────────────────────────────────

IMPL_SOURCES   := $(API_NAME)_impl.cpp
SHIM_SOURCE    := $(GEN_DIR)$(API_NAME)_shim.cpp

# Ensure codegen runs before any target needs generated files
$(GEN_HEADER) $(GEN_SWIFT_BINDING) $(GEN_KOTLIN_BINDING) $(GEN_JS_BINDING) $(GEN_JNI_SOURCE) $(SHIM_SOURCE): $(STAMP)

run: $(STAMP)
	@mkdir -p $(BUILD_DIR)
ifneq (,$(EXE))
	$(CXX) $(CXXFLAGS) /Fe:$(BUILD_DIR)/$(API_NAME).exe \
		$(IMPL_SOURCES) $(SHIM_SOURCE) $(PLATFORM_SERVICES)/desktop.c main.cpp
else
	$(CXX) $(CXXFLAGS) -o $(BUILD_DIR)/$(API_NAME) \
		$(IMPL_SOURCES) $(SHIM_SOURCE) $(PLATFORM_SERVICES)/desktop.c main.cpp
endif
	./$(BUILD_DIR)/$(API_NAME)$(EXE)

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(STAMP)
	@mkdir -p $(BUILD_DIR)
ifeq ($(HOST_OS),Darwin)
	$(CXX) $(CXXFLAGS) $(LIB_VISIBILITY_FLAGS) -shared -fPIC \
		-Wl,-install_name,@rpath/$(LIB_NAME).$(DYLIB_EXT) \
		-o $@ $(IMPL_SOURCES) $(SHIM_SOURCE) $(PLATFORM_SERVICES)/desktop.c
else ifneq (,$(EXE))
	$(CXX) /LD $(LIB_VISIBILITY_FLAGS) $(CXXFLAGS) \
		$(IMPL_SOURCES) $(SHIM_SOURCE) $(PLATFORM_SERVICES)/desktop.c \
		/Fe:$@ /link /IMPLIB:$(BUILD_DIR)/$(API_NAME).lib
else
	$(CXX) $(CXXFLAGS) $(LIB_VISIBILITY_FLAGS) -shared -fPIC \
		-o $@ $(IMPL_SOURCES) $(SHIM_SOURCE) $(PLATFORM_SERVICES)/desktop.c
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

func (g *CppMakefileGenerator) writeIOSArchRules(b *strings.Builder) {
	b.WriteString(`# $(1) = arch dir name, $(2) = clang target triple, $(3) = SDK name
define BUILD_IOS_ARCH

$(DIST_DIR)/ios/obj/$(1)/impl.o: $(IMPL_SOURCES) $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	xcrun --sdk $(3) $(CXX) $(CXXFLAGS) $(LIB_VISIBILITY_FLAGS) \
		-target $(2) -c -o $$@ $$<

$(DIST_DIR)/ios/obj/$(1)/shim.o: $(SHIM_SOURCE) $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	xcrun --sdk $(3) $(CXX) $(CXXFLAGS) $(LIB_VISIBILITY_FLAGS) \
		-target $(2) -c -o $$@ $$<

$(DIST_DIR)/ios/obj/$(1)/platform.o: $(PLATFORM_SERVICES)/ios.c $(GEN_HEADER)
	@mkdir -p $$(dir $$@)
	xcrun --sdk $(3) clang $(LIB_C_FLAGS) \
		-target $(2) -c -o $$@ $$<

$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(DIST_DIR)/ios/obj/$(1)/impl.o $(DIST_DIR)/ios/obj/$(1)/shim.o $(DIST_DIR)/ios/obj/$(1)/platform.o
	ar rcs $$@ $$^

endef

$(eval $(call BUILD_IOS_ARCH,ios-arm64,arm64-apple-ios$(IOS_MIN),iphoneos))
$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,arm64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))
$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,x86_64-apple-ios$(IOS_MIN)-simulator,iphonesimulator))

`)
}

func (g *CppMakefileGenerator) writeAndroidABIRules(b *strings.Builder) {
	b.WriteString(`# $(1) = ABI name, $(2) = NDK target triple
define BUILD_ANDROID_ABI
$(DIST_DIR)/android/src/main/jniLibs/$(1)/$(LIB_NAME).so: $(IMPL_SOURCES) $(SHIM_SOURCE) $(GEN_JNI_SOURCE) $(PLATFORM_SERVICES)/android.c $(GEN_HEADER)
	@mkdir -p $(DIST_DIR)/android/obj/$(1) $$(dir $$@)
	"$(NDK_BIN)/$(2)-clang++" $(CROSS_CXXFLAGS) -fPIC $(CROSS_VISIBILITY) \
		-c -o $(DIST_DIR)/android/obj/$(1)/impl.o $(IMPL_SOURCES)
	"$(NDK_BIN)/$(2)-clang++" $(CROSS_CXXFLAGS) -fPIC $(CROSS_VISIBILITY) \
		-c -o $(DIST_DIR)/android/obj/$(1)/shim.o $(SHIM_SOURCE)
	"$(NDK_BIN)/$(2)-clang" $(CROSS_LIB_C_FLAGS) -fPIC \
		-c -o $(DIST_DIR)/android/obj/$(1)/jni.o $(GEN_JNI_SOURCE)
	"$(NDK_BIN)/$(2)-clang" $(CROSS_LIB_C_FLAGS) -fPIC \
		-c -o $(DIST_DIR)/android/obj/$(1)/platform.o $(PLATFORM_SERVICES)/android.c
	"$(NDK_BIN)/$(2)-clang++" -shared -static-libstdc++ -llog \
		$(DIST_DIR)/android/obj/$(1)/impl.o \
		$(DIST_DIR)/android/obj/$(1)/shim.o \
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

func (g *CppMakefileGenerator) writeWASMBuildRule(b *strings.Builder) {
	b.WriteString(`$(DIST_DIR)/web/$(API_NAME).wasm: $(STAMP) CMakeLists.txt
	@mkdir -p $(DIST_DIR)/web
	cmake -S . -B build/web \
		-DCMAKE_TOOLCHAIN_FILE=$(EMSCRIPTEN_TOOLCHAIN) \
		-DCMAKE_BUILD_TYPE=Release
	cmake --build build/web
	cp build/web/$(API_NAME).wasm $@

`)
}
