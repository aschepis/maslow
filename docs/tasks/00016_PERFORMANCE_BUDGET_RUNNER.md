---
id: 16
title: Implement performance budget runner
status: draft
priority: medium
created: 2026-02-20
updated: 2026-02-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, budget, performance]
---

## Objective

The performance budget schema is fully defined (target, p50/p90/p95/p99/max, error_rate, concurrency, duration, warmup) but the runner skips all performance budgets. This means MaslowTok's feed latency (p90 < 150ms) and video endpoint latency (p90 < 80ms) budgets are unverifiable.

## Requirements

- Implement performance budget execution in `internal/runner/budget.go`
- For HTTP targets (format: `http:<METHOD> <path>`):
  - Send concurrent requests at the specified concurrency level
  - Run for the specified duration after warmup period
  - Collect response times and compute percentile distribution
  - Calculate error rate (non-2xx responses / total)
  - Compare against budget thresholds (p50, p90, p95, p99, max, error_rate)
- For non-HTTP targets (e.g., `eventual_consistency:video_ready`, `boot`, `frame_time_1080p`):
  - These require external measurement — skip with a clear message or support a `command` field that outputs timing data
- Results include: actual percentile values, actual error rate, pass/fail per threshold
- Respect the target URL's environment variable substitution (e.g., `${API_BASE_URL}`)

## Acceptance Criteria

- [ ] HTTP performance budgets execute concurrent requests and measure latencies
- [ ] Warmup period requests are excluded from measurement
- [ ] Percentile calculations are correct (p50, p90, p95, p99)
- [ ] Error rate calculation counts non-2xx responses
- [ ] Each threshold is independently evaluated (can pass p90 but fail p99)
- [ ] Budget result includes actual measured values for all thresholds
- [ ] Non-HTTP targets skip gracefully with informative message
- [ ] Environment variable substitution works in target URLs
- [ ] Unit tests cover: percentile math, error rate calculation, threshold comparison, timeout handling
- [ ] Integration test with a simple HTTP server verifying end-to-end budget execution

## Notes

- Keep the implementation simple — this doesn't need to be k6. A basic Go HTTP load generator with goroutines is sufficient for v1.
- Consider whether to use `net/http` directly or a lightweight load testing library
- The MaslowTok spec has performance budgets for feed (p90: 150ms, p99: 350ms, concurrency: 50) and video (p90: 80ms, p99: 200ms, concurrency: 100) — use these as validation that the implementation handles real-world specs
- The RPG game example has non-HTTP budgets (boot time, frame time, scene load) — these will need the command-based approach in a follow-up
- Concurrency of 50-100 is modest enough that goroutines without a pool should be fine
