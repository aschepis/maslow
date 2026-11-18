package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
)

const passingSpec = `mas: "1.0"
project:
  name: demo
checks:
  runner:
    - name: ok
      kind: command
      run: "true"
profiles:
  full:
    checks: [ok]
`

// repoWithBranch makes a repository whose main branch holds baseFiles and
// whose checked-out feature branch adds or replaces branchFiles.
func repoWithBranch(t *testing.T, baseFiles, branchFiles map[string]string) {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	put := func(files map[string]string) {
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	run("init", "-q", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	put(baseFiles)
	run("add", ".")
	run("commit", "-qm", "base")
	run("checkout", "-qb", "feature")
	put(branchFiles)
	run("add", ".")
	run("commit", "-qm", "feature")
	t.Chdir(dir)
}

func verifyFromBase(t *testing.T) (int, *evidence.Report) {
	t.Helper()
	reportPath := filepath.Join(t.TempDir(), "verify.json")
	code := cmdVerify([]string{"--profile", "full", "--base", "main", "--spec-from", "base", "--report", reportPath})

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected an evidence file (exit %d): %v", code, err)
	}
	var report evidence.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	return code, &report
}

func specFinding(report *evidence.Report) *evidence.PolicyResult {
	for i := range report.Policy {
		if report.Policy[i].Kind == "spec" {
			return &report.Policy[i]
		}
	}
	return nil
}

// The pull request that adopts Maslow has no spec at its base; it must be
// verified against its own spec and routed to a person, not fail blind.
func TestVerify_SpecFromBaseWithNoBaseSpecFallsBackForReview(t *testing.T) {
	repoWithBranch(t, map[string]string{"README.md": "hi"}, map[string]string{"maslow.yaml": passingSpec})

	code, report := verifyFromBase(t)

	if report.Verdict != "needs_review" || code != 0 {
		t.Errorf("expected needs_review with exit 0, got %s with exit %d", report.Verdict, code)
	}
	if report.SpecSource != "head" {
		t.Errorf("expected spec_source head, got %q", report.SpecSource)
	}
	if f := specFinding(report); f == nil || f.Status != "needs_review" {
		t.Errorf("expected a spec finding that needs review, got %+v", f)
	}
	if len(report.CheckResults) != 1 || report.CheckResults[0].Status != "pass" {
		t.Errorf("expected the branch's checks to run, got %+v", report.CheckResults)
	}
}

func TestVerify_SpecFromBaseFallbackStillFailsOnFailures(t *testing.T) {
	failing := `mas: "1.0"
project:
  name: demo
checks:
  runner:
    - name: broken
      kind: command
      run: "false"
profiles:
  full:
    checks: [broken]
`
	repoWithBranch(t, map[string]string{"README.md": "hi"}, map[string]string{"maslow.yaml": failing})

	code, report := verifyFromBase(t)

	if report.Verdict != "fail" || code != 1 {
		t.Errorf("expected fail with exit 1, got %s with exit %d", report.Verdict, code)
	}
}

func TestVerify_InvalidSpecAtBaseWritesFailingEvidence(t *testing.T) {
	repoWithBranch(t, map[string]string{"maslow.yaml": "mas: nope\n"}, map[string]string{"maslow.yaml": passingSpec})

	code, report := verifyFromBase(t)

	if report.Verdict != "fail" || code != 1 {
		t.Errorf("expected fail with exit 1, got %s with exit %d", report.Verdict, code)
	}
	if f := specFinding(report); f == nil || f.Status != "error" || f.Error == "" {
		t.Errorf("expected a spec finding with the load error, got %+v", f)
	}
	if report.SpecSource != "base" {
		t.Errorf("expected spec_source base, got %q", report.SpecSource)
	}
}
