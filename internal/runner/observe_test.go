package runner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// The half second shows that evaluation times are truncated to the second.
var observeStart = time.Date(2026, 11, 19, 12, 0, 0, 500_000_000, time.UTC)

const observeT0 = "2026-11-19T12:00:00Z"

// fakeClock advances only when slept, so a ten-minute hold runs instantly.
type fakeClock struct{ t time.Time }

func (c *fakeClock) clock() clock {
	return clock{now: func() time.Time { return c.t }, sleep: func(d time.Duration) { c.t = c.t.Add(d) }}
}

// fakeObservability answers the nth request (from 0) with reply and records every request.
type fakeObservability struct {
	srv      *httptest.Server
	mu       sync.Mutex
	requests []*http.Request
}

func newFake(t *testing.T, reply func(n int, r *http.Request) (int, string)) *fakeObservability {
	t.Helper()
	f := &fakeObservability{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		n := len(f.requests)
		f.requests = append(f.requests, r)
		f.mu.Unlock()
		status, body := reply(n, r)
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// always answers every request with the same body.
func always(body string) func(int, *http.Request) (int, string) {
	return func(int, *http.Request) (int, string) { return http.StatusOK, body }
}

func (f *fakeObservability) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeObservability) request(i int) *http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[i]
}

func vector(series ...string) string {
	return `{"status":"success","data":{"resultType":"vector","result":[` + strings.Join(series, ",") + `]}}`
}

func promSeries(labels, value string) string {
	return `{"metric":` + labels + `,"value":[1700000000,"` + value + `"]}`
}

func float(v float64) *float64 { return &v }

func metricStep(url, query string, expect spec.Expectation) spec.Step {
	return spec.Step{Action: "metric", Provider: "prometheus", URL: url, Query: query, Expect: &expect}
}

func held(step spec.Step, hold, interval string) spec.Step {
	step.For, step.Interval = hold, interval
	return step
}

func runObserve(t *testing.T, contract spec.Contract) evidence.ContractResult {
	t.Helper()
	clk := &fakeClock{t: observeStart}
	return RunContractsWith([]spec.Contract{contract}, ContractOptions{clock: clk.clock()})[0]
}

func scenarioOf(steps ...spec.Step) spec.Contract {
	return spec.Contract{Name: "c", Scenarios: []spec.Scenario{{Name: "s", Steps: steps}}}
}

func onlyObservation(t *testing.T, r evidence.ContractResult) evidence.Observation {
	t.Helper()
	if len(r.Observations) != 1 {
		t.Fatalf("expected one observation, got %+v", r)
	}
	return r.Observations[0]
}

func TestMetric_InstantUnderMaxPasses(t *testing.T) {
	fake := newFake(t, always(vector(promSeries(`{"route":"/checkout"}`, "0.004"))))
	r := runObserve(t, scenarioOf(metricStep(fake.srv.URL, "error_ratio", spec.Expectation{Max: float(0.01)})))

	if r.Status != "pass" {
		t.Fatalf("expected pass, got %+v", r)
	}
	obs := onlyObservation(t, r)
	if obs.Phase != "steps" || obs.Index != 0 || obs.Action != "metric" || obs.Provider != "prometheus" || obs.Query != "error_ratio" {
		t.Errorf("unexpected observation header %+v", obs)
	}
	if len(obs.Samples) != 1 || obs.Samples[0].At != observeT0 || !obs.Samples[0].OK || obs.Samples[0].Max != "0.004" || *obs.Samples[0].Series != 1 {
		t.Errorf("unexpected samples %+v", obs.Samples)
	}
	req := fake.request(0)
	if req.URL.Path != "/api/v1/query" || req.URL.Query().Get("query") != "error_ratio" || req.URL.Query().Get("time") != observeT0 {
		t.Errorf("unexpected request %s", req.URL)
	}
}

