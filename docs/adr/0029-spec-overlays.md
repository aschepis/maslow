# ADR 0029: Spec Overlays Add Contracts for One Run

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

A pipeline that hands a change to an agent often has acceptance scenarios for that change alone. Putting them in `maslow.yaml` makes them permanent and, with `--spec-from base`, they would not run until merged. The pipeline needs to attach scenarios to one run without letting the change edit them or weaken the spec.

## Decision

`maslow verify --overlay <file>` (repeatable) loads a file that may hold only `description` and `contracts`, runs its contracts on every profile beside those the profile selects, and refuses a contract name already used by the spec or another overlay.

## Consequences

- An overlay can add scenarios but cannot remove, replace or relax anything in the spec: other keys fail schema validation and shadowing a name is an error.
- The evidence lists each overlay's path and SHA-256 under `overlays`, and marks each result it produced with `overlay`, so a reader can tell which scenarios came from where.
- Maslow reads the overlay from the path given. Keeping it immutable is the caller's job: store it outside the workspace, or in the repository under a `protected` path so A1's base-ref policy fails a change that edits it.
- Overlay contracts ignore profile selection; a profile cannot opt out of a change's acceptance scenarios.
