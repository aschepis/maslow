# ADR 0043: Requirement Links with `covers`

- **Status**: Accepted
- **Date**: 2026-11-19

## Context

Nothing in a spec or its evidence linked a check or contract to the requirement it proves, so a consumer could not tell which contract shows issue N is fixed, or that some acceptance criterion has no contract at all. The gate that runs `evidence check` (for example the Damocles reconciler) already holds the issue's acceptance criteria; Maslow holds no requirement list of its own.

## Decision

Checks, contracts and scenarios take an optional `covers` list of requirement IDs, which the evidence records per result and summarizes in a derived top-level `coverage`, and `maslow evidence check --require-covered <id>` (repeatable) declines unless results that all passed cover each given ID.

## Consequences

- **ID format.** An ID is an opaque, non-empty string with no whitespace, compared byte for byte and case-sensitively; Maslow never parses or normalizes it, so `#7`, `7` and `acme/app#7` are three IDs. Recommended forms, not enforced: `owner/repo#N` for an issue, a URL for anything with one, a project prefix such as `REQ-12` otherwise, and a suffix such as `acme/app#7/ac-2` for one criterion of an issue.
- **Where links live.** `covers` goes on `#CheckRunner`, `#Contract` and `#Scenario`; overlays inherit it through `#Contract`. A scenario result covers the sorted, deduplicated union of its contract's and its own IDs. Budgets take no `covers` yet. `covers` never changes the verdict.
- **Where the requirement list lives.** Not in the spec or overlay: it is a trusted input to `evidence check`, like the trusted overlays, so `verify` cannot report an uncovered ID and the evidence's `coverage` lists only IDs something covered. There is no file form of the flag yet. An empty or whitespace-containing `--require-covered` is a usage error (exit 2).
- **Derived `coverage`.** Each entry is `{id, status, results}`, sorted by ID, with results named `check:<name>` or `contract:<name>`. Status takes the verdict's precedence: `fail` if any covering result is `fail`, `error` or `not_red`, `needs_review` if any is a flaky overlay contract, `pass` if any passed or was flaky, `skip` otherwise. `evidence check` ignores this field and recomputes from the per-result `covers`, because the evidence is self-reported. `evidence summary` lists it in a "Requirements" section.
- **Check rule.** Every covering result must be `pass` or `flaky`, and one reason per ID (field `covers`) names the first that is not. With `--require-red-green`, only overlay contracts count (amending ADR 0037), and only `pass` is accepted: a spec result already green on the base, or a flaky overlay contract, is not proof of the change.
- **Trust.** An agent editing `covers` to claim coverage changes a hashed input: overlay edits fail `--overlay`/`--overlay-sha256` (ADR 0033) and spec edits fail `--spec`. `evidence_version` stays `"2"`; `covers` and `coverage` are optional and additive, as in ADR 0034.
