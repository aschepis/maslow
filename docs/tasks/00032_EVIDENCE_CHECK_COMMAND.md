---
id: 32
title: Add maslow evidence check to judge an evidence file
status: draft
priority: high
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, evidence, cli]
---

## Objective

Every adopter that gates a merge on Maslow re-implements the same checks on the evidence file:
format version, head commit, spec hash, profile and verdict. Damocles carries this as 115 lines of
Ruby (`bin/lib/maslow_evidence.rb`), and getting one check wrong approves a bad change. Maslow
should own this decision.

## Requirements

- `maslow evidence check <file> --head <sha> --spec <trusted spec> [--profile <name>]` answers
  `pass`, `needs_review` or `decline`.
- `decline` when the evidence version is unknown, `git_sha` differs from `--head`, `spec_sha256`
  differs from the trusted spec's hash, the profile differs, or the verdict is `fail` or
  `inconclusive`.
- `needs_review` when the verdict is `needs_review`; `pass` only when everything matches and the
  verdict is `pass`.
- Prints the reason for anything other than `pass`; `--json` emits the decision and reasons.
- Exit codes: `0` pass, `1` needs_review or decline, `2` usage error.

## Acceptance Criteria

- [ ] Command implemented and wired in `cmd/maslow/main.go`.
- [ ] Table tests cover each decline reason, needs_review, and pass.
- [ ] CLI contract scenario added to `maslow.yaml`.
- [ ] ADR records the decision rules; README documents the command.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V1. Overlays complicate the spec
hash: the check should also accept a list of trusted overlay hashes, matching `overlays` in the
evidence.
