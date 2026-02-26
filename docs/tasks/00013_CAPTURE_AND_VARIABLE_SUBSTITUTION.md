---
id: 13
title: Implement capture action and variable substitution in contract runner
status: draft
priority: critical
created: 2026-02-20
updated: 2026-02-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, contract, runner]
---

## Objective

The `capture` step action exists in the CUE schema and Go types but is a no-op in the contract runner. Without variable capture and substitution, contracts cannot test any authenticated flow — which is the vast majority of a real application's API surface.

This is the single highest-impact verification gap. The MaslowTok example has a login scenario that returns an access token, but there is no way to use that token in subsequent requests.

## Requirements

- Implement the `capture` step action in `internal/runner/contract.go`
- `capture` extracts a value from the previous step's state and stores it as a named variable
  - `from` field: a JSON path expression (e.g., `$.access_token`) or a keyword (`output`, `status`, `header:<name>`)
  - `as` field: the variable name to store the captured value
- Implement variable substitution in all step fields that accept strings (url, headers, body, command, args)
  - Syntax: `${var_name}` for captured variables (distinct from `${ENV_VAR}` which reads from environment)
  - Or use a distinct syntax like `${{var_name}}` to avoid ambiguity with env vars
- Variables are scoped to the scenario (not shared across scenarios unless we add scenario dependencies later)
- Clear error messages when a capture fails (JSON path not found, no previous step output)
- Clear error messages when a variable reference is unresolved

## Acceptance Criteria

- [ ] `capture` step extracts values from previous step's HTTP response body via JSON path
- [ ] `capture` step extracts values from previous step's HTTP response headers
- [ ] `capture` step extracts values from previous step's CLI output
- [ ] Captured variables are substituted into subsequent step URLs, headers, body, and CLI args
- [ ] Environment variable substitution (`${ENV_VAR}`) continues to work alongside captured variables
- [ ] Unresolved variable references produce clear error messages with the variable name and step context
- [ ] Failed captures (e.g., JSON path not found) produce clear error messages
- [ ] Unit tests cover: capture from HTTP response, capture from headers, capture from CLI output, substitution in all field types, error cases
- [ ] Update the todo-app or maslow-tok example to demonstrate an authenticated contract flow using capture

## Notes

- The `stepState` struct in contract.go already tracks output, httpStatus, and httpHeaders — these are the capture sources
- JSON path extraction already works via `extractJSONPath` — reuse it for capture
- Consider whether captured variables should be logged in evidence for debuggability
- This unblocks scenario chaining patterns like: login → capture token → use token in subsequent requests
- Syntax decision (${var} vs ${{var}}) should be recorded as an ADR
