# ADR 0031: Policy Observe Mode

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

A new `deny`, `allow`, `protected` or `gated` rule is enforced from the first run, so a false positive shows up as a blocked change. Policy lists are plain strings, so a per-rule flag would need an object form beside strings in every list, and each new policy kind (task 34) would need its own handling of the flag.

## Decision

`policy` gains an optional `observe` block with the same rule shape as `policy` itself; its rules run exactly as enforced rules do, and each result is then marked `mode: "observe"` with `fail`, `needs_review` and `error` renamed `would_fail`, `would_need_review` and `would_error`, and `ComputeVerdict` skips every observe-mode result.

## Consequences

- Any policy kind added to `#PolicyRules` and `runRules` gets observe mode for free; promoting a rule is moving it out of `observe`, and the same rule may appear in both while it is being compared.
- Observed `allow` patterns form their own allow list, judged independently of the enforced one. A run whose only results are observed is `inconclusive`, because an observed pass proves nothing.
- Evidence stays version 2 (ADR 0020): `mode` is a new optional field and the `would_*` statuses are new enum values, both described in `schema/evidence.schema.json`. A reader that treats unknown statuses as failures must learn to skip `mode: observe` entries.
