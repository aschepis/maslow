# ADR 0019: Load the Spec From the Base Commit

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

With `--base`, policy judges the whole change, but the change still supplies its own `maslow.yaml`. A branch that fails a check can delete that check, or the policy rule that catches it, and pass. The author of a change must not be able to weaken the verification of that change.

## Decision

`maslow verify --base <ref> --spec-from base` reads the spec from the merge-base commit, verifies the current tree against it, and emits a `spec` policy finding that fails when the branch changed the spec.

## Consequences

- A branch that removes a failing check still fails, and the evidence says the spec was changed and which version was used.
- Every legitimate spec change also fails under `--spec-from base`. That is deliberate until `gated` gives a `needs_review` verdict, at which point a spec change routes to a person instead.
- `--spec-from base` requires `--base` and a relative `--spec` path; the default (`head`) keeps the old behaviour for local runs.
