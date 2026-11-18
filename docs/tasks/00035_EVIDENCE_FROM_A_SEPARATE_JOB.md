---
id: 35
title: Produce evidence outside the job that runs the code under test
status: draft
priority: medium
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: [32]
tags: [kind:gap, action, evidence, security]
---

## Objective

Today the pull request's code runs in the same job that writes the evidence, so it could write its
own evidence file. A consumer cannot tell a real pass from a forged one.

## Requirements

- A documented two-job pattern for the GitHub Action: one job runs checks and contracts against the
  pull request's code; a second job, which never runs that code, assembles and publishes the
  evidence.
- The published evidence is signed or attested (e.g. GitHub artifact attestations) by the job that
  did not run the code.
- `maslow evidence check` (task 32) can verify the attestation when asked.

## Acceptance Criteria

- [ ] `action/README.md` documents the pattern with a working example workflow.
- [ ] Maslow's own `ci.yml` uses it.
- [ ] ADR records the trust model and what the attestation does and does not prove.

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V4. This is the known limit noted in
the Maslow adoption plan.
