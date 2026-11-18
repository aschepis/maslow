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
| `internal/spec/` | Parses `maslow.yaml`; exposes typed MAS struct |
| `internal/schema/` | Loads `schema/maslow.cue`; validates spec using CUE Go library |
| `internal/verify/` | Executes verification: profile selection, check sequencing, result collection |
| `internal/runner/` | Runs individual checks (shell, HTTP, CLI); abstracts runner kinds |
| `internal/audit/` | Black-box audit mode; binary targets, endpoint targets that run contracts against a base URL (ADR 0028), container stub |
| `internal/evidence/` | Builds the evidence report, computes the verdict, caps output, writes JSON |
| `internal/gitrepo/` | Git queries shared by policy and spec loading |
| `schema/maslow.cue` | CUE schema defining the MAS format |
| `action/action.yml` | Composite GitHub Action wrapping `maslow verify` |
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
| Test fixtures | `testdata/invalid/` | Invalid maslow.yaml examples that must fail validation |
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
- `policy` - path allow/deny lists; protected files; gated directories; an `observe` block of the same rules reported without enforcement (ADR 0031)
- `contracts` - scenario-based behavioral contracts (HTTP and CLI calls); each scenario keeps its own cookie jar (ADR 0027)
- `budgets` - performance, availability, artifact size, complexity limits
- `checks` - named check definitions and runner configuration
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
- `overlays` - path and SHA-256 of each `--overlay` file; its contract results carry `overlay` (ADR 0029)
- `git_sha`, `base_sha` - the commit verified and the merge base policy was judged against
- `profile`, `timestamp`
- `check_results`, `contracts`, `budgets`, `refs`, `policy` - one entry per item with a `status`;
  an observed policy entry has `mode: observe`, a `would_*` status, and no effect on the verdict
- `verdict` - `fail`, `needs_review`, `inconclusive` or `pass` (in that precedence)

Check output keeps the last `--output-limit` bytes and records `output_truncated` when cut.
A failing contract records its `failed_step` and that step's `output` under the same cap, and
lists files matching the scenario's `artifacts` patterns (ADR 0025). Values of the variables a
contract allow-lists in `env` are redacted from its result (ADR 0026).

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
