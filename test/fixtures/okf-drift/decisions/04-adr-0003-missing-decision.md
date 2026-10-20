---
type: TechnicalDecision
title: Missing Decision
description: Decision that exists on disk but has no catalog entry anywhere
status: stable
generated:
  by: human:paladm
  at: 2026-10-18T00:00:00Z
sources:
  - id: fixture-missing
    resource: decisions/index.md
    title: Decisions catalog
---

# Missing Decision

**Deciders:** paladm

## Context and Problem Statement

This file is not listed in the root `index.md` nor in `decisions/index.md` —
the index-complete check must report it in both directions.

## Decision Outcome

Chosen option: "Stay unindexed", because it is the fixture's index drift seed.

## Consequences

- Good, because it keeps check 2 armed.
- Bad, because readers of the catalog never see it.
