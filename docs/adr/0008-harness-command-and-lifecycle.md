# ADR-0008: Harness Command and Lifecycle

## Status

Accepted

## Context

The `maslow scaffold` command generates a full project structure including an "agentic harness" (CLAUDE.md, docs/ templates, task convention, skill file, etc.). However, once scaffolded, there is no way to:

1. Install the harness into an existing project that wasn't scaffolded by Maslow
2. Update harness files when Maslow ships improvements
3. Opt out of future harness updates while keeping a customized version

This limits the harness's usefulness to greenfield projects and creates a divergence problem where older projects fall behind on best practices.

## Decision

We introduce `maslow harness` with three subcommands: `install`, `update`, and `detach`.

### Separate `internal/harness/` package

The harness package is separate from `internal/scaffold/` because scaffold is for greenfield project creation (creates maslow.yaml and all files at once), while harness manages a subset of those files through an ongoing lifecycle. The harness package imports scaffold's exported generator functions to avoid content duplication.

### Harness file manifest

The harness manages a specific subset of scaffold output. Notably, `maslow.yaml` is NOT a harness file — it is managed by `maslow init` and contains project-specific configuration.

Harness files: `CLAUDE.md`, `docs/MAP.md`, `docs/PLAN.md`, `docs/tasks/CONVENTION.md`, `.skills/maslow/SKILL.md`, `.gitignore`, `reports/.gitkeep`

### Detach sentinel: `.maslow-harness-detached`

A simple file in the project root that blocks `update` and `install` unless `--force` is used. This enables teams to fork the harness for their own conventions without accidental overwrites.

### Conflict resolution

When a harness file conflicts with an existing file, the user is prompted with four options:

- **abort**: Stop the operation immediately
- **overwrite**: Replace the existing file
- **skip**: Leave the existing file untouched
- **rename-and-reference**: Rename existing to `<name>.original`, write new file with a note referencing the original

The `--force` flag bypasses prompts (overwrite all). The `--dry-run` flag skips all writes. EOF on stdin triggers abort.

### "Rename and reference" strategy

This preserves the user's existing agent instructions (which may contain project-specific context) alongside the latest harness template. The new file includes a comment pointing to the original so the user can merge content at their convenience.

## Consequences

- Existing projects can adopt the Maslow harness without re-scaffolding
- Harness improvements ship to all projects via `maslow harness update`
- Teams that customize the harness can detach to prevent accidental overwrites
- The scaffold generators are now exported, making them reusable by both scaffold and harness packages
- The sentinel file pattern is simple and git-friendly (can be committed or gitignored per team preference)
