---
id: 40
title: Add maslow evidence diff to compare two evidence files
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

Comparing two runs answers what a push broke, what it fixed, and what got slower. Today that means
diffing JSON by hand.

## Requirements

- `maslow evidence diff <old> <new>` lists results that regressed, were fixed, appeared or
  disappeared, and duration changes above a threshold.
- Results are matched by section and name; output is deterministic.
- `--json` emits the same as structured data.

## Acceptance Criteria

- [ ] Command implemented and wired in `cmd/maslow/main.go`.
- [ ] Tests for each change kind, and for evidence files of different versions.
- [ ] CLI contract scenario added to `maslow.yaml`.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V9.
