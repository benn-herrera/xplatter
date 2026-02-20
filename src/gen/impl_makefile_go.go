package gen

import (
	"fmt"
	"strings"
)

func init() {
	Register("impl_makefile_go", func() Generator { return &GoMakefileGenerator{} })
}

// GoMakefileGenerator produces a scaffold Makefile for Go implementations.
// Uses CGO for cross-compilation: c-archive for iOS, c-shared for Android/Desktop.
type GoMakefileGenerator struct{}

func (g *GoMakefileGenerator) Name() string { return "impl_makefile_go" }

func (g *GoMakefileGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	apiName := ctx.API.API.Name

	var b strings.Builder

	MakefileHeader(&b, ctx, "go")
	MakefileTargetConfig(&b)
	MakefileBindingVars(&b, apiName, "generated/")
	MakefileWASMExports(&b, apiName, ctx.API)

	b.WriteString(`# Ensure codegen runs before any target needs generated files
$(GEN_HEADER): $(STAMP)

# ── Codegen ──────────────────────────────────────────────────────────────────

$(STAMP): $(API_DEF)
	@mkdir -p $(BUILD_DIR)
	$(XPLATTER) generate --impl-lang go -o generated $(API_DEF)
	cp generated/$(API_NAME)_*.go .
	@touch $@

`)

	// Generated Go source copies (for .gitignore and clean)
	fmt.Fprintf(&b, "GEN_GO_SOURCES := $(wildcard generated/%s_*.go)\n", apiName)
	b.WriteString("GEN_GO_COPIES  := $(notdir $(GEN_GO_SOURCES))\n\n")

	b.WriteString(`.PHONY: run shared-lib clean

# ── Local build ──────────────────────────────────────────────────────────────

run: $(STAMP)
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(API_NAME) .
	./$(BUILD_DIR)/$(API_NAME)

shared-lib: $(SHARED_LIB)

$(SHARED_LIB): $(STAMP)
	go build -buildmode=c-shared -ldflags='-extldflags "-Wl,-install_name,@rpath/$(LIB_NAME).$(DYLIB_EXT)"' -o $(SHARED_LIB) .

clean:
	rm -rf generated $(BUILD_DIR) $(DIST_DIR)
	rm -f $(GEN_GO_COPIES)

`)

	// iOS packaging
	MakefilePackageIOS(&b, func(b *strings.Builder) {
		g.writeIOSArchRules(b)
	})

	// Android packaging
	MakefilePackageAndroid(&b, func(b *strings.Builder) {
		g.writeAndroidABIRules(b, apiName)
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

func (g *GoMakefileGenerator) writeIOSArchRules(b *strings.Builder) {
	b.WriteString(`IOS_SDK := $(shell xcrun --sdk iphoneos --show-sdk-path 2>/dev/null)
SIM_SDK := $(shell xcrun --sdk iphonesimulator --show-sdk-path 2>/dev/null)
IOS_CC  := $(shell xcrun --sdk iphoneos --find clang 2>/dev/null)
SIM_CC  := $(shell xcrun --sdk iphonesimulator --find clang 2>/dev/null)

# $(1) = arch dir name, $(2) = GOARCH, $(3) = clang -arch flag,
# $(4) = clang -target triple, $(5) = CC path, $(6) = sysroot
define BUILD_IOS_ARCH

$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(STAMP)
	@mkdir -p $$(dir $$@)
	CGO_ENABLED=1 GOOS=ios GOARCH=$(2) \
		CC="$(5)" \
		CGO_CFLAGS="-arch $(3) -target $(4) -isysroot $(6)" \
		CGO_LDFLAGS="-arch $(3) -target $(4) -isysroot $(6)" \
		go build -buildmode=c-archive -o $$@ .

endef

$(eval $(call BUILD_IOS_ARCH,ios-arm64,arm64,arm64,arm64-apple-ios$(IOS_MIN),$(IOS_CC),$(IOS_SDK)))
$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,arm64,arm64,arm64-apple-ios$(IOS_MIN)-simulator,$(SIM_CC),$(SIM_SDK)))
$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,amd64,x86_64,x86_64-apple-ios$(IOS_MIN)-simulator,$(SIM_CC),$(SIM_SDK)))

`)
}

func (g *GoMakefileGenerator) writeAndroidABIRules(b *strings.Builder, apiName string) {
	fmt.Fprintf(b, "GEN_JNI_SOURCE_LOCAL := %s_jni.c\n\n", apiName)

	b.WriteString(`# $(1) = ABI name, $(2) = GOARCH, $(3) = NDK clang prefix, $(4) = extra env (e.g. GOARM=7)
#
# Strategy: CGO auto-compiles all .c files in the package directory alongside
# import "C" files. We temporarily copy the JNI bridge into the package root so
# CGO includes it when building the c-shared .so. (Do not run ABI targets in
# parallel — each target writes/removes the same jni.c copy in the package dir.)
define BUILD_ANDROID_ABI

$(DIST_DIR)/android/src/main/jniLibs/$(1)/$(LIB_NAME).so: $(STAMP)
	@mkdir -p $$(dir $$@)
	cp $(GEN_JNI_SOURCE) $(GEN_JNI_SOURCE_LOCAL)
	CGO_ENABLED=1 GOOS=android GOARCH=$(2) $(4) \
		CC=$(NDK_BIN)/$(3)-clang \
		CGO_CFLAGS="-I generated" \
		go build -buildmode=c-shared -o $$@ . || (rm -f $(GEN_JNI_SOURCE_LOCAL); exit 1)
	rm -f $(GEN_JNI_SOURCE_LOCAL)

endef

$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,arm64,aarch64-linux-android$(ANDROID_MIN_API),))
$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,arm,armv7a-linux-androideabi$(ANDROID_MIN_API),GOARM=7))
$(eval $(call BUILD_ANDROID_ABI,x86_64,amd64,x86_64-linux-android$(ANDROID_MIN_API),))
$(eval $(call BUILD_ANDROID_ABI,x86,386,i686-linux-android$(ANDROID_MIN_API),))

`)
}

func (g *GoMakefileGenerator) writeWASMBuildRule(b *strings.Builder) {
	b.WriteString(`$(DIST_DIR)/web/$(API_NAME).wasm: $(STAMP)
	@mkdir -p $(dir $@)
	GOOS=wasip1 GOARCH=wasm go build -o $@ .

`)
}
