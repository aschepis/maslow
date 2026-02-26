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

## Decision Making

Agents are trusted to make decisions autonomously. Technology choices, architecture, design, library selection — decide, record, and keep building. The human can always revert via git if they disagree.

**What to do for each type of decision:**

| Decision | Action |
|----------|--------|
| Technology stack, database, framework, architectural pattern | Decide and write an ADR in `docs/adr/` |
| Library choice within a stack, naming conventions, file structure | Decide and write an ADR if non-obvious |
| Code organization, variable names, implementation details | Just code it |

**ADR format** — keep them short (`docs/adr/NNN-title.md`):

1. **Context**: What situation prompted this decision? (2-3 sentences)
2. **Decision**: What did you decide? (1 sentence)
3. **Consequences**: What are the trade-offs? (2-3 bullet points)

**Constraints from refs**: Before making decisions, read all refs that point to documentation (`docs/` files, URLs). If a ref contains explicit constraints ("use PostgreSQL", "use Tailwind"), follow them. If no ref constrains the choice, you decide.

## Draft Task Protocol

Draft tasks are how agents signal platform gaps — NOT how agents ask for permission or decompose work.

**When to create a draft task:**
- You discover a gap in maslow's verification capabilities that prevents you from confidently verifying what you've built (e.g., "can't test auth flows because variable capture isn't implemented in contracts")
- You need a tool or MCP capability that isn't available (e.g., "need browser MCP for visual regression testing")
- You discover a harness limitation that blocks the workflow

**When NOT to create a draft task:**
- To decompose your current work — just do the work
- To ask permission for a technology choice — make the choice, write an ADR
- To propose refactoring or improvements — write an ADR or just do it

**Format**: Create the task in `docs/tasks/` with `status: draft` and tag it:
- `kind:gap` — for verification or harness capability gaps
- `kind:capability` — for missing tools, MCPs, or external access

The human reviews draft tasks at their own pace and promotes important ones to `todo`.

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

## Task System

Tasks are how humans inject work into the project via git. Full convention: `docs/tasks/CONVENTION.md`.

### Quick Reference

- Tasks live in `docs/tasks/<id>_<SLUG>.md` with YAML frontmatter
- **Scan frontmatter only** to find actionable work — do not read the full body until you've chosen a task
- Only pick up tasks with `status: todo` and empty `assigned_to`
- Claim by setting `status: in_progress`, `assigned_to`, `assigned_at` — then commit and push
- If push fails (someone else claimed it), pull and pick another task
- When done, set `status: done` and commit

### Responding to Task Prompts

When asked to `implement the next task`: scan `docs/tasks/` for the lowest-ID `todo` task with no unresolved dependencies.

When asked to `implement task N`: go directly to `docs/tasks/N_*.md`.

When asked to `implement any task tagged X`: scan frontmatter for matching tags.

### Status Lifecycle

`draft` → `todo` → `in_progress` → `done` (with `blocked` as a side state)

- **Never** start work on `draft` tasks — those are human works-in-progress
- **Always** claim before starting work (commit + push the status change)

## Key Paths

| Path | Purpose |
|------|---------|
| `cmd/maslow/main.go` | CLI entrypoint — all six commands |
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
| `docs/tasks/` | Human-authored tasks with frontmatter metadata |
| `docs/tasks/CONVENTION.md` | Task format, lifecycle, and agent protocol |

## Conventions

- Go for all CLI and library code
- CUE for schema validation (`schema/maslow.cue`)
- YAML for the spec format (`maslow.yaml`)
- Deterministic output required: the same input must always produce the same result
- All error messages must reference file paths and spec section names
- Exit codes: `0` = success, `1` = validation or verification failure, `2` = usage error

## Progressive Verification

As you build, add corresponding verifications to `maslow.yaml`. Don't wait until the end — verify as you go.

| When you... | Add to maslow.yaml |
|-------------|-------------------|
| Create a new API endpoint | Add an HTTP contract scenario for it |
| Build a CLI command | Add a CLI contract scenario for it |
| Produce a build artifact | Add an `artifact_size` budget for it |
| Implement a performance-sensitive path | Add a performance budget for it |
| Add a new dependency or config file | Add it to refs |
| Create a file that should never be modified by agents | Add it to `policy.deny` or `policy.protected` |

Use what the schema can express today. When you hit something you can't express (e.g., need variable capture for auth flows, need database assertions), create a draft task tagged `kind:gap` describing the verification gap. Keep building — verify what you can, document what you can't.

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

## Harness Propagation Rule

**All improvements to the agentic harness (CLAUDE.md structure, docs/ conventions, task system, operating principles, scaffold templates) MUST be propagated into the harness generated by `maslow scaffold`.** This ensures every new project benefits from lessons learned.

When you improve any of these:
- CLAUDE.md content or structure
- `docs/tasks/CONVENTION.md` or the task system
- `docs/MAP.md` or `docs/PLAN.md` templates
- Operating principles or agent workflow conventions
- Any documentation pattern that helps agents work effectively

You must also update `internal/scaffold/scaffold.go` so that `maslow scaffold` generates the improved version. Update scaffold tests accordingly.
