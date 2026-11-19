# ADR 0039: Retries and the Flaky Status

- **Status**: Accepted
- **Date**: 2026-11-19

## Context

Every check and contract scenario ran exactly once, so a flaky failure looked like any other failure. Whoever acts on the evidence then fixes something the change did not break, and an automated loop with a fixed number of fix rounds spends one on noise. The evidence needs to separate "failed" from "failed, then passed".

## Decision

Checks and contracts take an optional `retries` (default 0, at most the spec's `max_retries`, which defaults to 5); a failing result is rerun until it passes, and one that fails and then passes gets status `flaky`, which counts as `pass` in the verdict for checks and spec contracts and as `needs_review` for overlay contracts.

## Consequences

- **Retry rule.** Only `fail` is retried; `error` is a configuration problem a rerun cannot fix, and a timeout is already `fail`. Attempts stop at the first `pass` or `error` or after `1 + retries`, with no delay between them, so the attempt count is the only variable. The final status is `pass` on a first-time pass, `flaky` on a later pass, `fail` when every attempt failed and `error` when the last errored. One function (`finalStatus` in `internal/runner/retry.go`) holds the rule.
- **Contracts.** A contract attempt is its setup, the scenarios still failing, and its teardown, so a retry reruns the contract setup with the rest of the attempt. A scenario that passes is not rerun. A setup that fails once and then passes makes the scenarios it starts `flaky`; one that fails every attempt fails every scenario, with each attempt recorded. Retries after the first take their own screenshot (`<scenario>.attempt-N.png`), so a retry never deletes the screenshot of the failure before it. Contract `retries` apply in `maslow audit` too, through the shared runner; `#AuditTarget.retries` still does nothing (ADR 0028).
- **Verdict.** A `flaky` check or spec contract counts as `pass`: with `needs_review` the same commit would pass on one run and stall a gate on the next, which is the noise this exists to remove, and the attempts stay visible in the evidence and in the "Flaky results" section of `evidence summary`. A `flaky` overlay contract makes the verdict `needs_review`, because an acceptance contract is the change's definition of done and is where a flaky fix would hide. `evidence check --require-red-green` declines an overlay contract whose `head_status` is `flaky` ("not green").
- **Red-green.** The base run uses the same contract and so retries with the same count. A base that fails and then passes shows the contract can pass without the change, so it is `not_red`. A genuinely red contract therefore runs `1 + retries` times on the base; that cost is accepted, since without base retries one unlucky base failure would count as red. A contract red on the base and flaky on the head is `flaky`.
- **Evidence.** `attempts` (status, duration, output and error for a check; status, error, `failed_step`, output and artifacts for a contract) is written only when more than one attempt ran, on check results, contract results and `base`. The other fields describe the last attempt, except a check's `duration_seconds`, which is the total. `--output-limit` caps each attempt's output. `evidence_version` stays `"2"`: `flaky` and `attempts` are additive and the verdict values are unchanged, as with `not_red` in 0.8.0 (ADR 0020).
- **Determinism.** With `retries: 0` nothing changes. A deterministic check gives the same evidence at any `retries`, since a pass passes first time and a failure fails every time; retries record nondeterminism in the system under test rather than add any. The verdict for a check or spec contract is the same whether or not the flake fired.
- **Spec protection.** Raising `retries` or `max_retries` loosens the gate the way a lowered threshold does, but `test_integrity` reads only threshold files, not the spec. Under `--spec-from base` the base spec's values apply and a spec edit needs review; without it, list the spec under `policy.gated`. An overlay's `retries` is held to the spec's `max_retries` at run time, and is covered by the overlay hash that `evidence check` trusts.
