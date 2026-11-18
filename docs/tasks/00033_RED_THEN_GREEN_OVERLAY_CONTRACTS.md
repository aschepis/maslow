---
id: 33
title: Prove overlay contracts fail on the base and pass on the head
status: in_progress
priority: high
created: 2026-11-18
updated: 2026-11-18
assigned_to: "claude-workflow"
assigned_at: "2026-11-18T12:00:00Z"
depends_on: []
tags: [kind:gap, contracts, overlays, evidence]
---

## Objective

An overlay contract states what "fixed" means for one change. If it already passes on the base
commit, it proves nothing about the change. This is the strongest guard against an agent, or a
vague acceptance step, producing a hollow pass.

## Requirements

- A mode (e.g. `maslow verify --overlay <file> --red-green --base <ref>`) runs each overlay
  contract against the base commit and then the head.
- An overlay contract that passes on the base gets a distinct status (e.g. `not_red`) that makes
  the verdict `fail` or `needs_review` (decide which, record in the ADR).
- The evidence records both runs per contract: base result, head result, base SHA.
- Base contracts (not from an overlay) are unaffected.

## Acceptance Criteria

- [ ] Tests: a contract red on base and green on head passes; green on both is flagged; red on
      both fails as today.
- [ ] Evidence schema updated with the base run.
- [ ] ADR records how the base tree is prepared (worktree, build steps) and the status chosen.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V2. Running against the base needs a
built base tree; `--spec-from-base` (ADR 0030) already resolves the base, which may be reusable.
