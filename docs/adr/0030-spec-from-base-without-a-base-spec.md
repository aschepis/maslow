# ADR 0030: Spec From Base When the Base Has No Spec

- **Status**: Accepted
- **Date**: 2026-11-18
- **Amends**: ADR 0019

## Context

With `--spec-from base`, `maslow verify` read the spec at the merge base and exited 1 before writing evidence if that failed. The pull request that adopts Maslow always has no spec at its base, so a pipeline using the recommended settings failed that pull request with no evidence file and no verdict to act on.

## Decision

When the base has no spec file, verify against the branch's spec with `spec_source: "head"` and add a `spec` policy result with `needs_review`; any other failure to load the base spec writes evidence with a `spec` policy `error` and verdict `fail`.

## Consequences

- The adopting change is verified for real: its failures still fail, and a person must accept the new spec before it gates anything.
- A branch cut from a commit older than the spec also falls back, since its merge base has no spec. That run still ends at `needs_review` at best, so the fallback never lets a branch pass on a spec it wrote.
- A malformed or unreadable base spec still blocks the run, but now the evidence says why.
