---
id: 25
title: Implement ref verification runner
status: todo
priority: critical
created: 2026-03-01
updated: 2026-03-01
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, refs, runner]
---

## Objective

Refs declared in maslow.yaml (kind: file, binary, url) are currently parsed but never verified. A required ref pointing to a missing file silently passes. Implement a ref verification runner that checks declared refs exist.

## Requirements

- `RunRefs(refs []spec.Ref) []evidence.RefResult` iterates refs and dispatches by kind
- `file` refs: `os.Stat(path)` — fail if not found and required, skip if not required
- `binary` refs: `os.Stat(path)` + check executable bit
- `url` refs: `http.Head(url)` with 10s timeout — pass on 2xx, fail otherwise
- Unimplemented kinds (package, container, endpoint, mcp): return status "skip"
- Results included in evidence report and affect verdict

## Acceptance Criteria

- [ ] RunRefs correctly dispatches by ref kind
- [ ] Required file ref missing → fail
- [ ] Non-required file ref missing → skip
- [ ] Binary ref checks executable permission
- [ ] URL ref validates with HTTP HEAD
- [ ] Unknown kinds return skip
- [ ] Results appear in verify report JSON
- [ ] Failing ref causes verdict to fail
- [ ] Unit tests cover all cases
