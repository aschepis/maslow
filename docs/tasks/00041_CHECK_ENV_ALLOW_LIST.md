---
id: 41
title: Run checks with a scrubbed environment and an env allow-list
status: draft
priority: low
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, checks, security]
---

## Objective

A check command is the change's own code. It should not see tokens that happen to be present in
the CI job. Contracts already run with a scrubbed environment and an `env` allow-list (ADR 0026);
checks do not.

## Requirements

- Checks run with a minimal environment (e.g. `PATH`, `HOME`, `TMPDIR`) plus the names listed in
  the check's `env` field.
- Values of allow-listed variables are redacted from check output in the evidence, as for contracts.
- A spec-level opt-out exists for adopters who need the old behaviour while migrating.

## Acceptance Criteria

- [ ] Schema and fixtures for `env` on checks.
- [ ] Tests: an unlisted variable is absent; a listed one is present and redacted.
- [ ] `maslow.yaml` updated so Maslow's own checks still pass; ADR records the default and opt-out.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V10. This is a breaking change for
checks that rely on inherited variables, so it needs a release note.
