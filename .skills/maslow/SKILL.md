---
name: maslow
description: >-
  Run Maslow verification and read its evidence. Use when a repository has a
  maslow.yaml, when asked to verify a change with maslow, when a maslow verdict
  or reports/verify.json needs interpreting, or when adding checks, contracts,
  budgets or policy to a maslow.yaml.
compatibility: Requires maslow CLI binary on PATH.
metadata:
  author: maslow
  version: "0.2"
---

# Maslow

Maslow verifies a repository against `maslow.yaml` and writes a JSON evidence file with one verdict. It does not decide how you work; it only tells you whether the repository meets its spec.

## Commands

```bash
maslow validate maslow.yaml       # schema-check the spec
maslow verify --profile quick     # fast checks while working
maslow verify --profile full      # everything, before handing a change off
maslow audit --profile full       # black-box checks against built artifacts
maslow init                       # add a starter maslow.yaml to a project without one
maslow version
```

Exit codes: `0` pass or needs_review, `1` fail or inconclusive (and needs_review with `--strict`), `2` usage error.

## Reading the Evidence

`maslow verify` writes `reports/verify.json` (override with `--report <path>`); `schema/evidence.schema.json` in the Maslow repository is its exact shape:

- `spec_source`, `spec_sha256`, `base_sha` — which spec was used and what the change was judged against.
- `verdict` — `pass`, `fail`, `needs_review` or `inconclusive`. Act on this first. `needs_review` means everything passed but the change touched a `gated` path (or the spec itself) and a named `owner` must approve it; do not try to work around it.
- `check_results` — one entry per check in the profile, with `status`, `output` and `error`. `output` holds the tail; `output_truncated` says it was cut.
- `contracts`, `budgets`, `refs`, `policy` — one entry per item, each with a `status` and, on failure, an `error` or `detail`.

To fix a failure, find entries whose `status` is `fail` or `error` and read their `error`, `detail` and `output`. A `policy` failure means the change touched a path the spec forbids; revert that part of the change rather than editing the policy.

## Growing a Spec

Add verification as you add code:

| When you... | Add to maslow.yaml |
|-------------|-------------------|
| Create an API endpoint | An HTTP contract scenario for it |
| Build a CLI command | A CLI contract scenario for it |
| Produce a build artifact | An `artifact_size` budget for it |
| Implement a performance-sensitive path | A performance budget for it |
| Add a dependency or config file | A ref for it |
| Create a file that a change must not touch | A `policy.deny` or `policy.protected` entry |

### Minimal Example

```yaml
mas: "1.0"
project:
  name: my-project

checks:
  runner:
    - name: build
      kind: command
      run: "make build"
      timeout: 120s
    - name: test
      kind: command
      run: "make test"
      timeout: 300s

profiles:
  quick:
    checks: [build]
  full:
    checks: [build, test]
```
