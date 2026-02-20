---
id: 8
title: Add a maslow harness command
status: done
created: 2026-02-19
updated: 2026-02-19
assigned_to: claude
assigned_at: 2026-02-19T00:00:00Z
tags: [maslow, harness]
---

## Objective

Part of the special sauce of Maslow is that a scaffolded project brings an opinionated agentic harness along with it. Adding a `maslow harness` command will introduce this as a first concept in the system.

## Requirements

- Add a `maslow harness` command that will manage the harness in a project.
- `maslow harness install` will install the maslow harness into any project that doesn't have it. It will smartly test and prompt to make sure it doesn't overwrite existing files.
  - Options:
    - `--force` to overwrite existing files
    - `--dry-run` to show what would be done without actually doing it
  - On Conflict:
    - If a file exists in the project and would be overwritten by installing the harness the user will be given the following options:
      - `abort` to abort the operation
      - `overwrite` to overwrite the existing file
      - `skip` to skip the file
      - `rename and reference` to rename the existing file and reference it in the new file. This allows you to install the harness and keep the current project's existing agent instructions.
- `maslow harness update` will update the maslow harness to the latest version.
  - Options:
    - `--force` to overwrite existing files
    - `--dry-run` to show what would be done without actually doing it
- `maslow harness detach` will detach the maslow harness so that it can no longer be
  updated to the latest version. This will have to create a sentinel file in the project that prevents the harness from being updated. The purpose of this is to allow developers to modify the harness in their own way without having to worry about it being overwritten by the latest version.
