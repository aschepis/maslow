---
id: 45
title: Run checks and contracts isolated from the evidence writer
status: draft
priority: medium
created: 2026-11-19
updated: 2026-11-21
assigned_to: ""
assigned_at: ""
depends_on: [35]
tags: [kind:gap, evidence, security]
---

## Objective

Task 35 moved evidence production into a `pull_request_target` workflow that `main` defines (ADR
0040). One risk remains, and it was accepted. The checks and contracts still run the change's code in
the job that writes `verify.json`. On a GitHub-hosted runner that code has passwordless sudo, so a
change written to do it can:

- rewrite the report after `maslow verify` writes it, or edit it before the upload step;
- leave a background process running after its check, because Maslow does not run checks in their
  own process group or kill one when the check ends;
- read the runner's `ACTIONS_RUNTIME_TOKEN` and upload its own `maslow-evidence-<head>` artifact
  before the real upload.

The evidence's results are trustworthy only if the code under test cannot touch the evidence writer.

## Questions to answer

- Where do checks run? Options include a container or VM with no access to the report path or the
  runner's tokens, a separate job that hands back only raw results that the writer re-validates, or
  a remote executor.
- What does the writer trust from the isolated run, and how does it know the results are complete?
- How does this generalise beyond GitHub Actions, given Maslow stays a verifier (ADR 0016)?
- Is process-group cleanup worth doing anyway as cheap hardening, even though it is not a trust fix?

## Notes

Found while implementing task 35. See ADR 0040, "What this proves and what it does not".

Review (2026-11-20): process-group cleanup is split out as task 49. The isolation design stays a
draft until Damocles acts on evidence without a human reading the diff.

## Proposal (2026-11-21)

Maslow cannot isolate code, and should not try: building sandboxes is the caller's job, as
starting services is (ADR 0047). Maslow can own two narrow things. First, a split, so the job
that runs the change never writes or publishes the evidence. Second, a hook that lets the caller
run each command inside a sandbox the caller provides, so the Maslow process sits outside it.

The framing needs one correction. The split alone does not make check results trustworthy. If
the change runs with sudo in the same VM as `maslow verify`, it can forge any result Maslow emits
there. The split protects everything that needs no code execution (spec, policy, refs,
completeness, verdict, publishing). Only the sandbox hook (phase 2) protects the results.

**Phase 1: split execution from finalising.** Smallest valuable step.

- `maslow verify --no-policy` skips policy and refs. It writes `policy_skipped: true` and verdict
  `inconclusive`, so raw results can never pass a gate, even with an old `evidence check`.
- New `maslow evidence finalize <results> --spec <file> --profile <name> --head <sha> --base <sha>
  [--overlay <file>]... [--source-run <id>] --report <file>`. It never runs a check. It:
  1. parses the results strictly (unknown fields and status values are an error);
  2. requires `git_sha`, `base_sha`, `profile`, `spec_sha256` and overlay hashes to equal its own
     trusted inputs;
  3. checks completeness: every check, contract scenario and budget the profile selects has
     exactly one result, with no extras or duplicates;
  4. drops any policy, refs, coverage and verdict in the input, runs policy and refs itself on
     `git diff base..head` in its own checkout, then recomputes coverage and the verdict;
  5. adds a `finalize` block: `source_sha256`, `source_run`, `completeness: pass`.
  A failure in steps 1 to 3 gives verdict `inconclusive`, with the reasons in the `finalize`
  block, and exits 1. Unreadable trusted inputs exit 2.
- `evidence check --require-finalized` declines evidence without a `finalize` block.
- Workflows. `maslow-evidence.yml` keeps its `verify` job but runs `--no-policy` and uploads
  `maslow-results-<head>`. The `record` job goes away. A new `maslow-evidence-finalize.yml` runs
  on `workflow_run` (completed) of that workflow. It has `permissions: { actions: read }`. It
  checks the source run through the API (path, event `pull_request_target`, head SHA, same repo),
  downloads its artifact, builds maslow from the base, fetches the head without credentials, and
  runs `evidence finalize`. It never runs the change: no build, no script, no action from the head.
  It uploads `maslow-evidence-<head>` and writes the summary the `record` job used to write.
- Why a separate run, not a job in the same run: every job's `ACTIONS_RUNTIME_TOKEN` can upload
  any artifact name in its own run, and the artifacts API does not say which job uploaded one.
  A separate run has its own token, which the change never sees.
- Consumers (ADR 0040 steps, updated): find `maslow-evidence-<head>`, keep it only if its run's
  `path` is `maslow-evidence-finalize.yml` and `event` is `workflow_run`, then look up
  `finalize.source_run` and check it as today. Then run `evidence check --require-finalized`.
- `evidence_version` stays `"2"`. The new fields are additive, as `flaky` was (ADR 0039).
- `scripts/check-trusted-workflow.sh` learns the finalize workflow's rules (one trigger, only
  `actions: read`, no secrets, nothing run from the head). zizmor's `dangerous-triggers` finding
  is suppressed for it with a recorded reason.

**Phase 2: a caller-provided sandbox for execution.**

