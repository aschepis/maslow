package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// stepState tracks the result of the last executed step for use by assertions.
type stepState struct {
	exitCode    int
	output      string
	httpStatus  int
	httpHeaders http.Header
}

// Phases name the step list a failing step belongs to in the evidence.
const (
	phaseContractSetup = "contract_setup"
	phaseSetup         = "setup"
	phaseSteps         = "steps"
)

// stepFailure says which step failed and keeps the output it produced.
type stepFailure struct {
	phase  string
	index  int
	step   spec.Step
	output string
	err    error
}

func (f *stepFailure) Error() string { return f.err.Error() }

func (f *stepFailure) evidence() *evidence.FailedStep {
	return &evidence.FailedStep{
		Phase:  f.phase,
		Index:  f.index,
		Action: f.step.Action,
		Target: stepTarget(f.step),
	}
}

// stepTarget names what a step acted on, after variable substitution.
func stepTarget(step spec.Step) string {
	switch step.Action {
	case "cli":
		return strings.Join(append([]string{step.Command}, step.Args...), " ")
	case "http":
		method := step.Method
		if method == "" {
			method = "GET"
		}
		return method + " " + step.URL
	case "poll":
		return "GET " + step.URL
	case "capture":
		return step.From
	}
	return ""
}

// scope is what the steps of one scenario share.
type scope struct {
	// env holds only the environment variables the contract allow-lists.
	env  map[string]string
	vars map[string]string
	// client keeps cookies between HTTP steps, so a scenario can sign in once.
	client *http.Client
}

func newScope(env map[string]string) *scope {
	jar, _ := cookiejar.New(nil) // New never fails without options.
	return &scope{env: env, vars: map[string]string{}, client: &http.Client{Jar: jar}}
}

// RunContracts executes all contract scenarios and returns results.
func RunContracts(contracts []spec.Contract) []evidence.ContractResult {
	var results []evidence.ContractResult
	for _, contract := range contracts {
		env := allowedEnv(contract.Env)
		for _, result := range runContract(contract, env) {
			results = append(results, redact(result, env))
		}
	}
	return results
}

func runContract(contract spec.Contract, env map[string]string) []evidence.ContractResult {
	// Contract-level setup runs once before all scenarios.
	if len(contract.Setup) > 0 {
		state, sc := &stepState{}, newScope(env)
		if err := runSteps(phaseContractSetup, contract.Setup, state, sc); err != nil {
			var results []evidence.ContractResult
			for _, scenario := range contract.Scenarios {
				result := failedResult(fmt.Sprintf("%s/%s", contract.Name, scenario.Name), "contract setup failed: ", err)
				attachArtifacts(&result, scenario.Artifacts)
				results = append(results, result)
			}
			runStepsIgnoreError(contract.Teardown, state, sc)
			return results
		}
	}

	var results []evidence.ContractResult
	for _, scenario := range contract.Scenarios {
		results = append(results, runScenario(contract.Name, scenario, env))
	}

	runStepsIgnoreError(contract.Teardown, &stepState{}, newScope(env))
	return results
}

func runScenario(contractName string, scenario spec.Scenario, env map[string]string) evidence.ContractResult {
	name := fmt.Sprintf("%s/%s", contractName, scenario.Name)
	state, sc := &stepState{}, newScope(env)

	var result evidence.ContractResult
	if err := runSteps(phaseSetup, scenario.Setup, state, sc); err != nil {
		result = failedResult(name, "setup failed: ", err)
	} else if err := runSteps(phaseSteps, scenario.Steps, state, sc); err != nil {
		result = failedResult(name, "", err)
	} else {
		result = evidence.ContractResult{Name: name, Status: "pass"}
	}

	// Collected before teardown, which may clean the artifacts up.
	attachArtifacts(&result, scenario.Artifacts)
	runStepsIgnoreError(scenario.Teardown, state, sc)
	return result
}

func failedResult(name, prefix string, err error) evidence.ContractResult {
	result := evidence.ContractResult{Name: name, Status: "fail", Error: prefix + err.Error()}
	if f, ok := err.(*stepFailure); ok {
		result.FailedStep = f.evidence()
		result.Output = f.output
	}
	return result
}

// runSteps executes a sequence of steps, returning a *stepFailure for the first one that fails.
func runSteps(phase string, steps []spec.Step, state *stepState, sc *scope) error {
	for i, step := range steps {
		if err := runStep(step, state, sc); err != nil {
			return &stepFailure{phase: phase, index: i, step: substitutedOrRaw(step, sc), output: state.output, err: err}
		}
	}
	return nil
}

// substitutedOrRaw shows the step as it ran, or as written if substitution failed.
func substitutedOrRaw(step spec.Step, sc *scope) spec.Step {
	if s, err := substituteStep(step, sc); err == nil {
		return s
	}
	return step
}

func runStep(step spec.Step, state *stepState, sc *scope) error {
	step, err := substituteStep(step, sc)
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
		statusCode, headers, body, err := runHTTPStep(step, sc.client)
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
		sc.vars[step.As] = val

	case "poll":
		statusCode, headers, body, err := runPollStep(step, sc.client)
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
	return nil
}

