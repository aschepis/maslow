---
id: 48
title: Add observability providers for New Relic, CloudWatch and AWS Managed Prometheus
status: draft
priority: low
created: 2026-11-19
updated: 2026-11-19
assigned_to: ""
assigned_at: ""
depends_on: [36]
tags: [kind:gap, contracts, audit]
---

## Objective

`metric`, `log` and `alerts` steps (task 36, ADR 0042) work with Prometheus-compatible APIs, Loki and
Alertmanager, with Datadog to follow (task 47). Other common backends cannot be asserted on yet.

## Candidates

- **New Relic.** NRQL through NerdGraph (`POST /graphql`) for metrics and logs; alert issues for
  `alerts`. Auth is an `API-Key` header.
- **CloudWatch.** `GetMetricData` for metrics and Logs Insights for logs; alarms in `ALARM` state for
  `alerts`. Needs SigV4 request signing, which header-only auth cannot express.
- **AWS Managed Prometheus.** The existing `prometheus` provider speaks its API, but every request
  must be SigV4-signed.

## Questions to answer

- How are AWS credentials supplied without breaking env redaction (ADR 0026)? SigV4 signs with a
  secret that never appears in a header, so redaction is easy, but which env variables, and is an
  instance or OIDC role allowed?
- Should signing be an option on the existing `prometheus` provider (`auth: sigv4`) rather than a
  new provider?
- Which of these does Damocles need first?

## Notes

Review (2026-11-20): stays a draft until someone needs one of these. Leaning: AWS Managed Prometheus
is `auth: sigv4` on the existing `prometheus` provider, not a new provider.
