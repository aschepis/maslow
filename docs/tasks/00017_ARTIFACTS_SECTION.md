---
id: 17
title: Add artifacts section to schema for required output verification
status: draft
priority: medium
created: 2026-02-20
updated: 2026-02-20
assigned_to: ""
assigned_at: ""
depends_on: []
tags: [maslow, verification, schema, artifacts]
---

## Objective

There's no way to declare "these files must exist after a build." The MaslowTok spec originally had an `artifacts` section with required and optional paths (docs/openapi.yaml, docs/runbook.md, reports/verify.json, reports/junit.xml, etc.). This is a common verification need: did the build produce the expected outputs?

This overlaps with but is distinct from the `refs` section (which tracks input/config files) and `artifact_size` budgets (which check size of specific files). Artifacts declares the expected outputs of the build/verification process.

## Requirements

- Add an `artifacts` section to the CUE schema with required and optional file paths
- Add an artifacts verification step to the verify pipeline
  - Check required artifacts exist (fail if missing)
  - Check optional artifacts and note presence/absence (don't fail if missing)
  - Support glob patterns (e.g., `reports/*.json`)
- Include artifact verification results in the evidence report
- Update embedded schema copy and Go types

## Acceptance Criteria

- [ ] Schema accepts `artifacts` section with `required` and `optional` string arrays
- [ ] `maslow verify` checks required artifacts exist and fails if any are missing
- [ ] `maslow verify` reports optional artifact presence without failing
- [ ] Glob patterns work in artifact paths
- [ ] Evidence report includes artifact verification results (which exist, which missing)
- [ ] Missing required artifact error messages include the expected path
- [ ] Unit tests cover: all required present, required missing, optional missing, glob patterns
- [ ] Update maslow.yaml (self-hosting spec) to declare its own artifacts (reports/verify.json)

## Notes

- Simple schema:
  ```yaml
  artifacts:
    required:
      - "docs/openapi.yaml"
      - "reports/verify.json"
    optional:
      - "reports/junit.xml"
      - "reports/k6-summary.json"
  ```
- This pairs naturally with the filesystem assertions task (task 11) but is simpler — just existence and glob matching, no content inspection
- Consider whether artifacts should support size constraints inline or if that stays in budgets
- The RPG game example would use this for `dist/` build outputs
