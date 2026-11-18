---
id: 37
title: Link checks and contracts to the requirements they cover
status: draft
priority: medium
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, schema, evidence]
---

## Objective

A consumer cannot answer two basic questions from a spec or its evidence: which requirements have
no contract, and which contract proves issue N is fixed.

## Requirements

- Checks, contracts and scenarios accept a `covers` list of free-form references (issue numbers,
  requirement IDs, URLs).
- The evidence lists, per result, what it covers, and a top-level summary of every reference
  exercised and its outcome.
- Overlay contracts may carry `covers` like any other contract.

## Acceptance Criteria

- [ ] Schema and fixtures for `covers`.
- [ ] Tests: covered references appear in the evidence with the right status.
- [ ] Evidence schema updated; ADR records the reference format.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V6. Whether Maslow should also
report uncovered requirements depends on having a requirement list to compare against; leave that
out unless a clear source exists.
