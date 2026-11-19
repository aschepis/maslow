---
id: 38
title: Rerun failed checks and contracts and report flakes distinctly
status: in_progress
priority: medium
created: 2026-11-18
updated: 2026-11-19
assigned_to: "claude-subagent"
assigned_at: "2026-11-19T16:44:46Z"
depends_on: []
tags: [kind:gap, checks, contracts, evidence]
---

## Objective

A flaky failure sends whoever acts on the evidence to fix something the change did not break. An
automated loop with a fixed number of fix rounds wastes one on it.

## Requirements

- Checks and contracts accept an optional `retries` count (default 0).
- Every attempt is recorded in the evidence with its status, duration and output.
- A result that failed then passed gets a distinct `flaky` status; the ADR decides whether `flaky`
  makes the verdict `pass` or `needs_review`.
- A result that fails every attempt is `fail` as today.

## Acceptance Criteria

- [ ] Schema and fixtures for `retries`.
- [ ] Tests: pass first time, fail then pass (`flaky`), fail every time.
- [ ] Evidence schema updated; ADR records the verdict rule.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V7.

## Decisions (human review, 2026-11-19)

These override the proposal below where they differ.

1. Verdict: a `flaky` check or spec contract counts as `pass`. A `flaky` overlay contract counts as
   `needs_review`, because acceptance contracts are where a flaky fix would hide.
2. `evidence check --require-red-green` declines an overlay contract that was flaky on the head.
3. `retries` defaults to 0 per check and contract. The upper bound is a spec-level setting that
   defaults to 5, not a fixed cap; a `retries` above it is a validation error.
4. Contract setup is retried with the rest of the attempt.

## Proposed implementation

### 1. What happens today

Every check and every contract scenario runs exactly once.

- `runCheck` (`internal/runner/runner.go:55-101`) runs the command once. A non-zero exit or a timeout
  gives `fail` (`runner.go:83-92`); an empty command gives `error` (`runner.go:69-77`). There is no
  `retries` field on `#CheckRunner` (`schema/maslow.cue:145-151`) or `spec.CheckRunner`
  (`internal/spec/spec.go:99-105`).
- `runScenario` (`internal/runner/contract.go:207-225`) runs setup, steps and teardown once, in a
  fresh scope each call (`contract.go:209`). Contract-level setup runs once, and a failure fails every
  scenario through `failAll` (`contract.go:171-183,197-205`). `#Contract` (`maslow.cue:164-175`) and
  `spec.Contract` (`spec.go:118-129`) have no `retries`. `withScreenshot` deletes the previous file at
  the scenario's fixed screenshot path (`contract.go:120-126`, `browser.go:358-361`).
- The only `retries` in the schema is on `#AuditTarget` (`maslow.cue:345`), and it does nothing
  (ADR 0028, line 20). Audit runs contracts through the same `RunContractsWith`
  (`internal/audit/audit.go:180`).
- `ComputeVerdict` (`internal/evidence/report.go:208-230`) fails on `fail`, `error` or `not_red`,
  counts `pass` as "ran", and silently ignores any other status. A new `flaky` status added without
  touching it would be invisible: a run whose only results were `flaky` would be `inconclusive`.
- Red-green (`internal/verify/redgreen.go:57-90`) runs each overlay contract on the base with the same
  runner and pairs results by position (`redgreen.go:68`). `redGreenStatus` passes only when head is
  `pass` and base is `fail`; base `pass` gives `not_red`; any other base status gives `error`; any
  non-`pass` head keeps its own status.
- `evidence check` (`internal/evidence/check.go:66-107`) requires `evidence_version` to equal `"2"`
  (`check.go:72-74`) and trusts the verdict (`check.go:91-93`). It reads per-contract statuses only
  under `--require-red-green`, where `redGreenProblem` (`check.go:142-156`) needs base exactly `fail`,
  `head_status` exactly `pass` and `status` exactly `pass`.
