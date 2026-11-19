package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// shSteps runs body with sh and asserts it exited 0.
func shSteps(body string) []spec.Step {
	zero := 0
	return []spec.Step{
		{Action: "cli", Command: "sh", Args: []string{"-c", body}},
		{Action: "assert", Expect: &spec.Expectation{ExitCode: &zero}},
	}
}

// logStep appends word to the file log, so a test can see which steps ran.
func logStep(word string) spec.Step {
	return spec.Step{Action: "cli", Command: "sh", Args: []string{"-c", "echo " + word + " >> log"}}
}

func runIn(t *testing.T, dir string, contract spec.Contract) []evidence.ContractResult {
	t.Helper()
	return RunContractsWith([]spec.Contract{contract}, ContractOptions{Dir: dir})
}

func readLog(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "log"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return strings.Join(strings.Fields(string(data)), " ")
}

func contractStatuses(r evidence.ContractResult) string {
	return strings.Join(attemptStatuses(r.Attempts, func(a evidence.ContractAttempt) string { return a.Status }), ",")
}

func TestContractRetries_PassingFirstTimeRunsOnce(t *testing.T) {
	dir := t.TempDir()
	got := runIn(t, dir, spec.Contract{Name: "c", Retries: 2, Scenarios: []spec.Scenario{{Name: "s", Steps: shSteps(passScript)}}})[0]

	if got.Status != "pass" || got.Attempts != nil {
		t.Errorf("expected pass with no attempts, got %+v", got)
	}
	if n := runCount(t, dir); n != 1 {
		t.Errorf("expected one run, got %d", n)
	}
}

func TestContractRetries_FailingThenPassingIsFlaky(t *testing.T) {
	dir := t.TempDir()
	got := runIn(t, dir, spec.Contract{Name: "c", Retries: 2, Scenarios: []spec.Scenario{{Name: "s", Steps: shSteps(failOnceScript)}}})[0]

	if got.Status != evidence.StatusFlaky {
		t.Errorf("expected flaky, got %q", got.Status)
	}
	if s := contractStatuses(got); s != "fail,pass" {
		t.Fatalf("expected attempts fail,pass, got %q", s)
	}
	first := got.Attempts[0]
	if first.FailedStep == nil || first.FailedStep.Phase != "steps" || first.FailedStep.Index != 1 {
		t.Errorf("expected the failed attempt to name its failing step, got %+v", first.FailedStep)
	}
	if !strings.Contains(first.Output, "first") || first.Error == "" {
		t.Errorf("expected the failed attempt's output and error kept, got %+v", first)
	}
	if got.FailedStep != nil || got.Error != "" {
		t.Errorf("expected top-level fields to describe the passing attempt, got %+v", got)
	}
}

func TestContractRetries_FailingEveryTimeFails(t *testing.T) {
	dir := t.TempDir()
	got := runIn(t, dir, spec.Contract{Name: "c", Retries: 2, Scenarios: []spec.Scenario{{Name: "s", Steps: shSteps(failScript)}}})[0]

	if got.Status != "fail" || got.FailedStep == nil {
		t.Errorf("expected fail naming the step, got %+v", got)
	}
	if s := contractStatuses(got); s != "fail,fail,fail" {
		t.Errorf("expected three failed attempts, got %q", s)
	}
	if n := runCount(t, dir); n != 3 {
		t.Errorf("expected three runs, got %d", n)
	}
}

func TestContractRetries_WithoutRetriesIsUnchanged(t *testing.T) {
	dir := t.TempDir()
	got := runIn(t, dir, spec.Contract{Name: "c", Scenarios: []spec.Scenario{{Name: "s", Steps: shSteps(failScript)}}})[0]

	if got.Status != "fail" || got.Attempts != nil {
		t.Errorf("expected fail with no attempts, got %+v", got)
	}
}

func TestContractRetries_SetupAndTeardownRunEachAttempt(t *testing.T) {
	dir := t.TempDir()
	contract := spec.Contract{
		Name:     "c",
		Retries:  2,
		Setup:    []spec.Step{logStep("contract-setup")},
		Teardown: []spec.Step{logStep("contract-teardown")},
		Scenarios: []spec.Scenario{{
			Name:     "s",
			Setup:    []spec.Step{logStep("setup")},
			Teardown: []spec.Step{logStep("teardown")},
			Steps:    shSteps(failOnceScript),
		}},
	}
	got := runIn(t, dir, contract)[0]

	if got.Status != evidence.StatusFlaky {
		t.Errorf("expected flaky, got %q", got.Status)
	}
	once := "contract-setup setup teardown contract-teardown"
	if log := readLog(t, dir); log != once+" "+once {
		t.Errorf("expected setup and teardown around each attempt, got %q", log)
	}
}

