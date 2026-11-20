---
id: 36
title: Add contract steps that assert on metrics, logs and alerts through pluggable providers
status: todo
priority: medium
created: 2026-11-18
updated: 2026-11-19
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, contracts, audit]
---

## Objective

The real post-deploy requirement is often about behaviour over time: "the error rate for this route
stays under 1% for ten minutes after deploy". An endpoint audit today can only probe. Asserting on
metrics the system already collects turns the audit into a check on live behaviour, without a
load generator.

## Requirements

- A `metric` step that runs a fixed PromQL query and a `log` step that runs a fixed LogQL query
  against a configured URL.
- Assertions on the result: value thresholds, sample count, and "holds for duration" by polling.
- Credentials come only from the contract's `env` allow-list and are redacted from evidence.
- The query and the observed value are recorded in the evidence.

## Acceptance Criteria

- [ ] Schema and fixtures for both steps.
- [ ] Tests against a stub Prometheus and Loki HTTP API.
- [ ] ADR records the step shape and why this replaces a built-in load runner for production
      (see task 16).

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V5.

## Decisions (human review, 2026-11-19)

These override the Requirements and the proposal below where they differ.

1. **Providers.** `metric`, `log` and `alerts` steps take a `provider` field. A provider turns a
   query and an evaluation time into numbers (or, for `alerts`, matching alerts); Maslow owns
   thresholds, hold windows, the no-data rule and evidence, so semantics are identical across
   vendors. Providers sit behind one Go interface so a new vendor is one implementation.
2. **Providers in this task:** `metric`: `prometheus` (any Prometheus-compatible query API) and
   `datadog`; `log`: `loki` and `datadog`; `alerts`: `alertmanager` and `datadog`.
3. **Datadog is a stub.** `provider: datadog` validates in the schema, so specs can be written
   now, but at run time the step returns status `error` with "provider datadog is not implemented".
   File a `kind:gap` draft task for the real Datadog providers, plus one listing later vendors
   (New Relic, CloudWatch, SigV4 for AWS Managed Prometheus).
4. **`alerts` step.** Queries Alertmanager `GET /api/v2/alerts` with label matchers, polled forward
   like `metric`. Counts active, unsilenced, uninhibited alerts. Takes a `settle` duration because
   alert rules have their own `for:` lag, and an option to ignore alerts whose `startsAt` precedes
   the step's start. The ADR says alerts complement thresholds, not replace them.
5. **`log` steps accept metric LogQL only** (e.g. `count_over_time(...)`), so one evaluator serves
   `metric` and `log`.
6. Otherwise the proposal's recommendations stand: poll forward, no data fails unless
   `if_no_data: pass`, breaches are never retried, a failed query mid-hold fails the step at once,
   at most 1000 polls per hold, header auth only.

## Proposed implementation

### 1. What happens today

Nothing in Maslow reads metrics or logs. The pieces a metric assertion would be built from are these:

- **Step actions.** `#Step.action` is one of `http`, `cli`, `poll`, `assert`, `capture`, `wait` or
  `browser` (`schema/maslow.cue:196`). `runStep` dispatches on it (`internal/runner/contract.go:313-380`).
  `spec.Step` (`internal/spec/spec.go:156-175`) has no query field. `substituteStep`
  (`contract.go:454-509`) templates URL, body, headers, args and browser fields.
- **Generic HTTP plus JSON assertions cannot express a threshold.** `#Expectation` and `#Assertion`
  (`maslow.cue:273-294`) compare a JSON path to a value for equality only. `jsonValuesEqual`
  (`contract.go:895-921`) has no less-than or greater-than. Prometheus returns each sample as
  `[<unix time>, "<value as string>"]`, one entry per series, so even equality would compare a string
  per series. `every` and `some` (`contract.go:759-804`) iterate arrays but still compare for
  equality. A PromQL query in a URL must also be URL-encoded by hand.
- **Poll cannot hold a condition over time.** `runPollStep` (`contract.go:611-664`) sends only `GET`
  (`:629`), sends no headers (so no auth), and returns on the first response that matches `status` or
  `body_contains` (`:646-653`). It is "wait until true", not "stays true for N minutes". It swallows
  transport errors (`:635-640`) and reports only `poll timed out` (`:663`). `wait` sleeps
  (`:346-351`).
- **Env and redaction.** `RunContractsWith` builds the contract's allow-listed env (`contract.go:154`,
  `contract_env.go:16-24`) and redacts every result (`contract.go:156`). `redact` covers `error`,
  `output`, `failed_step` and attempts (`contract_env.go:28-47`). Header values may read
  `{{.env.NAME}}` (`contract.go:486-495`), and headers never reach the evidence. Values shorter than
  four bytes are not redacted (`contract_env.go:12`), and transformed values are not caught (ADR 0026).
