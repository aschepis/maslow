# ADR 0033: Evidence Check

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

Every adopter that gates a merge on Maslow re-implements the same checks on the evidence file (format version, head commit, spec hash, profile, verdict), and getting one wrong approves a bad change. Overlays (ADR 0029) add contracts per run, so the spec hash alone no longer says what was verified.

## Decision

`maslow evidence check <file> --head <sha> --spec <file> [--profile <name>] [--overlay <file>] [--overlay-sha256 <hex>] [--json]` declines on any mismatch with the trusted inputs or a verdict other than `pass` or `needs_review`, otherwise passes `needs_review` through and passes only a `pass` verdict; it exits 0 on pass, 1 otherwise, and 2 on usage errors.

## Consequences

- Rules are strict equality: only the current `evidence_version` is known, `git_sha` must equal `--head` exactly (no short SHAs), the trusted spec is hashed byte-for-byte as `verify` hashes it, and the run's overlay hashes must equal the trusted set in both directions, so a missing acceptance overlay declines as surely as an unknown one.
- The command trusts the evidence's own `verdict` and does not recompute it from the sections; observe-mode `would_*` policy statuses therefore never affect the decision, matching `ComputeVerdict` (ADR 0031). Unknown verdict strings decline.
- An unreadable or malformed evidence file is a `decline` (exit 1), while an unreadable trusted spec or overlay is a usage error (exit 2), because the latter is the gate's own misconfiguration.
