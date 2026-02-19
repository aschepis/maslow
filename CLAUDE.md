# Maslow Project - Agent Guide

## Project Overview

Maslow is an executable specification system for agent-built software. It takes a declarative `maslow.yaml` spec, validates it against a CUE schema, runs verification tasks, and produces structured reports.

Stack: Go CLI, CUE schema, YAML specs.

## Key Paths

| Path | Purpose |
|------|---------|
| `cmd/maslow/main.go` | CLI entrypoint |
| `schema/maslow.cue` | CUE schema (authoritative spec definition) |
| `internal/schema/` | CUE API wrapper |
| `internal/spec/` | Spec loading and parsing |
| `internal/verify/` | Verification logic |
| `internal/audit/` | Audit trail |
| `internal/evidence/` | Evidence collection |
| `internal/runner/` | Task execution |
| `testdata/valid/` | Fixtures that must pass validation |
| `testdata/invalid/` | Fixtures that must fail validation |
| `reports/` | Generated output |
| `docs/` | Architecture docs, ADRs, plans |

## Conventions

- Go for all CLI and library code
- CUE for schema validation (`schema/maslow.cue`)
- YAML for the spec format (`maslow.yaml`)
- Deterministic output required: the same input must always produce the same result
- All error messages must reference file paths and spec section names
- Exit codes: `0` = success, `1` = validation or verification failure, `2` = usage error

## Working Rules

- Read `docs/MAP.md` for architecture overview before making structural changes
- Read `docs/PLAN.md` for current milestones and priorities
- ADRs in `docs/adr/` document the reasons behind key decisions; read the relevant ADR before changing a decision
- Test fixtures in `testdata/` are part of the spec contract; do not modify them without a corresponding schema or logic change
- Keep changes narrow and focused; prefer small, reviewable diffs
- Run tests frequently: `go test ./...`
- Run CUE validation after schema changes: `cue vet schema/maslow.cue`

## Adding a New Feature

1. Check `docs/PLAN.md` for the relevant milestone
2. Update `schema/maslow.cue` if the spec format changes
3. Add or update fixtures in `testdata/valid/` and `testdata/invalid/`
4. Implement in the appropriate `internal/` package
5. Wire up in `cmd/maslow/main.go` if a new subcommand or flag is needed
6. Run `go test ./...` and `cue vet schema/maslow.cue` before marking done

## Notes

No emojis. Keep it concise.
