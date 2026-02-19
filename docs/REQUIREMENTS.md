# Maslow — Product Requirements Document (Ultimate Vision)

## 1. Overview

**Maslow** is an executable specification system for agent-built software.

It allows developers to define:

- Behavioral contracts
- Structural constraints
- Toolchain requirements
- Performance budgets
- Audit rules

…in a single declarative spec (`maslow.yaml`) and then enforce those requirements through a deterministic CLI tool.

Maslow’s purpose is to make software development:

- Verifiable
- Agent-compatible
- Black-box auditable
- Multi-agent safe
- Self-hosting

Maslow is designed to be used by humans and AI agents equally.

---

## 2. Product Goals

Maslow must:

1. Turn a repository into an executable contract.
2. Enable parallel agent development without chaos.
3. Provide deterministic verification outputs.
4. Support black-box auditing of artifacts or running systems.
5. Scale from small CLI tools to large monorepos and games.
6. Be self-hostable (Maslow builds Maslow).
7. Remain language-agnostic.
8. Integrate with existing tooling rather than replace it.

Maslow is not:

- A build system
- A package manager
- A linter replacement
- A CI provider

Maslow orchestrates and verifies those things.

---

## 3. Core Concepts

### 3.1 MAS (Maslow Agent Spec)

A `maslow.yaml` file defines:

- Toolchain requirements
- External references
- Policy constraints
- Contracts (scenarios)
- Budgets (performance, size, complexity, etc.)
- Check runners
- Audit configuration

MAS is:

- Declarative
- Deterministic
- Versioned
- Validated via CUE

---

### 3.2 The Kernel

Maslow must always support:

```
maslow validate <file>
```

This validates a MAS file against the CUE schema.

This is the smallest self-hosting primitive.

---

### 3.3 Verification

```
maslow verify --profile quick|full
```

Verify must:

- Load maslow.yaml
- Select profile
- Execute `checks.runner` in order
- Collect results
- Emit `reports/verify.json`
- Exit non-zero on failure

Verification is deterministic and reproducible.

---

### 3.4 Audit

```
maslow audit --profile full
```

Audit must:

- Run in black-box mode
- Validate artifacts or running endpoints
- Avoid requiring source checkout (optional)
- Produce evidence identical in format to verify

Audit enables third-party enforcement.

---

## 4. Functional Requirements

### 4.1 CLI Commands

Maslow must provide:

- `validate`
- `verify`
- `audit`
- `init`
- `version`

Future:

- `explain`
- `plan`
- `profile`
- `diff`

---

### 4.2 Toolchain Integration

Maslow must:

- Support toolchain managers:
  - asdf
  - mise
  - nix
  - custom

- Detect required lockfiles
- Optionally scaffold them
- Support `maslow init --apply`

Maslow must not silently mutate version files.

---

### 4.3 Profiles

MAS supports named profiles:

- quick
- full
- custom

Profiles select subsets of checks.

Profiles must reference defined check kinds only.

---

### 4.4 Contracts

MAS supports scenario-based contracts.

Scenarios must support:

- HTTP calls
- CLI calls
- Polling
- Repetition
- Assertions
- Captures
- JSON path expectations
- Header expectations
- Deterministic replay

Contracts must:

- Be machine-executable
- Be order-sensitive
- Support fixtures

---

### 4.5 Budgets

MAS must support:

- Performance budgets
- Availability budgets
- Rate limits
- Artifact size limits
- Complexity assertions

Performance budgets must support:

- p50 / p90 / p95 / p99
- max latency
- error rate thresholds
- concurrency
- duration
- warmup

---

### 4.6 Policies

MAS must support:

- Deny-list paths
- Allow-list paths
- Protected files
- Gated shared directories

Policy violations must fail verification.

---

### 4.7 Evidence

Maslow must emit:

```
reports/verify.json
```

It must contain:

- Timestamp
- Git SHA (if available)
- Profile used
- Check results
- Contract results
- Budget results
- Verdict (pass/fail/inconclusive)

Evidence must be:

- Stable
- Deterministic
- Diffable
- Machine-readable

---

### 4.8 Self-Hosting

Maslow must:

- Validate its own MAS
- Verify itself via MAS
- Audit its own binary

Self-hosting must not require special casing.

---

## 5. Multi-Agent Support

Maslow must support safe concurrent development via:

- Folder scoping
- Policy enforcement
- Explicit contracts
- Deterministic verification

Maslow does not coordinate agents directly.

Maslow enforces invariants that make coordination safe.

---

## 6. Monorepo Support

Maslow must:

- Support monorepo package declarations
- Allow per-package checks
- Allow per-package budgets
- Allow path-scoped refs

Maslow must not require a specific repo layout.

---

## 7. Extensibility

Maslow must support:

- `extensions` section for domain profiles
- Domain overlays (e.g., Game, Web, Mobile)
- Additional ref kinds
- Custom check kinds

Extensions must not break core schema.

---

## 8. Domain Profiles (Future)

Maslow may ship official profiles:

