package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// query is what a metric, log or alerts step asks a provider at one evaluation time.
type query struct {
	url      string
	headers  map[string]string
	expr     string
	matchers []string
	at       time.Time
}

// reading is what a provider saw: series for metric and log steps, alerts for alerts steps.
type reading struct {
	series []series
	alerts []alert
}

type series struct {
	labels string
	value  float64
}

type alert struct {
	labels   string
	startsAt time.Time
}

// provider turns a query into a reading for one vendor. Maslow judges the
// reading, so thresholds, holds and no-data mean the same for every vendor.
type provider interface {
	read(q query) (reading, error)
}

// providers is keyed by step action, then by the step's provider field.
var providers = map[string]map[string]provider{
	"metric": {"prometheus": instantQuery{path: "/api/v1/query"}, "datadog": notImplemented("datadog")},
	"log":    {"loki": instantQuery{path: "/loki/api/v1/query"}, "datadog": notImplemented("datadog")},
	"alerts": {"alertmanager": alertmanager{}, "datadog": notImplemented("datadog")},
}

type notImplemented string

func (p notImplemented) read(query) (reading, error) {
	return reading{}, &configError{fmt.Errorf("provider %s is not implemented", string(p))}
}

// badResponse is a reply that cannot be judged; its body becomes the step's output.
type badResponse struct {
	reason string
	body   string
}

func (e *badResponse) Error() string { return e.reason }

// maxResponseBytes keeps a runaway query from exhausting memory.
const maxResponseBytes = 10 << 20

func get(q query, path string, params url.Values) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	target := strings.TrimSuffix(q.url, "/") + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, &configError{fmt.Errorf("failed to create request: %w", err)}
	}
	for k, v := range q.headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &badResponse{reason: fmt.Sprintf("HTTP %d from %s", resp.StatusCode, path), body: string(body)}
	}
	return body, nil
}

// instantQuery speaks the Prometheus instant-query API, which Loki also
// serves for metric LogQL under /loki.
type instantQuery struct {
	path string
}

func (p instantQuery) read(q query) (reading, error) {
	body, err := get(q, p.path, url.Values{"query": {q.expr}, "time": {stamp(q.at)}})
	if err != nil {
		return reading{}, err
	}

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string          `json:"resultType"`
			Result     json.RawMessage `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || resp.Status != "success" {
		return reading{}, &badResponse{reason: "response is not a successful instant-query result", body: string(body)}
	}

	switch resp.Data.ResultType {
	case "vector":
		var vector []struct {
			Metric map[string]string  `json:"metric"`
			Value  [2]json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(resp.Data.Result, &vector); err != nil {
			return reading{}, &badResponse{reason: "vector result: " + err.Error(), body: string(body)}
		}
		var r reading
		for _, v := range vector {
			value, err := sampleValue(v.Value)
			if err != nil {
				return reading{}, &badResponse{reason: err.Error(), body: string(body)}
			}
			r.series = append(r.series, series{labels: labelString(v.Metric), value: value})
		}
		return r, nil
	case "scalar":
		var scalar [2]json.RawMessage
		if err := json.Unmarshal(resp.Data.Result, &scalar); err != nil {
			return reading{}, &badResponse{reason: "scalar result: " + err.Error(), body: string(body)}
		}
		value, err := sampleValue(scalar)
		if err != nil {
			return reading{}, &badResponse{reason: err.Error(), body: string(body)}
		}
		return reading{series: []series{{labels: "{}", value: value}}}, nil
	case "streams":
		return reading{}, &configError{errors.New(`the query returned log lines; log steps accept metric LogQL only, such as sum(count_over_time({app="x"} |= "panic" [5m]))`)}
	}
	return reading{}, &configError{fmt.Errorf("the query returned a %q result; it must return an instant vector or scalar", resp.Data.ResultType)}
}

// sampleValue reads the value half of a [time, "value"] pair, which
// Prometheus sends as a string so that NaN and Inf survive JSON.
func sampleValue(pair [2]json.RawMessage) (float64, error) {
	var text string
	if err := json.Unmarshal(pair[1], &text); err != nil {
		return 0, fmt.Errorf("sample value %s is not a string", pair[1])
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("sample value %q is not a number", text)
	}
	return value, nil
}

type alertmanager struct{}

func (alertmanager) read(q query) (reading, error) {
	params := url.Values{"active": {"true"}, "silenced": {"false"}, "inhibited": {"false"}}
	for _, m := range q.matchers {
		params.Add("filter", m)
	}
	body, err := get(q, "/api/v2/alerts", params)
	if err != nil {
		return reading{}, err
	}

	var alerts []struct {
		Labels   map[string]string `json:"labels"`
		StartsAt time.Time         `json:"startsAt"`
		Status   struct {
			State       string   `json:"state"`
			SilencedBy  []string `json:"silencedBy"`
			InhibitedBy []string `json:"inhibitedBy"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &alerts); err != nil {
		return reading{}, &badResponse{reason: "response is not a list of alerts: " + err.Error(), body: string(body)}
	}

	// The query parameters already ask for this; checking again keeps a proxy
	// that drops them from counting silenced alerts.
	var r reading
	for _, a := range alerts {
		if a.Status.State == "active" && len(a.Status.SilencedBy) == 0 && len(a.Status.InhibitedBy) == 0 {
			r.alerts = append(r.alerts, alert{labels: labelString(a.Labels), startsAt: a.StartsAt})
		}
	}
	return r, nil
}

// labelString writes labels sorted by name, as Prometheus prints them.
func labelString(labels map[string]string) string {
	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	sort.Strings(names)
	pairs := make([]string, len(names))
	for i, name := range names {
		pairs[i] = fmt.Sprintf("%s=%q", name, labels[name])
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }
