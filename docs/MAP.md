# Maslow Repository Map

## What Maslow Is

Maslow is a verifier. It validates a declarative spec file (`maslow.yaml`) against a CUE schema,
runs verification, and produces deterministic evidence with one verdict. It runs the same way on a
laptop, inside an agent session and in CI, and imposes no workflow on the project (ADR 0016).

---

## Architecture Overview

```
maslow.yaml (spec)
     |
     v
cmd/maslow/main.go          <- CLI entrypoint; parses subcommands and flags
     |
     +-- validate           <- loads spec, runs CUE schema validation
     +-- verify             <- runs checks, contracts, budgets, refs, policy; emits evidence
     +-- audit              <- black-box mode; runs against artifact/endpoint targets
     +-- evidence check     <- judges an evidence file against trusted inputs (cmd/maslow/evidence.go)
     +-- evidence summary   <- prints an evidence file as the Action's Markdown job summary
     +-- init               <- writes a starter maslow.yaml; detects toolchain
     +-- version            <- prints build metadata
     |
     v
internal/
     |
     +-- spec/              <- parses and represents maslow.yaml
     +-- schema/            <- loads and applies schema/maslow.cue via CUE Go library
     +-- verify/            <- verification logic; profile selection; check execution
     +-- runner/            <- orchestrates check runners in defined order
     +-- audit/             <- black-box audit execution; target abstraction layer
     +-- evidence/          <- builds and writes the evidence file; output capping
     +-- progress/          <- start/finish events verify and audit report; the CLI prints them
     +-- gitrepo/           <- git queries: changed files, merge base, file at a commit
     +-- buildinfo/         <- version stamped into the binary and the evidence
     |
     v
schema/maslow.cue           <- CUE schema; single source of truth for MAS structure
schema/evidence.schema.json <- JSON Schema for the evidence file
reports/verify.json         <- verify evidence (reports/audit.json for audit)
```

---

## Key Entrypoints

| Path | Role |
|---|---|
| `cmd/maslow/main.go` | CLI entrypoint; command dispatch |
| `cmd/maslow/progress.go` | `--progress=auto\|always\|never` for verify and audit; prints `internal/progress` events to stderr only (ADR 0045) |
| `internal/spec/` | Parses `maslow.yaml`; exposes typed MAS struct |
| `internal/schema/` | Loads `schema/maslow.cue`; validates spec using CUE Go library |
| `internal/verify/` | Executes verification: profile selection, check sequencing, result collection; `--red-green` reruns overlay contracts in a base worktree (ADR 0034); reports each item to `Options.Progress` (ADR 0045) |
| `internal/runner/` | Runs individual checks (shell, HTTP, CLI); abstracts runner kinds; `retry.go` reruns failing checks and scenarios and folds the attempts into `pass`, `flaky`, `fail` or `error` (ADR 0039); `procgroup*.go` runs each check and CLI step in its own process group on Unix and kills the group when the step ends (ADR 0044) |
| `internal/audit/` | Black-box audit mode; binary targets, endpoint targets that run contracts against a base URL (ADR 0028), container stub |
| `internal/evidence/` | Builds the evidence report, computes the verdict, caps output, writes JSON; `Check` decides pass, needs_review or decline for a report against trusted inputs (ADR 0033); `coverage.go` derives `coverage` from per-result `covers` and holds the `--require-covered` rule (ADR 0043); `Summary` (`summary.go`) renders the job summary, requirements, flaky results and observed policy findings in their own sections (ADR 0036, ADR 0039) |
| `internal/gitrepo/` | Git queries shared by policy and spec loading, including line-level diffs against a base and temporary worktrees |
| `schema/maslow.cue` | CUE schema defining the MAS format |
| `action/action.yml` | Composite GitHub Action wrapping `maslow verify`; its job summary comes from `maslow evidence summary`; `action/README.md` documents the trusted-evidence pattern (ADR 0040) |
| `.github/workflows/maslow-evidence.yml` | Trusted evidence: a `pull_request_target` workflow, defined by `main`, for same-repository pull requests. `verify` runs a maslow built from the base with `contents: read`, no secrets and no cache. `record` never runs the change and records the evidence's sha256. Consumers find the run through the GitHub API (ADR 0040) |
| `scripts/check-trusted-workflow.sh` | Check `trusted-workflow`: fails if the trusted workflow gains another trigger, a write permission, secrets, a persisted token, a cache or a job without the same-repository guard |
| `zizmor.yml` | Config for the `zizmor` check (full profile only), with each suppression's reason |
| `scripts/build-release.sh` | Builds release archives and checksums |

