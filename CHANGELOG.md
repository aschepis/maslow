# Changelog

Notable changes to Maslow, newest first. Versions follow [semantic versioning](https://semver.org/); until 1.0, a minor version may break specs.

## [Unreleased]

### Added

- **`evidence check --require-red-green`.** With `--base <sha>` and a trusted overlay, `maslow evidence check` declines unless every overlay contract failed on exactly that base and passed on the head, so evidence from a run without `--red-green` cannot pass the gate. Each failing contract gets its own reason. `--base` alone checks the evidence's `base_sha`. No evidence format change; v0.8.0 `--red-green` evidence already passes. (ADR 0037)
- **`maslow evidence summary <file>`.** Prints an evidence file as Markdown: the verdict, run metadata, the enforced results that failed or need review, and observe-mode policy findings in a separate "Observed policy rules (not enforced)" section. Exit 0, 1 if the file cannot be read or parsed, 2 on wrong arguments. (ADR 0036)

### Changed

- **The Action's job summary now lists observe-mode policy findings** (`would_fail`, `would_need_review`, `would_error`), closing the gap noted under 0.8.0. The step runs `maslow evidence summary` and falls back to its old jq when the installed release lacks the command. It also lists `not_red` contracts, which the old summary omitted. (ADR 0036)

## [0.8.0] - 2026-11-18

### Added

- **Browser contract steps.** A `browser` step drives a headless Chrome through chromedp: `navigate`, `click`, `fill`, `wait`, and `assert` on text, URL, visibility or enabled state. It shares the scenario's cookies with HTTP steps and resolves `/` URLs against `base_url`. A failing step records `page_url` and `console` on `failed_step`, and a screenshot in `artifacts`. Needs Chrome or Chromium, found on the usual paths or named by `MASLOW_BROWSER`. Maslow's own CI has no Chrome, so these tests skip there. (#26, ADR 0035)
- **Red-then-green overlay contracts.** `maslow verify --red-green --base <ref> --overlay <file>` runs each overlay contract on the head and again in a temporary checkout of the merge base. A contract that passes on both gets status `not_red`, which fails the verdict. Maslow does not build the base; a contract's `setup` must. (#25, ADR 0034)
- **`maslow evidence check`.** Given an evidence file, `--head <sha>` and a trusted `--spec`, answers `pass`, `needs_review` or `decline`. It declines on any mismatch of evidence version, commit, spec hash, profile (with `--profile`) or overlay hashes (`--overlay`, `--overlay-sha256`), and on a `fail` or `inconclusive` verdict. `--json` prints the decision and reasons. Exit 0 for pass, 1 otherwise. (#24, ADR 0033)
- **Shrink-only files and test-integrity policy.** `policy.shrink_only` fails a change that adds a line to a matching file. `policy.test_integrity` reports deleted test files, removed tests, added skip markers and lowered numbers in `thresholds` files as `needs_review`, with defaults for Go, Ruby, JavaScript and Python. A renamed test file is reported as deleted. (#23, ADR 0032)
- **Policy observe mode.** Rules under `policy.observe` (any policy rule kind) are evaluated and reported with `mode: observe` and a `would_fail`, `would_need_review` or `would_error` status, without changing the verdict or exit code. The Action's job summary does not list them yet. (#22, ADR 0031)

### Evidence format

The format is still `evidence_version: "2"`, and every new field is optional: `mode` and `lines` on policy results; `head_status` and `base` on overlay contract results; `page_url` and `console` on `failed_step`. There are new status values: `not_red` on contracts, and `would_fail`, `would_need_review` and `would_error` on observed policy results. A consumer that rejects statuses it does not know must add these. `schema/evidence.schema.json` describes them all.

## [0.7.0] - 2026-11-18

### Breaking

- **Contracts read only the environment variables they list.** A `{{.env.NAME}}` template now fails the step unless `NAME` is in the contract's new `env` list. Add an `env` list to each contract that reads the environment. Listed values of four or more bytes are replaced with `[redacted:NAME]` in the contract's evidence. (#12, ADR 0026)

### Added

- **Spec overlays.** `maslow verify --overlay <file>` (repeatable) runs a file of extra contracts beside the spec, on every profile. An overlay may hold only `description` and `contracts`, and cannot reuse a contract name from the spec. The evidence lists each overlay's path and SHA-256 under `overlays`, and the Action has a matching `overlays` input. (#15, ADR 0029)
- **Endpoint audit.** An `endpoint` audit target runs the contracts its `checks` select against its base URL, which may come from `{{.env.NAME}}`. It used to return `skip`. Contracts can set `base_url`, and HTTP and poll URLs that start with `/` resolve against it. (#14, ADR 0028)
- **HTTP sessions.** Each contract scenario has its own cookie jar, shared by its setup, steps, poll steps and teardown. A scenario that signs in stays signed in, including across redirects. (#13, ADR 0027)
- **Richer contract evidence.** A failing contract records `failed_step` (phase, index, action and the substituted command or URL) and that step's output, cut to the same `--output-limit` as check output. Scenarios can list `artifacts` patterns, and matching files such as screenshots are recorded. (#11, ADR 0025)

### Fixed

- **`--spec-from base` when the base has no spec.** The change that adopts Maslow is now verified against its own spec, and the verdict is at best `needs_review`. Any other failure to load the base spec writes evidence with verdict `fail` instead of exiting with no file. (#18, ADR 0030)
- **The Action pins `actions/upload-artifact` to a commit SHA** (v4.6.2), not the moving `v4` tag. A new `action-pins` check fails verification if any action in `action/action.yml` is not pinned to a full SHA. (#17)

### Evidence format

The format is still `evidence_version: "2"`. All new fields are optional and additive: `overlays` at the top level, and `overlay`, `failed_step`, `output`, `output_bytes`, `output_truncated` and `artifacts` on contract results. `schema/evidence.schema.json` describes them.

## [0.6.0]

Maslow became a verifier. The agent harness was removed and the module moved to `github.com/aschepis/maslow`. The verdict and `**` policy glob bugs were fixed. This release added base-ref policy (`--base`), `--spec-from base`, evidence format 2, `gated` paths with a `needs_review` verdict, profile scoping of contracts and budgets, a release pipeline and the GitHub Action. See ADRs 0016 to 0024.

[0.8.0]: https://github.com/aschepis/maslow/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/aschepis/maslow/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/aschepis/maslow/releases/tag/v0.6.0
