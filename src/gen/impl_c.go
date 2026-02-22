package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
)

func init() {
	Register("impl_c", func() Generator { return &ImplCGenerator{} })
}

// ImplCGenerator produces a C implementation scaffold:
//   - A stub .c file that includes the generated header and provides
//     TODO-marked function bodies for every exported API function.
type ImplCGenerator struct{}

func (g *ImplCGenerator) Name() string { return "impl_c" }

func (g *ImplCGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name

	scaffoldHeader := GeneratedFileHeaderBlock(ctx, true)
	scaffoldCMakeHeader := GeneratedFileHeader(ctx, "#", true)

	implFile, err := g.generateImplSource(api, apiName)
	if err != nil {
		return nil, fmt.Errorf("generating C impl stub: %w", err)
	}
	implFile.Content = prependHeader(scaffoldHeader, implFile.Content)

	cmakeFile := g.generateCMakeLists(api, apiName)
	cmakeFile.Content = prependHeader(scaffoldCMakeHeader, cmakeFile.Content)

	return []*OutputFile{implFile, cmakeFile}, nil
}

// generateImplSource produces the stub .c implementation file.
func (g *ImplCGenerator) generateImplSource(api *model.APIDefinition, apiName string) (*OutputFile, error) {
	var b strings.Builder

	// Includes
	fmt.Fprintf(&b, "#include \"%s.h\"\n\n", apiName)
	b.WriteString("#include <stdlib.h>\n")
	b.WriteString("#include <string.h>\n")
	b.WriteString("\n")

	// Per-interface function stubs: constructors, auto-destructor, then methods
	for _, iface := range api.Interfaces {
		for i := range iface.Constructors {
			g.writeMethodStub(&b, apiName, iface.Name, &iface.Constructors[i])
			b.WriteString("\n")
		}
		if handleName, ok := iface.ConstructorHandleName(); ok {
			destructor := SyntheticDestructor(handleName)
			g.writeMethodStub(&b, apiName, iface.Name, &destructor)
			b.WriteString("\n")
		}
		for i := range iface.Methods {
			g.writeMethodStub(&b, apiName, iface.Name, &iface.Methods[i])
			b.WriteString("\n")
		}
	}

	return &OutputFile{
		Path:        apiName + "_impl.c",
		Content:     []byte(b.String()),
		Scaffold:    true,
		ProjectFile: true,
	}, nil
}

// generateCMakeLists produces a scaffold CMakeLists.txt for the C implementation (WASM only).
func (g *ImplCGenerator) generateCMakeLists(api *model.APIDefinition, apiName string) *OutputFile {
	projectName := strings.ReplaceAll(apiName, "_", "-")
	exports := ComputeWASMExportsCSV(apiName, api)
	var b strings.Builder

	fmt.Fprintf(&b, `cmake_minimum_required(VERSION 3.15)
project(%[1]s VERSION %[2]s LANGUAGES C)

set(CMAKE_C_STANDARD 17)
set(CMAKE_C_STANDARD_REQUIRED ON)

if(EMSCRIPTEN)
    add_executable(%[3]s
        %[3]s_impl.c
        platform_services/web.c
    )
    set_target_properties(%[3]s PROPERTIES SUFFIX ".wasm")
    target_compile_options(%[3]s PRIVATE
        -std=c17 -Wall -Wextra -fvisibility=hidden
    )
    target_link_options(%[3]s PRIVATE
        "SHELL:--no-entry"
        "SHELL:-s EXPORTED_FUNCTIONS=%[4]s"
        "SHELL:-s STANDALONE_WASM"
        "SHELL:-O2"
    )
    target_include_directories(%[3]s PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}
        ${CMAKE_CURRENT_SOURCE_DIR}/generated
    )
endif()
`, projectName, api.API.Version, apiName, exports)

	return &OutputFile{
		Path:        "CMakeLists.txt",
		Content:     []byte(b.String()),
		Scaffold:    true,
		ProjectFile: true,
	}
}

// writeMethodStub writes a single C function stub body.
func (g *ImplCGenerator) writeMethodStub(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build parameter list
	var params []string
	for _, p := range method.Parameters {
		params = append(params, formatCParam(&p)...)
	}

	// Determine return type and out-parameter
	var returnType string
	switch {
	case hasError && hasReturn:
		returnType = "int32_t"
		params = append(params, COutParamType(method.Returns.Type)+" out_result")
	case hasError && !hasReturn:
		returnType = "int32_t"
	case !hasError && hasReturn:
		returnType = CReturnType(method.Returns.Type)
	default:
		returnType = "void"
	}

	paramStr := strings.Join(params, ", ")
	if paramStr == "" {
		paramStr = "void"
	}

	exportMacro := ExportMacroName(apiName)
	fmt.Fprintf(b, "%s %s %s(%s) {\n", exportMacro, returnType, funcName, paramStr)
	b.WriteString("    // TODO: implement\n")

	switch {
	case hasError:
		b.WriteString("    return 0;\n")
	case hasReturn:
		b.WriteString("    return 0;\n")
	}

	b.WriteString("}\n")
}
