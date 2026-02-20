// Package scaffold generates a full project structure for a new Maslow-managed project.
// It includes the agentic harness: CLAUDE.md, docs/, maslow.yaml, and supporting files.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options configures the scaffold output.
type Options struct {
	// ProjectName is the name of the new project (required).
	ProjectName string
	// Dir is the target directory. Defaults to ProjectName if empty.
	Dir string
	// Description is an optional project description.
	Description string
	// Toolchain is the detected or specified toolchain manager (asdf, mise, nix, or empty).
	Toolchain string
}

// Run generates the project scaffold at the target directory.
func Run(opts Options) error {
	if opts.ProjectName == "" {
		return fmt.Errorf("scaffold: project name is required")
	}

	dir := opts.Dir
	if dir == "" {
		dir = opts.ProjectName
	}

	description := opts.Description
	if description == "" {
		description = "TODO: describe your project"
	}

	// Create directory structure.
	dirs := []string{
		dir,
		filepath.Join(dir, "docs"),
		filepath.Join(dir, "docs", "adr"),
		filepath.Join(dir, "docs", "templates"),
		filepath.Join(dir, "docs", "tasks"),
		filepath.Join(dir, "reports"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("scaffold: failed to create directory %s: %w", d, err)
		}
	}

	// Generate all files.
	files := map[string]string{
		filepath.Join(dir, "maslow.yaml"):                       generateMaslowYAML(opts.ProjectName, description, opts.Toolchain),
		filepath.Join(dir, "CLAUDE.md"):                         generateClaudeMD(opts.ProjectName),
		filepath.Join(dir, "docs", "MAP.md"):                    generateMapMD(opts.ProjectName),
		filepath.Join(dir, "docs", "PLAN.md"):                   generatePlanMD(opts.ProjectName),
		filepath.Join(dir, "docs", "tasks", "CONVENTION.md"):    generateTaskConvention(),
		filepath.Join(dir, ".gitignore"):                        generateGitignore(),
		filepath.Join(dir, "reports", ".gitkeep"):               "",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("scaffold: failed to write %s: %w", path, err)
		}
	}

	return nil
}

func generateMaslowYAML(name, description, toolchain string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`mas: "1.0"
project:
  name: %s
  description: "%s"
`, name, description))

	if toolchain != "" {
		b.WriteString(fmt.Sprintf(`
toolchain:
  manager: %s
`, toolchain))
	}

	b.WriteString(`
checks:
  runner:
    - name: build
      kind: command
      run: "echo 'TODO: add build command'"
      timeout: 120s
      tags:
        - build
    - name: test
      kind: command
      run: "echo 'TODO: add test command'"
      timeout: 300s
      tags:
        - test
    - name: lint
      kind: command
      run: "echo 'TODO: add lint command'"
      timeout: 60s
      tags:
        - lint
    - name: self-validate
      kind: command
      run: "maslow validate maslow.yaml"
      timeout: 30s
      tags:
        - self-host

profiles:
  quick:
    description: Fast checks for development
    checks:
      - build
      - lint
      - self-validate
  full:
    description: All checks including tests
    checks:
      - build
      - test
      - lint
      - self-validate

policy:
  deny:
    - "**/.env"
    - "**/credentials*"
    - "vendor/**"
  protected:
    - maslow.yaml
`)
	return b.String()
}

