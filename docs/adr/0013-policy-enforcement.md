# ADR-0013: Policy Enforcement

## Context

Policy rules (`deny`, `protected`) in maslow.yaml were parsed into Go types but never evaluated at runtime. A deny pattern matching existing files or a protected file with uncommitted changes would not be detected during verification.

## Decision

Implement a policy enforcement runner (`internal/runner/policy.go`). Deny rules use `filepath.Glob` to check if matching files exist (fail if any do). Protected rules use `git diff` and `git status` to detect uncommitted modifications (fail if modified). `gated` and `allow` rules are deferred to a future version — gated requires owner-based authorization semantics and allow needs clarification on whether it's an allowlist or exception mechanism.

## Consequences

- Deny patterns now actively prevent forbidden files from existing in the repo
- Protected files are verified as unmodified, preventing accidental changes to critical config
- Policy failures cause the verdict to fail, making maslow a real safety guardrail
- Git dependency: protected file checks require a git repository; non-git repos will get error results
- `gated` directories and `allow` rules are not yet enforced — they return skip status
