---
id: 42
title: Make CI fail, not skip, when the browser tests have no Chrome
status: done
priority: medium
created: 2026-11-18
updated: 2026-11-18
assigned_to: "claude-subagent"
assigned_at: "2026-11-19T02:40:57Z"
depends_on: []
tags: [kind:gap, ci, contracts, browser]
---

## Objective

The browser tests in `internal/runner/browser_test.go` skip when no Chrome is found. CI shows no
skip messages because `go test` runs without `-v`, so nothing in CI proves they run, and nothing would
notice if they stopped. The v0.8.0 CHANGELOG says they skip in CI. The evidence below says they run.
CI should prove the tests ran, fail if Chrome goes missing, and dogfood one browser contract through
the built binary.

## Requirements

- In CI, a missing browser fails the browser tests instead of skipping them. Local runs without
  Chrome still skip.
- CI logs which browser binary and version the tests used.
- `maslow.yaml` has at least one browser contract, run by `bin/maslow` in the full profile.
- The CHANGELOG line that says CI has no Chrome is corrected.

## Acceptance Criteria

- [x] With `MASLOW_REQUIRE_BROWSER=1` and `MASLOW_BROWSER=/nonexistent`, `go test -run Browser
      ./internal/runner/` fails with a message naming both variables.
- [x] Without `MASLOW_REQUIRE_BROWSER`, the same command still skips.
- [ ] A CI run on the PR shows the Chrome version and `--- PASS: TestBrowser_*` lines. (Needs a CI
      run; not yet observed.)
- [ ] `maslow verify --profile full` runs a browser contract that passes in CI. The quick profile is
      unchanged. (Passes locally with `bin/maslow`; CI uses `spec-from: base` on PRs, so the
      contract first runs in CI on `main` after merge.)
- [x] CHANGELOG v0.8.0 entry and ADR 0035 no longer say CI lacks Chrome.

## Notes

Found while checking the task 28 claim that "CI has no Chrome, so these tests skip there". The claim
was wrong. The tests have most likely run in CI on every push since PR #26.

## Proposed implementation

### 1. What happens today

