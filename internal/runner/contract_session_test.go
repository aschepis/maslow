package runner

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

// sessionServer signs a client in at /login and serves /me only with that cookie.
func sessionServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
		http.Redirect(w, r, "/me", http.StatusSeeOther)
	})
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err != nil || c.Value != "abc" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte("signed in"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestContractSession_CookiesPersistWithinScenario(t *testing.T) {
	srv := sessionServer(t)
	ok := 200
	results := RunContracts([]spec.Contract{{
		Name: "c",
		Scenarios: []spec.Scenario{{
			Name: "sign-in",
			Setup: []spec.Step{
				{Action: "http", Method: "POST", URL: srv.URL + "/login", Expect: &spec.Expectation{Status: &ok, BodyContains: "signed in"}},
			},
			Steps: []spec.Step{
				{Action: "http", Method: "GET", URL: srv.URL + "/me", Expect: &spec.Expectation{Status: &ok}},
				{Action: "poll", URL: srv.URL + "/me", Interval: "1s", Timeout: "2s", Expect: &spec.Expectation{Status: &ok}},
			},
		}},
	}})

	if results[0].Status != "pass" {
		t.Errorf("expected the session cookie to carry across steps, got %s: %s", results[0].Status, results[0].Error)
	}
}

func TestContractBaseURL_ResolvesRelativeURLs(t *testing.T) {
	srv := sessionServer(t)
	ok := 200
	contract := spec.Contract{
		Name:    "c",
		Env:     []string{"MASLOW_TEST_BASE"},
		BaseURL: "{{.env.MASLOW_TEST_BASE}}/",
		Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{
			{Action: "http", Method: "POST", URL: "/login", Expect: &spec.Expectation{Status: &ok}},
		}}},
	}
	t.Setenv("MASLOW_TEST_BASE", srv.URL)

	if got := RunContracts([]spec.Contract{contract})[0]; got.Status != "pass" {
		t.Errorf("expected base_url from the contract, got %s: %s", got.Status, got.Error)
	}

	contract.BaseURL = "http://127.0.0.1:1"
	if got := RunContractsWith([]spec.Contract{contract}, ContractOptions{BaseURL: srv.URL})[0]; got.Status != "pass" {
		t.Errorf("expected ContractOptions.BaseURL to replace the contract's, got %s: %s", got.Status, got.Error)
	}
}

func TestContractBaseURL_RelativeURLWithoutBaseFails(t *testing.T) {
	got := RunContracts([]spec.Contract{{Name: "c", Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{
		{Action: "http", Method: "GET", URL: "/healthz"},
	}}}}})[0]
	if got.Status != "fail" || !strings.Contains(got.Error, "base_url") {
		t.Errorf("expected a failure naming base_url, got %s: %s", got.Status, got.Error)
	}
}

func TestContractSession_ScenariosDoNotShareCookies(t *testing.T) {
	srv := sessionServer(t)
	unauthorized := 401
	results := RunContracts([]spec.Contract{{
		Name: "c",
		Scenarios: []spec.Scenario{
			{Name: "sign-in", Steps: []spec.Step{{Action: "http", Method: "POST", URL: srv.URL + "/login"}}},
			{Name: "anonymous", Steps: []spec.Step{
				{Action: "http", Method: "GET", URL: srv.URL + "/me", Expect: &spec.Expectation{Status: &unauthorized}},
			}},
		},
	}})

	if results[1].Status != "pass" {
		t.Errorf("expected a fresh cookie jar per scenario, got %s: %s", results[1].Status, results[1].Error)
	}
}
