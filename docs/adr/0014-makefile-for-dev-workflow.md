# ADR 0014: Makefile for Development Workflow

## Context

The project had multiple CLI commands for building, testing, linting, and verifying, but no standard entry point for developers or agents. Each operation required remembering the exact command (e.g., `go build -o bin/maslow ./cmd/maslow/`, `cue vet schema/maslow.cue`). A consistent interface was needed.

## Decision

Use a Makefile as the canonical entry point for all development operations: `make build`, `make test`, `make vet`, `make lint`, `make validate`, `make verify`, `make clean`.

## Consequences

- Standard Make targets map 1:1 to the common operations described in CLAUDE.md and README.md
- Agents and humans can use `make verify` instead of remembering the full `go build && bin/maslow verify --profile full` sequence
- `lint` depends on `vet` so running lint also runs go vet, matching the convention in CLAUDE.md
