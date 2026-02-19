package gen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

func init() {
	Register("impl_rust", func() Generator { return &RustImplGenerator{} })
}

// RustImplGenerator produces a complete Rust crate:
// Cargo.toml, src/lib.rs, trait definitions, C ABI shim, stub implementation,
// and type definitions from FlatBuffer schemas.
type RustImplGenerator struct{}

func (g *RustImplGenerator) Name() string { return "impl_rust" }

func (g *RustImplGenerator) Generate(ctx *Context) ([]*OutputFile, error) {
	api := ctx.API
	apiName := api.API.Name
	hasTypes := len(ctx.ResolvedTypes) > 0

	genHeader := GeneratedFileHeader(ctx, "//", false)
	scaffoldHeader := GeneratedFileHeader(ctx, "//", true)
	scaffoldTomlHeader := GeneratedFileHeader(ctx, "#", true)

	traitFile, err := g.generateTrait(api, apiName, hasTypes)
	if err != nil {
		return nil, fmt.Errorf("generating trait file: %w", err)
	}
	traitFile.Content = prependHeader(genHeader, traitFile.Content)

	ffiFile, err := g.generateFFI(api, apiName, hasTypes)
	if err != nil {
		return nil, fmt.Errorf("generating FFI file: %w", err)
	}
	ffiFile.Content = prependHeader(genHeader, ffiFile.Content)

	implFile, err := g.generateImpl(api, apiName, hasTypes)
	if err != nil {
		return nil, fmt.Errorf("generating impl file: %w", err)
	}
	implFile.Content = prependHeader(scaffoldHeader, implFile.Content)

	files := []*OutputFile{traitFile, ffiFile, implFile}

	if hasTypes {
		typesFile := g.generateTypes(ctx.ResolvedTypes, apiName)
		typesFile.Content = prependHeader(genHeader, typesFile.Content)
		files = append(files, typesFile)
	}

	// Package metadata (scaffold — preserved across regeneration)
	cargoToml := g.generateCargoToml(api, apiName)
	cargoToml.Content = prependHeader(scaffoldTomlHeader, cargoToml.Content)
	files = append(files, cargoToml)

	libRs := g.generateLibRs(apiName, hasTypes)
	libRs.Content = prependHeader(scaffoldHeader, libRs.Content)
	files = append(files, libRs)

	return files, nil
}

// generateCargoToml produces the Cargo.toml package manifest.
func (g *RustImplGenerator) generateCargoToml(api *model.APIDefinition, apiName string) *OutputFile {
	var b strings.Builder
	fmt.Fprintf(&b, "[package]\n")
	fmt.Fprintf(&b, "name = %q\n", apiName)
	fmt.Fprintf(&b, "version = %q\n", api.API.Version)
	fmt.Fprintf(&b, "edition = \"2021\"\n")
	b.WriteString("\n")
	fmt.Fprintf(&b, "[lib]\n")
	fmt.Fprintf(&b, "crate-type = [\"cdylib\", \"staticlib\", \"rlib\"]\n")

	return &OutputFile{
		Path:        "Cargo.toml",
		Content:     []byte(b.String()),
		Scaffold:    true,
		ProjectFile: true,
	}
}

// generateLibRs produces the src/lib.rs entry point with module declarations.
// Generated (non-scaffold) modules use #[path] to reference files in ../generated/.
func (g *RustImplGenerator) generateLibRs(apiName string, hasTypes bool) *OutputFile {
	var b strings.Builder
	if hasTypes {
		fmt.Fprintf(&b, "#[path = \"../generated/%s_types.rs\"]\n", apiName)
		fmt.Fprintf(&b, "pub mod %s_types;\n", apiName)
	}
	fmt.Fprintf(&b, "#[path = \"../generated/%s_trait.rs\"]\n", apiName)
	fmt.Fprintf(&b, "pub mod %s_trait;\n", apiName)
	fmt.Fprintf(&b, "#[path = \"../generated/%s_ffi.rs\"]\n", apiName)
	fmt.Fprintf(&b, "pub mod %s_ffi;\n", apiName)
	fmt.Fprintf(&b, "pub mod %s_impl;\n", apiName)

	return &OutputFile{
		Path:        "src/lib.rs",
		Content:     []byte(b.String()),
		Scaffold:    true,
		ProjectFile: true,
	}
}

