package runner

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestRunContracts_HTTPPass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "http-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "http-get",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL},
						{Action: "assert", Expect: &spec.Expectation{
							Status:       &status200,
							BodyContains: "ok",
						}},
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

func TestRunContracts_HTTPFail(t *testing.T) {
	// Connect to a port that's not listening.
	contracts := []spec.Contract{
		{
			Name: "http-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "http-fail",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: "http://127.0.0.1:1"},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "fail" {
		t.Errorf("expected fail for unreachable http, got %s", results[0].Status)
	}
}

func TestRunContracts_HTTPPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(201)
		w.Write([]byte(`{"id":1,"title":"test"}`))
	}))
	defer srv.Close()

	status201 := 201
	contracts := []spec.Contract{
		{
			Name: "post-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "post-create",
					Steps: []spec.Step{
						{
							Action:  "http",
							Method:  "POST",
							URL:     srv.URL,
							Headers: map[string]string{"Content-Type": "application/json"},
							Body:    `{"title":"test"}`,
						},
						{Action: "assert", Expect: &spec.Expectation{
							Status:       &status201,
							BodyContains: "test",
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

// --- Capture & Variable Substitution Tests ---

func TestRunContracts_CaptureFromHTTPBody(t *testing.T) {
	// Server returns a token on POST /login, then expects it in Authorization header on GET /me.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"access_token":"tok_abc123","token_type":"bearer"}`))
		case "/me":
			auth := r.Header.Get("Authorization")
			if auth != "Bearer tok_abc123" {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"id":1,"name":"Alice"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "auth-flow",
			Scenarios: []spec.Scenario{
				{
					Name: "login-then-get-profile",
					Steps: []spec.Step{
						{
							Action:  "http",
							Method:  "POST",
							URL:     srv.URL + "/login",
							Headers: map[string]string{"Content-Type": "application/json"},
							Body:    `{"email":"alice@example.com","password":"secret"}`,
							Expect:  &spec.Expectation{Status: &status200},
						},
						{Action: "capture", From: "$.access_token", As: "token"},
						{
							Action:  "http",
							Method:  "GET",
							URL:     srv.URL + "/me",
							Headers: map[string]string{"Authorization": "Bearer ${{token}}"},
							Expect: &spec.Expectation{
								Status:   &status200,
								JSONPath: "$.name",
								Value:    "Alice",
							},
						},
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

func TestRunContracts_CaptureFromHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-42")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	contracts := []spec.Contract{
		{
			Name: "header-capture",
			Scenarios: []spec.Scenario{
				{
					Name: "capture-request-id",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL},
						{Action: "capture", From: "header:X-Request-Id", As: "req_id"},
						{Action: "assert", Expect: &spec.Expectation{BodyContains: "ok"}},
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

func TestRunContracts_CaptureFromCLIOutput(t *testing.T) {
	contracts := []spec.Contract{
		{
			Name: "cli-capture",
			Scenarios: []spec.Scenario{
				{
					Name: "capture-echo-output",
					Steps: []spec.Step{
						{Action: "cli", Command: "echo", Args: []string{"hello-world"}},
						{Action: "capture", From: "output", As: "greeting"},
						// Use captured var in a CLI arg (echo it back and check).
						{Action: "cli", Command: "echo", Args: []string{"got: ${{greeting}}"}},
						{Action: "assert", Expect: &spec.Expectation{BodyContains: "got: hello-world"}},
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

func TestRunContracts_CaptureFromHTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	contracts := []spec.Contract{
		{
			Name: "status-capture",
			Scenarios: []spec.Scenario{
				{
					Name: "capture-status-code",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL},
						{Action: "capture", From: "status", As: "code"},
						{Action: "cli", Command: "echo", Args: []string{"status=${{code}}"}},
						{Action: "assert", Expect: &spec.Expectation{BodyContains: "status=201"}},
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

func TestRunContracts_CaptureFailsOnMissingFrom(t *testing.T) {
	contracts := []spec.Contract{
		{
			Name: "bad-capture",
			Scenarios: []spec.Scenario{
				{
					Name: "missing-from",
					Steps: []spec.Step{
						{Action: "cli", Command: "echo", Args: []string{"hello"}},
						{Action: "capture", As: "val"},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
	if results[0].Error == "" {
		t.Error("expected error message for missing from")
	}
}

func TestRunContracts_CaptureFailsOnMissingAs(t *testing.T) {
	contracts := []spec.Contract{
		{
			Name: "bad-capture",
			Scenarios: []spec.Scenario{
				{
					Name: "missing-as",
					Steps: []spec.Step{
						{Action: "cli", Command: "echo", Args: []string{"hello"}},
						{Action: "capture", From: "output"},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
	if results[0].Error == "" {
		t.Error("expected error message for missing as")
	}
}

func TestRunContracts_CaptureJSONPathNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"name":"Alice"}`))
	}))
	defer srv.Close()

	contracts := []spec.Contract{
		{
			Name: "bad-path",
			Scenarios: []spec.Scenario{
				{
					Name: "nonexistent-field",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL},
						{Action: "capture", From: "$.nonexistent", As: "val"},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
}

func TestRunContracts_CaptureNumericJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id":42,"price":9.99}`))
	}))
	defer srv.Close()

	contracts := []spec.Contract{
		{
			Name: "numeric-capture",
			Scenarios: []spec.Scenario{
				{
					Name: "capture-int-and-float",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL},
						{Action: "capture", From: "$.id", As: "item_id"},
						{Action: "capture", From: "$.price", As: "item_price"},
						{Action: "cli", Command: "echo", Args: []string{"${{item_id}}/${{item_price}}"}},
						{Action: "assert", Expect: &spec.Expectation{BodyContains: "42/9.99"}},
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

func TestRunContracts_EnvVarSubstitution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	os.Setenv("MASLOW_TEST_BASE_URL", srv.URL)
	defer os.Unsetenv("MASLOW_TEST_BASE_URL")

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "env-var",
			Scenarios: []spec.Scenario{
				{
					Name: "uses-env-var",
					Steps: []spec.Step{
						{
							Action: "http",
							Method: "GET",
							URL:    "${MASLOW_TEST_BASE_URL}",
							Expect: &spec.Expectation{Status: &status200},
						},
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

func TestRunContracts_MixedEnvAndCapturedVars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.WriteHeader(200)
			w.Write([]byte(`{"token":"secret123"}`))
		case "/data":
			if r.Header.Get("Authorization") == "Bearer secret123" {
				w.WriteHeader(200)
				w.Write([]byte(`{"result":"success"}`))
			} else {
				w.WriteHeader(401)
				w.Write([]byte(`{"error":"unauthorized"}`))
			}
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	os.Setenv("MASLOW_TEST_API", srv.URL)
	defer os.Unsetenv("MASLOW_TEST_API")

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "mixed-vars",
			Scenarios: []spec.Scenario{
				{
					Name: "env-and-capture",
					Steps: []spec.Step{
						{
							Action: "http",
							Method: "GET",
							URL:    "${MASLOW_TEST_API}/token",
							Expect: &spec.Expectation{Status: &status200},
						},
						{Action: "capture", From: "$.token", As: "auth_token"},
						{
							Action:  "http",
							Method:  "GET",
							URL:     "${MASLOW_TEST_API}/data",
							Headers: map[string]string{"Authorization": "Bearer ${{auth_token}}"},
							Expect: &spec.Expectation{
								Status:   &status200,
								JSONPath: "$.result",
								Value:    "success",
							},
						},
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

// --- Setup & Teardown Tests ---

func TestRunContracts_ScenarioSetupAndTeardown(t *testing.T) {
	var callLog []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callLog = append(callLog, r.URL.Path)
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "fixture-test",
			Scenarios: []spec.Scenario{
				{
					Name: "with-setup-teardown",
					Setup: []spec.Step{
						{Action: "http", Method: "POST", URL: srv.URL + "/seed"},
					},
					Teardown: []spec.Step{
						{Action: "http", Method: "POST", URL: srv.URL + "/reset"},
					},
					Steps: []spec.Step{
						{
							Action: "http", Method: "GET", URL: srv.URL + "/data",
							Expect: &spec.Expectation{Status: &status200},
						},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}

	// Verify call order: seed -> data -> reset
	if len(callLog) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(callLog), callLog)
	}
	if callLog[0] != "/seed" || callLog[1] != "/data" || callLog[2] != "/reset" {
		t.Errorf("unexpected call order: %v", callLog)
	}
}

func TestRunContracts_TeardownRunsOnStepFailure(t *testing.T) {
	teardownCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/teardown" {
			teardownCalled = true
		}
		w.WriteHeader(500) // All requests fail
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "teardown-on-fail",
			Scenarios: []spec.Scenario{
				{
					Name: "step-fails",
					Teardown: []spec.Step{
						{Action: "http", Method: "POST", URL: srv.URL + "/teardown"},
					},
					Steps: []spec.Step{
						{
							Action: "http", Method: "GET", URL: srv.URL + "/data",
							Expect: &spec.Expectation{Status: &status200}, // Will fail (500 != 200)
						},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
	if !teardownCalled {
		t.Error("teardown should run even when steps fail")
	}
}

func TestRunContracts_SetupFailureSkipsSteps(t *testing.T) {
	stepsCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/data" {
			stepsCalled = true
		}
		if r.URL.Path == "/seed" {
			w.WriteHeader(500) // Setup fails
			w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "setup-fail",
			Scenarios: []spec.Scenario{
				{
					Name: "bad-setup",
					Setup: []spec.Step{
						{
							Action: "http", Method: "POST", URL: srv.URL + "/seed",
							Expect: &spec.Expectation{Status: &status200}, // Will fail
						},
					},
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL + "/data"},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
	if !strings.Contains(results[0].Error, "setup failed") {
		t.Errorf("error should mention setup, got: %s", results[0].Error)
	}
	if stepsCalled {
		t.Error("scenario steps should not run when setup fails")
	}
}

func TestRunContracts_ContractLevelSetupAndTeardown(t *testing.T) {
	var callLog []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callLog = append(callLog, r.URL.Path)
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "contract-fixtures",
			Setup: []spec.Step{
				{Action: "http", Method: "POST", URL: srv.URL + "/global-seed"},
			},
			Teardown: []spec.Step{
				{Action: "http", Method: "POST", URL: srv.URL + "/global-reset"},
			},
			Scenarios: []spec.Scenario{
				{
					Name: "scenario-a",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL + "/a",
							Expect: &spec.Expectation{Status: &status200}},
					},
				},
				{
					Name: "scenario-b",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL + "/b",
							Expect: &spec.Expectation{Status: &status200}},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	for _, r := range results {
		if r.Status != "pass" {
			t.Errorf("%s: expected pass, got %s: %s", r.Name, r.Status, r.Error)
		}
	}

	// Verify: global-seed -> a -> b -> global-reset
	if len(callLog) != 4 {
		t.Fatalf("expected 4 calls, got %d: %v", len(callLog), callLog)
	}
	if callLog[0] != "/global-seed" {
		t.Errorf("first call should be /global-seed, got %s", callLog[0])
	}
	if callLog[3] != "/global-reset" {
		t.Errorf("last call should be /global-reset, got %s", callLog[3])
	}
}

func TestRunContracts_SetupCaptureAvailableInSteps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/seed":
			w.WriteHeader(200)
			w.Write([]byte(`{"user_id":"usr_42"}`))
		case "/users/usr_42":
			w.WriteHeader(200)
			w.Write([]byte(`{"id":"usr_42","name":"Test User"}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "setup-capture",
			Scenarios: []spec.Scenario{
				{
					Name: "use-seeded-id",
					Setup: []spec.Step{
						{Action: "http", Method: "POST", URL: srv.URL + "/seed"},
						{Action: "capture", From: "$.user_id", As: "uid"},
					},
					Steps: []spec.Step{
						{
							Action: "http", Method: "GET",
							URL: srv.URL + "/users/${{uid}}",
							Expect: &spec.Expectation{
								Status:   &status200,
								JSONPath: "$.name",
								Value:    "Test User",
							},
						},
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

// --- Multiple Assertions Tests ---

func TestRunContracts_MultipleAssertionsPass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"access_token":"tok_123","token_type":"bearer","expires_in":3600}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "multi-assert",
			Scenarios: []spec.Scenario{
				{
					Name: "login-response",
					Steps: []spec.Step{
						{
							Action: "http",
							Method: "POST",
							URL:    srv.URL,
							Expect: &spec.Expectation{
								Status: &status200,
								Assertions: []spec.Assertion{
									{JSONPath: "$.access_token", Value: "tok_123"},
									{JSONPath: "$.token_type", Value: "bearer"},
									{JSONPath: "$.expires_in", Value: 3600},
								},
							},
						},
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

func TestRunContracts_MultipleAssertionsOneFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"name":"Alice","role":"user"}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "multi-assert-fail",
			Scenarios: []spec.Scenario{
				{
					Name: "wrong-role",
					Steps: []spec.Step{
						{
							Action: "http",
							Method: "GET",
							URL:    srv.URL,
							Expect: &spec.Expectation{
								Status: &status200,
								Assertions: []spec.Assertion{
									{JSONPath: "$.name", Value: "Alice"},
									{JSONPath: "$.role", Value: "admin"}, // wrong
								},
							},
						},
					},
				},
			},
		},
	}

	results := RunContracts(contracts)
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
	if results[0].Error == "" {
		t.Error("expected error message")
	}
	// Error should reference the assertion index and path.
	if !strings.Contains(results[0].Error, "assertions[1]") {
		t.Errorf("error should reference assertions[1], got: %s", results[0].Error)
	}
	if !strings.Contains(results[0].Error, "$.role") {
		t.Errorf("error should reference $.role, got: %s", results[0].Error)
	}
}

func TestRunContracts_AssertionsExistenceOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id":1,"name":"Test","created_at":"2026-01-01"}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "existence-check",
			Scenarios: []spec.Scenario{
				{
					Name: "fields-exist",
					Steps: []spec.Step{
						{
							Action: "http",
							Method: "GET",
							URL:    srv.URL,
							Expect: &spec.Expectation{
								Status: &status200,
								Assertions: []spec.Assertion{
									{JSONPath: "$.id"},
									{JSONPath: "$.name"},
									{JSONPath: "$.created_at"},
								},
							},
						},
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

func TestRunContracts_AssertionsMixedWithOtherExpect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok","count":5}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "mixed",
			Scenarios: []spec.Scenario{
				{
					Name: "all-checks",
					Steps: []spec.Step{
						{
							Action: "http",
							Method: "GET",
							URL:    srv.URL,
							Expect: &spec.Expectation{
								Status:       &status200,
								BodyContains: "ok",
								Headers:      map[string]string{"Content-Type": "application/json"},
								Assertions: []spec.Assertion{
									{JSONPath: "$.status", Value: "ok"},
									{JSONPath: "$.count", Value: 5},
								},
							},
						},
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

func TestRunContracts_BackwardCompatSingleJSONPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"name":"Alice"}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "compat",
			Scenarios: []spec.Scenario{
				{
					Name: "single-json-path",
					Steps: []spec.Step{
						{
							Action: "http",
							Method: "GET",
							URL:    srv.URL,
							Expect: &spec.Expectation{
								Status:   &status200,
								JSONPath: "$.name",
								Value:    "Alice",
							},
						},
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

// --- Unit tests for substitution functions ---

func TestSubstituteEnvVars(t *testing.T) {
	os.Setenv("MASLOW_TEST_HOST", "localhost")
	defer os.Unsetenv("MASLOW_TEST_HOST")

	got := substituteEnvVars("http://${MASLOW_TEST_HOST}:8080/api")
	if got != "http://localhost:8080/api" {
		t.Errorf("expected http://localhost:8080/api, got %s", got)
	}
}

func TestSubstituteEnvVars_Unresolved(t *testing.T) {
	got := substituteEnvVars("http://${MASLOW_NONEXISTENT_VAR_12345}/api")
	if got != "http://${MASLOW_NONEXISTENT_VAR_12345}/api" {
		t.Errorf("unresolved env var should be left as-is, got %s", got)
	}
}

func TestSubstituteCapturedVars(t *testing.T) {
	vars := map[string]string{"token": "abc", "id": "42"}
	got := substituteCapturedVars("Bearer ${{token}} for user ${{id}}", vars)
	if got != "Bearer abc for user 42" {
		t.Errorf("expected 'Bearer abc for user 42', got %s", got)
	}
}

func TestSubstituteCapturedVars_Unresolved(t *testing.T) {
	vars := map[string]string{"token": "abc"}
	got := substituteCapturedVars("${{token}} and ${{missing}}", vars)
	if got != "abc and ${{missing}}" {
		t.Errorf("unresolved captured var should be left as-is, got %s", got)
	}
}

func TestSubstituteAllVars_NoConflict(t *testing.T) {
	os.Setenv("MASLOW_TEST_ENV", "prod")
	defer os.Unsetenv("MASLOW_TEST_ENV")

	vars := map[string]string{"user": "alice"}
	got := substituteAllVars("${MASLOW_TEST_ENV}/${{user}}", vars)
	if got != "prod/alice" {
		t.Errorf("expected 'prod/alice', got %s", got)
	}
}
