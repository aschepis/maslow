package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/spec"
)

// stepState tracks the result of the last executed step for use by assertions.
type stepState struct {
	exitCode    int
	output      string
	httpStatus  int
	httpHeaders http.Header
}

// RunContracts executes all contract scenarios and returns results.
func RunContracts(contracts []spec.Contract) []evidence.ContractResult {
	var results []evidence.ContractResult
	for _, contract := range contracts {
		// Run contract-level setup once before all scenarios.
		if len(contract.Setup) > 0 {
			state := &stepState{}
			vars := make(map[string]string)
			if err := runSteps(contract.Setup, state, vars); err != nil {
				// Contract setup failed — mark all scenarios as failed, run teardown.
				for _, scenario := range contract.Scenarios {
					results = append(results, evidence.ContractResult{
						Name:   fmt.Sprintf("%s/%s", contract.Name, scenario.Name),
						Status: "fail",
						Error:  fmt.Sprintf("contract setup failed: %v", err),
					})
				}
				runStepsIgnoreError(contract.Teardown, state, vars)
				continue
			}
		}

		for _, scenario := range contract.Scenarios {
			result := runScenario(contract.Name, scenario)
			results = append(results, result)
		}

		// Run contract-level teardown once after all scenarios.
		if len(contract.Teardown) > 0 {
			state := &stepState{}
			vars := make(map[string]string)
			runStepsIgnoreError(contract.Teardown, state, vars)
		}
	}
	return results
}

func runScenario(contractName string, scenario spec.Scenario) evidence.ContractResult {
	name := fmt.Sprintf("%s/%s", contractName, scenario.Name)
	state := &stepState{}
	vars := make(map[string]string)

	// Run scenario-level setup.
	if len(scenario.Setup) > 0 {
		if err := runSteps(scenario.Setup, state, vars); err != nil {
			runStepsIgnoreError(scenario.Teardown, state, vars)
			return evidence.ContractResult{
				Name:   name,
				Status: "fail",
				Error:  fmt.Sprintf("setup failed: %v", err),
			}
		}
	}

	// Run scenario steps.
	stepErr := runSteps(scenario.Steps, state, vars)

	// Run scenario-level teardown (always, even on failure).
	if len(scenario.Teardown) > 0 {
		runStepsIgnoreError(scenario.Teardown, state, vars)
	}

	if stepErr != nil {
		return evidence.ContractResult{
			Name:   name,
			Status: "fail",
			Error:  stepErr.Error(),
		}
	}

	return evidence.ContractResult{
		Name:   name,
		Status: "pass",
	}
}

// runSteps executes a sequence of steps, returning the first error encountered.
func runSteps(steps []spec.Step, state *stepState, vars map[string]string) error {
	for _, step := range steps {
		var err error
		step, err = substituteStep(step, vars)
		if err != nil {
			return fmt.Errorf("variable substitution: %w", err)
		}

		switch step.Action {
		case "cli":
			exitCode, output, err := runCLIStep(step)
			if err != nil {
				return fmt.Errorf("step cli %q: %v", step.Command, err)
			}
			state.exitCode = exitCode
			state.output = output

		case "http":
			statusCode, headers, body, err := runHTTPStep(step)
			if err != nil {
				return fmt.Errorf("step http %s %s: %v", step.Method, step.URL, err)
			}
			state.httpStatus = statusCode
			state.httpHeaders = headers
			state.exitCode = 0
			state.output = body

			if step.Expect != nil {
				if err := checkExpectation(step.Expect, state); err != nil {
					return fmt.Errorf("step http %s %s: %v", step.Method, step.URL, err)
				}
			}

		case "assert":
			if step.Expect == nil {
				return fmt.Errorf("assert step missing expect")
			}
			if err := checkExpectation(step.Expect, state); err != nil {
				return err
			}

		case "wait":
			if step.Duration != "" {
				if d, err := parseDuration(step.Duration); err == nil {
					time.Sleep(d)
				}
			}

		case "capture":
			val, err := captureValue(step, state)
			if err != nil {
				return fmt.Errorf("capture %q as %q: %v", step.From, step.As, err)
			}
			if step.As == "" {
				return fmt.Errorf("capture step missing 'as' field")
			}
			vars[step.As] = val

		case "poll":
			statusCode, headers, body, err := runPollStep(step)
			if err != nil {
				return fmt.Errorf("step poll %s: %v", step.URL, err)
			}
			state.httpStatus = statusCode
			state.httpHeaders = headers
			state.exitCode = 0
			state.output = body

		default:
			return fmt.Errorf("unknown step action: %s", step.Action)
		}
	}
	return nil
}

// runStepsIgnoreError runs steps but does not propagate errors (for teardown).
func runStepsIgnoreError(steps []spec.Step, state *stepState, vars map[string]string) {
	if len(steps) == 0 {
		return
	}
	_ = runSteps(steps, state, vars)
}

