# ADR 0018: Judge Policy Against a Base Ref

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

`protected` compared the working tree with `HEAD`, so it only saw uncommitted edits. On a branch where the edit to a protected file was already committed, which is every pull request, the rule passed. A gate has to judge the whole change, not the last uncommitted part of it.

## Decision

`maslow verify --base <ref>` computes the merge base of `<ref>` and `HEAD` and judges change-based policy on every file changed since it (committed, uncommitted and untracked); without `--base` the working-tree behaviour stays.

## Consequences

- A pipeline passes the pull request's base SHA and a committed change to a protected file fails. Deletions count as changes; renames count as both paths.
- The evidence records `base_sha`, and each failing rule lists the files it matched.
- Git operations live in `internal/gitrepo`, so the next step (loading the spec from the base commit) reuses them.
