---
id: 39
title: Add maslow evidence summary for short failure feedback
status: draft
priority: low
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: [43]
tags: [kind:gap, evidence, cli]
---

## Objective

Whoever fixes a failing change needs the failing checks and steps, not the whole evidence file.
Adopters write their own summarisers today, and Maslow already knows the redaction rules, so the
summary should come from Maslow.

## Requirements

Task 43 added `maslow evidence summary <file>` and `evidence.Summary` (`internal/evidence/summary.go`)
with the Action's table of failing and needs-review results plus observed policy findings. This task
extends them:

- For each failing and needs-review result, add the failed step and target and the tail of the output.
- Cap output with `--max-bytes`, and keep redaction matching the evidence.
- Passing evidence produces a one-line summary.

## Acceptance Criteria

- [ ] `Summary` and `evidence summary` extended; no second command.
- [ ] Golden files in `testdata/evidence/*.summary.md` updated or added for fail, needs_review and pass evidence.
- [ ] CLI contract scenario in `maslow.yaml` for `--max-bytes`.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V8. The Action's job summary already
calls `maslow evidence summary` (task 43, ADR 0036), so extending `Summary` changes it too.
