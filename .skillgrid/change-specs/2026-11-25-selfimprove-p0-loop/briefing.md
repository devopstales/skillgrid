# Self-Improving Loop — Phase 0: Close the loop with what exists — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-25)
> **Author:** devopstales
> **Intent frozen:** no (pending) ← the agent sets this to "yes (YYYY-MM-DD)" once the STATUS banner above shows `approved` (the user's hand-edit IS the approval — the agent never writes `approved` itself); after that, the frozen block below is read-only for agent steps

> **Phase 0 of the self-improving-skills plan** (Phases 0–4 are stacked changes; this is the
> first and smallest). No new binary, no new daemon, no new schema — only the improvement
> **inbox**, a **problem-report** format, two routing edits to `ship` (Reflect + Environment
> Retro — including a **new step that asks the user for their experiences**), and a
> lifecycle-doc drift fix. The loop becomes closed: a failure observed in a session reaches a
> destination where it can be turned into an approved skill edit.

## Problem / Intent

When a skillgrid session ends, `ship` Step 2 Reflect triages every Lesson/Surprise and says a
*process bug* should be "route[d] back to the governing skill or `_shared` rule as a follow-up
item" — **there is no destination**: the follow-up dies in `report.md`. Step 3 Environment
Retro likewise ends in "propose, don't implement" with proposals that go nowhere. The library
already has the *producers* of improvement evidence (reflect, env retro, projectwiki gotchas,
deferred-work) and the *verification* machinery (skill-creator TDD gate, evals, 10-rule audit,
frontmatter/size checks) — but the two halves are not connected, so the library cannot edit
itself from its own usage.

**Job to be done:** when a change ships and its retro found a reusable skill-level failure,
the team wants that failure captured as a reviewable, approvable **proposal** against a named
skill, so that a later (human- or agent-run) curation pass can turn it into a tested skill edit
instead of repeating the same diagnosis.

Why now: `docs/04-lifecycle.md` already specifies the curation transitions and the
`.usage.json` sidecar format, and `2026-11-24-brainstorm-skill-improvements/findings.md` shows
the team actively porting external skill-design ideas — the missing *loop*, not more content,
is the constraint.

## Purpose & Success Criteria

- **Purpose:** give the self-improvement loop its missing sink — a versioned improvement
  inbox that `ship` writes into and that a later curation pass (Phases 1–2) consumes — and add
  the **user-experience** signal to reflect (what the user experienced, in their words).
- **Success criteria:**
  1. After any change's `ship`, every process-bug Lesson/Surprise and every env-retro finding
     has a corresponding proposal file in `.skillgrid/skill-improvements/` (or an explicit
     "one-off — stays in report" record for the ones that don't qualify).
  2. The reflect step asks the user for their experiences of the change; the user's answers are
     captured as *sourced, attributed* evidence in the retro half, and any user correction is
     recorded as a proposal with `signal_class: correction` (first-class, outranking agent
     self-assessment). If the user declines, the record says so.
  3. A proposal file validates against the format contract (required fields present;
     `original_snippet` appears exactly once in the target skill file; `evidence` has ≥1
     file:line/commit/quote).
  4. `docs/04-lifecycle.md` and `docs/02-pipeline.md` name the inbox; the broken pointer to
     `skill-creator/references/skill-lifecycle.md` is fixed.
  5. The loop is demonstrated end-to-end on this repo: one real proposal is filed during
     Phase 0's own ship and tracked to a disposition (applied / rejected / deferred) — the
     self-hosting proof.
- **Out of scope:** anything that *applies* proposals automatically (Phase 2); the `.usage.json`
  writer hook and eval/scenario runner (Phase 1); projectwiki counters/tiers and the
  user-project write path (Phase 3); outcome metrics (Phase 4); a curation *daemon* (explicitly
  never — the loop is a scheduled, human-triggered pass per `docs/04-lifecycle.md`); new skills
  beyond the two edits to existing skill bodies; changing `ship`'s phase order or gate structure.

## Context

- **Producers of evidence (all exist, all current):**
  - `ship/SKILL.md` Step 2 Reflect — "Triage every Lesson and Surprise: process bug vs
    one-off … route it back to the governing skill or `_shared` rule as a follow-up item"
    (the sentence this change closes); Step 3 Environment Retro — six-category sweep,
    "Propose, don't implement — real builds become `mem_save` findings + offered follow-up
    items, not inline edits."
  - `docs/04-lifecycle.md` — provenance (`created_by` human/agent), `.usage.json` sidecar
    format, curation transitions (90d stale / 180d archive, never-delete, backup-before-
    archive, pin exemption), curation interval procedure.
  - `_shared/craft/deferred-work.md` + `.skillgrid/deferred-work.md` — the existing
    cross-change ledger pattern (append-only `DW-<n>` entries with source/why/re-entry) —
    the closest in-repo analogue to the proposed inbox; the inbox borrows its discipline
    (monotonic ids, source round, re-entry path) and its sweep moment (ship).
  - `_shared/rules/sdd-structure.md` — spec-zone layout, STATUS banners, the Status Contract,
    and the "this file wins" rule for any layout question.
  - `skill-creator/SKILL.md` — Scope Bar (observed failure + composes-or-new), TDD gate
    (RED before edit / GREEN after), "match the form to the failure", 10-rule audit.
- **The drift being fixed:** `docs/04-lifecycle.md` opens with "The full specification lives
  in `skill-creator/references/skill-lifecycle.md`" — that file **does not exist**. The spec
  actually lives in `skill-creator/references/anatomy-and-checklist.md` (§ Usage telemetry /
  Curation transitions) and in the doc itself.
- **Upstream patterns this phase adapts (source study 2026-11-25):**
  - BerriAI/self-improving-agent — proposal = minimal diff + `originalSnippet` that must
    appear **exactly once** (machine-verifiable edit); apply gated on explicit user
    confirmation in the most recent message.
  - Perplexity Brain (self-improving skills) — the four-stage loop (Evidence → Reflection →
    Verification → Activation); "short, structured diagnostic" when the agent sees
    frustration or broken behavior.
  - viktor.com (self-improving skills) — signal classes with different bars; "require a
    verifiable signal before any lesson becomes a rule"; review gate on every write.
- **Repo state:** 19 skills + `_shared` under `.agents/skills/`; no `.usage.json` on disk; no
  eval/scenario runner; `plugins/omp/` hosts the `pi-worktree` and `skillgrid-guards` plugins
  (the pattern Phases 1+ extend — not touched here).

## User Context

- **Who:** the maintainer(s) of this skills library (docs-first; the deliverable is markdown)
  and, downstream, any project that installs skillgrid.
- **Direct evidence:** `ship` retro halves in `change-archive/` already contain process-bug
  Lessons with nowhere to land (e.g. the pi-worktree review rounds' deferred findings — the
  `DW-<n>` ledger exists precisely because review findings needed a sink); the user explicitly
  asked that reflect **ask them about their experiences** — today reflect is fully
  agent-side (git + ledgers), and the user's subjective signal (what felt wrong, what they'd
  correct) never enters the record unless they interrupt.
- **Why it fails today:** the user's corrections are the highest-quality evidence available
  (viktor signal class 1), and they currently evaporate into chat; the agent's self-assessment
  is the only captured signal, and the research the user cited (Shao et al. via viktor) warns
  self-assessed lessons can be causally wrong — a user quote is the antidote.

## Pre-mortem

- **Desirability:** the inbox becomes a second `deferred-work.md` — findings filed, never
  dispositioned. Countermeasure: Phase 2's curation pass *reads* the inbox, and the sweep at
  every ship reports the open-proposal count (a number the user sees every close-out).
- **Feasibility:** `ship` is a max-effort terminal skill already carrying six steps; adding a
  user question + file writes risks bloating Step 2 beyond its size budget. Countermeasure:
  the edit is deliberately small (~25 lines incl. the ask), the user-experience ask is one
  question with one short answer, and the proposal file is a copy-forward (the evidence is
  already in the retro half — the proposal is a pointer + snippet, not a restatement).
- **Viability:** if proposals never become edits, the loop is decoration. Countermeasure:
  success criterion 5 requires Phase 0's own ship to file at least one real proposal and
  track it to a disposition; Phase 2 is the change whose sole job is applying them.

## Timebox

Target: this phase ships within the current week (2026-11-25 → 2026-12-02). If exceeded:
shrink to the inbox + format + reflect routing (R1–R4); defer the env-retro routing (R5) and
the docs updates (R7) to a follow-up commit of the same change. The drift fix (R6) never
slips — it is one line.

## Approaches Considered

| Approach | Key trade-off | Verdict |
|----------|---------------|---------|
| Dedicated `.skillgrid/skill-improvements/` inbox, one file per proposal, written by `ship` | New directory in the spec zone; must be added to the sdd-structure layout | **Chosen** |
| Reuse `.skillgrid/deferred-work.md` with a new entry type | Zero new files, but that ledger is *change-scoped debt* (re-entry = verify against code); proposals are *skill-scoped edits* (re-entry = apply + test) — mixing them muddies both sweeps | Rejected |
| Per-skill `memory.md` lesson logs (Peter Yang pattern) | Right shape for *user-project* mode (Phase 3); wrong here — the library repo needs reviewable diffs, not append logs | Deferred to Phase 3 |
| Let the curation pass invent the destination | No dead-letter today, but Phase 0's point is that the loop works *before* the pass exists | Rejected |

- **Chosen:** dedicated inbox + format contract + `ship` routing edits + drift fix.
- **Rejected by author:** extending `deferred-work.md` (scope mismatch, named above).

## Requirements

1. **R1 — Improvement inbox with format contract:** `.skillgrid/skill-improvements/` exists
   (committed, with a `README.md` stating the contract). One file per proposal:
   `PROP-<n>.md` (monotonic, project-wide, same id discipline as `DW-<n>`). Required fields:
   `id`, `date`, `target_skill` (a path under `.agents/skills/`), `signal_class`
   (`correction` | `test` | `eval` | `observation`), `evidence` (≥1 file:line / commit /
   ticket / verbatim user quote), `original_snippet` (the exact text in the target file the
   edit will anchor to), `proposed_delta` (the edit, expressed as a replacement of that
   snippet — an edit, not a rewrite), `recurrence` (count of independent observed instances),
   `risk` (one line), `status` (`proposed` | `approved` | `applied` | `rejected` |
   `deferred`), `disposition_note`, `backup_path` (filled on apply, not before).
   - **Current:** no such directory or format exists anywhere in the repo.
   - **Target:** directory + `README.md` contract + a `scripts/`-style validation check for
     the format (see R4's acceptance for the snippet rule).
   - **Acceptance:** a well-formed `PROP-001.md` passes the validation check; a proposal
     missing `evidence`, or with an empty `original_snippet`, fails with the field named.
   - **Acceptance scenario:** `happy path well-formed-proposal-validates` → `acceptance.feature`
2. **R2 — Problem-report format:** a ~10-line structured format any skill may file when it
   (or the user) hits a skill failure: `skill`, `what_happened`, `expected`, `signal`
   (the verifiable signal: RED test/eval output, command error, or user quote), `severity`,
   `change` (the change id it occurred under). Filed in the change folder
   (`problem-report-<n>.md` beside `report.md`) — the in-flight sink; `ship` triages them
   like any other learning.
   - **Current:** failures are described in prose in `report.md` / review notes; no
     machine-triagable shape.
   - **Target:** the format documented in `_shared/craft/` (one small new reference, one
     pointer from the skills that file it) and used for at least the Phase 0 change itself.
   - **Acceptance:** a problem report missing a verifiable `signal` is rejected by its own
     rules ("a lesson with no verifiable signal stays a note, not a report"); the Phase 0
     change folder contains ≥1 real problem report or an explicit "none observed" line.
   - **Acceptance scenario:** `happy path problem-report-needs-verifiable-signal` → `acceptance.feature`
3. **R3 — Reflect routes process bugs to the inbox:** `ship` Step 2 Reflect's triage paragraph
   is edited so that a *process bug* that names a skill or `_shared` rule is **filed as a
   proposal** (R1 format) at ship time, not merely named as a follow-up; a *one-off* records
   "one-off — stays in report" in the retro half. The retro half lists the filed proposal ids.
   - **Current:** the triage says "route it back … as a follow-up item" with no destination.
   - **Target:** the edited Step 2 text (provided in the proposed edits below) + the routing
     executed in Phase 0's own ship.
   - **Acceptance:** after Phase 0 ships, `report.md`'s retro half names ≥0 proposal ids and
     every process-bug line in that retro half either has a matching `PROP-<n>` file or an
     explicit one-off record (checked by reading the two artifacts together — a mechanical
     cross-check script is Phase 1's job).
   - **Acceptance scenario:** `happy path process-bug-becomes-proposal` → `acceptance.feature`
4. **R4 — Reflect asks the user for their experiences (NEW):** `ship` Step 2 Reflect gains a
   sub-step — placed after the Lesson/Surprise triage, before the Acceptance verdict — that
   **asks the user, in the user's language, what they experienced** on this change: what
   worked for them, what didn't, and what they'd want the agent to do differently next time.
   Rules: one question, short answer; the user's words are captured **verbatim, attributed**
   ("user, YYYY-MM-DD") into the retro half as a `## User Experience` block; any user
   correction is filed as a proposal with `signal_class: correction` (corrections outrank
   agent self-assessment as evidence); if the user declines or there is genuinely nothing to
   capture, the retro half records `User experience: none captured (asked YYYY-MM-DD)` —
   the ask itself is the record. The block is *additive*: it does not change the
   final-state authority, the sourced-learnings rule, or the acceptance-verdict mechanics.
   - **Current:** reflect is fully agent-side (git, ledgers, report.md); the user's
     experience enters only by interruption.
   - **Target:** the edited Step 2 (proposed text below) + the `## User Experience` block in
     the retro half of every subsequent change (including Phase 0's own report).
   - **Acceptance:** Phase 0's own `report.md` contains a `## User Experience` block that is
     either a verbatim attributed user answer or the "none captured (asked …)" record — no
     third state exists.
   - **Acceptance scenario:** `happy path reflect-asks-user-experience` → `acceptance.feature`
5. **R5 — Environment retro targets the inbox:** `ship` Step 3's "Propose, don't implement"
   rule is edited so that a finding whose fix lands in a skill or `_shared` rule is filed as a
   proposal (R1) at the same time it is written into the `## Environment Retro` table
   (`Fix lands in` = `PROP-<n>`); findings that land in tooling/config stay as offered
   follow-ups exactly as today.
   - **Current:** "real builds become `mem_save` findings + offered follow-up items, not
     inline edits" — follow-ups have no persistent home.
   - **Target:** one sentence changed in Step 3's rules + the table's `Fix lands in` column
     gaining `PROP-<n>` as a valid value.
   - **Acceptance:** the edited text names the inbox; the `## Environment Retro` section
     template shows `PROP-<n>` in the `Fix lands in` column.
   - **Acceptance scenario:** `happy path env-retro-finding-filed` → `acceptance.feature`
6. **R6 — Lifecycle doc drift fixed:** `docs/04-lifecycle.md`'s pointer to
   `skill-creator/references/skill-lifecycle.md` is repaired by **materializing** that
   reference file — moving the spec out of `anatomy-and-checklist.md` (§ Usage telemetry /
   Curation transitions / size budget / frontmatter) into
   `skill-creator/references/skill-lifecycle.md`, leaving a one-line pointer behind in the
   anatomy file (same convention as the skill's other "see references/…" splits).
   - **Current:** broken pointer; the spec is split across two files, one of which
   (`anatomy-and-checklist.md`) is about directory shape, not lifecycle.
   - **Target:** the doc's sentence becomes true; `anatomy-and-checklist.md` points at the
     new reference; `docs/04-lifecycle.md` keeps its summary (it is the human-facing doc,
     the reference is the spec).
   - **Acceptance:** `test -f .agents/skills/skill-creator/references/skill-lifecycle.md`
     passes; no `grep` in the doc tree finds a reference to a non-existent path (the
     `check-lens-contract`-style link check, if extended, stays green).
   - **Acceptance scenario:** `happy path lifecycle-reference-materialized` → `acceptance.feature`
7. **R7 — The loop is named in the docs:** `docs/04-lifecycle.md` gains a short
   "Improvement inbox" section (the R1 format in one compact block + the ship routing rule),
   and `docs/02-pipeline.md`'s `ship` row + composition graph note that reflect/env-retro
   file proposals into the inbox.
   - **Current:** neither doc mentions any improvement sink.
   - **Target:** both docs name the inbox in one place each; the format itself is defined
     once (in the inbox `README.md`) and *cited*, not restated (the repo's
     cite-don't-restate craft rule).
   - **Acceptance:** both docs contain a pointer to `.skillgrid/skill-improvements/`; the
     format block appears in full exactly once (in the README) — verified by grep count.
   - **Acceptance scenario:** `happy path docs-name-the-inbox` → `acceptance.feature`

## Implementation Decisions

- **Files to modify:** `.agents/skills/ship/SKILL.md` (Step 2 + Step 3 edits, ~30 lines net);
  `docs/04-lifecycle.md` (pointer fix + one section); `docs/02-pipeline.md` (one note);
  `.agents/skills/skill-creator/references/anatomy-and-checklist.md` (pointer behind);
  `.skillgrid/state.yaml` (nothing — proposals are not changes).
- **Files to create:** `.skillgrid/skill-improvements/README.md` (the contract);
  `_shared/craft/problem-report.md` (the R2 format);
  `.agents/skills/skill-creator/references/skill-lifecycle.md` (R6, moved spec);
  `.agents/skills/skill-creator/scripts/check-proposal.mjs` +
  `scripts/test-check-proposal.mjs` (R1 validation — same script-with-exit-code discipline as
  `check-frontmatter.mjs` / `skill-size-budget.mjs`; the snippet exact-once rule is its core
  assertion, per the BerriAI pattern); Phase 0's own `problem-report-1.md` if observed.
- **Interfaces:** the proposal file is the contract; the field set is fixed (R1). The
  `status` lifecycle is `proposed → approved → applied` or `proposed → rejected | deferred` —
  `approved`/`applied` are written only by a curation pass (Phases 1–2) or a human, never by
  the filing step. Phase 0's `check-proposal.mjs` validates *format*, not *content*.
- **Data flow:** session → `ship` Step 2 (reflect: triage + **ask user** → file proposals;
  Step 3: env retro → file proposals) → `.skillgrid/skill-improvements/PROP-<n>.md` →
  (Phase 1/2) curation pass → (Phase 2) approved edit + eval + backup + commit.
- **Error handling:** `original_snippet` not found or found >1× in the target file → the
  validation check fails with the count named (the edit is not yet safely expressible; the
  proposal stays `proposed`); user declines the experience ask → recorded, no block, ship
  continues (reflect must never hard-block the close-out, same as the `rejected` acceptance
  verdict: recorded, not enforced).
- **Dependencies:** none new. Reuses: `deferred-work.md` id discipline, the
  script-with-exit-code contract, the cite-don't-restate rule, the work-unit commit contract.

### Proposed edit — `ship/SKILL.md` Step 2 Reflect (R4, the user-experience ask)

Insert after the "Triage every Lesson and Surprise" paragraph, before "Acceptance verdict":

```markdown
**User experience (ask, don't assume).** Before the acceptance verdict, ask the user — in
the language they used with you — one question: *what was your experience of this change —
what worked for you, what didn't, and what should the agent do differently next time?*
Keep it to one short answer. Capture the answer **verbatim and attributed**
(`— user, YYYY-MM-DD`) in a `## User Experience` block in the retro half. A user correction
is a first-class signal: it outranks the agent's self-assessment as evidence, and a
correction that names a skill or rule is filed as a proposal
(`.skillgrid/skill-improvements/`, `signal_class: correction`) exactly like a process-bug
Lesson. If the user declines, or there is genuinely nothing to capture, write
`User experience: none captured (asked YYYY-MM-DD)` — the ask is the record; never
fabricate an answer to fill the block.
```

And in the "Where it lands" paragraph of Step 2, extend the retro-half list to include
"the `## User Experience` block" alongside the final-state facts / learnings / verdict.
Corresponding Step 2 triage edit (R3): the process-bug sentence becomes —

```markdown
A *process bug* names a reusable failure class … — file it as a proposal in
`.skillgrid/skill-improvements/` (one file per proposal; the retro half lists the proposal
id), or record "one-off — stays in report" if it names no skill/rule and would not recur.
```

And in Step 3's rules (R5): append to "Propose, don't implement" —

```markdown
A finding whose fix lands in a skill or `_shared` rule is **filed as a proposal**
(`.skillgrid/skill-improvements/`) in the same edit; the `## Environment Retro` table's
`Fix lands in` column then names `PROP-<n>` instead of a free-text follow-up.
```

**Size check:** the net addition to `ship/SKILL.md` is ~30 lines; `ship` is in the
`standard`/`large` tier — confirm headroom with `skill-size-budget.mjs check .` at execute
time (the skill's own token-audit rule applies to this edit).

## Shootdown Findings

**Lens status:** all seven lenses **not run** — this spec is drafted from the source study
and repo investigation (2026-11-25) ahead of the formal spec gate; Goldfish runs when the
spec is reviewed for approval. A `not run` lens caps its contribution at "not verified."

**Anchored hypothesis (as proposed, pending user confirmation at the spec gate):** the
library's self-improvement loop is blocked on its *sink*, not its evidence or its
verification — a destination the ship phase can write into (plus the user-experience signal)
closes the loop for every later phase without new machinery.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| — | (not run) | — | pending spec gate | — |

**Overall verdict:** pending — drafted, not yet shot down.

## Testing Decisions

- **What makes a good test:** external behavior of the format contract and the routing rules,
  not the wording of the skill prose. The snippet exact-once rule and the field-set rule are
  the testable cores; prose edits are verified by the cross-check in R3's acceptance
  (retro half ↔ proposal files) and by re-reading the skill.
- **Modules to test:** `check-proposal.mjs` (unit, alongside `test-check-proposal.mjs`, same
  seam as `test-check-frontmatter.mjs`): well-formed pass; missing field fail; empty
  snippet fail; snippet found 0× / 2× in a fixture skill both fail with the count named;
  status transitions validated only at the field level.
- **Prior art:** `skill-creator/scripts/check-frontmatter.mjs` + its test;
  `deferred-work.md`'s DW-<n> id convention.
- **Edge cases:** proposal targeting a file that has been edited after filing (snippet no
  longer exact-once → stale, reported not auto-fixed); user answer containing the snippet
  text (attribution must keep it from being mistaken for the anchor); `created_by: human`
  target skill (proposal still filed — proposals are never silent edits; the approval gate
  in Phase 2 is what protects human skills).

## Impact on Global Docs

- `.skillgrid/projectwiki/reference/00-prd.md`: None (not present in this repo's vault).
- `.skillgrid/projectwiki/ASSUMPTIONS.md`: add one INFERRED→VERIFIED-at-ship line if the
  inbox location survives the spec gate unchanged (it is a new durable fact: "improvement
  proposals live in `.skillgrid/skill-improvements/`").
- `.skillgrid/projectwiki/ARCHITECTURE.md`: add the inbox to the `.skillgrid/` zone map if
  the file exists at ship time (it is created lazily per sdd-structure — do not create it
  for this).
- `.skillgrid/projectwiki/gotchas/`: one page if the execute phase finds any non-obvious
  behavior (e.g. snippet exact-once edge cases) — per the projectwiki skill, not pre-filed.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | ✓      | user directed the phases and the reflect ask |
| Boundary Clarity    | 0.85  | 0.70 | ✓      | out-of-scope named per phase       |
| Constraint Clarity  | 0.80  | 0.65 | ✓      | size budgets, no-daemon, never-delete all cited |
| Acceptance Criteria | 0.80  | 0.70 | ✓      | all seven requirements falsifiable |
| **Clarity**         | **0.15** | ≤0.20 | ✓ | based on the source study + user direction, not a formal frontier-round interview |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | (prior session) make the skills self-improving; propose, don't change | 5-phase plan agreed in outline |
| 2     | write change-specs for all phases; add a reflect edit asking the user for their experiences | one change-spec per phase; the user-experience ask lands in Phase 0 R4 |

## Open Questions & Assumptions

| Type | Item | Owner | Due |
|------|------|-------|-----|
| Question | Should `check-proposal.mjs` live under `skill-creator/scripts/` (curation tooling) or a new `skill-creator/scripts/` sibling? Default: `skill-creator/scripts/` — it is lifecycle tooling. | user | spec gate |
| Question | Is `## User Experience` a fixed retro-half section name (machine-greppable) or free prose? Default: fixed name — Phase 4's metrics will grep it. | user | spec gate |
| Assumption | The inbox is *not* a spec-dir change (no `state.yaml` entry per proposal) — it is library-level state like `deferred-work.md`. | devopstales | until disproved |
| Assumption | One user question at reflect is acceptable friction for the terminal phase (the user has already approved the change; this asks about the *process*, not the *work*). | devopstales | until disproved |

## Decisions (ADR)

None yet — the inbox location clears the "hard to reverse" bar only if Phase 2 adopts it as
the curation input; it is recorded in ASSUMPTIONS.md instead until then. If the spec gate
confirms the location, file `NNNN-improvement-inbox-location.md` at ship.

## Terms

- **Improvement proposal** (new): a reviewable, approvable edit against a named skill, in
  the R1 format. Added to `.skillgrid/projectwiki/glossary/01-business-terms.md` at ship.
- **Problem report** (new): the R2 in-flight failure record. Same glossary.
- **Signal class** (new, from the viktor source): `correction` | `test` | `eval` |
  `observation` — the provenance of a proposal's evidence. Same glossary.

## Retrospective

**Status:** pending

- **Intent met?** [filled by ship]
- **Pre-mortem score:** [filled by ship]
- **Assumptions:** [filled by ship]
- **Timebox:** [filled by ship]
- **Next time:** [filled by ship]
