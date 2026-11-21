# Goldfish Review — 5-Stage Pipeline (Track B)

The validation-seam dispatch protocol for blueprint review. Stages iterate
under the sharpening loop in `goldfish-review-loop.md`; finding resolution
runs per `goldfish-resolve.md`; harness capability and the freshness STOP
rule live in `harnesses.md`. Persona briefs live in
`.agents/skills/_shared/elephant-goldfish/` (the 7 review briefs:
`zoom-out.md`, `comprehension.md`, `design-critic.md`, `structural.md`,
`style.md`, `implementation-readiness.md`, `codebase-details.md`).

## The five stages (R7)

One line per stage. The stage list is NOT a single combined prompt — each
stage is a separate dispatch round against fresh subagents.

- **Stage 5 — Initial Zoom Out** (target: full `blueprint.md`; YAGNI,
  reuse, anti-weeds, altitude → dispatch `zoom-out.md` →
  `zoom_out_report.md`)
- **Stage 6 — Comprehension Test** (target: `blueprint.md` + referenced
  files; paraphrase first, self-containment → dispatch `comprehension.md`
  → `comprehension_report.md`)
- **Stage 7 — Critic Review** (target: `blueprint.md`, the *how*; dispatch
  `design-critic.md` + `structural.md` + `style.md` — three fresh
  subagents, one per brief, findings merged → `critic_report.md`)
- **Stage 8 — Intermediate Zoom Out** (target: full `blueprint.md`,
  re-read after Stages 6+7 fixes; catches altitude drift + scope creep
  introduced by fixes — re-dispatch `zoom-out.md` as a *fresh* subagent,
  not the Stage 5 one → `intermediate_zoom_out_report.md`)
- **Stage 9 — Implementation Readiness** (target: `blueprint.md` task
  steps + codebase; actionability, codebase accuracy → dispatch
  `implementation-readiness.md` + `codebase-details.md` + `style.md` →
  `implementation_readiness_report.md`)

After each stage's resolution the main agent folds findings inline — no
synthesizer persona. The Consolidated preset's cross-stage dedup +
traceability matrix is also main-agent work. R11 stage mapping:
`style.md` is used by Stages 7 and 9 — not folded away.

## Presets (single-select, R8)

The four presets, selected by the user or inferred from the invocation
context. If not provided at invocation: a single-select gate. Multi-select
is prohibited. The topological dependency 5→6→7→8→9 is preserved under
every preset.

- **Interactive** — checkpoint after each stage: full pipeline with user
  checkpoint gates between stages. Default for New Project / New Function
  blueprints.
- **Consolidated** — no pauses: full pipeline, findings consolidated into
  `consolidated_review_report.md` with cross-stage dedup + traceability
  matrix.
- **Feature-spec-only** — Stages 5-8, skip 9.
- **Implementation-only** — Stage 9 only.

## Dispatch protocol (R7/R36/R37)

Freshness check first: R36's STOP rule applies before Stage 5 is
dispatched (verbatim rule in `harnesses.md` — a same-context run MUST NOT
count toward R7).

Each stage dispatches fresh `reviewer` subagents (fallback `general`),
one per brief:

- prompt = persona brief path + target path + explicit reading list (the
  files the target references) + the R37 four-part isolation contract
  re-asserted verbatim + `Mode: REVIEW`
- subagents never see the author conversation, project instructions, or
  the network

Error handling: a reviewer that times out / returns no report → retry
once → record `not run` (never zero findings); a stage whose reviewer is
`not run` caps that stage's verdict at "not verified," not "clean."

Every stage report's header carries the one-line harness-isolation
attestation (which of the four parts the harness enforced vs. could not; a
harness that cannot enforce part 3 says so — no silent pass).

## Record boundary (R7)

Findings carry reviewer vocabulary (Blocker / Major / Nit) in transit. At
the ledger round entry (per `_shared/craft/review-ledger.md`) they are
normalized to CRITICAL / WARNING / SUGGESTION. This is the single
normalization point.

## Committed report artifacts (R13)

Each stage writes its findings to `.skillgrid/change-specs/<topic>/review/`:

- `zoom_out_report.md` (Stage 5)
- `comprehension_report.md` (Stage 6)
- `critic_report.md` (Stage 7)
- `intermediate_zoom_out_report.md` (Stage 8)
- `implementation_readiness_report.md` (Stage 9)

The five stage files are committed, not chat-only. The Consolidated
preset additionally writes `consolidated_review_report.md`.

The resolution record is the `review-ledger.md` round entry (per-finding
AI suggestion + justification + proposed edit in the ledger's
Dispositions, with finding IDs, dispositions, fix-commit lineage — no
separate goldfish report file for resolutions), per
`_shared/craft/review-ledger.md`.

The blueprint's `## Review Summary` section carries the condensed
verdict. Parked findings flow to qa via the debt-surfacing step
(`qa/SKILL.md` → Step 9.9: Surface review debt).

**Consolidated chat response (R13):** after any run — all presets — the
chat turn ends with the stage/reviewer status table, the severity
breakdown in normalized vocabulary, every CRITICAL/WARNING finding with
its proposed disposition, and the report path + ledger entry. A turn that
ends with only a file path does not satisfy this requirement.

Discipline: per `_shared/craft/review-discipline.md`.
