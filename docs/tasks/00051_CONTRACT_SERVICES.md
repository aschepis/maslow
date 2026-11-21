---
id: 51
title: Let a contract start the servers its scenarios hit
status: draft
priority: high
created: 2026-11-21
updated: 2026-11-21
assigned_to: ""
assigned_at: ""
depends_on: [49]
tags: [kind:gap, contracts]
---

## Objective

Since v0.10.0 (ADR 0044, task 49, PR #42) every check and CLI step, including contract `setup`,
runs in its own process group, and the group is killed when the step ends. A setup step can no
longer leave the change's server running for the scenarios. So there is no supported way for a
contract to test a server it starts itself. HTTP, browser, multipart (ADR 0041) and
metric/log/alerts (ADR 0042) contracts can only target a server that is already running. That
also breaks red-green: the base run needs the base commit's server, and only Maslow can start it.

## Proposal

**Spec.** Add `services:` to `#Contract`. Each entry:

```yaml
services:
  - name: api                  # unique in the contract; [a-z0-9_-]+
    command: go                # command + args, as in CLI steps; no shell
    args: [run, .]
    dir: examples/api-demo     # relative to the run's directory; default "."
    env: [DATABASE_URL]        # allow-list, ADR 0026 semantics
    port: auto                 # auto (default) | an int | none
    ready:                     # exactly one of http or tcp
      http: {url: /healthz, status: 200}  # "/" URLs resolve against the service url
      # tcp: true              # connect to the service's port
      timeout: 30s             # default 30s; also the startup timeout
      interval: 200ms          # default 200ms
```

`command`/`args` match CLI steps, since services live in contracts. A shell is one arg away
(`command: sh, args: [-c, ...]`).

**Ports.** `port: auto` picks a free loopback port, passes it as `PORT` and exposes it as
`{{.services.api.port}}` and `{{.services.api.url}}` (`http://127.0.0.1:PORT`). A fixed `port`
is passed the same way. `port: none` sets no `PORT`, and `tcp` readiness is then invalid. Auto
ports stop a retried or red-green run colliding with a server a developer left running, or with
a parallel CI job on the same host. Bind-then-release leaves a small race; we accept it. A
contract's `base_url` can be `{{.services.api.url}}`, so scenarios keep relative URLs.

**Env.** The service gets `PATH`, `HOME`, `TMPDIR`, `PORT` and the names in its `env` list,
nothing else. This matches where task 41 takes checks. Listed values are redacted from its output,
as for contract output (ADR 0026).

**Lifecycle.** Per contract attempt (ADR 0039):

1. Start services in list order, each in its own process group, registered with the
   `running` registry so `KillRunning` (Ctrl-C, SIGTERM) kills them too.
2. Probe each until ready, or until `timeout`. A service that exits first is not ready.
3. Run contract setup, the scenarios still pending, then teardown, as today.
4. Kill each group in reverse order, after teardown, and wait for it.

A retry restarts services, so every attempt starts clean. That matches ADR 0039, where an attempt
reruns contract setup.

Implementation: a `startService` in `procgroup_unix.go` that starts the command and does not
wait. Output goes into a capped ring buffer. It returns a stop func that kills the group and
reaps the leader.

**Evidence.**
- A service that is not ready in time makes every scenario `error` (not `fail`, so not retried),
  with error `service api not ready after 30s: <last probe error>` and the service's output.
- A failed or errored scenario gets `services: [{name, exit_code?, output, output_truncated}]`,
  holding the tail of each service's output. `--output-limit` caps it, and values are redacted.
  `exit_code` is set if the service died early, so "connection refused" comes with its cause.
- Passing scenarios carry no service output. The field is additive, so `evidence_version`
  stays `"2"` and gets a CHANGELOG "Evidence format" note.

**Red-green.** `dir` resolves against `ContractOptions.Dir`, so on the base run the service
starts from the base worktree and runs the base code. No other change is needed.

**Overlays.** Overlay contracts may declare services under the same rules. An overlay service
runs the change's code, as an overlay CLI step already does.

**Audit.** Endpoint targets (ADR 0028) replace `base_url` and do not start services. A template
that reads `.services` in audit fails the step with an error naming the service. A relative-URL
contract therefore works in both `verify` (local service) and `audit` (deployed URL).

**Windows.** There are no process groups. Maslow starts the service, kills the direct child after
teardown, and documents that its descendants may leak. On Windows the console's Ctrl-C reaches the
child directly. Job objects can come later.

## Alternatives considered

- **`background: true` on setup steps.** This opts out of ADR 0044's cleanup, with no readiness
  probe, no logs in the evidence and no defined kill point. Specs would go back to `sleep 2`.
- **Rely on CI to start servers.** This cannot serve red-green, which needs the base commit's
  server. Local and CI runs would differ, and Maslow could not attest what it tested.
- **docker-compose integration.** It is a heavy dependency, slow, and not on every runner. A
  service can still run `docker compose up` (foreground) when a project wants it.

`services` wins because it is the smallest change that keeps ADR 0044's guarantees: every process
is still in a group Maslow kills. It gives readiness and logs as evidence, and works on the base.

## Questions for the human

1. **Per-contract services, or also a shared top-level `services:` list?** Recommendation:
   per-contract only for now. Each contract stays self-contained, overlays carry their own, and
   retries can restart cleanly. Add shared services later if startup cost shows up in practice.
2. **Default port mode?** Recommendation: `port: auto`, passed as `PORT` and exposed as
   `{{.services.NAME.url}}`. Fixed ports are allowed for servers that cannot read `PORT`.
3. **Restart services on each retry attempt, or keep them across attempts?** Recommendation:
   restart. A retry should start from a clean server, as it does from a clean setup (ADR 0039).
4. **Is a readiness timeout `error` or `fail`?** Recommendation: `error`, not retried. It means
   the contract could not run. Slow startup is handled by `ready.timeout`, not retries.
5. **Should a service inherit Maslow's environment, as CLI steps do today?** Recommendation:
   no. Give it a minimal environment plus its `env` list, as task 41 proposes for checks. A
   server is the change's own code and is the likeliest thing to log a token.
6. **What about `{{.services}}` in audit?** Recommendation: never start services in audit, and
   fail any step that reads `.services`. Contracts meant for both modes use `base_url`.

## Acceptance Criteria

- [ ] `#Contract.services` in `schema/maslow.cue` (copied to `internal/schema/`), with
      `spec.Service` types and fixtures in `testdata/valid/` and `testdata/invalid/`. Invalid
      fixtures cover a duplicate name, both or neither of `http`/`tcp`, and `tcp` with `port: none`.
- [ ] `{{.services.NAME.url}}` and `.port` resolve in step templates and `base_url`.
- [ ] Tests (Unix): http and tcp readiness pass; readiness timeout gives `error` with output;
      a service that exits before ready gives `error` with its exit code; a crash during
      scenarios fails them with the service tail and exit code; the group is gone after
      teardown, including a grandchild; `KillRunning` kills a running service; a retry starts
      a fresh service on a new auto port; output is capped and redacted.
- [ ] Red-green test: the base run starts the service from the base worktree.
- [ ] Audit test: an endpoint target starts no service and a `.services` template errors.
- [ ] `examples/api-demo` reads `PORT`, and its contract declares the server as a service, uses
      `base_url: "{{.services.api.url}}"` and relative URLs. Maslow's own `maslow.yaml` gains a
      contract that runs `maslow verify` on that example (or a tiny fixture server) and passes.
- [ ] Evidence schema gains the optional `services` field; CHANGELOG notes it.
- [ ] A new ADR (next free number) records the decisions; ADR 0044 gets a pointer from its first consequence.
- [ ] README, `docs/MAP.md` updated; `go test ./...`, `go vet ./...`, `cue vet schema/maslow.cue`
      and `maslow verify --profile quick` pass.
