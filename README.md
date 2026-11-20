# Maslow

A verifier for repositories. Maslow checks a repository against a declarative spec and writes evidence that a pipeline can trust.

Given a `maslow.yaml`, Maslow runs checks, behavioural contracts, budgets, reference checks and policy rules, then writes a JSON evidence file with one verdict. It runs the same way on a laptop, inside an agent session and in CI.

Maslow imposes no workflow. It does not generate agent instructions, manage tasks or dispatch agents. A project adopts it by adding one file.

## Install

```bash
go install github.com/aschepis/maslow/cmd/maslow@latest
```

Or download a release binary for linux or darwin (amd64, arm64) from the GitHub releases page and check it against `checksums.txt`:

```bash
v=0.6.0; os=linux; arch=amd64
curl -fsSLO "https://github.com/aschepis/maslow/releases/download/v${v}/maslow_${v}_${os}_${arch}.tar.gz"
curl -fsSLO "https://github.com/aschepis/maslow/releases/download/v${v}/checksums.txt"
grep "maslow_${v}_${os}_${arch}.tar.gz" checksums.txt | sha256sum -c -
```

Or build from source:

```bash
git clone https://github.com/aschepis/maslow.git
cd maslow
make build
```

## Adopting Maslow

```bash
cd your-project
maslow init                      # writes a starter maslow.yaml and detects your toolchain
maslow validate maslow.yaml
maslow verify --profile quick
```

Edit `maslow.yaml` to point the starter checks at your real build and test commands, then add contracts, budgets and policy as the project grows.

## Commands

```bash
maslow init                      # Add a starter maslow.yaml to an existing project
maslow validate <file>           # Validate a maslow.yaml against the schema
maslow verify --profile <name>   # Run verification and write evidence
maslow audit --profile <name>    # Run a black-box audit against built artifacts
maslow evidence check <file> --head <sha> --spec <file>  # Decide whether evidence approves a change
maslow evidence summary <file>   # Print evidence as the Markdown job summary
maslow version                   # Print version info
```

## Verification Pipeline

`maslow verify` runs the following in order:

1. **Checks** — shell commands (build, test, lint) filtered by profile
2. **Contracts** — scenario-based behavioural tests (CLI, HTTP and headless browser); a profile's `contracts` list selects by name or tag, and `--overlay <file>` adds a file of extra contracts for one run, such as a change's acceptance scenarios. With `--red-green --base <ref>`, each overlay contract also runs in a temporary worktree of the merge base and passes only if it fails there; one that already passes on the base gets status `not_red` and fails the verdict, because it proves nothing about the change (ADR 0034). The base checkout is not built: put any build the scenario needs in the contract's `setup`, which runs in that checkout
3. **Budgets** — artifact size, performance, and complexity limits; selected by a profile's `budgets` list
4. **Refs** — verify declared files, binaries, and URLs exist
5. **Policy** — `deny`, `protected`, `allow`, `gated`, `shrink_only` and `test_integrity` rules, judged on the change since `--base <ref>` when given. Rules under `policy.observe` are evaluated and reported with `mode: observe` and a `would_fail`, `would_need_review` or `would_error` status, but never change the verdict or exit code, so a new rule can be tried on real changes before it is enforced:

   ```yaml
   policy:
     protected: [maslow.yaml]
     observe:
       protected: [go.sum]   # reported, not enforced
   ```

   Two rules read the changed lines, not just paths. `shrink_only` files may lose lines but never gain any, and fail otherwise. `test_integrity` asks for review when a change deletes a test file, removes a test, adds a skip marker, or lowers a number in a `thresholds` file such as a coverage floor; it has defaults for Go, Ruby, JavaScript/TypeScript and Python, and `languages` overrides or adds them (ADR 0032). Flagged lines appear in the evidence entry's `lines`:

   ```yaml
   policy:
     shrink_only: [config/authz_baseline.txt]
     test_integrity:
       thresholds: [coverage.yml]
       languages:
         go: {}                     # built-in defaults
         elixir:
           files: ["test/**/*_test.exs"]
           skips: ["@tag :skip"]
   ```

The verdict is `fail`, `needs_review`, `inconclusive` or `pass`, in that order of precedence. `needs_review` means everything passed but the change touched a `gated` path, and names its owner, or an overlay contract was flaky. It exits 0 unless `--strict` is passed.

A check or contract can be rerun when it fails, so a flake is reported as one instead of as a failure (ADR 0039):

```yaml
max_retries: 3          # cap for every retries below; defaults to 5
checks:
  runner:
    - name: integration
      kind: command
      run: "go test ./integration/..."
      retries: 2        # default 0
contracts:
  - name: api
    retries: 1          # each failing scenario reruns with the contract's setup and teardown
    scenarios: [...]
```

