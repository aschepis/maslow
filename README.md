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
maslow version                   # Print version info
```

## Verification Pipeline

`maslow verify` runs the following in order:

1. **Checks** — shell commands (build, test, lint) filtered by profile
2. **Contracts** — scenario-based behavioural tests (CLI and HTTP); a profile's `contracts` list selects by name or tag
3. **Budgets** — artifact size, performance, and complexity limits; selected by a profile's `budgets` list
4. **Refs** — verify declared files, binaries, and URLs exist
5. **Policy** — `deny`, `protected`, `allow` and `gated` rules, judged on the change since `--base <ref>` when given

The verdict is `fail`, `needs_review`, `inconclusive` or `pass`, in that order of precedence. `needs_review` means everything passed but the change touched a `gated` path, and names its owner. It exits 0 unless `--strict` is passed.

Results are written to `reports/verify.json` (or `--report <path>`) as structured evidence. `schema/evidence.schema.json` describes the format. `maslow audit` writes `reports/audit.json`.

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
