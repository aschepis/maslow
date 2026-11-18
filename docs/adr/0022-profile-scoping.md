# ADR 0022: Profiles Scope Contracts and Budgets

- **Status**: Accepted
- **Date**: 2026-11-18

## Context

Profiles filtered checks, but every contract, budget, ref and policy rule ran on every profile, so `quick` ran the performance budget and every contract. A pipeline needs a fast profile and a full one that differ in more than their checks.

## Decision

A profile may list `contracts` and `budgets` by name or tag. An omitted list selects all of them and `[]` selects none, refs and policy run unless the profile sets `refs: false` or `policy: false`, and a selector that matches nothing is an error.

## Consequences

- Existing specs behave exactly as before, because omitted selectors mean "all".
- Contracts and budgets gain optional `tags`, matching checks.
- Turning policy off is explicit and visible in the spec, so a reviewer can see when a profile stops being a gate.
