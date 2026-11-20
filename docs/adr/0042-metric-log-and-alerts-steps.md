# ADR 0042: Metric, Log and Alerts Steps Through Pluggable Providers

- **Status**: Accepted
- **Date**: 2026-11-19

## Context

Post-deploy requirements are often about behaviour over time, such as "the checkout error rate stays under 1% for ten minutes after deploy" (task 36). An endpoint audit could only probe. Generic `http` and `poll` steps cannot express it: assertions compare for equality only, Prometheus sends one string value per series, and `poll` waits until something is true rather than checking that it stays true.

## Decision

Contracts gain three step actions, `metric` (PromQL), `log` (metric LogQL) and `alerts` (active alerts), each naming a `provider` that `internal/runner/providers.go` implements behind one Go interface, while `internal/runner/observe.go` owns thresholds, hold windows, the no-data rule and evidence.

## Details

- **Providers.** `provider.read(query)` turns a query and an evaluation time into series values, or for `alerts` into matching alerts. Working providers: `prometheus` (any Prometheus-compatible `GET /api/v1/query`), `loki` (`GET /loki/api/v1/query`) and `alertmanager` (`GET /api/v2/alerts`). `datadog` validates for all three actions but returns status `error` with "provider datadog is not implemented" (task 47). Later vendors are task 48. A new vendor is one implementation and one map entry.
- **Steps, not checks, budgets or audit targets.** A step inherits the contract's `env` allow-list and redaction, profile and tag selection, endpoint audit selection, overlays and red-green. Contracts tagged for post-deploy run under `maslow audit` with an endpoint target, and against staging under `verify`.
- **Direct APIs, not generic HTTP.** Thresholds would otherwise need numeric comparators, string-to-number coercion, per-series iteration, NaN handling and a hold loop spread across generic primitives, and a pass would still record nothing.
- **Absolute URLs.** `url` is templated but never joined to `base_url`. A relative URL is status `error`. Otherwise an endpoint audit would send the query to the application.
- **Evaluation times.** Each query sets `time` to the clock truncated to the second, so the evidence's `at` replays the same query. Alertmanager has no evaluation time; `at` is when it was asked.
- **Holds look forward.** Without `for`, a step queries once. With `for`, it queries at `t0`, `t0+interval`, ... up to `t0+for` and stops at the first breach. "Over the last N minutes" needs no feature: write `max_over_time(...[10m])`. The schema rejects more than 1000 polls.
- **No data fails.** An empty result, or one where every series is NaN, fails unless `if_no_data: pass`. An error ratio with no traffic would otherwise hide a dead service. NaN series beside real ones are dropped. `+Inf` and `-Inf` compare as numbers.
- **Log steps take metric LogQL only,** such as `sum(count_over_time({app="x"} |= "panic" [5m]))`, so one evaluator serves `metric` and `log`. A stream query is status `error`. Line counting from stream queries was rejected: it is capped by Loki's entry limit.
- **Alerts** counts active alerts that are neither silenced nor inhibited, filtered by `matchers`. `settle` extends polling past `for`, because an alert rule's own `for:` delays firing: a fault late in the window needs that long to show. `ignore_preexisting` drops alerts whose `startsAt` is before the step's first sample. Alerts complement thresholds and do not replace them: they show only what someone already wrote a rule for.
- **Retries.** A breach (a value out of bounds, or no data) is a fact about the system. It is never retried, because a retry costs another hold and could relabel a real breach as `flaky` (ADR 0039). A failed query, non-200 response or malformed body fails the step at once, and is retried. A gap is not evidence that the condition held.
- **Evidence.** Contract results and attempts gain `observations`, one per step that ran, pass or fail: the step's position, provider, the URL as written, the substituted query or matchers, the hold, the bounds, and one sample per query. A sample records `at`, the series or alert count, `min` and `max` across series as strings (so `+Inf` survives JSON), `no_data` and `ok`. `breach` names the first offending series by its sorted labels, so the same responses give byte-identical observations. The field is additive, so `evidence_version` stays `2` (ADR 0025 precedent).
- **Credentials.** Headers only, never recorded. Allow-listed env values are redacted from the query, matchers, URL, breach and sample errors. Values built from parts, such as Basic auth, are not caught (ADR 0026): put the whole header value in one variable.

## Consequences

- **Determinism changes meaning.** The same spec can give a different verdict an hour later. The verdict is a pure function of the recorded samples, and a reviewer can replay each query at its recorded time. Label values appear in `breach`, so queries should aggregate away sensitive labels.
- **This replaces a built-in load runner for production (task 16).** The deployed system's real traffic is the measurement, so nothing sends synthetic load to production and there is no load generator to size. Task 16 remains for pre-deploy latency budgets.
- Endpoint audit now runs these steps. Audit target `timeout` still does nothing for endpoints (ADR 0028), so a long hold is bounded only by the 1000-poll cap. Maslow's own `maslow.yaml` can only validate fixtures, because it has no Prometheus to query; the runner is tested against `httptest` fakes with a fake clock.
