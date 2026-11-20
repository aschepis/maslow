# Maslow GitHub Action

Runs `maslow verify`, uploads the evidence file, writes a job summary, and fails the job on a `fail` or `inconclusive` verdict. The Action only installs the binary and passes your inputs through; all verification logic is in Maslow.

```yaml
on: pull_request

jobs:
  maslow:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0          # --base needs history to find the merge base
      - uses: aschepis/maslow/action@<sha>
        with:
          version: 0.6.0
          profile: ci
          base: ${{ github.event.pull_request.base.sha }}
          spec-from: base
          report: ${{ runner.temp }}/maslow/verify.json
```

| Input | Default | Meaning |
|---|---|---|
| `version` | | Release to download; its archive is checked against the release's `checksums.txt` |
| `binary` | | Use this maslow binary instead of downloading one |
| `profile` | `ci` | Profile to run |
| `spec` | `maslow.yaml` | Spec path |
| `base` | | Judge the change since the merge base with this ref or SHA |
| `spec-from` | `head` | `base` verifies against the spec at the merge base, so a change cannot weaken its own checks. If the base has no spec, as on the change that adopts Maslow, the branch's spec is used and the verdict is at best `needs_review` |
| `overlays` | | Overlay files of extra contracts, one path per line (`--overlay`) |
| `report` | `$RUNNER_TEMP/maslow/verify.json` | Evidence file path |
| `strict` | `false` | Also fail on `needs_review` |
| `working-directory` | `.` | Directory to run in |
| `artifact-name` | `maslow-evidence` | Name of the uploaded evidence artifact |

Outputs: `verdict` (`pass`, `fail`, `needs_review`, `inconclusive`) and `report`.

The job summary is printed by `maslow evidence summary`. From the first release with that command, it also lists observe-mode policy findings (`would_fail`, `would_need_review`, `would_error`) in their own section, which does not affect the verdict. With an older `version`, the Action writes its previous summary, which omits them.

A `needs_review` verdict passes the step unless `strict` is set. Read `steps.<id>.outputs.verdict` to request review from the owners named in the evidence.

## Trusted evidence

With `on: pull_request`, the workflow file is the pull request's own copy, so a change can edit it to skip `maslow verify` and upload a hand-written evidence file. If something acts on the evidence without a human reading the diff, produce it from a `pull_request_target` workflow instead. Its definition always comes from the base branch. Maslow's own `.github/workflows/maslow-evidence.yml` follows this pattern, and ADR 0040 records the trust model.

```yaml
name: Maslow evidence

on:
  pull_request_target:
    branches: [main]

permissions: {}

jobs:
  verify:
    # Fork pull requests never get trusted evidence; give them your ordinary pull_request workflow.
    if: github.event.pull_request.head.repo.full_name == github.repository
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@<sha>
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          fetch-depth: 0
          persist-credentials: false
      # Install your toolchains here, with every cache off (cache: false on actions/setup-go).
      - uses: aschepis/maslow/action@<sha>
        with:
          version: 0.8.0
          profile: ci
          base: ${{ github.event.pull_request.base.sha }}
          spec-from: base
          artifact-name: maslow-evidence-${{ github.event.pull_request.head.sha }}

  # Never checks out or runs the change.
  record:
    needs: verify
    if: ${{ !cancelled() && github.event.pull_request.head.repo.full_name == github.repository }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@<sha>
        with:
          name: maslow-evidence-${{ github.event.pull_request.head.sha }}
          path: ${{ runner.temp }}/evidence
      - name: Record the evidence
        env:
          EVIDENCE: ${{ runner.temp }}/evidence/verify.json
        run: sha256sum "$EVIDENCE" >> "$GITHUB_STEP_SUMMARY"
```

The rules that keep this safe:

- Top-level `permissions: {}`. The `verify` job gets `contents: read` and nothing else.
- No secrets and no `github.token` anywhere in the workflow, since `verify` runs the change's code.
- `persist-credentials: false` on the checkout, so no token is left on disk for that code to read.
- No `actions/cache`, and no caching setup action, in any job. A `pull_request_target` run writes to the base branch's cache scope, so the change could poison what `main` and release builds restore. Turn the cache off in your other workflows too.
- The same-repository `if:` on every job.

`scripts/check-trusted-workflow.sh` in this repository checks these rules. Copy it, or run [zizmor](https://docs.zizmor.sh), which flags `pull_request_target` itself. Suppress that finding for this one workflow with a recorded reason, as `zizmor.yml` here does.

**Consuming the evidence.** Do not trust an artifact because of its name, and never take a run ID from the pull request. Look the run up yourself:

1. `GET /repos/O/R/actions/artifacts?name=maslow-evidence-<head sha>` lists candidates.
2. For each candidate, `GET /repos/O/R/actions/runs/<workflow_run.id>`. Keep it only if `path` is your evidence workflow file, `event` is `pull_request_target`, `head_sha` is the head SHA, and both `repository.full_name` and `head_repository.full_name` are `O/R`. GitHub reports these fields, not the run. The `pull_request_target` event is what guarantees the workflow came from your default branch; `head_branch` is the pull request's branch, so do not check it.
3. Download it with `GET /repos/O/R/actions/artifacts/<id>/zip`.
4. Run `maslow evidence check verify.json --head <head sha> --spec <trusted spec>`.

This works on every GitHub plan, including private repositories.

**What this does not prove.** The checks still run the change's code in the job that writes the evidence, and that code has sudo. A change written to do it can rewrite the report or upload its own artifact first. This pattern stops forgery by editing the workflow, the spec or the verifier. It does not stop forgery by code under test. That code must be in the diff, where a reviewer or a policy rule can see it. The pull request that adds or edits the evidence workflow is not covered either, because the base branch's copy runs until it merges.
