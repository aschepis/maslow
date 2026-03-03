---
id: 23
title: Add MCP and capability discovery convention to harness
status: done
priority: medium
created: 2026-02-22
updated: 2026-03-01
assigned_to: claude
assigned_at: "2026-03-01T00:00:00Z"
depends_on: [21]
tags: [maslow, harness, scaffold, mcp, capabilities]
---

## Objective

An agent's ability to build and verify depends on what tools and MCPs are available to it. A browser MCP means it can visually test the UI. A design MCP means it can generate logos. A deployment MCP means it can ship. The harness doesn't teach agents to inventory their capabilities or flag missing ones.

When an agent discovers it needs a capability it doesn't have (e.g., "I need to generate a logo but have no image generation MCP"), it should create a draft task flagging the gap — exactly the gap-discovery pattern from task 21.

## Requirements

- Update the CLAUDE.md template to include guidance: "At the start of a session, note what MCPs and tools are available to you. If you need a capability you don't have (image generation, browser testing, deployment, database access), create a draft task tagged `kind:capability` describing what you need and why."
- ~~Document a convention for how refs can signal expected capabilities using `kind: url` with `mcp://` pseudo-scheme~~ **Superseded by ADR-0010**: The schema now supports `kind: mcp` with first-class fields (`name`, `transport`, `command`, `args`, `env`, etc.). Use this instead of the `mcp://` pseudo-scheme:
  ```yaml
  refs:
    - kind: mcp
      path: "@anthropic/mcp-browser"
      name: browser
      description: Browser for visual testing
      transport: stdio
      command: npx
      args: ["-y", "@anthropic/mcp-browser"]
      required: false
  ```
  These refs don't verify anything — they signal to the agent "you should have this capability for this project." If the ref is `required: true` and the capability is missing, the agent should flag it immediately.
- Update the maslow skill with a "Capability Check" step in workflows

## Acceptance Criteria

- [ ] Generated CLAUDE.md includes capability discovery guidance
- [ ] Generated maslow skill includes capability check in workflows
- [ ] Convention documented for using refs to signal expected MCPs/tools
- [ ] Agent knows to create `kind:capability` draft tasks when a needed capability is missing
- [ ] Scaffold tests pass
- [ ] Ensure no drift is created between the harness and scaffold

## Notes

- This is especially important for the "Build a TikTok clone" scenario: the agent might need browser testing (to verify the web app renders), image generation (to create placeholder assets), deployment tooling (to stand up a staging environment for contract testing against a live endpoint), etc.
- The `mcp://` scheme in refs is a convention, not a protocol — it just signals "this is a tool capability, not a file or URL to fetch"
- For v1, this is purely a harness convention change. Future work could have maslow verify that declared MCP capabilities are actually available.
- If the agent has access to a browser MCP, it could do visual regression testing. If it has a database MCP, it could seed test data. The capability inventory directly affects what verifications are possible.