// captureValue extracts a value from the current step state based on the capture's "from" field.
// Supported from values:
//   - A JSON path (e.g., "$.access_token") — extracts from the response/output body
//   - "output" — the raw output string
//   - "status" — the HTTP status code as a string
//   - "header:<name>" — a specific HTTP response header value
func captureValue(step spec.Step, state *stepState) (string, error) {
	from := step.From
	if from == "" {
		return "", fmt.Errorf("capture step missing 'from' field")
	}

	switch {
	case from == "output":
		return state.output, nil

	case from == "status":
		return strconv.Itoa(state.httpStatus), nil

	case strings.HasPrefix(from, "header:"):
		headerName := strings.TrimPrefix(from, "header:")
		if state.httpHeaders == nil {
			return "", fmt.Errorf("no HTTP headers available to capture header %q", headerName)
		}
		val := state.httpHeaders.Get(headerName)
		if val == "" {
			return "", fmt.Errorf("header %q not found in response", headerName)
		}
		return val, nil

	case strings.HasPrefix(from, "$"):
		// JSON path extraction from output body.
		result, err := evaluateJSONPath(state.output, from)
		if err != nil {
			return "", fmt.Errorf("json path %q: %w", from, err)
		}
		// Convert to string for variable storage.
		switch v := result.(type) {
		case string:
			return v, nil
		case float64:
			if v == float64(int(v)) {
				return strconv.Itoa(int(v)), nil
			}
			return strconv.FormatFloat(v, 'f', -1, 64), nil
		case bool:
			return strconv.FormatBool(v), nil
		case nil:
			return "", fmt.Errorf("json path %q resolved to null", from)
		default:
			// For objects/arrays, marshal back to JSON.
			b, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("json path %q: failed to serialize result: %w", from, err)
			}
			return string(b), nil
		}

	default:
		return "", fmt.Errorf("unknown capture source %q (expected JSON path like $.field, \"output\", \"status\", or \"header:<name>\")", from)
	}
}

// substituteStep applies Go text/template substitution to all string fields in a step.
// Use {{.env.NAME}} for environment variables and {{.cap.NAME}} for captured variables.
func substituteStep(step spec.Step, vars map[string]string) (spec.Step, error) {
	var err error

	step.URL, err = applyTemplate(step.URL, vars)
	if err != nil {
		return step, fmt.Errorf("url: %w", err)
	}
	step.Body, err = applyTemplate(step.Body, vars)
	if err != nil {
		return step, fmt.Errorf("body: %w", err)
	}
	step.Command, err = applyTemplate(step.Command, vars)
	if err != nil {
		return step, fmt.Errorf("command: %w", err)
	}
	step.Stdin, err = applyTemplate(step.Stdin, vars)
	if err != nil {
		return step, fmt.Errorf("stdin: %w", err)
	}

	if len(step.Headers) > 0 {
		headers := make(map[string]string, len(step.Headers))
		for k, v := range step.Headers {
			headers[k], err = applyTemplate(v, vars)
			if err != nil {
				return step, fmt.Errorf("header %s: %w", k, err)
			}
		}
		step.Headers = headers
	}

	if len(step.Args) > 0 {
		args := make([]string, len(step.Args))
		for i, a := range step.Args {
			args[i], err = applyTemplate(a, vars)
			if err != nil {
				return step, fmt.Errorf("arg[%d]: %w", i, err)
			}
		}
		step.Args = args
	}

	return step, nil
}

// applyTemplate applies Go text/template substitution to a string.
// Returns the original string unchanged if it contains no template markers.
func applyTemplate(s string, vars map[string]string) (string, error) {
	if s == "" || !strings.Contains(s, "{{") {
		return s, nil
	}

	data := map[string]any{
		"env": buildEnvMap(),
		"cap": vars,
	}

	tmpl, err := template.New("").Option("missingkey=error").Parse(s)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// buildEnvMap creates a map of all environment variables for template access.
func buildEnvMap() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if i := strings.IndexByte(e, '='); i >= 0 {
			env[e[:i]] = e[i+1:]
		}
	}
	return env
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

func runHTTPStep(step spec.Step) (statusCode int, headers http.Header, body string, err error) {
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
		return 0, nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range step.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, resp.Header, "", fmt.Errorf("failed to read response body: %w", err)
	}

	return resp.StatusCode, resp.Header, string(respBody), nil
}

func runPollStep(step spec.Step) (statusCode int, headers http.Header, body string, err error) {
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
			return 0, nil, "", err
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
				return resp.StatusCode, resp.Header, string(respBody), nil
			}
			if step.Expect.BodyContains != "" && strings.Contains(string(respBody), step.Expect.BodyContains) {
				return resp.StatusCode, resp.Header, string(respBody), nil
			}
		}

		// If no specific expectation, a 200 is success.
		if step.Expect == nil && resp.StatusCode == 200 {
			return resp.StatusCode, resp.Header, string(respBody), nil
		}

		time.Sleep(interval)
	}

	return 0, nil, "", fmt.Errorf("poll timed out after %s", timeout)
}

