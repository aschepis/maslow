# Maslow Project - Agent Guide

## Project Overview

Maslow is an executable specification system for agent-built software. It takes a declarative `maslow.yaml` spec, validates it against a CUE schema, runs verification tasks, and produces structured reports.

Stack: Go CLI, CUE schema, YAML specs.

## What to Read First

1. This file (CLAUDE.md) — conventions, principles, and process
2. `docs/MAP.md` — architecture overview and key entrypoints
3. `docs/PLAN.md` — milestones, workstreams, and Definition of Done
4. `maslow.yaml` — the self-hosting spec (source of truth for what Maslow enforces)
5. `docs/REQUIREMENTS.md` — full product requirements

## Operating Principles

1. **Repository knowledge is the system of record.** Prefer adding/maintaining small, discoverable repo docs over long chat explanations. Keep MAP.md updated.
2. **Humans steer, agents execute.** Ask questions when required, but default to making progress by scaffolding, implementing, running checks, and iterating with feedback loops.
3. **Agent legibility is the goal.** Structure code/docs so an agent can reliably reason about it. Minimize assumptions. Prefer typed/validated boundaries and explicit conventions.
4. **Encode golden principles into the repo.** Mechanical rules (formatting, lint, directory conventions, invariants) must be enforced continuously. Treat cleanup like garbage collection, not a one-off refactor.
5. **Small increments.** Work in PR-sized changes. Keep changes narrow. Keep main green.
6. **Parallelize wherever possible.** Use subagents. Partition workstreams by folder/concern. Avoid collisions with explicit scopes.

## Non-Negotiable Behaviors

- All requirements in `docs/REQUIREMENTS.md` must be met, including the self-hosting bootstrap constraint.
- Log and document everything: decisions, questions, conventions, and current state. Keep it in-repo.
- If you encounter a large, unscoped question or unknown requirement:
  (a) write a template doc into `docs/templates/` with questions and placeholders,
  (b) ask the user to fill it,
  (c) continue only on work that does not depend on the missing info.
- Run tests/checks frequently and use failures as feedback loops.
- Each material decision must be captured as an ADR in `docs/adr/`.

## Process for New Milestones

1. Read the goal. Load REQUIREMENTS.md, PLAN.md, MAP.md, and relevant code. Identify what exists vs what needs to be built.
2. Ask blocking questions upfront. For non-blocking questions, state your default and proceed. For big unscoped questions, write a template to `docs/templates/`.
3. Create a task list with concrete, ordered tasks and dependencies.
4. Launch parallel workstreams (docs, tests, research via subagents) while handling core implementation.
5. Build depth-first, smallest kernel first. Each increment must compile, pass tests, and not break existing functionality.
6. Run `go test ./...`, `go vet`, `cue vet schema/maslow.cue`, and `maslow verify --profile quick` frequently.
7. Before declaring done, audit against PLAN.md's exit criteria line by line. Flag gaps honestly.
8. Encode new decisions as ADRs. Update `maslow.yaml` and MAP.md if the architecture changed.
9. Commit narrowly with focused messages.

## Key Paths

| Path | Purpose |
|------|---------|
| `cmd/maslow/main.go` | CLI entrypoint — all five commands |
| `schema/maslow.cue` | CUE schema (authoritative spec definition) |
| `internal/schema/` | CUE API wrapper and embedded schema |
| `internal/spec/` | YAML spec loading and Go types |
| `internal/verify/` | Verification orchestration |
| `internal/audit/` | Black-box audit against artifacts |
| `internal/evidence/` | Report creation and JSON emission |
| `internal/runner/` | Check, contract, and budget execution |
| `internal/scaffold/` | Project scaffolding with agentic harness |
| `testdata/valid/` | Fixtures that must pass validation |
| `testdata/invalid/` | Fixtures that must fail validation |
| `reports/` | Generated output (gitignored) |
| `docs/` | MAP, PLAN, ADRs, templates |
| `docs/adr/` | Architecture Decision Records |
| `docs/templates/` | Decision templates for unscoped questions |

## Conventions

- Go for all CLI and library code
- CUE for schema validation (`schema/maslow.cue`)
- YAML for the spec format (`maslow.yaml`)
- Deterministic output required: the same input must always produce the same result
- All error messages must reference file paths and spec section names
- Exit codes: `0` = success, `1` = validation or verification failure, `2` = usage error

## Adding a New Feature

1. Check `docs/PLAN.md` for the relevant milestone
2. Update `schema/maslow.cue` if the spec format changes
3. Copy schema to `internal/schema/maslow.cue` (embedded via `go:embed`)
4. Add or update fixtures in `testdata/valid/` and `testdata/invalid/`
5. Implement in the appropriate `internal/` package
6. Wire up in `cmd/maslow/main.go` if a new subcommand or flag is needed
7. Run `go test ./...` and `cue vet schema/maslow.cue`
8. Update `maslow.yaml` if new checks, contracts, or budgets apply
9. Write an ADR if the change involves a material decision
10. Update `docs/MAP.md` if architecture changed
