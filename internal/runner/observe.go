package runner

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

// clock lets tests run a ten-minute hold instantly.
type clock struct {
	now   func() time.Time
	sleep func(time.Duration)
}

var realClock = clock{now: time.Now, sleep: time.Sleep}

// configError is a problem with how a step is written. Retrying cannot fix
// it, so the scenario's status is "error" rather than "fail".
type configError struct{ err error }

func (e *configError) Error() string { return e.err.Error() }
func (e *configError) Unwrap() error { return e.err }

// breachError is a failure on what the system reported, as opposed to a
// failure to ask; it is never retried.
type breachError struct{ reason string }

func (e *breachError) Error() string { return e.reason }

// maxPolls matches the schema's cap on one hold.
const maxPolls = 1000

func isObserveAction(action string) bool {
	return action == "metric" || action == "log" || action == "alerts"
}

// runObserveStep runs a metric, log or alerts step and records what it saw,
// whether it passes or not. rawURL is the url as written, before templating.
func runObserveStep(phase string, index int, step spec.Step, rawURL string, state *stepState, sc *scope) error {
	obs := evidence.Observation{
		Phase:             phase,
		Index:             index,
		Action:            step.Action,
		Provider:          step.Provider,
		URL:               rawURL,
		Query:             step.Query,
		Matchers:          step.Matchers,
		IgnorePreexisting: step.IgnorePreexisting,
		For:               step.For,
		Interval:          step.Interval,
		Settle:            step.Settle,
		Samples:           []evidence.Sample{},
	}
	if step.Expect != nil {
		obs.Expect = evidence.ObservationBounds{
			Min:       step.Expect.Min,
			Max:       step.Expect.Max,
			MinSeries: step.Expect.MinSeries,
			MaxSeries: step.Expect.MaxSeries,
			IfNoData:  step.Expect.IfNoData,
		}
	}
	err := observe(step, &obs, state, sc)
	sc.observations = append(sc.observations, obs)
	return err
}

func observe(step spec.Step, obs *evidence.Observation, state *stepState, sc *scope) error {
	p, ok := providers[step.Action][step.Provider]
	if !ok {
		return &configError{fmt.Errorf("unknown %s provider %q", step.Action, step.Provider)}
	}
	if !strings.HasPrefix(step.URL, "http://") && !strings.HasPrefix(step.URL, "https://") {
		return &configError{fmt.Errorf("url %q is not absolute; %s URLs are never joined to base_url", step.URL, step.Action)}
	}
	if step.Expect == nil {
		return &configError{fmt.Errorf("%s step missing expect", step.Action)}
	}
	times, err := sampleTimes(step, sc.clock.now())
	if err != nil {
		return &configError{err}
	}

	for _, at := range times {
		if wait := at.Sub(sc.clock.now()); wait > 0 {
			sc.clock.sleep(wait)
		}
		r, err := p.read(query{url: step.URL, headers: step.Headers, expr: step.Query, matchers: step.Matchers, at: at})
		if err != nil {
			obs.Samples = append(obs.Samples, evidence.Sample{At: stamp(at), Error: err.Error()})
			var bad *badResponse
			if errors.As(err, &bad) {
				state.output = bad.body
			}
			return err
		}
		if step.IgnorePreexisting {
			r.alerts = startedSince(r.alerts, times[0])
		}

		var sample evidence.Sample
		var breach string
		if step.Action == "alerts" {
			sample, breach = judgeAlerts(step.Expect, r.alerts)
		} else {
			sample, breach = judgeSeries(step.Expect, r.series)
		}
		sample.At = stamp(at)
		obs.Samples = append(obs.Samples, sample)
		if breach != "" {
			obs.Breach = "at " + sample.At + ": " + breach
			return &breachError{obs.Breach}
		}
	}
	return nil
}