// generateTrait produces the trait definition file.
func (g *RustImplGenerator) generateTrait(api *model.APIDefinition, apiName string, hasTypes bool) (*OutputFile, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "use std::ffi::c_void;\n")
	if hasTypes {
		fmt.Fprintf(&b, "use crate::%s_types::*;\n", apiName)
	}
	b.WriteString("\n")

	for _, iface := range api.Interfaces {
		traitName := ToPascalCase(iface.Name)
		fmt.Fprintf(&b, "/// %s interface methods.\n", traitName)
		fmt.Fprintf(&b, "pub trait %s {\n", traitName)

		for _, method := range iface.Methods {
			writeTraitMethod(&b, &method)
		}

		b.WriteString("}\n\n")
	}

	return &OutputFile{
		Path:    apiName + "_trait.rs",
		Content: []byte(b.String()),
	}, nil
}

// generateFFI produces the C ABI shim file.
func (g *RustImplGenerator) generateFFI(api *model.APIDefinition, apiName string, hasTypes bool) (*OutputFile, error) {
	var b strings.Builder

	b.WriteString("use std::ffi::{c_void, CStr};\n")
	b.WriteString("use std::os::raw::c_char;\n")
	if hasTypes {
		fmt.Fprintf(&b, "use crate::%s_types::*;\n", apiName)
	}
	fmt.Fprintf(&b, "use crate::%s_trait::*;\n", apiName)
	fmt.Fprintf(&b, "use crate::%s_impl::*;\n\n", apiName)

	for _, iface := range api.Interfaces {
		fmt.Fprintf(&b, "// %s\n", iface.Name)

		for _, method := range iface.Methods {
			writeFFIFunction(&b, apiName, iface.Name, &method)
			b.WriteString("\n")
		}
	}

	return &OutputFile{
		Path:    apiName + "_ffi.rs",
		Content: []byte(b.String()),
	}, nil
}

// generateImpl produces the stub implementation file (scaffold — not overwritten).
func (g *RustImplGenerator) generateImpl(api *model.APIDefinition, apiName string, hasTypes bool) (*OutputFile, error) {
	var b strings.Builder

	b.WriteString("use std::ffi::c_void;\n")
	if hasTypes {
		fmt.Fprintf(&b, "use crate::%s_types::*;\n", apiName)
	}
	fmt.Fprintf(&b, "use crate::%s_trait::*;\n\n", apiName)

	// Collect all trait names this struct implements.
	b.WriteString("/// Main implementation struct.\n")
	b.WriteString("pub struct Impl;\n\n")

	for _, iface := range api.Interfaces {
		traitName := ToPascalCase(iface.Name)
		fmt.Fprintf(&b, "impl %s for Impl {\n", traitName)

		for i, method := range iface.Methods {
			writeImplMethod(&b, &method)
			if i < len(iface.Methods)-1 {
				b.WriteString("\n")
			}
		}

		b.WriteString("}\n\n")
	}

	return &OutputFile{
		Path:        "src/" + apiName + "_impl.rs",
		Content:     []byte(b.String()),
		Scaffold:    true,
		ProjectFile: true,
	}, nil
}

