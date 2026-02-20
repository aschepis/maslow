# Maslow Repository Map

## What Maslow Is

Maslow is an executable specification system for agent-built software. It validates a declarative
spec file (`maslow.yaml`) against a CUE schema, runs verification checks, and produces deterministic
evidence. It is designed to be used equally by humans and AI agents.

---

## Architecture Overview

```
maslow.yaml (spec)
     |
     v
cmd/maslow/main.go          <- CLI entrypoint; parses subcommands and flags
     |
     +-- validate           <- loads spec, runs CUE schema validation
     +-- verify             <- runs check harness, emits reports/verify.json
     +-- audit              <- black-box mode; runs against artifact/endpoint targets
     +-- scaffold           <- scaffolds a new project with agentic harness
     +-- harness            <- manages harness lifecycle (install, update, detach)
     +-- init               <- scaffolds maslow.yaml; detects toolchain
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
     +-- evidence/          <- emits reports/verify.json; deterministic serialization
     +-- scaffold/          <- project scaffolding with agentic harness generation
     +-- harness/           <- harness lifecycle management (install, update, detach)
     |
     v
schema/maslow.cue           <- CUE schema; single source of truth for MAS structure
reports/verify.json         <- output evidence file; written by verify and audit
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
| `internal/audit/` | Black-box audit mode; supports binary, container, and HTTP endpoint targets |
| `internal/evidence/` | Serializes and writes `reports/verify.json`; ensures determinism |
| `internal/scaffold/` | Generates new project structure with agentic harness (CLAUDE.md, docs/, maslow.yaml) |
| `internal/harness/` | Manages harness lifecycle: install into existing projects, update to latest, detach |
| `schema/maslow.cue` | CUE schema defining the MAS format |

---

## Canonical File Locations

| Artifact | Location | Notes |
|---|---|---|
| CUE schema | `schema/maslow.cue` | Versioned; controls all valid MAS structure |
| Project spec | `maslow.yaml` | Project root; one per repo (or per package in monorepos) |
| Verify output | `reports/verify.json` | Written by `maslow verify` and `maslow audit` |
| Test fixtures | `testdata/valid/` | Valid maslow.yaml examples for schema and verify tests |
| Test fixtures | `testdata/invalid/` | Invalid maslow.yaml examples that must fail validation |
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
- `policy` - path allow/deny lists; protected files; gated directories
- `contracts` - scenario-based behavioral contracts (HTTP and CLI calls)
- `budgets` - performance, availability, artifact size, complexity limits
- `checks` - named check definitions and runner configuration
- `profiles` - named subsets of checks (`quick`, `full`, custom)
- `audit` - black-box audit targets and configuration
- `extensions` - domain overlays (web, game, cli, service)

---

## Evidence Format (reports/verify.json)

Every run of `maslow verify` or `maslow audit` writes `reports/verify.json`. It contains:

- `timestamp` - ISO 8601 run time
- `git_sha` - current HEAD SHA (if available)
- `profile` - profile name used
- `check_results` - result per check runner
- `contract_results` - result per contract scenario
- `budget_results` - result per budget assertion
- `verdict` - `pass`, `fail`, or `inconclusive`

The file is stable, deterministic, diffable, and machine-readable.

---

## Task System

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
