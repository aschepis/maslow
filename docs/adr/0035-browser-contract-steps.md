# ADR 0035: Browser Contract Steps Driven by chromedp

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

Contracts could drive HTTP and CLI flows but not a browser, so a web UI was verified only by a `cli` step running a separate test tool, whose failure the evidence saw as "the test command". Driving the browser from a contract step puts each UI action in the evidence, and lets it share the scenario's session (ADR 0027) and an endpoint audit's base URL (ADR 0028).

## Decision

A `browser` step runs one operation (`navigate`, `click`, `fill`, `wait`, `assert` on text, URL, visibility or enabled state) in a headless Chrome that Maslow drives in process through chromedp, the Chrome DevTools Protocol client, rather than by shelling out to Playwright or a WebDriver server.

## Consequences

- No Node, driver binary or version pairing is needed, only Chrome or Chromium: `MASLOW_BROWSER` names it, otherwise the usual names and install paths are searched. Without one, a browser step fails with an error naming `MASLOW_BROWSER`, and the repository's own browser tests skip so `go test ./...` stays green. Other browsers are out of reach.
- Each scenario starts its own browser on its first browser step and closes it after teardown, so scenarios stay independent at about half a second each. Cookies flow both ways around every browser step; the HTTP jar exposes only names and values, so its cookies enter the browser scoped to the whole host.
- Runs gate deterministically: `click`, `fill` and `assert` fail at once on a missing or hidden element, and only `wait` waits, with a 30-second default `timeout`. A failing step records `page_url` and console errors on `failed_step` and a screenshot at `reports/browser/<contract>/<scenario>.png` in the result's `artifacts`; a scenario's stale screenshot is removed before it runs.
