---
id: 43
title: Show observe-mode policy findings in the Action's job summary
status: done
priority: medium
created: 2026-11-18
updated: 2026-11-18
assigned_to: "claude-subagent"
assigned_at: "2026-11-19T02:37:11Z"
depends_on: []
tags: [kind:gap, action, evidence, policy, cli]
---

## Objective

Observe mode (task 31, ADR 0031) exists so people can see what a policy rule would flag before it
enforces. The GitHub Action's job summary lists only `fail`, `error` and `needs_review` results, so
`would_fail`, `would_need_review` and `would_error` findings never appear there. A run whose only
findings are observed shows "Nothing failed." Users see nothing unless they download the evidence
artifact or read the step log. Maslow's own CI hits this: `maslow.yaml` observes `test_integrity`.

## Requirements

- The job summary lists every observe-mode policy result whose status is `would_*`, in a section
  separate from the enforced results and labelled as not affecting the verdict.
- Enforced results are listed exactly as today.
- A run with no observe-mode results shows no observe section.
- The summary is rendered by Maslow (`maslow evidence summary <file>`), not by jq in the Action, so
  it is covered by Go tests.
- The Action still writes a useful summary when the installed `maslow` predates the command.

## Acceptance Criteria

- [x] `maslow evidence summary <file>` prints the Markdown summary; wired in `cmd/maslow/evidence.go`.
- [x] Golden-file tests for pass, needs_review, fail-with-observe and observe-only evidence.
- [x] The Action's "Write job summary" step calls the command, falling back to the old jq when the
      binary exits 2.
- [x] CLI contract scenario in `maslow.yaml` for the observe case.
- [x] `action/README.md`, `docs/MAP.md` and the CHANGELOG gap note (CHANGELOG.md:13) updated; ADR written.

## Notes

The CHANGELOG for v0.8.0 already says "The Action's job summary does not list them yet."
Task 39 proposes the same command with more content (failed step, output tail, `--max-bytes`).
This task builds the command's first version; 39 then extends it. See "Relation to task 39" below.

## Proposed implementation

### 1. What happens today

`action/action.yml:169-196` ("Write job summary") writes to `$GITHUB_STEP_SUMMARY`:

- `:178` heading `## Maslow: <verdict>`, from the verify step's output.
- `:179-183` if no evidence file exists, prints a note and stops.
- `:184` one jq line with profile, spec path, spec source, maslow version and base SHA.
- `:186-195` one jq program builds the table. `:187` defines `bad` as `fail` or `error`. `:189-192`
  select bad checks, contracts, budgets and refs. `:193` selects policy results that are bad or
  `needs_review`. `:194` prints "Nothing failed." when no row matched.

Policy results under `policy.observe` carry `mode: "observe"` and a `would_*` status
(`internal/evidence/report.go:138-151`, `schema/evidence.schema.json:55` and `:145`). None of those
statuses match `:187` or `:193`, so they are dropped. Two cases:

- Only observed findings: the verdict is `pass` (or `inconclusive` if the run's only results are
  observed ones, per ADR 0031). The summary says "Nothing failed." and nothing else.
- An enforced failure plus observed findings: the table lists the enforced failure only.

The CLI does show them: `printReport` (`cmd/maslow/main.go:391-409`) prints every policy result
with its mode, e.g. `[would_fail] policy/deny (observe): vendor/**`. Nothing tests the Action's jq.

### 2. Proposed change

Recommendation: move the summary into Maslow rather than patching the jq.

