# ADR 0009: Use Go text/template for Contract Variable Substitution

## Context

Task 13 introduced variable capture and substitution in contract steps using a homegrown regex approach: `${VAR}` for environment variables and `${{var}}` for captured scenario variables. This worked but had limitations:

- No escaping mechanism (can't produce a literal `${{` in output)
- No expressions, pipelines, or functions
- Two different syntaxes for two variable sources, ad-hoc rather than standard
- The `${{...}}` syntax resembles GitHub Actions but behaves differently
- Extending it (e.g., default values, string manipulation) would mean growing a custom mini-language

Go's `text/template` is in the standard library, well-documented, battle-tested, and already uses `{{...}}` syntax. It provides escaping, pipelines, custom functions, and conditional logic out of the box.

## Decision

Replace the homegrown regex-based substitution with Go `text/template`. Template data uses two namespaces:

- `{{.env.NAME}}` — environment variables
- `{{.cap.NAME}}` — captured scenario variables

Example: `{{.env.BASE_URL}}/api/v1/users/{{.cap.user_id}}`

## Consequences

- **Standard syntax**: agents and humans already know Go template syntax; no custom grammar to learn
- **Extensibility**: pipelines (`{{.cap.token | upper}}`), conditionals, and custom functions come free when needed
- **Zero new dependencies**: `text/template` is in the Go standard library
- **Breaking change**: the old `${VAR}` and `${{var}}` syntax is no longer recognized; existing specs using them must migrate to `{{.env.VAR}}` and `{{.cap.var}}`
- **Error messages change**: template parse/exec errors replace the old substitution behavior; these are generally more informative but reference Go template internals
