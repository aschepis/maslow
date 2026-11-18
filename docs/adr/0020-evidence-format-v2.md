# ADR 0020: Evidence Format Version 2

- **Status**: Accepted
- **Date**: 2026-11-18
- **Supersedes**: ADR 0002

## Context

A pipeline that gates on Maslow stores the evidence file and acts on it later, often without the repository. Version 1 did not say which spec, which Maslow or which command produced it. `verify` and `audit` both wrote `reports/verify.json`, so one overwrote the other. Check output was stored in full, although ADR 0002 promised a 4 KB cap.

## Decision

Evidence carries `evidence_version: "2"`, `maslow_version`, `command`, `spec_path`, `spec_sha256`, `spec_source`, `git_sha` and `base_sha`. Its shape is fixed by `schema/evidence.schema.json`, and a test checks real output against that schema. Each command writes its own default file (`reports/verify.json`, `reports/audit.json`), and `--report` overrides it.

## Consequences

- A reader with only the file can tell what was checked, against which spec bytes, with which Maslow, and what failed.
- Check output keeps the last `--output-limit` bytes (default 4096), where failures are usually printed, and records `output_truncated` and the original `output_bytes`.
- Adding a field means changing the schema in the same change, because the schema rejects unknown fields; renaming or removing one needs a new `evidence_version`.
