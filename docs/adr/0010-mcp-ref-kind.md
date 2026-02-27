# ADR-0010: First-class `kind: "mcp"` for Ref

## Context

Projects built with Maslow increasingly depend on MCP servers (browser automation, GitHub, database access, etc.). Task 23 proposed a convention using `kind: url` with `mcp://` pseudo-scheme, but that hack cannot carry installation metadata (transport, command, args, env vars). Agents need enough detail to potentially install and configure MCP servers themselves.

## Decision

Add `"mcp"` to the `#Ref` kind enum with conditional fields for MCP-specific metadata: `name` (required), `source`, `transport`, `command`, `args`, `url`, and `env`. For MCP refs, `path` holds the package identifier and `name` is the human-readable server name.

## Consequences

- Agents can discover MCP dependencies declaratively and potentially auto-install them.
- Supersedes the `mcp://` pseudo-scheme convention from task 23.
- MCP refs are declarative only for now — no verification runner. Verification is future work.
- The `env` map signals which environment variables are required, with values as defaults or placeholders.
