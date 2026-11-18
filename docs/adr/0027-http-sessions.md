# ADR 0027: A Cookie Jar per Scenario

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

HTTP steps used Go's default client, which keeps no cookies. A contract that signed in had to capture the `Set-Cookie` header and rebuild a `Cookie` header by hand on every later step, and a redirect after sign-in lost the session entirely.

## Decision

Each scenario gets its own HTTP client with a cookie jar, shared by its setup, steps and teardown; contract-level setup and teardown each get their own.

## Consequences

- A scenario can sign in during setup and stay signed in, including across redirects and `poll` steps.
- Scenarios stay independent: one scenario's session never leaks into the next, so their order does not change results.
- A scenario cannot reuse a session created in contract-level setup. Signing in per scenario costs a request each; sharing would make scenarios order-dependent.
- A server that sets cookies now receives them back on later steps where it did not before. A scenario that needs an anonymous request after signing in should be split.
