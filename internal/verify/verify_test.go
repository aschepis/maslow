package verify

import (
	"testing"

	"github.com/aschepis/maslow-agentic/internal/spec"
)

func TestRun_QuickProfile(t *testing.T) {
	mas := &spec.MAS{
		Checks: &spec.Checks{
			Runner: []spec.CheckRunner{
				{Name: "echo", Kind: "command", Run: "echo hello"},
			},
		},
		Profiles: map[string]spec.Profile{
			"quick": {Checks: []string{"echo"}},
		},
	}

	report, err := Run(mas, "quick")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verdict != "pass" {
		t.Errorf("expected pass, got %s", report.Verdict)
	}
	if len(report.CheckResults) != 1 {
		t.Errorf("expected 1 check result, got %d", len(report.CheckResults))
	}
}

func TestRun_MissingProfile(t *testing.T) {
	mas := &spec.MAS{
		Checks: &spec.Checks{
			Runner: []spec.CheckRunner{},
		},
		Profiles: map[string]spec.Profile{},
	}

	_, err := Run(mas, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestRun_WithContracts(t *testing.T) {
	exitZero := 0
	mas := &spec.MAS{
		Checks: &spec.Checks{
			Runner: []spec.CheckRunner{
				{Name: "echo", Kind: "command", Run: "echo hello"},
			},
		},
		Profiles: map[string]spec.Profile{
			"full": {Checks: []string{"echo"}},
		},
		Contracts: []spec.Contract{
			{
				Name: "test-contract",
				Scenarios: []spec.Scenario{
					{
						Name: "echo-test",
						Steps: []spec.Step{
							{Action: "cli", Command: "echo", Args: []string{"test"}},
							{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
						},
					},
				},
			},
		},
	}

	report, err := Run(mas, "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verdict != "pass" {
		t.Errorf("expected pass, got %s", report.Verdict)
	}
	if len(report.Contracts) != 1 {
		t.Errorf("expected 1 contract result, got %d", len(report.Contracts))
	}
}
