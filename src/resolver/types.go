package resolver

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// TypeKind represents the kind of a FlatBuffers type definition.
type TypeKind int

const (
	TypeKindEnum TypeKind = iota
	TypeKindTable
	TypeKindStruct
	TypeKindUnion
)

func (k TypeKind) String() string {
	switch k {
	case TypeKindEnum:
		return "enum"
	case TypeKindTable:
		return "table"
	case TypeKindStruct:
		return "struct"
	case TypeKindUnion:
		return "union"
	default:
		return "unknown"
	}
}

// EnumValue represents a single value in a FlatBuffers enum.
type EnumValue struct {
	Name  string
	Value int64
}

// FieldDef represents a single field in a FlatBuffers table or struct.
type FieldDef struct {
	Name string
	Type string // FBS field type: "string", "int32", "float", "[TouchEvent]", etc.
}

// TypeInfo holds full information about a FlatBuffers type definition.
type TypeInfo struct {
	Kind       TypeKind
	BaseType   string      // Enums: underlying type (e.g., "int32")
	EnumValues []EnumValue // Enums only
	Fields     []FieldDef  // Tables/structs only
}

// ResolvedTypes maps fully-qualified FlatBuffers type names to their type info.
type ResolvedTypes map[string]*TypeInfo

var (
	namespacePattern = regexp.MustCompile(`^\s*namespace\s+([A-Za-z][A-Za-z0-9_.]*)\s*;`)
	enumPattern      = regexp.MustCompile(`^\s*enum\s+([A-Z][a-zA-Z0-9]*)\s*:\s*(\w+)`)
	tablePattern     = regexp.MustCompile(`^\s*table\s+([A-Z][a-zA-Z0-9]*)\s*`)
	structPattern    = regexp.MustCompile(`^\s*struct\s+([A-Z][a-zA-Z0-9]*)\s*`)
	unionPattern     = regexp.MustCompile(`^\s*union\s+([A-Z][a-zA-Z0-9]*)\s*`)
	enumValuePattern = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*(?:=\s*(-?\d+))?\s*,?\s*$`)
	fieldPattern     = regexp.MustCompile(`([a-z_][a-zA-Z0-9_]*)\s*:\s*(\S+?)\s*;`)
)

// fbsTypeAlias normalizes FBS type aliases to their canonical form.
func fbsTypeAlias(t string) string {
	switch t {
	case "byte":
		return "int8"
	case "ubyte":
		return "uint8"
	case "short":
		return "int16"
	case "ushort":
		return "uint16"
	case "int":
		return "int32"
	case "uint":
		return "uint32"
	case "long":
		return "int64"
	case "ulong":
		return "uint64"
	case "float":
		return "float32"
	case "double":
		return "float64"
	default:
		return t
	}
}

// ResolveFBSPath resolves a relative .fbs path by searching directories in order.
// Returns the first path that exists. Absolute paths are returned as-is.
func ResolveFBSPath(relPath string, searchDirs []string) (string, error) {
	if filepath.IsAbs(relPath) {
		return relPath, nil
	}
	for _, dir := range searchDirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s not found in search directories: %v", relPath, searchDirs)
}

// ParseFBSFiles parses multiple .fbs files and returns all resolved types.
// Relative paths are resolved by searching directories in order.
func ParseFBSFiles(searchDirs []string, fbsPaths []string) (ResolvedTypes, error) {
	types := make(ResolvedTypes)
	for _, p := range fbsPaths {
		fullPath, err := ResolveFBSPath(p, searchDirs)
		if err != nil {
			return nil, fmt.Errorf("resolving %s: %w", p, err)
		}
		fileTypes, err := ParseFBSFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", p, err)
		}
		for name, info := range fileTypes {
			if existing, ok := types[name]; ok {
				return nil, fmt.Errorf("duplicate type %s (defined as %s and %s)", name, existing.Kind, info.Kind)
			}
			types[name] = info
		}
	}
	return types, nil
}

// ParseFBSFile parses a single .fbs file and extracts type definitions
// including enum values and table/struct fields.
func ParseFBSFile(path string) (ResolvedTypes, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	types := make(ResolvedTypes)
	namespace := ""

	// State for tracking type bodies
	var currentType *TypeInfo
	var currentName string
	var nextEnumValue int64
	braceDepth := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Strip comments
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}

		// Track brace depth for type bodies
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")

		if m := namespacePattern.FindStringSubmatch(line); m != nil {
			namespace = m[1]
			continue
		}

		// Only parse type headers when not inside a type body
		if currentType == nil {
			if m := enumPattern.FindStringSubmatch(line); m != nil {
				currentName = qualifiedName(namespace, m[1])
				currentType = &TypeInfo{
					Kind:     TypeKindEnum,
					BaseType: fbsTypeAlias(m[2]),
				}
				nextEnumValue = 0
				types[currentName] = currentType
				braceDepth = openBraces - closeBraces
				if braceDepth <= 0 {
					currentType = nil
				}
				continue
			} else if m := tablePattern.FindStringSubmatch(line); m != nil {
				currentName = qualifiedName(namespace, m[1])
				currentType = &TypeInfo{Kind: TypeKindTable}
				types[currentName] = currentType
				braceDepth = openBraces - closeBraces
				if braceDepth <= 0 {
					currentType = nil
				}
				continue
			} else if m := structPattern.FindStringSubmatch(line); m != nil {
				currentName = qualifiedName(namespace, m[1])
				currentType = &TypeInfo{Kind: TypeKindStruct}
				types[currentName] = currentType
				braceDepth = openBraces - closeBraces
				if braceDepth <= 0 {
					currentType = nil
				}
				continue
			} else if m := unionPattern.FindStringSubmatch(line); m != nil {
				currentName = qualifiedName(namespace, m[1])
				currentType = &TypeInfo{Kind: TypeKindUnion}
				types[currentName] = currentType
				braceDepth = openBraces - closeBraces
				if braceDepth <= 0 {
					currentType = nil
				}
				continue
			}
		} else {
			// Inside a type body — parse fields/values
			braceDepth += openBraces - closeBraces
			if braceDepth <= 0 {
				currentType = nil
				continue
			}

			switch currentType.Kind {
			case TypeKindEnum:
				if m := enumValuePattern.FindStringSubmatch(line); m != nil {
					if m[2] != "" {
						val, _ := strconv.ParseInt(m[2], 10, 64)
						nextEnumValue = val
					}
					currentType.EnumValues = append(currentType.EnumValues, EnumValue{
						Name:  m[1],
						Value: nextEnumValue,
					})
					nextEnumValue++
				}
			case TypeKindTable, TypeKindStruct:
				for _, m := range fieldPattern.FindAllStringSubmatch(line, -1) {
					currentType.Fields = append(currentType.Fields, FieldDef{
						Name: m[1],
						Type: fbsTypeAlias(m[2]),
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return types, nil
}

func qualifiedName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "." + name
}
