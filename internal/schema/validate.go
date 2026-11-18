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
	return validateAgainst(yamlData, "#MAS", "maslow.yaml")
}

// ValidateOverlay checks a spec overlay, which may hold only contracts.
func ValidateOverlay(yamlData []byte) (*ValidationResult, error) {
	return validateAgainst(yamlData, "#Overlay", "overlay")
}

func validateAgainst(yamlData []byte, definition, label string) (*ValidationResult, error) {
	schemaData, err := schemaFS.ReadFile("maslow.cue")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded schema: %w", err)
	}

	ctx := cuecontext.New()

	schemaVal := ctx.CompileBytes(schemaData)
	if schemaVal.Err() != nil {
		return nil, fmt.Errorf("failed to compile CUE schema: %w", schemaVal.Err())
	}

	defType := schemaVal.LookupPath(cue.ParsePath(definition))
	if defType.Err() != nil {
		return nil, fmt.Errorf("failed to find %s in schema: %w", definition, defType.Err())
	}

	err = cueyaml.Validate(yamlData, defType)
	if err != nil {
		result := &ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{Path: label, Message: err.Error()},
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
