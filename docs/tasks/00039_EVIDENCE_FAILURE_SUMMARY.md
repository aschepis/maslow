---
id: 39
title: Add maslow evidence summary for short failure feedback
status: draft
priority: low
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, evidence, cli]
---

## Objective

Whoever fixes a failing change needs the failing checks and steps, not the whole evidence file.
Adopters write their own summarisers today, and Maslow already knows the redaction rules, so the
summary should come from Maslow.

## Requirements

- `maslow evidence summary <file>` prints the failing and needs-review results as short Markdown:
  name, status, failed step and target, and the tail of the output.
- Output is capped by `--max-bytes`, and redaction matches the evidence.
- Passing evidence produces a one-line summary.

## Acceptance Criteria

- [ ] Command implemented and wired in `cmd/maslow/main.go`.
- [ ] Golden-file tests for fail, needs_review and pass evidence.
- [ ] CLI contract scenario added to `maslow.yaml`.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V8. The Action's job summary step
could use this instead of its own formatting.
