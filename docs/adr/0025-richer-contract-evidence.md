# ADR 0025: Richer Contract Evidence

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

A contract result carried only a name, a status and an error string. A reader of the evidence could not tell which step failed, what the system under test printed or returned, or where a project's own browser tests left their screenshots.

## Decision

A failing contract result records `failed_step` (phase, zero-based index, action and the substituted target) and the output of the most recent `cli`, `http` or `poll` step, capped like check output; a scenario may list repository-relative `artifacts` patterns, and every matching file is listed on the result whatever its status.

## Consequences

- Artifacts are collected after the scenario's steps and before its teardown, so teardown can clean up without losing the record.
- Artifact patterns use the policy matcher (`**` recurses); a malformed pattern turns a passing scenario into `error`.
- The evidence lists artifact paths, not their contents; storing the files is the caller's job, as with the report itself.
- The fields are additive and optional, so the format stays at version 2.
