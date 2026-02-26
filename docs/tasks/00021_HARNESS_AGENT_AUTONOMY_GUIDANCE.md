---
id: 21
title: Update harness with agent autonomy and gap-discovery conventions
status: todo
priority: high
created: 2026-02-22
updated: 2026-02-22
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, harness, scaffold, conventions]
---

## Objective

The harness (CLAUDE.md, maslow skill, task convention) needs to teach agents the right operating model for generative work:

- **Agents are trusted to make decisions.** Technology choices, architecture, design, library selection — the agent decides, writes an ADR, and keeps building. The human can always revert via git if they disagree.
- **Draft tasks are for platform gaps, not permission.** When the agent discovers that maslow's verification or harness capabilities are insufficient for what it's building (e.g., can't test auth flows because capture isn't implemented), it creates a draft task describing the gap. Draft tasks are NOT for decomposing work or asking approval for decisions.
- **ADRs are the decision record.** Every material decision (stack choice, database, framework, architectural pattern) gets a short ADR. This is the audit trail. Keep them short: context, decision, consequences.

This is the foundational convention that makes the entire generative workflow function. Without it, agents either over-ask (slow) or under-document (opaque).

## Requirements

- Update the CLAUDE.md template in `internal/scaffold/scaffold.go` (`GenerateClaudeMD()`) with:
  - A "Decision Making" section: agents make decisions autonomously, record as ADRs, keep building. No need to ask permission for technology or design choices unless refs explicitly constrain them.
  - A "Draft Task Protocol" section: draft tasks are specifically for discovering gaps in maslow verification/harness capabilities that prevent the agent from confidently verifying what it's building. Not for work decomposition or decision approval.
  - A "Progressive Verification" section: as you build, add corresponding verifications to maslow.yaml. Every API endpoint gets a contract. Every build artifact gets a size budget. Use what the schema can express; create draft tasks for what it can't.
- Update the maslow skill in `GenerateSkillMD()` with:
  - A "Workflow: Greenfield Build" section that emphasizes: read refs → start building → make decisions (ADRs) → add verifications → flag gaps (draft tasks)
  - Guidance on when to create a draft task vs when to write an ADR vs when to just code
- Update the task convention in `GenerateTaskConvention()` with:
  - A tag convention for draft tasks created by agents: `kind:gap` for platform/verification gaps, `kind:capability` for missing tools/MCPs
  - Clarify that agents creating draft tasks is for gap discovery, not work decomposition
- Propagate all changes: scaffold generates the updated versions, harness install/update picks them up

## Acceptance Criteria

- [ ] Generated CLAUDE.md includes Decision Making section with ADR guidance
- [ ] Generated CLAUDE.md includes Draft Task Protocol section scoped to gap discovery
- [ ] Generated CLAUDE.md includes Progressive Verification section
- [ ] Generated maslow skill includes Greenfield Build workflow
- [ ] Generated task convention includes agent draft task tag conventions
- [ ] Scaffold tests pass with updated content
- [ ] Harness install/update propagates the new content
- [ ] Self-hosting: update maslow-agentic's own CLAUDE.md to match the new template

## Notes

- The key insight: the human's review surface is ADRs (decisions already made, revertible) and draft tasks (platform gaps, actionable). This is a pull model — the human reviews at their pace, not a push model where the agent blocks waiting for approval.
- This convention makes refs more powerful implicitly: if a ref points to a doc saying "use PostgreSQL," the agent reads it and follows it. If no ref constrains the choice, the agent decides.
- The "just revert it" safety model only works because maslow.yaml verification catches regressions. If the agent makes a bad choice, verification failures surface it.