- `evidence summary` lists enforced results whose status is `fail`, `error` or `not_red`
  (`internal/evidence/summary.go:35-68,113-116`). Any other status is not shown.
- The evidence schema has one `status` enum, `pass|fail|skip|error`, shared by check status,
  `head_status`, `base.status`, budgets and refs (`schema/evidence.schema.json:54,63,85,96,121,134`);
  contracts add `not_red` (`evidence.schema.json:77`). No `attempts` field exists, and the schema
  rejects unknown fields.
- `test_integrity` (`internal/runner/policy_lines.go:218-266`) reads diffs, not results. Raising a
  number in a `thresholds` file is not flagged; only lowering is (ADR 0032).

### 2. Proposed change

**Spec.** Add `retries?: int & >=0 & <=5` to `#CheckRunner` and `#Contract` in `schema/maslow.cue`,
copy it to `internal/schema/maslow.cue`, and add `Retries int` to `spec.CheckRunner` and
`spec.Contract`. On a contract it applies to each scenario. Default 0 keeps today's behaviour.

**Retry rule.** Only `fail` is retried. `error` (a config problem such as an empty command) is not.
Attempts stop at the first `pass` or `error`, or after `1 + retries` attempts. There is no delay
between attempts, so the attempt count is the only variable. The final status is:

- `pass` when the first attempt passed;
- `flaky` when a later attempt passed;
- `fail` when every attempt failed;
- `error` when the last attempt errored.

One pure function, `foldAttempts`, computes this for checks and contracts, so the rule lives in one
place.

**Evidence (`internal/evidence/report.go`, `schema/evidence.schema.json`).**

- Add `const StatusFlaky = "flaky"`.
- Add `Attempts []CheckAttempt` to `CheckResult` and `Attempts []ContractAttempt` to
  `ContractResult` and `BaseRun`. A check attempt holds status, duration, output (with
  `output_bytes`/`output_truncated`) and error. A contract attempt holds status, error, `failed_step`,
  output and artifacts.
- `attempts` is written only when more than one attempt ran. With `retries: 0`, or a first-time pass,
  the evidence is byte-for-byte the shape it is today.
- The top-level fields describe the last attempt, except `duration_seconds`, which is the total time
  of all attempts. For a `flaky` result the failure details are in `attempts`.
- `CapOutput` (`report.go:173-185`) caps every attempt's output as well.
- Schema: a new `$defs/resultStatus` adds `flaky` to the four statuses. Check `status`, `head_status`
  and `base.status` use it, and contract `status` gains `flaky`. Budgets and refs keep `status`, since
  they never retry. Add `$defs/checkAttempt` and `$defs/contractAttempt`.

**Verdict.** `ComputeVerdict` treats `flaky` as `pass` (it sets `ran`). The verdict enum does not
change. ADR 0039 records why (open question 1).

**Runner.** `RunChecks` calls `runCheck` in an attempt loop and folds the results. `runScenario`
runs in an attempt loop; each attempt already gets a fresh scope, setup and teardown. Attempts after
the first get their own screenshot path (`<scenario>.attempt-2.png`), so a retry does not delete the
evidence of the failure before it. A failing contract-level setup is retried the same way (setup,
then teardown) before `failAll`; its attempts are recorded on every scenario it fails.

**Red-green (`internal/verify/redgreen.go`).** No change to `judgeRedGreen` itself: the base run uses
the same contract spec, so it retries with the same count, and it still yields one result per
scenario, so positional pairing holds. `BaseRun` copies the base result's `Attempts`.
`redGreenStatus` becomes:

- head is neither `pass` nor `flaky`: keep head's status;
- base is `pass` or `flaky`: `not_red`;
- base is `fail`: head's status (`pass` or `flaky`);
- otherwise: `error`.

A base that fails and then passes has shown that the contract can pass without the change, so it is
not red. Retrying the base is what makes this visible. Without base retries, one unlucky base failure
would count as red. A contract that is red on the base and flaky on the head gets `flaky`.

