---
id: 20
title: Add multipart/form-data support for HTTP contract steps
status: done
priority: low
created: 2026-02-20
updated: 2026-11-19
assigned_to: "claude-subagent"
assigned_at: "2026-11-20T00:44:09Z"
depends_on: [13]
tags: [maslow, verification, contract, http]
---

## Objective

File upload is a core feature of many applications (MaslowTok's entire value prop is video upload) but HTTP contract steps only support string bodies. There's no way to send multipart/form-data requests with file attachments. This means upload endpoints — often the most complex and error-prone part of an API — can't be contract-tested.

## Requirements

- Extend the HTTP step schema to support multipart/form-data requests
- Add a `files` or `multipart` field to HTTP steps:
  - Each part has a field name, file path (or inline content), and optional content type
  - Support mixing file parts with regular form fields
- The runner constructs a proper multipart request with boundaries
- Expectations should work normally on the response (status, json_path, headers, etc.)
- Update CUE schema, embedded copy, Go types, and runner

## Acceptance Criteria

- [x] Schema accepts `multipart` field on HTTP steps with file and field parts
- [x] Runner constructs valid multipart/form-data requests
- [x] File parts read from the specified path relative to the project root
- [x] Form field parts are sent as text values
- [x] Content-Type header is automatically set to multipart/form-data with boundary
- [x] Response expectations work normally after multipart request
- [x] Missing file paths produce clear error messages
- [x] Unit tests cover: single file upload, multiple files, mixed files and fields, missing file
- [ ] Update maslow-tok example to include a video upload contract (dropped by the Decisions; `testdata/valid/http_multipart.yaml` and the `validate-command` contract replace it)

## Notes

- Example usage:
  ```yaml
  - action: http
    method: POST
    url: "${API_BASE_URL}/v1/videos"
    headers:
      Authorization: "Bearer ${{access_token}}"
    multipart:
      - field: title
        value: "Test Video"
      - field: video
        file: testdata/sample.mp4
        content_type: video/mp4
    expect:
      status: 201
      json_path: "$.id"
  ```
- Depends on task 13 (capture) because upload endpoints almost always require authentication
- Go's `mime/multipart` package handles the heavy lifting
- Consider file size limits for test fixtures — don't want huge files in the repo. Small sample files or generated test data is preferred.
- The RPG game example could use this for asset upload/import testing

## Decisions (human review, 2026-11-19)

All recommendations in the proposal below are accepted, and its updated requirements replace the
original ones: test files come from the head checkout (documented in ADR 0041, with a `kind:gap`
draft task), a missing or bad file is `error`, the body cap is 10 MiB, file paths are not
templated, and the maslow-tok example is dropped in favour of a `testdata` fixture.

## Proposed implementation

### 1. What happens today

An `http` step can send only a string body:

- The schema allows `method`, `url`, `headers`, `body` and `expect` on an `http` step
  (`schema/maslow.cue:199-205`). `#Step` is a closed definition, so a `multipart` key fails
  validation today. The Go type has the same fields (`internal/spec/spec.go:156-175`).
- `runHTTPStep` (`internal/runner/contract.go:574-609`) wraps `step.Body` in a `strings.Reader`,
  sets each header and calls the scenario's client. Because the body is a `strings.Reader`,
  `http.NewRequestWithContext` sets `GetBody`, so a 307 or 308 redirect replays it.
- Templating (`substituteStep`, `contract.go:454-509`) runs on `url`, `body`, `command`, `stdin`,
  `selector`, `text`, header values and args. Templates see only `{{.env.NAME}}` for allow-listed
  variables and `{{.cap.NAME}}` for captures (`applyTemplate`, `contract.go:524-548`; ADR 0026).
  A relative URL is joined to `base_url` (`resolveURL`, `contract.go:512-520`; ADR 0028).
- Each scenario gets one client with a cookie jar (`newScope`, `contract.go:115-118`; ADR 0027).
- Files are found relative to the scope's `dir` (`onDisk`, `contract.go:128-131`). `dir` is `""`
  (the directory Maslow runs in) except in a red-green base run, where it is a temporary worktree
  (`internal/verify/redgreen.go:60`). Artifact patterns are "repository-relative" and reject `.`,
  `..` and empty segments (`internal/runner/pathmatch.go:20-33`). Nothing reads a file to send it.
- A failed step becomes `failed_step` through `runSteps` and `failedResult`
  (`contract.go:260-284`). It records phase, index, action and `target`
  (`internal/evidence/report.go:137-145`). For HTTP the target is `"METHOD url"` after substitution
  (`stepTarget`, `contract.go:70-75`). The request body and headers are never recorded. `redact`
  replaces allow-listed env values in `error`, `output` and `target`
  (`internal/runner/contract_env.go:28-64`). Every failure has status `fail`
  (`contract.go:261`), and only `fail` is retried (`internal/runner/retry.go:12-14`).

**What is out of date in the task.** Task 13 is done, so the dependency is met. The example uses
`${API_BASE_URL}` and `${{access_token}}`, which no longer exist. Today it would be
`base_url: "{{.env.API_BASE_URL}}"` with `env: [API_BASE_URL]`, `url: /v1/videos`, and
`Bearer {{.cap.access_token}}`. "Relative to the project root" needs a precise rule (below).
Nothing validates or runs `examples/`, so "update maslow-tok example" would prove nothing. The
RPG and MaslowTok motivation comes from the removed harness. The task does not cover determinism,
redaction, red-green or the size of files read into memory. The proposed requirements below
replace the original ones for review. The original sections are left as they are.

### 2. Proposed change

**Proposed requirements**

- An `http` step may have `multipart`, a non-empty ordered list of parts, instead of `body`.
- Each part has `field` and exactly one of `value` (a text field), `file` (bytes read from a
  fixture) or `content` (inline text sent as a file). `file` and `content` parts may set
  `filename` and `content_type`. `content` requires `filename`. `file` defaults `filename` to the
  file's base name.
- `content_type` defaults to `application/octet-stream`. Maslow does not guess types from
  extensions, because `mime.TypeByExtension` reads the host's MIME tables and would differ between
  machines.
- `file` is a slash-separated path relative to the directory Maslow runs in, which is the same
  base `--spec` and `artifacts` use. It may not be absolute or contain `.`, `..` or empty
  segments. It is read through `os.Root`, so a symlink cannot escape that directory.
- The runner sets `Content-Type: multipart/form-data; boundary=...`. A step that also sets a
  `Content-Type` header, in any case, fails before sending.
- The boundary is deterministic: `maslow-` plus the first 32 hex characters of the SHA-256 of the
  encoded parts.
- `value`, `content` and `filename` are templated like `body`. `field`, `file` and `content_type`
  are not.
- Files are always read from the head working tree, including in a red-green base run.
- A missing, escaping or oversized fixture gives the scenario status `error`, not `fail`, and is
  not retried.
- The whole multipart body is capped at 10 MiB.
- Evidence format is unchanged. Response expectations, captures, cookies and `base_url` work as
  they do for any `http` step.

Updated example:

```yaml
contracts:
  - name: videos
    env: [API_BASE_URL]
    base_url: "{{.env.API_BASE_URL}}"
    scenarios:
      - name: upload
        steps:
          - action: http
            method: POST
            url: /v1/videos
            headers:
              Authorization: "Bearer {{.cap.access_token}}"
            multipart:
              - field: title
                value: "Test Video"
              - field: video
                file: testdata/sample.mp4
                content_type: video/mp4
            expect:
              status: 201
              json_path: "$.id"
```

**`schema/maslow.cue` and `internal/schema/maslow.cue`**: add `multipart?: [#Part, ...#Part]` to
the `http` branch. `#Part` is `field: string & !=""` plus a disjunction of three closed shapes for
`value`, `file` and `content`. `file` is constrained by regex to no leading `/`, no backslash and
no `.`, `..` or empty segment, matching `validatePattern`. `body` and `multipart` are mutually
exclusive. If CUE cannot express that cleanly, the runner rejects it before sending, and that
case becomes a runner test instead of an invalid fixture.

**`internal/spec/spec.go`**: add `Multipart []Part` to `Step`, and add a `Part` type with
`Field`, `Value`, `File`, `Content`, `Filename` and `ContentType`.

**`internal/runner/contract.go`** and a new `internal/runner/multipart.go`:

- `substituteStep` templates each part's `Value`, `Content` and `Filename`. Errors name the part,
  for example `multipart[1].value: ...`.
- `runHTTPStep` calls `buildMultipart(parts)` when `len(step.Multipart) > 0`. It opens
  `os.OpenRoot(".")`, reads each `file` part, enforces the 10 MiB cap and hashes the parts to get
  the boundary. It writes the parts in order with `multipart.Writer.SetBoundary` into a
  `bytes.Buffer`, so `GetBody` is still set and redirects replay the body. It returns the bytes
  and the content type.
- Errors use the spec's relative path, never an absolute one. `os.Root` reports the name it was
  given, so evidence is the same on every machine and in a temporary base worktree. For example:
  `step http POST http://h/v1/videos: multipart[1] (field "video"): file "testdata/sample.mp4": open testdata/sample.mp4: no such file or directory`.
- Fixture problems are wrapped in a `fixtureError` type. `failedResult` sets status `error` when
  `errors.As` finds one. `failed_step` is filled in the same way as for any other failure.
- `stepTarget` stays `"METHOD url"`. Part values are never written to evidence. An env value
  echoed back by the server is redacted from `output` by the existing `redact`.

**`internal/verify`, `internal/evidence`, `schema/evidence.schema.json`**: no change. Files come
from the working directory, not `sc.dir`, so `judgeRedGreen` needs no new option.

**Docs**: add a "Multipart uploads" subsection under "Writing a Spec" in `README.md`, add a
CHANGELOG entry, and update `docs/MAP.md` only if `multipart.go` is listed there.

**`maslow.yaml`**: add scenarios to `validate-command` (around line 170). The new valid fixture
should exit 0 with `is valid`. An invalid fixture with `file: ../secret` should exit 1, with output
naming `multipart`.

### 3. Tests

Write these first. Each must fail before the implementation, and for the stated reason. The new
runner tests do not compile until `spec.Part` exists, so add only the empty type first and
confirm they fail on behaviour.

`internal/runner/contract_multipart_test.go` uses an `httptest` server modelled on
`sessionServer` (`contract_session_test.go:13-29`). The server parses the form and replies with
JSON listing each part's field, filename, content type, length and SHA-256, plus the request's
`Content-Type`.

| Case                                                                                   | Expectation                                                                                                                                                    |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| single `file` part from a temp fixture                                                 | pass; the server sees the bytes, base-name filename and `application/octet-stream`                                                                             |
| two files and two values, interleaved                                                  | pass; parts arrive in spec order                                                                                                                               |
| `content` part with `filename` and `content_type`                                      | pass; the server sees both                                                                                                                                     |
| `value` from `{{.cap.id}}` and `{{.env.TOKEN}}`                                        | pass; if the server echoes `TOKEN`, `output` shows `[redacted:TOKEN]`                                                                                          |
| missing file                                                                           | status `error`; `failed_step` is `{steps, 0, http, "POST <url>"}`; error names the field and relative path; no absolute path; one attempt despite `retries: 2` |
| `file: ../x` and absolute path (bypassing CUE)                                         | status `error`, nothing sent                                                                                                                                   |
| symlink in the fixture dir pointing outside                                            | status `error` from `os.Root`, nothing sent                                                                                                                    |
| author sets `content-type` header                                                      | fails before sending; error names the header                                                                                                                   |
| `body` and `multipart` both set                                                        | fails before sending                                                                                                                                           |
| body over 10 MiB                                                                       | status `error`; message names the cap                                                                                                                          |
| same step run twice                                                                    | identical request `Content-Type`; identical marshalled result JSON                                                                                             |
| `ContractOptions{Dir: tmp}` where `tmp` lacks the fixture                              | pass; the file is read from the working directory, not `Dir`                                                                                                   |
| login sets a cookie, then a 307 to an upload endpoint that checks the cookie and parts | pass (ADR 0027 and `GetBody`)                                                                                                                                  |
| relative `url` with `base_url`                                                         | pass (ADR 0028)                                                                                                                                                |

`internal/schema/validate_test.go`: add `testdata/valid/http_multipart.yaml`, covering all three
part kinds. Add invalid fixtures for an absolute `file`, a `..` segment, `value` together with
`file`, a missing `field`, `content` without `filename`, and `body` together with `multipart`.
These invalid fixtures already fail today because of the unknown `multipart` key, so each test
must assert the specific constraint in the error, not just that validation failed.

Run `go test ./...`, `go vet ./...`, `cue vet schema/maslow.cue`,
`diff schema/maslow.cue internal/schema/maslow.cue` and `maslow verify --profile quick`.

### 4. ADR

Write ADR 0041, "Multipart HTTP steps". It records these decisions: parts are explicit and
ordered; fixtures are read through `os.Root` from the working directory and never from the base
worktree; the boundary is content-derived; types are not sniffed; fixture problems are `error`;
there is a 10 MiB cap; and evidence does not change. Its consequences note the gap in open
question 1. No accepted ADR is amended.

### 5. Risks and open questions

1. **Under `--spec-from base` (ADR 0019), fixtures still come from the head.** A pull request
   cannot change the spec, but it can replace `testdata/sample.mp4` with a file that weakens the
   test. _Recommendation:_ accept this for now. Write it in the ADR, advise listing fixture
   directories in `policy.protected`, and open a `kind:gap` draft task to read spec contracts'
   fixtures with `gitrepo.Show` at the base commit.
2. **Should red-green base runs read fixtures from the base worktree, as CLI steps do?** If they
   did, a fixture added in the same pull request would be missing on the base. The base run would
   fail for the wrong reason and count as red. _Recommendation:_ no. Read from the head. The
   fixture is test input, like the overlay, not code under test.
3. **Should a missing fixture be `error` rather than `fail`?** This adds a second failure status to
   `failedResult`. _Recommendation:_ yes. It is a spec problem, a retry cannot fix it, and
   red-green already treats a base `error` as not red.
4. **Should `file` be templated?** Templating would allow per-environment fixtures, but paths
   could then come from env, differ between machines and be harder to review.
   _Recommendation:_ no. Revisit if someone needs it.
5. **What should the size cap be?** The body is held in memory so that redirects can replay it.
   _Recommendation:_ 10 MiB, as a constant. Make it configurable only if real specs need more.
6. **Should evidence record the parts, for example field names and fixture SHA-256s in
   `failed_step`?** This would help reviewers when a fixture is untracked. _Recommendation:_ no,
   not in this task. Tracked fixtures are pinned by `git_sha`, and adding the field changes the
   evidence schema.
7. **Should the maslow-tok example still be updated?** Nothing validates `examples/`.
   _Recommendation:_ skip it and use the self-hosted `testdata` fixture and `validate-command`
   contract instead. If the example is wanted, use a `content` part so no binary file is
   committed.
8. **Risk: CUE may not express the mutual exclusions cleanly** (exactly one of
   `value`/`file`/`content`; `body` versus `multipart`). _Recommendation:_ try closed
   disjunctions first. Fall back to a runner check, keep it tested, and note it in the ADR.
