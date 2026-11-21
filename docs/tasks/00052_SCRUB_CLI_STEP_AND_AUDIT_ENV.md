---
id: 52
title: Run CLI contract steps and audit binaries with a scrubbed environment
status: done
priority: medium
created: 2026-11-21
updated: 2026-11-21
assigned_to: "claude"
assigned_at: "2026-11-21"
depends_on: [41]
tags: [kind:gap, contracts, audit, security]
---

## Objective

Task 41 (ADR 0046, #46) gave checks a minimal base environment plus their `env` list. Two other
places still run the change's code with Maslow's whole environment, CI tokens included:

- **CLI contract steps.** `runCLIStep` (`internal/runner/contract.go`) never sets `cmd.Env`. A
  contract's `env` list (ADR 0026) only limits what `{{.env.NAME}}` templates read; the process
  itself inherits everything.
- **Audit binary targets.** `runVersionCheck` (`internal/audit/audit.go`) builds
  `append(os.Environ(), ...)`. It also overwrites `cmd.Env` on each loop pass, so only the last
  entry of `target.Env` survives, which is a bug.

## Requirements

- A CLI step's process gets the ADR 0046 base set plus the contract's `env` names, through the same
  helper as checks. Listed values are already redacted in contract evidence (ADR 0026).
- An audit binary gets the base set plus every entry of `target.Env`, all of them, not only the
  last.
- The migration opt-out follows ADR 0046: a contract-level or spec-level `inherit_env` that is
  recorded in the evidence. Decide whether `checks.inherit_env` is extended or a new field is added,
  and record it in ADR 0046 or a new ADR.
- HTTP, browser and provider steps are unchanged; they start no process.

## Acceptance Criteria

- [x] Tests: an unlisted variable is absent from a CLI step and from an audit binary; a listed one
      is present; every `target.Env` entry reaches the audit binary; the opt-out restores
      inheritance and shows in the evidence.
- [x] `maslow.yaml` and `examples/` still pass, with `env` lists added where a CLI step needs one.
- [x] CHANGELOG `### Breaking` entry for CLI steps, `### Fixed` for the audit `Env` bug.

## Notes

Found while implementing task 41. Approved by the human on 2026-11-21, to build once #46 merges.
