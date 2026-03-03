---
id: 26
title: Implement policy enforcement runner
status: todo
priority: critical
created: 2026-03-01
updated: 2026-03-01
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, policy, runner]
---

## Objective

Policy rules (deny, protected) in maslow.yaml are parsed into Go types but never evaluated at runtime. Implement a policy enforcement runner that checks deny patterns don't match existing files and protected files haven't been modified.

## Requirements

- `RunPolicy(policy *spec.Policy) []evidence.PolicyResult` orchestrates all policy checks
- Deny rules: use `filepath.Glob` to check if matching files exist; fail if any do
- Protected rules: check `git diff --name-only HEAD -- <path>` and `git status --porcelain <path>`; fail if file has uncommitted modifications
- Gated dirs: skip for v1 (document in ADR)
- Allow rules: skip for v1 (semantics need clarification)
- Nil or empty policy → empty results
- Results included in evidence report and affect verdict

## Acceptance Criteria

- [ ] Deny patterns that match existing files → fail
- [ ] Deny patterns with no matches → pass
- [ ] Protected files with uncommitted changes → fail
- [ ] Protected files unchanged → pass
- [ ] Nil policy → empty results
- [ ] Results appear in verify report JSON
- [ ] Failing policy causes verdict to fail
- [ ] Unit tests cover all cases
