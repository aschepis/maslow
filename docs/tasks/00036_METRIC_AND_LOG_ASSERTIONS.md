---
id: 36
title: Add contract steps that assert on Prometheus and Loki queries
status: draft
priority: medium
created: 2026-11-18
updated: 2026-11-18
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
