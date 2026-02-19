package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/spec"
)

// RunContracts executes all contract scenarios and returns results.
func RunContracts(contracts []spec.Contract) []evidence.ContractResult {
	var results []evidence.ContractResult
	for _, contract := range contracts {
		for _, scenario := range contract.Scenarios {
			result := runScenario(contract.Name, scenario)
			results = append(results, result)
		}
	}
	return results
}

func runScenario(contractName string, scenario spec.Scenario) evidence.ContractResult {
	name := fmt.Sprintf("%s/%s", contractName, scenario.Name)

	// Track the last command result for assertions.
	var lastExitCode int
	var lastOutput string

	for _, step := range scenario.Steps {
		switch step.Action {
		case "cli":
			exitCode, output, err := runCLIStep(step)
			if err != nil {
				return evidence.ContractResult{
					Name:   name,
					Status: "fail",
					Error:  fmt.Sprintf("step cli %q: %v", step.Command, err),
				}
			}
			lastExitCode = exitCode
			lastOutput = output

		case "assert":
			if step.Expect == nil {
				return evidence.ContractResult{
					Name:   name,
					Status: "fail",
					Error:  "assert step missing expect",
				}
			}
			if err := checkExpectation(step.Expect, lastExitCode, lastOutput); err != nil {
				return evidence.ContractResult{
					Name:   name,
					Status: "fail",
					Error:  err.Error(),
				}
			}

		case "wait":
			if step.Duration != "" {
				if d, err := parseDuration(step.Duration); err == nil {
					time.Sleep(d)
				}
			}

		case "capture":
			// Capture is a no-op in the basic implementation.
			// Future: store lastOutput[step.From] as variable step.As.

		case "http", "poll":
			// HTTP and poll steps are not yet implemented.
			return evidence.ContractResult{
				Name:   name,
				Status: "skip",
				Error:  fmt.Sprintf("step action %q not yet implemented", step.Action),
			}

		default:
			return evidence.ContractResult{
				Name:   name,
				Status: "fail",
				Error:  fmt.Sprintf("unknown step action: %s", step.Action),
			}
		}
	}

	return evidence.ContractResult{
		Name:   name,
		Status: "pass",
	}
}

func runCLIStep(step spec.Step) (exitCode int, output string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := step.Args
	cmd := exec.CommandContext(ctx, step.Command, args...)
	if step.Stdin != "" {
		cmd.Stdin = strings.NewReader(step.Stdin)
	}

	out, runErr := cmd.CombinedOutput()
	output = string(out)

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return exitErr.ExitCode(), output, nil
		}
		return -1, output, runErr
	}

	return 0, output, nil
}

func checkExpectation(expect *spec.Expectation, exitCode int, output string) error {
	if expect.ExitCode != nil {
		if exitCode != *expect.ExitCode {
			return fmt.Errorf("expected exit code %d, got %d", *expect.ExitCode, exitCode)
		}
	}

	if expect.BodyContains != "" {
		if !strings.Contains(output, expect.BodyContains) {
			return fmt.Errorf("expected output to contain %q", expect.BodyContains)
		}
	}

	if expect.Status != nil {
		// Status is for HTTP responses; skip for CLI.
	}

	return nil
}
