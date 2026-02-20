---
id: 1
title: Bootstrap Maslow CLI and self-hosting system
status: done
priority: critical
created: 2026-02-19
updated: 2026-02-19
assigned_to: claude-bootstrap
assigned_at: "2026-02-19T00:00:00Z"
depends_on: []
tags: [bootstrap, core]
---

## Objective

Build "Maslow" (a CLI + spec + harness) as defined in REQUIREMENTS.md, and then ensure Maslow v1 is bootstrapped and subsequently built/verified by Maslow itself.

## Operating Principles (must follow)

1. Repository knowledge is the system of record. Prefer adding/maintaining small, discoverable repo docs over long chat explanations. Create a "map" of the repo and keep it updated.
2. Humans steer, agents execute. You must ask questions when required, but you should default to making progress by scaffolding, implementing, running checks, and iterating with feedback loops.
3. Agent legibility is the goal: structure code/docs so an agent can reliably reason about it; minimize "YOLO" assumptions; prefer typed/validated boundaries and explicit conventions.
4. Encode "golden principles" (mechanical rules) into the repo and enforce them continuously (formatting, lint, directory conventions, invariants). Treat cleanup like garbage collection, not a one-off refactor.
5. Throughput changes merge philosophy: work in small PR-sized increments, keep changes narrow, and keep the main branch green.
6. Parallelize wherever possible: use all configured Claude agents / subagents. Partition workstreams by folder/concern and avoid collisions with explicit scopes.

## Non-Negotiable Behaviors

- You MUST NOT stop until ALL requirements in REQUIREMENTS.md are complete, including the self-hosting bootstrap constraint (Maslow verifies Maslow; Maslow audits its own binary).
- You MUST log and document everything: decisions, questions, chosen conventions, and current state. Keep this in-repo.
- If you encounter a large, unscoped question or unknown requirement, you MUST:
  (a) write a template doc into /docs/templates/ (or /docs/decision-templates/) with the questions and placeholders,
  (b) ask the user to fill it,
  (c) continue only on work that does not depend on the missing info.
- You MUST ask for tool access when necessary (shell commands, file system edits, git operations, network access, CI integration).
- You MUST run tests/checks frequently and use failures as feedback loops.
- You MUST keep an audit trail: each material decision is captured as an ADR (Architecture/Agent Decision Record) in /docs/adr/.

## Deliverables

- A) A working Maslow CLI with at least: validate, verify, audit, init, version
- B) CUE schema and MAS spec format implemented
- C) Deterministic reports/verify.json emitted by verify/audit
- D) Black-box audit works without source checkout (binary/container/endpoint targets as per requirements)
- E) Self-hosting: the Maslow repo has maslow.yaml that describes itself; `maslow verify --profile full` passes on itself; `maslow audit --profile full` passes on its own built artifact.
- F) A repo "map" doc, ADRs, runbook, and contributor/agent workflow guidance (AGENTS.md / CLAUDE.md compatible)

## Acceptance Criteria

- [x] Maslow CLI with validate, verify, audit, scaffold, init, version commands
- [x] CUE schema and YAML spec format working
- [x] Deterministic reports/verify.json
- [x] Black-box audit mode
- [x] Self-hosting maslow.yaml
- [x] Documentation: MAP.md, PLAN.md, ADRs, CLAUDE.md
