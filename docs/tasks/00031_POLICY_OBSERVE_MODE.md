---
id: 31
title: Let a new policy rule observe before it enforces
status: done
priority: low
created: 2026-11-18
updated: 2026-11-18
assigned_to: "claude-workflow"
assigned_at: "2026-11-18T20:21:44Z"
depends_on: []
tags: [kind:gap, policy, evidence]
---

## Objective

A new `deny`, `allow`, `protected` or `gated` rule takes effect immediately, so the first sign of a
false positive is a blocked change. Running a rule in observe mode first would show what it would
have flagged, on real changes, without affecting the verdict.

## Requirements

- A policy rule can be marked observe-only.
- An observe-only finding appears in the evidence with the status it would have had
  (e.g. `would_fail`, `would_need_review`) and does not change the verdict or exit code.
- Enforced rules behave exactly as today.

## Acceptance Criteria

- [x] Schema and fixtures for marking a rule observe-only.
- [x] Tests: an observed violation leaves the verdict `pass`; the same rule enforced fails.
- [x] Evidence schema updated; ADR records the design.

## Notes

The policy lists are plain strings today, so a per-rule flag needs either an object form
(`{path, mode}`) beside strings or a parallel `observe:` block. Checks and contracts could get the
same mode later; policy comes first because a wrong rule there blocks every change.
