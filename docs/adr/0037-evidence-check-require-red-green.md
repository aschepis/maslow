# ADR 0037: Evidence Check Can Require Red-Green

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

`maslow evidence check` (ADR 0033) trusts the verdict, so evidence from a plain `--overlay` run passes the same as evidence from `verify --red-green` (ADR 0034). A gate cannot require that every acceptance contract failed on the merge base and passed on the head, and an agent that leaves off `--red-green` gets a hollow pass.

## Decision

`maslow evidence check --require-red-green --base <sha>` declines unless every overlay contract has a base run on exactly `--base` with status `fail`, a `head_status` of `pass` and a combined `status` of `pass`, and every overlay in the run has at least one contract result; `--base` alone checks only that `base_sha` equals it.

## Consequences

- `--require-red-green` without `--base`, or without `--overlay`/`--overlay-sha256`, is a usage error (exit 2), since both are the gate's own misconfiguration. `--base` takes a full SHA compared exactly, as `--head` does; the caller runs `git merge-base`, so the gate needs no checkout.
- The rule applies to every overlay in the run, and the base status must be exactly `fail`: a base `error` or `skip` cannot show red, matching how `verify` combines the runs. Spec contracts are not inspected, and each failing overlay contract gives one reason naming the contract and its overlay.
- No evidence field is added, so `evidence_version` stays `2` and v0.8.0 `--red-green` evidence already satisfies the check. The evidence is still self-reported: this makes a run without `--red-green` impossible to pass off as one, not proof the base run happened. ADR 0040 covers where trusted evidence comes from. A base that fails for an unrelated reason, such as a broken build, still counts as red; `base.failed_step` lets a reviewer see it.
- Combined with `--require-covered` (ADR 0043), only overlay contracts count toward coverage, and each covering one must have status `pass`.
