# ADR 0036: The Job Summary Is Rendered by Maslow

- **Status**: Accepted
- **Date**: 2026-11-18
- **Amends**: ADR 0024

## Context

ADR 0024 keeps all logic in the binary, but the Action built its job summary with an untested jq program. That jq dropped observe-mode policy results (ADR 0031), so a run whose only findings were observed showed "Nothing failed.", and every later summary change would have grown the jq.

## Decision

The Action's "Write job summary" step runs `maslow evidence summary <file>`, which prints Markdown from `evidence.Summary`, and falls back to the old jq, unchanged, when the command fails because the installed release lacks it.

## Consequences

- The summary is covered by golden-file tests. Observed findings with a `would_*` status get their own "Observed policy rules (not enforced)" table; observed passes are a one-line count, and a run with no observed rules has no such section. Enforced rows are as before, plus `not_red` contracts, which fail the verdict but the jq never listed.
- Two renderers exist until the fallback is removed, one minor release after the command ships; after that an old binary should get a note to upgrade instead.
- There is no size cap on the output yet; `$GITHUB_STEP_SUMMARY` allows 1 MiB per step, and task 39's `--max-bytes` covers it.