---

## Canonical File Locations

| Artifact | Location | Notes |
|---|---|---|
| CUE schema | `schema/maslow.cue` | Versioned; controls all valid MAS structure |
| Project spec | `maslow.yaml` | Project root; one per repo (or per package in monorepos) |
| Evidence | `reports/verify.json`, `reports/audit.json` | Default paths; `--report` overrides |
| Evidence schema | `schema/evidence.schema.json` | JSON Schema for evidence version 2 |
| Test fixtures | `testdata/valid/` | Valid maslow.yaml examples for schema and verify tests |
| Test fixtures | `testdata/flaky/` | A spec whose check fails once and then passes, for the `verify-retries` contract |
| Test fixtures | `testdata/trusted-workflow/` | One good and several broken trusted-evidence workflows for the `trusted-workflow-check` contract |
| Test fixtures | `testdata/invalid/` | Invalid maslow.yaml examples that must fail validation |
| Test fixtures | `testdata/evidence/` | Evidence files for `maslow evidence check` and `evidence summary` tests and contracts; `*.summary.md` are the summary goldens |
| README | `README.md` | Install, usage, commands, dev instructions |
| Makefile | `Makefile` | Standard dev targets: build, test, vet, lint, validate, verify, clean |
| Documentation | `docs/` | MAP, PLAN, ADRs, templates, runbooks |
| ADRs | `docs/adr/` | Architecture and agent decision records |
| Templates | `docs/templates/` | Decision templates; unfilled design questions |
| Tasks | `docs/tasks/` | Human-authored tasks with YAML frontmatter |
| Task convention | `docs/tasks/CONVENTION.md` | Task format, lifecycle, and agent protocol |

---

## MAS Spec Structure (maslow.yaml)

A valid `maslow.yaml` defines:

- `mas` - schema version
- `name` - project name
- `toolchain` - required tools and version managers (asdf, mise, nix, custom)
- `refs` - external references (repos, modules, APIs)
- `policy` - path allow/deny lists; protected files; gated directories; shrink-only files and test-integrity checks judged on changed lines (ADR 0032); an `observe` block of the same rules reported without enforcement (ADR 0031)
- `contracts` - scenario-based behavioral contracts (HTTP, CLI and headless-browser steps); each scenario keeps its own cookie jar (ADR 0027), shared with its browser, which `internal/runner/browser.go` drives through chromedp (ADR 0035); an HTTP step's `multipart` parts are encoded by `internal/runner/multipart.go`, reading fixtures from the working directory (ADR 0041); `metric`, `log` and `alerts` steps hold thresholds in `internal/runner/observe.go` over providers in `internal/runner/providers.go` (`prometheus`, `loki`, `alertmanager`; `datadog` is a not-implemented stub) and record `observations` (ADR 0042)
- `budgets` - performance, availability, artifact size, complexity limits
- `checks` - named check definitions and runner configuration; checks and contracts take `retries`,
  capped by the top-level `max_retries` (default 5) (ADR 0039); checks, contracts and scenarios take
  `covers`, the requirement IDs they prove (ADR 0043); a check runs with a base environment plus its
  `env` list, unless `checks.inherit_env` is set (`internal/runner/check_env.go`, ADR 0046)
- `profiles` - named subsets of checks, contracts and budgets (`quick`, `full`, custom); refs and policy run unless a profile turns them off
- `audit` - black-box audit targets; an `endpoint` target runs selected contracts against its base URL
- `extensions` - domain overlays (web, game, cli, service)