func TestContractRetries_OnlyFailingScenariosRerun(t *testing.T) {
	dir := t.TempDir()
	contract := spec.Contract{
		Name:    "c",
		Retries: 2,
		Scenarios: []spec.Scenario{
			{Name: "steady", Steps: []spec.Step{logStep("steady")}},
			{Name: "flaky", Steps: shSteps(failOnceScript)},
		},
	}
	results := runIn(t, dir, contract)

	if results[0].Name != "c/steady" || results[0].Status != "pass" || results[0].Attempts != nil {
		t.Errorf("expected the steady scenario to pass once, got %+v", results[0])
	}
	if results[1].Name != "c/flaky" || results[1].Status != evidence.StatusFlaky {
		t.Errorf("expected the second scenario flaky, got %+v", results[1])
	}
	if log := readLog(t, dir); log != "steady" {
		t.Errorf("expected the passing scenario to run once, got %q", log)
	}
}

func TestContractRetries_FailingSetupIsRetriedThenFailsEveryScenario(t *testing.T) {
	dir := t.TempDir()
	contract := spec.Contract{
		Name:      "c",
		Retries:   1,
		Setup:     shSteps(failScript),
		Scenarios: []spec.Scenario{{Name: "a", Steps: []spec.Step{logStep("a")}}, {Name: "b", Steps: []spec.Step{logStep("b")}}},
	}
	results := runIn(t, dir, contract)

	for _, got := range results {
		if got.Status != "fail" || !strings.HasPrefix(got.Error, "contract setup failed: ") {
			t.Errorf("expected a contract setup failure, got %+v", got)
		}
		if s := contractStatuses(got); s != "fail,fail" {
			t.Errorf("expected both setup attempts recorded on %s, got %q", got.Name, s)
		}
		for _, a := range got.Attempts {
			if a.FailedStep == nil || a.FailedStep.Phase != "contract_setup" {
				t.Errorf("expected each attempt to name the setup step, got %+v", a.FailedStep)
			}
		}
	}
	if n := runCount(t, dir); n != 2 {
		t.Errorf("expected the setup to run twice, got %d", n)
	}
	if log := readLog(t, dir); log != "" {
		t.Errorf("expected no scenario to run, got %q", log)
	}
}

// A setup that fails once and then passes is a flake of every scenario it starts.
func TestContractRetries_FlakySetupMakesScenariosFlaky(t *testing.T) {
	dir := t.TempDir()
	contract := spec.Contract{
		Name:      "c",
		Retries:   1,
		Setup:     shSteps(failOnceScript),
		Scenarios: []spec.Scenario{{Name: "a", Steps: []spec.Step{logStep("a")}}},
	}
	got := runIn(t, dir, contract)[0]

	if got.Status != evidence.StatusFlaky || contractStatuses(got) != "fail,pass" {
		t.Errorf("expected flaky with attempts fail,pass, got %+v", got)
	}
	if got.Attempts[0].FailedStep == nil || got.Attempts[0].FailedStep.Phase != "contract_setup" {
		t.Errorf("expected the first attempt to name the setup step, got %+v", got.Attempts[0].FailedStep)
	}
}

func TestContractRetries_AttemptsAreRedacted(t *testing.T) {
	t.Setenv("MASLOW_RETRY_SECRET", "hunter2-secret")
	dir := t.TempDir()
	contract := spec.Contract{
		Name:      "c",
		Retries:   1,
		Env:       []string{"MASLOW_RETRY_SECRET"},
		Scenarios: []spec.Scenario{{Name: "s", Steps: shSteps("echo {{.env.MASLOW_RETRY_SECRET}}; " + failOnceScript)}},
	}
	got := runIn(t, dir, contract)[0]

	if len(got.Attempts) != 2 {
		t.Fatalf("expected two attempts, got %+v", got)
	}
	first := got.Attempts[0]
	if strings.Contains(first.Output, "hunter2-secret") || strings.Contains(first.FailedStep.Target, "hunter2-secret") {
		t.Errorf("expected the secret redacted from the attempt, got %+v %+v", first, first.FailedStep)
	}
}

func TestAttemptScreenshotPath(t *testing.T) {
	path := screenshotPath("c", "s")
	if got := attemptScreenshotPath(path, 0); got != path {
		t.Errorf("expected the first attempt to keep %s, got %s", path, got)
	}
	if got, want := attemptScreenshotPath(path, 1), "reports/browser/c/s.attempt-2.png"; got != want {
		t.Errorf("attemptScreenshotPath(%s, 1) = %s, want %s", path, got, want)
	}
}
