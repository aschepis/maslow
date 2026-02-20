---
id: 11
title: Add filesystem assertion support for contracts
status: draft
created: 2026-02-19
tags: [maslow, verification, contract, filesystem]
---

## Objective

Add a `file` action type to contracts so Maslow can verify filesystem artifacts produced by builds, generators, or CLI tools.

## Requirements

- New step action type `file` with fields: path, action (exists, not_exists, contains, matches, size_lt, size_gt)
- Expectation support: file existence, content matching (substring, regex), size bounds, permissions
- Support glob patterns for checking multiple files (e.g. "dist/*.js exists")

## Notes

- Critical for verifying build outputs, code generators, scaffolding tools, and asset pipelines
- Complements artifact_size budgets with more granular file-level assertions
- Useful for CLI tools: run a command, then verify it produced the expected files