Only `fail` is retried, never `error`, and there is no delay between attempts. A result that fails and then passes has status `flaky` and lists every run in `attempts`. A flaky check or spec contract counts as a pass; a flaky overlay contract makes the verdict `needs_review`, and under `--red-green` a base run that fails and then passes is `not_red`. `maslow evidence summary` lists flaky results in their own section.

Results are written to `reports/verify.json` (or `--report <path>`) as structured evidence. `schema/evidence.schema.json` describes the format. `maslow audit` writes `reports/audit.json`; an `endpoint` audit target runs the contracts it selects against a deployed base URL, resolving step URLs that start with `/`, so the same scenarios work as a post-deploy smoke test.

## Checking Evidence

A merge gate should not trust an evidence file just because it says `pass`. `maslow evidence check` compares it with what the gate knows independently and answers `pass`, `needs_review` or `decline` (ADR 0033):

```bash
maslow evidence check reports/verify.json \
  --head "$(git rev-parse HEAD)" \
  --spec trusted/maslow.yaml \
  --profile full \
  --overlay trusted/acceptance.yaml   # or --overlay-sha256 <hex>; repeatable
```

It declines when `evidence_version` is unknown, `git_sha` is not `--head`, `spec_sha256` is not the hash of `--spec`, the run's `overlays` are not exactly the trusted ones, `profile` differs from `--profile` (when given), the verdict is `fail`, `inconclusive` or anything else unknown, or the file cannot be read. With every input matching, a `needs_review` verdict gives `needs_review` and a `pass` verdict gives `pass`. Every reason is printed; `--json` prints the decision and reasons as `{"evidence", "decision", "reasons": [{"field", "message"}]}`. It exits 0 on `pass`, 1 on `needs_review` or `decline`, and 2 on a usage error such as a missing flag or an unreadable trusted file.

To require that the run proved its acceptance contracts red first, add `--require-red-green --base "$(git merge-base origin/main HEAD)"`. It then declines unless every overlay contract ran on exactly that base SHA with status `fail` and passed on the head (a `flaky` head is not green), so evidence from a plain `--overlay` run without `--red-green` is declined with a reason per contract. `--require-red-green` needs `--base` (a full SHA) and `--overlay` or `--overlay-sha256`, or it exits 2. `--base` on its own only checks that `base_sha` matches (ADR 0037).

## In CI

`action/` holds a GitHub Action that installs a pinned release, runs `maslow verify` against the pull request's base with the base branch's spec, uploads the evidence and summarises it. See [action/README.md](action/README.md).

## Writing a Spec

Grow the spec alongside the code rather than writing it all up front:

| When you... | Add to maslow.yaml |
|-------------|-------------------|
| Create an API endpoint | An HTTP contract scenario for it |
| Build a CLI command | A CLI contract scenario for it |
| Produce a build artifact | An `artifact_size` budget for it |
| Implement a performance-sensitive path | A performance budget for it |
| Add a dependency or config file | A ref for it |
| Create a file that a change must not touch | A `policy.deny` or `policy.protected` entry |

Contract templates read captured values as `{{.cap.NAME}}` and environment variables as `{{.env.NAME}}`. A contract can read only the variables listed in its `env`, and their values are redacted from the evidence:

```yaml
contracts:
  - name: api
    env: [API_BASE_URL, API_TOKEN]
    scenarios:
      - name: health
        steps:
          - action: http
            method: GET
            url: "{{.env.API_BASE_URL}}/healthz"
            headers: { Authorization: "Bearer {{.env.API_TOKEN}}" }
            expect: { status: 200 }
```

Each scenario keeps its own cookie jar across its setup, steps and teardown, so a scenario that signs in stays signed in without hand-built `Cookie` headers.

### Multipart uploads

