---
id: 45
title: Run checks and contracts isolated from the evidence writer
status: draft
priority: medium
created: 2026-11-19
updated: 2026-11-19
assigned_to: ""
assigned_at: ""
depends_on: [35]
tags: [kind:gap, evidence, security]
---

## Objective

Task 35 moved evidence production into a `pull_request_target` workflow that `main` defines (ADR
0040). One risk remains, and it was accepted. The checks and contracts still run the change's code in
the job that writes `verify.json`. On a GitHub-hosted runner that code has passwordless sudo, so a
change written to do it can:

- rewrite the report after `maslow verify` writes it, or edit it before the upload step;
- leave a background process running after its check, because Maslow does not run checks in their
  own process group or kill one when the check ends;
- read the runner's `ACTIONS_RUNTIME_TOKEN` and upload its own `maslow-evidence-<head>` artifact
  before the real upload.

The evidence's results are trustworthy only if the code under test cannot touch the evidence writer.

## Questions to answer

- Where do checks run? Options include a container or VM with no access to the report path or the
  runner's tokens, a separate job that hands back only raw results that the writer re-validates, or
  a remote executor.
- What does the writer trust from the isolated run, and how does it know the results are complete?
- How does this generalise beyond GitHub Actions, given Maslow stays a verifier (ADR 0016)?
- Is process-group cleanup worth doing anyway as cheap hardening, even though it is not a trust fix?

## Notes

Found while implementing task 35. See ADR 0040, "What this proves and what it does not".

Review (2026-11-20): process-group cleanup is split out as task 49. The isolation design stays a
draft until Damocles acts on evidence without a human reading the diff.
