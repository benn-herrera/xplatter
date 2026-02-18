package loader

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// schemaJSON is the embedded JSON Schema for API definition validation.
// It is loaded from the project's docs/api_definition_schema.json at init time.
//
//go:generate cp ../../docs/api_definition_schema.json schema_embedded.json
var schemaJSON = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://xplatter.dev/schemas/api-definition/v1",
  "title": "xplatter API Definition",
  "description": "Schema for xplatter API definition YAML files.",
  "type": "object",
  "required": ["api", "flatbuffers", "interfaces"],
  "additionalProperties": false,
  "properties": {
    "api": { "$ref": "#/$defs/api_metadata" },
    "flatbuffers": {
      "type": "array",
      "items": { "type": "string", "pattern": "\\.fbs$" },
      "minItems": 1
    },
    "handles": {
      "type": "array",
      "items": { "$ref": "#/$defs/handle_definition" }
    },
    "interfaces": {
      "type": "array",
      "items": { "$ref": "#/$defs/interface_definition" },
      "minItems": 1
    }
  },
  "$defs": {
    "api_metadata": {
      "type": "object",
      "required": ["name", "version", "impl_lang"],
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string", "pattern": "^[a-z][a-z0-9_]*$" },
        "version": { "type": "string", "pattern": "^\\d+\\.\\d+\\.\\d+$" },
        "description": { "type": "string" },
        "impl_lang": { "type": "string", "enum": ["cpp", "rust", "go", "c"] },
        "targets": {
          "type": "array",
          "items": { "type": "string", "enum": ["android", "ios", "web", "windows", "macos", "linux"] },
          "minItems": 1,
          "uniqueItems": true
        }
      }
    },
    "handle_definition": {
      "type": "object",
      "required": ["name"],
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string", "pattern": "^[A-Z][a-zA-Z0-9]*$" },
        "description": { "type": "string" }
      }
    },
    "interface_definition": {
      "type": "object",
      "required": ["name", "methods"],
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string", "pattern": "^[a-z][a-z0-9_]*$" },
        "description": { "type": "string" },
        "methods": {
          "type": "array",
          "items": { "$ref": "#/$defs/method_definition" },
          "minItems": 1
        }
      }
    },
    "method_definition": {
      "type": "object",
      "required": ["name"],
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string", "pattern": "^[a-z][a-z0-9_]*$" },
        "description": { "type": "string" },
        "parameters": {
          "type": "array",
          "items": { "$ref": "#/$defs/parameter_definition" }
        },
        "returns": { "$ref": "#/$defs/return_definition" },
        "error": { "type": "string", "pattern": "^[A-Z][a-zA-Z0-9]*(\\.[A-Z][a-zA-Z0-9]*)*$" }
      }
    },
    "parameter_definition": {
      "type": "object",
      "required": ["name", "type"],
      "additionalProperties": false,
      "properties": {
        "name": { "type": "string", "pattern": "^[a-z][a-z0-9_]*$" },
        "type": {
          "type": "string",
          "pattern": "^(int8|int16|int32|int64|uint8|uint16|uint32|uint64|float32|float64|bool|string|buffer<(int8|int16|int32|int64|uint8|uint16|uint32|uint64|float32|float64)>|handle:[A-Z][a-zA-Z0-9]*|[A-Z][a-zA-Z0-9]*(\\.[A-Z][a-zA-Z0-9]*)*)$"
        },
        "transfer": { "type": "string", "enum": ["value", "ref", "ref_mut"] },
        "description": { "type": "string" }
      }
    },
    "return_definition": {
      "type": "object",
      "required": ["type"],
      "additionalProperties": false,
      "properties": {
        "type": {
          "type": "string",
          "pattern": "^(int8|int16|int32|int64|uint8|uint16|uint32|uint64|float32|float64|bool|handle:[A-Z][a-zA-Z0-9]*|[A-Z][a-zA-Z0-9]*(\\.[A-Z][a-zA-Z0-9]*)*)$"
        },
        "description": { "type": "string" }
      }
    }
  }
}`

var compiledSchema *jsonschema.Schema

func init() {
	// Decode the schema JSON into a generic value first
	var schemaDoc interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		panic(fmt.Sprintf("failed to decode schema JSON: %v", err))
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", schemaDoc); err != nil {
		panic(fmt.Sprintf("failed to add schema resource: %v", err))
	}
	var err error
	compiledSchema, err = c.Compile("schema.json")
	if err != nil {
		panic(fmt.Sprintf("failed to compile schema: %v", err))
	}
}

// ValidateSchema validates raw YAML bytes against the API definition JSON Schema.
func ValidateSchema(yamlData []byte) error {
	// Parse YAML into a generic structure
	var raw interface{}
	if err := yaml.Unmarshal(yamlData, &raw); err != nil {
		return fmt.Errorf("parsing YAML: %w", err)
	}

	// Convert to JSON-compatible types (yaml.v3 uses map[string]interface{} already)
	converted := convertYAMLToJSON(raw)

	err := compiledSchema.Validate(converted)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// convertYAMLToJSON converts YAML-parsed values to JSON-compatible types.
// yaml.v3 parses maps as map[string]interface{} which is already JSON-compatible,
// but we need to handle nested maps recursively.
func convertYAMLToJSON(v interface{}) interface{} {
	switch v := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			result[k] = convertYAMLToJSON(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = convertYAMLToJSON(val)
		}
		return result
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return v
	}
}

// ValidateSchemaJSON validates a JSON string against the schema (for testing).
func ValidateSchemaJSON(jsonData []byte) error {
	var raw interface{}
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	err := compiledSchema.Validate(raw)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}
