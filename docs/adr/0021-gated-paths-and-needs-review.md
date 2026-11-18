# ADR 0021: Gated Paths, Allow, and the needs_review Verdict

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

`gated` paths were parsed but always skipped, and `allow` was parsed and ignored. Some changes are neither acceptable nor forbidden; they need a named person to look. A pass/fail gate cannot express that, so pipelines either block those changes or wave them through.

## Decision

A change touching a `gated` path gives the policy status `needs_review` with the path's `owner`, and the verdict precedence becomes fail, needs_review, inconclusive, pass; `allow`, when set, fails any changed file outside every allowed pattern.

## Consequences

- `needs_review` exits 0, so a pipeline reads the verdict and requests review; `--strict` makes it exit 1. `inconclusive` still exits 1 because a gate that ran nothing has not passed anything.
- A changed spec under `--spec-from base` is now `needs_review`, not `fail` (ADR 0019): the run was already verified against the base spec, so the open question is whether to accept the new spec.
- `owner` stays a free string. A GitHub handle or team (`@org/team`) is recommended so a pipeline can request review directly.
