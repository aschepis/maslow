---
id: 22
title: Establish refs as generative input convention in harness
status: done
priority: high
created: 2026-02-22
updated: 2026-02-26
assigned_to: claude
assigned_at: "2026-02-26"
depends_on: [21]
tags: [maslow, harness, scaffold, refs]
---

## Objective

Refs today are verification-only: "does this file exist?" But in the generative model, refs are the agent's primary input — PRDs, branding guides, tech decision docs, aspirational notes, API specs. A ref pointing to `docs/PRD.md` should tell the agent "read this before you start building; this is your north star."

The harness needs to teach agents to read refs as context, not just check their existence. This doesn't require schema changes — it's a convention change in how the harness instructs agents to use refs.

## Requirements

- Update the CLAUDE.md template to include guidance: "Before starting any generative work, read all refs with `kind: file` that point to documentation. These are your requirements, constraints, and context. Refs that point to code configs (.prettierrc, .eslint) are verification targets. Refs that point to docs/ are input."
- Update the maslow skill's workflows to include "Read all doc refs" as the first step
- Document the convention that humans should add their requirements, aspirations, branding guidelines, and tech preferences as refs pointing to docs/ files
- Consider whether to add a `purpose` field to refs in the schema (e.g., `purpose: input` vs `purpose: verify`) or keep it as a pure convention (docs/ = input, config files = verify)

## Acceptance Criteria

- [ ] Generated CLAUDE.md instructs agents to read doc refs before starting work
- [ ] Generated maslow skill includes ref-reading in all workflows
- [ ] Convention is clear about which refs are input vs verification targets
- [ ] Update maslow-tok example to include sample input refs (e.g., a brief PRD, branding notes)
- [ ] Scaffold tests pass
- [ ] Ensure no drift is created between the harness and scaffold
- [ ] An ADR is written to document the change

## Notes

- The beauty of refs is that they're lightweight. A human can add `docs/ideas.md` as a ref containing a stream-of-consciousness about what they want. The agent reads it, interprets it, makes decisions, writes ADRs.
- Refs could point to external URLs too (kind: url) — e.g., a competitor's app, a design inspiration, an API standard. The agent fetches and uses as context.
- This convention makes maslow projects self-documenting: the refs section is a reading list for anyone (human or agent) who wants to understand the project's intent.
- Schema change (adding `purpose` field) is optional and could be a separate task if the convention alone isn't sufficient.
