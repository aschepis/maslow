# ADR-0012: Ref Verification Runner

## Context

Refs in maslow.yaml declare external dependencies (files, binaries, URLs, MCPs) but were never verified at runtime. A required ref pointing to a missing file silently passed verification, making maslow's safety guarantees decorative.

## Decision

Implement a ref verification runner (`internal/runner/refs.go`) that dispatches by ref kind: `file` and `binary` use `os.Stat`, `url` uses `http.Head` with a 10s timeout. Unimplemented kinds (`package`, `container`, `endpoint`, `mcp`) return `skip`. Non-required missing refs return `skip` instead of `fail`.

## Consequences

- Refs are now verified as part of `maslow verify`, catching missing files/binaries/URLs
- Required refs that are missing cause the verdict to fail
- Future ref kinds (package, container, mcp) can be incrementally implemented by adding dispatch cases
- URL verification adds network dependency to verify runs; the 10s timeout prevents hangs
