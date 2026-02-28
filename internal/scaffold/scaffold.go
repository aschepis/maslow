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
		filepath.Join(dir, ".skills", "maslow"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("scaffold: failed to create directory %s: %w", d, err)
		}
	}

	// Generate all files.
	files := map[string]string{
		filepath.Join(dir, "maslow.yaml"):                       generateMaslowYAML(opts.ProjectName, description, opts.Toolchain),
		filepath.Join(dir, "CLAUDE.md"):                         GenerateClaudeMD(opts.ProjectName),
		filepath.Join(dir, "docs", "MAP.md"):                    GenerateMapMD(opts.ProjectName),
		filepath.Join(dir, "docs", "PLAN.md"):                   GeneratePlanMD(opts.ProjectName),
		filepath.Join(dir, "docs", "tasks", "CONVENTION.md"):    GenerateTaskConvention(),
		filepath.Join(dir, ".gitignore"):                        GenerateGitignore(),
		filepath.Join(dir, "reports", ".gitkeep"):               "",
		filepath.Join(dir, ".skills", "maslow", "SKILL.md"):    GenerateSkillMD(),
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

// GenerateClaudeMD generates the CLAUDE.md agent guide content for a project.
func GenerateClaudeMD(name string) string {
	return fmt.Sprintf(`# %s — Agent Guide

## Project Overview

%s is managed by Maslow, an executable specification system for agent-built software.
The project is defined by a declarative maslow.yaml spec that is validated, verified, and auditable.

## What to Read First

1. This file (CLAUDE.md) — conventions, principles, and process
2. maslow.yaml — the spec (source of truth); read all refs that point to docs/ files
3. docs/MAP.md — architecture overview and key entrypoints
4. docs/PLAN.md — milestones, workstreams, and Definition of Done

## Refs as Generative Input

Refs in maslow.yaml are not just verification targets — they are your primary input. Before starting any
generative work, read all refs to understand your context:

- **Doc refs** (kind: file pointing to docs/) are requirements, constraints, and context. These are your north star.
  Read them before making any decisions.
- **Config refs** (kind: file pointing to config files like .prettierrc, .eslint) are verification targets.
  Ensure they exist and are respected, but don't treat them as requirements.
- **URL refs** (kind: url) may point to external context — competitor apps, design inspiration, API standards.
  Fetch and use them as context when relevant.
- **MCP refs** (kind: mcp) declare tool dependencies. Check if you have these capabilities available.

**Convention**: If a ref points to a file in docs/, treat it as input. If it points to a config file
or binary, treat it as a verification target. When in doubt, read it — more context is always better.

Humans should add their requirements, aspirations, branding guidelines, and tech preferences as refs
pointing to docs/ files. The refs section is a reading list for anyone (human or agent) who wants to
understand the project's intent.

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
| Technology stack, database, framework, architectural pattern | Decide and write an ADR in docs/adr/ |
| Library choice within a stack, naming conventions, file structure | Decide and write an ADR if non-obvious |
| Code organization, variable names, implementation details | Just code it |

**ADR format** — keep them short (docs/adr/NNN-title.md):

1. **Context**: What situation prompted this decision? (2-3 sentences)
2. **Decision**: What did you decide? (1 sentence)
3. **Consequences**: What are the trade-offs? (2-3 bullet points)

**Constraints from refs**: Before making decisions, read all refs that point to documentation (docs/ files, URLs). If a ref contains explicit constraints ("use PostgreSQL", "use Tailwind"), follow them. If no ref constrains the choice, you decide.

## Draft Task Protocol

Draft tasks are how agents signal platform gaps — NOT how agents ask for permission or decompose work.

**When to create a draft task:**
- You discover a gap in maslow's verification capabilities that prevents you from confidently verifying what you've built (e.g., "can't test auth flows because variable capture isn't implemented in contracts")
- You need a tool or MCP capability that isn't available (e.g., "need browser MCP for visual regression testing")
- You discover a harness limitation that blocks the workflow (e.g., "task convention doesn't support cross-package dependencies")

**When NOT to create a draft task:**
- To decompose your current work — just do the work
- To ask permission for a technology choice — make the choice, write an ADR
- To propose refactoring or improvements — write an ADR or just do it

**Format**: Create the task in docs/tasks/ with status: draft and tag it:
- ` + "`kind:gap`" + ` — for verification or harness capability gaps
- ` + "`kind:capability`" + ` — for missing tools, MCPs, or access

The human reviews draft tasks at their own pace and promotes important ones to todo.

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

## Progressive Verification

As you build, add corresponding verifications to maslow.yaml. Don't wait until the end — verify as you go.

| When you... | Add to maslow.yaml |
|-------------|-------------------|
| Create a new API endpoint | Add an HTTP contract scenario for it |
| Build a CLI command | Add a CLI contract scenario for it |
| Produce a build artifact | Add an artifact_size budget for it |
| Implement a performance-sensitive path | Add a performance budget for it |
| Add a new dependency or config file | Add it to refs |
| Create a file that should never be modified by agents | Add it to policy.deny or policy.protected |

Use what the schema can express today. When you hit something you can't express (e.g., need variable capture for auth flows, need database assertions), create a draft task tagged kind:gap describing the verification gap. Keep building — verify what you can, document what you can't.

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

// GenerateMapMD generates the docs/MAP.md repository map content.
func GenerateMapMD(name string) string {
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

// GeneratePlanMD generates the docs/PLAN.md execution plan content.
func GeneratePlanMD(name string) string {
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

// GenerateSkillMD generates the .skills/maslow/SKILL.md agent skill content.
func GenerateSkillMD() string {
	return `---
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

Maslow is an executable specification system for agent-built software. It takes a declarative ` + "`maslow.yaml`" + ` spec, validates it against a schema, runs verification checks, and produces structured evidence reports.

## Core Commands

` + "```bash" + `
# Validate a spec file against the schema
maslow validate maslow.yaml

# Run verification checks (quick profile for dev, full before merge)
maslow verify --profile quick
maslow verify --profile full

# Scaffold a new maslow-managed project
maslow scaffold --name my-project

# Initialize maslow.yaml in an existing project
maslow init

# Install the agentic harness into an existing project
maslow harness install

# Update harness files to the latest version
maslow harness update

# Detach the harness to prevent future updates
maslow harness detach

# Print version info
maslow version
` + "```" + `

## Workflow: Starting a New Project

1. Run ` + "`maslow scaffold --name <project-name>`" + ` to generate the full project structure.
2. Read all refs in maslow.yaml that point to docs/ files — these are your requirements and context.
3. Edit ` + "`maslow.yaml`" + ` to define your checks, contracts, budgets, and policies.
4. Implement the project code.
5. Run ` + "`maslow verify --profile quick`" + ` frequently during development.
6. Run ` + "`maslow verify --profile full`" + ` before merging.

## Workflow: Working on an Existing Project

1. Read ` + "`maslow.yaml`" + ` to understand the project spec. Read all refs that point to docs/ files for context.
2. Read ` + "`CLAUDE.md`" + ` for agent conventions and operating principles.
3. Read ` + "`docs/MAP.md`" + ` for architecture overview.
4. Read ` + "`docs/PLAN.md`" + ` for milestones and execution plan.
5. Check ` + "`docs/tasks/`" + ` for available work (scan frontmatter only).
6. Run ` + "`maslow verify --profile quick`" + ` to confirm the project is green before making changes.
7. Make changes in small increments, running verification after each.

## Workflow: Greenfield Build

When starting from a vague goal (e.g., "Build a TikTok clone with web and mobile apps"):

1. Read ` + "`maslow.yaml`" + ` for project structure, packages, and any existing refs.
2. **Read all refs** that point to documentation — PRDs, requirements, branding guides, tech decision docs. These are your north star.
3. Start building. Make technology and design decisions as you go. Record each material decision as an ADR in ` + "`docs/adr/`" + `.
4. As features take shape, add corresponding verifications to ` + "`maslow.yaml`" + `: contracts for API endpoints, budgets for artifacts and performance, refs for new config files.
5. When you hit a verification or harness gap you can't work around, create a draft task in ` + "`docs/tasks/`" + ` tagged ` + "`kind:gap`" + ` describing what you need.
6. Run ` + "`maslow verify --profile quick`" + ` frequently. Keep it green.
7. Update ` + "`docs/MAP.md`" + ` as the architecture emerges.

**Key principle**: Don't block on decisions. Decide, record (ADR), build, verify. The human reviews ADRs and draft tasks at their own pace. They can always revert via git.

## maslow.yaml Structure

A valid ` + "`maslow.yaml`" + ` defines:

- **mas** - Schema version (e.g., "1.0")
- **project** - Project name, description, version
- **toolchain** - Required tools and version managers (asdf, mise, nix)
- **refs** - External references and generative input (docs, configs, APIs, MCP servers)
- **policy** - Path deny/protected lists for agent safety
- **checks** - Named verification checks with runner configuration
- **profiles** - Named subsets of checks (quick, full, custom)
- **contracts** - Scenario-based behavioral contracts (CLI and HTTP)
- **budgets** - Performance, size, and complexity limits
- **audit** - Black-box audit targets

### Minimal Example

` + "```yaml" + `
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
` + "```" + `

## Task System

Tasks in ` + "`docs/tasks/`" + ` are how humans inject work for agents.

### Discovering Tasks

1. List files matching ` + "`docs/tasks/[0-9]*_*.md`" + `
2. Read only the YAML frontmatter (between --- markers)
3. Filter for status: todo with empty assigned_to and no unresolved depends_on
4. Pick the lowest-ID matching task

### Claiming a Task

1. Set status: in_progress, assigned_to, and assigned_at in the frontmatter
2. Commit and push the claim
3. If push fails (conflict), pull and pick another task

### Completing a Task

1. Do the work, committing as you go
2. Set status: done in the frontmatter
3. Commit the status change

### Status Lifecycle

draft -> todo -> in_progress -> done (with blocked as a side state)

**Never** work on draft tasks. Only pick up todo tasks.

## Verification Evidence

Each run of ` + "`maslow verify`" + ` writes ` + "`reports/verify.json`" + ` containing:
- Timestamp, git SHA, profile used
- Per-check results, contract results, budget results
- Overall verdict: pass, fail, or inconclusive

The file is deterministic and machine-readable.

## Key Conventions

- Run ` + "`maslow verify --profile quick`" + ` after every significant change
- Run ` + "`maslow verify --profile full`" + ` before merging
- All error messages reference file paths and context
- Exit codes: 0 = success, 1 = validation/verification failure, 2 = usage error
- Keep maslow.yaml as the single source of truth for project verification
- Record material decisions as ADRs in docs/adr/
`
}

// GenerateGitignore generates the .gitignore content.
func GenerateGitignore() string {
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

// GenerateTaskConvention generates the docs/tasks/CONVENTION.md content.
func GenerateTaskConvention() string {
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

The assigned_to field should be a descriptive identifier, e.g., claude-<machine-hostname> or agent-<session-id>.

### Working on a Task

1. Read the full task body for requirements and acceptance criteria
2. Do the work in one or more commits (reference the task: task(<id>): <description>)
3. When done, update frontmatter: status: done, updated: <today>
4. Commit the status change: task(<id>): complete task <id> - <title>

### Handling Blocks

If the agent cannot proceed:
1. Set status: blocked with a note in the task body explaining why
2. Commit and push so other agents (or humans) can see the block
3. Move on to the next available task

## Prompt Patterns

Humans can invoke agents with these patterns:

- "read CLAUDE.md and implement the next task" — agent picks the lowest-ID todo task
- "read CLAUDE.md and implement task 5" — agent works on task 5 specifically
- "read CLAUDE.md and implement any task tagged infra" — agent filters by tag

## Creating New Tasks

Humans create tasks by:

1. Choosing the next available ID (one higher than the highest existing)
2. Creating docs/tasks/<id>_<SLUG>.md with frontmatter and body
3. Setting status: draft while iterating on the description
4. Changing to status: todo when ready for agent pickup
5. Committing and pushing

## Agent-Created Draft Tasks

Agents may create draft tasks to signal gaps they've discovered during work. These are NOT for decomposing work or asking permission — they are for flagging platform limitations.

**When agents should create a draft task:**
- A verification capability is missing (e.g., "contract runner doesn't support variable capture, can't test auth flows")
- A tool or MCP is needed but unavailable (e.g., "need browser MCP for visual testing")
- A harness limitation blocks the workflow

**Tag conventions for agent-created drafts:**
- ` + "`kind:gap`" + ` — verification or harness capability gap
- ` + "`kind:capability`" + ` — missing tool, MCP, or external access

**Format**: Same as human-created tasks, but always set status: draft. The human reviews and promotes to todo if the gap is worth addressing. Agents must never promote their own draft tasks.
`
}
