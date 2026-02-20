# Task Convention

This document defines the format, lifecycle, and multi-agent protocol for tasks stored in `docs/tasks/`.

## Purpose

Tasks are the mechanism for humans to inject work into the project via git and for agents to discover, claim, and execute that work. The system is designed for:

- **Human authoring**: humans write tasks in markdown, store drafts, and promote to `todo` when ready.
- **Agent discovery**: agents scan frontmatter only (not full body) to find actionable work.
- **Multi-agent safety**: claiming protocol prevents duplicate work across clones.

## File Format

Each task is a markdown file with YAML frontmatter:

```markdown
---
id: 3
title: Short imperative title
status: todo
priority: medium
created: 2026-02-19
updated: 2026-02-19
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [area-x, area-y]
---

## Objective

What needs to be done and why.

## Requirements

- Requirement 1
- Requirement 2

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Notes

Any additional context, links, or design considerations.
```

### Frontmatter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | integer | yes | Unique task ID, matches filename prefix |
| `title` | string | yes | Short imperative description (< 80 chars) |
| `status` | enum | yes | Current lifecycle state (see below) |
| `priority` | enum | no | `critical`, `high`, `medium`, `low` (default: `medium`) |
| `created` | date | yes | ISO 8601 date when task was created |
| `updated` | date | yes | ISO 8601 date of last status change |
| `assigned_to` | string | no | Identifier of the agent/human working on it |
| `assigned_at` | datetime | no | ISO 8601 timestamp when assignment happened |
| `depends_on` | list[int] | no | IDs of tasks that must complete first |
| `tags` | list[string] | no | Categorization tags |

### Naming Convention

Files are named `<id>_<SLUG>.md` where:
- `<id>` is a sequential integer (zero-padded not required but allowed)
- `<SLUG>` is an uppercase snake_case summary
- Example: `3_ADD_CACHING_LAYER.md`

The special file `CONVENTION.md` (this file) is not a task.

## Status Lifecycle

```
draft ──> todo ──> in_progress ──> done
                       │
                       └──> blocked ──> in_progress
```

| Status | Meaning | Who sets it |
|--------|---------|-------------|
| `draft` | Work in progress by the author; not ready for agents | Human |
| `todo` | Ready to be picked up; requirements are complete | Human |
| `in_progress` | Actively being worked on by an assigned agent | Agent |
| `blocked` | Cannot proceed; dependency or question unresolved | Agent |
| `done` | All acceptance criteria met; work is complete | Agent |

Rules:
- **Humans** control `draft` → `todo` transitions. Agents must never start work on `draft` tasks.
- **Agents** control `todo` → `in_progress` → `done` transitions.
- Only tasks with `status: todo` and empty `assigned_to` are available for pickup.
- Tasks with unresolved `depends_on` (dependencies not in `done` status) should not be started.

## Agent Protocol

### Discovering Tasks

Agents should scan task files by reading only frontmatter (first ~15 lines). The protocol:

1. List files in `docs/tasks/` matching `[0-9]*_*.md`
2. For each file, read the frontmatter block (between `---` markers)
3. Filter for `status: todo` with empty `assigned_to` and no unresolved dependencies
4. Select a task based on priority and ID order (lower ID = older = prefer first)

### Claiming a Task

To prevent duplicate work across agents on different clones:

1. **Pull latest**: `git pull --rebase` before claiming
2. **Check status**: re-read the frontmatter to confirm still `todo` and unassigned
3. **Update frontmatter**: set `status: in_progress`, `assigned_to: <agent-id>`, `assigned_at: <now>`, `updated: <today>`
4. **Commit and push**: commit the status change with message `task(<id>): claim task <id> - <title>`
5. **If push fails** (conflict): pull, check if someone else claimed it, and move to the next task

The `assigned_to` field should be a descriptive identifier, e.g., `claude-<machine-hostname>` or `agent-<session-id>`.

### Working on a Task

1. Read the full task body for requirements and acceptance criteria
2. Do the work in one or more commits (reference the task: `task(<id>): <description>`)
3. When done, update frontmatter: `status: done`, `updated: <today>`
4. Commit the status change: `task(<id>): complete task <id> - <title>`

### Handling Blocks

If the agent cannot proceed:
1. Set `status: blocked` with a note in the task body explaining why
2. Commit and push so other agents (or humans) can see the block
3. Move on to the next available task

## Prompt Patterns

Humans can invoke agents with these patterns:

- `read CLAUDE.md and implement the next task` — agent picks the lowest-ID `todo` task
- `read CLAUDE.md and implement task 5` — agent works on task 5 specifically
- `read CLAUDE.md and implement any task tagged infra` — agent filters by tag

## Creating New Tasks

Humans create tasks by:

1. Choosing the next available ID (one higher than the highest existing)
2. Creating `docs/tasks/<id>_<SLUG>.md` with frontmatter and body
3. Setting `status: draft` while iterating on the description
4. Changing to `status: todo` when ready for agent pickup
5. Committing and pushing

Agents may also create tasks (e.g., when discovering follow-up work), but should set them to `draft` and note that human review is needed before promotion to `todo`.
