# Maslow Execution Plan

## Guiding Principle

Build the smallest kernel that can validate itself first. Expand depth-first. Every new capability
must be describable in MAS, verifiable by MAS, and auditable in black-box mode before Maslow v1 is
considered complete.

---

## Milestones

### M1 - Kernel

Goal: `maslow validate` works. Schema and spec parsing are correct. This is the smallest
self-hosting primitive.

Deliverables:
- `schema/maslow.cue` - minimal CUE schema covering required MAS fields
- `cmd/maslow/main.go` - CLI entrypoint with `validate` subcommand
- `internal/spec/` - parses `maslow.yaml` into typed Go struct
- `internal/schema/` - loads `schema/maslow.cue` and validates spec
- `testdata/valid/` - at least one passing fixture
- `testdata/invalid/` - at least one failing fixture per violation class
- `go test ./...` passes

Exit criteria: `maslow validate maslow.yaml` exits 0 on valid input, non-zero with actionable error
message on invalid input.

---

### M2 - Verify

Goal: `maslow verify --profile quick|full` runs checks, collects results, and emits
`reports/verify.json`.

Deliverables:
- `internal/verify/` - profile selection; check sequencing; result collection
- `internal/runner/` - check runner abstraction (shell, CLI, HTTP)
- `internal/evidence/` - deterministic `reports/verify.json` emission
- Profiles `quick` and `full` wired end-to-end
- Exit non-zero on any check failure
- `go test ./...` covers runner, evidence serialization

Exit criteria: `maslow verify --profile quick` produces a valid, deterministic `reports/verify.json`
on a fixture repository.

---

### M3 - Audit

Goal: `maslow audit --profile full` operates in black-box mode without source checkout.

Deliverables:
- `internal/audit/` - audit target abstraction (binary, Docker container, HTTP endpoint)
- Environment variable injection for audit targets
- Timeout and retry controls
- Audit produces evidence in identical format to verify
- `go test ./...` covers audit target execution stubs

Exit criteria: `maslow audit` runs against a pre-built binary target and emits `reports/verify.json`
identical in structure to the verify output.

---

### M4 - Init

Goal: `maslow init` scaffolds a valid `maslow.yaml` and detects existing toolchain.

Deliverables:
- `maslow init` generates `maslow.yaml` with detected toolchain manager (asdf, mise, nix, custom)
- `maslow init --apply` writes lockfiles without prompting
- Generated file is immediately valid against `schema/maslow.cue`
- No silent mutation of existing version files

Exit criteria: `maslow init` on an empty directory produces a file that passes `maslow validate`.

---

### M5 - Self-Hosting

Goal: Maslow builds, verifies, and audits itself. v1 is complete.

Deliverables:
- `maslow.yaml` in the Maslow repository fully describes itself (toolchain, checks, contracts,
  budgets, audit targets)
- `maslow verify --profile full` passes on the Maslow repository using the Maslow binary
- `maslow audit --profile full` passes on the Maslow-built binary
- Bootstrap kernel is minimal; no hardcoded checks remain outside MAS
- ADR documents bootstrap escape hatch removal

Exit criteria: CI runs `maslow verify --profile full` on the Maslow repo and exits 0.

---

## Parallel Workstreams

Workstreams are scoped by directory to prevent agent collisions. Each stream owns its paths
exclusively during active development.

| Stream | Owner Paths | Milestone Scope |
|---|---|---|
| Schema | `schema/maslow.cue`, `internal/schema/` | M1, ongoing |
| CLI | `cmd/maslow/`, `internal/spec/` | M1, M4 |
| Harness | `internal/verify/`, `internal/evidence/`, `internal/runner/` | M2 |
| Audit | `internal/audit/` | M3 |
| Docs | `docs/` | All milestones |

Shared integration point: `reports/verify.json` format. Changes to the evidence schema must be
coordinated across Harness and Audit streams.

---

## Risk Register

### R1 - CUE Go Library API Stability

Risk: The `cuelang.org/go` library has had breaking API changes between minor versions.

Mitigation:
- Pin the CUE library version in `go.mod` at project start.
- Document the pinned version in `docs/adr/`.
- Write a thin adapter in `internal/schema/` so the CUE API surface is isolated; a version upgrade
  touches one package only.

---

### R2 - Contract Execution Complexity

Risk: HTTP and CLI contract scenarios involve polling, captures, JSON path assertions, and
deterministic replay. This is a significant surface area.

Mitigation:
- Implement a minimal contract runner in M2 covering the most common patterns first.
- Define the full contract schema in CUE early (M1) so the contract structure is locked before
  execution logic is built.
- Use table-driven tests against fixture scenarios in `testdata/`.

---

### R3 - Performance Budget Enforcement Needs Benchmarking Framework

Risk: p50/p90/p95/p99 latency budgets require a benchmarking harness, not just a timer.

Mitigation:
- Treat budget enforcement as a distinct runner kind (`budget`) in `internal/runner/`.
- Use Go's `testing/B` or a minimal external HTTP load tool in the first pass.
- Budget enforcement correctness is a hard requirement for M2 exit; accuracy against real latency
  distributions is a M3 concern.

---

### R4 - Self-Hosting Circular Dependency During Bootstrap

Risk: Maslow cannot verify itself before Maslow exists. The bootstrap kernel must be written
manually before the self-hosting constraint can be met.

Mitigation:
- Document the bootstrap kernel as explicitly temporary in `docs/adr/`.
- Keep the bootstrap kernel as small as possible (M1 scope only: validate + minimal check execution
  + evidence emission).
- Define the self-hosting exit criteria precisely (see M5) so there is a clear, binary test for
  when the circular dependency is resolved.
- Maslow vN must be able to verify Maslow vN+1; this constraint governs future upgrades.

---

## ADR Index

ADRs are in `docs/adr/`. Each records a material decision with context, options considered,
decision made, and consequences.

Planned ADRs:
- `ADR-0001` - Why CUE for schema; why YAML for spec
- `ADR-0002` - Evidence format (reports/verify.json structure)
- `ADR-0003` - Check runner abstraction design
- `ADR-0004` - Bootstrap kernel scope and escape hatch removal plan
- `ADR-0005` - Audit target abstraction (binary / container / endpoint)

---

## Definition of Done for v1

- `maslow validate` - validates MAS files against CUE schema
- `maslow verify --profile quick|full` - runs checks, emits deterministic evidence
- `maslow audit --profile full` - black-box mode; no source required
- `maslow init` - scaffolds valid `maslow.yaml`; detects toolchain
- `maslow version` - prints version and build SHA
- The Maslow repository has a `maslow.yaml` that fully describes itself
- `maslow verify --profile full` passes on the Maslow repository
- `maslow audit --profile full` passes on the Maslow binary
- No hardcoded checks remain outside MAS
- All ADRs are written
- `docs/MAP.md` is current
