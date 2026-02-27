---
id: 18
title: Add array and collection assertions for contract expectations
status: todo
priority: medium
created: 2026-02-20
updated: 2026-02-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, contract, schema]
---

## Objective

API responses frequently return arrays/collections, but the current expectation model can only check exact values at specific JSON paths. There's no way to express "the feed has at least 5 items," "every item has an id field," or "at least one item matches this condition." The MaslowTok original spec used `at_least_one` and `one_of` constructs that the current schema cannot express.

## Requirements

- Extend the `#Expectation` schema with collection-aware assertion fields:
  - `min_length` — array at json_path has at least N items
  - `max_length` — array at json_path has at most N items
  - `length` — array at json_path has exactly N items
  - `every` — every item in the array satisfies a sub-expectation (json_path relative to each item)
  - `some` / `at_least_one` — at least one item satisfies a sub-expectation
- Update CUE schema, embedded copy, Go types, and runner
- Collection assertions apply to the array found at the step's `json_path`

## Acceptance Criteria

- [ ] Schema accepts `min_length`, `max_length`, and `length` fields on expectations
- [ ] Schema accepts `every` and `some` fields with nested sub-expectations
- [ ] Runner evaluates length constraints against arrays at the specified JSON path
- [ ] Runner evaluates `every` by checking sub-expectation against all array items
- [ ] Runner evaluates `some` by checking sub-expectation against array items (pass if any match)
- [ ] Clear error messages: "expected array at $.items to have at least 5 items, got 2"
- [ ] Non-array values at the json_path produce clear type error
- [ ] Unit tests cover: length checks, every with pass/fail, some with pass/fail, empty arrays, non-array values
- [ ] Update maslow-tok example to use collection assertions on feed endpoint

## Notes

- Example usage:
  ```yaml
  expect:
    status: 200
    json_path: "$.items"
    min_length: 5
    every:
      json_path: "$.id"
      # just asserting the path exists (non-null)
    some:
      json_path: "$.featured"
      value: true
  ```
- Keep `every` and `some` simple for v1 — they take a sub-expectation with json_path + value
- This pairs with task 14 (multiple assertions) — together they make contract expectations expressive enough for real API testing
- The `one_of` construct from MaslowTok's original spec (multiple acceptable outcomes) is a separate concern — could be a follow-up