func TestMetric_OverMaxFailsNamingTheSeries(t *testing.T) {
	fake := newFake(t, always(vector(promSeries(`{"code":"500"}`, "0.02"))))
	r := runObserve(t, scenarioOf(metricStep(fake.srv.URL, "error_ratio", spec.Expectation{Max: float(0.01)})))

	if r.Status != "fail" || r.FailedStep == nil || r.FailedStep.Action != "metric" || r.FailedStep.Target != "metric error_ratio" {
		t.Fatalf("expected a failed metric step, got %+v", r)
	}
	obs := onlyObservation(t, r)
	if !strings.Contains(obs.Breach, `{code="500"}`) || !strings.Contains(obs.Breach, "0.02") || !strings.Contains(r.Error, obs.Breach) {
		t.Errorf("expected the breach to name the series and value, got %q (error %q)", obs.Breach, r.Error)
	}
}

func TestMetric_BreachNamesTheFirstOffendingSeriesInLabelOrder(t *testing.T) {
	fake := newFake(t, always(vector(
		promSeries(`{"pod":"c"}`, "0.05"),
		promSeries(`{"pod":"a"}`, "0.001"),
		promSeries(`{"pod":"b"}`, "0.03"),
	)))
	r := runObserve(t, scenarioOf(metricStep(fake.srv.URL, "q", spec.Expectation{Max: float(0.01)})))

	obs := onlyObservation(t, r)
	if !strings.Contains(obs.Breach, `{pod="b"}`) {
		t.Errorf("expected the breach to name pod b, got %q", obs.Breach)
	}
	if s := obs.Samples[0]; s.Min != "0.001" || s.Max != "0.05" || *s.Series != 3 {
		t.Errorf("expected min and max across series, got %+v", s)
	}
}

func TestMetric_HeldUnderMaxPollsUntilTheEnd(t *testing.T) {
	fake := newFake(t, always(vector(promSeries(`{}`, "0.001"))))
	r := runObserve(t, scenarioOf(held(metricStep(fake.srv.URL, "q", spec.Expectation{Max: float(0.01)}), "10m", "30s")))

	if r.Status != "pass" {
		t.Fatalf("expected pass, got %+v", r)
	}
	obs := onlyObservation(t, r)
	if len(obs.Samples) != 21 || fake.count() != 21 {
		t.Fatalf("expected 21 samples and requests, got %d and %d", len(obs.Samples), fake.count())
	}
	if obs.For != "10m" || obs.Interval != "30s" {
		t.Errorf("expected the hold recorded, got %+v", obs)
	}
	if last := obs.Samples[20].At; last != "2026-11-19T12:10:00Z" {
		t.Errorf("expected the last sample at t0+10m, got %s", last)
	}
	for i, s := range obs.Samples {
		if got := fake.request(i).URL.Query().Get("time"); got != s.At {
			t.Errorf("sample %d: request time %s, recorded %s", i, got, s.At)
		}
	}
}

func TestMetric_HeldStopsAtTheFirstBreach(t *testing.T) {
	fake := newFake(t, func(n int, _ *http.Request) (int, string) {
		if n == 4 {
			return http.StatusOK, vector(promSeries(`{}`, "0.5"))
		}
		return http.StatusOK, vector(promSeries(`{}`, "0.001"))
	})
	r := runObserve(t, scenarioOf(held(metricStep(fake.srv.URL, "q", spec.Expectation{Max: float(0.01)}), "10m", "30s")))

	obs := onlyObservation(t, r)
	if r.Status != "fail" || len(obs.Samples) != 5 || fake.count() != 5 {
		t.Fatalf("expected a fail after 5 samples, got %s with %d samples and %d requests", r.Status, len(obs.Samples), fake.count())
	}
	if obs.Samples[4].OK || obs.Samples[4].At != "2026-11-19T12:02:00Z" {
		t.Errorf("expected the fifth sample to breach at 12:02, got %+v", obs.Samples[4])
	}
}

func TestMetric_NoData(t *testing.T) {
	cases := map[string]struct {
		body   string
		expect spec.Expectation
		status string
	}{
		"empty vector fails":             {vector(), spec.Expectation{Max: float(1)}, "fail"},
		"empty vector may pass":          {vector(), spec.Expectation{Max: float(1), IfNoData: "pass"}, "pass"},
		"NaN fails":                      {vector(promSeries(`{}`, "NaN")), spec.Expectation{Max: float(1)}, "fail"},
		"NaN beside a value is not data": {vector(promSeries(`{"a":"1"}`, "NaN"), promSeries(`{"a":"2"}`, "0.5")), spec.Expectation{Max: float(1)}, "pass"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fake := newFake(t, always(tc.body))
			r := runObserve(t, scenarioOf(metricStep(fake.srv.URL, "q", tc.expect)))
			if r.Status != tc.status {
				t.Fatalf("expected %s, got %+v", tc.status, r)
			}
			obs := onlyObservation(t, r)
			if tc.status == "fail" && (!obs.Samples[0].NoData || !strings.Contains(obs.Breach, "no data")) {
				t.Errorf("expected a no-data breach, got %+v", obs)
			}
		})
	}
}