**`evidence check`.** No code change. The verdict rule already covers `flaky` once `ComputeVerdict`
counts it as `pass`. Under `--require-red-green`, `redGreenProblem` already declines a base of `flaky`
(it needs exactly `fail`) and a `head_status` of `flaky` (it needs exactly `pass`). Tests pin both
cases (open question 2).

**`evidence summary` (`summary.go`).** `isBad` is unchanged. Add a `### Flaky results` section after
the enforced table, listing each flaky check, contract or base run with its attempt count and the
first failed attempt's error. It is omitted when nothing was flaky, the way the observe section is.

**`evidence_version`.** Stays `"2"`. Following ADR 0020 and the 0.8.0 precedent, a new status and a
new optional field are additive, and the verdict values are unchanged, so a consumer that reads only
`verdict` (including every `evidence check` gate) is unaffected. The CHANGELOG gets an "Evidence
format" note: `flaky` is a new status on checks, contracts, `head_status` and `base.status`, and
`attempts` is a new optional field. A consumer that rejects statuses it does not know must add
`flaky`.

**Determinism.** Maslow stays deterministic. With `retries: 0` nothing changes. A deterministic check
gives the same evidence with any `retries` value: a pass passes on the first attempt, and a failure
fails every attempt. Retries record nondeterminism in the system under test rather than add any. Two
choices keep it that way: no delay or backoff, and a verdict that is the same whether or not the flake
fired (`pass` and `flaky` both give `pass`). The per-result status and `attempts` differ between such
runs, and that is the point. Durations and timestamps already differ today.

**`test_integrity`.** No change. Raising `retries` loosens the gate in the way a lowered threshold
does, but it lives in the spec. Under `--spec-from-base` the base spec's `retries` apply, and a spec
edit is reported `needs_review` (`internal/runner/policy.go:40-41,170-197`). The ADR says so, and it
suggests listing the spec under `policy.gated` for runs without `--spec-from-base`. An overlay's
`retries` is covered by the overlay hash that `evidence check` trusts.

**Docs and spec.** README (the `retries` field, the `flaky` status, the summary section),
`docs/MAP.md`, and CHANGELOG. In `maslow.yaml`, add a CLI contract that runs `maslow verify` on a
fixture spec whose check fails on its first run and passes after. The contract asserts exit 0,
`"status": "flaky"` and `"attempts"`. Its setup step removes the marker file under `reports/`.

### 3. Tests and checks

First add the `Retries` fields and the empty `Attempts` types with no behaviour, so the new tests
compile and fail on their assertions rather than on the build. Then write the tests below, and see
them fail. The flaky fixtures are scripts that fail when a marker file is absent, create it, and pass
when it is present. They are deterministic, and on red-green the base worktree and the head directory
have separate markers.

`internal/runner/runner_test.go`:
- `retries: 2`, passes first time: `pass`, no `attempts`, the script ran once.
- `retries: 2`, fails then passes: `flaky`, attempts `[fail, pass]`, total duration.
- `retries: 2`, always fails: `fail`, three attempts, each with its output.
- `retries: 0`, fails: `fail`, no `attempts` (unchanged).
- empty command, `retries: 2`: `error`, one run, no `attempts`.

`internal/runner/contract_test.go`: the same three main cases for a CLI scenario, plus: a flaky attempt
records its `failed_step`; a failing contract setup is retried and then fails every scenario with
attempts; each scenario's own setup and teardown run once per attempt.

`internal/evidence/report_test.go`: `ComputeVerdict` gives `pass` for only `flaky`, `fail` for `flaky`
plus `fail`, and `needs_review` for `flaky` plus `needs_review`. `CapOutput` truncates attempt output.

`internal/verify/redgreen_test.go`: a table test of `redGreenStatus` covering every head and base pair,
including `flaky`. Three temp-repo tests: base flaky, head pass gives `not_red` with `base.attempts` of
two; base fail, head flaky gives `flaky` with `head_status` `flaky`; base fail on every attempt, head
pass gives `pass` with `1 + retries` base attempts.

