# ADR 0046: Run Checks With a Scrubbed, Allow-Listed Environment

- **Status**: Accepted
- **Date**: 2026-11-21

## Context

A check command is the change's own code, but it inherited Maslow's whole environment, so any token in the CI job (`GITHUB_TOKEN`, `ACTIONS_RUNTIME_TOKEN`, cloud credentials) was readable by it. Contracts already read only the variables in their `env` list and have those values redacted from evidence (ADR 0026); checks did not (task 41).

## Decision

A check runs with a fixed base set plus the names in its new `env` list, the listed values (4 bytes or longer) are redacted from its output and error, and `checks.inherit_env: true` restores the old inheritance and is recorded as `inherited_env: true` on every check result.

The base set, each passed only when set:

| Variable | Why |
|----------|-----|
| `PATH` | Finding the tools a check runs. |
| `HOME` | Git, Go, npm and most toolchains keep config and caches under it; Go's default `GOCACHE` and `GOPATH` derive from it. |
| `TMPDIR` | Where temp files go; some runners point it at a dedicated disk. |
| `USER`, `LOGNAME` | Read by tools that label files or look up the current user. |
| `LANG`, `LC_ALL`, `LC_CTYPE` | Locale, so a check's text handling and messages match a plain shell. |
| `XDG_CACHE_HOME`, `XDG_CONFIG_HOME` | Where Linux tools, Go included, put caches and config when the user moved them. |
| `CI` | Switches many tools to non-interactive behaviour; it is a flag, not a secret. |
| `SystemRoot`, `ComSpec`, `PATHEXT`, `TEMP`, `TMP`, `USERPROFILE`, `LOCALAPPDATA`, `APPDATA` | Windows programs fail without these, and Go's cache lives under `LOCALAPPDATA`. Unset on Unix, so they add nothing there. |

Left out on purpose, so a check lists them when it needs them: `TERM` and `SHELL` (checks run without a terminal or a shell), `GOCACHE`, `GOPATH`, `GOMODCACHE`, `GOFLAGS` and other toolchain variables (language-specific, and `GOFLAGS` changes what a check does), `HTTP_PROXY` and friends (a proxy URL can carry a password), `GITHUB_ACTIONS` and other vendor variables (the `GITHUB_*` and `ACTIONS_*` namespaces hold tokens next to harmless values, so none are let through by default).

## Consequences

- Breaking: a check that read an inherited variable now sees it unset until the spec lists it in `env` or sets `checks.inherit_env: true`. A toolchain variable a user set by hand, such as a custom `GOCACHE`, falls back to its default, which costs cache hits rather than failing.
- A listed variable's value is redacted wherever it appears in output, so a listed path such as `GOPATH` shows as `[redacted:GOPATH]` in check output. Base-set values are not redacted.
- CLI contract steps are unchanged and still inherit Maslow's environment; only their templates are limited to the contract's `env` list. ADR 0048 later scrubbed them and audit binaries with the same base set.
