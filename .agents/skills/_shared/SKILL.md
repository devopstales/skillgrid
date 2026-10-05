---
name: _shared
description: Shared Skillgrid references consumed by other skills (standards for AGENTS.md, skill-craft rules, mnemonic protocol, threat matrix, TDD cycle, agent-config block). Not invokable.
disable-model-invocation: true
user-invocable: false
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

## Purpose

This directory stores shared reference documents consumed by Skillgrid skills. Do not invoke it as a skill. The project standards live in `rules/` (workflow, coding, testing, Mnemonic, ticketing), `planning/` (rigor, waivers), `craft/` (skill-authoring), `verification/` (the gate's judgment rules), and `knowledge/` (hybrid degradation contract) — these are the **only** set of files `AGENTS.md` (via the `## Skillgrid` block) may reference. `references/` holds canonical deep-dive material skills load on demand. A new standard that belongs in `AGENTS.md` goes into the matching folder by domain.

- `rules/` — project standards:
   - **Workflow**
     - [rules/sdd-structure.md](rules/sdd-structure.md) — the Skillgrid workflow: phase order, canonical `.skillgrid/` directory structure, artifact names and paths.
   - **Coding & commits**
     - [rules/code-standards.md](rules/code-standards.md) — global, language-agnostic code standards: design principles, deep modules, structure, naming, errors, concurrency, and the review contract.
     - [rules/commits.md](rules/commits.md) — commit message contract.
   - **Testing & verification**
     - [rules/testing-standards.md](rules/testing-standards.md) — test layers, duplication/dead-code detection, security scanning (Trivy, Wapiti, Nuclei, AKCA), and the toolchain per change class.
     - [rules/verification-ladder.md](rules/verification-ladder.md) — L1-L4 verification levels per change class.
     - [rules/verification-scope.md](rules/verification-scope.md) — a verification result carries its scope; a zero count is never a bare zero.
   - **Mnemonic memory (MCP usage)**
     - [rules/mnemonic-memory.md](rules/mnemonic-memory.md) — save shape, session protocol, recovery ladder, six-section session summary.
     - [rules/mnemonic-artifacts.md](rules/mnemonic-artifacts.md) — deterministic artifact naming, artifact-type table, upserts, compaction/recovery protocol.
     - [rules/mnemonic-code-indexing.md](rules/mnemonic-code-indexing.md) — `code_*` index ladder and search router (MCP + CLI).
- `planning/` — the Skillgrid workflow standards:
   - [planning/rigor-tiers.md](planning/rigor-tiers.md) — the per-change rigor dial (T0–T3) that sets each gate's floor.
   - [planning/fast-track.md](planning/fast-track.md) — trivial/small waiver policy (when the pipeline shrinks).
- `verification/` — the gate's judgment rules:
   - [verification/floor.md](verification/floor.md) — multi-part judgments gate on the weakest part, never the average.
   - [verification/calibration.md](verification/calibration.md) — anti-inflation for self-scores and self-judgment.
- `craft/` — skill-authoring rules:
   - [craft/skill-anatomy.md](craft/skill-anatomy.md) — canonical SKILL.md format: section order, frontmatter, writing principles, line budget.
   - [craft/deterministic-boundary.md](craft/deterministic-boundary.md) — the script/prose split: deterministic logic belongs in a script, not skill prose.
   - [craft/measurement.md](craft/measurement.md) — the `## How to measure it` section (one leading + one lagging indicator, each with source and direction).
   - [craft/cite-dont-restate.md](craft/cite-dont-restate.md) — cite decisions and ADRs by identifier; never restate their wording.
   - [craft/effort-budgets.md](craft/effort-budgets.md) — the `effort:` frontmatter signal and context-spend budgets.
- `knowledge/` — persistence contracts:
   - [knowledge/hybrid-degradation.md](knowledge/hybrid-degradation.md) — the Mnemonic dual-write contract + degradation line.
- `rules/ticketing/` — per-tracker CLI + formatting standards:
   - [rules/ticketing/backlogmd.md](rules/ticketing/backlogmd.md) + [backlogmd-formatting.md](rules/ticketing/backlogmd-formatting.md)
   - [rules/ticketing/github.md](rules/ticketing/github.md) + [github-formatting.md](rules/ticketing/github-formatting.md)
   - [rules/ticketing/gitlab.md](rules/ticketing/gitlab.md) + [gitlab-formatting.md](rules/ticketing/gitlab-formatting.md)
   - [rules/ticketing/jira.md](rules/ticketing/jira.md) + [jira-formatting.md](rules/ticketing/jira-formatting.md)
- `references/` — canonical single-source references:
   - [references/threat-matrix.md](references/threat-matrix.md) — applicability-driven threat matrix.
   - [references/strict-tdd.md](references/strict-tdd.md) — RED → GREEN → TRIANGULATE → REFACTOR cycle.
- `agent-config/` — agent config block family:
   - [agent-config/block.md](agent-config/block.md) — the canonical `## Skillgrid` payload + idempotent upsert sentinels.
- `templates/` — fill-in skeletons:
   - [templates/skill-template.md](templates/skill-template.md) — the canonical `SKILL.md` skeleton to copy for any new skill (see craft/skill-anatomy.md).
