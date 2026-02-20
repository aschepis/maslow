package runner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aschepis/maslow-agentic/internal/spec"
)

func TestCheckExpectation_HTTPStatus(t *testing.T) {
	status200 := 200
	state := &stepState{httpStatus: 200}

	err := checkExpectation(&spec.Expectation{Status: &status200}, state)
	if err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}

	status404 := 404
	err = checkExpectation(&spec.Expectation{Status: &status404}, state)
	if err == nil {
		t.Error("expected error for status mismatch")
	}
}

func TestCheckExpectation_BodyMatches(t *testing.T) {
	state := &stepState{output: `{"id": 42, "name": "test-item"}`}

	// Matching regex.
	err := checkExpectation(&spec.Expectation{BodyMatches: `"id":\s*\d+`}, state)
	if err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}

	// Non-matching regex.
	err = checkExpectation(&spec.Expectation{BodyMatches: `"missing":\s*true`}, state)
	if err == nil {
		t.Error("expected error for non-matching regex")
	}

	// Invalid regex.
	err = checkExpectation(&spec.Expectation{BodyMatches: `[invalid`}, state)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestCheckExpectation_Headers(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("X-Request-Id", "abc123")

	state := &stepState{httpHeaders: headers}

	// Matching header.
	err := checkExpectation(&spec.Expectation{
		Headers: map[string]string{"Content-Type": "application/json"},
	}, state)
	if err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}

	// Missing header.
	err = checkExpectation(&spec.Expectation{
		Headers: map[string]string{"X-Missing": "value"},
	}, state)
	if err == nil {
		t.Error("expected error for missing header")
	}

	// Wrong header value.
	err = checkExpectation(&spec.Expectation{
		Headers: map[string]string{"Content-Type": "text/html"},
	}, state)
	if err == nil {
		t.Error("expected error for wrong header value")
	}

	// No HTTP headers in state.
	err = checkExpectation(&spec.Expectation{
		Headers: map[string]string{"Content-Type": "application/json"},
	}, &stepState{})
	if err == nil {
		t.Error("expected error when no HTTP response headers available")
	}
}

