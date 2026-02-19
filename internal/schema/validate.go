// Package schema handles CUE schema validation of maslow.yaml files.
package schema

import (
	"embed"
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueyaml "cuelang.org/go/encoding/yaml"
)

//go:embed maslow.cue
var schemaFS embed.FS

// ValidationError represents a schema validation failure.
type ValidationError struct {
	Path    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationResult holds the outcome of schema validation.
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// Validate checks a YAML byte slice against the embedded CUE schema.
func Validate(yamlData []byte) (*ValidationResult, error) {
	schemaData, err := schemaFS.ReadFile("maslow.cue")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded schema: %w", err)
	}

	ctx := cuecontext.New()

	schemaVal := ctx.CompileBytes(schemaData)
	if schemaVal.Err() != nil {
		return nil, fmt.Errorf("failed to compile CUE schema: %w", schemaVal.Err())
	}

	masType := schemaVal.LookupPath(cue.ParsePath("#MAS"))
	if masType.Err() != nil {
		return nil, fmt.Errorf("failed to find #MAS in schema: %w", masType.Err())
	}

	err = cueyaml.Validate(yamlData, masType)
	if err != nil {
		result := &ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Path: "maslow.yaml", Message: err.Error()},
			},
		}
		return result, nil
	}

	return &ValidationResult{Valid: true}, nil
}

// ValidateFile reads a file and validates it against the CUE schema.
func ValidateFile(path string) (*ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return Validate(data)
}
