---
id: 5
title: Add support for plugins in maslow
status: draft
created: 2026-02-19
tags: [maslow, verification, plugins]
---

- Plugins (Otel, wasp ZAP, etc) that can be used to verify local and remote. For instance, Maslow could be run periodically as a probe looking at Otel information to see if performance in production is meeting expectations. If not, an agent can go fix that!
  - Modeled like terraform? e.g. an executable called `maslow-<plugin>` and they communicate over STDIN/STDOUT?

notifications via slack etc?

maybe need for `maslow doctor` to ensure plugins are installed and working?
