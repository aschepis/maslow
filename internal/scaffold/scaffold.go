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
		filepath.Join(dir, "reports"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("scaffold: failed to create directory %s: %w", d, err)
		}
	}

	// Generate all files.
	files := map[string]string{
		filepath.Join(dir, "maslow.yaml"):     generateMaslowYAML(opts.ProjectName, description, opts.Toolchain),
		filepath.Join(dir, "CLAUDE.md"):        generateClaudeMD(opts.ProjectName),
		filepath.Join(dir, "docs", "MAP.md"):   generateMapMD(opts.ProjectName),
		filepath.Join(dir, "docs", "PLAN.md"):  generatePlanMD(opts.ProjectName),
		filepath.Join(dir, ".gitignore"):       generateGitignore(),
		filepath.Join(dir, "reports", ".gitkeep"): "",
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
| reports/ | Generated verification output (gitignored) |

## Conventions

- Deterministic output required: the same input must always produce the same result
- All error messages must reference file paths and relevant context
- Exit codes: 0 = success, non-zero = failure

## Verification

Run verification frequently:

` + "```" + `bash
# Quick checks during development
maslow verify --profile quick

# Full checks before merging
maslow verify --profile full

# Validate the spec itself
maslow validate maslow.yaml
` + "```" + `

## Adding a New Feature

1. Check docs/PLAN.md for the relevant milestone
2. Update maslow.yaml if checks, contracts, or budgets change
3. Implement the feature
4. Add or update tests
5. Run maslow verify --profile quick
6. Write an ADR in docs/adr/ if the change involves a material decision
7. Update docs/MAP.md if architecture changed
8. Commit narrowly with focused messages
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
