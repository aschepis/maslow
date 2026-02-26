---
id: 14
title: Support multiple assertions per contract step
status: todo
priority: high
created: 2026-02-20
updated: 2026-02-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, contract, schema]
---

## Objective

Today, a contract step's `expect` block supports only one `json_path` + `value` pair. Real API responses have multiple fields that need validation. For example, a login response needs to assert both `$.access_token` exists AND `$.token_type` equals "bearer" AND `$.expires_in` is a positive number.

The current workaround is adding multiple `assert` steps after an HTTP step, but this is verbose and loses the connection between the request and its expectations.

## Requirements

- Extend the `#Expectation` schema to support multiple JSON path assertions in a single expect block
- Two possible approaches (pick one, record as ADR):
  - **Option A**: `assertions` array field — `assertions: [{json_path: "$.a", value: "x"}, {json_path: "$.b", value: 1}]`
  - **Option B**: `json_paths` map field — `json_paths: {"$.a": "x", "$.b": 1}`
- Update the CUE schema (`schema/maslow.cue`) and the embedded copy
- Update the Go types in `internal/spec/`
- Update the expectation validation logic in `internal/runner/contract.go`
- Maintain backward compatibility with the existing single `json_path` + `value` fields
- All assertions in a single expect block are AND'd (all must pass)

## Acceptance Criteria

- [ ] Schema accepts multiple JSON path assertions in a single expect block
- [ ] Runner evaluates all assertions and reports which specific ones failed
- [ ] Existing single `json_path` + `value` specs continue to work unchanged
- [ ] Error messages identify which assertion within the expect block failed
- [ ] Unit tests cover: multiple assertions all passing, one failing, mixed with other expect fields (status, headers, body_contains)
- [ ] Update maslow-tok example contracts to use multiple assertions where appropriate
- [ ] CUE schema and embedded copy are in sync

## Notes

- Option B (map) is more concise but loses ordering; Option A (array) is more explicit
- Consider whether to support assertion operators beyond equality (e.g., `exists`, `not_null`, `gt`, `lt`, `matches`) — this could be a follow-up task
- The MaslowTok login scenario currently checks `json_path: "$.access_token"` and `body_contains: "bearer"` separately; with this change it could check both fields structurally
