package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

func TestRun_InheritEnvIsRecordedInEvidence(t *testing.T) {
	mas := &spec.MAS{
		Checks: &spec.Checks{
			InheritEnv: true,
			Runner:     []spec.CheckRunner{{Name: "echo", Kind: "command", Run: "echo hello"}},
		},
		Profiles: map[string]spec.Profile{"quick": {Checks: []string{"echo"}}},
	}

	report, err := Run(mas, Options{Profile: "quick"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "verify.json")
	if err := report.Write(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), `"inherited_env": true`) {
		t.Errorf("expected inherited_env in the evidence:\n%s", data)
	}
	validateAgainstSchema(t, data)
}

func TestRun_ContractInheritEnvIsRecordedInEvidence(t *testing.T) {
	mas := &spec.MAS{
		Checks: &spec.Checks{Runner: []spec.CheckRunner{{Name: "echo", Kind: "command", Run: "echo hello"}}},
		Contracts: []spec.Contract{{
			Name:       "legacy",
			InheritEnv: true,
			Scenarios:  []spec.Scenario{{Name: "s", Steps: []spec.Step{{Action: "cli", Command: "go", Args: []string{"version"}}}}},
		}},
		Profiles: map[string]spec.Profile{"quick": {Checks: []string{"echo"}}},
	}

	report, err := Run(mas, Options{Profile: "quick"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Contracts) != 1 || !report.Contracts[0].InheritedEnv {
		t.Fatalf("expected the contract result to record the inherited environment, got %+v", report.Contracts)
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