func TestMetric_SeriesCountBounds(t *testing.T) {
	fake := newFake(t, always(vector(promSeries(`{"a":"1"}`, "1"), promSeries(`{"a":"2"}`, "1"), promSeries(`{"a":"3"}`, "1"))))
	two := 2
	r := runObserve(t, scenarioOf(metricStep(fake.srv.URL, "q", spec.Expectation{MaxSeries: &two})))
	if r.Status != "fail" || !strings.Contains(r.Observations[0].Breach, "3 series") {
		t.Errorf("expected a series-count breach, got %+v", r)
	}
}

func TestMetric_ScalarIsOneSeries(t *testing.T) {
	fake := newFake(t, always(`{"status":"success","data":{"resultType":"scalar","result":[1700000000,"5"]}}`))
	r := runObserve(t, scenarioOf(metricStep(fake.srv.URL, "scalar(up)", spec.Expectation{Min: float(1)})))
	if r.Status != "pass" || *r.Observations[0].Samples[0].Series != 1 {
		t.Errorf("expected a pass on one series, got %+v", r)
	}
}

func TestMetric_InfinityIsComparedAsANumber(t *testing.T) {
	fake := newFake(t, always(vector(promSeries(`{}`, "+Inf"))))
	r := runObserve(t, scenarioOf(metricStep(fake.srv.URL, "q", spec.Expectation{Max: float(1)})))
	if r.Status != "fail" || r.Observations[0].Samples[0].Max != "+Inf" {
		t.Fatalf("expected +Inf to breach max 1, got %+v", r)
	}
	if _, err := json.Marshal(r); err != nil {
		t.Errorf("expected the result to marshal, got %v", err)
	}
}

