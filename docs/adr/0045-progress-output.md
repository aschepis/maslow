# ADR 0045: Live Progress Output on stderr

- **Status**: Accepted
- **Date**: 2026-11-20

## Context

`maslow verify` and `maslow audit` printed nothing until every check, contract, budget, ref and policy rule had run, so a slow check looked like a hang. The report on stdout and the evidence file must stay deterministic, and the verify, audit and runner layers do not write to the terminal.

## Decision

`verify.Options` and `audit.Options` take an optional `progress.Func` that receives a plan (the item count), then a start and a finish event per item; the CLI prints them to stderr under `--progress=auto|always|never`, where `auto` (the default) prints only when stderr is a character device.

## Consequences

- **What an item is.** For verify: each check, each contract (all its scenarios, retries and any red-green base run), each budget, each ref, and the policy rules as one. For audit: each target. A finish carries every result's status, so a contract line reads `contract cli 2 pass, 1 fail (0.5s)` and a retried one `flaky (2 attempts, 3.1s)`. Durations are wall time and appear only in progress, never in the evidence.
- **Determinism.** Progress goes to stderr only; stdout, the printed report and the evidence are byte-identical with or without it, apart from fields that already varied (timestamp, durations). CI logs stay quiet by default because stderr there is not a terminal; `--progress=always` turns it on.
- **Granularity.** Verify now runs contracts, budgets and refs one at a time through the existing runner entry points, which already treated each independently. A contract is one line rather than one per scenario because its scenarios share setup, teardown and retries. An invalid `--progress` value is a usage error (exit 2).
