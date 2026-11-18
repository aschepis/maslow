package verify

import (
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
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

	report, err := Run(mas, Options{Profile: "quick"})
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

	_, err := Run(mas, Options{Profile: "nonexistent"})
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

	report, err := Run(mas, Options{Profile: "full"})
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

func scopedSpec() *spec.MAS {
	exitZero := 0
	scenario := func(name string) []spec.Scenario {
		return []spec.Scenario{{Name: name, Steps: []spec.Step{
			{Action: "cli", Command: "true"},
			{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
		}}}
	}
	return &spec.MAS{
		Checks: &spec.Checks{Runner: []spec.CheckRunner{{Name: "echo", Kind: "command", Run: "echo hi"}}},
		Contracts: []spec.Contract{
			{Name: "cli-basics", Tags: []string{"smoke"}, Scenarios: scenario("a")},
			{Name: "slow-e2e", Tags: []string{"e2e"}, Scenarios: scenario("b")},
		},
		Budgets: []spec.Budget{
			{Name: "size", Kind: "artifact_size", Target: "verify.go", Max: "1MB", Tags: []string{"fast"}},
			{Name: "speed", Kind: "performance", Target: "x", Max: "1s"},
		},
		Refs:   []spec.Ref{{Kind: "file", Path: "verify.go", Required: true}},
		Policy: &spec.Policy{},
	}
}

func contractNames(results []evidence.ContractResult) []string {
	var names []string
	for _, r := range results {
		names = append(names, r.Name)
	}
	return names
}

func TestRun_ProfileWithoutSelectorsRunsEverything(t *testing.T) {
	mas := scopedSpec()
	mas.Profiles = map[string]spec.Profile{"full": {Checks: []string{"echo"}}}
	report, err := Run(mas, Options{Profile: "full"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Contracts) != 2 || len(report.Budgets) != 2 || len(report.Refs) != 1 {
		t.Errorf("expected every contract, budget and ref, got %d/%d/%d", len(report.Contracts), len(report.Budgets), len(report.Refs))
	}
}

func TestRun_ProfileSelectsContractsAndBudgetsByNameOrTag(t *testing.T) {
	mas := scopedSpec()
	mas.Profiles = map[string]spec.Profile{"quick": {
		Checks:    []string{"echo"},
		Contracts: []string{"smoke"},
		Budgets:   []string{"size"},
	}}
	report, err := Run(mas, Options{Profile: "quick"})
	if err != nil {
		t.Fatal(err)
	}
	if got := contractNames(report.Contracts); len(got) != 1 || got[0] != "cli-basics/a" {
		t.Errorf("expected only the smoke-tagged contract, got %v", got)
	}
	if len(report.Budgets) != 1 || report.Budgets[0].Name != "size" {
		t.Errorf("expected only the size budget, got %+v", report.Budgets)
	}
}

func TestRun_EmptySelectorRunsNone(t *testing.T) {
	mas := scopedSpec()
	mas.Profiles = map[string]spec.Profile{"quick": {Checks: []string{"echo"}, Contracts: []string{}, Budgets: []string{}}}
	report, err := Run(mas, Options{Profile: "quick"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Contracts) != 0 || len(report.Budgets) != 0 {
		t.Errorf("expected no contracts or budgets, got %d/%d", len(report.Contracts), len(report.Budgets))
	}
	if len(report.Refs) != 1 {
		t.Errorf("expected refs to keep running, got %d", len(report.Refs))
	}
}

func TestRun_ProfileCanOptOutOfRefsAndPolicy(t *testing.T) {
	no := false
	mas := scopedSpec()
	mas.Policy = &spec.Policy{Deny: []string{"**/*.never"}}
	mas.Profiles = map[string]spec.Profile{"quick": {Checks: []string{"echo"}, Refs: &no, Policy: &no}}
	report, err := Run(mas, Options{Profile: "quick"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Refs) != 0 || len(report.Policy) != 0 {
		t.Errorf("expected refs and policy skipped, got %d/%d", len(report.Refs), len(report.Policy))
	}
}

func TestRun_UnknownSelectorIsAnError(t *testing.T) {
	mas := scopedSpec()
	mas.Profiles = map[string]spec.Profile{"quick": {Checks: []string{"echo"}, Contracts: []string{"nope"}}}
	if _, err := Run(mas, Options{Profile: "quick"}); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("expected an error naming the unknown selector, got %v", err)
	}
}