func TestMetric_MatrixResultIsAnError(t *testing.T) {
	fake := newFake(t, always(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
	r := runObserve(t, scenarioOf(metricStep(fake.srv.URL, "q[5m]", spec.Expectation{Max: float(1)})))
	if r.Status != "error" || !strings.Contains(r.Error, "matrix") {
		t.Errorf("expected an error naming the result type, got %+v", r)
	}
}

func TestMetric_FailedQueryMidHoldFailsAtOnce(t *testing.T) {
	fake := newFake(t, func(n int, _ *http.Request) (int, string) {
		if n == 2 {
			return http.StatusInternalServerError, "prometheus is overloaded"
		}
		return http.StatusOK, vector(promSeries(`{}`, "0"))
	})
	r := runObserve(t, scenarioOf(held(metricStep(fake.srv.URL, "q", spec.Expectation{Max: float(1)}), "10m", "30s")))

	if r.Status != "fail" || fake.count() != 3 || !strings.Contains(r.Output, "prometheus is overloaded") {
		t.Fatalf("expected a fail after 3 requests with the body as output, got %+v (%d requests)", r, fake.count())
	}
	if s := r.Observations[0].Samples[2]; s.OK || !strings.Contains(s.Error, "500") {
		t.Errorf("expected the failed query recorded on the sample, got %+v", s)
	}
}

func TestMetric_URLIsNeverJoinedToBaseURL(t *testing.T) {
	fake := newFake(t, always(vector(promSeries(`{}`, "0"))))
	contract := scenarioOf(metricStep("/api", "q", spec.Expectation{Max: float(1)}))
	contract.BaseURL = fake.srv.URL
	r := runObserve(t, contract)
	if r.Status != "error" || !strings.Contains(r.Error, "base_url") || fake.count() != 0 {
		t.Errorf("expected an error naming base_url and no request, got %+v", r)
	}
}

func TestMetric_BreachIsNotRetried(t *testing.T) {
	fake := newFake(t, always(vector(promSeries(`{}`, "2"))))
	contract := scenarioOf(metricStep(fake.srv.URL, "q", spec.Expectation{Max: float(1)}))
	contract.Retries = 2
	r := runObserve(t, contract)
	if r.Status != "fail" || r.Attempts != nil || fake.count() != 1 {
		t.Errorf("expected one attempt that fails, got %+v (%d requests)", r, fake.count())
	}
}

func TestMetric_FailedQueryIsRetried(t *testing.T) {
	fake := newFake(t, func(n int, _ *http.Request) (int, string) {
		if n == 0 {
			return http.StatusBadGateway, "bad gateway"
		}
		return http.StatusOK, vector(promSeries(`{}`, "0"))
	})
	contract := scenarioOf(metricStep(fake.srv.URL, "q", spec.Expectation{Max: float(1)}))
	contract.Retries = 1
	r := runObserve(t, contract)
	if r.Status != evidence.StatusFlaky || len(r.Attempts) != 2 {
		t.Fatalf("expected flaky with two attempts, got %+v", r)
	}
	if len(r.Attempts[0].Observations) != 1 || len(r.Attempts[1].Observations) != 1 {
		t.Errorf("expected each attempt to keep its observation, got %+v", r.Attempts)
	}
}

func TestMetric_CredentialsReachTheServerButNotTheEvidence(t *testing.T) {
	t.Setenv("MASLOW_TEST_PROM_TOKEN", "s3cret-token")
	t.Setenv("MASLOW_TEST_TENANT", "tenant-42")
	fake := newFake(t, always(vector(promSeries(`{"tenant":"tenant-42"}`, "2"))))
	step := metricStep(fake.srv.URL, `up{tenant="{{.env.MASLOW_TEST_TENANT}}"}`, spec.Expectation{Max: float(1)})
	step.Headers = map[string]string{"Authorization": "Bearer {{.env.MASLOW_TEST_PROM_TOKEN}}"}
	contract := scenarioOf(step)
	contract.Env = []string{"MASLOW_TEST_PROM_TOKEN", "MASLOW_TEST_TENANT"}
	r := runObserve(t, contract)

	if got := fake.request(0).Header.Get("Authorization"); got != "Bearer s3cret-token" {
		t.Errorf("expected the token sent, got %q", got)
	}
	data, _ := json.Marshal(r)
	if strings.Contains(string(data), "s3cret-token") || strings.Contains(string(data), "tenant-42") {
		t.Errorf("expected credentials redacted, got %s", data)
	}
	if !strings.Contains(r.Observations[0].Query, "[redacted:MASLOW_TEST_TENANT]") {
		t.Errorf("expected the query redacted, got %q", r.Observations[0].Query)
	}
}

func TestMetric_SameResponsesGiveTheSameObservations(t *testing.T) {
	marshal := func() string {
		fake := newFake(t, always(vector(promSeries(`{"b":"2"}`, "0.5"), promSeries(`{"a":"1"}`, "0.2"))))
		r := runObserve(t, scenarioOf(held(metricStep(fake.srv.URL, "q", spec.Expectation{Max: float(0.4)}), "1m", "30s")))
		data, err := json.Marshal(r.Observations)
		if err != nil {
			t.Fatal(err)
		}
		return strings.ReplaceAll(string(data), fake.srv.URL, "URL")
	}
	if a, b := marshal(), marshal(); a != b {
		t.Errorf("expected identical observations:\n%s\n%s", a, b)
	}
}

func TestLog_QueriesLokiMetricAPI(t *testing.T) {
	fake := newFake(t, always(vector(promSeries(`{"app":"checkout"}`, "2"))))
	step := spec.Step{Action: "log", Provider: "loki", URL: fake.srv.URL, Query: `sum(count_over_time({app="checkout"} |= "panic" [5m]))`, Expect: &spec.Expectation{Max: float(0)}}
	r := runObserve(t, scenarioOf(step))

	if r.Status != "fail" || r.Observations[0].Samples[0].Max != "2" {
		t.Errorf("expected two panics to breach max 0, got %+v", r)
	}
	if req := fake.request(0); req.URL.Path != "/loki/api/v1/query" || req.URL.Query().Get("time") != observeT0 {
		t.Errorf("unexpected request %s", req.URL)
	}
}

func TestLog_StreamQueryIsAnError(t *testing.T) {
	fake := newFake(t, always(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	step := spec.Step{Action: "log", Provider: "loki", URL: fake.srv.URL, Query: `{app="checkout"} |= "panic"`, Expect: &spec.Expectation{Max: float(0)}}
	r := runObserve(t, scenarioOf(step))
	if r.Status != "error" || !strings.Contains(r.Error, "metric LogQL") {
		t.Errorf("expected an error asking for metric LogQL, got %+v", r)
	}
}

const alertsBody = `[
  {"labels":{"alertname":"HighErrorRate","service":"checkout"},"startsAt":"2026-11-19T12:00:30Z","status":{"state":"active","silencedBy":[],"inhibitedBy":[]}},
  {"labels":{"alertname":"OldAlert","service":"checkout"},"startsAt":"2026-11-19T11:00:00Z","status":{"state":"active","silencedBy":[],"inhibitedBy":[]}},
  {"labels":{"alertname":"Silenced","service":"checkout"},"startsAt":"2026-11-19T12:00:30Z","status":{"state":"suppressed","silencedBy":["s1"],"inhibitedBy":[]}}
]`

func alertsStep(url string, max float64) spec.Step {
	return spec.Step{Action: "alerts", Provider: "alertmanager", URL: url, Matchers: []string{`service="checkout"`}, Expect: &spec.Expectation{Max: &max}}
}

func TestAlerts_CountsActiveUnsilencedAlerts(t *testing.T) {
	fake := newFake(t, always(alertsBody))
	r := runObserve(t, scenarioOf(alertsStep(fake.srv.URL, 1)))

	if r.Status != "fail" || *r.Observations[0].Samples[0].Alerts != 2 {
		t.Fatalf("expected two active alerts to breach max 1, got %+v", r)
	}
	if !strings.Contains(r.Observations[0].Breach, "HighErrorRate") {
		t.Errorf("expected the breach to name an alert, got %q", r.Observations[0].Breach)
	}
	q := fake.request(0).URL.Query()
	if fake.request(0).URL.Path != "/api/v2/alerts" || q.Get("filter") != `service="checkout"` || q.Get("silenced") != "false" || q.Get("inhibited") != "false" || q.Get("active") != "true" {
		t.Errorf("unexpected request %s", fake.request(0).URL)
	}
}

func TestAlerts_IgnorePreexistingDropsAlertsStartedBeforeTheStep(t *testing.T) {
	fake := newFake(t, always(alertsBody))
	step := alertsStep(fake.srv.URL, 1)
	step.IgnorePreexisting = true
	r := runObserve(t, scenarioOf(step))
	if r.Status != "pass" || *r.Observations[0].Samples[0].Alerts != 1 {
		t.Errorf("expected only the new alert counted, got %+v", r)
	}
}

func TestAlerts_SettleExtendsTheHold(t *testing.T) {
	fake := newFake(t, always(`[]`))
	step := alertsStep(fake.srv.URL, 0)
	step.For, step.Settle, step.Interval = "2m", "1m", "1m"
	r := runObserve(t, scenarioOf(step))
	obs := onlyObservation(t, r)
	if r.Status != "pass" || len(obs.Samples) != 4 || obs.Samples[3].At != "2026-11-19T12:03:00Z" || obs.Settle != "1m" {
		t.Errorf("expected 4 samples up to t0+3m, got %+v", obs)
	}
}

func TestDatadogIsNotImplemented(t *testing.T) {
	for _, action := range []string{"metric", "log", "alerts"} {
		t.Run(action, func(t *testing.T) {
			fake := newFake(t, always(vector()))
			step := spec.Step{Action: action, Provider: "datadog", URL: fake.srv.URL, Query: "q", Expect: &spec.Expectation{Max: float(0)}}
			r := runObserve(t, scenarioOf(step))
			if r.Status != "error" || !strings.Contains(r.Error, "provider datadog is not implemented") || fake.count() != 0 {
				t.Errorf("expected a not-implemented error and no request, got %+v", r)
			}
		})
	}
}
