---
id: 34
title: Add shrink-only files and test-integrity checks to policy
status: in_progress
priority: high
created: 2026-11-18
updated: 2026-11-18
assigned_to: "claude-workflow"
assigned_at: "2026-11-18T20:24:43Z"
depends_on: []
tags: [kind:gap, policy]
---

## Objective

Deleting a failing test, adding a skip or lowering a threshold is the cheapest way for an agent to
turn a build green. Some files, such as an authorization baseline, should only ever shrink, but
today that is enforced by a test the change itself can edit.

## Requirements

- A `shrink_only` policy list: a change may remove lines from a matching file but not add any.
- A test-integrity rule that flags, in the diff against the base: deleted test files, removed test
  functions, newly added skip markers, and lowered numeric thresholds in configured files.
- Skip markers and test file patterns are configurable per language, with defaults for Go, Ruby,
  JavaScript and Python.
- Findings appear in the policy section of the evidence with the files and lines involved.

## Acceptance Criteria

- [ ] Schema and valid/invalid fixtures for both rules.
- [ ] Tests: added line in a shrink-only file fails; removed line passes; a new skip is flagged.
- [ ] ADR records whether a finding is `fail` or `needs_review` by default.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V3. Pairs with task 31: both rules
are likely to need observe mode before they enforce.
