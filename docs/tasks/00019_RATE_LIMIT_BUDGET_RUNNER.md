---
id: 19
title: Implement rate limit budget runner
status: draft
priority: medium
created: 2026-02-20
updated: 2026-02-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, budget, rate-limit]
---

## Objective

The rate limit budget schema supports `max_rps` and `burst` but the runner skips all rate limit budgets. MaslowTok specifies a comment rate limit (max_rps: 1, burst: 20) that can't be verified. Rate limiting is a critical safety property for any user-facing API.

## Requirements

- Implement rate limit budget execution in `internal/runner/budget.go`
- For HTTP targets (format: `http:<METHOD> <path>`):
  - Send requests at a rate exceeding `max_rps` to trigger rate limiting
  - Verify that the server eventually returns 429 Too Many Requests (or configurable status)
  - Verify that `burst` requests succeed before rate limiting kicks in
  - Optionally verify `Retry-After` header presence
- Results include: requests sent, requests succeeded, requests rate-limited, whether rate limiting engaged at expected threshold
- Respect environment variable substitution in target URLs

## Acceptance Criteria

- [ ] Rate limit budgets execute against HTTP endpoints
- [ ] Runner sends burst + N requests and verifies rate limiting engages
- [ ] Pass condition: rate limiting engages within a reasonable margin of the burst threshold
- [ ] Budget result includes: total requests, accepted count, rejected count, threshold where limiting started
- [ ] Environment variable substitution works in target URLs
- [ ] Non-HTTP targets skip gracefully with informative message
- [ ] Unit tests cover: rate limiting detection, burst threshold verification, no-rate-limit detection (fail case)
- [ ] Integration test with a rate-limited HTTP handler

## Notes

- The MaslowTok original spec had `limit_per_minute: 20, retry_after_present` — the current schema uses `max_rps` and `burst` which is a different model. Either the runner adapts or the schema should be revisited.
- For v1, focus on "does rate limiting exist and roughly match the budget" rather than precise threshold verification
- Consider whether authenticated requests are needed (depends on task 13 for capture)
- This is simpler than the performance budget runner — it's a sequential burst test, not a sustained load test
