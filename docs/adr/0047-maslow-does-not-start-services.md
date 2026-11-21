# ADR 0047: Maslow Does Not Start the Services It Verifies

- **Status**: Accepted
- **Date**: 2026-11-21
- **Refines**: ADR 0016

## Context

Since ADR 0044 a contract's `setup` step cannot leave a server running, so HTTP, browser and
`metric`/`log`/`alerts` contracts only reach servers started outside Maslow. Task 51 proposed a
`services:` section so Maslow would start, probe and stop the change's own servers. The human
rejected it: provisioning what is under test is the caller's job, not a verifier's.

## Decision

Maslow never starts long-running services; contracts verify a system the caller has already
started, reached through `base_url` and `env`.

## Consequences

- The caller (CI, a harness, Damocles) starts servers before `maslow verify` and stops them after,
  and passes their address in through an allow-listed variable, for example
  `base_url: "{{.env.API_URL}}"`.
- `--red-green` for server-backed contracts needs the caller to also run the base commit's server
  and point the base run at it; Maslow cannot do that itself. Red-green stays most useful for CLI
  contracts.
- Setup and teardown steps remain short commands (seed data, reset state), never daemons, which
  keeps ADR 0044's cleanup intact.
