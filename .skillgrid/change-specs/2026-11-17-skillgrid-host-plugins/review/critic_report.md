# Stage 7 — Critic Review (Round 1)

**Reviewers:** three fresh `flint` subagents (design-critic, structural, style) — same harness deviation as stages 5/6 (`reviewer` backend 429)
**Personas:** `.agents/skills/_shared/elephant-goldfish/{design-critic,structural,style}.md`
**Target:** blueprint.md (post-stage-6 state)

Isolation attestation (all three): fresh context, read-only, no project
instructions, no network/MCP — all four enforced by the dispatch contract.

## Findings (reviewer outputs, condensed verbatim)

### design-critic

| ID | Severity | Scope | Finding |
|---|---|---|---|
| F-1 | Major | §2 omp row / §1 exception / Must-Haves R3,R4 | stage-5 fold-in left no evaluation record of what the shipped hook already guarantees; the subagent guard is not placed on the changed full-content path — R3/R4 asserted but unanchored for omp |
| F-2 | Major | §3 / §6 | installer design written against the in-flight ADR-0067 rework with a conditional ("if the ADR-0067 change re-creates the trees…") — no fixed sequencing/conflict decision |
| F-3 | Minor | Hypothesis / §5 smoke | the positive smoke oracle is an either/or (announce OR first-read); injection-present-before-first-action is not a separate observable; the hypothesis's two clauses are collapsed |
| F-4 | Minor | Classification / §5 | L3 (build+coverage+lint) not operationalized for the new TS/JS/bash artifacts — only `go test` named |
| F-5 | Nit | §6 | kilo risk conflates skill-list visibility (`disable-model-invocation`) with bootstrap injection; the named contingency addresses list handling, not injection |

### structural

| ID | Severity | Scope | Finding |
|---|---|---|---|
| F-S7-1 | Major | whole doc | template conformance: no per-task Files/Interfaces/SATISFIES/Steps substrate, no `## Review Summary`; wave-based §7 preview instead of auditable task ordering (template + repo precedent `2026-11-17-mnemonic-v2-host-memory-plugins/blueprint.md` both mandate the task structure) |
| F-S7-2 | Minor | Findings / Decision notes | cross-refs (`review/zoom_out_report.md`, `review-ledger.md`, `state.yaml` journal id) not resolvable from document + reading list |
| F-S7-3 | Minor | §7 | wave-based granularity substitutes for task ordering (same root as F-S7-1) |
| F-S7-4 | Nit | header / §1 / §5 | placeholder path syntax (`{pi,opencode,…}-tools.md`, `/tmp/…/{…}.md`) not literal |

### style

| ID | Severity | Scope | Finding |
|---|---|---|---|
| F-STYLE-1 | Minor | Must-Haves intro | contraction "doesn't" (RULE-I) |
| F-STYLE-2 | Nit | header lines 11/15/22/24/26 | em-dash/semicolon density as casual clause punctuation (RULE-B) |

## Resolutions (all PROPOSE-FIX, folded this stage)

- **F-1 → fixed:** Design contract now carries an explicit *evaluation record* for the shipped hook (guaranteed: full-content + fail-open; not guaranteed: project condition, child-session skip). The R3 truth names the omp probe host; Task 2 Step 5 is the probe with a conditional child-guard.
- **F-2 → fixed:** Task 6 carries the fixed sequencing decision: standalone `routerPluginRel` constant (NOT folded into `nativePluginBases`) + idempotent tree creation (create-if-missing) — if ADR-0067 lands first, Task 6's diff shrinks, delivered files identical. No conditional remains.
- **F-3 → fixed:** Hypothesis claim/right-condition now assert both clauses separately; Task 7 asserts (a) injection present in first-turn context (JSONL check) AND (b) announce-or-first-read, per host, positive + negative.
- **F-4 → fixed:** Global Constraints gains an operationalized L3 gate line (`go build`, `go test`, coverage, `gofmt -l .`; `bash -n`, `node --check`, the Node test seam for TS; no TS linter configured — the Node run is the TS gate).
- **F-5 → fixed:** kilo risk re-scoped in Task 5 (visibility vs injection; the `skills.paths` hook + slash command are the visibility-side belt).
- **F-S7-1 / F-S7-3 → fixed (the round's largest fix):** the blueprint tail was restructured to the template: `## File Structure` (responsibility list), `## Tasks` with Tasks 1–8 (each with Files / Interfaces (Consumes/Produces/Seam/Deletion test/Adapters) / SATISFIES / checkbox Steps), `## Review Summary` (final section, filled at round end). The old §1–§7 were re-homed: design content → Design contract + Per-host mechanism (kept as shared top-level sections, per the memory-plugins blueprint's own extra-section precedent); installer changes → Task 6; test plan → task steps; risks → the tasks they concern; the slicing preview → the Tasks preamble (wave grouping).
- **F-S7-2 → fixed (partial):** Findings line + Decision notes now name the change-folder / `.skillgrid/meta/state.yaml` locations explicitly. The ADR/ledger references are spec-zone convention (files exist in the repo); not a defect.
- **F-S7-4 → fixed:** R6 + the Design contract load-hint now name the literal files that exist (`pi-tools.md`, `opencode-tools.md`); kilo/cursor/omp have no tools file (plain load). The `/tmp/skillgrid-plugin-research/…` brace list in the Spec line is a transient-session path (documented as such) — left as-is.
- **F-STYLE-1 → fixed:** "does not". **F-STYLE-2 → fixed (partial):** the flagged header lines re-pointed (colons/periods); the remaining em-dashes are the document's established register in table cells and the Per-host table (declined kernel, recorded).
