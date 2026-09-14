---
name: _shared
description: Shared Skillgrid references consumed by other skills (conventions, threat matrix, TDD cycle, agent-config block). Not invokable.
disable-model-invocation: true
user-invocable: false
---

## Purpose

This directory stores shared reference documents consumed by Skillgrid skills. Do not invoke it as a skill.

- `conventions/` — shared contract documents every skill must honor:
  - [conventions/sdd-structure.md](conventions/sdd-structure.md) — canonical directory layout, artifact paths, phase order.
  - [conventions/fast-track.md](conventions/fast-track.md) — trivial/small waiver policy.
  - [conventions/mnemonic-memory.md](conventions/mnemonic-memory.md) — save shape, session protocol, topic-key rules.
  - [conventions/commits.md](conventions/commits.md) — commit message contract.
  - [conventions/verification-ladder.md](conventions/verification-ladder.md) — L1-L4 verification levels per change class.
  - [conventions/hybrid-degradation.md](conventions/hybrid-degradation.md) — dual-write contract + degradation line.
- `references/` — canonical single-source references:
  - [references/threat-matrix.md](references/threat-matrix.md) — applicability-driven threat matrix.
  - [references/strict-tdd.md](references/strict-tdd.md) — RED → GREEN → TRIANGULATE → REFACTOR cycle.
- `agent-config/` — agent config block family:
  - [agent-config/block.md](agent-config/block.md) — the canonical `## Skillgrid` payload + idempotent upsert sentinels.
