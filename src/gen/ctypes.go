package gen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/benn-herrera/xplatter/model"
	"github.com/benn-herrera/xplatter/resolver"
)

// WriteCTypedefs emits minimal C type definitions (handles, enums, structs, tables)
// suitable for embedding in a cgo preamble or other non-header contexts.
// Unlike the full C header, this omits include guards, export macros, platform
// services, and function declarations.
func WriteCTypedefs(b *strings.Builder, handles []model.HandleDef, resolved resolver.ResolvedTypes) {
	// Handle typedefs
	for _, h := range handles {
		snake := model.HandleToSnake(h.Name)
		fmt.Fprintf(b, "typedef struct %s_s* %s_handle;\n", snake, snake)
	}
	if len(handles) > 0 {
		b.WriteString("\n")
	}

	// FlatBuffer type definitions
	if len(resolved) > 0 {
		writeFBSTypedefs(b, resolved)
	}
}

// writeFBSTypedefs emits C type definitions for all FlatBuffer types.
// Enums first, then structs, then tables.
func writeFBSTypedefs(b *strings.Builder, resolved resolver.ResolvedTypes) {
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

	for _, name := range enumNames {
		info := resolved[name]
		cName := model.FlatBufferCType(name)
		b.WriteString("typedef enum {\n")
		for i, val := range info.EnumValues {
			if i < len(info.EnumValues)-1 {
				fmt.Fprintf(b, "    %s_%s = %d,\n", cName, val.Name, val.Value)
			} else {
				fmt.Fprintf(b, "    %s_%s = %d\n", cName, val.Name, val.Value)
			}
		}
		fmt.Fprintf(b, "} %s;\n\n", cName)
	}

	for _, name := range structNames {
		info := resolved[name]
		cName := model.FlatBufferCType(name)
		fmt.Fprintf(b, "typedef struct %s {\n", cName)
		writeCStructFields(b, info.Fields)
		fmt.Fprintf(b, "} %s;\n\n", cName)
	}

	for _, name := range tableNames {
		info := resolved[name]
		cName := model.FlatBufferCType(name)
		fmt.Fprintf(b, "typedef struct %s {\n", cName)
		writeCStructFields(b, info.Fields)
		fmt.Fprintf(b, "} %s;\n\n", cName)
	}
}
