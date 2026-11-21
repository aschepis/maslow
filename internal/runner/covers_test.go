package runner

import (
	"reflect"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

var trueSteps = []spec.Step{{Action: "cli", Command: "true"}}

func TestCovers_ScenarioCarriesSortedUnionWithContract(t *testing.T) {
	results := RunContracts([]spec.Contract{{
		Name:   "c",
		Covers: []string{"b", "a"},
		Scenarios: []spec.Scenario{
			{Name: "s1", Covers: []string{"c", "a"}, Steps: trueSteps},
			{Name: "s2", Steps: trueSteps},
		},
	}})

	if !reflect.DeepEqual(results[0].Covers, []string{"a", "b", "c"}) {
		t.Errorf("s1 covers = %v, want [a b c]", results[0].Covers)
	}
	if !reflect.DeepEqual(results[1].Covers, []string{"a", "b"}) {
		t.Errorf("s2 covers = %v, want [a b]", results[1].Covers)
	}
}

func TestCovers_NoneLeavesFieldEmpty(t *testing.T) {
	results := RunContracts([]spec.Contract{{Name: "c", Scenarios: []spec.Scenario{{Name: "s", Steps: trueSteps}}}})
	if results[0].Covers != nil {
		t.Errorf("covers = %v, want nil", results[0].Covers)
	}
}

func TestCovers_ContractSetupFailureKeepsCovers(t *testing.T) {
	results := RunContracts([]spec.Contract{{
		Name:      "c",
		Covers:    []string{"x"},
		Setup:     []spec.Step{{Action: "cli", Command: "/no/such/binary"}},
		Scenarios: []spec.Scenario{{Name: "a", Covers: []string{"y"}}, {Name: "b"}},
	}})
	if !reflect.DeepEqual(results[0].Covers, []string{"x", "y"}) || !reflect.DeepEqual(results[1].Covers, []string{"x"}) {
		t.Errorf("covers = %v, %v", results[0].Covers, results[1].Covers)
	}
}

func TestCovers_RetriedScenarioKeepsCovers(t *testing.T) {
	dir := t.TempDir()
	got := runIn(t, dir, spec.Contract{Name: "c", Retries: 2, Covers: []string{"x"},
		Scenarios: []spec.Scenario{{Name: "s", Steps: shSteps(failOnceScript)}}})[0]
	if got.Status != evidence.StatusFlaky || !reflect.DeepEqual(got.Covers, []string{"x"}) {
		t.Errorf("got status %q covers %v", got.Status, got.Covers)
	}
}

func TestCovers_CheckCarriesSortedDeduplicatedCovers(t *testing.T) {
	results := RunChecks([]spec.CheckRunner{{Name: "unit", Kind: "command", Run: "true", Covers: []string{"b", "a", "b"}}}, false)
	if !reflect.DeepEqual(results[0].Covers, []string{"a", "b"}) {
		t.Errorf("covers = %v, want [a b]", results[0].Covers)
	}
}
