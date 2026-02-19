# ADR 0002: Evidence Format - reports/verify.json Structure

- **Status**: Accepted
- **Date**: 2026-02-19

## Context

Maslow needs a deterministic, diffable, machine-readable output format for verification and audit results. Both the `verify` and `audit` commands must produce evidence that can be stored, compared across runs, and consumed by downstream tooling without ambiguity.

Key requirements for the evidence format:

- Diffable: the structure must be stable enough that two passing runs on the same codebase produce meaningfully comparable output
- Machine-readable: CI systems, audit pipelines, and agents must be able to parse results without custom logic
- Self-describing: a single file should contain enough context to interpret the results without external state
- Shared: `verify` and `audit` must use identical evidence structures so downstream consumers handle one format

Several options were considered:

| Option | Considered |
|--------|------------|
| Output format | JSON, YAML, NDJSON, protobuf |
| File location | stdout only, reports/ directory, configurable path |
| Verdict placement | top-level field, computed by consumer, separate file |

## Decision

**Output format: JSON (`reports/verify.json`)**

A single JSON file written to `reports/verify.json` after each run. The file has the following top-level fields:

- `timestamp`: RFC3339 UTC string representing when the run completed
- `git_sha`: the current HEAD commit SHA of the repository being verified
- `profile`: the active profile name from `maslow.yaml`
- `verdict`: one of `pass`, `fail`, or `inconclusive`
- `check_results`: array of individual check result objects
- `contracts`: array of contract evaluation results
- `budgets`: array of budget evaluation results

Each entry in `check_results` has the following fields:

- `name`: the check name as defined in `maslow.yaml`
- `status`: one of `pass`, `fail`, `skip`, or `error`
- `duration`: elapsed wall-clock time in milliseconds
- `output`: captured stdout and stderr from the check
- `error`: error message if the check could not be executed, otherwise null

**Verdict computation**

The verdict is computed deterministically from `check_results`:

- If any result has status `fail` or `error`, the verdict is `fail`
- If all results have status `pass`, the verdict is `pass`
- If `check_results` is empty or all results are `skip`, the verdict is `inconclusive`

The verdict field is always written; consumers must not recompute it.

## Consequences

**Positive**

- JSON is universally parseable without special libraries
- The top-level `verdict` field allows fast pass/fail detection without inspecting the full results array
- Stable structure enables git-diffing evidence files across branches or runs
- A single shared format means `verify` and `audit` consumers use identical parsing logic
- `git_sha` and `timestamp` provide full provenance without external metadata

**Negative**

- `timestamp` changes on every run, making exact byte-equality across runs impossible; consumers that diff evidence files must exclude this field
- A single flat file may grow large for repositories with many checks, contracts, and budgets; no pagination or streaming is provided in v1
- `reports/verify.json` is a fixed path; projects with unusual layouts cannot relocate it without wrapping the CLI

**Mitigations**

- Tooling that diffs evidence files should compare `verdict`, `check_results`, `contracts`, and `budgets` while ignoring `timestamp`; this is documented in the CLI reference
- File size is bounded in practice by the number of checks defined in `maslow.yaml`; checks with large output should trim stdout/stderr to a configurable limit (default 4 KB per check)