- **Endpoint audit.** `auditEndpoint` (`internal/audit/audit.go:159-185`) selects contracts by name
  or tag and runs them with the target path as every contract's base URL (`:180`). `resolveURL`
  (`contract.go:512-520`) joins any URL starting with `/` to that base. A Prometheus URL written as
  `/api/v1/query` would therefore be sent to the application, not to Prometheus. Audit target
  `timeout` and `retries` do nothing for endpoints (ADR 0028).
- **Evidence.** A passing scenario records only `name` and `status` (`contract.go:251`).
  `ContractResult` (`internal/evidence/report.go:101-121`) has no place for an observed value.
  `evidence.schema.json` sets `additionalProperties: false` on `contract` (`:90-111`). Evidence
  version is `"2"` (`report.go:18`).
- **Budgets.** `performance` and `availability` budgets are in the schema (`maslow.cue:302-325`) but
  the runner skips them (`internal/runner/budget.go:29-47`). Budgets have no env allow-list, and
  endpoint audit does not run them.
- **Clock.** Steps call `time.Now` and `time.Sleep` directly, so a ten-minute hold would take ten
  minutes in a test.

### 2. Proposed change

**Shape: two new step actions, `metric` and `log`.** A new check kind would have no env allow-list
(task 41 is still open) and no structured result. A new audit target would only run under `audit`,
and would duplicate env handling and contract selection. A budget is not selected by endpoint
audit and has no env list. A step gets all of this for free: the env allow-list and redaction,
profile and tag selection, endpoint audit selection (Damocles tags its post-deploy contracts and
names the tag in the target's `checks`), overlays and red-green. A metric contract also runs under
`verify` against a staging Prometheus.

**Direct API, not generic HTTP.** Generic steps would need numeric comparators, string-to-number
coercion, per-series iteration, NaN handling, empty-result handling, URL encoding and a hold
loop. That is the whole feature spread across generic primitives, and it would still leave nothing
in the evidence for a pass. The steps call `GET <url>/api/v1/query` (metric) and
`GET <url>/loki/api/v1/query_range` (log), and parse the documented response shape. Loki's metric
API has the same response shape under `/loki`, so a metric-shaped LogQL query (`count_over_time`,
`rate`) uses a `metric` step with `url: http://loki:3100/loki`. Adding numeric comparators to
`#Assertion` is worth doing separately; it is not needed here.

**Schema (`schema/maslow.cue`, copied to `internal/schema/maslow.cue`).**

```yaml
- action: metric
  url: "{{.env.PROM_URL}}"            # absolute; never joined to base_url
  headers: {Authorization: "Bearer {{.env.PROM_TOKEN}}"}
  query: 'sum(rate(http_requests_total{route="/checkout",code=~"5.."}[1m]))
          / sum(rate(http_requests_total{route="/checkout"}[1m]))'
  for: 10m                            # optional; omitted means one query
  interval: 30s                       # required with for
  expect: {max: 0.01, if_no_data: fail}
- action: log
  url: "{{.env.LOKI_URL}}"
  headers: {X-Scope-OrgID: "{{.env.LOKI_TENANT}}"}
  query: '{app="checkout"} |= "panic"'
  window: 5m                          # look-back of each query
  for: 10m
  interval: 5m
  expect: {max_lines: 0}
```

- `#MetricExpectation`: `min?`, `max?` (inclusive numbers), `min_series?`, `max_series?`,
  `if_no_data: *"fail" | "pass"`. `#LogExpectation`: `min_lines?`, `max_lines?`. These are separate
  closed definitions, as `#BrowserExpectation` is, so `max` on an `http` step stays invalid.
- `query` and `url` are required. `interval` is required when `for` is set. `window` is required for
  `log`. Durations use the existing `s|m|h` pattern.
- `spec.Step` gains `Query`, `For` and `Window`. `spec.Expectation` gains the new fields, as it did for
  the browser fields (`spec.go:191-195`). `substituteStep` templates `Query`.

**Runner (`internal/runner/observe.go`, new).**

- `url` is templated but not passed through `resolveURL`. A URL that is not absolute after
  templating fails the step with an error saying metric and log URLs are never joined to
  `base_url`. This keeps endpoint audit from pointing the query at the application.
- Each query sets an explicit evaluation time. `time=<t>` is used for metric, and
  `start=<t-window>&end=<t>&limit=<n>` for log, where `t` is the clock truncated to the second. The
  same query can be replayed later from the evidence. The request carries the step's headers and a
  30-second timeout, as `http` does.
- **Hold semantics.** Without `for`, the step queries once. With `for`, it queries at `t0`,
  `t0+interval`, ... until `t0+for` inclusive, so there are `for/interval + 1` samples. It stops at
  the first breach. A non-200 response, a body that is not the documented shape or a transport
  error fails the step at once, with the response body as output. A gap in the data is not counted
  as holding. "Over the last N minutes" needs no Maslow feature: write `max_over_time(...[10m])` and
  omit `for`.
- **Judging a metric sample.** A `vector` result is checked series by series: every series must be
  within `[min, max]`. The series count must be within `[min_series, max_series]`. A `scalar` result
  is one series. `NaN`, an empty vector, or a ratio with no traffic all count as no data, which
  `if_no_data` decides. `+Inf` and `-Inf` are compared as numbers. A `matrix` or `streams` result is
  an error that names the query type.
- **Judging a log sample.** The step counts entries across all streams. The request asks for
  `limit = max(min_lines, max_lines) + 1` lines, so it never pages and still knows whether the bound
  was crossed. A count that reaches the limit is recorded as capped.
- Judging is a pure function, `judge(expect, sample) (ok bool, reason string)`, separate from I/O.
- `contractConfig` gains a `clock` (`now`, `sleep`). It defaults to the real clock. Tests set a fake
  one, so a ten-minute hold runs instantly.
- `stepTarget` returns `"metric <query>"` or `"log <query>"`.

**Evidence.**

- `ContractResult` and `ContractAttempt` gain `observations`, one per metric or log step that ran,
  on pass or fail. Each has `phase`, `index`, `action`, `query` (substituted), `for`, `interval`,
  `window` and the bounds. It also has `samples`, one per query: `at` (RFC 3339 evaluation time),
  `series`, `min`, `max` (or `lines` and `capped`), `no_data` and `ok`. On a breach it adds
  `breach`, a reason that names the first offending series by its sorted label set.
- Values are recorded per sample as min and max across series, not every series, so size grows with
  the number of samples, not cardinality. Series are sorted by their label string before judging,
  so the same responses always produce the same observation and the same reason.
- `redact` is extended to `query` and `breach`. The URL is recorded as written in the spec, before
  templating, as endpoint audit does for its path. Headers are never recorded.
- The evidence JSON schema gets an `observation` definition. The field is additive and optional, so
  `evidence_version` stays `2` (ADR 0025 precedent).

**Retries.** A threshold breach is a fact about production, and rerunning it would take another
`for` of wall time and could turn a real breach into `flaky`. A scenario that failed on a metric or
log breach is not retried. A transport error still is. `shouldRetry` (`internal/runner/retry.go:12`)
sees only a status, so `stepFailure` (`contract.go:40-50`) gains a `breach` flag. `failedResult`
copies it to a `json:"-"` field on the result, and `runContract` (`contract.go:188`) checks it beside
`shouldRetry`.

**Docs and spec.** README contract-step section and `docs/MAP.md` runner section. A CHANGELOG entry.
`maslow.yaml` gets CLI contracts that run `maslow validate` on the new valid and invalid fixtures. No
self-hosted contract needs a real Prometheus.

### 3. Tests and checks

All of these are written first and must fail before any runner or schema code changes. The schema
tests fail on the closed `#Step`. The runner tests fail with `unknown step action: metric`.

**Schema fixtures** (picked up by the glob at `internal/schema/validate_test.go:244`):

- valid: `testdata/valid/metric_log_contracts.yaml`, with an instant metric step, a held metric
  step, a Loki-backed metric step and a log step.
- invalid: `metric_without_query.yaml`, `metric_for_without_interval.yaml`, `log_without_window.yaml`,
  `metric_http_expectation.yaml` (`status` on a metric step), `http_metric_expectation.yaml` (`max`
  on an http step), `metric_unknown_if_no_data.yaml`.

**`internal/runner/observe_test.go`**, using an `httptest.Server` fake Prometheus and Loki. The fake
answers by the request's `time` (or `end`) parameter from a scripted table. It records the request
path, query string and headers. A fake clock advances on `sleep`.

| Case | Fake returns | Expect | Result |
|---|---|---|---|
| instant, under max | vector [0.004] | max 0.01 | pass, 1 sample |
| instant, over max | vector [0.02] | max 0.01 | fail, breach names series |
| two series, one over | [0.001, 0.03] | max 0.01 | fail, breach names the second by labels |
| held, all under | 21 samples under | for 10m, interval 30s | pass, 21 samples, times t0..t0+10m |
| held, breach at sample 5 | sample 5 over | for 10m | fail, 5 samples recorded, no further requests |
| empty vector | [] | default | fail, `no_data` |
| empty vector, allowed | [] | if_no_data pass | pass |
| NaN | ["NaN"] | max 0.01 | fail, `no_data` |
| series count | 3 series | max_series 2 | fail |
| scalar | scalar 5 | min 1 | pass |
| matrix result | matrix | max 1 | fail, error names result type |
| HTTP 500 mid-hold | 500 at sample 3 | for 10m | fail, body in output, 3 requests |
| relative URL | n/a | url "/api" | fail, error names base_url |
| explicit time | any | any | request has `time=` equal to recorded `at` |
| log over max | 2 entries | max_lines 0 | fail, lines 2 |
| log limit | n/a | max_lines 3 | request has `limit=4`, `start` = end - window |
| Loki metric via /loki | vector | max 1 | request path `/loki/api/v1/query` |
| breach not retried | over | retries 2 | 1 attempt, status fail |
| transport error retried | 500 then ok | retries 1 | flaky |

**Redaction**, in `contract_env_test.go`: a token header reaches the fake server but appears nowhere
in the marshalled result. An env value used in `query` is recorded as `[redacted:NAME]`.

**Determinism**: two runs against the same scripted fake with the same fake clock marshal to
byte-identical `observations`.

**Audit**, in `internal/audit/audit_test.go`: an endpoint target selects a tagged metric contract.
The fake Prometheus is hit at its own URL, not the target path, and the observation appears on the
audit's contract result.

**Evidence**: `schema/evidence.schema.json` validation test with an `observations` entry.

Run `go test ./...`, `go vet ./...`, `cue vet schema/maslow.cue` and `maslow verify --profile quick`.

### 4. ADR

ADR 0042, "Metric and log steps assert on Prometheus and Loki". It records the step shape, the
direct-API choice over generic HTTP, absolute URLs that ignore `base_url`, explicit evaluation times,
look-forward hold semantics, no retry on breach, and the `observations` evidence field. It also
says why this replaces a built-in load runner for production (task 16): the deployed system's real
traffic is the measurement, so there is no synthetic load against production and no load generator
to size. Task 16 remains for pre-deploy latency budgets. ADR 0028 gets no edit; the new ADR notes
that endpoint audit now runs these steps.

### 5. Risks and open questions

1. **Look-forward polling or look-back range query for "holds for N minutes"?** A single
   `query_range` over the past N minutes would be one request, but right after a deploy there is no
   post-deploy data to look back on. *Recommendation:* look-forward polling with `for`. Look-back is
   already a single PromQL query with `max_over_time`.
2. **What does no data mean?** An error-rate ratio is `NaN` or empty when there is no traffic, and
   that could hide a dead service. *Recommendation:* fail by default, with `if_no_data: pass` as the
   explicit opt-out. Pair it with a traffic `min` step when the rate must be meaningful.
3. **Log steps: count lines from stream queries, or only allow metric LogQL?** Stream counting is
   capped by Loki's entry limit, but "no panic lines" is the common case and reads naturally.
   *Recommendation:* support line counts with the `limit = bound + 1` trick, and route exact counts
   through a `metric` step pointed at `/loki`.
4. **Retry on breach?** Rerunning a ten-minute hold doubles the wait and can relabel a real breach as
   `flaky` (ADR 0039). *Recommendation:* never retry a breach, and retry only transport and
   shape errors.
5. **Missed polls mid-hold.** A Prometheus blip fails a ten-minute check on its last sample.
   *Recommendation:* fail at once in v1, because a gap is not evidence that the condition held. Add
   `max_missed` later if it bites.
6. **Cap on `for`.** A typo such as `for: 10h` stalls a pipeline, and the audit target `timeout` does
   not apply (ADR 0028). *Recommendation:* reject more than 1000 samples (`for/interval`) at
   validation, and make audit target `timeout` apply to endpoint targets in a follow-up task.
7. **Auth beyond headers.** Basic auth needs a base64 value, which redaction would not catch if
   Maslow built it from parts (ADR 0026). mTLS needs client certificates. *Recommendation:* headers
   only in v1. The user puts the full `Basic ...` value in one env variable, which is then redacted
   whole. File a `kind:gap` task for mTLS if Damocles needs it.
8. **Risk: the verdict depends on a live system.** The same spec can give a different result an hour
   later, so the evidence cannot be reproduced by rerunning. *Mitigation:* the evidence records each
   query, its exact evaluation time and the values judged. The verdict is a pure function of those
   records, and a reviewer can replay any query against Prometheus at the recorded time. The ADR
   should state this as the meaning of "deterministic" for these steps.
9. **Risk: label values in the breach reason may be sensitive** (user IDs in high-cardinality
   labels). *Recommendation:* record labels as returned, apply env redaction, and say in the ADR that
   queries should aggregate away sensitive labels.