// sampleTimes are the evaluation times of a step: now, then every interval
// until for and settle have passed. Whole seconds keep each query replayable.
func sampleTimes(step spec.Step, now time.Time) ([]time.Time, error) {
	var hold time.Duration
	for _, field := range []string{step.For, step.Settle} {
		if field == "" {
			continue
		}
		d, err := parseDuration(field)
		if err != nil {
			return nil, err
		}
		hold += d
	}

	t0 := now.Truncate(time.Second)
	if hold == 0 {
		return []time.Time{t0}, nil
	}
	interval, err := parseDuration(step.Interval)
	if err != nil || interval <= 0 {
		return nil, fmt.Errorf("interval %q must be a positive duration when for or settle is set", step.Interval)
	}
	polls := int(hold / interval)
	if polls > maxPolls {
		return nil, fmt.Errorf("%s at every %s is %d polls; the most is %d", hold, interval, polls, maxPolls)
	}

	times := make([]time.Time, 0, polls+1)
	for k := 0; k <= polls; k++ {
		times = append(times, t0.Add(time.Duration(k)*interval))
	}
	return times, nil
}

// judgeSeries checks every series against the bounds. NaN is no data, so a
// ratio over no traffic cannot pass by accident.
func judgeSeries(expect *spec.Expectation, all []series) (evidence.Sample, string) {
	var data []series
	for _, s := range all {
		if !math.IsNaN(s.value) {
			data = append(data, s)
		}
	}
	sort.Slice(data, func(i, j int) bool { return data[i].labels < data[j].labels })

	n := len(data)
	sample := evidence.Sample{Series: &n, OK: true}
	if n == 0 {
		sample.NoData = true
		if expect.IfNoData == "pass" {
			return sample, ""
		}
		sample.OK = false
		return sample, "no data: the query returned no series, or only NaN"
	}

	lo, hi := data[0].value, data[0].value
	for _, s := range data {
		lo, hi = math.Min(lo, s.value), math.Max(hi, s.value)
	}
	sample.Min, sample.Max = formatValue(lo), formatValue(hi)

	breach := seriesBreach(expect, data)
	sample.OK = breach == ""
	return sample, breach
}

// seriesBreach names the first bound the series cross, or returns "".
func seriesBreach(expect *spec.Expectation, data []series) string {
	n := len(data)
	if expect.MinSeries != nil && n < *expect.MinSeries {
		return fmt.Sprintf("%d series is below min_series %d", n, *expect.MinSeries)
	}
	if expect.MaxSeries != nil && n > *expect.MaxSeries {
		return fmt.Sprintf("%d series is above max_series %d", n, *expect.MaxSeries)
	}
	for _, s := range data {
		if expect.Min != nil && s.value < *expect.Min {
			return fmt.Sprintf("series %s = %s is below min %s", s.labels, formatValue(s.value), formatValue(*expect.Min))
		}
		if expect.Max != nil && s.value > *expect.Max {
			return fmt.Sprintf("series %s = %s is above max %s", s.labels, formatValue(s.value), formatValue(*expect.Max))
		}
	}
	return ""
}

// judgeAlerts checks the number of alerts against the bounds, naming the
// first alert in label order so the same alerts give the same reason.
func judgeAlerts(expect *spec.Expectation, alerts []alert) (evidence.Sample, string) {
	sort.Slice(alerts, func(i, j int) bool { return alerts[i].labels < alerts[j].labels })

	n := len(alerts)
	sample := evidence.Sample{Alerts: &n, OK: true}
	breach := ""
	switch {
	case expect.Max != nil && float64(n) > *expect.Max:
		breach = fmt.Sprintf("%d alerts firing is above max %s, first %s", n, formatValue(*expect.Max), alerts[0].labels)
	case expect.Min != nil && float64(n) < *expect.Min:
		breach = fmt.Sprintf("%d alerts firing is below min %s", n, formatValue(*expect.Min))
	}
	sample.OK = breach == ""
	return sample, breach
}

func startedSince(alerts []alert, start time.Time) []alert {
	var kept []alert
	for _, a := range alerts {
		if !a.startsAt.Before(start) {
			kept = append(kept, a)
		}
	}
	return kept
}

func formatValue(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
