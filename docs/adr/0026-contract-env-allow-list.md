# ADR 0026: Contracts Read Only Allow-Listed Environment Variables

- **Status**: Accepted
- **Date**: 2026-11-18
- **Amends**: ADR 0009

## Context

`{{.env.NAME}}` could read every variable in Maslow's environment, and contract output and errors went into the evidence verbatim. In CI that environment holds deploy keys and tokens, so a spec, or a change to one, could copy a secret into an evidence file that is uploaded as a build artifact.

## Decision

A contract lists the variables its templates may read in `env`; any other `{{.env.NAME}}` fails the step, and every listed value of four or more bytes is replaced with `[redacted:NAME]` in that contract's error, output and failed-step target.

## Consequences

- **Breaking change**: a spec that used `{{.env.NAME}}` without listing `NAME` now fails the scenario with an error pointing at the `env` list. The bundled examples were updated.
- Every listed value is redacted, including harmless ones such as a base URL. Shorter values are kept because redacting `1` or `dev` would mangle the evidence and they are not credentials.
- Redaction matches exact values only. A secret that a system under test transforms (encodes, truncates) before printing is not caught.
