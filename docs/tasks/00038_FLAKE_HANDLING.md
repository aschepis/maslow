---
id: 38
title: Rerun failed checks and contracts and report flakes distinctly
status: draft
priority: medium
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, checks, contracts, evidence]
---

## Objective

A flaky failure sends whoever acts on the evidence to fix something the change did not break. An
automated loop with a fixed number of fix rounds wastes one on it.

## Requirements

- Checks and contracts accept an optional `retries` count (default 0).
- Every attempt is recorded in the evidence with its status, duration and output.
- A result that failed then passed gets a distinct `flaky` status; the ADR decides whether `flaky`
  makes the verdict `pass` or `needs_review`.
- A result that fails every attempt is `fail` as today.

## Acceptance Criteria

- [ ] Schema and fixtures for `retries`.
- [ ] Tests: pass first time, fail then pass (`flaky`), fail every time.
- [ ] Evidence schema updated; ADR records the verdict rule.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V7.
