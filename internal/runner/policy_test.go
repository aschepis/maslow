package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
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
	results := checkDenyRules([]string{"vendor/**"}, []string{"main.go", "internal/a.go"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass for non-matching deny pattern, got %s", results[0].Status)
	}
}

func TestCheckDenyRules_MatchesAtAnyDepth(t *testing.T) {
	for _, file := range []string{"a/.env", "a/b/.env", "a/b/c/.env"} {
		results := checkDenyRules([]string{"**/.env"}, []string{"main.go", file})
		if results[0].Status != "fail" {
			t.Errorf("expected **/.env to fail on %s, got %s", file, results[0].Status)
		}
	}
	for _, file := range []string{"vendor/a.go", "vendor/a/b.go", "vendor/a/b/c.go"} {
		results := checkDenyRules([]string{"vendor/**"}, []string{file})
		if results[0].Status != "fail" {
			t.Errorf("expected vendor/** to fail on %s, got %s", file, results[0].Status)
		}
	}
}

func TestCheckDenyRules_ReportsMatchedFiles(t *testing.T) {
	results := checkDenyRules([]string{"**/secret*"}, []string{"secret.txt", "public.txt", "x/secret.key"})
	if results[0].Status != "fail" {
		t.Fatalf("expected fail, got %s", results[0].Status)
	}
	want := []string{"secret.txt", "x/secret.key"}
	if len(results[0].Files) != len(want) || results[0].Files[0] != want[0] || results[0].Files[1] != want[1] {
		t.Errorf("expected matched files %v, got %v", want, results[0].Files)
	}
}

func TestCheckDenyRules_InvalidPattern(t *testing.T) {
	results := checkDenyRules([]string{"a/["}, []string{"a/b"})
	if results[0].Status != "error" {
		t.Errorf("expected error for malformed pattern, got %s", results[0].Status)
	}
}

func TestCandidateFiles_IgnoresGitignoredFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".gitignore", ".env\n")
	writeFile(t, dir, "main.go", "package main")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	writeFile(t, dir, "a/b/.env", "SECRET=1")

	files, err := candidateFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkDenyRules([]string{"**/.env"}, files)
	if results[0].Status != "pass" {
		t.Errorf("expected an ignored, untracked .env to pass, got %s: %s", results[0].Status, results[0].Detail)
	}
}

func TestCandidateFiles_IncludesTrackedAndUnignoredFiles(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, ".gitignore", ".env\n")
	writeFile(t, dir, "a/.env", "SECRET=1")
	runGit(t, dir, "add", ".gitignore")
	runGit(t, dir, "add", "-f", "a/.env")
	runGit(t, dir, "commit", "-m", "initial")
	writeFile(t, dir, "b/credentials.json", "{}")

	files, err := candidateFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkDenyRules([]string{"**/.env", "**/credentials*"}, files)
	if results[0].Status != "fail" {
		t.Errorf("expected a tracked .env to fail even though it is ignored, got %s", results[0].Status)
	}
	if results[1].Status != "fail" {
		t.Errorf("expected an untracked, unignored credentials file to fail, got %s", results[1].Status)
	}
}

func TestProtected_Unchanged(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "protected.txt", "original")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	changed, err := workingTreeChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkProtectedFiles([]string{"protected.txt"}, changed)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass for unchanged protected file, got %s: %s %s", results[0].Status, results[0].Detail, results[0].Error)
	}
}

func TestProtected_Modified(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "protected.txt", "original")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	writeFile(t, dir, "protected.txt", "modified")

	changed, err := workingTreeChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkProtectedFiles([]string{"protected.txt"}, changed)
	if results[0].Status != "fail" {
		t.Errorf("expected fail for modified protected file, got %s", results[0].Status)
	}
}

func TestProtected_GlobMatchesNestedChange(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "docs/a/b/spec.md", "original")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	writeFile(t, dir, "docs/a/b/spec.md", "modified")

	changed, err := workingTreeChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkProtectedFiles([]string{"docs/**"}, changed)
	if results[0].Status != "fail" {
		t.Errorf("expected docs/** to catch a nested change, got %s", results[0].Status)
	}
	if len(results[0].Files) != 1 || results[0].Files[0] != "docs/a/b/spec.md" {
		t.Errorf("expected matched file docs/a/b/spec.md, got %v", results[0].Files)
	}
}

func TestProtected_UntrackedNewFileFails(t *testing.T) {
	dir := newGitRepo(t)
	writeFile(t, dir, "main.go", "package main")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	writeFile(t, dir, "protected.txt", "new")

	changed, err := workingTreeChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	results := checkProtectedFiles([]string{"protected.txt"}, changed)
	if results[0].Status != "fail" {
		t.Errorf("expected fail for a new untracked protected file, got %s", results[0].Status)
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
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
