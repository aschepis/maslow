---
id: 30
title: Compare a contract's responses across two targets
status: draft
priority: low
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, contracts, audit]
---

## Objective

For migrations and rewrites the spec is "behave like the old system", which is tedious to write
as explicit expectations. An endpoint audit target can already run contracts against a base URL
(ADR 0028), but nothing compares two runs. A differential mode would run the same scenarios
against a reference and a candidate and fail on unexplained differences.

## Requirements

- Run selected contracts against a reference and a candidate target with the same inputs.
- Compare status, headers and bodies (stdout and exit code for CLI steps) step by step.
- Normalization rules to ignore irrelevant differences: JSON paths to drop (timestamps, IDs),
  whitespace, header names.
- An approved-differences list so known, intended changes do not fail.
- Evidence records each difference with the step, the field, and both values, under the output cap
  and env redaction (ADR 0026).

## Acceptance Criteria

- [ ] Schema and fixtures for declaring the two targets and normalization.
- [ ] Tests: identical targets pass; a changed field fails; a normalized or approved field passes.
- [ ] Evidence schema updated; ADR records the design.

## Notes

Open questions: whether this belongs under `audit` (two endpoint targets) or as a contract option;
how to handle steps with side effects, since running them against both targets doubles them.
