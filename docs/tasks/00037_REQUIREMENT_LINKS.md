---
id: 37
title: Link checks and contracts to the requirements they cover
status: in_progress
priority: medium
created: 2026-11-18
updated: 2026-11-19
assigned_to: "claude-subagent"
assigned_at: "2026-11-20T01:02:47Z"
depends_on: []
tags: [kind:gap, schema, evidence]
---

## Objective

A consumer cannot answer two basic questions from a spec or its evidence: which requirements have
no contract, and which contract proves issue N is fixed.

## Requirements

- Checks, contracts and scenarios accept a `covers` list of free-form references (issue numbers,
  requirement IDs, URLs).
- The evidence lists, per result, what it covers, and a top-level summary of every reference
  exercised and its outcome.
- Overlay contracts may carry `covers` like any other contract.

## Acceptance Criteria

- [ ] Schema and fixtures for `covers`.
- [ ] Tests: covered references appear in the evidence with the right status.
- [ ] Evidence schema updated; ADR records the reference format.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V6. Whether Maslow should also
report uncovered requirements depends on having a requirement list to compare against; leave that
out unless a clear source exists.

## Decisions (human review, 2026-11-19)

All recommendations in the proposal below are accepted: the gate supplies the requirement list
(`--require-covered`), IDs match exactly, only red-green overlay contracts count under
`--require-red-green`, every covering result must pass, no file form of the flag yet, no `covers`
on budgets, and the derived top-level `coverage` field stays, documented as derived.

## Proposed implementation

### 1. What happens today

Nothing in the spec or the evidence links a result to a requirement.

- `#CheckRunner` (`schema/maslow.cue:148-156`), `#Contract` (`maslow.cue:171-184`) and `#Scenario`
  (`maslow.cue:186-193`) have no `covers` field, and the CUE definitions are closed, so a `covers`
  key fails `maslow validate` today. The Go types match: `CheckRunner` (`internal/spec/spec.go:112-119`),
  `Contract` (`spec.go:132-145`), `Scenario` (`spec.go:147-154`).
- Overlays reuse the same contract type: `#Overlay` is `description` plus `[...#Contract]`
  (`maslow.cue:50-53`), validated by `schema.ValidateOverlay` (`internal/schema/validate.go:39-40`) and
  parsed into `spec.Overlay` (`internal/spec/overlay.go:16-19`). Anything added to `#Contract` is
  therefore accepted in overlays with no further work.
- The closest existing field is `tags` (`maslow.cue:155,174`). Tags only select checks, contracts and
  budgets for a profile (`internal/runner/profile.go:13-19`) and never reach the evidence.
- Refs (`maslow.cue:75-91`, ADRs 0011 and 0012) are files, URLs and binaries that must exist. They are
  verified one by one and reported as `RefResult` (`internal/evidence/report.go:158-163`). Nothing
  points from a contract to a ref, and a ref's `path` is not an identifier for a requirement, so refs do
  not overlap with this task beyond the convention that `docs/` refs are what an agent reads first.
- There is no declared requirement list anywhere: not in the spec, not in overlays, and not in
  `docs/REQUIREMENTS.md`, whose sections are prose headings rather than IDs.

Results are produced like this:

- Check results come from `runCheckWithRetries` (`internal/runner/runner.go:26-36`). `CheckResult`
  (`report.go:61-74`) has no link field.
- Contract results are one per scenario, named `contract/scenario` (`internal/runner/contract.go:240`).
  `runContract` returns them in scenario order (`contract.go:195-199`) and `RunContractsWith` appends
  them per contract (`contract.go:151-160`). `ContractResult` (`report.go:101-121`) has no link field.
- `verify.Run` runs spec contracts, then each overlay's, and stamps `Overlay` on the latter
  (`internal/verify/verify.go:121-139`). Under `--red-green`, `judgeRedGreen` edits the head results in
  place (`internal/verify/redgreen.go:62-74`), so any field set earlier survives.
- `audit` runs contracts through the same `RunContractsWith`, so it would carry the field too.

On the consuming side:

