---
id: 44
title: Let evidence check require that overlay contracts were proven red first
status: done
priority: high
created: 2026-11-18
updated: 2026-11-18
assigned_to: "claude-subagent"
assigned_at: "2026-11-19T02:33:37Z"
depends_on: [32, 33]
tags: [kind:gap, evidence, contracts, overlays]
---

## Objective

`maslow evidence check` (ADR 0033) cannot tell evidence from `verify --red-green` apart from evidence
from a plain `--overlay` run. Both pass the check if the verdict is `pass`, so a gate cannot require
that every acceptance contract failed on the merge base and passed on the head. An agent that leaves
off `--red-green` gets a hollow pass through the gate. ADR 0034 names this gap ("it does not yet
require that a run used `--red-green`").

## Requirements

- `evidence check` takes an opt-in flag that declines evidence unless every overlay contract carries
  a red-green result: base run on the trusted base SHA, base `fail`, head `pass`, combined `pass`.
- The trusted base is given as a full SHA and compared exactly, as `--head` is.
- Without the new flag, `evidence check` behaves exactly as today.
- Every failed requirement is a separate reason naming the contract and its overlay.

## Acceptance Criteria

- [x] `--require-red-green` and `--base <sha>` on `maslow evidence check`, with usage errors (exit 2)
      for bad combinations.
- [x] Table tests in `internal/evidence/check_test.go` cover every case listed below.
- [x] CLI tests in `cmd/maslow/evidence_test.go`: a real `--red-green` evidence file passes, the same
      run without `--red-green` declines.
- [x] `maslow.yaml` gets CLI contracts for the new flag (one pass, one decline).
- [x] ADR written; README, MAP.md and the `main.go` usage text updated.

## Notes

The evidence file is self-reported. This check makes a run without `--red-green` impossible to pass
off as one; it does not prove the base run happened. Task 35 (evidence from a separate job) addresses
trust in the producer.

## Proposed implementation

### 1. What happens today

There is no top-level red-green marker. The only trace of a red-green run is on each overlay
contract result:

- `ContractResult.HeadStatus` and `ContractResult.Base` (`internal/evidence/report.go:87-90`) are
  set only by `judgeRedGreen` (`internal/verify/redgreen.go:57-75`). `Base` is
  `{sha, status, error, failed_step}` (`report.go:97-103`).
- `redGreenStatus` (`redgreen.go:79-90`) sets the combined `status`: `pass` only when head is `pass`
  and base is `fail`; `not_red` when both pass; `error` when the base errored or could not be
  checked out; otherwise the head's own status.
- `ContractResult.Overlay` (`report.go:79`) is the overlay path, and `Report.Overlays`
  (`report.go:36`) lists `{path, sha256}` for each overlay (`verify.go:126-135`).

The top-level `base_sha` (`report.go:33`) is not a marker. `verify.Run` copies `opts.BaseSHA` into
it whenever `--base` is passed (`verify.go:90`), with or without `--red-green`, because `--base`
also drives policy (`cmd/maslow/main.go:116,147-154`). So a plain `--base --overlay` run has
`base_sha` set and no per-contract `base`, and a run without `--base` has neither.

The base SHA can be checked against a trusted value. `--base <ref>` is resolved to the full merge-base
SHA (`verify.go:69-76`, `main.go:148`), and `checkoutBase` stamps that same SHA on every contract's
`base.sha` (`redgreen.go:26,63`), so `base_sha` and every `base.sha` are equal full SHAs.

`Check` (`internal/evidence/check.go:63-96`) compares version, `git_sha`, `spec_sha256`, overlay hashes
and profile, then trusts `verdict`. It never reads `Contracts`. A `not_red` or base `error` result
declines only because it makes the verdict `fail` (`report.go:212`, covered by
`TestCheck_DeclinesNotRedContract`, `check_test.go:182-196`). A plain overlay run whose contracts
pass on the head has verdict `pass` and passes the check. `Trusted` (`check.go:19-24`) has no base
field, and the CLI (`cmd/maslow/evidence.go:22-29`) has no base flag.

### 2. Proposed change

**`internal/evidence/check.go`**

- Add `Base string` and `RequireRedGreen bool` to `Trusted`.
- If `t.Base != ""`, decline when `r.BaseSHA != t.Base` (field `base_sha`).
- If `t.RequireRedGreen`, call a new `redGreenMismatches(r, t.Base)`, and add each result under the
  field `contracts`. It checks:
  - each entry in `r.Overlays` has at least one contract with that `overlay` path. Overlays cannot be
    empty (`internal/spec/overlay.go:51`), so a missing result means the evidence was edited;
  - every contract with `Overlay != ""` has `Base != nil`, `Base.SHA == t.Base`,
    `Base.Status == "fail"`, `HeadStatus == "pass"` and `Status == "pass"`. Each failure is one reason,
    such as `overlay contract "x" (overlay acc.yaml) has no base run; was verify run with --red-green?`
    or `... base status "pass" on <sha>: not red`.
  - `r.Overlays` is empty: one reason, `--require-red-green but the run used no overlays`.
- The required base status is exactly `fail`, not "anything except pass". This matches
  `redGreenStatus`, where a base `error` or `skip` cannot show red.
- Spec contracts (no `overlay`) are not inspected.
- The rule order stays the same: any reason means decline, and `needs_review` passes through only when
  there are no reasons.

**`cmd/maslow/evidence.go`**

- Add `--require-red-green` (bool) and `--base <sha>` (string), and pass them into `Trusted`.
- Usage errors (exit 2): `--require-red-green` without `--base`; `--require-red-green` with no
  `--overlay`/`--overlay-sha256`. Both are mistakes in the gate's own config, following ADR 0033's
  split between decline and usage error.
- `--base` without `--require-red-green` is allowed and checks only `base_sha` (see open question 2).
- Update the usage line at `evidence.go:15`.

**`cmd/maslow/main.go`**: add the two flags to the help text (`main.go:61-67`).

**`internal/verify`, `report.go`, `schema/evidence.schema.json`**: no change. A top-level
`red_green: {base_sha}` would also be self-reported, and check would still have to inspect every
contract to catch a partial or edited run. It would repeat what `base_sha` plus the per-contract `base`
already say. Without a schema change, `evidence_version` stays `2` and evidence written by v0.8.0
`--red-green` runs already satisfies the new check.

**Docs**: README evidence-check paragraph, `docs/MAP.md` evidence section, CHANGELOG entry under the
next release.

**`maslow.yaml`**: add CLI contracts next to the existing `evidence check` ones (around lines 418-434)
using fixture evidence files: red-green evidence passes with `--require-red-green --base`, and plain
overlay evidence declines with a reason containing `--red-green`.

### 3. Tests

`internal/evidence/check_test.go`, extending the `TestCheck` table (`goodReport` gets `BaseSHA` and one
overlay with one red-green contract when the case needs it):

| Case | Trusted | Evidence | Decision | Reason field |
|---|---|---|---|---|
| red-green satisfied | require, base B | base_sha B; contract base {B, fail}, head pass, status pass | pass | none |
| flag off, plain overlay run | no require | contract with no `base` | pass | none (unchanged) |
| plain overlay run | require, base B | contract with no `base`, status pass | decline | contracts |
| base SHA differs on contract | require, base B | contract base.sha C | decline | contracts |
| top-level base_sha differs | base B | base_sha C | decline | base_sha |
| base_sha missing | base B | base_sha "" | decline | base_sha |
| not_red | require, base B | base pass, head pass, status not_red | decline | verdict, contracts |
| base error | require, base B | base error, status error | decline | verdict, contracts |
| head fails | require, base B | base fail, head fail, status fail | decline | verdict, contracts |
| inconsistent fields | require, base B | base pass, head pass, status pass (edited) | decline | contracts |
| one of two contracts lacks base | require, base B | one good, one with no base | decline | contracts (one reason) |
| overlay with no contract results | require, base B | overlays [o], contracts [] | decline | contracts |
| no overlays in run | require, base B, trusted overlay o | overlays [] | decline | overlays, contracts |
| spec contract has no base | require, base B | spec contract without `base` plus a good overlay contract | pass | none |
| needs_review with red-green satisfied | require, base B | verdict needs_review | needs_review | verdict |

`cmd/maslow/evidence_test.go`:
- usage errors: `--require-red-green` without `--base` and without any overlay flag both exit 2.
- end to end: build a temp repo the way `cmd/maslow/red_green_test.go` does, run
  `verify --red-green --base`, and check the result with `--require-red-green --base <merge-base>`
  (exit 0). Then run `verify --overlay` without `--red-green` and check it again (exit 1, reason
  mentions `--red-green`).

### 4. ADR

Write a new ADR (0037; task 43 reserves 0036), "Evidence check can require red-green", rather than
amending ADR 0033. It adds a rule beyond strict equality with trusted inputs, and ADR 0033's decision
sentence would no longer describe the command. Change the last sentence of ADR 0034's consequences
from "it does not yet require" to point at the new ADR. That is the only edit to an accepted ADR.

### 5. Risks and open questions

1. **Should `--require-red-green` require `--base`?** If it does not, base SHAs would only be checked
   against each other, and a run red on an arbitrary old commit (or one where the base build breaks)
   would pass. *Recommendation:* require it. The gate already knows the merge base, as it knows the
   head.
2. **Should `--base` alone check `base_sha`?** This is useful for policy runs: it confirms policy was
   judged against the right base. It does widen the change slightly. *Recommendation:* yes. It is one
   comparison, and it gives `--base` the same meaning in both modes.
3. **Should base `error` or `skip` count as red?** ADR 0034 says no, and verify already fails the
   verdict. *Recommendation:* require exactly `fail`, so check and verify agree.
4. **Should the gate accept a ref for `--base` and resolve the merge base itself?** That needs git and
   a checkout in the gate, which `--head` avoids. *Recommendation:* full SHA only. The caller runs
   `git merge-base`.
5. **Should the requirement be per overlay?** A gate might want red-green for the acceptance overlay
   but not for a regression overlay in the same run. *Recommendation:* all overlays for now. Add
   `--require-red-green-overlay <sha256>` later if someone needs it.
6. **Should a top-level `red_green` field be added to evidence?** It makes evidence easier to read
   but adds no assurance. *Recommendation:* no, for the reasons in section 2. Revisit if
   `evidence summary` (task 39) needs it.
7. **Risk: a base failure for an unrelated reason counts as red.** For example, the base fails to
   build. This is ADR 0034's known weakness; `base.failed_step` lets a reviewer see it. Out of scope
   here, so it is noted in the ADR's consequences.
