---
type: TechnicalDecision
title: Orphan Decision
description: Stable decision missing generated provenance
status: stable
---

# Orphan Decision

**Deciders:** paladm

## Context and Problem Statement

`status: stable` without a `generated` block — the provenance-required check
must report a provenance-missing finding.

## Decision Outcome

Chosen option: "Skip the stamp", because it is the fixture's provenance drift seed.

## Consequences

- Good, because it keeps check 5 armed.
- Bad, because a stable page without provenance is an unattested claim.
