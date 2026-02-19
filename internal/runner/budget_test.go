package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aschepis/maslow-agentic/internal/spec"
)

func TestCheckArtifactSize_Pass(t *testing.T) {
	// Create a small temp file.
	dir := t.TempDir()
	path := filepath.Join(dir, "small.bin")
	if err := os.WriteFile(path, make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	budgets := []spec.Budget{
		{Name: "size-check", Kind: "artifact_size", Target: path, Max: "1MB"},
	}
	results := RunBudgets(budgets)
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}
}

func TestCheckArtifactSize_Fail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.bin")
	if err := os.WriteFile(path, make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	budgets := []spec.Budget{
		{Name: "size-check", Kind: "artifact_size", Target: path, Max: "1KB"},
	}
	results := RunBudgets(budgets)
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
}

func TestCheckArtifactSize_MissingFile(t *testing.T) {
	budgets := []spec.Budget{
		{Name: "missing", Kind: "artifact_size", Target: "/nonexistent/file", Max: "1MB"},
	}
	results := RunBudgets(budgets)
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		err   bool
	}{
		{"100B", 100, false},
		{"1KB", 1024, false},
		{"1MB", 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"50MB", 50 * 1024 * 1024, false},
		{"1024", 1024, false},
		{"bad", 0, true},
	}

	for _, tt := range tests {
		got, err := parseSize(tt.input)
		if tt.err {
			if err == nil {
				t.Errorf("parseSize(%q): expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSize(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestRunBudgets_PerformanceSkipped(t *testing.T) {
	budgets := []spec.Budget{
		{Name: "perf", Kind: "performance", Target: "something"},
	}
	results := RunBudgets(budgets)
	if results[0].Status != "skip" {
		t.Errorf("expected skip for performance, got %s", results[0].Status)
	}
}
