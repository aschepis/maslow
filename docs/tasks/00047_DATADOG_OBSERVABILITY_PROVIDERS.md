---
id: 47
title: Implement the datadog provider for metric, log and alerts steps
status: draft
priority: medium
created: 2026-11-19
updated: 2026-11-19
assigned_to: ""
assigned_at: ""
depends_on: [36]
tags: [kind:gap, contracts, audit]
---

## Objective

Task 36 added `metric`, `log` and `alerts` steps behind one provider interface
(`internal/runner/providers.go`, ADR 0042). `provider: datadog` validates in the schema, so specs
can be written now, but at run time the step returns status `error` with "provider datadog is not
implemented". Teams on Datadog cannot use post-deploy assertions until it works.

## Questions to answer

- **Metrics.** Use `GET /api/v1/query` (from, to, query) or the v2 timeseries API? Datadog has no
  instant query, so which point in the returned series is the value "at" the evaluation time, and
  how is the rollup chosen so the same query replays to the same value?
- **Logs.** The log `query` would be Datadog log search syntax, not LogQL. Is the value the count
  from `POST /api/v2/logs/analytics/aggregate` over a window, and where does the window come from,
  since a Datadog query has no `[5m]` range?
- **Alerts.** Map to monitors in `Alert` state (`GET /api/v1/monitor` with `monitor_tags`), or to
  the events stream? How do `matchers`, `settle` and `ignore_preexisting` translate?
- **Auth.** `DD-API-KEY` and `DD-APPLICATION-KEY` headers through the contract's `env`. Is `url`
  the site (`https://api.datadoghq.eu`)?
- **Tests.** An `httptest` fake per API, as for Prometheus, Loki and Alertmanager.

## Acceptance Criteria

- [ ] Datadog metric, log and alerts providers return readings that the existing evaluator judges.
- [ ] Tests against a stub Datadog API.
- [ ] ADR 0042 or a new ADR records the query semantics chosen.
