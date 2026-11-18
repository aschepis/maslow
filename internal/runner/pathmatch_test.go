package runner

import "testing"

func TestMatchPath(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"**/.env", ".env", true},
		{"**/.env", "a/.env", true},
		{"**/.env", "a/b/.env", true},
		{"**/.env", "a/b/c/.env", true},
		{"**/.env", "a/.envrc", false},
		{"vendor/**", "vendor/x.go", true},
		{"vendor/**", "vendor/a/b.go", true},
		{"vendor/**", "vendor/a/b/c.go", true},
		{"vendor/**", "vendorx/a.go", false},
		{"vendor/**", "src/vendor/a.go", false},
		{"**/credentials*", "credentials.json", true},
		{"**/credentials*", "a/b/credentials.json", true},
		{"**/credentials*", "a/b/c/credentials", true},
		{"a/**/b", "a/b", true},
		{"a/**/b", "a/x/y/b", true},
		{"a/**/b", "a/x/y/c", false},
		{"docs/*.md", "docs/a.md", true},
		{"docs/*.md", "docs/x/a.md", false},
		{"maslow.yaml", "maslow.yaml", true},
		{"maslow.yaml", "sub/maslow.yaml", false},
		{"**", "any/depth/at/all", true},
	}
	for _, c := range cases {
		got, err := matchPath(c.pattern, c.path)
		if err != nil {
			t.Errorf("matchPath(%q, %q) error: %v", c.pattern, c.path, err)
			continue
		}
		if got != c.want {
			t.Errorf("matchPath(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestMatchPath_InvalidPattern(t *testing.T) {
	if _, err := matchPath("a/[", "a/b"); err == nil {
		t.Error("expected error for malformed pattern")
	}
}
