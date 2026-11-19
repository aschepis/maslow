package schema

import (
	"path/filepath"
	"testing"
)

func TestValidate_RetriesFixtures(t *testing.T) {
	cases := []struct {
		fixture string
		valid   bool
	}{
		{"valid/retries.yaml", true},
		{"invalid/retries_negative.yaml", false},
		{"invalid/retries_too_many.yaml", false},
		{"invalid/retries_string.yaml", false},
		{"invalid/retries_above_max_retries.yaml", false},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			result, err := ValidateFile(filepath.Join("..", "..", "testdata", c.fixture))
			if err != nil {
				t.Fatal(err)
			}
			if result.Valid != c.valid {
				t.Errorf("valid = %v, want %v (errors %v)", result.Valid, c.valid, result.Errors)
			}
		})
	}
}

// max_retries raises the cap above the default as well as lowering it.
func TestValidate_MaxRetriesRaisesTheCap(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test
max_retries: 8
checks:
  runner:
    - name: unit
      kind: command
      run: "true"
      retries: 8
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Errorf("expected retries up to max_retries to be valid, got %v", result.Errors)
	}
}

func TestValidateOverlay_AcceptsRetries(t *testing.T) {
	data := []byte(`contracts:
  - name: acceptance
    retries: 2
    scenarios:
      - name: s
        steps:
          - action: cli
            command: "true"
`)
	result, err := ValidateOverlay(data)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Errorf("expected an overlay contract with retries to be valid, got %v", result.Errors)
	}
}
