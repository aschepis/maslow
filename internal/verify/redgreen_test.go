package verify

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// redGreenRepo commits feature.txt as "old" on main, then "new" on a branch,
// and returns the merge base.
func redGreenRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@test.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, "feature.txt", "old\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	git(t, dir, "checkout", "-qb", "feature")
	write(t, dir, "feature.txt", "new\n")
	git(t, dir, "commit", "-qam", "fix")
	t.Chdir(dir)

	base, err := ResolveBase("main")
	if err != nil {
		t.Fatal(err)
	}
	return base
}

func grepOverlay(word string) *spec.LoadedOverlay {
	zero := 0
	return overlayOf("change/acceptance.yaml",
		spec.Step{Action: "cli", Command: "grep", Args: []string{"-q", word, "feature.txt"}},
		spec.Step{Action: "assert", Expect: &spec.Expectation{ExitCode: &zero}},
	)
}

func runRedGreen(t *testing.T, base string, mas *spec.MAS, overlay *spec.LoadedOverlay) *evidence.Report {
	t.Helper()
	report, err := Run(mas, Options{Profile: "quick", BaseSHA: base, RedGreen: true, Overlays: []*spec.LoadedOverlay{overlay}})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func quickSpec() *spec.MAS {
	return &spec.MAS{Profiles: map[string]spec.Profile{"quick": {Checks: []string{}, Contracts: []string{}}}}
}

func TestRedGreen_RedOnBaseGreenOnHeadPasses(t *testing.T) {
	base := redGreenRepo(t)
	report := runRedGreen(t, base, quickSpec(), grepOverlay("new"))

	got := report.Contracts[0]
	if got.Status != "pass" || got.HeadStatus != "pass" {
		t.Errorf("expected pass, got %+v", got)
	}
	if got.Base == nil || got.Base.SHA != base || got.Base.Status != "fail" {
		t.Errorf("expected the base run recorded as fail at %s, got %+v", base, got.Base)
	}
	if got.Base != nil && got.Base.FailedStep == nil {
		t.Error("expected the base run to say which step failed")
	}
	if report.Verdict != "pass" {
		t.Errorf("expected verdict pass, got %s", report.Verdict)
	}
}

func TestRedGreen_KeepsCoversAndDerivesCoverage(t *testing.T) {
	base := redGreenRepo(t)
	overlay := grepOverlay("new")
	overlay.Overlay.Contracts[0].Covers = []string{"acme/app#7"}
	report := runRedGreen(t, base, quickSpec(), overlay)

	if got := report.Contracts[0].Covers; len(got) != 1 || got[0] != "acme/app#7" {
		t.Errorf("covers = %v, want [acme/app#7]", got)
	}
	if len(report.Coverage) != 1 || report.Coverage[0].ID != "acme/app#7" || report.Coverage[0].Status != "pass" {
		t.Errorf("coverage = %+v", report.Coverage)
	}
}

// A contract already green on the base proves nothing about the change.
func TestRedGreen_GreenOnBothIsNotRed(t *testing.T) {
	base := redGreenRepo(t)
	report := runRedGreen(t, base, quickSpec(), grepOverlay("."))

	got := report.Contracts[0]
	if got.Status != evidence.StatusNotRed || got.HeadStatus != "pass" || got.Base == nil || got.Base.Status != "pass" {
		t.Errorf("expected not_red with both runs passing, got %+v base=%+v", got, got.Base)
	}
	if report.Verdict != "fail" {
		t.Errorf("expected verdict fail, got %s", report.Verdict)
	}
}

func TestRedGreen_RedOnBothFails(t *testing.T) {
	base := redGreenRepo(t)
	report := runRedGreen(t, base, quickSpec(), grepOverlay("absent"))

	got := report.Contracts[0]
	if got.Status != "fail" || got.HeadStatus != "fail" || got.Base == nil || got.Base.Status != "fail" {
		t.Errorf("expected fail on both, got %+v base=%+v", got, got.Base)
	}
	if report.Verdict != "fail" {
		t.Errorf("expected verdict fail, got %s", report.Verdict)
	}
}

func TestRedGreen_SpecContractsAreUnaffected(t *testing.T) {
	base := redGreenRepo(t)
	mas := quickSpec()
	mas.Contracts = []spec.Contract{{Name: "smoke", Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{{Action: "cli", Command: "true"}}}}}}
	mas.Profiles["quick"] = spec.Profile{Contracts: []string{"smoke"}}

	report := runRedGreen(t, base, mas, grepOverlay("new"))

	smoke := report.Contracts[0]
	if smoke.Name != "smoke/s" || smoke.Status != "pass" || smoke.Base != nil || smoke.HeadStatus != "" {
		t.Errorf("expected the spec contract untouched by red-green, got %+v", smoke)
	}
}

func TestRedGreen_RemovesTheBaseWorktree(t *testing.T) {
	base := redGreenRepo(t)
	runRedGreen(t, base, quickSpec(), grepOverlay("new"))

	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), "worktree "); n != 1 {
		t.Errorf("expected only the main worktree left, got:\n%s", out)
	}
}

func TestRedGreen_RequiresBase(t *testing.T) {
	_, err := Run(quickSpec(), Options{Profile: "quick", RedGreen: true, Overlays: []*spec.LoadedOverlay{grepOverlay("x")}})
	if err == nil || !strings.Contains(err.Error(), "--base") {
		t.Errorf("expected red-green without a base to be refused, got %v", err)
	}
}
