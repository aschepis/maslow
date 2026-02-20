---
id: 4
title: Add support for agent work handoff
status: draft
created: 2026-02-19
tags: [agents, harness]
---

NOTE: SIMPLE HUMAN PROTOCOL WORKAROUND FOR THIS IS TO JUST SET THE TASK STATUS TO `todo` and let a new agent pick it up from scratch.

for instance, if an agent crashes or shuts down and the work isn't done, how does
another agent know that it is there to be picked up and completed? Can it get
the work from the previous agent in a branch? Does it have to start over?
