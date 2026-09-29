---
name: _shared
description: Shared Skillgrid references consumed by other skills (conventions, threat matrix, TDD cycle, agent-config block). Not invokable.
disable-model-invocation: true
user-invocable: false
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

## Purpose

This directory stores shared reference documents consumed by Skillgrid skills. Do not invoke it as a skill.

- `rules/` — shared contract documents every skill must honor:
   - [rules/skill-anatomy.md](rules/skill-anatomy.md) — canonical SKILL.md format: section order, frontmatter, writing principles, line budget.
   - [rules/sdd-structure.md](rules/sdd-structure.md) — canonical directory layout, artifact paths, phase order.
   - [rules/rigor-tiers.md](rules/rigor-tiers.md) — the per-change rigor dial (T0–T3) that sets each gate's floor.
  - [rules/fast-track.md](rules/fast-track.md) — trivial/small waiver policy.
  - [rules/mnemonic-memory.md](rules/mnemonic-memory.md) — save shape, session protocol, topic-key rules.
  - [rules/commits.md](rules/commits.md) — commit message contract.
  - [rules/verification-ladder.md](rules/verification-ladder.md) — L1-L4 verification levels per change class.
   - [rules/hybrid-degradation.md](rules/hybrid-degradation.md) — dual-write contract + degradation line.
    - [rules/deterministic-boundary.md](rules/deterministic-boundary.md) — the script/prose split: deterministic logic belongs in a script, not skill prose.
     - [rules/code-standards.md](rules/code-standards.md) — global, language-agnostic code standards: design principles, deep modules (module/interface/depth/seam/adapter/leverage/locality), structure, naming, errors, concurrency, testing, and the two-axis review contract + smell baseline.
- `references/` — canonical single-source references:
  - [references/threat-matrix.md](references/threat-matrix.md) — applicability-driven threat matrix.
  - [references/strict-tdd.md](references/strict-tdd.md) — RED → GREEN → TRIANGULATE → REFACTOR cycle.
- `agent-config/` — agent config block family:
  - [agent-config/block.md](agent-config/block.md) — the canonical `## Skillgrid` payload + idempotent upsert sentinels.
- `templates/` — fill-in skeletons:
  - [templates/skill-template.md](templates/skill-template.md) — the canonical `SKILL.md` skeleton to copy for any new skill (see rules/skill-anatomy.md).
