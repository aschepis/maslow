---
id: 49
title: Run each check and CLI step in its own process group and kill it when done
status: todo
priority: medium
created: 2026-11-20
updated: 2026-11-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, checks, contracts, security]
---

## Objective

Split from task 45 as cheap hardening, not a trust fix. Today a check or CLI contract step runs with
`exec.CommandContext` and `CombinedOutput` (`internal/runner/runner.go:106-107`,
`internal/runner/contract.go:620-626`), with no process group. A timeout kills only the direct
child. A background child that keeps the output pipe open hangs the step past its timeout, and one
that redirects its output outlives the check and keeps running through the rest of the verify run
and the evidence upload.

## Requirements

- On Unix, each check and CLI step runs in a new process group. When it exits or times out, Maslow
  kills the whole group.
- A step cannot hang past its timeout because a descendant holds its output open (`WaitDelay`).
- Windows keeps today's behaviour, behind a build tag.
- The evidence and verdict rules are unchanged, except that a timeout caused by a lingering
  descendant now ends the step as a timeout instead of hanging.

## Acceptance Criteria

- [ ] Tests: a check that backgrounds a sleeping child leaves no process behind; a check whose
      background child holds stdout open ends at its timeout; a normal check is unaffected.
- [ ] Same for CLI contract steps.
- [ ] ADR records what this does and does not stop (a process that calls `setsid` or uses sudo
      escapes the group; real isolation is task 45).

## Notes

Decided with the human on 2026-11-20 when reviewing the draft gap tasks.
