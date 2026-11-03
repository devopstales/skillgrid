---
type: TechnicalDecision
title: Example Decision
description: Decision with a broken YAML frontmatter block
status: stable
generated: { by: human:paladm, at: 2026-10-18T00:00:00Z
---

# Example Decision

**Deciders:** paladm

## Context and Problem Statement

The frontmatter above has an unclosed flow mapping — the YAML parser must
reject it and the lint must report a frontmatter-invalid finding.

## Decision Outcome

Chosen option: "Keep it broken on purpose", because it is the fixture's
frontmatter-invalid drift seed.

## Consequences

- Good, because it keeps check 1 armed.
- Bad, because it makes no sense as a real decision.