// runStepsIgnoreError runs steps but does not propagate errors (for teardown).
func runStepsIgnoreError(steps []spec.Step, state *stepState, sc *scope) {
	_ = runSteps("teardown", steps, state, sc)
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
func substituteStep(step spec.Step, sc *scope) (spec.Step, error) {
	var err error

	step.URL, err = applyTemplate(step.URL, sc)
	if err != nil {
		return step, fmt.Errorf("url: %w", err)
	}
	step.Body, err = applyTemplate(step.Body, sc)
	if err != nil {
		return step, fmt.Errorf("body: %w", err)
	}
	step.Command, err = applyTemplate(step.Command, sc)
	if err != nil {
		return step, fmt.Errorf("command: %w", err)
	}
	step.Stdin, err = applyTemplate(step.Stdin, sc)
	if err != nil {
		return step, fmt.Errorf("stdin: %w", err)
	}

	if len(step.Headers) > 0 {
		headers := make(map[string]string, len(step.Headers))
		for k, v := range step.Headers {
			headers[k], err = applyTemplate(v, sc)
			if err != nil {
				return step, fmt.Errorf("header %s: %w", k, err)
			}
		}
		step.Headers = headers
	}

	if len(step.Args) > 0 {
		args := make([]string, len(step.Args))
		for i, a := range step.Args {
			args[i], err = applyTemplate(a, sc)
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
func applyTemplate(s string, sc *scope) (string, error) {
	if s == "" || !strings.Contains(s, "{{") {
		return s, nil
	}

	data := map[string]any{
		"env": sc.env,
		"cap": sc.vars,
	}

	tmpl, err := template.New("").Option("missingkey=error").Parse(s)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		if strings.Contains(err.Error(), ".env.") {
			return "", fmt.Errorf("%w (a contract reads only the variables in its env list, and they must be set)", err)
		}
		return "", err
	}

	return buf.String(), nil
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

func runHTTPStep(step spec.Step, client *http.Client) (statusCode int, headers http.Header, body string, err error) {
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

	resp, err := client.Do(req)
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

func runPollStep(step spec.Step, client *http.Client) (statusCode int, headers http.Header, body string, err error) {
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

		resp, err := client.Do(req)
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

	// Collection assertions: evaluate against the array at json_path.
	if expect.MinLength != nil || expect.MaxLength != nil || expect.Length != nil || expect.Every != nil || expect.Some != nil {
		if expect.JSONPath == "" {
			return fmt.Errorf("collection assertions require json_path to identify the target array")
		}
		arrVal, err := evaluateJSONPath(state.output, expect.JSONPath)
		if err != nil {
			return fmt.Errorf("collection assertion json_path %q: %w", expect.JSONPath, err)
		}
		arr, ok := arrVal.([]any)
		if !ok {
			return fmt.Errorf("expected array at %s, got %T", expect.JSONPath, arrVal)
		}

		if expect.Length != nil && len(arr) != *expect.Length {
			return fmt.Errorf("expected array at %s to have exactly %d items, got %d", expect.JSONPath, *expect.Length, len(arr))
		}
		if expect.MinLength != nil && len(arr) < *expect.MinLength {
			return fmt.Errorf("expected array at %s to have at least %d items, got %d", expect.JSONPath, *expect.MinLength, len(arr))
		}
		if expect.MaxLength != nil && len(arr) > *expect.MaxLength {
			return fmt.Errorf("expected array at %s to have at most %d items, got %d", expect.JSONPath, *expect.MaxLength, len(arr))
		}

		if expect.Every != nil {
			for i, item := range arr {
				itemJSON, err := json.Marshal(item)
				if err != nil {
					return fmt.Errorf("every[%d]: failed to serialize array item: %w", i, err)
				}
				result, err := evaluateJSONPath(string(itemJSON), expect.Every.JSONPath)
				if err != nil {
					return fmt.Errorf("every[%d] json_path %q: %w", i, expect.Every.JSONPath, err)
				}
				if expect.Every.Value != nil {
					if !jsonValuesEqual(result, expect.Every.Value) {
						return fmt.Errorf("every[%d] json_path %q: expected %v, got %v", i, expect.Every.JSONPath, expect.Every.Value, result)
					}
				}
			}
		}

		if expect.Some != nil {
			matched := false
			for _, item := range arr {
				itemJSON, err := json.Marshal(item)
				if err != nil {
					continue
				}
				result, err := evaluateJSONPath(string(itemJSON), expect.Some.JSONPath)
				if err != nil {
					continue
				}
				if expect.Some.Value == nil {
					// Existence check — the path resolved successfully.
					matched = true
					break
				}
				if jsonValuesEqual(result, expect.Some.Value) {
					matched = true
					break
				}
			}
			if !matched {
				if expect.Some.Value != nil {
					return fmt.Errorf("no item in array at %s matched json_path %q with value %v", expect.JSONPath, expect.Some.JSONPath, expect.Some.Value)
				}
				return fmt.Errorf("no item in array at %s had json_path %q", expect.JSONPath, expect.Some.JSONPath)
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
