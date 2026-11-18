---
id: 29
title: Select contracts by the paths a change touches
status: draft
priority: medium
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, contracts, profiles, policy]
---

## Objective

A profile selects contracts by name or tag, whatever the change touched. A large suite then either
runs everything on every change (slow) or relies on someone choosing the right profile. Letting a
contract declare the paths that make it relevant would run the scenarios a diff can break, and
record why each ran.

## Requirements

- A contract may declare `when_changed: [patterns]`, using the policy path-matching rules (ADR 0017).
- With `--base`, a contract with `when_changed` runs only if a changed file matches; without
  `--base` it always runs, so local runs stay unchanged.
- Contracts without `when_changed` behave as today.
- Evidence records, per contract, whether it ran because of a match (and which file) or was
  skipped because nothing matched. A skipped contract does not make the verdict `inconclusive`.
- Overlay contracts ignore `when_changed`, as they already ignore profile selection (ADR 0029).

## Acceptance Criteria

- [ ] Schema and fixtures for `when_changed`.
- [ ] Tests cover match, no match, no `--base`, and an overlay contract.
- [ ] Evidence schema updated; MAP.md and README describe the field.
- [ ] ADR records how this combines with profile selection.

## Notes

Open question: should a profile be able to turn diff selection off (e.g. a nightly `full` run)?
Under `--spec-from base` the patterns come from the base spec, so a change cannot narrow them.