- `maslow verify --exec-wrapper <argv>` (CLI only, never in the spec). Maslow runs each check, CLI
  step, setup and teardown as `<argv> sh -c <command>`. It passes the scrubbed environment (ADR
  0046, 0048) and the working directory to the wrapper. It knows nothing about Docker.
- A reference wrapper and workflow in `action/`: the caller starts a container before verify
  (non-root user, `--cap-drop ALL`, `no-new-privileges`, repo mounted, no Docker socket, no
  `ACTIONS_*` or `GITHUB_*` variables, report directory not mounted, network optional). Services
  under test start inside it too (ADR 0047). The job tears it down.
- Evidence records `execution: {wrapper: "<argv0>"}`, so a consumer can require it.
- On GitHub-hosted runners the runner user can use Docker, which is root on the host. That is
  safe only while the container gets no socket and no privileges. Timeouts kill the wrapper's
  process group. Killing `docker exec` does not kill the process in the container, so the
  reference wrapper must handle that.

**Phase 3 (optional): attestations.** The finalize job alone gets `id-token: write` and
`attestations: write` and runs `actions/attest-build-provenance` on `verify.json`. Consumers can
run `gh attestation verify --signer-workflow`. It needs a public repo or Enterprise Cloud, so it
stays optional, as ADR 0040 planned.

Beyond GitHub Actions, phases 1 and 2 are plain CLI: any CI with a privileged step that never runs
the change, and a way to pass it a file, can use them.

### What this proves and what it does not

- Phase 1 proves: the published file came from a run that executed no change code. The spec,
  policy, refs, verdict and completeness were computed there by `main`'s verifier. A change can no
  longer skip or fake policy, drop a failing check from the report, or plant its own artifact
  in the run consumers trust.
- Phase 1 does not prove that any check or contract result is true. They come from a job the
  change controls. We know which job produced them, not what really happened in it.
- Phase 2 adds: a forged result needs a container escape, not just sudo. It does not cover a
  wrapper the caller misconfigured, kernel bugs, or a check that passes because the change
  weakened the test itself (that is `test_integrity` and review).
- Budgets stay untrusted in both phases: they measure files the change's checks built.

### Alternatives considered

- **Finalize as a second job in the same run.** Rejected: the untrusted job can upload artifacts
  into that run, and a consumer cannot tell which job uploaded one.
- **Maslow runs checks in a container itself.** Rejected: Maslow would pick images, mounts and
  runtimes. That is the orchestrator ADR 0047 refused. The wrapper keeps that with the caller.
- **Attestations only.** They say which workflow signed the file, not that its contents are
  honest. Useful on top of phase 1, not instead of it.
- **Remote executor.** Out of scope under ADR 0016. A caller can still point `--exec-wrapper` at
  one (for example over SSH).
- **Self-hosted ephemeral runners or a VM per check.** Valid for callers, not something Maslow
  should build. The wrapper supports it.

## Questions for the human

1. Is a separate `workflow_run` finalize run acceptable, given zizmor flags the trigger?
   Recommended: yes. It is the only way the untrusted job cannot write into the run consumers
   trust, and the finalize job never runs head code.
2. Is `--exec-wrapper` inside Maslow's scope after ADR 0047? Recommended: yes. The caller builds
   and tears down the sandbox; Maslow only prefixes a command, as it only reads `base_url`.
3. Bump `evidence_version` to `"3"`? Recommended: no. The fields are additive, and raw results
   are `inconclusive`, so an old `evidence check` already declines them.
4. Should finalize run refs as well as policy? Recommended: yes. Refs read files and URLs and run
   no change code. Budgets stay with the untrusted results.
5. Should `evidence check` require a `finalize` block by default? Recommended: not yet. Ship
   `--require-finalized`, make it the default in the next breaking release.
6. Build attestations (phase 3) now? Recommended: no. Do it after phase 2, as an optional signal.

## Acceptance criteria (phase 1)

- [ ] ADR recording the split, the separate finalize run and the trust model.
- [ ] `verify --no-policy` skips policy and refs, writes `policy_skipped: true` and verdict
      `inconclusive`; `evidence check` declines it with a reason naming `policy_skipped`.
- [ ] `evidence finalize` implemented with strict parsing, trusted-field matching, completeness,
      its own policy and refs, recomputed coverage and verdict, and the `finalize` block.
- [ ] Tests: missing, extra and duplicate results; wrong spec hash, head, base, profile or
      overlay; a tampered verdict or policy section in the input is replaced; exit codes 0/1/2.
- [ ] `evidence check --require-finalized`.
- [ ] `maslow-evidence.yml` uploads `maslow-results-<head>`; `record` removed;
      `maslow-evidence-finalize.yml` added; `check-trusted-workflow.sh`, its fixtures and
      `zizmor.yml` updated.
- [ ] `action/README.md` and ADR 0040's consumer steps updated; CLI contract for `evidence
      finalize` in `maslow.yaml`; `docs/MAP.md` and `CHANGELOG.md` updated.
- [ ] A real pull request gets finalized evidence that `evidence check --require-finalized`
      passes, found by the updated consumer steps.
