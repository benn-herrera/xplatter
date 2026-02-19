package gen

import (
	"fmt"
	"strings"
)

func init() {
	Register("impl_platform_services", func() Generator { return &PlatformServicesGenerator{} })
}

// PlatformServicesGenerator produces scaffold platform services stub files
// for each target platform. These are link-time functions declared in the
// generated C header that the implementation can call.
type PlatformServicesGenerator struct{}

func (g *PlatformServicesGenerator) Name() string { return "impl_platform_services" }

func (g *PlatformServicesGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	apiName := ctx.API.API.Name

	var files []*OutputFile

	targets := ctx.API.EffectiveTargets()
	targetSet := make(map[string]bool)
	for _, t := range targets {
		targetSet[t] = true
	}

	// Desktop covers windows, linux, macos
	needsDesktop := targetSet["windows"] || targetSet["linux"] || targetSet["macos"]

	if needsDesktop {
		files = append(files, g.generateDesktop(apiName))
	}
	if targetSet["ios"] {
		files = append(files, g.generateIOS(apiName))
	}
	if targetSet["android"] {
		files = append(files, g.generateAndroid(apiName))
	}
	if targetSet["web"] {
		files = append(files, g.generateWeb(apiName))
	}

	return files, nil
}

func (g *PlatformServicesGenerator) generateDesktop(apiName string) *OutputFile {
	var b strings.Builder
	b.WriteString("/*\n")
	fmt.Fprintf(&b, " * Desktop platform services for %s.\n", apiName)
	b.WriteString(" * Stub implementations — fill in with platform-specific behavior.\n")
	b.WriteString(" */\n\n")
	b.WriteString("#include <stdint.h>\n\n")
	writePlatformServiceStubs(&b, apiName)
	return &OutputFile{Path: "platform_services/desktop.c", Content: []byte(b.String()), Scaffold: true, ProjectFile: true}
}

func (g *PlatformServicesGenerator) generateIOS(apiName string) *OutputFile {
	var b strings.Builder
	b.WriteString("/*\n")
	fmt.Fprintf(&b, " * iOS platform services for %s.\n", apiName)
	b.WriteString(" * Logging uses os_log; resource functions are stubs.\n")
	b.WriteString(" */\n\n")
	b.WriteString("#include <stdint.h>\n")
	b.WriteString("#include <os/log.h>\n\n")

	// iOS-specific logging
	fmt.Fprintf(&b, "void %s_log_sink(int32_t level, const char* tag, const char* message) {\n", apiName)
	b.WriteString("    os_log_type_t type = (level <= 1) ? OS_LOG_TYPE_DEBUG : OS_LOG_TYPE_DEFAULT;\n")
	b.WriteString("    os_log_with_type(OS_LOG_DEFAULT, type, \"[%{public}s] %{public}s\", tag, message);\n")
	b.WriteString("}\n\n")

	writeResourceStubs(&b, apiName)
	return &OutputFile{Path: "platform_services/ios.c", Content: []byte(b.String()), Scaffold: true, ProjectFile: true}
}

func (g *PlatformServicesGenerator) generateAndroid(apiName string) *OutputFile {
	var b strings.Builder
	b.WriteString("/*\n")
	fmt.Fprintf(&b, " * Android platform services for %s.\n", apiName)
	b.WriteString(" * Logging uses __android_log_print; resource functions are stubs.\n")
	b.WriteString(" */\n\n")
	b.WriteString("#include <stdint.h>\n")
	b.WriteString("#include <android/log.h>\n\n")

	// Android-specific logging
	fmt.Fprintf(&b, "void %s_log_sink(int32_t level, const char* tag, const char* message) {\n", apiName)
	b.WriteString("    int prio = (level <= 1) ? ANDROID_LOG_DEBUG : ANDROID_LOG_INFO;\n")
	b.WriteString("    __android_log_print(prio, tag, \"%s\", message);\n")
	b.WriteString("}\n\n")

	writeResourceStubs(&b, apiName)
	return &OutputFile{Path: "platform_services/android.c", Content: []byte(b.String()), Scaffold: true, ProjectFile: true}
}

func (g *PlatformServicesGenerator) generateWeb(apiName string) *OutputFile {
	var b strings.Builder
	b.WriteString("/*\n")
	fmt.Fprintf(&b, " * Web/WASM platform services for %s.\n", apiName)
	b.WriteString(" * No-op stubs compiled into the WASM binary.\n")
	b.WriteString(" */\n\n")
	b.WriteString("#include <stdint.h>\n\n")
	writePlatformServiceStubs(&b, apiName)
	return &OutputFile{Path: "platform_services/web.c", Content: []byte(b.String()), Scaffold: true, ProjectFile: true}
}

// writePlatformServiceStubs emits all platform service functions as no-op stubs.
func writePlatformServiceStubs(b *strings.Builder, apiName string) {
	fmt.Fprintf(b, "void %s_log_sink(int32_t level, const char* tag, const char* message) {\n", apiName)
	b.WriteString("    (void)level;\n")
	b.WriteString("    (void)tag;\n")
	b.WriteString("    (void)message;\n")
	b.WriteString("}\n\n")

	writeResourceStubs(b, apiName)
}

// writeResourceStubs emits resource access function stubs.
func writeResourceStubs(b *strings.Builder, apiName string) {
	fmt.Fprintf(b, "uint32_t %s_resource_count(void) {\n", apiName)
	b.WriteString("    return 0;\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(b, "int32_t %s_resource_name(uint32_t index, char* buffer, uint32_t buffer_size) {\n", apiName)
	b.WriteString("    (void)index;\n")
	b.WriteString("    (void)buffer;\n")
	b.WriteString("    (void)buffer_size;\n")
	b.WriteString("    return -1;\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(b, "int32_t %s_resource_exists(const char* name) {\n", apiName)
	b.WriteString("    (void)name;\n")
	b.WriteString("    return 0;\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(b, "uint32_t %s_resource_size(const char* name) {\n", apiName)
	b.WriteString("    (void)name;\n")
	b.WriteString("    return 0;\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(b, "int32_t %s_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size) {\n", apiName)
	b.WriteString("    (void)name;\n")
	b.WriteString("    (void)buffer;\n")
	b.WriteString("    (void)buffer_size;\n")
	b.WriteString("    return -1;\n")
	b.WriteString("}\n")
}
