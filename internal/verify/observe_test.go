package verify

import (
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

const observedSpec = `mas: "1.0"
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
policy:
  observe:
    protected: [locked.txt]
`

const enforcedSpec = `mas: "1.0"
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
policy:
  protected: [locked.txt]
`

func TestRun_ObservedViolationLeavesVerdictPass(t *testing.T) {
	report := runAgainstChangedLockedFile(t, observedSpec)
	if report.Verdict != "pass" {
		t.Errorf("expected pass with an observed violation, got %s: %+v", report.Verdict, report.Policy)
	}
	finding := findPolicy(report.Policy, "protected")
	if finding == nil || finding.Status != "would_fail" || finding.Mode != "observe" {
		t.Errorf("expected an observed would_fail finding, got %+v", report.Policy)
	}
}

func TestRun_SameRuleEnforcedFails(t *testing.T) {
	report := runAgainstChangedLockedFile(t, enforcedSpec)
	if report.Verdict != "fail" {
		t.Errorf("expected fail with the rule enforced, got %s", report.Verdict)
	}
}

func runAgainstChangedLockedFile(t *testing.T, specYAML string) *evidence.Report {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@test.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "maslow.yaml", specYAML)
	write(t, dir, "locked.txt", "v1")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	write(t, dir, "locked.txt", "v2")
	t.Chdir(dir)

	loaded, err := spec.Read("maslow.yaml", "")
	if err != nil {
		t.Fatal(err)
	}
	report, err := Run(loaded.MAS, Options{Profile: "full", SpecPath: "maslow.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	return report
}
