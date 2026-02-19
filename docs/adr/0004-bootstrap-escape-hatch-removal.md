# ADR 0004: Bootstrap Kernel Scope and Escape Hatch Removal

- **Status**: Accepted
- **Date**: 2026-02-19

## Context

Maslow must be self-hosting per the spec (section 16). The bootstrap kernel is the minimal set of code that must be written before Maslow can verify itself. Without this bootstrapping phase, no MAS-described toolchain can be verified because the verification engine does not yet exist.

Per spec section 16.2, there must be no permanent escape hatches: mechanisms that allow a check to be bypassed, a validation to be skipped, or a verdict to be overridden by convention rather than configuration. Permanent escape hatches undermine the integrity guarantee that Maslow is built to provide.

The bootstrap kernel must therefore:

- Be as small as possible so its correctness can be audited by reading the code
- Contain no domain logic beyond what is strictly necessary to execute verification
- Drive all enforcement from `maslow.yaml` rather than hardcoded rules
- Leave no bypass mechanisms in place after M5

Options considered for scope:

| Concern | Options Considered |
|---------|--------------------|
| Kernel boundary | Minimal (3 primitives), moderate (includes toolchain detection), maximal (includes full audit) |
| Escape hatch policy | Remove all, allow with expiry flag, allow with documented justification |
| Self-hosting timeline | M3, M5, post-v1 |

## Decision

**Bootstrap kernel scope**

The bootstrap kernel contains exactly three primitives:

1. **CUE validation**: load `maslow.yaml`, validate it against `schema/maslow.cue`, and return structured errors
2. **Check execution**: run the checks defined in `maslow.yaml` using the runner abstraction (ADR 0003), collect results
3. **Evidence emission**: write `reports/verify.json` in the format defined in ADR 0002

All higher-level features are layered on top of these primitives and are not part of the kernel:

- Contract execution and evaluation
- Budget enforcement
- Toolchain detection and version resolution
- Container and endpoint audit (ADR 0005)

The kernel has no hardcoded checks. Every enforcement decision is driven by the contents of `maslow.yaml`. The kernel's only job is to faithfully execute what the spec describes.

**Escape hatch removal**

All escape hatches are removed as of M5. An escape hatch is defined as any code path that:

- Skips CUE validation without user configuration
- Executes checks without recording results in evidence
- Emits a `pass` verdict without executing the checks that would justify it
- Allows `maslow.yaml` to be absent and proceeds without error

No flags, environment variables, or build tags introduce such bypass mechanisms after M5.

**Self-hosting completion**

The bootstrap kernel is considered complete at M5. Maslow verifies itself on every commit using its own `maslow.yaml`. The kernel does not need further reduction because it contains no domain logic: it is the execution engine, not the policy.

## Consequences

**Positive**

- The kernel is small, approximately 500 lines across `internal/schema/`, `internal/runner/`, and `internal/evidence/`; it can be audited in a single sitting
- All behavior is MAS-describable: any check that Maslow enforces on other projects can be expressed in `maslow.yaml` and enforced on Maslow itself
- No escape hatches means the self-hosting guarantee is unconditional after M5
- The three-primitive structure maps cleanly to three internal packages with minimal coupling

**Negative**

- The kernel cannot itself be generated from a MAS description; it is the execution engine, which means it sits outside the system it enables. This is an inherent bootstrapping constraint, not a design flaw
- Removing all escape hatches makes development of the kernel itself harder: a bug in CUE validation blocks all other work until it is fixed. There is no fallback mode
- The kernel boundary excludes toolchain detection, which means early milestones require manual environment setup that a broader kernel could automate

**Mitigations**

- The bootstrapping constraint is acknowledged in spec section 16.3, which explicitly permits the execution engine to be outside the MAS envelope
- The no-fallback policy is mitigated by comprehensive unit tests for each primitive before the escape hatches are removed; the removal happens in a single M5 commit so the transition is auditable
- Toolchain detection is prioritized immediately after M5 so the manual setup period is short
