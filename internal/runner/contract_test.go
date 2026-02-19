package runner

import (
	"testing"

	"github.com/aschepis/maslow-agentic/internal/spec"
)

func TestRunContracts_CLIPass(t *testing.T) {
	exitZero := 0
	contracts := []spec.Contract{
		{
			Name: "echo-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "echo-passes",
					Steps: []spec.Step{
						{Action: "cli", Command: "echo", Args: []string{"hello"}},
						{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}
}

func TestRunContracts_CLIFail(t *testing.T) {
	exitZero := 0
	contracts := []spec.Contract{
		{
			Name: "false-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "false-fails",
					Steps: []spec.Step{
						{Action: "cli", Command: "false"},
						{Action: "assert", Expect: &spec.Expectation{ExitCode: &exitZero}},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
}

func TestRunContracts_BodyContains(t *testing.T) {
	exitZero := 0
	contracts := []spec.Contract{
		{
			Name: "grep-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "output-check",
					Steps: []spec.Step{
						{Action: "cli", Command: "echo", Args: []string{"hello world"}},
						{Action: "assert", Expect: &spec.Expectation{
							ExitCode:     &exitZero,
							BodyContains: "hello",
						}},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}
}

func TestRunContracts_HTTPSkipped(t *testing.T) {
	contracts := []spec.Contract{
		{
			Name: "http-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "http-step",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: "http://localhost:8080"},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "skip" {
		t.Errorf("expected skip for http, got %s", results[0].Status)
	}
}
