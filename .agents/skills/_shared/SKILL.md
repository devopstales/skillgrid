---
name: shared
description: Shared Skillgrid references consumed by other skills (standards for AGENTS.md, skill-craft rules, measurement convention, threat matrix, ADR records, work-unit commit contract). Not invokable.
disable-model-invocation: true
user-invocable: false
metadata:
  author: devopstales
  version: "2.5"
  part-of: skillgrid
  created_by: human
  version: "2.5"
---

## Purpose

This directory stores shared reference documents consumed by Skillgrid skills.
Do not invoke it as a skill. The project standards live in `rules/` (workflow,
prose style, mnemonic, verification), `craft/` (skill-authoring), `planning/`
(rigor, waivers), `verification/` (the gate's judgment rules), and
`references/` (canonical deep-dive material) — these are the **only** set of
files `AGENTS.md` (via the `## Skillgrid` block) may reference. The
`adr-records/` and `work-unit-commits/` folders are shared
reference bundles that skills load on demand. A new standard that belongs in
`AGENTS.md` goes into the matching folder by domain.

- `rules/` — project standards:
   - **Workflow**
     - [rules/sdd-structure.md](rules/sdd-structure.md) — the Skillgrid workflow: phase order, canonical `.skillgrid/` directory structure, artifact names and paths.
     - [rules/chain-strategy.md](rules/chain-strategy.md) — the subagent chain strategy: when to dispatch a fresh context vs. continue in-session.
   - **Prose & writing style**
     - [rules/prose-style.md](rules/prose-style.md) — the 21-rule prose style (agent-style) the Markdown the agent writes must follow.
   - **Mnemonic memory (MCP usage)**
     - [rules/mnemonic-memory.md](rules/mnemonic-memory.md) — save shape, session protocol, recovery ladder, session summary.
   - **Verification**
     - [rules/verification-ladder.md](rules/verification-ladder.md) — L1–L4 verification levels per change class.
     - [rules/evidence-claims.md](rules/evidence-claims.md) — when a claim counts as evidence: repro-before-closure, named-test closure, baseline discipline, deletion-first fix ranking, executable exits.
   - **Repo mapping**
     - [rules/repo-mapping.md](rules/repo-mapping.md) — the `architecture/` map: page shapes (component vs blueprint), scoped refresh (the execute hook), full-map recipe, correction policy.
     - [rules/scout-contract.md](rules/scout-contract.md) — zero-inference code reading: `[file:symbol]` citations, `None` for absence, ~1,500-line scout chunks, read order, JSON return, link hygiene.
   - **Evals**
     - [rules/eval-assertions.md](rules/eval-assertions.md) — `required_pattern` / `forbidden_pattern` eval assertions.
   - **User gates**
     - [rules/user-gates.md](rules/user-gates.md) — decisions the user makes with information: gate format invariants (one question, no recommendation labels, artifact first), the per-finding disposition contract (CONTEST / ESCALATE / PROPOSE FIX / DECLINE), the architecture-option table (Option / Mechanism / Advantages / Disadvantages / Fails-when), honesty tags (`not run`, `[⚠️ UNVERIFIED]`, `[⏳ ACK]`).
- `planning/` — the Skillgrid workflow standards:
   - [planning/rigor-tiers.md](planning/rigor-tiers.md) — the per-change rigor dial (T0–T3) that sets each gate's floor.
   - [planning/fast-track.md](planning/fast-track.md) — the trivial/small waiver policy (when the pipeline shrinks).
   - [planning/fan-out-gate.md](planning/fan-out-gate.md) — cost-aware approval before parallel fan-out: >15 units, ≤5 concurrent, ~1,500 lines per scout, cost stated in the gate message.
- `verification/` — the gate's judgment rules:
   - [verification/floor.md](verification/floor.md) — multi-part judgments gate on the weakest part, never the average.
- `craft/` — skill-authoring rules:
   - [craft/skill-anatomy.md](craft/skill-anatomy.md) — the canonical `SKILL.md` format: directory shape, frontmatter, body rules, cross-reference rules.
   - [craft/leading-words.md](craft/leading-words.md) — the leading-word lever: the no-op test, pretrain-first, positive-over-negation, the collapse pattern, and the double-anchoring (execution + invocation) rule.
   - [craft/measurement.md](craft/measurement.md) — the `## How to measure it` section contract (one leading + one lagging indicator, each with source and direction).
   - [craft/deterministic-boundary.md](craft/deterministic-boundary.md) — the script/prose split (deterministic logic belongs in a script, not skill prose) and the agentic script contract (JSON stdout, exit 0/1/2, atomic writes, no prompts, tests beside the script).
   - [craft/cite-dont-restate.md](craft/cite-dont-restate.md) — cite decisions and ADRs by identifier; never restate their wording.
   - [craft/effort-budgets.md](craft/effort-budgets.md) — the `effort:` frontmatter budget signal (low / standard / max) and the honest-overhead "when NOT to use the pipeline" rule.
   - [craft/review-discipline.md](craft/review-discipline.md) — the review loop: blindness, verdicts, the fix loop, and the ❌/✅ paired-example format for steering by contrast.
   - [craft/review-ledger.md](craft/review-ledger.md) — the per-change review ledger: findings, dispositions, lineage, the frozen-intent contract.
   - [craft/deferred-work.md](craft/deferred-work.md) — the cross-change deferred-work ledger: the `DW-<n>` entries the review triage "defer" sink writes, the sweep (verify against code → resolved / bundle / blocked / skip / decision), staleness.
- `references/` — canonical single-source references:
   - [references/threat-matrix.md](references/threat-matrix.md) — the applicability-driven threat matrix.
- `scripts/` — shared deterministic helpers (JSON stdout, exit 0/1/2, tests beside):
   - [scripts/map-scope.mjs](scripts/map-scope.mjs) — repo-mapping scope: BFS discovery, prune, manifest diff, partitioning, gate verdict. `node map-scope.mjs --base <sha> --mode refresh|full` / `--merge <returns.json>`.
- `adr-records/` — ADR template family (load the matching format when authoring a decision record):
   - [adr-records/README.md](adr-records/README.md) — which template to pick, and the shared conventions (numbering, supersession).
   - [adr-records/nygard.md](adr-records/nygard.md) — the default Context / Decision / Consequences form.
   - [adr-records/y-statement.md](adr-records/y-statement.md) — the one-line Y-statement form for small decisions.
   - [adr-records/madr-minimal.md](adr-records/madr-minimal.md) + [madr-full.md](adr-records/madr-full.md) — the MADR minimal and full forms.
   - [adr-records/custom.md](adr-records/custom.md) — escape hatch for a project-specific form.
- `work-unit-commits/` — the commit contract the execution and ship skills enforce:
   - [work-unit-commits/README.md](work-unit-commits/README.md) — conventional-commit format, atomic sizing, when-to-commit, the `[skillgrid-context]` block, the git-hook guards.
   - [work-unit-commits/state-schema.md](work-unit-commits/state-schema.md) — the execution-state schema and the resume decision tree.
- `templates/` — fill-in skeletons:
   - [templates/skill-template.md](templates/skill-template.md) — the canonical `SKILL.md` skeleton to copy for any new skill (see craft/skill-anatomy.md).