func generateClaudeMD(name string) string {
	return fmt.Sprintf(`# %s — Agent Guide

## Project Overview

%s is managed by Maslow, an executable specification system for agent-built software.
The project is defined by a declarative maslow.yaml spec that is validated, verified, and auditable.

## What to Read First

1. This file (CLAUDE.md) — conventions, principles, and process
2. maslow.yaml — the spec (source of truth for what Maslow enforces)
3. docs/MAP.md — architecture overview and key entrypoints
4. docs/PLAN.md — milestones, workstreams, and Definition of Done

## Operating Principles

1. **Repository knowledge is the system of record.** Prefer adding/maintaining small, discoverable repo docs over long chat explanations. Keep MAP.md updated.
2. **Humans steer, agents execute.** Ask questions when required, but default to making progress by scaffolding, implementing, running checks, and iterating with feedback loops.
3. **Agent legibility is the goal.** Structure code/docs so an agent can reliably reason about it. Minimize assumptions. Prefer typed/validated boundaries and explicit conventions.
4. **Encode golden principles into the repo.** Mechanical rules (formatting, lint, directory conventions, invariants) must be enforced continuously. Treat cleanup like garbage collection, not a one-off refactor.
5. **Small increments.** Work in PR-sized changes. Keep changes narrow. Keep main green.
6. **Parallelize wherever possible.** Use subagents. Partition workstreams by folder/concern. Avoid collisions with explicit scopes.

## Non-Negotiable Behaviors

- All requirements must be captured in docs and enforced via maslow.yaml.
- Log and document everything: decisions, questions, conventions, and current state. Keep it in-repo.
- If you encounter a large, unscoped question or unknown requirement:
  (a) write a template doc into docs/templates/ with questions and placeholders,
  (b) ask the user to fill it,
  (c) continue only on work that does not depend on the missing info.
- Run checks frequently and use failures as feedback loops.
- Each material decision must be captured as an ADR in docs/adr/.

## Task System

Tasks are how humans inject work into the project via git. Full convention: docs/tasks/CONVENTION.md.

### Quick Reference

- Tasks live in docs/tasks/<id>_<SLUG>.md with YAML frontmatter
- **Scan frontmatter only** to find actionable work — do not read the full body until you've chosen a task
- Only pick up tasks with status: todo and empty assigned_to
- Claim by setting status: in_progress, assigned_to, assigned_at — then commit and push
- If push fails (someone else claimed it), pull and pick another task
- When done, set status: done and commit

### Responding to Task Prompts

When asked to "implement the next task": scan docs/tasks/ for the lowest-ID todo task with no unresolved dependencies.

When asked to "implement task N": go directly to docs/tasks/N_*.md.

When asked to "implement any task tagged X": scan frontmatter for matching tags.

### Status Lifecycle

draft -> todo -> in_progress -> done (with blocked as a side state)

- **Never** start work on draft tasks — those are human works-in-progress
- **Always** claim before starting work (commit + push the status change)

## Process for New Work

1. Read the goal. Load maslow.yaml, docs/MAP.md, docs/PLAN.md, and relevant code. Identify what exists vs what needs to be built.
2. Ask blocking questions upfront. For non-blocking questions, state your default and proceed. For big unscoped questions, write a template to docs/templates/.
3. Create a task list with concrete, ordered tasks and dependencies.
4. Launch parallel workstreams (docs, tests, research via subagents) while handling core implementation.
5. Build depth-first, smallest kernel first. Each increment must compile, pass tests, and not break existing functionality.
6. Run maslow verify --profile quick frequently during development.
7. Before declaring done, audit against docs/PLAN.md exit criteria line by line. Flag gaps honestly.
8. Encode new decisions as ADRs. Update maslow.yaml and docs/MAP.md if the architecture changed.
9. Commit narrowly with focused messages.

## Key Paths

| Path | Purpose |
|------|---------|
| maslow.yaml | Project spec — source of truth |
| CLAUDE.md | Agent guide — this file |
| docs/MAP.md | Architecture overview |
| docs/PLAN.md | Milestones and execution plan |
| docs/adr/ | Architecture Decision Records |
| docs/templates/ | Decision templates for unscoped questions |
| docs/tasks/ | Human-authored tasks with frontmatter metadata |
| docs/tasks/CONVENTION.md | Task format, lifecycle, and agent protocol |
| reports/ | Generated verification output (gitignored) |

## Conventions

- Deterministic output required: the same input must always produce the same result
- All error messages must reference file paths and relevant context
- Exit codes: 0 = success, non-zero = failure

## Verification

Run verification frequently:

`+"`"+`bash
# Quick checks during development
maslow verify --profile quick

# Full checks before merging
maslow verify --profile full

# Validate the spec itself
maslow validate maslow.yaml
`+"`"+`

## Adding a New Feature

1. Check docs/PLAN.md for the relevant milestone
2. Update maslow.yaml if checks, contracts, or budgets change
3. Implement the feature
4. Add or update tests
5. Run maslow verify --profile quick
6. Write an ADR in docs/adr/ if the change involves a material decision
7. Update docs/MAP.md if architecture changed
8. Commit narrowly with focused messages

## Harness Propagation Rule

**All improvements to the agentic harness (CLAUDE.md structure, docs/ conventions, task system,
operating principles, scaffold templates) MUST be propagated into the harness generated by
maslow scaffold.** This ensures every new project benefits from lessons learned.

When you improve any of these: CLAUDE.md content, task conventions, MAP.md/PLAN.md templates,
operating principles, or agent workflow conventions — you must also update the scaffold code
so that maslow scaffold generates the improved version.
`, name, name)
}

