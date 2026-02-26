---
name: maslow
description: >-
  Define, build, and verify software using Maslow executable specifications.
  Use when creating a new project spec, validating a maslow.yaml file, running
  verification checks, scaffolding a new project, or working with the maslow
  task system. Activates for any task involving maslow.yaml, maslow CLI commands,
  or agent-driven software verification.
compatibility: Requires maslow CLI binary on PATH. Designed for Claude Code or similar coding agents.
metadata:
  author: maslow
  version: "0.1"
---

# Maslow Agent Skill

Maslow is an executable specification system for agent-built software. It takes a declarative `maslow.yaml` spec, validates it against a schema, runs verification checks, and produces structured evidence reports.

## Core Commands

```bash
# Validate a spec file against the schema
maslow validate maslow.yaml

# Run verification checks (quick profile for dev, full before merge)
maslow verify --profile quick
maslow verify --profile full

# Scaffold a new maslow-managed project
maslow scaffold --name my-project

# Initialize maslow.yaml in an existing project
maslow init

# Print version info
maslow version
```

## Workflow: Starting a New Project

1. Run `maslow scaffold --name <project-name>` to generate the full project structure.
2. Edit `maslow.yaml` to define your checks, contracts, budgets, and policies.
3. Implement the project code.
4. Run `maslow verify --profile quick` frequently during development.
5. Run `maslow verify --profile full` before merging.

## Workflow: Working on an Existing Project

1. Read `maslow.yaml` to understand the project spec.
2. Read `CLAUDE.md` for agent conventions and operating principles.
3. Read `docs/MAP.md` for architecture overview.
4. Read `docs/PLAN.md` for milestones and execution plan.
5. Check `docs/tasks/` for available work (scan frontmatter only).
6. Run `maslow verify --profile quick` to confirm the project is green before making changes.
7. Make changes in small increments, running verification after each.

## Workflow: Greenfield Build

When starting from a vague goal (e.g., "Build a TikTok clone with web and mobile apps"):

1. Read `maslow.yaml` for project structure, packages, and any existing refs.
2. **Read all refs** that point to documentation — PRDs, requirements, branding guides, tech decision docs. These are your north star.
3. Start building. Make technology and design decisions as you go. Record each material decision as an ADR in `docs/adr/`.
4. As features take shape, add corresponding verifications to `maslow.yaml`: contracts for API endpoints, budgets for artifacts and performance, refs for new config files.
5. When you hit a verification or harness gap you can't work around, create a draft task in `docs/tasks/` tagged `kind:gap` describing what you need.
6. Run `maslow verify --profile quick` frequently. Keep it green.
7. Update `docs/MAP.md` as the architecture emerges.

**Key principle**: Don't block on decisions. Decide, record (ADR), build, verify. The human reviews ADRs and draft tasks at their own pace. They can always revert via git.

## maslow.yaml Structure

A valid `maslow.yaml` defines:

- **`mas`** - Schema version (e.g., `"1.0"`)
- **`project`** - Project name, description, version
- **`toolchain`** - Required tools and version managers (asdf, mise, nix)
- **`refs`** - External references (files, repos, APIs)
- **`policy`** - Path deny/protected lists for agent safety
- **`checks`** - Named verification checks with runner configuration
- **`profiles`** - Named subsets of checks (`quick`, `full`, custom)
- **`contracts`** - Scenario-based behavioral contracts (CLI and HTTP)
- **`budgets`** - Performance, size, and complexity limits
- **`audit`** - Black-box audit targets

### Minimal Example

```yaml
mas: "1.0"
project:
  name: my-project
  description: "My project description"

checks:
  runner:
    - name: build
      kind: command
      run: "make build"
      timeout: 120s
      tags: [build]
    - name: test
      kind: command
      run: "make test"
      timeout: 300s
      tags: [test]

profiles:
  quick:
    description: Fast checks for development
    checks: [build]
  full:
    description: All checks
    checks: [build, test]
```

## Task System

Tasks in `docs/tasks/` are how humans inject work for agents.

### Discovering Tasks

1. List files matching `docs/tasks/[0-9]*_*.md`
2. Read only the YAML frontmatter (between `---` markers)
3. Filter for `status: todo` with empty `assigned_to` and no unresolved `depends_on`
4. Pick the lowest-ID matching task

### Claiming a Task

1. Set `status: in_progress`, `assigned_to`, and `assigned_at` in the frontmatter
2. Commit and push the claim
3. If push fails (conflict), pull and pick another task

### Completing a Task

1. Do the work, committing as you go
2. Set `status: done` in the frontmatter
3. Commit the status change

### Status Lifecycle

`draft` -> `todo` -> `in_progress` -> `done` (with `blocked` as a side state)

**Never** work on `draft` tasks. Only pick up `todo` tasks.

## Verification Evidence

Each run of `maslow verify` writes `reports/verify.json` containing:
- Timestamp, git SHA, profile used
- Per-check results, contract results, budget results
- Overall verdict: `pass`, `fail`, or `inconclusive`

The file is deterministic and machine-readable.

## Key Conventions

- Run `maslow verify --profile quick` after every significant change
- Run `maslow verify --profile full` before merging
- All error messages reference file paths and context
- Exit codes: `0` = success, `1` = validation/verification failure, `2` = usage error
- Keep `maslow.yaml` as the single source of truth for project verification
- Record material decisions as ADRs in `docs/adr/`
