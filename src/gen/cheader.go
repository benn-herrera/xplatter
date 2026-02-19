package gen

import (
	"fmt"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

func init() {
	Register("cheader", func() Generator { return &CHeaderGenerator{} })
}

// CHeaderGenerator produces the Pure C API header file.
type CHeaderGenerator struct{}

func (g *CHeaderGenerator) Name() string { return "cheader" }

func (g *CHeaderGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name
	guardName := UpperSnakeCase(apiName) + "_H"

	var b strings.Builder

	// Include guard
	fmt.Fprintf(&b, "#ifndef %s\n", guardName)
	fmt.Fprintf(&b, "#define %s\n\n", guardName)

	// Standard includes
	b.WriteString("#include <stdint.h>\n")
	b.WriteString("#include <stdbool.h>\n\n")

	// Symbol visibility export macro
	writeExportMacro(&b, apiName)

	// C++ compatibility
	b.WriteString("#ifdef __cplusplus\n")
	b.WriteString("extern \"C\" {\n")
	b.WriteString("#endif\n\n")

	// Handle typedefs
	if len(api.Handles) > 0 {
		for _, h := range api.Handles {
			snake := model.HandleToSnake(h.Name)
			fmt.Fprintf(&b, "typedef struct %s_s* %s_handle;\n", snake, snake)
		}
		b.WriteString("\n")
	}

	// FlatBuffer type definitions
	if len(ctx.ResolvedTypes) > 0 {
		writeFBSTypedefs(&b, ctx.ResolvedTypes)
	}

	// Platform services
	writePlatformServices(&b, apiName)

	// Interfaces
	exportMacro := ExportMacroName(apiName)
	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "/* %s */\n", iface.Name)
		for _, method := range iface.Methods {
			writeMethodSignature(&b, apiName, iface.Name, &method, exportMacro)
		}
		b.WriteString("\n")
	}

	// Close C++ compatibility
	b.WriteString("#ifdef __cplusplus\n")
	b.WriteString("}\n")
	b.WriteString("#endif\n\n")

	fmt.Fprintf(&b, "#endif\n")

	filename := apiName + ".h"
	return []*OutputFile{
		{Path: filename, Content: []byte(b.String())},
	}, nil
}

func writeExportMacro(b *strings.Builder, apiName string) {
	exportMacro := ExportMacroName(apiName)
	buildMacro := BuildMacroName(apiName)
	b.WriteString("/* Symbol visibility */\n")
	b.WriteString("#if defined(_WIN32) || defined(_WIN64)\n")
	fmt.Fprintf(b, "  #ifdef %s\n", buildMacro)
	fmt.Fprintf(b, "    #define %s __declspec(dllexport)\n", exportMacro)
	b.WriteString("  #else\n")
	fmt.Fprintf(b, "    #define %s __declspec(dllimport)\n", exportMacro)
	b.WriteString("  #endif\n")
	b.WriteString("#elif defined(__GNUC__) || defined(__clang__)\n")
	fmt.Fprintf(b, "  #define %s __attribute__((visibility(\"default\")))\n", exportMacro)
	b.WriteString("#else\n")
	fmt.Fprintf(b, "  #define %s\n", exportMacro)
	b.WriteString("#endif\n\n")
}

func writePlatformServices(b *strings.Builder, apiName string) {
	b.WriteString("/* Platform services — implement these per platform */\n")
	fmt.Fprintf(b, "void %s_log_sink(int32_t level, const char* tag, const char* message);\n", apiName)
	fmt.Fprintf(b, "uint32_t %s_resource_count(void);\n", apiName)
	fmt.Fprintf(b, "int32_t  %s_resource_name(uint32_t index, char* buffer, uint32_t buffer_size);\n", apiName)
	fmt.Fprintf(b, "int32_t  %s_resource_exists(const char* name);\n", apiName)
	fmt.Fprintf(b, "uint32_t %s_resource_size(const char* name);\n", apiName)
	fmt.Fprintf(b, "int32_t  %s_resource_read(const char* name, uint8_t* buffer, uint32_t buffer_size);\n", apiName)
	b.WriteString("\n")
}

