package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// failOnce fails the first time it runs in a directory and passes after, so
// the head directory and the base worktree each flake once.
const failOnce = "if [ -f marker ]; then exit 0; fi; touch marker; exit 1"

// shOverlay is an overlay whose one scenario passes when body exits 0.
func shOverlay(body string, retries int) *spec.LoadedOverlay {
	zero := 0
	overlay := overlayOf("change/acceptance.yaml",
		spec.Step{Action: "cli", Command: "sh", Args: []string{"-c", body}},
		spec.Step{Action: "assert", Expect: &spec.Expectation{ExitCode: &zero}},
	)
	overlay.Overlay.Contracts[0].Retries = retries
	return overlay
}

func baseStatuses(run *evidence.BaseRun) string {
	var out []string
	for _, a := range run.Attempts {
		out = append(out, a.Status)
	}
	return strings.Join(out, ",")
}

func TestRedGreenStatus(t *testing.T) {
	flaky := evidence.StatusFlaky
	cases := []struct{ head, base, want string }{
		{"pass", "fail", "pass"},
		{"pass", "pass", evidence.StatusNotRed},
		{"pass", flaky, evidence.StatusNotRed},
		{"pass", "error", "error"},
		{flaky, "fail", flaky},
		{flaky, "pass", evidence.StatusNotRed},
		{flaky, flaky, evidence.StatusNotRed},
		{flaky, "error", "error"},
		{"fail", "fail", "fail"},
		{"fail", "pass", "fail"},
		{"fail", flaky, "fail"},
		{"error", "fail", "error"},
	}
	for _, c := range cases {
		if got := redGreenStatus(c.head, c.base); got != c.want {
			t.Errorf("redGreenStatus(head %s, base %s) = %s, want %s", c.head, c.base, got, c.want)
		}
	}
}

// A base that fails and then passes shows the contract can pass without the change.
func TestRedGreen_FlakyBaseIsNotRed(t *testing.T) {
	base := redGreenRepo(t)
	write(t, ".", "marker", "")
	report := runRedGreen(t, base, quickSpec(), shOverlay(failOnce, 1))

	got := report.Contracts[0]
	if got.HeadStatus != "pass" || got.Status != evidence.StatusNotRed {
		t.Errorf("expected not_red with the head passing, got %+v", got)
	}
	if got.Base == nil || got.Base.Status != evidence.StatusFlaky || baseStatuses(got.Base) != "fail,pass" {
		t.Errorf("expected a flaky base with two attempts, got %+v", got.Base)
	}
	if report.Verdict != "fail" {
		t.Errorf("expected verdict fail, got %s", report.Verdict)
	}
}

func TestRedGreen_RedBaseFlakyHeadIsFlaky(t *testing.T) {
	base := redGreenRepo(t)
	report := runRedGreen(t, base, quickSpec(), shOverlay("grep -q new feature.txt && { "+failOnce+"; }", 1))

	got := report.Contracts[0]
	if got.Status != evidence.StatusFlaky || got.HeadStatus != evidence.StatusFlaky {
		t.Errorf("expected flaky on the head, got %+v", got)
	}
	if got.Base == nil || got.Base.Status != "fail" {
		t.Errorf("expected the base to fail, got %+v", got.Base)
	}
	if report.Verdict != "needs_review" {
		t.Errorf("expected a flaky acceptance contract to need review, got %s", report.Verdict)
	}
}

func TestRedGreen_BaseRetriesEveryAttempt(t *testing.T) {
	base := redGreenRepo(t)
	report := runRedGreen(t, base, quickSpec(), shOverlay("grep -q new feature.txt", 2))

	got := report.Contracts[0]
	if got.Status != "pass" || got.HeadStatus != "pass" {
		t.Errorf("expected pass, got %+v", got)
	}
	if got.Base == nil || got.Base.Status != "fail" || baseStatuses(got.Base) != "fail,fail,fail" {
		t.Errorf("expected the base to fail three times, got %+v", got.Base)
	}
	if report.Verdict != "pass" {
		t.Errorf("expected verdict pass, got %s", report.Verdict)
	}
}

func TestFlakyEvidenceMatchesSchema(t *testing.T) {
	base := redGreenRepo(t)
	write(t, ".", "flaky-check.sh", "if [ -f check-marker ]; then exit 0; fi; touch check-marker; exit 1\n")
	mas := quickSpec()
	mas.Checks = &spec.Checks{Runner: []spec.CheckRunner{{Name: "flaky", Kind: "script", Run: "sh flaky-check.sh", Retries: 1}}}
	contract := shOverlay(strings.ReplaceAll(failOnce, "marker", "contract-marker"), 1).Overlay.Contracts[0]
	contract.Name = "spec-contract"
	mas.Contracts = []spec.Contract{contract}
	mas.Profiles["quick"] = spec.Profile{Checks: []string{"flaky"}, Contracts: []string{"spec-contract"}}

	report := runRedGreen(t, base, mas, shOverlay(failOnce, 1))
	report.CapOutput(evidence.DefaultOutputLimit)

	if c := report.CheckResults[0]; c.Status != evidence.StatusFlaky || len(c.Attempts) != 2 {
		t.Errorf("expected a flaky check with attempts, got %+v", c)
	}
	if c := report.Contracts[0]; c.Status != evidence.StatusFlaky || len(c.Attempts) != 2 {
		t.Errorf("expected a flaky spec contract with attempts, got %+v", c)
	}
	if c := report.Contracts[1]; c.Base == nil || len(c.Base.Attempts) != 2 || c.HeadStatus != evidence.StatusFlaky {
		t.Errorf("expected a flaky head and a base run with attempts, got %+v base=%+v", c, c.Base)
	}

	path := filepath.Join(t.TempDir(), "verify.json")
	if err := report.Write(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	validateAgainstSchema(t, data)
}

func TestFlakyGoldenMatchesSchema(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "evidence", "flaky.json"))
	if err != nil {
		t.Fatal(err)
	}
	validateAgainstSchema(t, data)
}

func TestRun_OverlayRetriesAboveSpecLimitAreRefused(t *testing.T) {
	limit := 1
	mas := quickSpec()
	mas.MaxRetries = &limit
	_, err := Run(mas, Options{Profile: "quick", Overlays: []*spec.LoadedOverlay{shOverlay("true", 2)}})
	if err == nil || !strings.Contains(err.Error(), "max_retries") || !strings.Contains(err.Error(), "change/acceptance.yaml") {
		t.Errorf("expected overlay retries above max_retries refused, got %v", err)
	}
}