- CI runs on `ubuntu-latest` (`.github/workflows/ci.yml:17`), which resolved to `Image: ubuntu-26.04,
  Version: 20261111.196.1` in run 44442980284. Tests run through `make test` (`ci.yml:45-46`), which
  is plain `go test ./...` (`Makefile:11-12`). Without `-v`, skips are not printed. Both CI logs
  (PR #26 run 44394817128, main run 44442980284) contain no `SKIP`, `chrome` or `browser` test lines.
- Chrome lookup (`internal/runner/browser.go:41-54`): `MASLOW_BROWSER` if set, otherwise
  `exec.LookPath` over `google-chrome`, `google-chrome-stable`, `chromium`, `chromium-browser`,
  `chrome`, `headless-shell` and the macOS and Windows paths (`browser.go:28-38`). GitHub's Ubuntu
  images install Google Chrome as `/usr/bin/google-chrome`, which the first candidate finds.
  `browser.go:71-74` turns off the sandbox only when running as root. The runner user is not root,
  and the tests pass with the sandbox on.
- `requireBrowser` (`internal/runner/browser_test.go:15-20`) calls `t.Skipf` whenever `findBrowser`
  errors. Nothing forces a failure in CI.
- Timing evidence that the tests run in CI (`internal/runner` package time):
  - Before PR #26 (task 31–34 runs on the same day): `0.651s`, `0.630s`, `1.688s`, `1.644s`.
  - PR #26 run: `ok github.com/aschepis/maslow/internal/runner 13.239s`.
  - Main after merge: `ok github.com/aschepis/maslow/internal/runner 15.081s`.
  - Locally on macOS with Chrome, the five browser tests pass in 1.3s, 0.9s, 0.8s, 1.9s and 0.8s,
    and the package takes `11.356s`. With `MASLOW_BROWSER=/nonexistent` they `SKIP` in 0.00s and
    `-run Browser` takes `0.260s`.
  - The non-browser code PR #26 added cannot explain a jump of about 12s. The browser tests ran.
- PR #26's own evidence flagged this pattern. In observe mode, `policy/test_integrity` reported
  `added_skips` at `internal/runner/browser_test.go:18`.
- **The original claim was wrong.** CI has Chrome. The tests run, but only the timing shows it, and a
  future image without Chrome would make them skip silently. `CHANGELOG.md:9` publishes the wrong
  claim, and ADR 0035 (line 16) describes the skip as the repository's general behaviour.
- `maslow.yaml` has no browser contract. A web server is not actually required: `navigate` passes
  absolute URLs through unchanged (`internal/runner/contract.go:479-485`), so a `data:text/html,...`
  URL needs no server. A server started from a `cli` step would not work. `runCLIStep`
  (`contract.go:517-536`) waits on `CombinedOutput` with a 30s timeout, so a backgrounded server
  would hang the step.
- Profiles: `quick` (`maslow.yaml:119-129`) runs only `smoke`-tagged contracts and has a 5s p95
  speed budget. `full` (`maslow.yaml:130-138`) runs every contract. CI runs `full` through
  `./action` (`ci.yml:56-63`).

### 2. Proposed change

- `internal/runner/browser_test.go`: in `requireBrowser`, when `MASLOW_REQUIRE_BROWSER` is non-empty,
  call `t.Fatalf` with the `findBrowser` error and a hint naming both variables. Otherwise skip as
  today. Use an explicit variable, not `CI`, so a contributor's CI-like local shell is not affected.
- `.github/workflows/ci.yml`:
  - Set job-level `env: MASLOW_REQUIRE_BROWSER: "1"`. Checks inherit the environment (nothing in
    `internal/` sets `cmd.Env` for checks), so both `make test` and the `go-test` check inside the
    full-profile Action step get it.
  - Add a step before `Test` that runs `google-chrome --version` and fails if it is absent. This puts
    the version in the log.
  - Add a step `go test -v -run Browser ./internal/runner/` so the CI log shows the `--- PASS` lines.
    It costs a few seconds because the same tests already ran.
  - Do not add `browser-actions/setup-chrome` for now. The preinstalled Chrome works, and the new
    failure makes its absence loud. If it is ever added, pin it by full commit SHA. `action-pins`
    (`maslow.yaml:111-116`, `scripts/check-action-pins.sh`) checks only `action/action.yml`, and
    `ci.yml` uses tag pins for first-party `actions/*` (`ci.yml:19,23`). A third-party action in CI
    still gets workflow secrets and write access through `GITHUB_TOKEN`, so it should follow the same
    rule. Optionally extend `action-pins` to cover third-party actions in `.github/workflows/*.yml`.
- `maslow.yaml`: add a contract `browser-steps` (not tagged `smoke`, so it runs only in `full`) with
  one scenario. It should `navigate` to a `data:text/html,` page holding an input, a button that
  writes into a `#out` element, and a delayed element. Then it should `fill`, `click`, `wait` for the
  delayed element and `assert` the text of `#out`. This drives Chrome through the built binary and
  the evidence pipeline without a server. Add a second scenario that asserts missing text and expects
  a failure only if the schema supports expected-failure scenarios. Otherwise leave it out.
  `maslow.yaml` is gated to `@aschepis`.
- `CHANGELOG.md`: change line 9 to "Maslow's own CI uses the runner's preinstalled Chrome and fails if
  it is missing."
- `docs/adr/0035-browser-contract-steps.md`: amend line 16 to say the tests skip locally without
  Chrome and are required in CI.
- `docs/MAP.md`: no change needed.

### 3. Tests and checks

- `MASLOW_REQUIRE_BROWSER=1 MASLOW_BROWSER=/nonexistent go test -run Browser ./internal/runner/`
  fails. Without `MASLOW_REQUIRE_BROWSER` it skips. With Chrome present it passes.
- `go test ./...`, `go vet ./...`, `cue vet schema/maslow.cue`, `go run ./cmd/maslow validate
  maslow.yaml`.
- `maslow verify --profile quick` is unchanged and stays within its budget.
- `maslow verify --profile full` locally (Chrome present) shows `[pass] contract:
  browser-steps/...`.
- On the PR's CI run, confirm the Chrome version line and the `--- PASS: TestBrowser_*` lines. The
  Action step uses `spec-from: base` on pull requests, so the new contract runs only after merge.
  Confirm it on the first `main` run.

### 4. ADR

No new ADR. This makes the existing ADR 0035 decision accurate rather than making a new one. Amend
ADR 0035's consequence line instead. If `action-pins` is extended to workflows, record that in a
short ADR, because it changes what the check enforces.

### 5. Risks and open questions

- **`data:` URL contract versus a real served page.** A `data:` page does not exercise `base_url` or
  cookie sharing. Recommendation: accept that, because the Go tests cover both. Serving a page needs
  a background-process step. If that is wanted, file it as its own `kind:gap` task rather than fold
  it in here.
- **Full profile fails on developer machines without Chrome.** Recommendation: accept it. The failure
  names `MASLOW_BROWSER`, and the full profile already assumes a complete toolchain. Alternatively,
  put the contract in a new `browser` profile that CI runs as an extra step, at the cost of a second
  verify run and a second evidence file.
- **Runner image drift** (Chrome removed or a breaking version). Recommendation: rely on the
  fail-not-skip change and the version log line. Pin Chrome with `setup-chrome` only if drift
  actually breaks a run.
- **Task 41 (scrubbed check environment, draft).** If it lands, `MASLOW_REQUIRE_BROWSER` and
  `MASLOW_BROWSER` must be on the allow-list for the `go-test` check. Recommendation: note it in
  task 41 when it is promoted.
- **Extend `action-pins` to `.github/workflows/`?** Recommendation: yes for third-party actions only.
  Keep tag pins for `actions/*`, or pin those too if you prefer one rule everywhere. This is your call.
