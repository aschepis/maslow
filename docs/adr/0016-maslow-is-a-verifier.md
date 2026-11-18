# ADR 0016: Maslow Is a Verifier

- **Status**: Accepted
- **Date**: 2026-11-18
- **Supersedes**: ADR 0008, ADR 0015

## Context

Up to v0.5.0 Maslow shipped two products: a verifier and an agent harness (`maslow scaffold` and `maslow harness`, which generated CLAUDE.md, MAP/PLAN templates and a task queue). Agent scaffolding goes stale as models improve, every orchestrator ships its own, and it was about a third of the code. A neutral spec and evidence format is the part other tools lack.

## Decision

Remove `maslow scaffold`, `maslow harness` and their packages; Maslow only verifies a repository against `maslow.yaml` and writes evidence.

## Consequences

- Adoption is one file: `maslow init` writes a starter spec into an existing project, with no workflow imposed. The v0.5.0 tag preserves the harness code for anyone who needs it.
- This repository's own `CLAUDE.md` and `docs/tasks/` remain as its working process, but the Harness Propagation Rule is gone; ADRs 0006, 0007 and 0011 now describe this repository only.
- The agent skill in `.skills/maslow/` is trimmed to running `verify` and reading evidence, and carries the progressive-verification advice about writing specs.