- `evidence.schema.json` sets `additionalProperties: false` on `check` (`:57-76`) and `contract`
  (`:90-112`) and on the top level (`:7`). `TestEvidenceSchemaRejectsUnknownFields`
  (`internal/verify/evidence_schema_test.go:140`) enforces this.
- `evidence check` (`internal/evidence/check.go:66-107`, flags at `cmd/maslow/evidence.go:39-70`)
  compares trusted inputs and then trusts the verdict. Its one rule that reads individual results is
  `--require-red-green` (`check.go:94-98,111-156`, ADR 0037).
- `evidence summary` (`internal/evidence/summary.go:11-19`, ADR 0036) prints the enforced failures,
  then the flaky results, then the observed policy findings. It has no coverage section.

### 2. Proposed change

**Answers to the four questions**

- *Shape of a requirement ID:* a free-form, non-empty string with no whitespace, compared byte for
  byte and case-sensitively. Maslow does not parse it. The ADR recommends `owner/repo#N` for issues,
  a URL for anything with a URL, and a project prefix such as `REQ-12` otherwise, but none of this is
  enforced. `#12`, `12` and `aschepis/maslow#12` are three different IDs.
- *Where requirements are declared:* links are declared in the spec and overlays, as `covers` on
  checks, contracts and scenarios. The list of requirements that must be covered is not declared in
  the spec. It is a trusted input to `evidence check`, given by the gate, in the same way as the
  trusted overlays. The gate (the Damocles reconciler) already holds the issue's acceptance criteria;
  a list in the spec or overlay would be one more file an agent's run reports about itself.
- *How `evidence check` requires coverage:* with a new repeatable flag `--require-covered <id>`.
- *How an uncovered requirement is reported:* by `evidence check`, as one decline reason per ID under
  the field `covers`. `verify` cannot report it, because it has no list. The evidence's top-level
  `coverage` lists only IDs that something covered.

**Schema (`schema/maslow.cue`, copied to `internal/schema/maslow.cue`)**

- Add `#Covers: [...string & =~"^\\S+$"]`.
- Add `covers?: #Covers` to `#CheckRunner`, `#Contract` and `#Scenario`. Overlays inherit it through
  `#Contract`.
- Budgets get nothing for now (see open question 6).

**Spec types (`internal/spec/spec.go`)**: add `Covers []string \`yaml:"covers,omitempty"\`` to
`CheckRunner`, `Contract` and `Scenario`.

**Runner**

- `runCheckWithRetries` sets `result.Covers` from the check, sorted and deduplicated.
- `RunContractsWith` sets each scenario result's `Covers` to the union of the contract's and the
  scenario's lists, sorted and deduplicated. The index of each result matches `contract.Scenarios`
  (`contract.go:195-199`). Setting it here, and not in `runScenario`, keeps `failAll` and the retry fold
  untouched.
- `judgeRedGreen` needs no change. It runs the base through the same `RunContractsWith` and copies only
  the status fields across.

**Evidence (`internal/evidence/report.go`, `schema/evidence.schema.json`)**

- Add `Covers []string \`json:"covers,omitempty"\`` to `CheckResult` and `ContractResult`.
- Add a top-level `Coverage []CoverageEntry \`json:"coverage,omitempty"\``, where each entry is
  `{id, status, results}`, sorted by `id`. `results` names each covering result as `check:<name>` or
  `contract:<name>`, which keeps checks and contracts with the same name apart. `status` uses the
  verdict's precedence over the covering results: `fail` if any is `fail`, `error` or `not_red`;
  `needs_review` if any is a flaky overlay contract (`contractVerdictStatus`, `report.go:304-309`);
  `pass` if any passed or was flaky; `skip` otherwise.
- `ComputeCoverage()` builds it from the per-result `covers`. `verify.Run` calls it beside
  `ComputeVerdict` (`verify.go:161`), and so does `audit`.
- `covers` never changes the verdict.
- `evidence_version` stays `2`. The fields are optional and additive, following ADR 0034's precedent.
  The JSON schema gets `covers` on `check` and `contract` and `coverage` at the top level.

