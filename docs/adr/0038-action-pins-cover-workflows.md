# ADR 0038: Action Pins Cover Third-Party Actions in Workflows

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

The `action-pins` check required full commit SHAs only in `action/action.yml`, because callers pin our Action by SHA. A third-party action in `.github/workflows/` runs with the workflow's secrets and `GITHUB_TOKEN`, so a moved tag there is the same supply-chain risk. Task 42 considered adding `browser-actions/setup-chrome`, which made the gap concrete.

## Decision

`scripts/check-action-pins.sh --workflows .github/workflows action/action.yml` requires SHA pins for every action in the published Action and for every action not under `actions/` in the workflows.

## Consequences

- First-party `actions/*` in workflows may keep tag pins (`actions/checkout@v5`); GitHub owns those tags, and pinning them would add Dependabot churn for little gain. The published Action still pins them by SHA.
- Adding a third-party action to CI now fails `maslow verify` until it is pinned by SHA. The `action-pins-check` contract proves this against fixtures in `testdata/action-pins/`.
- The script exits 2 on a missing file or directory; before, a grep error read as "all actions pinned".
