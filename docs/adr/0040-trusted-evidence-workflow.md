# ADR 0040: Trusted Evidence Comes From a Base-Branch Workflow, With Provenance From the GitHub API

- **Status**: Accepted
- **Date**: 2026-11-19

## Context

The pull request's code ran in the job that wrote the evidence, and the workflow file that defined that job was the pull request's own copy. The cheapest forgery was to edit `ci.yml` to skip `maslow verify` and upload a hand-written `verify.json`; a consumer could not tell it from a real pass (task 35).

## Decision

Trusted evidence is produced by `.github/workflows/maslow-evidence.yml`, a `pull_request_target` workflow whose definition always comes from `main`, which runs only for pull requests whose head repository is this repository; a consumer establishes provenance through the GitHub API, not through signatures.

## How it works

- **`verify` job.** `contents: read` only, no secrets, `persist-credentials: false` and no cache. It checks out the head, builds maslow from the base SHA into `$RUNNER_TEMP`, then runs `verify --profile full --spec-from base` and uploads `verify.json` as the artifact `maslow-evidence-<head sha>`. The verifier, the spec and the workflow are `main`'s; only the checks and contracts run the change.
- **`record` job.** Never checks out or runs the change. It downloads the artifact, fails if it is missing or its `git_sha` is not the pull request head, and writes the file's sha256, the claimed verdict and the head SHA to the job summary and a job output. It is kept because it gives the pull request a status from code the change never touched, and gives a reviewer a hash to compare with the downloaded artifact. It does not make the evidence's contents trustworthy, and it does not run `evidence check`; that needs a trusted spec and merge base, which the consumer already holds.
- **Fork pull requests** never get trusted evidence. Both jobs have `if: github.event.pull_request.head.repo.full_name == github.repository`; forks get ordinary `ci.yml` feedback.
- **Mechanical rules.** `scripts/check-trusted-workflow.sh` (check `trusted-workflow`, contract `trusted-workflow-check`, fixtures in `testdata/trusted-workflow/`) fails the workflow if it gains another trigger, top-level permissions other than `{}`, any write permission, any `secrets.` or `github.token`, a job without the same-repository guard, a checkout that keeps its token, `actions/setup-go` with its cache, or `actions/cache`. zizmor runs as the `zizmor` check in the full profile. `zizmor.yml` records the suppression of `dangerous-triggers` for this one workflow. Another CI system swaps the check command; Maslow has no code for it.

## What the consumer does

1. List candidates with `GET /repos/O/R/actions/artifacts?name=maslow-evidence-<head>`. Any workflow can upload an artifact by that name, so the name only finds candidates. Never take a run or artifact ID from the pull request author or from the evidence.
2. For each candidate, fetch its run with `GET /repos/O/R/actions/runs/<workflow_run.id>` and keep it only if `path` is `.github/workflows/maslow-evidence.yml`, `event` is `pull_request_target`, `head_branch` is `main` and `repository.full_name` is `O/R`. A `pull_request_target` run takes its definition from the base branch, so `head_branch: main` is what says `main`'s workflow ran. GitHub reports these fields, not the run.
3. Download that artifact (`GET /repos/O/R/actions/artifacts/<id>/zip`). Optionally compare its sha256 with the `record` job's summary.
4. Run `maslow evidence check verify.json --head <head> --spec <trusted spec> --profile full`. `--head` stops evidence for one commit being replayed for another.

## What this proves and what it does not

- Proves: the workflow definition, the verifier binary and the spec came from `main`; the evidence was uploaded by a run of that workflow for that head commit; a pull request cannot get trusted evidence by editing a workflow, the spec or the verifier, or by uploading a file from another run.
- Does not prove: that the checks and contracts ran honestly. Code under test runs in `verify` as a user with sudo, and Maslow does not kill a check's process group, so a deliberate change can rewrite `verify.json` or upload its own artifact before the real one. That has to be written into the diff on purpose. Task 45 tracks running checks isolated from the evidence writer. This is the accepted remaining risk.
- Does not cover the pull request that adds or edits this workflow. `main`'s copy runs until the change merges, as with `spec-from: base` (ADR 0030). `.github/workflows/**` and `zizmor.yml` are gated paths owned by `@aschepis`, so workflow edits need review.

## Consequences

- No artifact attestations or Sigstore signing, so this works on every GitHub plan, including private repositories without Enterprise Cloud. Attestations may be added later as an optional, stronger signal. `maslow evidence check` stays offline and does not verify provenance.
- A `pull_request_target` run can write `main`'s Actions cache scope through the runtime token, so no workflow in this repository restores a cache any more: `ci.yml` and `release.yml` set `cache: false` on `actions/setup-go`. Builds are slower.
- `ci.yml` is unchanged in purpose. It still runs on `pull_request` and dogfoods `./action`. Its evidence artifact is untrusted feedback, and a consumer never selects it, because step 2 checks the workflow path and event.