**`evidence check` (`internal/evidence/check.go`, `cmd/maslow/evidence.go`)**

- Add `RequireCovered []string` to `Trusted`, and the repeatable flag `--require-covered <id>`. An
  empty value or one containing whitespace is a usage error (exit 2).
- New `coverageMismatches(r, ids, requireRedGreen)` recomputes coverage from the per-result `covers`
  and ignores the top-level `coverage`, because that field is a convenience and the evidence is
  self-reported (as with red-green, ADR 0037). For each ID it adds one reason under the field `covers`:
  - none: `requirement "X" is not covered by any check or contract in the run`;
  - any covering result is not `pass` or `flaky`: `requirement "X" is covered by contract "c/s"
    (overlay o.yaml) with status "fail"`. This names the first bad result only, so one ID gives one
    reason.
- When `--require-red-green` is also set, only overlay contracts count toward coverage, and the
  existing rule already demands that each of them was red then green. A requirement covered only by a
  spec check gives `requirement "X" is covered by no overlay contract; --require-red-green needs one`.
  See open question 3.
- The rule order is unchanged: any reason declines, and `needs_review` passes through only with no
  reasons.

**`evidence summary`**: add a "Requirements" section after the enforced table, listing ID, status and
covering results. It is omitted when `coverage` is empty, so the existing golden files do not change.

**`maslow.yaml`**: add CLI contracts next to `evidence-check-command` (`maslow.yaml:513`): fixture
evidence with covers passes `--require-covered`, and the same evidence declines for an ID nothing
covers. Those contracts carry `covers: [docs/tasks/00037_REQUIREMENT_LINKS.md]`, so the self-hosting
spec uses the feature.

**Docs**: README sections on the spec and on evidence check, the `docs/MAP.md` evidence section,
`main.go` usage text, `evidence.go:20` usage line, and a CHANGELOG entry.

### 3. Tests and checks

Each test below is written and run before its code. For new Go struct fields, the first red is a
compile failure. Schema fixtures and CLI tests fail on their assertions.

- **Schema fixtures** (`TestValidate_Fixtures`, `internal/schema/validate_test.go:242`):
  `testdata/valid/covers.yaml` has covers on a check, a contract and a scenario.
  `testdata/invalid/covers_empty_string.yaml`, `covers_whitespace.yaml` and `covers_not_list.yaml` must
  fail validation. Today the valid one fails, because `covers` is an unknown key.
- **Overlay**: `testdata/overlays/with-covers.yaml` loads through `ReadOverlay`
  (`internal/spec/overlay_test.go:30`) with `Covers` populated.
- **Runner** (`internal/runner/contract_evidence_test.go`, `runner_test.go`): a scenario result
  carries the sorted union of contract and scenario covers, with no duplicates. A check result
  carries the check's covers. A contract whose setup fails still carries covers on every scenario
  through `failAll`. A retried scenario keeps its covers.
- **Red-green** (`internal/verify/redgreen_test.go`): an overlay contract with covers keeps them after
  `judgeRedGreen`.
- **Coverage** (`internal/evidence/report_test.go`): a table over `ComputeCoverage`. One ID covered by
  a pass and a fail gives `fail`. A flaky overlay contract gives `needs_review`. A flaky spec contract
  gives `pass`. `not_red` gives `fail`. Skip only gives `skip`. No covers gives no `coverage` key. The
  output is sorted, so two runs produce identical JSON.
- **Evidence schema** (`TestEvidenceMatchesSchema`, `evidence_schema_test.go:23`): add covers to the
  test spec's check and contract and assert the evidence still validates.
- **Check** (`internal/evidence/check_test.go`), extending the `TestCheck` table:

