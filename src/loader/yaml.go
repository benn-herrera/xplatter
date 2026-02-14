package loader

import (
	"fmt"
	"os"

	"github.com/benn-herrera/xplattergy/model"
	"gopkg.in/yaml.v3"
)

// LoadAPIDefinition reads and parses a YAML API definition file.
// It validates the YAML against the JSON Schema before unmarshalling.
func LoadAPIDefinition(path string) (*model.APIDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading API definition: %w", err)
	}

	// First validate against JSON Schema
	if err := ValidateSchema(data); err != nil {
		return nil, fmt.Errorf("schema validation: %w", err)
	}

	var def model.APIDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing API definition: %w", err)
	}

	return &def, nil
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
