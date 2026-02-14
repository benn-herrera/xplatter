package resolver

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TypeKind represents the kind of a FlatBuffers type definition.
type TypeKind int

const (
	TypeKindEnum   TypeKind = iota
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

// ResolvedTypes maps fully-qualified FlatBuffers type names to their kinds.
type ResolvedTypes map[string]TypeKind

var (
	namespacePattern = regexp.MustCompile(`^\s*namespace\s+([A-Za-z][A-Za-z0-9_.]*)\s*;`)
	enumPattern      = regexp.MustCompile(`^\s*enum\s+([A-Z][a-zA-Z0-9]*)\s*`)
	tablePattern     = regexp.MustCompile(`^\s*table\s+([A-Z][a-zA-Z0-9]*)\s*`)
	structPattern    = regexp.MustCompile(`^\s*struct\s+([A-Z][a-zA-Z0-9]*)\s*`)
	unionPattern     = regexp.MustCompile(`^\s*union\s+([A-Z][a-zA-Z0-9]*)\s*`)
)

// ParseFBSFiles parses multiple .fbs files and returns all resolved types.
// Paths are resolved relative to baseDir.
func ParseFBSFiles(baseDir string, fbsPaths []string) (ResolvedTypes, error) {
	types := make(ResolvedTypes)
	for _, p := range fbsPaths {
		fullPath := p
		if !filepath.IsAbs(p) {
			fullPath = filepath.Join(baseDir, p)
		}
		fileTypes, err := ParseFBSFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", p, err)
		}
		for name, kind := range fileTypes {
			if existing, ok := types[name]; ok {
				return nil, fmt.Errorf("duplicate type %s (defined as %s and %s)", name, existing, kind)
			}
			types[name] = kind
		}
	}
	return types, nil
}

// ParseFBSFile parses a single .fbs file and extracts type definitions.
// Returns a map of fully-qualified type names to their kinds.
func ParseFBSFile(path string) (ResolvedTypes, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	types := make(ResolvedTypes)
	namespace := ""

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Strip comments
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}

		if m := namespacePattern.FindStringSubmatch(line); m != nil {
			namespace = m[1]
			continue
		}

		if m := enumPattern.FindStringSubmatch(line); m != nil {
			types[qualifiedName(namespace, m[1])] = TypeKindEnum
		} else if m := tablePattern.FindStringSubmatch(line); m != nil {
			types[qualifiedName(namespace, m[1])] = TypeKindTable
		} else if m := structPattern.FindStringSubmatch(line); m != nil {
			types[qualifiedName(namespace, m[1])] = TypeKindStruct
		} else if m := unionPattern.FindStringSubmatch(line); m != nil {
			types[qualifiedName(namespace, m[1])] = TypeKindUnion
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