// checkExpectation validates all expectation fields against the current step state.
func checkExpectation(expect *spec.Expectation, state *stepState) error {
	if expect.ExitCode != nil {
		if state.exitCode != *expect.ExitCode {
			return fmt.Errorf("expected exit code %d, got %d", *expect.ExitCode, state.exitCode)
		}
	}

	if expect.Status != nil {
		if state.httpStatus != *expect.Status {
			return fmt.Errorf("expected HTTP status %d, got %d", *expect.Status, state.httpStatus)
		}
	}

	if expect.BodyContains != "" {
		if !strings.Contains(state.output, expect.BodyContains) {
			return fmt.Errorf("expected body to contain %q", expect.BodyContains)
		}
	}

	if expect.BodyMatches != "" {
		re, err := regexp.Compile(expect.BodyMatches)
		if err != nil {
			return fmt.Errorf("invalid regex %q: %w", expect.BodyMatches, err)
		}
		if !re.MatchString(state.output) {
			return fmt.Errorf("expected body to match regex %q", expect.BodyMatches)
		}
	}

	if len(expect.Headers) > 0 {
		if state.httpHeaders == nil {
			return fmt.Errorf("expected response headers but no HTTP response available")
		}
		for k, v := range expect.Headers {
			actual := state.httpHeaders.Get(k)
			if actual == "" {
				return fmt.Errorf("expected header %q to be %q, but header is missing", k, v)
			}
			if !strings.EqualFold(actual, v) {
				return fmt.Errorf("expected header %q to be %q, got %q", k, v, actual)
			}
		}
	}

	if expect.JSONPath != "" {
		result, err := evaluateJSONPath(state.output, expect.JSONPath)
		if err != nil {
			return fmt.Errorf("json_path %q: %w", expect.JSONPath, err)
		}
		if expect.Value != nil {
			if !jsonValuesEqual(result, expect.Value) {
				return fmt.Errorf("json_path %q: expected %v, got %v", expect.JSONPath, expect.Value, result)
			}
		}
	}

	for i, assertion := range expect.Assertions {
		result, err := evaluateJSONPath(state.output, assertion.JSONPath)
		if err != nil {
			return fmt.Errorf("assertions[%d] json_path %q: %w", i, assertion.JSONPath, err)
		}
		if assertion.Value != nil {
			if !jsonValuesEqual(result, assertion.Value) {
				return fmt.Errorf("assertions[%d] json_path %q: expected %v, got %v", i, assertion.JSONPath, assertion.Value, result)
			}
		}
	}

	return nil
}

// evaluateJSONPath resolves a simple JSON path expression against a JSON string.
// Supports dotted paths ($.field.nested), array indexing ($.items[0]), and
// root array indexing ($[0].field).
func evaluateJSONPath(jsonStr string, path string) (any, error) {
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Normalize: strip leading "$" or "$."
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")

	if path == "" {
		return data, nil
	}

	return resolveJSONPath(data, path)
}

func resolveJSONPath(data any, path string) (any, error) {
	if path == "" {
		return data, nil
	}

	// Handle array index at start: [0].rest or [0]
	if strings.HasPrefix(path, "[") {
		closeBracket := strings.Index(path, "]")
		if closeBracket == -1 {
			return nil, fmt.Errorf("unclosed bracket in path")
		}
		indexStr := path[1:closeBracket]
		idx, err := strconv.Atoi(indexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid array index %q", indexStr)
		}

		arr, ok := data.([]any)
		if !ok {
			return nil, fmt.Errorf("expected array, got %T", data)
		}
		if idx < 0 || idx >= len(arr) {
			return nil, fmt.Errorf("array index %d out of bounds (length %d)", idx, len(arr))
		}

		rest := path[closeBracket+1:]
		rest = strings.TrimPrefix(rest, ".")
		return resolveJSONPath(arr[idx], rest)
	}

	// Handle dotted field: field.rest or field[0].rest
	// Find the next separator (dot or bracket).
	nextDot := strings.Index(path, ".")
	nextBracket := strings.Index(path, "[")

	var field string
	var rest string

	switch {
	case nextDot == -1 && nextBracket == -1:
		field = path
		rest = ""
	case nextBracket != -1 && (nextDot == -1 || nextBracket < nextDot):
		field = path[:nextBracket]
		rest = path[nextBracket:]
	default:
		field = path[:nextDot]
		rest = path[nextDot+1:]
	}

	obj, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object, got %T at field %q", data, field)
	}

	val, exists := obj[field]
	if !exists {
		return nil, fmt.Errorf("field %q not found", field)
	}

	return resolveJSONPath(val, rest)
}

// jsonValuesEqual compares a JSON-parsed value against an expected value,
// handling type coercion between YAML-parsed expectations and JSON-parsed responses.
func jsonValuesEqual(got any, expected any) bool {
	// Direct equality.
	if fmt.Sprintf("%v", got) == fmt.Sprintf("%v", expected) {
		return true
	}

	// Numeric comparison: JSON numbers are float64, YAML may parse as int.
	switch g := got.(type) {
	case float64:
		switch e := expected.(type) {
		case int:
			return g == float64(e)
		case float64:
			return g == e
		}
	case string:
		if e, ok := expected.(string); ok {
			return g == e
		}
	case bool:
		if e, ok := expected.(bool); ok {
			return g == e
		}
	}

	return false
}