// generateTypes produces the Rust type definitions file from FBS schemas.
func (g *RustImplGenerator) generateTypes(resolved resolver.ResolvedTypes, apiName string) *OutputFile {
	var b strings.Builder

	b.WriteString("use std::os::raw::c_char;\n\n")

	// Collect and sort type names
	var enumNames, structNames, tableNames []string
	for name, info := range resolved {
		switch info.Kind {
		case resolver.TypeKindEnum:
			enumNames = append(enumNames, name)
		case resolver.TypeKindStruct:
			structNames = append(structNames, name)
		case resolver.TypeKindTable:
			tableNames = append(tableNames, name)
		}
	}
	sort.Strings(enumNames)
	sort.Strings(structNames)
	sort.Strings(tableNames)

	// Enums
	for _, name := range enumNames {
		info := resolved[name]
		rustName := rustFlatBufferType(name)
		baseType := rustPrimitiveType(info.BaseType)
		fmt.Fprintf(&b, "#[repr(%s)]\n", baseType)
		b.WriteString("#[derive(Debug, Clone, Copy, PartialEq, Eq)]\n")
		fmt.Fprintf(&b, "pub enum %s {\n", rustName)
		for _, val := range info.EnumValues {
			fmt.Fprintf(&b, "    %s = %d,\n", val.Name, val.Value)
		}
		b.WriteString("}\n\n")
	}

	// Structs
	for _, name := range structNames {
		info := resolved[name]
		rustName := rustFlatBufferType(name)
		b.WriteString("#[repr(C)]\n")
		b.WriteString("#[derive(Debug, Clone, Copy)]\n")
		fmt.Fprintf(&b, "pub struct %s {\n", rustName)
		for _, f := range info.Fields {
			fmt.Fprintf(&b, "    pub %s: %s,\n", f.Name, fbsFieldToRustType(f.Type))
		}
		b.WriteString("}\n\n")
	}

	// Tables
	for _, name := range tableNames {
		info := resolved[name]
		rustName := rustFlatBufferType(name)
		b.WriteString("#[repr(C)]\n")
		b.WriteString("#[derive(Debug)]\n")
		fmt.Fprintf(&b, "pub struct %s {\n", rustName)
		for _, f := range info.Fields {
			fmt.Fprintf(&b, "    pub %s: %s,\n", f.Name, fbsFieldToRustType(f.Type))
		}
		b.WriteString("}\n\n")
	}

	return &OutputFile{
		Path:    apiName + "_types.rs",
		Content: []byte(b.String()),
	}
}

// fbsFieldToRustType maps an FBS field type to a Rust type.
func fbsFieldToRustType(t string) string {
	switch t {
	case "string":
		return "*const c_char"
	case "bool":
		return "bool"
	case "int8":
		return "i8"
	case "uint8":
		return "u8"
	case "int16":
		return "i16"
	case "uint16":
		return "u16"
	case "int32":
		return "i32"
	case "uint32":
		return "u32"
	case "int64":
		return "i64"
	case "uint64":
		return "u64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	}
	// Vector type: [T]
	if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
		// Vectors are not directly representable in repr(C); use pointer + count
		return "*const " + fbsFieldToRustType(t[1:len(t)-1])
	}
	// FlatBuffer type reference
	return rustFlatBufferType(t)
}

// --- Trait helpers ---

// writeTraitMethod writes a single trait method signature.
func writeTraitMethod(b *strings.Builder, method *model.MethodDef) {
	if method.Description != "" {
		fmt.Fprintf(b, "    /// %s\n", method.Description)
	}

	params := rustTraitParams(method.Parameters)
	retType := rustTraitReturnType(method)

	if len(params) > 0 {
		fmt.Fprintf(b, "    fn %s(&self, %s)%s;\n", method.Name, strings.Join(params, ", "), retType)
	} else {
		fmt.Fprintf(b, "    fn %s(&self)%s;\n", method.Name, retType)
	}
}

// rustTraitParams builds the Rust trait parameter list.
func rustTraitParams(parameters []model.ParameterDef) []string {
	var params []string
	for _, p := range parameters {
		params = append(params, rustTraitParam(&p))
	}
	return params
}

// rustTraitParam maps a single parameter to a Rust trait parameter string.
func rustTraitParam(p *model.ParameterDef) string {
	return fmt.Sprintf("%s: %s", p.Name, rustTraitParamType(p.Type, p.Transfer))
}

// rustTraitParamType returns the Rust type for a trait parameter.
func rustTraitParamType(paramType string, transfer string) string {
	if model.IsString(paramType) {
		return "&str"
	}

	if elemType, ok := model.IsBuffer(paramType); ok {
		rustElem := rustPrimitiveType(elemType)
		if transfer == "ref_mut" {
			return "&mut [" + rustElem + "]"
		}
		return "&[" + rustElem + "]"
	}

	if _, ok := model.IsHandle(paramType); ok {
		return "*mut c_void"
	}

	if model.IsPrimitive(paramType) {
		return rustPrimitiveType(paramType)
	}

	// FlatBuffer type — passed as reference
	rustType := rustFlatBufferType(paramType)
	if transfer == "ref_mut" {
		return "&mut " + rustType
	}
	return "&" + rustType
}