- maslow/web
- maslow/game
- maslow/cli
- maslow/service

Profiles may add:

- Additional schema validation
- Domain-specific checks
- Golden test patterns

Profiles must layer on top of core schema.

---

## 9. Black-Box Audit Requirements

Audit must support:

- Binary targets
- Docker container targets
- HTTP endpoints
- Environment variable injection
- Determinism controls (timeout, retries)

Audit must not require source access.

Audit must produce same evidence format as verify.

---

## 10. Developer Workflow

The canonical workflow:

1. Write maslow.yaml
2. Run `maslow init`
3. Run `maslow validate`
4. Run `maslow verify --profile quick`
5. Commit
6. CI runs `maslow verify --profile full`
7. Optional: third party runs `maslow audit`

---

## 11. Non-Functional Requirements

### 11.1 Determinism

Repeated runs with identical inputs must produce identical verify.json.

### 11.2 Speed

Quick profile must complete in < 5 seconds for small repos.

### 11.3 Clarity

Error messages must:

- Reference file paths
- Reference spec section
- Be human-readable
- Be machine-parsable

### 11.4 Stability

Maslow must not introduce breaking changes to MAS without:

- Schema version bump
- Migration guidance

---

## 12. Versioning Strategy

- `mas` field defines MAS version
- Maslow binary supports N backward versions
- Schema evolution must be monotonic

Breaking changes require major MAS version bump.

---

## 13. Long-Term Vision

Maslow becomes:

- The contract layer between humans and agents.
- The verification layer between builders and auditors.
- The deterministic boundary in agentic systems.
- The kernel of self-improving software ecosystems.

Maslow should eventually allow:

- Agent-planned tasks validated against MAS
- Automatic regression prevention
- Spec-first development at scale
- Multi-agent collaboration without chaos

---

## 14. Out of Scope

Maslow will not:

- Replace CI systems
- Replace version control
- Manage dependencies
- Replace test frameworks
- Judge narrative quality or subjective aesthetics

---

## 15. Success Criteria

Maslow is successful when:

- A repository without tests can become testable via MAS.
- Multiple agents can work in parallel without corrupting invariants.
- A third party can audit a system without reading source.
- Maslow builds Maslow.

## 16. Self-Hosting Requirement (Bootstrap Constraint)

### 16.1 Maslow v1 Must Be Self-Built

Maslow v1 must be:

1. **Bootstrapped minimally**
   - A small kernel (validate + basic verify) may be written manually.
   - This bootstrap code must be minimal and documented.

2. **Then rebuilt using Maslow itself**
   - The Maslow repository must contain a valid `maslow.yaml`.
   - That spec must fully describe the Maslow project.
   - Maslow must be able to:
     - Validate its own MAS
     - Verify its own repository
     - Emit `reports/verify.json` for itself
     - Audit its own built binary

Maslow v1 is complete only when:

```
maslow verify --profile full
```

passes **on the Maslow repository itself**, using Maslow.

---

### 16.2 No Permanent “Bootstrap Escape Hatches”

The bootstrap kernel must not become a hidden backdoor.

Specifically:

- All enforcement logic must eventually be described in MAS.
- Manual or hardcoded checks must be migrated into:
  - `checks.runner`
  - `contract.scenarios`
  - `budgets`
  - `policy`

There must be no behavior in Maslow v1 that:

- Cannot be described in MAS
- Cannot be verified by MAS
- Cannot be audited in black-box mode

---

### 16.3 The Kernel Reduction Principle

The bootstrap kernel must:

- Be as small as possible
- Contain no domain logic beyond:
  - CUE validation
  - Check execution
  - Evidence emission

All higher-level features (init scaffolding, advanced audit modes, profiles, etc.) must be layered on top and validated via MAS.

---

### 16.4 Reproducible Binary Requirement

Maslow v1 must be able to:

1. Build its own binary
2. Audit that binary
3. Confirm:
   - Version metadata matches Git SHA
   - Contracts pass
   - Budgets pass
   - Evidence is emitted

Self-audit must not require source checkout when in black-box mode.

---

### 16.5 Self-Improvement Constraint

Future Maslow versions must be built and verified by the previous stable version.

That is:

- Maslow vN must be able to verify Maslow vN+1.
- Breaking schema changes require explicit version migration.

Maslow must not require a rewrite outside the MAS system to evolve.

---

### 16.6 Completion Definition for v1

Maslow v1 is considered complete when:

- The bootstrap code is minimal.
- The MAS file in the Maslow repo fully defines:
  - Toolchain
  - Checks
  - Contracts
  - Budgets
  - Audit behavior

- `maslow verify --profile full` passes on itself.
- `maslow audit --profile full` passes on the built binary.
- The bootstrap kernel could theoretically be deleted and reconstructed from the MAS.

---

This requirement makes Maslow:

- A specification-first system
- A self-hosting verification engine
- A contract-driven build tool
- A stable kernel for agentic systems

And philosophically, it means:

Maslow does not just enforce contracts.

Maslow lives under its own.
