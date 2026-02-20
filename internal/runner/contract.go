package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

	// Track the last command/response result for assertions.
	var lastExitCode int
	var lastOutput string
	var lastHTTPStatus int

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

		case "http":
			statusCode, body, err := runHTTPStep(step)
			if err != nil {
				return evidence.ContractResult{
					Name:   name,
					Status: "fail",
					Error:  fmt.Sprintf("step http %s %s: %v", step.Method, step.URL, err),
				}
			}
			lastHTTPStatus = statusCode
			lastExitCode = 0
			lastOutput = body

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
			if step.Expect.Status != nil && lastHTTPStatus != *step.Expect.Status {
				return evidence.ContractResult{
					Name:   name,
					Status: "fail",
					Error:  fmt.Sprintf("expected HTTP status %d, got %d", *step.Expect.Status, lastHTTPStatus),
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

		case "poll":
			statusCode, body, err := runPollStep(step)
			if err != nil {
				return evidence.ContractResult{
					Name:   name,
					Status: "fail",
					Error:  fmt.Sprintf("step poll %s: %v", step.URL, err),
				}
			}
			lastHTTPStatus = statusCode
			lastExitCode = 0
			lastOutput = body

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

func runHTTPStep(step spec.Step) (statusCode int, body string, err error) {
	method := step.Method
	if method == "" {
		method = "GET"
	}

	var reqBody io.Reader
	if step.Body != "" {
		reqBody = strings.NewReader(step.Body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, step.URL, reqBody)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range step.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("failed to read response body: %w", err)
	}

	return resp.StatusCode, string(respBody), nil
}

func runPollStep(step spec.Step) (statusCode int, body string, err error) {
	interval := 1 * time.Second
	timeout := 30 * time.Second

	if step.Interval != "" {
		if d, e := parseDuration(step.Interval); e == nil {
			interval = d
		}
	}
	if step.Timeout != "" {
		if d, e := parseDuration(step.Timeout); e == nil {
			timeout = d
		}
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), interval)
		req, err := http.NewRequestWithContext(ctx, "GET", step.URL, nil)
		if err != nil {
			cancel()
			return 0, "", err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			time.Sleep(interval)
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if step.Expect != nil {
			if step.Expect.Status != nil && resp.StatusCode == *step.Expect.Status {
				return resp.StatusCode, string(respBody), nil
			}
			if step.Expect.BodyContains != "" && strings.Contains(string(respBody), step.Expect.BodyContains) {
				return resp.StatusCode, string(respBody), nil
			}
		}

		// If no specific expectation, a 200 is success.
		if step.Expect == nil && resp.StatusCode == 200 {
			return resp.StatusCode, string(respBody), nil
		}

		time.Sleep(interval)
	}

	return 0, "", fmt.Errorf("poll timed out after %s", timeout)
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
