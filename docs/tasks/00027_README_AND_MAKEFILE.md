---
id: 27
title: Add README and Makefile, update scaffold next-steps
status: done
priority: medium
created: 2026-03-04
updated: 2026-03-05
assigned_to: adam
assigned_at: "2026-03-04T00:00:00Z"
depends_on: []
tags: [maslow, docs, developer-experience]
---

## Objective

Add a proper README.md with install instructions, project creation guide, commands reference, verification pipeline overview, and development instructions. Add a Makefile with standard build/test/verify targets. Update the scaffold command's "Next steps" output to align with the recommended workflow.

## What Was Done

- Added `README.md` covering install, scaffold usage, commands, verification pipeline, and dev workflow
- Added `Makefile` with build, test, vet, lint, validate, verify, and clean targets
- Updated `cmd/maslow/main.go` scaffold next-steps to guide users through: git init, write docs as refs, edit maslow.yaml, write first task, tell agent
- Added refs for README.md and Makefile to maslow.yaml
- Added contract verification for the Makefile

## Notes

This work was done outside the maslow task system. Task created retroactively.
