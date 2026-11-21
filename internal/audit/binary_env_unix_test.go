//go:build unix

package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

// versionEnv audits a binary whose version command prints its environment.
func versionEnv(t *testing.T, target spec.AuditTarget) (output string, inherited bool) {
	t.Helper()
	target.Kind = "binary"
	target.Path = filepath.Join(t.TempDir(), "env-bin")
	target.Checks = []string{"version-check"}
	if err := os.WriteFile(target.Path, []byte("#!/bin/sh\nenv\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := Run(&spec.MAS{Audit: &spec.Audit{Targets: []spec.AuditTarget{target}}}, Options{Profile: "full"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range report.CheckResults {
		if strings.HasSuffix(c.Name, "/version") {
			return c.Output, c.InheritedEnv
		}
	}
	t.Fatalf("no version result in %+v", report.CheckResults)
	return "", false
}

func TestBinaryEnv_EveryTargetEnvEntryIsPresent(t *testing.T) {
	out, _ := versionEnv(t, spec.AuditTarget{Env: map[string]string{
		"MASLOW_AUDIT_A": "one", "MASLOW_AUDIT_B": "two", "MASLOW_AUDIT_C": "three",
	}})

	for _, want := range []string{"MASLOW_AUDIT_A=one", "MASLOW_AUDIT_B=two", "MASLOW_AUDIT_C=three"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in the binary's environment:\n%s", want, out)
		}
	}
}

func TestBinaryEnv_UnlistedVariableIsAbsent(t *testing.T) {
	t.Setenv("MASLOW_AUDIT_UNLISTED", "visible-value")

	out, inherited := versionEnv(t, spec.AuditTarget{Env: map[string]string{"MASLOW_AUDIT_A": "one"}})

	if strings.Contains(out, "MASLOW_AUDIT_UNLISTED") {
		t.Errorf("an unlisted variable reached the audit binary:\n%s", out)
	}
	if inherited {
		t.Error("a scrubbed target was recorded as inheriting the environment")
	}
}

func TestBinaryEnv_BaseSetIsPresent(t *testing.T) {
	t.Setenv("CI", "true")

	out, _ := versionEnv(t, spec.AuditTarget{})

	for _, want := range []string{"PATH=", "HOME=", "CI=true"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in the binary's environment:\n%s", want, out)
		}
	}
}

func TestBinaryEnv_InheritEnvRestoresTheFullEnvironment(t *testing.T) {
	t.Setenv("MASLOW_AUDIT_UNLISTED", "visible-value")

	out, inherited := versionEnv(t, spec.AuditTarget{InheritEnv: true, Env: map[string]string{"MASLOW_AUDIT_A": "one"}})

	for _, want := range []string{"MASLOW_AUDIT_UNLISTED=visible-value", "MASLOW_AUDIT_A=one"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in the binary's environment:\n%s", want, out)
		}
	}
	if !inherited {
		t.Error("expected the result to record the inherited environment")
	}
}
