# ADR 0024: A Composite GitHub Action With No Logic of Its Own

- **Status**: Accepted; amended by ADR 0036 (the job summary is rendered by `maslow evidence summary`)
- **Date**: 2026-11-18

## Context

The intended consumer is a pull-request pipeline. Each adopter would otherwise write the same steps: download a pinned binary, check its checksum, pass the base SHA, keep the report and summarise it.

## Decision

Ship a composite Action in `action/` that installs a pinned, checksum-verified release (or uses a given binary), runs `maslow verify` with its inputs, uploads the evidence, writes a job summary, exposes the verdict as an output and exits with Maslow's exit code.

## Consequences

- All verification behaviour stays in the binary, so a laptop, an agent session and CI get the same verdict for the same inputs.
- Maslow's own CI runs the Action with `binary: bin/maslow`, `--base` set to the pull request's base and `spec-from: base`, so every pull request here is gated the way an adopter's would be.
- The Action's inputs and the evidence format are now public surface; changing either needs a deprecation path.
