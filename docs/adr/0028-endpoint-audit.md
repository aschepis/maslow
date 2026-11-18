# ADR 0028: Endpoint Audit Runs Contracts Against a Base URL

- **Status**: Accepted
- **Date**: 2026-11-18
- **Amends**: ADR 0005 (endpoint targets)

## Context

The `endpoint` audit target was a stub that returned `skip`. The scenarios a project writes as contracts already describe how its service should behave; running them against a deployed URL would make them a post-deploy smoke test without a second spec. Contracts, however, hard-coded their URLs, usually through `{{.env.BASE_URL}}`.

## Decision

A contract may set `base_url`, and HTTP and poll step URLs that start with `/` resolve against it; an `endpoint` audit target runs the contracts its `checks` select (by name or tag, all when omitted) with its `path` as every contract's base URL.

## Consequences

- One scenario serves both `verify` against a local server and `audit` against a deployment. Absolute URLs are left alone, so existing contracts behave as before.
- The target `path` may read `{{.env.NAME}}` from the whole environment, because it is the operator's audit configuration rather than a contract. Results are named by the path as written, so the expanded URL does not reach the evidence.
- A relative URL with no base URL fails the step with an error naming `base_url`, and a selector that matches no contract is an error, as in profiles.
- `timeout` and `retries` do not yet apply to endpoint targets; each HTTP step keeps its 30-second limit. The `container` target is still a stub.
