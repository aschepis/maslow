# ADR 0015: Scaffold Post-Creation Workflow

## Context

The scaffold command's "Next steps" output originally directed users to "edit maslow.yaml" and "run maslow validate" — generic instructions that didn't teach the intended workflow. Users (especially agents) need to understand that refs are the steering mechanism and tasks are the execution mechanism.

## Decision

Update scaffold next-steps to prescribe the full workflow: (1) git init, (2) write docs as refs, (3) edit maslow.yaml, (4) write first task, (5) tell agent to read CLAUDE.md and implement. The README mirrors this guidance.

## Consequences

- New project onboarding is now opinionated toward the refs-and-tasks workflow
- Agents receiving a freshly scaffolded project get clear guidance on where to start
- The README and scaffold output are aligned, reducing confusion
