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
| `spec-from` | `head` | `base` verifies against the spec at the merge base, so a change cannot weaken its own checks |
| `report` | `$RUNNER_TEMP/maslow/verify.json` | Evidence file path |
| `strict` | `false` | Also fail on `needs_review` |
| `working-directory` | `.` | Directory to run in |
| `artifact-name` | `maslow-evidence` | Name of the uploaded evidence artifact |

Outputs: `verdict` (`pass`, `fail`, `needs_review`, `inconclusive`) and `report`.

A `needs_review` verdict passes the step unless `strict` is set. Read `steps.<id>.outputs.verdict` to request review from the owners named in the evidence.

## A limit to know about

Checks run in the same job that writes the evidence, so code under test could overwrite the report. The default report path is outside the workspace, which narrows this. Closing it needs the checks and the evidence writer in separate jobs, which is a choice for your workflow.
