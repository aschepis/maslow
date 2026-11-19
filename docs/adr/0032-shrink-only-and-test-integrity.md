# ADR 0032: Shrink-Only Files and Test-Integrity Policy

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

The cheapest way for an agent to turn a build green is to weaken what judges it: delete a failing test, add a skip, lower a coverage floor, or add an entry to a baseline that should only ever shrink. Existing policy rules see only which paths changed, and the tests that guard such files can be edited by the same change.

## Decision

`policy` gains `shrink_only` (path patterns whose files may lose lines but not gain any, enforced as `fail`) and `test_integrity` (deleted test files, removed test declarations, added skip markers and lowered numbers in `thresholds` files, each reported as `needs_review`), both judged on a line-level diff from the base, or from HEAD without one, to the working tree including untracked files.

## Consequences

- `shrink_only` fails because it names exact files and the check is mechanical; `test_integrity` asks for review because its findings are heuristic and deleting an obsolete test is often right. Either can run under `policy.observe` (ADR 0031) first, and anyone who wants a hard stop can promote the review with `--strict`.
- A test declaration is matched by the text its expression matches, so a test whose body or signature line is edited, or which moves to another file, is not reported removed; a skip is cancelled only by an identical skip line removed elsewhere. A renamed test file still reports as deleted, since policy diffs run with renames off.
- `thresholds` assumes numbers are floors: a line pair that differs only in numbers, with one lower, is flagged, so a file of maximums (where lowering tightens) does not belong there. Built-in test patterns cover Go, Ruby, JavaScript/TypeScript and Python; a listed built-in fills unset fields from its defaults, and any other language must give `files`. Flagged lines appear in a new optional `lines` field on policy evidence, capped at 100 per result.
