//go:build unix

package runner

import (
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

// runEnvStep runs a contract whose CLI step prints its environment and then fails, so
// the output lands in the evidence.
func runEnvStep(t *testing.T, contract spec.Contract) (output string, inherited bool) {
	t.Helper()
	exitZero := 0
	contract.Name = "c"
	contract.Scenarios = []spec.Scenario{{
		Name: "s",
		Steps: []spec.Step{
			{Action: "cli", Command: "sh", Args: []string{"-c", "env; exit 1"}},
			{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
		},
	}}
	result := RunContracts([]spec.Contract{contract})[0]
	return result.Output, result.InheritedEnv
}

func TestCLIStepEnv_UnlistedVariableIsAbsent(t *testing.T) {
	t.Setenv("MASLOW_STEP_UNLISTED", "visible-value")

	out, inherited := runEnvStep(t, spec.Contract{})

	if strings.Contains(out, "MASLOW_STEP_UNLISTED") {
		t.Errorf("an unlisted variable reached the CLI step:\n%s", out)
	}
	if inherited {
		t.Error("a scrubbed contract was recorded as inheriting the environment")
	}
}

func TestCLIStepEnv_ListedVariableIsPresentAndRedacted(t *testing.T) {
	t.Setenv("MASLOW_STEP_SECRET", "hunter22-secret")

	out, _ := runEnvStep(t, spec.Contract{Env: []string{"MASLOW_STEP_SECRET"}})

	if !strings.Contains(out, "MASLOW_STEP_SECRET=[redacted:MASLOW_STEP_SECRET]") {
		t.Errorf("expected the listed variable, redacted, in the output:\n%s", out)
	}
	if strings.Contains(out, "hunter22-secret") {
		t.Errorf("the listed value leaked into the output:\n%s", out)
	}
}

func TestCLIStepEnv_BaseSetIsPresent(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp/maslow-step-env")
	t.Setenv("CI", "true")

	out, _ := runEnvStep(t, spec.Contract{})

	for _, want := range []string{"PATH=", "HOME=", "TMPDIR=/tmp/maslow-step-env", "CI=true"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in the CLI step's environment:\n%s", want, out)
		}
	}
}

func TestCLIStepEnv_InheritEnvRestoresTheFullEnvironment(t *testing.T) {
	t.Setenv("MASLOW_STEP_UNLISTED", "visible-value")

	out, inherited := runEnvStep(t, spec.Contract{InheritEnv: true})

	if !strings.Contains(out, "MASLOW_STEP_UNLISTED=visible-value") {
		t.Errorf("expected the inherited variable in the output:\n%s", out)
	}
	if !inherited {
		t.Error("expected the result to record the inherited environment")
	}
}