An `http` step can send a `multipart/form-data` body with `multipart` instead of `body`. Each part has a `field` and exactly one of `value` (a text field), `file` (a fixture's bytes) or `content` (inline text sent as a file, which needs a `filename`):

```yaml
- action: http
  method: POST
  url: /v1/videos
  headers: { Authorization: "Bearer {{.cap.access_token}}" }
  multipart:
    - { field: title, value: "Test Video" }
    - { field: video, file: testdata/sample.mp4, content_type: video/mp4 }
    - { field: notes, content: "made by {{.cap.user}}", filename: notes.txt }
  expect: { status: 201, json_path: "$.id" }
```

- `file` is a slash-separated path relative to the directory Maslow runs in, with no `.`, `..` or empty segment. Symlinks may not lead outside that directory. Fixtures are always read from the head checkout, even in a red-green base run or with `--spec-from base`, so list fixture directories in `policy.protected` if a change must not edit them.
- A file part's `filename` defaults to the file's base name, and `content_type` to `application/octet-stream`. Maslow does not guess types from extensions.
- `value`, `content` and `filename` are templated. `field`, `file` and `content_type` are not.
- Maslow sets `Content-Type` with a boundary derived from the parts, so the same parts send the same bytes. A step that also sets a `Content-Type` header fails.
- A missing or escaping fixture, or a body over 10 MiB, gives the scenario status `error`, which is not retried (ADR 0041).

### Browser steps

A `browser` step drives a headless Chrome, one operation per step. It needs Chrome or Chromium installed: Maslow looks for the usual names and install paths, or uses the binary named by `MASLOW_BROWSER`. Nothing else (Node, Playwright, a driver) is required.

```yaml
contracts:
  - name: web
    base_url: "http://localhost:3000"   # an endpoint audit target replaces it
    scenarios:
      - name: sign-in
        steps:
          - { action: browser, do: navigate, url: /login }
          - { action: browser, do: fill, selector: "#email", text: "ada@example.com" }
          - { action: browser, do: click, selector: "button[type=submit]" }
          - { action: browser, do: wait, selector: "#welcome", timeout: 5s }
          - action: browser
            do: assert
            selector: "#welcome"
            expect: { text_contains: "Welcome", visible: true }
          - { action: browser, do: assert, expect: { url_contains: /home } }
```

`click`, `fill` and `assert` never wait: an element that appears later needs a `wait` step first. Browser and HTTP steps share the scenario's cookies. A failing browser step records the page URL and console errors on `failed_step`, and a screenshot at `reports/browser/<contract>/<scenario>.png` in the result's `artifacts` (ADR 0035).

### Metric, log and alerts steps

`metric`, `log` and `alerts` steps judge what a running system reports, so a post-deploy audit can check live behaviour without generating load. Each names a `provider`:

| Action | Query | Providers |
|--------|-------|-----------|
| `metric` | PromQL returning a vector or scalar | `prometheus` (any Prometheus-compatible query API), `datadog` (not implemented yet) |
| `log` | Metric LogQL, such as `sum(count_over_time(...))` | `loki`, `datadog` (not implemented yet) |
| `alerts` | Alertmanager label `matchers` | `alertmanager`, `datadog` (not implemented yet) |

```yaml
contracts:
  - name: checkout-after-deploy
    tags: [post-deploy]
    env: [PROM_URL, PROM_TOKEN, LOKI_URL, ALERTMANAGER_URL]
    scenarios:
      - name: error-rate-holds
        steps:
          - action: metric
            provider: prometheus
            url: "{{.env.PROM_URL}}"
            headers: { Authorization: "Bearer {{.env.PROM_TOKEN}}" }
            query: sum(rate(http_requests_total{code=~"5.."}[1m])) / sum(rate(http_requests_total[1m]))
            for: 10m          # optional; omitted queries once
            interval: 30s     # required with for
            expect: { max: 0.01 }
          - action: log
            provider: loki
            url: "{{.env.LOKI_URL}}"
            query: sum(count_over_time({app="checkout"} |= "panic" [5m]))
            expect: { max: 0, if_no_data: pass }
          - action: alerts
            provider: alertmanager
            url: "{{.env.ALERTMANAGER_URL}}"
            matchers: ['service="checkout"']
            ignore_preexisting: true
            for: 10m
            settle: 5m        # keep polling while slow alert rules catch up
            interval: 1m
            expect: { max: 0 }
```

- `url` is the provider's base URL and must be absolute. It is never joined to `base_url`, so an endpoint audit target does not redirect it.
- `metric` and `log` take `min`, `max`, `min_series`, `max_series` and `if_no_data`. Every series must be within `min` and `max`, inclusive. An empty result, or only NaN, is no data and fails unless `if_no_data: pass`. A `count_over_time` with no matching lines returns nothing, so a "no panics" step needs `if_no_data: pass`.
- `alerts` counts active alerts that are not silenced or inhibited, bounded by `min` and `max`.
- With `for`, the step queries every `interval` until `for` (plus `settle`) has passed and fails at the first breach. At most 1000 polls are allowed. "Over the last 10 minutes" is a single query with `max_over_time(...[10m])`.
- A breach is never retried. A failed query is, and fails the step at once mid-hold. `provider: datadog` validates but gives status `error`.
- Each step adds an entry to the result's `observations`: the query, its exact evaluation times, and the values judged, so any query can be replayed (ADR 0042).

## Development

```bash
make build      # Build the binary
make test       # Run all tests
make vet        # Run go vet
make lint       # Run go vet + cue vet
make validate   # Validate maslow.yaml
make verify     # Run full verification
make release    # Build release archives and checksums into dist/
make clean      # Remove build artifacts
```
