package runner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

func runOne(t *testing.T, scenario spec.Scenario) evidence.ContractResult {
	t.Helper()
	results := RunContracts([]spec.Contract{{Name: "c", Scenarios: []spec.Scenario{scenario}}})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	return results[0]
}

func TestContractEvidence_FailingAssertRecordsStepAndOutput(t *testing.T) {
	exitZero := 0
	got := runOne(t, spec.Scenario{
		Name: "s",
		Steps: []spec.Step{
			{Action: "cli", Command: "sh", Args: []string{"-c", "echo boom; exit 3"}},
			{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
		},
	})

	if got.Status != "fail" {
		t.Fatalf("expected fail, got %s", got.Status)
	}
	want := &evidence.FailedStep{Phase: "steps", Index: 1, Action: "assert"}
	if !reflect.DeepEqual(got.FailedStep, want) {
		t.Errorf("failed_step = %+v, want %+v", got.FailedStep, want)
	}
	if got.Output != "boom\n" {
		t.Errorf("output = %q, want the cli step's output", got.Output)
	}
}

func TestContractEvidence_SetupFailureNamesSubstitutedCommand(t *testing.T) {
	got := runOne(t, spec.Scenario{
		Name: "s",
		Setup: []spec.Step{
			{Action: "capture", From: "status", As: "code"},
			{Action: "cli", Command: "/no/such/binary-{{.cap.code}}"},
		},
		Steps: []spec.Step{{Action: "cli", Command: "true"}},
	})

	want := &evidence.FailedStep{Phase: "setup", Index: 1, Action: "cli", Target: "/no/such/binary-0"}
	if !reflect.DeepEqual(got.FailedStep, want) {
		t.Errorf("failed_step = %+v, want %+v", got.FailedStep, want)
	}
}

func TestContractEvidence_ContractSetupFailure(t *testing.T) {
	results := RunContracts([]spec.Contract{{
		Name:      "c",
		Setup:     []spec.Step{{Action: "cli", Command: "/no/such/binary"}},
		Scenarios: []spec.Scenario{{Name: "a"}, {Name: "b"}},
	}})
	for _, r := range results {
		if r.FailedStep == nil || r.FailedStep.Phase != "contract_setup" {
			t.Errorf("%s: failed_step = %+v, want phase contract_setup", r.Name, r.FailedStep)
		}
	}
}

func TestContractEvidence_PassingScenarioHasNoFailure(t *testing.T) {
	got := runOne(t, spec.Scenario{Name: "s", Steps: []spec.Step{{Action: "cli", Command: "echo", Args: []string{"hi"}}}})
	if got.Status != "pass" || got.FailedStep != nil || got.Output != "" {
		t.Errorf("expected a bare pass, got %+v", got)
	}
}

func TestContractEvidence_ArtifactsCollectedBeforeTeardown(t *testing.T) {
	t.Chdir(t.TempDir())
	got := runOne(t, spec.Scenario{
		Name: "s",
		Steps: []spec.Step{
			{Action: "cli", Command: "sh", Args: []string{"-c", "mkdir -p out/a/b && touch out/top.png out/a/b/deep.png out/a/skip.txt"}},
		},
		Teardown:  []spec.Step{{Action: "cli", Command: "rm", Args: []string{"-rf", "out"}}},
		Artifacts: []string{"out/**/*.png", "out/top.png", "missing/**"},
	})

	if got.Status != "pass" {
		t.Fatalf("expected pass, got %s: %s", got.Status, got.Error)
	}
	want := []string{"out/a/b/deep.png", "out/top.png"}
	if !reflect.DeepEqual(got.Artifacts, want) {
		t.Errorf("artifacts = %v, want %v", got.Artifacts, want)
	}
	if _, err := os.Stat(filepath.Join("out", "top.png")); !os.IsNotExist(err) {
		t.Error("expected teardown to run after artifacts were collected")
	}
}

func TestContractEvidence_BadArtifactPatternIsAnError(t *testing.T) {
	t.Chdir(t.TempDir())
	got := runOne(t, spec.Scenario{
		Name:      "s",
		Steps:     []spec.Step{{Action: "cli", Command: "true"}},
		Artifacts: []string{"/abs/path.png"},
	})
	if got.Status != "error" {
		t.Errorf("expected error for a bad artifact pattern, got %s", got.Status)
	}
}
