---
id: 28
title: Verify web UIs with a browser contract step
status: draft
priority: medium
created: 2026-11-18
updated: 2026-11-18
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [kind:gap, kind:capability, contracts, verification]
---

## Objective

Contracts can drive HTTP and CLI flows but not a browser, so Maslow cannot verify that a web UI
actually works. End-to-end runs through the real UI are the hardest checks for an agent to fool,
and the step-by-step failure evidence contracts already record (ADR 0025) suits them well.

## Requirements

- A contract step that drives a headless browser: navigate, click, fill, wait for a selector, and
  assert on text, URL or element state.
- A failing browser step records the step, the page URL, console errors and a screenshot, listed
  through the scenario's existing `artifacts` field.
- Browser steps share the scenario's session the way HTTP steps share its cookie jar (ADR 0027).
- An endpoint audit target's base URL replaces the contract's `base_url` for browser steps too
  (ADR 0028).
- Runs are deterministic enough to gate on: no implicit retries, explicit waits only.

## Acceptance Criteria

- [ ] Schema accepts the new step and rejects malformed ones (fixtures in `testdata/`).
- [ ] A contract against a local test page passes, and a broken page fails with a screenshot in the evidence.
- [ ] `maslow.yaml` or a fixture exercises it; README documents the browser dependency.
- [ ] ADR records the driver choice.

## Notes

Today a `cli` step can run a Playwright or similar test and list its screenshots through
`artifacts`. That works but hides the steps from the evidence, so the failing step is just "the
test command". Open questions: embed a CDP client (e.g. chromedp) or shell out to a driver;
how the browser binary is found or declared in `toolchain`.