// rustTraitReturnType returns the full " -> T" suffix for a trait method,
// or empty string for void.
func rustTraitReturnType(method *model.MethodDef) string {
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	switch {
	case hasError && hasReturn:
		inner := rustReturnValueType(method.Returns.Type)
		errType := rustFlatBufferType(method.Error)
		return fmt.Sprintf(" -> Result<%s, %s>", inner, errType)
	case hasError && !hasReturn:
		errType := rustFlatBufferType(method.Error)
		return fmt.Sprintf(" -> Result<(), %s>", errType)
	case !hasError && hasReturn:
		inner := rustReturnValueType(method.Returns.Type)
		return fmt.Sprintf(" -> %s", inner)
	default:
		return ""
	}
}

// --- FFI helpers ---

// writeFFIFunction writes a single #[no_mangle] extern "C" shim function.
func writeFFIFunction(b *strings.Builder, apiName, ifaceName string, method *model.MethodDef) {
	funcName := CABIFunctionName(apiName, ifaceName, method.Name)
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Build C ABI parameter list
	var params []string
	for _, p := range method.Parameters {
		params = append(params, ffiParams(&p)...)
	}

	// Determine return type and out parameter
	var cReturnType string
	switch {
	case hasError && hasReturn:
		cReturnType = "i32"
		params = append(params, fmt.Sprintf("out_result: %s", ffiOutParamType(method.Returns.Type)))
	case hasError && !hasReturn:
		cReturnType = "i32"
	case !hasError && hasReturn:
		cReturnType = ffiReturnType(method.Returns.Type)
	default:
		cReturnType = ""
	}

	retSuffix := ""
	if cReturnType != "" {
		retSuffix = " -> " + cReturnType
	}

	fmt.Fprintf(b, "#[no_mangle]\n")
	fmt.Fprintf(b, "pub unsafe extern \"C\" fn %s(%s)%s {\n", funcName, strings.Join(params, ", "), retSuffix)

	// Body: convert parameters and delegate to trait method
	writeFFIBody(b, method, ifaceName)

	b.WriteString("}\n")
}

// ffiParams returns the FFI parameter strings for a single API parameter.
func ffiParams(p *model.ParameterDef) []string {
	if model.IsString(p.Type) {
		return []string{fmt.Sprintf("%s: *const c_char", p.Name)}
	}

	if elemType, ok := model.IsBuffer(p.Type); ok {
		rustElem := rustPrimitiveType(elemType)
		var ptrType string
		if p.Transfer == "ref_mut" {
			ptrType = "*mut " + rustElem
		} else {
			ptrType = "*const " + rustElem
		}
		return []string{
			fmt.Sprintf("%s: %s", p.Name, ptrType),
			fmt.Sprintf("%s_len: u32", p.Name),
		}
	}

	if _, ok := model.IsHandle(p.Type); ok {
		return []string{fmt.Sprintf("%s: *mut c_void", p.Name)}
	}

	if model.IsPrimitive(p.Type) {
		return []string{fmt.Sprintf("%s: %s", p.Name, rustPrimitiveType(p.Type))}
	}

	// FlatBuffer type
	rustType := rustFlatBufferType(p.Type)
	if p.Transfer == "ref_mut" {
		return []string{fmt.Sprintf("%s: *mut %s", p.Name, rustType)}
	}
	return []string{fmt.Sprintf("%s: *const %s", p.Name, rustType)}
}

// ffiReturnType returns the FFI return type for a value type.
func ffiReturnType(retType string) string {
	if _, ok := model.IsHandle(retType); ok {
		return "*mut c_void"
	}
	if model.IsPrimitive(retType) {
		return rustPrimitiveType(retType)
	}
	return rustFlatBufferType(retType)
}

// ffiOutParamType returns the FFI out-parameter pointer type.
func ffiOutParamType(retType string) string {
	return "*mut " + ffiReturnType(retType)
}