func writeMethodSignature(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef, exportMacro string) {
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
		// Fallible with return: returns error code, return value as out-param
		returnType = "int32_t"
		params = append(params, COutParamType(method.Returns.Type)+" out_result")
	case hasError && !hasReturn:
		// Fallible without return: returns error code
		returnType = "int32_t"
	case !hasError && hasReturn:
		// Infallible with return: returns the value directly
		returnType = CReturnType(method.Returns.Type)
	default:
		// Infallible without return: void
		returnType = "void"
	}

	// Format the function signature
	paramStr := strings.Join(params, ", ")
	if paramStr == "" {
		paramStr = "void"
	}

	// Decide formatting: single line or multi-line
	sig := fmt.Sprintf("%s %s %s(%s)", exportMacro, returnType, funcName, paramStr)
	if len(sig) > 80 {
		// Multi-line format
		fmt.Fprintf(b, "%s %s %s(\n", exportMacro, returnType, funcName)
		for i, p := range params {
			if i < len(params)-1 {
				fmt.Fprintf(b, "    %s,\n", p)
			} else {
				fmt.Fprintf(b, "    %s);\n", p)
			}
		}
	} else {
		fmt.Fprintf(b, "%s;\n", sig)
	}
}

// formatCParam formats a parameter as one or more C parameter strings.
// buffer<T> expands to two parameters (data pointer + length).
func formatCParam(p *model.ParameterDef) []string {
	if model.IsString(p.Type) {
		return []string{"const char* " + p.Name}
	}

	if elemType, ok := model.IsBuffer(p.Type); ok {
		cType := model.PrimitiveCType(elemType)
		var ptrType string
		if p.Transfer == "ref_mut" {
			ptrType = cType + "*"
		} else {
			ptrType = "const " + cType + "*"
		}
		return []string{
			ptrType + " " + p.Name,
			"uint32_t " + p.Name + "_len",
		}
	}

	if handleName, ok := model.IsHandle(p.Type); ok {
		return []string{HandleTypedefName(handleName) + " " + p.Name}
	}

	if model.IsPrimitive(p.Type) {
		return []string{model.PrimitiveCType(p.Type) + " " + p.Name}
	}

	// FlatBuffer type
	cType := model.FlatBufferCType(p.Type)
	transfer := p.Transfer
	if transfer == "ref_mut" {
		return []string{cType + "* " + p.Name}
	}
	if transfer == "ref" {
		return []string{"const " + cType + "* " + p.Name}
	}
	return []string{cType + " " + p.Name}
}

// writeCStructFields writes C struct field declarations for FBS fields.
func writeCStructFields(b *strings.Builder, fields []resolver.FieldDef) {
	for _, f := range fields {
		cType, extraField := fbsFieldToCType(f)
		fmt.Fprintf(b, "    %s %s;\n", cType, f.Name)
		if extraField != "" {
			fmt.Fprintf(b, "    %s;\n", extraField)
		}
	}
}

// fbsFieldToCType maps an FBS field type to a C type.
// Returns the C type and an optional extra field declaration (for vectors).
func fbsFieldToCType(f resolver.FieldDef) (cType string, extraField string) {
	t := f.Type
	switch t {
	case "string":
		return "const char*", ""
	case "bool":
		return "bool", ""
	case "int8":
		return "int8_t", ""
	case "uint8":
		return "uint8_t", ""
	case "int16":
		return "int16_t", ""
	case "uint16":
		return "uint16_t", ""
	case "int32":
		return "int32_t", ""
	case "uint32":
		return "uint32_t", ""
	case "int64":
		return "int64_t", ""
	case "uint64":
		return "uint64_t", ""
	case "float32":
		return "float", ""
	case "float64":
		return "double", ""
	}
	// Vector type: [T]
	if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
		elemType := t[1 : len(t)-1]
		elemCType, _ := fbsFieldToCType(resolver.FieldDef{Name: f.Name, Type: elemType})
		return "const " + elemCType + "*", fmt.Sprintf("uint32_t %s_count", f.Name)
	}
	// FlatBuffer type reference — use pointer for tables, value for structs
	return model.FlatBufferCType(t), ""
}