func generateMapMD(name string) string {
	return fmt.Sprintf(`# %s — Repository Map

## What This Project Is

TODO: Describe your project here.

---

## Architecture Overview

TODO: Add an architecture diagram or description.

` + "```" + `
TODO: Add key components and their relationships
` + "```" + `

---

## Key Entrypoints

| Path | Role |
|------|------|
| TODO | TODO |

---

## Canonical File Locations

| Artifact | Location | Notes |
|----------|----------|-------|
| Project spec | maslow.yaml | Source of truth for verification |
| Agent guide | CLAUDE.md | Conventions and process |
| Verify output | reports/verify.json | Written by maslow verify |
| Documentation | docs/ | MAP, PLAN, ADRs, templates |
| ADRs | docs/adr/ | Architecture Decision Records |
| Templates | docs/templates/ | Decision templates |
| Tasks | docs/tasks/ | Human-authored tasks for agents |
| Task convention | docs/tasks/CONVENTION.md | Task format and agent protocol |

---

## Multi-Agent Conventions

- Each workstream owns a distinct directory scope; see docs/PLAN.md for assignments.
- Policy enforcement in maslow.yaml governs which paths agents may modify.
- Verification is the shared integration point; all agents must leave maslow verify green.
- Every material decision is recorded as an ADR in docs/adr/.

---

## Navigation

- Architecture decisions: docs/adr/
- Execution plan and milestones: docs/PLAN.md
- Project spec: maslow.yaml
- Decision templates (unfilled questions): docs/templates/
`, name)
}

func generatePlanMD(name string) string {
	return fmt.Sprintf(`# %s — Execution Plan

## Guiding Principle

Build the smallest working increment first. Expand depth-first. Every change must
compile, pass tests, and leave maslow verify --profile quick green.

---

## Milestones

### M1 - Foundation

Goal: Basic project structure, build, and test pipeline working.

Deliverables:
- TODO: List deliverables

Exit criteria:
- maslow verify --profile quick passes
- TODO: Add specific exit criteria

---

### M2 - Core Features

Goal: TODO

Deliverables:
- TODO: List deliverables

Exit criteria:
- maslow verify --profile full passes
- TODO: Add specific exit criteria

---

## Parallel Workstreams

Workstreams are scoped by directory to prevent agent collisions.

| Stream | Owner Paths | Milestone Scope |
|--------|-------------|-----------------|
| TODO | TODO | TODO |

---

## Risk Register

### R1 - TODO

Risk: TODO

Mitigation:
- TODO

---

## Definition of Done

- maslow verify --profile full passes
- All ADRs are written
- docs/MAP.md is current
- TODO: Add project-specific criteria
`, name)
}

func generateGitignore() string {
	return `# Maslow reports
reports/verify.json

# Build artifacts
bin/

# Environment files
.env
.env.*

# OS files
.DS_Store
Thumbs.db

# IDE
.idea/
.vscode/
*.swp
*.swo
`
}

