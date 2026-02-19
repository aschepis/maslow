# ADR 0003: Check Runner Abstraction Design

- **Status**: Accepted
- **Date**: 2026-02-19

## Context

Maslow must execute various check types with timeouts and collect structured results for evidence. Checks may be shell commands, executable scripts, or built-in Go functions. The runner must be predictable enough that developers can reason about execution order and debug failures without special tooling.

Requirements:

- Support multiple check kinds without leaking kind-specific logic into callers
- Capture structured output (stdout, stderr, duration, exit code) for every check
- Enforce per-check timeouts so a single hanging check cannot block the entire run
- Allow profiles to select a subset of checks without duplicating check definitions
- Produce results in the evidence format defined in ADR 0002

Options considered for execution model:

| Concern | Options Considered |
|---------|--------------------|
| Execution order | Sequential, parallel, dependency-graph |
| Check kinds | Command only, command + script, command + script + builtin |
| Profile filtering | Separate check lists per profile, tag-based filtering, name-based filtering |

## Decision

**Check definition schema**

Checks are defined in `maslow.yaml` under `checks.runners` as a list of objects with the following fields:

- `name`: unique string identifier for the check
- `kind`: one of `command`, `script`, or `builtin`
- `run`: the command string, script path, or builtin function name to execute
- `timeout`: duration string (e.g., `30s`); defaults to `60s` if omitted
- `tags`: optional list of strings used for filtering and documentation

**Check kinds**

- `command`: the `run` field is passed to the system shell (`sh -c`); exit code 0 is `pass`, non-zero is `fail`
- `script`: the `run` field is a path relative to the repository root; the file is executed directly; same exit code mapping as `command`
- `builtin`: the `run` field names a registered Go function in the `internal/runner` package; the function returns a structured result directly without shell invocation

**Execution model**

The runner package executes checks sequentially in the order they appear in `maslow.yaml`. For each check it:

1. Starts a context with the configured timeout
2. Executes the check via its kind handler
3. Captures combined stdout and stderr
4. Records wall-clock duration
5. Maps exit code or function result to `pass`, `fail`, `skip`, or `error` status
6. Appends a `CheckResult` to the results slice

**Profile filtering**

Profiles are defined in `maslow.yaml` under `profiles` as a list of `{name, checks}` objects. The `checks` field is a list of check names. When a profile is active, only the named checks are executed; all others are omitted from the results entirely (not recorded as `skip`).

## Consequences

**Positive**

- Sequential execution is deterministic and easy to trace; log output appears in check order
- Per-check timeouts prevent hangs from blocking CI pipelines
- Three kinds cover all current use cases without over-engineering
- Profile filtering by name is explicit; there is no ambiguity about which checks run
- The runner package has no knowledge of the evidence format; it returns a plain slice of results

**Negative**

- Sequential execution is slower than parallel for independent checks; a repository with ten 10-second checks takes 100 seconds minimum
- There is no dependency graph between checks; a check that depends on the output of a previous check must use a script that encodes that dependency internally
- Builtin checks require Go code changes to add new behaviors; they cannot be extended from `maslow.yaml` alone

**Mitigations**

- Checks should be kept fast (under 10 seconds each); slow checks indicate a smell in the check design, not a runner limitation
- Parallel execution is recorded as a future optimization; the sequential interface is forward-compatible because results are collected into a slice
- The builtin registry is small and stable; new behaviors that require flexibility should use `command` or `script` kind instead
