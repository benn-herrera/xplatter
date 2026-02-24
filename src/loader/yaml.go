package loader

import (
	"fmt"
	"os"

	"github.com/benn-herrera/xplatter/model"
	"gopkg.in/yaml.v3"
)

// LoadAPIDefinition reads and parses a YAML API definition file.
// It validates the YAML against the JSON Schema before unmarshalling.
// The returned SourceMap maps JSONPath-style paths to 1-based line numbers.
func LoadAPIDefinition(path string) (*model.APIDefinition, map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading API definition: %w", err)
	}

	// First validate against JSON Schema
	if err := ValidateSchema(data); err != nil {
		return nil, nil, fmt.Errorf("schema validation: %w", err)
	}

	var def model.APIDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, nil, fmt.Errorf("parsing API definition: %w", err)
	}

	return &def, buildSourceMap(data), nil
}

// LoadAPIDefinitionNoValidate reads and parses without schema validation.
// Used internally when schema validation has already been performed.
func LoadAPIDefinitionNoValidate(data []byte) (*model.APIDefinition, error) {
	var def model.APIDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing API definition: %w", err)
	}
	return &def, nil
}

// buildSourceMap parses YAML bytes into a Node tree and returns a map from
// JSONPath-style paths (e.g. "interfaces[0].methods[1].returns.type") to
// 1-based line numbers.
func buildSourceMap(data []byte) map[string]int {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil
	}
	sm := make(map[string]int)
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		walkNode(sm, "", doc.Content[0])
	}
	return sm
}

// walkNode recursively populates sm with path→line entries.
func walkNode(sm map[string]int, path string, node *yaml.Node) {
	if path != "" {
		sm[path] = node.Line
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]
			childPath := keyNode.Value
			if path != "" {
				childPath = path + "." + keyNode.Value
			}
			sm[childPath] = keyNode.Line
			walkNode(sm, childPath, valNode)
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			walkNode(sm, childPath, child)
		}
	}
}
