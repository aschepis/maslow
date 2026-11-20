package schema

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_MultipartFixture(t *testing.T) {
	result, err := ValidateFile(filepath.Join("..", "..", "testdata", "valid", "http_multipart.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Fatalf("expected all three part kinds to be valid, got %v", result.Errors)
	}
}

// An unknown multipart key fails too, so each case names the constraint it breaks.
func TestValidate_MultipartInvalidFixtures(t *testing.T) {
	cases := []struct {
		fixture string
		want    string
	}{
		{"multipart_absolute_file.yaml", `multipart.0.file: invalid value "/etc/passwd"`},
		{"multipart_parent_segment.yaml", `multipart.0.file: invalid value "../secret"`},
		{"multipart_value_and_file.yaml", "2 matched, expected 1"},
		{"multipart_missing_field.yaml", "multipart.0.field: field is required"},
		{"multipart_content_without_filename.yaml", "multipart.0.filename: field is required"},
		{"multipart_with_body.yaml", "2 matched, expected <=1"},
	}
	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			result, err := ValidateFile(filepath.Join("..", "..", "testdata", "invalid", c.fixture))
			if err != nil {
				t.Fatal(err)
			}
			if result.Valid {
				t.Fatal("expected invalid, got valid")
			}
			if msg := result.Errors[0].Message; !strings.Contains(msg, c.want) {
				t.Errorf("expected error to contain %q, got %s", c.want, msg)
			}
		})
	}
}

func TestValidate_MultipartValueRejectsFileMetadata(t *testing.T) {
	data := []byte(`mas: "1.0"
project:
  name: test
contracts:
  - name: c
    scenarios:
      - name: s
        steps:
          - action: http
            method: POST
            url: http://localhost/upload
            multipart:
              - field: title
                value: "a title"
                filename: title.txt
`)
	result, err := Validate(data)
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid {
		t.Fatal("expected a text part with a filename to be invalid")
	}
	if msg := result.Errors[0].Message; !strings.Contains(msg, "multipart.0") || !strings.Contains(msg, "1 matched, expected 0") {
		t.Errorf("expected the error to reject the part's filename, got %s", msg)
	}
}
