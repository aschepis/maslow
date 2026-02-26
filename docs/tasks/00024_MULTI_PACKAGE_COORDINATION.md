---
id: 24
title: Add multi-package build coordination guidance to harness
status: draft
priority: medium
created: 2026-02-22
updated: 2026-02-22
assigned_to: ""
assigned_at: ""
depends_on: [21]
tags: [maslow, harness, scaffold, monorepo]
---

## Objective

The maslow.yaml `packages` section declares monorepo packages with name and path, but the harness gives no guidance on how an agent should coordinate building multiple packages. For MaslowTok (api, worker, web, mobile, shared), the agent needs to know: build shared first, then API, then web/mobile in parallel. Interface contracts between packages should be defined early.

This is harness guidance, not schema changes. The agent can infer dependency order from the code. What it needs is the convention for how to approach multi-package work.

## Requirements

- Update the CLAUDE.md template with a "Multi-Package Projects" section:
  - Build shared/library packages before services before apps
  - Define interface contracts (API specs, shared types) early — these are the integration surface
  - Scope tasks and commits to a single package where possible
  - Use `policy.gated` to declare package ownership if multiple agents work in parallel
  - Add contracts for cross-package integration points as they emerge
  - When you realize a package needs something from another package that doesn't exist yet, build the interface in the dependency first
- Update the maslow skill with multi-package workflow guidance
- Consider whether `packages` should support a `depends_on` field in the schema (optional, not blocking)

## Acceptance Criteria

- [ ] Generated CLAUDE.md includes Multi-Package Projects section
- [ ] Generated maslow skill includes monorepo workflow guidance
- [ ] Convention covers: build order, interface-first development, task scoping, cross-package contracts
- [ ] Scaffold tests pass

## Notes

- The agent doesn't need rigid build ordering prescribed in the schema. It can figure out "shared has no dependencies, api imports shared, web calls api" from the code. What it needs is the principle: "build leaves first, define interfaces early."
- For MaslowTok, the natural order would be: shared types → API service (defines the HTTP contract) → web + mobile in parallel (both consume the API)
- The `policy.gated` field already exists in the schema for ownership scoping — the harness just needs to teach agents to use it
- This becomes more important if/when maslow supports multi-agent parallel execution across packages