---

## Evidence Format (version 2)

`maslow verify` writes `reports/verify.json` and `maslow audit` writes `reports/audit.json`;
`--report <path>` overrides either. `schema/evidence.schema.json` is the authoritative shape
(ADR 0020). Top-level fields:

- `evidence_version`, `maslow_version`, `command` - what produced the file
- `spec_path`, `spec_sha256`, `spec_source` - which spec bytes were verified (`head` or `base`)
- `overlays` - path and SHA-256 of each `--overlay` file; its contract results carry `overlay` (ADR 0029);
  under `--red-green` they also carry `head_status` and `base` (`sha`, `status`, `error`, `failed_step`),
  and a contract already passing on the base has status `not_red`, which fails the verdict (ADR 0034);
  `evidence check --require-red-green --base <sha>` declines unless every overlay contract has a
  `base` with that `sha` and status `fail`, and passed on the head (ADR 0037); a `flaky` base is
  `not_red` and a `flaky` head is not green (ADR 0039)
- `git_sha`, `base_sha` - the commit verified and the merge base policy was judged against
- `profile`, `timestamp`
- `check_results`, `contracts`, `budgets`, `refs`, `policy` - one entry per item with a `status`;
  an observed policy entry has `mode: observe`, a `would_*` status, and no effect on the verdict;
  a line-level policy entry lists flagged `path:line: text` entries in `lines`
- `verdict` - `fail`, `needs_review`, `inconclusive` or `pass` (in that precedence)
- `coverage` - derived from the `covers` on check and contract results (the union of a contract's
  and its scenario's IDs): one `{id, status, results}` per requirement ID, sorted by ID; it never
  affects the verdict, and `evidence check --require-covered <id>` recomputes it from the per-result
  `covers` instead of trusting it; with `--require-red-green` only overlay contracts count (ADR 0043)

Check output keeps the last `--output-limit` bytes and records `output_truncated` when cut.
A failing contract records its `failed_step` and that step's `output` under the same cap, and
lists files matching the scenario's `artifacts` patterns (ADR 0025). A failing browser step adds
`page_url` and `console` to `failed_step` and its screenshot to `artifacts` (ADR 0035). Values of the variables a
contract allow-lists in `env` are redacted from its result (ADR 0026). A check or scenario that
ran more than once lists every run in `attempts`; one that failed and then passed has status
`flaky`, which counts as `pass` except on an overlay contract, where it needs review (ADR 0039).
A scenario with `metric`, `log` or `alerts` steps lists each in `observations`, pass or fail: the
query, every evaluation time and the values judged. A breach is never retried (ADR 0042).

---

## Task System

This repository's own working process; Maslow does not generate it for other projects.
Tasks in `docs/tasks/` are the mechanism for humans to inject work and for agents to discover it.
See `docs/tasks/CONVENTION.md` for the full protocol.

- Tasks use YAML frontmatter for status, priority, assignment, and dependencies
- Agents scan frontmatter only (not full body) to find actionable work
- Multi-agent claiming uses git commit-and-push as a distributed lock
- Status lifecycle: `draft` → `todo` → `in_progress` → `done`

---

## Multi-Agent Conventions

- Each workstream owns a distinct directory scope; see `docs/PLAN.md` for assignments.
- Policy enforcement in `maslow.yaml` governs which paths agents may modify.
- Verification is the shared integration point; all agents must leave `maslow verify` green.
- Every material decision is recorded as an ADR in `docs/adr/`.
- Tasks are claimed via the protocol in `docs/tasks/CONVENTION.md` to prevent duplicate work.

---

## Navigation

- Architecture decisions: `docs/adr/`
- Execution plan and milestones: `docs/PLAN.md`
- Requirements source of truth: `docs/REQUIREMENTS.md`
- Decision templates (unfilled questions): `docs/templates/`
- Task queue and convention: `docs/tasks/`
