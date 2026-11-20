# ADR 0041: Multipart HTTP Steps

- **Status**: Accepted
- **Date**: 2026-11-19

## Context

An `http` step could send only a string body, so upload endpoints could not be contract-tested (task 20). Sending a file means reading bytes from disk into a request, which raises questions about where files come from, how failures are classified and whether the request is deterministic.

## Decision

An `http` step may set `multipart`, an explicit ordered list of parts that `internal/runner/multipart.go` encodes and sends instead of `body`, reading `file` parts through `os.Root` from the directory Maslow runs in.

## Details

- Each part has `field` and exactly one of `value`, `file` or `content`. `content` needs `filename`. The schema enforces this with `matchN`/`matchIf`, along with `body` versus `multipart` and the `file` path rule (no leading `/`, backslash, `.`, `..` or empty segment). The runner checks the same rules, for runs that bypass the schema.
- Fixtures are read from the working directory, never from the red-green base worktree. A fixture added in the same change would be missing on the base, and the base run would count as red for the wrong reason. Fixtures are test input, like the overlay, not code under test.
- The boundary is `maslow-` plus the first 32 hex characters of a SHA-256 over the parts, so the same parts always produce the same bytes. `content_type` defaults to `application/octet-stream`; types are not guessed from extensions, because `mime.TypeByExtension` reads host tables.
- `value`, `content` and `filename` are templated; `field`, `file` and `content_type` are not, so fixture paths stay reviewable and do not depend on the environment.
- A missing, escaping or oversized fixture, or a body over 10 MiB, is a `fixtureError`. The scenario's status is `error` and it is not retried. A `Content-Type` header on the step, or `body` alongside `multipart`, fails before sending.
- The body is held in memory as a `bytes.Reader`, so a 307 or 308 replays it. Evidence does not change: `failed_step.target` stays `"METHOD url"` and part values are never recorded.

## Consequences

- Under `--spec-from base` (ADR 0019), fixtures still come from the head, so a change can weaken a test by replacing its fixture. List fixture directories in `policy.protected`. Task 46 tracks reading spec contracts' fixtures at the base commit.
- `failedResult` now has two failure statuses. Red-green already treats a base `error` as not red.
- The 10 MiB cap is a constant. Specs that need larger uploads would need it made configurable.
