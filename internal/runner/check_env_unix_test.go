//go:build unix

package runner

import (
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

func runEnvCheck(t *testing.T, check spec.CheckRunner, inheritEnv bool) (output, errText string, inherited bool) {
	t.Helper()
	result := RunChecks([]spec.CheckRunner{check}, inheritEnv)[0]
	return result.Output, result.Error, result.InheritedEnv
}

func TestCheckEnv_UnlistedVariableIsAbsent(t *testing.T) {
	t.Setenv("MASLOW_CHECK_UNLISTED", "visible-value")

	out, _, inherited := runEnvCheck(t, spec.CheckRunner{Name: "env", Kind: "command", Run: "env"}, false)

	if strings.Contains(out, "MASLOW_CHECK_UNLISTED") {
		t.Errorf("an unlisted variable reached the check:\n%s", out)
	}
	if inherited {
		t.Error("a scrubbed check was recorded as inheriting the environment")
	}
}

func TestCheckEnv_ListedVariableIsPresentAndRedacted(t *testing.T) {
	t.Setenv("MASLOW_CHECK_SECRET", "hunter22-secret")

	check := spec.CheckRunner{Name: "env", Kind: "command", Run: "env", Env: []string{"MASLOW_CHECK_SECRET"}}
	out, _, _ := runEnvCheck(t, check, false)

	if !strings.Contains(out, "MASLOW_CHECK_SECRET=[redacted:MASLOW_CHECK_SECRET]") {
		t.Errorf("expected the listed variable, redacted, in the output:\n%s", out)
	}
	if strings.Contains(out, "hunter22-secret") {
		t.Errorf("the listed value leaked into the output:\n%s", out)
	}
}

func TestCheckEnv_ListedValueIsRedactedFromError(t *testing.T) {
	t.Setenv("MASLOW_CHECK_BINARY", "no-such-binary-hunter22")

	check := spec.CheckRunner{Name: "missing", Kind: "command", Run: "no-such-binary-hunter22", Env: []string{"MASLOW_CHECK_BINARY"}}
	_, errText, _ := runEnvCheck(t, check, false)

	if errText == "" || strings.Contains(errText, "no-such-binary-hunter22") {
		t.Errorf("expected the value redacted from the error, got %q", errText)
	}
}

func TestCheckEnv_BaseSetIsPresent(t *testing.T) {
	t.Setenv("TMPDIR", "/tmp/maslow-check-env")
	t.Setenv("CI", "true")

	out, _, _ := runEnvCheck(t, spec.CheckRunner{Name: "env", Kind: "command", Run: "env"}, false)

	for _, want := range []string{"PATH=", "HOME=", "TMPDIR=/tmp/maslow-check-env", "CI=true"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in the check's environment:\n%s", want, out)
		}
	}
}

func TestCheckEnv_InheritEnvRestoresTheFullEnvironment(t *testing.T) {
	t.Setenv("MASLOW_CHECK_UNLISTED", "visible-value")

	out, _, inherited := runEnvCheck(t, spec.CheckRunner{Name: "env", Kind: "command", Run: "env"}, true)

	if !strings.Contains(out, "MASLOW_CHECK_UNLISTED=visible-value") {
		t.Errorf("expected the inherited variable in the output:\n%s", out)
	}
	if !inherited {
		t.Error("expected the result to record the inherited environment")
	}
}

func TestCheckEnv_InheritEnvIsRecordedOnRetriedChecks(t *testing.T) {
	_, _, inherited := runEnvCheck(t, spec.CheckRunner{Name: "fails", Kind: "command", Run: "false", Retries: 1}, true)

	if !inherited {
		t.Error("expected a retried check to record the inherited environment")
	}
}
