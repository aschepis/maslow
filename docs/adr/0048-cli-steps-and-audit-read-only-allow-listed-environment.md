# ADR 0048: Run CLI Steps and Audit Binaries With a Scrubbed, Allow-Listed Environment

- **Status**: Accepted
- **Date**: 2026-11-21
- **Extends**: ADR 0046, ADR 0026

## Context

ADR 0046 scrubbed the environment of checks, but two other places still started the change's code with Maslow's whole environment, CI tokens included: CLI contract steps, whose contract `env` list (ADR 0026) only limited `{{.env.NAME}}` templates, and `version-check` on binary audit targets. The audit code also rebuilt `cmd.Env` on every pass over `target.Env`, so only one of its entries reached the binary (task 52).

## Decision

A CLI step runs with the ADR 0046 base set plus its contract's `env` names, and a binary audit target runs with the base set plus every `target.Env` entry; `inherit_env: true` on a contract or on an audit target restores Maslow's whole environment and is recorded as `inherited_env: true` on that contract's scenario results or that target's `version` result.

- The opt-out sits on each contract and each target, not on the spec. `contracts` is a list with no spec-level object to hang a flag on, and a per-contract flag lets a spec migrate one contract at a time. `checks.inherit_env` is unchanged and still covers only checks.
- `target.Env` entries are applied after the base set in name order, so an entry overrides a base variable of the same name and the result does not depend on map order.
- HTTP, poll, browser and observability steps start no process and are unchanged. Endpoint audit targets run contracts, so their CLI steps follow the contract's own `env` and `inherit_env`.
- `{{.env.NAME}}` in an endpoint target's `path` still reads the whole environment: the audit target is the operator's configuration, not the change's code.
- The base set and its lookup live in `internal/runner/env.go` as `ScrubbedEnv`, which `internal/audit` already imports.

## Consequences

- Breaking: a CLI step that read an inherited variable, such as a tool's config path or a `GOFLAGS`, sees it unset until the contract lists it in `env` or sets `inherit_env: true`. Listed values are already redacted from contract evidence (ADR 0026).
- Breaking for audit: a binary that read an inherited variable must get it through `target.Env`, which holds literal values, or `inherit_env: true`.
- `maslow evidence summary` names every check, contract scenario and audit binary that ran with the inherited environment in one "Inherited environment" section.
