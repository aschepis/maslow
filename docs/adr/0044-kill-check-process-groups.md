# ADR 0044: Kill Each Check's Process Group When It Ends

- **Status**: Accepted
- **Date**: 2026-11-20

## Context

Checks and CLI contract steps ran with `exec.CommandContext` and `CombinedOutput`, so a timeout killed only the direct child. A background child that kept the output pipe open hung the step past its timeout, and one that redirected its output outlived the step and kept running through the rest of the verify run and the evidence upload (task 49, split from task 45).

## Decision

On Unix, each check and CLI step (including contract setup and teardown steps) runs in a new process group; a timeout kills the whole group, `WaitDelay` (3s) bounds the wait for output held open by a descendant, and when the step ends, normally or not, Maslow kills whatever is left in the group.

## Consequences

- A contract's `setup` can no longer leave a server running for its scenarios: the server is killed when its setup step ends. A spec that backgrounded a server in setup must start it outside Maslow, or in the same step as the commands that use it.
- A step whose descendant holds the output open still ends: at its timeout as a failure if the command is still running, or 3s after the command exits as a failure saying the output was held open. Verdict and evidence rules are otherwise unchanged.
- This is hardening, not isolation. A process that calls `setsid`, double-forks into a new session, or runs under `sudo` escapes the group and survives; real isolation is task 45. Windows keeps the old behaviour (a no-op helper behind a build tag).
- Steps no longer share Maslow's terminal process group, so Ctrl-C stops Maslow but not the step running at that moment, which runs on until it exits. Maslow has no signal handling to forward it yet.
