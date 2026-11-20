---
id: 35
title: Produce evidence outside the job that runs the code under test
status: done
priority: medium
created: 2026-11-18
updated: 2026-11-19
assigned_to: "claude-subagent"
assigned_at: "2026-11-19T16:55:02Z"
depends_on: [32]
tags: [kind:gap, action, evidence, security]
---

## Objective

Today the pull request's code runs in the same job that writes the evidence, so it could write its
own evidence file. A consumer cannot tell a real pass from a forged one.

## Requirements

- A documented two-job pattern for the GitHub Action: one job runs checks and contracts against the
  pull request's code; a second job, which never runs that code, assembles and publishes the
  evidence.
- The published evidence is signed or attested (e.g. GitHub artifact attestations) by the job that
  did not run the code.
- `maslow evidence check` (task 32) can verify the attestation when asked.

## Acceptance Criteria

- [x] `action/README.md` documents the pattern with a working example workflow. *Confirmed on the
  first real runs (PRs #36 to #38): the workflow passed, and the artifact lookup found run
  44651531826. `head_branch` turned out to be the PR's branch, so the consumer steps now check
  `head_sha` and `head_repository` instead.*
- [x] Maslow's own repository runs the trusted workflow as its own file beside `ci.yml` (Decision 7):
  `.github/workflows/maslow-evidence.yml`. *The first real run happens on the first pull request
  after merge. Check then that the run's `head_branch` is `main` and the artifact lookup in ADR 0040
  finds it, and that a fork pull request skips both jobs.*
- [x] ADR records the trust model and what the provenance does and does not prove (ADR 0040; no
  attestations, per Decision 5).

## Notes

Source: Damocles reconciler design note (18 Nov 2026), feature V4. This is the known limit noted in
the Maslow adoption plan.

## Decisions (human review, 2026-11-19)

These override the Requirements, Acceptance Criteria and proposal below where they differ.

1. Trigger: `pull_request_target`, workflow definition always from `main`.
2. `evidence check` does not verify provenance. Building incrementally.
3. The remaining risk (code under test rewriting the report inside `verify`) is accepted; file a
   `kind:gap` draft task for running checks in isolation.
4. Fork PRs never get trusted evidence. The trusted workflow's jobs run only when the PR's head
   repository is this repository; forks get ordinary `ci.yml` feedback.
5. Provenance comes from the GitHub API, not artifact attestations or Sigstore: the consumer looks
   up the run itself and checks its workflow path, `pull_request_target` event, `main` workflow ref
   and head SHA, then downloads the evidence artifact from that run. This works on every GitHub
   plan, including private repos. The ADR and `action/README.md` document the consumer's steps;
   attestations are out of scope and may be added later as an optional stronger signal.
6. Workflow linting uses zizmor as an ordinary `maslow.yaml` check (pinned version, `zizmor.yml`
   config, with a recorded suppression of `dangerous-triggers` for this one workflow). Our own
   invariants (the PR code never runs with write scope or secrets, no cache in `verify`) stay in a
   small script check like `check-action-pins.sh`. Another CI system swaps the check command; no
   Maslow code change.
7. The "Maslow's own `ci.yml` uses it" criterion becomes: Maslow's own repository runs the trusted
   workflow as its own file beside `ci.yml`.

## Proposed implementation

### 1. What happens today

Everything that produces evidence runs in one job whose definition the pull request controls:

- `ci.yml` runs on `pull_request` (`.github/workflows/ci.yml:3-6`), so the workflow file used is the
  pull request's own copy. The `verify` job builds `bin/maslow` from the head and then runs the Action
  with `binary: bin/maslow` (`ci.yml:62-74`). The verifier, its build and the code under test are all
  the change's code. `spec-from: base` (`ci.yml:74`, ADR 0030) protects the spec, but nothing protects
  the workflow: `.github/workflows/**` is not in `policy.gated` or `policy.protected`
  (`maslow.yaml:60-75`). The cheapest forgery is not clever test code. It is editing `ci.yml` to drop
  the verify step and upload a hand-written `verify.json`.
- The Action runs `maslow verify`, which runs every check and contract as a child process of the same
  runner user (`internal/runner/runner.go:79`, `internal/runner/contract.go:522`). Children are not
  put in a process group or killed as a group, so a background process started by a check outlives
  it. The report is a plain `os.WriteFile` at the end (`internal/evidence/report.go:256-273`), and the
  Action uploads whatever is at that path afterwards (`action/action.yml:207-213`). `rm -f` before the
  run (`action.yml:157`) and the default path outside the workspace (`action.yml:37`) only stop stale
  or accidental reports.
- `ci.yml` grants only `contents: read` (`ci.yml:8-9`), so nothing can sign or attest today.
  `actions/setup-go` runs with its default cache on (`ci.yml:26-28`, `release.yml:16-18`).
- `evidence check` (`internal/evidence/check.go:66-107`, `cmd/maslow/evidence.go:39-114`) compares the
  file's claims with trusted inputs. It has no notion of who wrote the file. ADR 0037 and task 44's
  Notes say so: the evidence is self-reported.
- `action/README.md:44-46` documents the limit and leaves the fix to the adopter.
- The release job runs `make verify` before publishing (`release.yml:23-27`) but publishes no evidence
  and attests nothing. The Action trusts a release by its `checksums.txt` (`action.yml:113-129`),
  which comes from the same release.

### 2. Proposed change

**What each option proves.** All four assume GitHub-hosted runners, where code in a job runs as a user
with passwordless sudo. Inside one VM, nothing the change's code runs next to is safe from it.

| Option | Proves | Does not prove |
|---|---|---|
| A separate job checks out the trusted ref, downloads the build artifact and runs verify on it | The verifier binary, spec and overlays came from the trusted ref; the change's build scripts and tests did not run in that job | That the artifact was built from the head, since a `pull_request` job the change controls built it. Contracts still run the artifact in the evidence job, where it can rewrite the report. Checks such as `go test` need source and cannot run there, so they are dropped or taken on trust |
| `workflow_run` gating | The gating workflow's definition comes from the default branch, not the change | Anything about content it takes from the triggering `pull_request` run: that run's definition is the change's, so attesting its artifact launders a forgery |
| Artifact attestations or Sigstore keyless signing | The file's bytes are unchanged since a named workflow file, at a named ref, in a named repo and run, signed them | That the workflow ran honestly. A job that runs the change's code with `id-token: write` exposes `ACTIONS_ID_TOKEN_REQUEST_*` to that code, which can sign anything as that workflow. The signer must be a job that never runs the change |
| Hash of the evidence as a job output | Two jobs in one run saw the same bytes | Anything about the producer: the code that could forge the file can also write `$GITHUB_OUTPUT`, and an outside consumer cannot easily read job outputs |

No option makes check and contract results trustworthy while the change's code runs in the job that
records them. That needs the checks to run isolated from the evidence writer, for example in a
container with no access to the report path. That is a separate gap (open question 3).

**Recommendation.** Use a `pull_request_target` workflow, so its definition always comes from the
base branch. Give it two jobs, and attest the evidence from the job that never runs the change. This
closes the cheap forgeries: an edited workflow, a hand-written evidence file, and a weakened spec,
verifier or overlay. What remains is code under test that rewrites the report at run time. That has to
be written into the diff on purpose, where a reviewer or a policy rule can see it.

```yaml
on:
  pull_request_target:
    branches: [main]
permissions: {}
jobs:
  verify:                      # runs the change; no write scope, no secrets, no cache
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
      - uses: aschepis/maslow/action@<sha>
        with:
          version: <pinned>
          base: ${{ github.event.pull_request.base.sha }}
          spec-from: base
  attest:                      # never checks out or runs the change
    needs: verify
    if: ${{ !cancelled() }}
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      attestations: write
    steps:
      - uses: actions/download-artifact@<sha>
        with: { name: maslow-evidence, path: evidence }
      - uses: actions/attest-build-provenance@<sha>
        with: { subject-path: evidence/verify.json }
```

The consumer, which is Damocles, runs two commands and requires both to pass:

```
gh attestation verify verify.json --repo O/R \
  --signer-workflow O/R/.github/workflows/maslow-evidence.yml \
  --source-ref refs/heads/main --deny-self-hosted-runners
maslow evidence check verify.json --head <head> --base <merge-base> --spec <trusted spec> ...
```

`--source-ref` matters. A `pull_request` run of a same-named workflow is signed with
`refs/pull/N/merge`. A `pull_request_target` run of a PR into some other branch is signed with that
branch's ref. Both decline. `--head` in `evidence check` stops evidence from one PR being replayed for
another.

**What belongs in Maslow and what does not.** Maslow stays a verifier (ADR 0016). Signing, and
fetching attestations from GitHub, is the job of the workflow and the consumer.

- `internal/evidence`, `cmd/maslow`, `internal/verify`, `schema/evidence.schema.json`: no change.
  `evidence check` stays offline and deterministic. Verifying an attestation in Go would pull in
  sigstore-go, a TUF client and network calls, and it would duplicate `gh attestation verify`. See open
  question 2, which proposes dropping the third requirement.
- `action/action.yml`: no change, and the Action must not gain an `attest` input. It runs in the job
  that runs the change, which is exactly where signing must not happen. It already uploads the
  evidence under `artifact-name`, which is all the attest job needs.
- `action/examples/trusted-evidence.yml` (new): the adopter template above, with every action pinned
  by SHA.
- `action/README.md`: replace "A limit to know about" (`README.md:44-46`) with a "Trusted evidence"
  section. It links the template, gives the consumer's two commands, and states the prove/does-not-prove
  table. It also covers the setup rules: top-level `permissions: {}`, no secrets in `verify`,
  `persist-credentials: false`, and no `actions/cache` or setup cache in `verify`. A
  `pull_request_target` cache is saved to the base branch's scope, so the change could poison the
  cache that `main` and release runs restore. It also notes that attestations on private repositories
  need GitHub Enterprise Cloud.
- `.github/workflows/maslow-evidence.yml` (new): Maslow's own instance. It differs from the template
  in one place: before running the change, `verify` builds maslow from the base SHA into
  `$RUNNER_TEMP` (`git worktree add` plus `go build`, with `actions/setup-go` and `cache: false`). It
  then passes `binary:` to the Action, so the verifier is trusted code and the head's `bin/maslow` is
  only exercised by contracts. It keeps `MASLOW_REQUIRE_BROWSER: "1"` and the CUE install from
  `ci.yml`.
- `.github/workflows/ci.yml`: limit the "Verify (full profile)" step (`ci.yml:67-74`) to `push`, so
  each pull request has only one `maslow-evidence` artifact, the trusted one. Hygiene, test, lint and
  release-smoke steps stay. Turn off the setup-go cache in `release.yml`, or document why it is safe.
- `maslow.yaml`: add the new workflow and the template to `refs`. Add `.github/workflows/**` and
  `action/examples/**` to `policy.gated` with owner `@aschepis`. Add the workflow check and contract
  described in section 3.
- `docs/MAP.md`: add the evidence workflow and the trust model to the Action row and the evidence
  section. `CHANGELOG`: note the README section and the template.

### 3. Tests

The trust model lives entirely in workflow YAML, and nothing runs it locally. So the tests make its
invariants mechanical, as `check-action-pins.sh` does for pins (ADR 0038).

**Workflow invariant checker** (`tools/workflowcheck/`, a small Go command using the existing
`gopkg.in/yaml.v3`; exit 0 clean, 1 findings, 2 usage). First, write `workflowcheck_test.go` and the
fixtures in `testdata/evidence-workflow/`, with a stub `Check` that returns no findings. Every
bad-fixture case then fails on its assertion, not on a compile error. Table cases:

| Fixture | Expected finding |
|---|---|
| `good.yml` (the template) | none |
| `pull-request-trigger.yml` | trigger is not `pull_request_target` or `workflow_run` |
| `top-level-id-token.yml` | top-level `permissions` is not `{}` |
| `verify-job-id-token.yml` | job that checks out the head has a write scope |
| `attest-job-checkout.yml` | job with `id-token: write` uses `actions/checkout` |
| `attest-job-run-step.yml` | job with `id-token: write` has a `run` step |
| `attest-job-local-action.yml` | job with `id-token: write` uses a `./` action |
| `two-signing-jobs.yml` | more than one job has `id-token: write` |
| `head-persist-credentials.yml` | head checkout without `persist-credentials: false` |
| `setup-go-cache.yml` | `actions/setup-go` without `cache: false` in a job that checks out the head |
| `actions-cache.yml` | `actions/cache` in a job that checks out the head |
| missing file | exit 2 |

**`maslow.yaml`**: add a check, `trusted-evidence-workflow`, that runs the checker on
`.github/workflows/maslow-evidence.yml` and `action/examples/trusted-evidence.yml`. Write it before the
files exist, so it fails first. Add a contract, `trusted-evidence-workflow-check`, beside
`action-pins-check` (`maslow.yaml:552`): `good.yml` exits 0, and `attest-job-checkout.yml` exits 1
with the finding in the output.

**Not automatable, recorded in the PR's Testing section:** after merge, on the first pull request, run
`gh attestation verify` against its evidence with the flags above (pass). Run it again with
`--source-ref refs/pull/<n>/merge` (fail). Then push a commit that edits `maslow-evidence.yml` and show
that `main`'s copy ran, not the edited one.

`go test ./...`, `go vet`, `cue vet schema/maslow.cue` and `maslow verify --profile full` stay green.
No existing test changes.

### 4. ADR

Write ADR 0040, "Trusted evidence comes from a base-branch workflow and is attested by a job that never
runs the change". Its decision sentence: the documented pipeline produces evidence in a
`pull_request_target` workflow, and signs it with GitHub artifact attestations from a separate job that
has `id-token: write` and never checks out or runs the change. Consumers check it with
`gh attestation verify --signer-workflow --source-ref` before `maslow evidence check`. Its consequences
state what the attestation proves (workflow file, ref, repo, run and unchanged bytes) and what it does
not prove (that code under test left the report alone). They also note that `evidence check` stays
offline with no new flag, and that attestations on private repositories need GitHub Enterprise Cloud.
Amend the last sentence of ADR 0037's consequences to point at 0040, and the Status line of ADR 0024
to note the template. Those are the only edits to accepted ADRs.

### 5. Risks and open questions

1. **`pull_request_target` or `workflow_run`?** Both run the base or default branch's definition, so
   they are equally trustworthy. `workflow_run` adds a dependency on a CI run whose outputs must not be
   trusted, waits for that run to finish, and gives no PR SHAs for forks (`pull_requests` is empty).
   *Recommendation:* `pull_request_target`, with `branches: [main]` and `--source-ref` pinned by the
   consumer.
2. **Drop the requirement that `evidence check` verifies the attestation?** Doing it in Maslow means
   sigstore-go, network access and GitHub API calls in an offline, deterministic command, and it
   repeats `gh attestation verify`. *Recommendation:* drop it, and change the requirement to "the README
   gives the consumer's verification commands". Revisit if a consumer cannot run `gh`.
3. **Code under test can still rewrite the evidence inside `verify`.** It has sudo, and Maslow does not
   kill a check's process group. *Recommendation:* accept and document this as the stated limit. File a
   `kind:gap` draft task for running checks and contracts isolated from the evidence writer, which is
   the only thing that would make results provable. Killing process groups is cheap hardening, not a
   trust fix, so leave it out of this task.
4. **Running the change in a privileged-trigger workflow (the "pwn request" pattern).**
   *Recommendation:* proceed, because `verify` gets `contents: read`, no secrets and no cache, and the
   checker enforces that. Run only same-repository PRs at first (the `if:` above). Damocles agents push
   branches to the same repository, and fork PRs fall back to untrusted `ci.yml` feedback.
5. **Acceptance criterion "Maslow's own `ci.yml` uses it".** *Recommendation:* reword it to "Maslow's
   own workflows use it". The pattern needs its own `pull_request_target` file, and `ci.yml` stays a
   `pull_request` workflow.
6. **Which maslow runs the trusted verify for Maslow itself?** If it is the head build, the change
   judges itself. *Recommendation:* build from the base SHA. The head binary is still exercised by the
   contracts, and with `spec-from: base` the base binary and the base spec match.
7. **Bootstrap.** The PR that adds `maslow-evidence.yml` is not covered, because `main` lacks the file.
   Later edits to it take effect only after merge. *Recommendation:* accept, as ADR 0030 does for the
   spec, and gate `.github/workflows/**` so workflow edits need the owner's review.
8. **Private repositories without GitHub Enterprise Cloud have no attestations.** *Recommendation:*
   document a fallback. The consumer downloads the artifact through the API from a run it looked up
   itself, and checks that run's `path`, `event` (`pull_request_target`) and `head_sha`. GitHub
   reports those fields, not the run. This proves the same thing as the attestation but only through
   the API. Signing with cosign on the public Sigstore instance would publish private repository names
   to Rekor, so it is not recommended.
9. **Use zizmor instead of our own checker?** It flags dangerous triggers and cache poisoning, but it
   does not encode "the signing job never runs the change", and it would flag our deliberate
   `pull_request_target`. *Recommendation:* keep our own narrow checker. Revisit if the rules grow.
10. **Attesting the release archives** so adopters can verify the binary the Action installs, instead
    of trusting the release's own `checksums.txt`. *Recommendation:* out of scope; open a separate task.
