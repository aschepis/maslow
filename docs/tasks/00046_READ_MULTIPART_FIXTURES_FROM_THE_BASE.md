---
id: 46
title: Read spec contracts' multipart fixtures at the base commit under --spec-from base
status: draft
priority: low
created: 2026-11-19
updated: 2026-11-19
assigned_to: ""
assigned_at: ""
depends_on: [20]
tags: [kind:gap, contract, security]
---

## Objective

Task 20 added `multipart` parts to `http` steps. A `file` part is always read from the head working
tree (ADR 0041). Under `--spec-from base` (ADR 0019) the spec comes from the base, so a pull request
cannot edit the contract, but it can replace a fixture such as `testdata/sample.mp4` with one that
makes the test weaker. Nothing in the evidence shows that the fixture changed.

Today the only defence is listing fixture directories in `policy.protected`, which the spec author
has to remember.

## Questions to answer

- Should spec contracts read `file` parts with `gitrepo.Show` at the base commit when the spec comes
  from the base, while overlay contracts keep reading the head, since their fixtures may be new?
- How does a red-green base run fit in? It reads fixtures from the head on purpose (ADR 0041).
- Would recording each fixture's path and SHA-256 in the evidence be enough instead, or as well? That
  changes the evidence schema.
- Should Maslow treat fixture paths as protected automatically when `--spec-from base` is set?

## Notes

Found while implementing task 20. See ADR 0041, "Consequences".

Review (2026-11-20): stays a low-priority draft; `policy.protected` on fixture directories is the
workaround. If a small step is wanted first, record each `file` part's path and SHA-256 in the
evidence so a reviewer can see a fixture changed.
