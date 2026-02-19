# ADR 0005: Audit Target Abstraction - Binary, Container, Endpoint

- **Status**: Accepted
- **Date**: 2026-02-19

## Context

Maslow audit must work in black-box mode without access to source code, per spec section 9. A project that ships a binary, a container image, or an HTTP service should be auditable by Maslow using only the artifact itself. This requires an abstraction that supports multiple target kinds while producing evidence in the shared format defined in ADR 0002.

Requirements:

- Support binary artifacts as the primary v1 target
- Provide a defined interface for container and endpoint targets so they can be implemented fully in v1.1 without breaking changes to `maslow.yaml`
- Use the same evidence format as `verify` so downstream consumers handle one schema
- Allow per-target configuration: environment variables, timeout, retries, and named check behaviors

Options considered:

| Concern | Options Considered |
|---------|--------------------|
| Target kinds | Binary only, binary + container + endpoint stubs, all three fully implemented |
| Configuration location | Separate audit config file, `maslow.yaml` top-level `audit` key, inline per-command flags |
| Unimplemented targets | Omit from schema, stub returning skip, stub returning error |

## Decision

**Target definition schema**

Audit targets are defined in `maslow.yaml` under `audit.targets` as a list of objects with the following fields:

- `kind`: one of `binary`, `container`, or `endpoint`
- `path`: path to the binary executable, container image reference, or base URL
- `env`: map of environment variable names to values passed to the target at execution time
- `timeout`: duration string for the entire audit of this target; defaults to `120s`
- `retries`: number of retry attempts on transient failure; defaults to `0`
- `checks`: list of named check behavior strings (e.g., `version-check`, `help-flag`)

**Target kinds**

- `binary`: the artifact at `path` is executed directly on the host. Named check behaviors map to specific invocation patterns (e.g., `version-check` runs `<path> --version` and validates exit code 0). Results are recorded as `CheckResult` entries in evidence.
- `container`: the artifact at `path` is an image reference passed to `docker run`. The same check behavior mapping applies. In v1 this kind returns `skip` status for all checks with a note that container audit is not yet implemented.
- `endpoint`: the artifact at `path` is an HTTP base URL. Named check behaviors map to HTTP requests and response validations. In v1 this kind returns `skip` status for all checks with a note that endpoint audit is not yet implemented.

**Evidence output**

Audit writes to `reports/verify.json` using the same structure as `verify`. The `profile` field is set to the target `kind` and `name` concatenated (e.g., `binary:myapp`). This allows evidence files from both commands to be compared and stored in the same directory without conflict when targets have distinct names.

**v1 scope**

Binary audit is fully implemented in v1. Container and endpoint kinds are registered in the schema (CUE validates them without error) and in the runner dispatch table, but their handlers return `skip` status unconditionally. This is an honest stub: the evidence file accurately reflects that no checks were executed for those kinds.

## Consequences

**Positive**

- The `audit.targets` schema is stable in v1; adding full container and endpoint implementations in v1.1 requires no `maslow.yaml` changes for existing users
- Binary audit covers the most common case: a compiled Go binary can be audited immediately after the v1 release
- Stubs return `skip` rather than a fake `pass`, so evidence files accurately represent what was and was not tested
- The same evidence format as `verify` means the same tooling reads both outputs

**Negative**

- Container and endpoint targets are not operational in v1; a `maslow.yaml` that defines only container targets will produce an evidence file with all `skip` results and an `inconclusive` verdict, which may be surprising
- The `path` field carries different semantics for each kind (filesystem path, image reference, URL); this is a single field serving three purposes, which may cause confusion
- Named check behaviors are a fixed set per kind; there is no way to define custom check behaviors in `maslow.yaml` without a code change in v1

**Mitigations**

- The CLI emits a warning when a non-binary target kind is used in v1, explaining that the target kind is not yet fully implemented and that `skip` results are expected
- The `path` field semantics are documented in the `maslow.yaml` reference for each kind; the CUE schema validates format where possible (e.g., URLs must begin with `http://` or `https://`)
- Custom check behaviors are planned for v1.2 via a `behaviors` extension point in the target schema; the schema is designed to accommodate this addition without breaking changes
