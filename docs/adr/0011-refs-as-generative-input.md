# ADR-0011: Refs as Generative Input Convention

## Context

Refs in maslow.yaml were originally designed as verification targets ("does this file exist?"). However, in agent-driven generative workflows, refs serve a dual purpose: documentation refs (PRDs, branding guides, tech decision docs) are the agent's primary input — they define what to build. The harness needed to teach agents this distinction.

## Decision

Establish a convention that refs pointing to docs/ files are generative input (read before building), while refs pointing to config files or binaries are verification targets. This is enforced purely through harness guidance (CLAUDE.md and the maslow skill), not through schema changes.

## Consequences

- Agents now read doc refs before starting work, leading to better-informed decisions.
- Humans can steer agent behavior by adding docs/ refs containing requirements, preferences, or inspiration.
- The refs section becomes a self-documenting reading list for the project's intent.
- No schema changes required — the `purpose` field idea is deferred unless the convention proves insufficient.
