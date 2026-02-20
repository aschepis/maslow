---
id: 10
title: Add database assertion support for contracts
status: draft
created: 2026-02-19
tags: [maslow, verification, contract, database]
---

## Objective

Add a `sql` action type to contracts so Maslow can verify database state before and after operations.

## Requirements

- New step action type `sql` with fields: driver (postgres, mysql, sqlite), dsn, query
- Expectation support: row_count, json_path on result rows, value matching
- Support setup/teardown queries (e.g. seed data before contract, clean up after)
- Connection string should support environment variable interpolation

## Notes

- Useful for verifying data migrations, CRUD correctness, and state invariants
- Pair with HTTP actions: make an HTTP call, then verify database state
- Security: DSN must never be committed; use env vars or a .env file (which maslow.yaml policy should deny)
