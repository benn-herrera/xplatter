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

	b.WriteString("# Ensure codegen runs before any target needs generated files\n")
	b.WriteString("$(GEN_HEADER): $(STAMP)\n\n")

	// Codegen stamp — Go needs generated .go files copied to package root
	// because `go build .` only compiles files in the current directory.
	b.WriteString("# ── Codegen ──────────────────────────────────────────────────────────────────\n\n")
	b.WriteString("$(STAMP): $(API_DEF)\n")
	b.WriteString("\t@mkdir -p $(BUILD_DIR)\n")
	b.WriteString("\t$(XPLATTER) generate --impl-lang go -o generated $(API_DEF)\n")
	b.WriteString("\tcp generated/$(API_NAME)_*.go .\n")
	b.WriteString("\t@touch $@\n\n")

	// Generated Go source copies (for .gitignore and clean)
	fmt.Fprintf(&b, "GEN_GO_SOURCES := $(wildcard generated/%s_*.go)\n", apiName)
	b.WriteString("GEN_GO_COPIES  := $(notdir $(GEN_GO_SOURCES))\n\n")

	// Phony declarations
	b.WriteString(".PHONY: run shared-lib clean\n\n")

	// Local build
	b.WriteString("# ── Local build ──────────────────────────────────────────────────────────────\n\n")

	b.WriteString("run: $(STAMP)\n")
	b.WriteString("\t@mkdir -p $(BUILD_DIR)\n")
	b.WriteString("\tgo build -o $(BUILD_DIR)/$(API_NAME) .\n")
	b.WriteString("\t./$(BUILD_DIR)/$(API_NAME)\n\n")

	// Shared library
	b.WriteString("shared-lib: $(SHARED_LIB)\n\n")
	b.WriteString("$(SHARED_LIB): $(STAMP)\n")
	b.WriteString("\tgo build -buildmode=c-shared -ldflags='-extldflags \"-Wl,-install_name,@rpath/$(LIB_NAME).$(DYLIB_EXT)\"' -o $(SHARED_LIB) .\n\n")

	// Clean — also remove copied Go source files from package root
	b.WriteString("clean:\n")
	b.WriteString("\trm -rf generated $(BUILD_DIR) $(DIST_DIR)\n")
	b.WriteString("\trm -f $(GEN_GO_COPIES)\n\n")

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
	b.WriteString("IOS_SDK := $(shell xcrun --sdk iphoneos --show-sdk-path 2>/dev/null)\n")
	b.WriteString("SIM_SDK := $(shell xcrun --sdk iphonesimulator --show-sdk-path 2>/dev/null)\n")
	b.WriteString("IOS_CC  := $(shell xcrun --sdk iphoneos --find clang 2>/dev/null)\n")
	b.WriteString("SIM_CC  := $(shell xcrun --sdk iphonesimulator --find clang 2>/dev/null)\n\n")

	// Macro for building one iOS arch via Go CGO c-archive
	b.WriteString("# $(1) = arch dir name, $(2) = GOARCH, $(3) = clang -arch flag,\n")
	b.WriteString("# $(4) = clang -target triple, $(5) = CC path, $(6) = sysroot\n")
	b.WriteString("define BUILD_IOS_ARCH\n\n")

	b.WriteString("$(DIST_DIR)/ios/obj/$(1)/$(LIB_NAME).a: $(STAMP)\n")
	b.WriteString("\t@mkdir -p $$(dir $$@)\n")
	b.WriteString("\tCGO_ENABLED=1 GOOS=ios GOARCH=$(2) \\\n")
	b.WriteString("\t\tCC=\"$(5)\" \\\n")
	b.WriteString("\t\tCGO_CFLAGS=\"-arch $(3) -target $(4) -isysroot $(6)\" \\\n")
	b.WriteString("\t\tCGO_LDFLAGS=\"-arch $(3) -target $(4) -isysroot $(6)\" \\\n")
	b.WriteString("\t\tgo build -buildmode=c-archive -o $$@ .\n\n")

	b.WriteString("endef\n\n")

	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-arm64,arm64,arm64,arm64-apple-ios$(IOS_MIN),$(IOS_CC),$(IOS_SDK)))\n")
	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-sim-arm64,arm64,arm64,arm64-apple-ios$(IOS_MIN)-simulator,$(SIM_CC),$(SIM_SDK)))\n")
	b.WriteString("$(eval $(call BUILD_IOS_ARCH,ios-sim-x86_64,amd64,x86_64,x86_64-apple-ios$(IOS_MIN)-simulator,$(SIM_CC),$(SIM_SDK)))\n\n")
}

func (g *GoMakefileGenerator) writeAndroidABIRules(b *strings.Builder, apiName string) {
	fmt.Fprintf(b, "GEN_JNI_SOURCE_LOCAL := %s_jni.c\n\n", apiName)

	// Macro for building one Android ABI via Go CGO c-shared
	b.WriteString("# $(1) = ABI name, $(2) = GOARCH, $(3) = NDK clang prefix, $(4) = extra env (e.g. GOARM=7)\n")
	b.WriteString("#\n")
	b.WriteString("# Strategy: CGO auto-compiles all .c files in the package directory alongside\n")
	b.WriteString("# import \"C\" files. We temporarily copy the JNI bridge into the package root so\n")
	b.WriteString("# CGO includes it when building the c-shared .so. (Do not run ABI targets in\n")
	b.WriteString("# parallel — each target writes/removes the same jni.c copy in the package dir.)\n")
	b.WriteString("define BUILD_ANDROID_ABI\n\n")

	b.WriteString("$(DIST_DIR)/android/src/main/jniLibs/$(1)/$(LIB_NAME).so: $(STAMP)\n")
	b.WriteString("\t@mkdir -p $$(dir $$@)\n")
	b.WriteString("\tcp $(GEN_JNI_SOURCE) $(GEN_JNI_SOURCE_LOCAL)\n")
	b.WriteString("\tCGO_ENABLED=1 GOOS=android GOARCH=$(2) $(4) \\\n")
	b.WriteString("\t\tCC=$(NDK_BIN)/$(3)-clang \\\n")
	b.WriteString("\t\tCGO_CFLAGS=\"-I generated\" \\\n")
	b.WriteString("\t\tgo build -buildmode=c-shared -o $$@ . || (rm -f $(GEN_JNI_SOURCE_LOCAL); exit 1)\n")
	b.WriteString("\trm -f $(GEN_JNI_SOURCE_LOCAL)\n\n")

	b.WriteString("endef\n\n")

	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,arm64-v8a,arm64,aarch64-linux-android$(ANDROID_MIN_API),))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,armeabi-v7a,arm,armv7a-linux-androideabi$(ANDROID_MIN_API),GOARM=7))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,x86_64,amd64,x86_64-linux-android$(ANDROID_MIN_API),))\n")
	b.WriteString("$(eval $(call BUILD_ANDROID_ABI,x86,386,i686-linux-android$(ANDROID_MIN_API),))\n\n")
}

func (g *GoMakefileGenerator) writeWASMBuildRule(b *strings.Builder) {
	b.WriteString("$(DIST_DIR)/web/$(API_NAME).wasm: $(STAMP)\n")
	b.WriteString("\t@mkdir -p $(dir $@)\n")
	b.WriteString("\tGOOS=wasip1 GOARCH=wasm go build -o $@ .\n\n")
}
