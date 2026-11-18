package runner

import (
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

func TestContractEnv_UnlistedVariableIsNotReadable(t *testing.T) {
	t.Setenv("MASLOW_TEST_SECRET", "hunter22")
	results := RunContracts([]spec.Contract{{
		Name: "c",
		Scenarios: []spec.Scenario{{
			Name:  "s",
			Steps: []spec.Step{{Action: "cli", Command: "echo", Args: []string{"{{.env.MASLOW_TEST_SECRET}}"}}},
		}},
	}})

	got := results[0]
	if got.Status != "fail" {
		t.Fatalf("expected an unlisted variable to fail the scenario, got %s", got.Status)
	}
	if !strings.Contains(got.Error, "env list") {
		t.Errorf("expected the error to point at the env list, got %q", got.Error)
	}
}

func TestContractEnv_ListedValuesAreRedacted(t *testing.T) {
	t.Setenv("MASLOW_TEST_TOKEN", "s3cr3t-token")
	t.Setenv("MASLOW_TEST_SHORT", "dev")
	exitZero := 0
	results := RunContracts([]spec.Contract{{
		Name: "c",
		Env:  []string{"MASLOW_TEST_TOKEN", "MASLOW_TEST_SHORT"},
		Scenarios: []spec.Scenario{{
			Name: "s",
			Steps: []spec.Step{
				{Action: "cli", Command: "sh", Args: []string{"-c", "echo token={{.env.MASLOW_TEST_TOKEN}} env={{.env.MASLOW_TEST_SHORT}}; exit 1"}},
				{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
			},
		}},
	}})

	got := results[0]
	for field, text := range map[string]string{"output": got.Output, "error": got.Error, "target": got.FailedStep.Target} {
		if strings.Contains(text, "s3cr3t-token") {
			t.Errorf("%s leaks the token: %q", field, text)
		}
	}
	if !strings.Contains(got.Output, "token=[redacted:MASLOW_TEST_TOKEN]") {
		t.Errorf("expected a redaction marker in output, got %q", got.Output)
	}
	if !strings.Contains(got.Output, "env=dev") {
		t.Errorf("expected values shorter than %d bytes to stay, got %q", minRedactLen, got.Output)
	}
}

func TestContractEnv_LongerValueRedactedWhole(t *testing.T) {
	r := redactor(map[string]string{"A": "abcd", "B": "abcdefgh"})
	if got := r.Replace("abcdefgh"); got != "[redacted:B]" {
		t.Errorf("got %q, want the longer value redacted whole", got)
	}
}
