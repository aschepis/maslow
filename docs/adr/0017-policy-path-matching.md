# ADR 0017: Policy Path Matching

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

Policy rules used `filepath.Glob`, which treats `**` like `*`, so `**/.env` missed `a/b/.env` and `vendor/**` missed nested files. `deny` also matched any file on disk, so a gitignored local `.env` would fail every developer machine once `**` recursed. `protected` passed each entry to `git diff` as a pathspec, a third set of semantics.

## Decision

All policy kinds use one matcher over repository-relative, slash-separated paths, where a `**` segment matches zero or more whole segments and other segments follow `path.Match`. `deny` applies to tracked files plus untracked files that are not ignored.

## Consequences

- Patterns mean the same thing in `deny`, `protected`, and later `gated` and `allow`, and the evidence lists the files each failing rule matched.
- An ignored file never fails `deny`. A new, unignored file does, because one `git add -A` would commit it. This is stricter than "tracked files only", on purpose.
- Patterns are anchored at the repository root. `.env` matches only the root file; use `**/.env` for any depth.