`internal/evidence/check_test.go`, extending the `TestCheck` table: verdict `pass` with flaky results
passes; `--require-red-green` with `head_status` `flaky` declines with "not green"; base `flaky` with
status `not_red` declines on both `verdict` and `contracts`.

`internal/evidence/summary_test.go`: new golden files `testdata/evidence/flaky.json` and
`flaky.summary.md`, with "Nothing failed." followed by the flaky section. The existing goldens stay
unchanged.

`internal/verify/evidence_schema_test.go`: real evidence holding a `flaky` check, a flaky contract
with attempts, and a red-green base with attempts validates against the schema.

Fixtures: `testdata/valid/retries.yaml`; `testdata/invalid/retries_negative.yaml`,
`retries_too_many.yaml` and `retries_string.yaml`.

Run `go test ./...`, `go vet ./...`, `cue vet schema/maslow.cue` and
`maslow verify --profile quick`.

### 4. ADR

Write ADR 0039, "Retries and the flaky status". Decision: checks and contracts take `retries` (0 to
5); a result that fails and then passes is `flaky`, which counts as `pass` in the verdict and is listed
in the summary. A flaky base run is not red. The consequences cover the determinism argument, the
unchanged `evidence_version`, the cost of retrying a genuinely red base, and the spec-protection note
from section 2. Add one sentence to ADR 0034's consequences pointing to ADR 0039 for flaky base runs,
and one to ADR 0028 saying contract `retries` now apply in audit while `#AuditTarget.retries` still
does nothing. Those are the only edits to accepted ADRs.

### 5. Risks and open questions

1. **Should `flaky` make the verdict `pass` or `needs_review`?** With `needs_review`, the same commit
   gives `pass` on one run and `needs_review` on the next, and `evidence check` exits 1 on
   `needs_review`, so a gate stalls on noise the change did not cause. That is the problem this task
   exists to fix. The cost of `pass` is that a change that introduces a flake passes, visibly.
   *Recommendation:* `pass`. The spec owner chose the retry count, the attempts are in the evidence,
   and the summary lists them. If a gate later wants to refuse flakes, add an opt-in
   `evidence check --decline-flaky`.
2. **Should `--require-red-green` accept a flaky head?** *Recommendation:* no, which keeps today's
   code. An acceptance contract is the change's definition of done, and a flaky one proves less. The
   reason message already says "not green".
3. **Should base runs retry?** They must, or a flaky base failure counts as red. A genuinely red
   contract then runs `1 + retries` times on the base. *Recommendation:* accept the cost, and say so
   in the ADR.
4. **Should `error` be retried?** *Recommendation:* no. In this runner an error means bad config,
   which a retry cannot fix; timeouts are already `fail` and are retried.
5. **What should the cap be?** *Recommendation:* 5 in the CUE schema. Raising it later is
   non-breaking; lowering it would break specs.
6. **Should contract-level setup be retried?** It is shared by every scenario, and it is where
   servers usually start. *Recommendation:* yes, with the same count, recorded on each scenario it
   fails.
7. **What should `#AuditTarget.retries` mean?** *Recommendation:* leave it unimplemented in this task.
   Contract `retries` already apply in audit through the shared runner.
8. **Should there be a CLI or profile override such as `--retries`?** *Recommendation:* no. The count
   belongs in the spec, where `--spec-from-base` protects it.
9. **Risk: evidence grows.** Each failed attempt keeps up to 4 KB of output. With a cap of 5 that is
   at most 24 KB per result. *Recommendation:* accept it; `--output-limit` already bounds each
   attempt.
10. **Risk: retried browser scenarios.** A retry reuses screenshot and artifact patterns.
    *Recommendation:* attempt-suffixed screenshot paths as in section 2, and attach artifacts per
    attempt before teardown, as `runScenario` already does for one attempt.