Patching the jq is a two-line change at `:193`, but it extends untested shell, and every later
summary change (task 39's output tails, redaction, size caps) would go into jq too. ADR 0024 and
`action/README.md:3` say all logic lives in Maslow and the Action only passes inputs through; the
summary is the one exception. Moving it costs one small Go function and a fallback in the Action.

**`internal/evidence/summary.go`** (new): `func Summary(r *Report) string` returns Markdown:

1. `## Maslow: \`<verdict>\`` and the metadata line, same content as `:178` and `:184`.
2. The enforced table, same rows and columns as `:186-195`, skipping any result with
   `Mode == "observe"`.
3. If any observe-mode result has a `would_*` status, a separate section after the enforced one:

```markdown
## Maslow: `pass`
Profile `full` · spec `maslow.yaml` from `base` · maslow 0.9.0 · base `3f2a9c1e0b7d`

Nothing failed.

### Observed policy rules (not enforced)

These rules run in observe mode. Their findings do not change the verdict.

| Kind | Rule | Would be | Detail |
|---|---|---|---|
| deny | vendor/** | would_fail | vendor/lib/x.go |
| test_integrity | internal/** | would_need_review | assertion removed in foo_test.go |

2 observed rules passed.
```

The observe rows are kept apart from enforced ones by their own heading, a sentence saying they do
not affect the verdict, and a "Would be" column instead of "Status". Cell escaping (newlines to
spaces, `|` escaped) matches `:188`. Detail is `Detail`, else the first of `Files`, else `Error`.

**`cmd/maslow/evidence.go`**: `cmdEvidence` dispatches `summary` beside `check`. `maslow evidence
summary <file>` reads the evidence, prints `Summary` to stdout, exits 0. Unreadable or invalid
evidence exits 1 with the path in the message; wrong arguments exit 2. Usage text in `main.go`.

**`action/action.yml`** "Write job summary": keep the no-evidence branch (`:179-183`), then:

```bash
if ! "$MASLOW" evidence summary "$REPORT" >> "$GITHUB_STEP_SUMMARY" 2>/dev/null; then
  # existing heading + jq from :178-195, unchanged
fi
```

The step gains `MASLOW: ${{ steps.install.outputs.path }}` in its `env`. The fallback runs only when
the installed release lacks the command, so users pinning `version: 0.6.0` keep today's summary.

**`action/README.md`**: one sentence that the summary includes observed policy findings from the
release that adds `evidence summary` onward. **`docs/MAP.md`**: add `summary.go`. **CHANGELOG**:
replace the "does not list them yet" note.

**Relation to task 39**: 39 wants the same command with failure detail (failed step and target,
output tail, `--max-bytes`, redaction). This task ships the command with only what the Action shows
today plus the observe section. 39 should be amended to depend on 43 and extend `Summary`, and its
Notes line about the Action becomes done here.

### 3. Tests and checks

- `internal/evidence/summary_test.go`: golden tests over fixtures in `testdata/evidence/`:
  existing `pass.json` and `needs-review.json`, plus new `observe-only.json` (verdict `pass`, one
  `would_fail` deny, one observed pass) and `fail-with-observe.json` (one failing check, one
  `would_need_review` gated). Goldens in `testdata/evidence/*.summary.md`. Assert the observe
  section is absent for `pass.json` and that observe rows never appear in the enforced table.
- A cell-escaping case: a detail with `|` and a newline.
- `cmd/maslow/evidence_test.go`: exit codes 0, 1 (missing file), 2 (no argument, two arguments).
- `maslow.yaml`: a CLI contract running `evidence summary testdata/evidence/observe-only.json`,
  expecting exit 0 and stdout containing `Observed policy rules` and `would_fail`.
- The Action: after this change its only logic is "call the command, else fall back". The dogfood
  step in `.github/workflows/ci.yml:56-63` runs it with `binary: bin/maslow`, so the primary path
  runs on every PR. The fallback is the current jq, unchanged. Run `go test ./...`, `go vet`,
  `cue vet schema/maslow.cue` and `maslow verify --profile quick`.

### 4. ADR

Yes, a short one (0036; task 44 takes 0037): "The Action's job summary is rendered by `maslow
evidence summary`." It amends ADR 0024, which lists the summary as Action behaviour, and records
the fallback for older releases.

### 5. Risks and open questions

1. **Fallback or minimum version?** Keeping the jq as a fallback means two renderers until it is
   removed. Recommendation: keep it for one minor release, then delete it and print "upgrade maslow
   for a job summary" instead.
2. **List observed passes?** Listing every observed rule that passed adds noise. Recommendation:
   rows for `would_*` only, plus a one-line count of observed passes so an empty section is not
   ambiguous.
3. **Fold into task 39?** Doing 39 in full here makes this task larger and blocks a visible gap on
   redaction and size-cap design. Recommendation: keep 43 narrow and make 39 depend on it.
4. **Summary size.** `$GITHUB_STEP_SUMMARY` is capped at 1 MiB per step. Many observed findings
   with long details could approach it. Recommendation: no cap now; task 39's `--max-bytes` covers it.
5. **`inconclusive` when a run has only observed results.** The heading says `inconclusive` with "Nothing
   failed." below, which reads oddly. Recommendation: out of scope; the observe section explains it.