func generateTaskConvention() string {
	return `# Task Convention

This document defines the format, lifecycle, and multi-agent protocol for tasks stored in ` + "`docs/tasks/`" + `.

## Purpose

Tasks are the mechanism for humans to inject work into the project via git and for agents to
discover, claim, and execute that work. The system is designed for:

- **Human authoring**: humans write tasks in markdown, store drafts, and promote to todo when ready.
- **Agent discovery**: agents scan frontmatter only (not full body) to find actionable work.
- **Multi-agent safety**: claiming protocol prevents duplicate work across clones.

## File Format

Each task is a markdown file with YAML frontmatter:

` + "```" + `markdown
---
id: 1
title: Short imperative title
status: todo
priority: medium
created: 2026-01-01
updated: 2026-01-01
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [area-x]
---

## Objective

What needs to be done and why.

## Requirements

- Requirement 1

## Acceptance Criteria

- [ ] Criterion 1
` + "```" + `

### Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | integer | yes | Unique task ID, matches filename prefix |
| title | string | yes | Short imperative description (< 80 chars) |
| status | enum | yes | Current lifecycle state (see below) |
| priority | enum | no | critical, high, medium, low (default: medium) |
| created | date | yes | ISO 8601 date when task was created |
| updated | date | yes | ISO 8601 date of last status change |
| assigned_to | string | no | Identifier of the agent/human working on it |
| assigned_at | datetime | no | ISO 8601 timestamp when assignment happened |
| depends_on | list[int] | no | IDs of tasks that must complete first |
| tags | list[string] | no | Categorization tags |

### Naming Convention

Files are named ` + "`<id>_<SLUG>.md`" + ` where:
- ` + "`<id>`" + ` is a sequential integer
- ` + "`<SLUG>`" + ` is an uppercase snake_case summary
- Example: ` + "`3_ADD_CACHING_LAYER.md`" + `

The special file CONVENTION.md (this file) is not a task.

## Status Lifecycle

` + "```" + `
draft --> todo --> in_progress --> done
                      |
                      +--> blocked --> in_progress
` + "```" + `

| Status | Meaning | Who sets it |
|--------|---------|-------------|
| draft | Work in progress by the author; not ready for agents | Human |
| todo | Ready to be picked up; requirements are complete | Human |
| in_progress | Actively being worked on by an assigned agent | Agent |
| blocked | Cannot proceed; dependency or question unresolved | Agent |
| done | All acceptance criteria met; work is complete | Agent |

Rules:
- **Humans** control draft -> todo transitions. Agents must never start work on draft tasks.
- **Agents** control todo -> in_progress -> done transitions.
- Only tasks with status: todo and empty assigned_to are available for pickup.
- Tasks with unresolved depends_on (dependencies not in done status) should not be started.

## Agent Protocol

### Discovering Tasks

Agents should scan task files by reading only frontmatter (first ~15 lines):

1. List files in docs/tasks/ matching [0-9]*_*.md
2. For each file, read the frontmatter block (between --- markers)
3. Filter for status: todo with empty assigned_to and no unresolved dependencies
4. Select a task based on priority and ID order (lower ID = older = prefer first)

### Claiming a Task

To prevent duplicate work across agents on different clones:

1. **Pull latest**: git pull --rebase before claiming
2. **Check status**: re-read the frontmatter to confirm still todo and unassigned
3. **Update frontmatter**: set status: in_progress, assigned_to, assigned_at, updated
4. **Commit and push**: commit with message ` + "`task(<id>): claim task <id> - <title>`" + `
5. **If push fails** (conflict): pull, check if someone else claimed it, pick another task

### Working on a Task

1. Read the full task body for requirements and acceptance criteria
2. Do the work in one or more commits (reference the task in commit messages)
3. When done, update frontmatter: status: done, updated: <today>
4. Commit the status change

### Handling Blocks

If the agent cannot proceed:
1. Set status: blocked with a note in the task body explaining why
2. Commit and push so other agents (or humans) can see the block
3. Move on to the next available task

## Creating New Tasks

Humans create tasks by:

1. Choosing the next available ID (one higher than the highest existing)
2. Creating docs/tasks/<id>_<SLUG>.md with frontmatter and body
3. Setting status: draft while iterating on the description
4. Changing to status: todo when ready for agent pickup
5. Committing and pushing
`
}
