# Maslow

An agentic harness and executable specification system for building verifiable software.

Maslow takes a declarative `maslow.yaml` spec, validates it against a schema, runs verification checks (tests, contracts, budgets, ref verification, policy enforcement), and produces structured evidence reports. It also scaffolds a complete agentic harness so AI coding agents can build your project autonomously.

## Install

```bash
go install github.com/aschepis/maslow-agentic/cmd/maslow@latest
```

Or build from source:

```bash
git clone https://github.com/aschepis/maslow-agentic.git
cd maslow-agentic
make build
```

## Creating a New Project

```bash
maslow scaffold my-project
cd my-project
git init && git add -A && git commit -m 'initial scaffold'
```

This creates the full project structure:

| File | Purpose |
|------|---------|
| `maslow.yaml` | Project spec — checks, contracts, budgets, refs, policy |
| `CLAUDE.md` | Agentic harness — operating principles, conventions, workflows |
| `docs/MAP.md` | Architecture overview (updated as the project evolves) |
| `docs/PLAN.md` | Milestones and execution plan |
| `docs/tasks/` | Task queue for agents |
| `.skills/maslow/` | Agent skill for maslow commands |

### After scaffolding

1. **Write your intent as docs and add them as refs.** Create `docs/REQUIREMENTS.md` (or `docs/VISION.md`, `docs/PRD.md` — whatever fits) describing what the project should do. Then reference it in `maslow.yaml`:

   ```yaml
   refs:
     - kind: file
       path: docs/REQUIREMENTS.md
       description: Product requirements
       required: true
   ```

   Refs are the agent's reading list. The more context you put in `docs/`, the better the agent's decisions will be.

2. **Edit `maslow.yaml`.** Fill in project name, description, tech constraints, policy rules. You don't need checks or contracts yet — those get added progressively as features take shape.

3. **Write the first task.** Create `docs/tasks/00001_YOUR_FIRST_TASK.md` with `status: todo`. Describe the first concrete deliverable — not the whole project. Think "scaffold the CLI entrypoint" or "implement the conversation parser", not "build everything".

4. **Tell your agent:**

   > Read CLAUDE.md and implement the next task

The agent reads refs for context, picks up the task, makes decisions (recorded as ADRs), builds, and runs `maslow verify` to confirm everything works.

### How it works

- **`maslow.yaml` refs are your steering wheel** — put vision, constraints, and requirements in docs/ files referenced by refs
- **Tasks are your gas pedal** — sequence the build incrementally via `docs/tasks/`
- **Agents decide and record** — technology choices, architecture, design are captured as ADRs in `docs/adr/`
- **Verification is continuous** — `maslow verify` runs checks, contracts, budgets, ref verification, and policy enforcement

## Commands

```bash
maslow scaffold <name>           # Create a new maslow-managed project
maslow init                      # Add maslow.yaml to an existing project
maslow validate <file>           # Validate a maslow.yaml against the schema
maslow verify --profile <name>   # Run verification checks
maslow audit --profile <name>    # Run black-box audit
maslow harness install           # Install the agentic harness into a project
maslow harness update            # Update harness files to latest version
maslow harness detach            # Detach harness from future updates
maslow version                   # Print version info
```

## Verification Pipeline

`maslow verify` runs the following in order:

1. **Checks** — shell commands (build, test, lint) filtered by profile
2. **Contracts** — scenario-based behavioral tests (CLI and HTTP)
3. **Budgets** — artifact size, performance, and complexity limits
4. **Refs** — verify declared files, binaries, and URLs exist
5. **Policy** — enforce deny patterns and protected file rules

Results are written to `reports/verify.json` as structured evidence.

## Development

```bash
make build      # Build the binary
make test       # Run all tests
make vet        # Run go vet
make lint       # Run go vet + cue vet
make validate   # Validate maslow.yaml
make verify     # Run full verification
make clean      # Remove build artifacts
```
