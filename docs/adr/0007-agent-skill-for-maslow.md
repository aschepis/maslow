# ADR-0007: Agent Skill for Maslow

## Status

Accepted

## Context

Maslow is designed for agent-driven software development. For agents to effectively use Maslow to define, build, and verify software, they need structured instructions that can be discovered and loaded automatically. The [Agent Skills specification](https://agentskills.io/specification) defines a standard format for packaging such instructions.

Key considerations:
- Agents need to know Maslow's core commands, workflows, spec format, and task protocol
- The skill should be auto-discoverable by agent runtimes that support the Agent Skills spec
- New projects scaffolded by `maslow scaffold` should include the skill out of the box
- The skill must follow progressive disclosure: lightweight metadata for discovery, full instructions on activation

## Decision

We adopt the Agent Skills specification and place the maslow skill at `.skills/maslow/SKILL.md` in every Maslow-managed project.

### Placement: `.skills/` directory

We chose `.skills/` as the parent directory because:
- It is a dotfile directory, keeping the project root clean
- It clearly signals "agent skills live here" without conflicting with project source directories
- It allows multiple skills to coexist (e.g., `.skills/maslow/`, `.skills/deploy/`)

### Content scope

The skill covers:
- Core CLI commands (validate, verify, scaffold, init)
- New project and existing project workflows
- `maslow.yaml` structure with a minimal example
- Task system discovery, claiming, and completion protocol
- Verification evidence format
- Key conventions (exit codes, determinism, ADRs)

### Propagation via scaffold

The `maslow scaffold` command generates `.skills/maslow/SKILL.md` in every new project, consistent with the Harness Propagation Rule in CLAUDE.md. The skill content is maintained in `internal/scaffold/scaffold.go` alongside the other generated files.

## Consequences

- Agents using runtimes that support Agent Skills will automatically discover Maslow's workflow instructions.
- The `.skills/maslow/SKILL.md` file must be kept in sync between this repo and the scaffold generator (Harness Propagation Rule).
- Future improvements to agent instructions should update both locations.
- Other domain-specific skills can be added to `.skills/` without conflicting with the maslow skill.