func TestCheckExpectation_JSONPath(t *testing.T) {
	jsonBody := `{"id": 1, "title": "Buy milk", "nested": {"key": "val"}, "items": [{"n": "a"}, {"n": "b"}]}`
	state := &stepState{output: jsonBody}

	tests := []struct {
		name     string
		path     string
		value    any
		wantErr  bool
	}{
		{"root field int", "$.id", 1, false},
		{"root field string", "$.title", "Buy milk", false},
		{"nested field", "$.nested.key", "val", false},
		{"array element", "$.items[0].n", "a", false},
		{"array element second", "$.items[1].n", "b", false},
		{"wrong value", "$.id", 999, true},
		{"missing field", "$.missing", nil, true},
		{"out of bounds", "$.items[5].n", nil, true},
		{"no value check", "$.id", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect := &spec.Expectation{JSONPath: tt.path}
			if tt.value != nil {
				expect.Value = tt.value
			}
			err := checkExpectation(expect, state)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestCheckExpectation_JSONPathRootArray(t *testing.T) {
	jsonBody := `[{"id": 1}, {"id": 2}]`
	state := &stepState{output: jsonBody}

	err := checkExpectation(&spec.Expectation{JSONPath: "$[0].id", Value: 1}, state)
	if err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}

	err = checkExpectation(&spec.Expectation{JSONPath: "$[1].id", Value: 2}, state)
	if err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}
}

func TestCheckExpectation_MultipleAssertions(t *testing.T) {
	status200 := 200
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	state := &stepState{
		httpStatus:  200,
		httpHeaders: headers,
		output:      `{"status": "ok", "count": 5}`,
	}

	err := checkExpectation(&spec.Expectation{
		Status:       &status200,
		BodyContains: "ok",
		JSONPath:     "$.count",
		Value:        5,
		Headers:      map[string]string{"Content-Type": "application/json"},
	}, state)
	if err != nil {
		t.Errorf("expected all assertions to pass, got: %v", err)
	}
}

func TestRunContracts_HTTPInlineExpect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write([]byte(`{"id": 1, "title": "test"}`))
	}))
	defer srv.Close()

	status201 := 201
	contracts := []spec.Contract{
		{
			Name: "inline-expect",
			Scenarios: []spec.Scenario{
				{
					Name: "http-with-inline",
					Steps: []spec.Step{
						{
							Action:  "http",
							Method:  "POST",
							URL:     srv.URL,
							Headers: map[string]string{"Content-Type": "application/json"},
							Body:    `{"title":"test"}`,
							Expect: &spec.Expectation{
								Status:       &status201,
								BodyContains: "test",
								JSONPath:     "$.id",
								Value:        1,
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

func TestRunContracts_HTTPInlineExpectFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer srv.Close()

	status201 := 201
	contracts := []spec.Contract{
		{
			Name: "inline-fail",
			Scenarios: []spec.Scenario{
				{
					Name: "wrong-status",
					Steps: []spec.Step{
						{
							Action: "http",
							Method: "GET",
							URL:    srv.URL,
							Expect: &spec.Expectation{Status: &status201},
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
}

func TestRunContracts_RegexBodyMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"items": [{"id": 1}, {"id": 2}, {"id": 3}]}`))
	}))
	defer srv.Close()

	status200 := 200
	contracts := []spec.Contract{
		{
			Name: "regex-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "matches-pattern",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL},
						{Action: "assert", Expect: &spec.Expectation{
							Status:      &status200,
							BodyMatches: `"id":\s*[0-9]+`,
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

func TestRunContracts_HeaderCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom", "value123")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	contracts := []spec.Contract{
		{
			Name: "header-contract",
			Scenarios: []spec.Scenario{
				{
					Name: "check-headers",
					Steps: []spec.Step{
						{Action: "http", Method: "GET", URL: srv.URL},
						{Action: "assert", Expect: &spec.Expectation{
							Headers: map[string]string{
								"Content-Type": "application/json",
								"X-Custom":     "value123",
							},
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

func TestEvaluateJSONPath(t *testing.T) {
	jsonStr := `{"a": {"b": [1, 2, 3]}, "c": "hello", "d": true}`

	tests := []struct {
		name    string
		path    string
		want    string // JSON representation of expected value
		wantErr bool
	}{
		{"root", "$", "", false},
		{"simple field", "$.c", `"hello"`, false},
		{"boolean field", "$.d", "true", false},
		{"nested object", "$.a", "", false},
		{"nested array element", "$.a.b[0]", "1", false},
		{"nested array last", "$.a.b[2]", "3", false},
		{"missing field", "$.z", "", true},
		{"out of bounds", "$.a.b[10]", "", true},
		{"non-array index", "$.c[0]", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateJSONPath(jsonStr, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (result: %v)", result)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.want != "" {
				got, _ := json.Marshal(result)
				if string(got) != tt.want {
					t.Errorf("got %s, want %s", string(got), tt.want)
				}
			}
		})
	}
}

func TestJsonValuesEqual(t *testing.T) {
	tests := []struct {
		name string
		got  any
		exp  any
		want bool
	}{
		{"float64 vs int", float64(42), 42, true},
		{"float64 vs float64", float64(3.14), float64(3.14), true},
		{"string match", "hello", "hello", true},
		{"string mismatch", "hello", "world", false},
		{"bool match", true, true, true},
		{"bool mismatch", true, false, false},
		{"float64 vs wrong int", float64(42), 43, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jsonValuesEqual(tt.got, tt.exp); got != tt.want {
				t.Errorf("jsonValuesEqual(%v, %v) = %v, want %v", tt.got, tt.exp, got, tt.want)
			}
		})
	}
}