| Case | Trusted | Evidence | Decision | Reason field |
|---|---|---|---|---|
| no flag | none | covers present | pass | none (unchanged) |
| covered by passing contract | require X | contract covers X, pass | pass | none |
| covered by passing check | require X | check covers X, pass | pass | none |
| uncovered | require X | nothing covers X | decline | covers |
| two IDs, one uncovered | require X, Y | only X covered | decline | covers (one reason) |
| covering result fails | require X | contract covers X, fail | decline | verdict, covers |
| one of two covering fails | require X | pass and fail both cover X | decline | verdict, covers |
| top-level coverage edited | require X | `coverage` says pass, per-result covers empty | decline | covers |
| case differs | require `REQ-1` | covers `req-1` | decline | covers |
| red-green, overlay covers | require X, red-green, base B | red-green overlay contract covers X | pass | none |
| red-green, only spec check covers | require X, red-green, base B | check covers X; overlay covers nothing | decline | covers |
| red-green, covering overlay is not_red | require X, red-green, base B | overlay covers X, not_red | decline | verdict, contracts, covers |
| flaky overlay covers | require X | overlay contract flaky, verdict needs_review | needs_review | verdict |

- **CLI** (`cmd/maslow/evidence_test.go`): `--require-covered ""` exits 2. End to end: run `verify`
  with an overlay whose contract covers `acme/app#7`, then `evidence check --require-covered
  acme/app#7` exits 0 and `--require-covered acme/app#8` exits 1 with the reason naming `acme/app#8`.
- **Summary**: a new golden, `testdata/evidence/covers.json` and `covers.summary.md`. The existing
  goldens must not change.
- Run `go test ./...`, `go vet ./...`, `cue vet schema/maslow.cue`, and
  `maslow verify --profile quick`.

### 4. ADR

Write ADR 0043, "Requirement links with `covers`". It records: the ID is an opaque string with no
whitespace, compared exactly; links are declared on checks, contracts and scenarios; the requirement
list is a trusted input to `evidence check` and is not kept in the spec; `coverage` in the evidence is
derived, and the check ignores it; `covers` never affects the verdict. Amend ADR 0037's consequences
in one sentence, to say that `--require-covered` combined with `--require-red-green` counts only
overlay contracts.

### 5. Risks and open questions

1. **Should Maslow hold a requirement list (in the spec or the overlay) and fail `verify` on an
   uncovered ID?** It would put uncovered IDs into the evidence itself. But the overlay is generated
   by the same reconciler that runs the gate, so the list adds a second copy of what the gate already
   knows. *Recommendation:* no. Use the gate-side `--require-covered` only, and revisit if a consumer
   with no gate step needs it.
2. **Should IDs be normalized, for example `#7` to `owner/repo#7`, or by case or trailing slashes on
   URLs?** Normalizing needs repository context and gives surprising matches. *Recommendation:* exact
   match only, documented in the ADR. The reconciler writes both sides, so it can use one form.
3. **Under `--require-red-green`, should only overlay contracts count toward coverage?** A criterion
   "covered" only by a spec check that already passed on the base is the hollow pass red-green exists
   to stop. *Recommendation:* yes, couple them as in section 2. A gate that wants spec coverage can
   leave off `--require-red-green`.
4. **One failing covering result: decline, or accept if another covering result passes?** A failing
   result already fails the verdict, so in practice it declines either way. *Recommendation:* every
   covering result must pass. That names the actual broken result in the reason.
5. **Should a long criteria list come from a file (`--require-covered-file`)?** Issues with many
   criteria make long command lines. *Recommendation:* the repeatable flag only for now, matching
   `--overlay`. Add a file form when Damocles hits a real limit.
6. **Should budgets accept `covers`?** A performance criterion could be one. *Recommendation:* not in
   this task. The schema and code shape extend to budgets trivially later.
7. **Should the top-level `coverage` exist at all, given that the check ignores it?** The task asks for
   it, and the summary and other readers want it. *Recommendation:* keep it, document it as derived,
   and test that the check does not trust it.
8. **Risk: an agent edits an overlay's `covers` to claim coverage.** The overlay is trusted by hash
   (`--overlay`/`--overlay-sha256`, ADR 0033), so an edit declines. Spec covers are bound by
   `--spec`. This is no new exposure; the ADR states it.
9. **Risk: a whitespace-free ID rules out pasting criterion text.** *Recommendation:* accept this. IDs
   are references, and the reconciler should assign one per criterion (for example `acme/app#7/ac-2`).
