# ADR 0001: Use CUE for Schema Validation; YAML for Spec Format

- **Status**: Accepted
- **Date**: 2026-02-19

## Context

Maslow requires two distinct things: a declarative spec format that humans and agents write (`maslow.yaml`), and a schema that validates that format. These are separate concerns with different requirements.

The spec format must be:
- Human-readable and writable without tooling
- Parseable by agents without special libraries
- Supported by standard editors, linters, and CI tooling

The schema and validation layer must be:
- Capable of enforcing structural constraints and type safety
- Able to express cross-reference validity (e.g., a spec section referencing a named agent)
- Deterministic: the same input always produces the same validation result
- Integrated cleanly with the Go CLI

Several options were considered for each layer:

| Layer | Options Considered |
|-------|--------------------|
| Spec format | YAML, TOML, JSON, HCL |
| Schema/validation | JSON Schema, CUE, Open Policy Agent, hand-written Go |

## Decision

**Spec format: YAML (`maslow.yaml`)**

YAML is the standard format for declarative configuration in the Go and cloud-native ecosystem. It is human-readable, widely understood by agents and developers, and has mature Go library support (`gopkg.in/yaml.v3`). The moderate complexity of YAML (anchors, multi-document, etc.) is not needed here; Maslow specs use a simple, flat-ish structure that avoids YAML's edge cases.

**Schema and validation: CUE**

CUE provides structural typing with constraints that go beyond what JSON Schema can express. Key advantages:

- Constraints like `len(name) > 0`, disjunctions, and field-level conditions are first-class
- The `cuelang.org/go` package provides a Go-native API for loading and evaluating CUE values programmatically
- Validation is deterministic: CUE evaluation is pure and order-independent
- A single `schema/maslow.cue` file serves as the authoritative definition of a valid spec
- CUE can also generate JSON Schema or OpenAPI if interoperability is needed later

## Consequences

**Positive**

- Invalid specs are caught at validation time with structured, field-level error messages
- The CUE schema is readable and serves as living documentation of the spec format
- Go integration via `cuelang.org/go` is first-class and well-maintained
- Deterministic validation means tests are reliable and validation can run in CI without side effects

**Negative**

- CUE is less widely known than JSON Schema; contributors may need to learn it
- CUE adds a Go module dependency and increases build complexity slightly
- The CUE Go API (`cue.Context`, `cue.Value`) has a learning curve and verbose error handling

**Mitigations**

- `schema/maslow.cue` is the single source of truth; it is the first thing a contributor should read to understand the spec format
- `internal/schema/` wraps all CUE API calls, keeping CUE-specific complexity isolated from the rest of the codebase
- Validation errors are translated into user-facing messages in `internal/verify/`, so callers never deal with raw CUE errors
