package runner

import (
	"testing"

	"github.com/aschepis/maslow-agentic/internal/spec"
)

func TestRunChecks_PassingCommand(t *testing.T) {
	checks := []spec.CheckRunner{
		{Name: "echo-test", Kind: "command", Run: "echo hello"},
	}
	results := RunChecks(checks)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}
}

func TestRunChecks_FailingCommand(t *testing.T) {
	checks := []spec.CheckRunner{
		{Name: "fail-test", Kind: "command", Run: "false"},
	}
	results := RunChecks(checks)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
}

func TestRunChecks_EmptyCommand(t *testing.T) {
	checks := []spec.CheckRunner{
		{Name: "empty", Kind: "command", Run: ""},
	}
	results := RunChecks(checks)
	if results[0].Status != "error" {
		t.Errorf("expected error, got %s", results[0].Status)
	}
}

func TestRunChecks_WithTimeout(t *testing.T) {
	checks := []spec.CheckRunner{
		{Name: "fast", Kind: "command", Run: "echo fast", Timeout: "10s"},
	}
	results := RunChecks(checks)
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s", results[0].Status)
	}
}

func TestRunProfileChecks(t *testing.T) {
	mas := &spec.MAS{
		Checks: &spec.Checks{
			Runner: []spec.CheckRunner{
				{Name: "a", Kind: "command", Run: "echo a"},
				{Name: "b", Kind: "command", Run: "echo b"},
				{Name: "c", Kind: "command", Run: "echo c"},
			},
		},
		Profiles: map[string]spec.Profile{
			"quick": {Checks: []string{"a"}},
			"full":  {Checks: []string{"a", "b", "c"}},
		},
	}

	results, err := RunProfileChecks(mas, "quick")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for quick profile, got %d", len(results))
	}

	results, err = RunProfileChecks(mas, "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results for full profile, got %d", len(results))
	}
}

func TestRunProfileChecks_MissingProfile(t *testing.T) {
	mas := &spec.MAS{
		Checks: &spec.Checks{
			Runner: []spec.CheckRunner{
				{Name: "a", Kind: "command", Run: "echo a"},
			},
		},
		Profiles: map[string]spec.Profile{},
	}

	_, err := RunProfileChecks(mas, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  int64 // seconds
		err   bool
	}{
		{"10s", 10, false},
		{"5m", 300, false},
		{"1h", 3600, false},
		{"x", 0, true},
		{"10x", 0, true},
	}

	for _, tt := range tests {
		d, err := parseDuration(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("parseDuration(%q): expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDuration(%q): unexpected error: %v", tt.input, err)
			continue
		}
		gotSec := int64(d.Seconds())
		if gotSec != tt.want {
			t.Errorf("parseDuration(%q) = %ds, want %ds", tt.input, gotSec, tt.want)
		}
	}
}
