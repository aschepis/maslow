---
id: 15
title: Add fixture setup and teardown for contract scenarios
status: draft
priority: high
created: 2026-02-20
updated: 2026-02-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, contract, schema]
---

## Objective

Real contract testing requires known state. A "list videos" contract needs videos to exist. A "delete user" contract needs a user to delete. Today there's no way to express setup/teardown — contracts assume external state management.

The MaslowTok aspirational spec originally had fixtures with `/__test/seed` and `/__test/reset` endpoints, plus a `given` block per scenario. The schema needs a way to express this.

## Requirements

- Add a `setup` block to contracts and/or scenarios that runs before the scenario steps
  - Setup steps use the same step types (http, cli, wait, etc.)
  - Common pattern: POST to a test seed endpoint, or run a CLI command to load fixtures
- Add a `teardown` block that runs after scenario steps (even on failure)
  - Common pattern: POST to a test reset endpoint, or run cleanup commands
- Setup/teardown can be defined at the contract level (runs once for all scenarios) or scenario level (per scenario)
- Update the CUE schema and embedded copy
- Update the Go types and contract runner
- Teardown must run even when scenario steps fail (finally semantics)

## Acceptance Criteria

- [ ] Schema accepts `setup` and `teardown` blocks on contracts and scenarios
- [ ] Contract-level setup runs before any scenarios; contract-level teardown runs after all scenarios
- [ ] Scenario-level setup runs before that scenario's steps; scenario-level teardown runs after
- [ ] Teardown executes even when scenario steps fail
- [ ] Setup/teardown steps support the same actions as regular steps (http, cli, wait, etc.)
- [ ] Setup failures prevent scenario execution and report clearly
- [ ] Teardown failures are reported but don't mask scenario results
- [ ] Unit tests cover: setup + steps + teardown happy path, setup failure skips scenario, teardown runs on step failure
- [ ] Update maslow-tok example to show a seed/reset fixture pattern

## Notes

- The MaslowTok original design had `system.test_controls` declaring reset/seed/explain/tick endpoints — these would be used in setup/teardown steps
- Consider whether setup steps should support `capture` (e.g., seed returns an ID that scenarios reference) — this depends on task 13
- Keep it simple: setup and teardown are just arrays of steps, same as the scenario body
- If contract-level and scenario-level setup both exist, contract-level runs first
