package audit

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

func TestRun_NoTargets(t *testing.T) {
	mas := &spec.MAS{}
	report, err := Run(mas, Options{Profile: "full"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verdict != "inconclusive" {
		t.Errorf("expected inconclusive, got %s", report.Verdict)
	}
}

func TestRun_BinaryExists(t *testing.T) {
	// Create a temp "binary".
	dir := t.TempDir()
	binPath := filepath.Join(dir, "test-bin")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho test v1"), 0o755); err != nil {
		t.Fatal(err)
	}

	mas := &spec.MAS{
		Audit: &spec.Audit{
			Targets: []spec.AuditTarget{
				{Kind: "binary", Path: binPath},
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
}

func TestRun_BinaryMissing(t *testing.T) {
	mas := &spec.MAS{
		Audit: &spec.Audit{
			Targets: []spec.AuditTarget{
				{Kind: "binary", Path: "/nonexistent/binary"},
			},
		},
	}

	report, err := Run(mas, Options{Profile: "full"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Verdict != "fail" {
		t.Errorf("expected fail, got %s", report.Verdict)
	}
}

func TestRun_ContainerSkipped(t *testing.T) {
	mas := &spec.MAS{
		Audit: &spec.Audit{
			Targets: []spec.AuditTarget{
				{Kind: "container", Path: "myimage:latest"},
			},
		},
	}

	report, err := Run(mas, Options{Profile: "full"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Container audit is a stub that returns skip.
	found := false
	for _, r := range report.CheckResults {
		if r.Status == "skip" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one skip result for container audit")
	}
}

func healthServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func endpointSpec(path string, checks []string) *spec.MAS {
	ok := 200
	health := []spec.Step{{Action: "http", Method: "GET", URL: "/healthz", Expect: &spec.Expectation{Status: &ok}}}
	missing := []spec.Step{{Action: "http", Method: "GET", URL: "/missing", Expect: &spec.Expectation{Status: &ok}}}
	return &spec.MAS{
		Contracts: []spec.Contract{
			{Name: "smoke", Tags: []string{"deploy"}, BaseURL: "http://127.0.0.1:1", Scenarios: []spec.Scenario{{Name: "health", Steps: health}}},
			{Name: "broken", Scenarios: []spec.Scenario{{Name: "missing", Steps: missing}}},
		},
		Audit: &spec.Audit{Targets: []spec.AuditTarget{{Kind: "endpoint", Path: path, Checks: checks}}},
	}
}

func TestRun_EndpointRunsSelectedContractsAgainstBaseURL(t *testing.T) {
	srv := healthServer(t)
	report, err := Run(endpointSpec(srv.URL, []string{"deploy"}), Options{Profile: "full"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "pass" {
		t.Fatalf("expected pass, got %s: %+v %+v", report.Verdict, report.CheckResults, report.Contracts)
	}
	if len(report.Contracts) != 1 || report.Contracts[0].Name != srv.URL+"/smoke/health" {
		t.Errorf("expected only the deploy-tagged contract, named by target, got %+v", report.Contracts)
	}
}

func TestRun_EndpointFailingContractFailsAudit(t *testing.T) {
	srv := healthServer(t)
	report, err := Run(endpointSpec(srv.URL, nil), Options{Profile: "full"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "fail" || len(report.Contracts) != 2 {
		t.Errorf("expected both contracts run and a fail, got %s with %d results", report.Verdict, len(report.Contracts))
	}
}

// A metric step queries its own URL, never the target, so a post-deploy audit
// can judge the deployed service by what its metrics report.
func TestRun_EndpointRunsMetricStepsAgainstTheirOwnURL(t *testing.T) {
	app := healthServer(t)
	var promPaths []string
	prom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promPaths = append(promPaths, r.URL.Path)
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[0,"0.001"]}]}}`))
	}))
	t.Cleanup(prom.Close)

	maxRatio := 0.01
	mas := endpointSpec(app.URL, []string{"post-deploy"})
	mas.Contracts = append(mas.Contracts, spec.Contract{Name: "error-rate", Tags: []string{"post-deploy"}, Scenarios: []spec.Scenario{{
		Name:  "holds",
		Steps: []spec.Step{{Action: "metric", Provider: "prometheus", URL: prom.URL, Query: "error_ratio", Expect: &spec.Expectation{Max: &maxRatio}}},
	}}})

	report, err := Run(mas, Options{Profile: "full"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "pass" || len(report.Contracts) != 1 {
		t.Fatalf("expected the metric contract to pass, got %s: %+v %+v", report.Verdict, report.CheckResults, report.Contracts)
	}
	if len(promPaths) != 1 || promPaths[0] != "/api/v1/query" {
		t.Errorf("expected one query to Prometheus, got %v", promPaths)
	}
	if obs := report.Contracts[0].Observations; len(obs) != 1 || obs[0].Query != "error_ratio" {
		t.Errorf("expected the observation on the audit's contract result, got %+v", obs)
	}
}

func TestRun_EndpointPathFromEnvIsNotRecorded(t *testing.T) {
	srv := healthServer(t)
	t.Setenv("MASLOW_TEST_DEPLOY_URL", srv.URL)
	report, err := Run(endpointSpec("{{.env.MASLOW_TEST_DEPLOY_URL}}", []string{"smoke"}), Options{Profile: "full"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "pass" {
		t.Fatalf("expected pass, got %s: %+v", report.Verdict, report.CheckResults)
	}
	if got := report.Contracts[0].Name; strings.Contains(got, srv.URL) {
		t.Errorf("expected the result named by the path as written, got %q", got)
	}
}

func TestRun_EndpointErrors(t *testing.T) {
	cases := map[string]*spec.MAS{
		"not a URL":        endpointSpec("localhost:8080", nil),
		"unknown selector": endpointSpec("http://localhost:1", []string{"nope"}),
		"no contracts":     {Audit: &spec.Audit{Targets: []spec.AuditTarget{{Kind: "endpoint", Path: "http://localhost:1"}}}},
		"unset variable":   endpointSpec("{{.env.MASLOW_TEST_UNSET_12345}}", nil),
	}
	for name, mas := range cases {
		report, err := Run(mas, Options{Profile: "full"})
		if err != nil {
			t.Fatal(err)
		}
		if report.Verdict != "fail" || len(report.CheckResults) != 1 || report.CheckResults[0].Status != "error" {
			t.Errorf("%s: expected one error result and a fail, got %s %+v", name, report.Verdict, report.CheckResults)
		}
	}
}

func TestParseDuration(t *testing.T) {
	d, err := parseDuration("30s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Seconds() != 30 {
		t.Errorf("expected 30s, got %v", d)
	}
}
