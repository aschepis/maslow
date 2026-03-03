package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/aschepis/maslow-agentic/internal/spec"
)

func TestRunPolicy_Nil(t *testing.T) {
	results := RunPolicy(nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil policy, got %d", len(results))
	}
}

func TestRunPolicy_Empty(t *testing.T) {
	results := RunPolicy(&spec.Policy{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty policy, got %d", len(results))
	}
}

func TestCheckDenyRules_NoMatches(t *testing.T) {
	results := checkDenyRules([]string{"/nonexistent/path/**"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass for non-matching deny pattern, got %s", results[0].Status)
	}
}

func TestCheckDenyRules_WithMatches(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("SECRET=123"), 0o644)

	pattern := filepath.Join(dir, ".env")
	results := checkDenyRules([]string{pattern})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Errorf("expected fail for matching deny pattern, got %s", results[0].Status)
	}
}

func TestCheckDenyRules_GlobPattern(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("data"), 0o644)
	os.WriteFile(filepath.Join(dir, "public.txt"), []byte("data"), 0o644)

	pattern := filepath.Join(dir, "secret*")
	results := checkDenyRules([]string{pattern})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Errorf("expected fail for matching glob, got %s", results[0].Status)
	}
}

func TestCheckProtectedFiles_Unchanged(t *testing.T) {
	// Create a temp git repo with a committed file.
	dir := t.TempDir()

	// Initialize git repo.
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create and commit a file.
	filePath := filepath.Join(dir, "protected.txt")
	os.WriteFile(filePath, []byte("original"), 0o644)
	runGit(t, dir, "add", "protected.txt")
	runGit(t, dir, "commit", "-m", "initial")

	// Change to the temp dir so git commands work relative to it.
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	results := checkProtectedFiles([]string{"protected.txt"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass for unchanged protected file, got %s: %s %s", results[0].Status, results[0].Detail, results[0].Error)
	}
}

func TestCheckProtectedFiles_Modified(t *testing.T) {
	dir := t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	filePath := filepath.Join(dir, "protected.txt")
	os.WriteFile(filePath, []byte("original"), 0o644)
	runGit(t, dir, "add", "protected.txt")
	runGit(t, dir, "commit", "-m", "initial")

	// Modify the file.
	os.WriteFile(filePath, []byte("modified"), 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	results := checkProtectedFiles([]string{"protected.txt"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Errorf("expected fail for modified protected file, got %s", results[0].Status)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