// writeFFIBody writes the function body of an FFI shim.
func writeFFIBody(b *strings.Builder, method *model.MethodDef, ifaceName string) {
	hasError := method.Error != ""
	hasReturn := method.Returns != nil

	// Convert parameters
	for _, p := range method.Parameters {
		writeParamConversion(b, &p)
	}

	// Build the call expression
	traitName := ToPascalCase(ifaceName)
	var callArgs []string
	for _, p := range method.Parameters {
		callArgs = append(callArgs, rustConvertedArgName(&p))
	}
	call := fmt.Sprintf("%s::%s(&Impl, %s)", traitName, method.Name, strings.Join(callArgs, ", "))
	if len(method.Parameters) == 0 {
		call = fmt.Sprintf("%s::%s(&Impl)", traitName, method.Name)
	}

	switch {
	case hasError && hasReturn:
		// Match on Result, write out_result on success, return error code
		fmt.Fprintf(b, "    match %s {\n", call)
		b.WriteString("        Ok(val) => {\n")
		b.WriteString("            *out_result = val;\n")
		b.WriteString("            0\n")
		b.WriteString("        }\n")
		b.WriteString("        Err(e) => e as i32,\n")
		b.WriteString("    }\n")
	case hasError && !hasReturn:
		fmt.Fprintf(b, "    match %s {\n", call)
		b.WriteString("        Ok(()) => 0,\n")
		b.WriteString("        Err(e) => e as i32,\n")
		b.WriteString("    }\n")
	case !hasError && hasReturn:
		fmt.Fprintf(b, "    %s\n", call)
	default:
		fmt.Fprintf(b, "    %s;\n", call)
	}
}

// writeParamConversion writes the unsafe conversion of a single C parameter to its Rust equivalent.
func writeParamConversion(b *strings.Builder, p *model.ParameterDef) {
	if model.IsString(p.Type) {
		fmt.Fprintf(b, "    let %s = CStr::from_ptr(%s).to_str().expect(\"invalid UTF-8\");\n", p.Name, p.Name)
		return
	}

	if _, ok := model.IsBuffer(p.Type); ok {
		if p.Transfer == "ref_mut" {
			fmt.Fprintf(b, "    let %s = std::slice::from_raw_parts_mut(%s, %s_len as usize);\n", p.Name, p.Name, p.Name)
		} else {
			fmt.Fprintf(b, "    let %s = std::slice::from_raw_parts(%s, %s_len as usize);\n", p.Name, p.Name, p.Name)
		}
		return
	}

	if _, ok := model.IsHandle(p.Type); ok {
		// Handle passed through as raw pointer — no conversion needed.
		return
	}

	if model.IsPrimitive(p.Type) {
		// Primitive passed directly — no conversion needed.
		return
	}

	// FlatBuffer type — dereference the pointer to a reference.
	if p.Transfer == "ref_mut" {
		fmt.Fprintf(b, "    let %s = &mut *%s;\n", p.Name, p.Name)
	} else {
		fmt.Fprintf(b, "    let %s = &*%s;\n", p.Name, p.Name)
	}
}

// rustConvertedArgName returns the name to use for a converted parameter in the call.
func rustConvertedArgName(p *model.ParameterDef) string {
	// All conversions shadow the original name, so just return the name.
	return p.Name
}

// --- Impl helpers ---

// writeImplMethod writes a single stub trait method implementation.
func writeImplMethod(b *strings.Builder, method *model.MethodDef) {
	params := rustTraitParams(method.Parameters)
	retType := rustTraitReturnType(method)

	if len(params) > 0 {
		fmt.Fprintf(b, "    fn %s(&self, %s)%s {\n", method.Name, strings.Join(params, ", "), retType)
	} else {
		fmt.Fprintf(b, "    fn %s(&self)%s {\n", method.Name, retType)
	}
	fmt.Fprintf(b, "        // TODO: implement %s\n", method.Name)
	b.WriteString("        todo!()\n")
	b.WriteString("    }\n")
}

// --- Type mapping helpers ---

// rustPrimitiveType maps an xplatter primitive type to its Rust equivalent.
func rustPrimitiveType(t string) string {
	switch t {
	case "int8":
		return "i8"
	case "int16":
		return "i16"
	case "int32":
		return "i32"
	case "int64":
		return "i64"
	case "uint8":
		return "u8"
	case "uint16":
		return "u16"
	case "uint32":
		return "u32"
	case "uint64":
		return "u64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	case "bool":
		return "bool"
	default:
		return t
	}
}

// rustReturnValueType returns the Rust type for a return value.
func rustReturnValueType(retType string) string {
	if _, ok := model.IsHandle(retType); ok {
		return "*mut c_void"
	}
	if model.IsPrimitive(retType) {
		return rustPrimitiveType(retType)
	}
	return rustFlatBufferType(retType)
}

// rustFlatBufferType converts a FlatBuffers type to a Rust type name.
// e.g., "Common.ErrorCode" -> "CommonErrorCode"
func rustFlatBufferType(t string) string {
	// In Rust, we use PascalCase with no separator for FlatBuffer namespaced types.
	return strings.ReplaceAll(t, ".", "")
}
