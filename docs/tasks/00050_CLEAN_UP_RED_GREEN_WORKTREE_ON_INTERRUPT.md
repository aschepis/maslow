---
id: 50
title: Remove the --red-green base worktree when verify is interrupted
status: todo
priority: low
created: 2026-11-20
updated: 2026-11-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, red-green, cli]
---

## Objective

`verify --red-green` checks the base commit out into a temporary git worktree
(`checkoutBase` in `internal/verify/redgreen.go`, under `$TMPDIR/maslow-base-*`) and removes it with
`baseTree.remove` when the run finishes. Since #44, SIGINT and SIGTERM make Maslow kill the running
steps and `os.Exit` (`cmd/maslow/interrupt.go`), so `remove` never runs. The directory is left in
the temp dir and the repository keeps a stale entry in `git worktree list` until
`git worktree prune` runs. This predates #44: before it the signal killed Maslow outright.

## Requirements

- An interrupted `verify --red-green` removes its base worktree and temp directory before exiting.
- The interrupt still exits 130 or 143 promptly and writes no evidence (ADR 0044).
- Cleanup that fails, or takes longer than a couple of seconds, must not block the exit.

## Acceptance Criteria

- [ ] A test interrupts the built binary during a `--red-green` base run and finds no `maslow-base-*`
      directory left and no extra entry in `git worktree list`.
- [ ] A normal `--red-green` run still removes the worktree as today.

## Notes

One approach: a small cleanup registry beside `runner.KillRunning` that the interrupt handler runs
before exiting, with a deadline. Requested by the human on 2026-11-20 after #44.
