package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

const gatedSpec = `mas: "1.0"
project:
  name: demo
checks:
  runner:
    - name: gate
      kind: command
      run: "false"
    - name: ok
      kind: command
      run: "true"
profiles:
  full:
    checks: [gate, ok]
`

const weakenedSpec = `mas: "1.0"
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

// A branch that deletes the check it fails must not pass when verified
// against the spec from its base.
func TestSpecFromBase_RemovingAFailingCheckStillFails(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@test.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "maslow.yaml", gatedSpec)
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	git(t, dir, "checkout", "-qb", "feature")
	write(t, dir, "maslow.yaml", weakenedSpec)
	git(t, dir, "commit", "-qam", "drop the failing check")
	t.Chdir(dir)

	baseSHA, err := ResolveBase("main")
	if err != nil {
		t.Fatal(err)
	}

	headSpec, err := spec.Read("maslow.yaml", "")
	if err != nil {
		t.Fatal(err)
	}
	headReport, err := Run(headSpec.MAS, Options{Profile: "full", BaseSHA: baseSHA, SpecPath: "maslow.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if headReport.Verdict != "pass" {
		t.Fatalf("verified against its own spec the branch passes; got %s", headReport.Verdict)
	}

	baseSpec, err := spec.Read("maslow.yaml", baseSHA)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(baseSpec.MAS, Options{Profile: "full", BaseSHA: baseSHA, SpecPath: "maslow.yaml", SpecFromBase: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "fail" {
		t.Errorf("expected fail against the base spec, got %s", report.Verdict)
	}

	finding := findPolicy(report.Policy, "spec")
	if finding == nil {
		t.Fatalf("expected a spec policy finding, got %+v", report.Policy)
	}
	if finding.Status != "fail" || finding.Rule != "maslow.yaml" {
		t.Errorf("expected a failing spec finding for maslow.yaml, got %+v", finding)
	}
}

func TestSpecFromBase_UnchangedSpecPasses(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@test.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "maslow.yaml", weakenedSpec)
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	git(t, dir, "checkout", "-qb", "feature")
	write(t, dir, "main.go", "package main")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "code only")
	t.Chdir(dir)

	baseSHA, err := ResolveBase("main")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := spec.Read("maslow.yaml", baseSHA)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(loaded.MAS, Options{Profile: "full", BaseSHA: baseSHA, SpecPath: "maslow.yaml", SpecFromBase: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "pass" {
		t.Errorf("expected pass, got %s: %+v", report.Verdict, report.Policy)
	}
	if f := findPolicy(report.Policy, "spec"); f == nil || f.Status != "pass" {
		t.Errorf("expected a passing spec finding, got %+v", f)
	}
}

func TestLoadSpec_InvalidSpecAtBase(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@test.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "maslow.yaml", "mas: nope\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	t.Chdir(dir)

	baseSHA, err := ResolveBase("main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := spec.Read("maslow.yaml", baseSHA); err == nil {
		t.Error("expected an invalid base spec to be rejected")
	}
}

func TestSpecFromBase_DetectsSpecChangeFromASubdirectory(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@test.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "svc/maslow.yaml", weakenedSpec)
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	git(t, dir, "checkout", "-qb", "feature")
	write(t, dir, "svc/maslow.yaml", weakenedSpec+"# edited\n")
	git(t, dir, "commit", "-qam", "edit spec")
	t.Chdir(filepath.Join(dir, "svc"))

	baseSHA, err := ResolveBase("main")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := spec.Read("maslow.yaml", baseSHA)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(loaded.MAS, Options{Profile: "full", BaseSHA: baseSHA, SpecPath: "maslow.yaml", SpecFromBase: true})
	if err != nil {
		t.Fatal(err)
	}
	if f := findPolicy(report.Policy, "spec"); f == nil || f.Status == "pass" {
		t.Errorf("expected the edited svc/maslow.yaml to be reported, got %+v", f)
	}
}

func findPolicy(results []evidence.PolicyResult, kind string) *evidence.PolicyResult {
	for i := range results {
		if results[i].Kind == kind {
			return &results[i]
		}
	}
	return nil
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
