---
id: 6
title: Prove Maslow Works: Build MVP TODO app
status: draft
created: 2026-02-19
tags: [maslow, apps]
---

## Objective

In order to prove that Maslow works, we will use it to build a simple
web-based TODO app. This will be a simple app in its own repo, described by a maslow yaml file, harnessed by maslow, built by an agent from the spec and harness, and verified by maslow.

## Requirements

- Identify and addressgaps in the maslow spec and harness that would prevent the app from being defined and built.
- create an `examples/todo-app` folder with the maslow definition of the app and a simple README.md that instructs a user how to create a new project using the maslow scaffold command and then copying the contents of the example into the new project.
- The app that is created should be a simple web-based TODO app that allows a user to create, read, update, and delete TODO items, view completed items, etc.
  - Backend written in Go
  - Frontend is a SPA written in React with a simple UI
  - Persistence using sqlite3

## Acceptance Criteria

- [ ] All gaps in the maslow spec and harness that would a web app from being defined and built are identified and addressed.
- [ ] An examples area in the repo exists with maslow yaml files for a TODO web app.
- [ ] An agent has used maslow to create a new project (outside of the repo), copied the contents of the example into the new project, and launched a subagent with its own context to build the app using the harness and maslow agent skill (see `00003_AGENT_SKILL.md`)
- [ ] maslow is used to verify the app by building it, running checks, verifying, and auditing.

## Notes
