# ADR 0034: Red-Green Overlay Contracts

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

An overlay contract (ADR 0029) states what "fixed" means for one change. If it already passes on the base commit, its pass on the head says nothing about the change, which is how a vague acceptance step or an agent produces a hollow pass. The run needs to show each overlay contract fails without the change.

## Decision

`maslow verify --red-green --base <ref> --overlay <file>` runs each overlay contract on the head as usual and again in a detached `git worktree` of the merge base, and passes it only when the base run fails and the head run passes; one that passes on both gets status `not_red`, which makes the verdict `fail`.

## Consequences

- `not_red` fails rather than needs review: the overlay is the change's definition of done, `needs_review` exits 0 by default, and a contract that should already hold belongs in the spec, not an overlay. A head failure keeps its status; a base run that errors, or a base checkout that cannot be made, gives `error`, since red cannot be shown.
- The base tree is a fresh checkout under the system temp directory, removed after the run, with CLI steps run in it from the same subdirectory as the head. Maslow does no build of its own there: whatever a scenario needs (`go build`, starting a server) goes in its contract or scenario `setup`, which runs in the checkout. Untracked or ignored files, such as `node_modules` or a built `bin/`, are not present on the base, and any base failure, including a setup or build failure, counts as red; the base `failed_step` lets a reviewer see why it failed.
- Evidence (still version 2; the fields are additive) records `head_status` and `base` (`sha`, `status`, `error`, `failed_step`) on each overlay contract; `status` is the combined result. Spec contracts and the base run's artifacts are not affected. `maslow evidence check` (ADR 0033) trusts the verdict, so a `not_red` run declines without change; its `--require-red-green` flag (ADR 0037) also requires that a run used `--red-green`.
- A base run that fails and then passes on a retry is `flaky` and so not red; see ADR 0039 for flaky base runs.
