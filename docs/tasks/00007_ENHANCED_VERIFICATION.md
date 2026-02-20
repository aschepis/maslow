---
id: 7
title: improved actions/contracts/verifications in maslow
status: in_progress
assigned_to: claude-opus
assigned_at: "2026-02-19T00:00:00Z"
created: 2026-02-19
tags: [maslow, verification, contract, actions]
---

## Objective

If you look at examples/todo-app/maslow.yaml you can see that the contracts are very simple. They are just a list of bash commands (e.g. curl) are expected to be performed and return with a 0 exit code.

Maslow must support more robust actions and contracts. For instance, the example todo app contracts should be able to specify http responses, verifying data in resulting json (jsonpath, jq, etc) or text.

## Requirements

- Support for HTTP based actions in maslow that can check http responses, json via jsonpath, text substring, or regular expressions. It should be able to do any or all of these on a single action.
- Perform research and write draft tasks for future functionality that would be useful in auditing and verifying other types of software than web apps.

## Acceptance Criteria

- [ ] Support HTTP based actions in maslow
- [ ] Clean, extensible code architecture for adding more in the future.
- [ ] Example app is updated to use the new actions
- [ ] Example app is built from scratch by a subagent in a new folder and the maslow skill is used verify it.
