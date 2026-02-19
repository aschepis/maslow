Your mission is to build “Maslow” (a CLI + spec + harness) as defined in REQUIREMENTS.md, and then ensure Maslow v1 is bootstrapped and subsequently built/verified by Maslow itself.

Operating principles (must follow):

1. Repository knowledge is the system of record. Prefer adding/maintaining small, discoverable repo docs over long chat explanations. Create a “map” of the repo and keep it updated.
2. Humans steer, agents execute. You must ask questions when required, but you should default to making progress by scaffolding, implementing, running checks, and iterating with feedback loops.
3. Agent legibility is the goal: structure code/docs so an agent can reliably reason about it; minimize “YOLO” assumptions; prefer typed/validated boundaries and explicit conventions.
4. Encode “golden principles” (mechanical rules) into the repo and enforce them continuously (formatting, lint, directory conventions, invariants). Treat cleanup like garbage collection, not a one-off refactor.
5. Throughput changes merge philosophy: work in small PR-sized increments, keep changes narrow, and keep the main branch green.
6. Parallelize wherever possible: use all configured Claude agents / subagents. Partition workstreams by folder/concern and avoid collisions with explicit scopes.

Non-negotiable behaviors:

- You MUST NOT stop until ALL requirements in REQUIREMENTS.md are complete, including the self-hosting bootstrap constraint (Maslow verifies Maslow; Maslow audits its own binary).
- You MUST log and document everything: decisions, questions, chosen conventions, and current state. Keep this in-repo.
- If you encounter a large, unscoped question or unknown requirement, you MUST:
  (a) write a template doc into /docs/templates/ (or /docs/decision-templates/) with the questions and placeholders,
  (b) ask the user to fill it,
  (c) continue only on work that does not depend on the missing info.
- You MUST ask for tool access when necessary (shell commands, file system edits, git operations, network access, CI integration).
- You MUST run tests/checks frequently and use failures as feedback loops.
- You MUST keep an audit trail: each material decision is captured as an ADR (Architecture/Agent Decision Record) in /docs/adr/.

What to read first:

- REQUIREMENTS.md (source of truth)
- Any existing maslow.yaml (if present)
- Any existing schema.cue (if present)
- Any existing docs in /docs

Deliverables (by the end):
A) A working Maslow CLI with at least: validate, verify, audit, init, version
B) CUE schema and MAS spec format implemented
C) Deterministic reports/verify.json emitted by verify/audit
D) Black-box audit works without source checkout (binary/container/endpoint targets as per requirements)
E) Self-hosting: the Maslow repo has maslow.yaml that describes itself; `maslow verify --profile full` passes on itself; `maslow audit --profile full` passes on its own built artifact.
F) A repo “map” doc, ADRs, runbook, and contributor/agent workflow guidance (AGENTS.md / CLAUDE.md compatible)

Immediate next steps (do these now):

1. Ask for and confirm tool access you need:
   - permission to run shell commands (go, cue, git)
   - permission to create/edit files in this repo
   - permission to run tests
   - permission to create git branches/commits/worktrees
   - permission to access network (only if needed for installing tools/deps)
2. Load REQUIREMENTS.md and produce a short repo “MAP” in /docs/MAP.md:
   - architecture overview
   - key entrypoints
   - where schema/spec lives
   - where verify outputs go
3. Create an execution plan in /docs/PLAN.md:
   - milestones (Kernel → Verify → Audit → Init → Self-hosting)
   - parallel workstreams and their file scopes
   - risk list + mitigation
4. Create ADR-0001: “Why CUE for schema; why YAML for spec”
5. Implement the smallest possible kernel that can validate maslow.yaml against schema.cue:
   - schema.cue minimal
   - cmd/maslow/main.go implementing `maslow validate`
   - testdata valid/invalid fixtures
   - go test target that exercises validate
6. Iterate depth-first: each time you hit a missing capability, encode it as:
   - a repo doc,
   - a check,
   - or a harness improvement.

Parallel workstreams (spin these up immediately with subagents):

- Agent A (Schema): define schema.cue incrementally; add internal consistency checks (profiles reference existing check kinds, required refs exist, etc.)
- Agent B (CLI Kernel): implement validate + plumbing for verify/audit; focus on exit codes, error messages, deterministic output
- Agent C (Harness/Evidence): define reports/verify.json structure + deterministic emission; implement verify runner orchestration
- Agent D (Docs/Legibility): MAP.md, PLAN.md, runbook, ADR templates, AGENTS.md/CLAUDE.md guidance
- Agent E (Audit/Black-box): audit target execution (binary/container/endpoint), no source checkout; evidence collection

Implementation constraints:

- Prefer Go for the CLI.
- Prefer simple, explicit code over frameworks.
- Prefer convention over configuration, but every convention must be documented and validated.

When you ask questions, format them as:

- “Blocking questions” (must answer to proceed)
- “Non-blocking questions” (defaults you’ll assume unless user overrides)
  If non-blocking, state your default and proceed.

Start now by requesting the necessary tool permissions, then read REQUIREMENTS.md and begin executing the plan.
