package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_ValidMinimal(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test-project
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_MissingMAS(t *testing.T) {
	data := []byte(`project:
  name: test
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid, got valid")
	}
}

func TestValidate_MissingProject(t *testing.T) {
	data := []byte(`mas: "1.0"
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid, got valid")
	}
}

func TestValidate_BadProjectName(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: "123-bad"
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid, got valid")
	}
}

func TestValidate_FullSpec(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "valid", "full.yaml"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateFile_NotFound(t *testing.T) {
	_, err := ValidateFile("/nonexistent/file.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidate_ValidMCPRef(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test
refs:
  - kind: mcp
    path: "@anthropic/mcp-server-github"
    name: github
    description: GitHub API
    transport: stdio
    command: npx
    args: ["-y", "@anthropic/mcp-server-github"]
    env:
      GITHUB_TOKEN: ""
    required: false
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_MCPRefMissingName(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test
refs:
  - kind: mcp
    path: "@anthropic/mcp-server-github"
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid (mcp ref missing required name), got valid")
	}
}

func TestValidate_InvalidRefKind(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test
refs:
  - kind: invalid_kind
    path: something
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid (bad ref kind), got valid")
	}
}

func TestValidate_InvalidCheckKind(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test
checks:
  runner:
    - name: broken
      kind: invalid_kind
      run: "echo hello"
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid, got valid")
	}
}

func TestValidate_CollectionAssertions(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test
contracts:
  - name: collection-test
    scenarios:
      - name: test-arrays
        steps:
          - action: http
            method: GET
            url: "http://example.com/items"
            expect:
              status: 200
              json_path: "$.items"
              min_length: 1
              max_length: 100
              every:
                json_path: "$.id"
              some:
                json_path: "$.featured"
                value: true
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_CollectionExactLength(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test
contracts:
  - name: exact-len
    scenarios:
      - name: test-exact
        steps:
          - action: http
            method: GET
            url: "http://example.com/items"
            expect:
              json_path: "$.items"
              length: 5
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}
