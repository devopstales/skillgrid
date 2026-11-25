# Self-Improving Loop — Phase 3: User-project mode (L2) — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-25)
> **Author:** devopstales
> **Intent frozen:** no (pending) ← the agent sets this to "yes (YYYY-MM-DD)" once the STATUS banner above shows `approved` (the user's hand-edit IS the approval — the agent never writes `approved` itself); after that, the frozen block below is read-only for agent steps

> **Phase 3 of the self-improving-skills plan.** Depends on Phase 0 (inbox format,
> problem-report format, the reflect user-experience ask) and Phase 2 (the approval-gated
> write path). Phases 0–2 make *this repo* (the library) self-improving. Phase 3 makes an
> *installed* skillgrid — in any user's project — learn from that project's sessions: user
> corrections become reviewed rules in the project's `projectwiki`, with tiers, counters,
> and the anti-poisoning gates.

## Problem / Intent

A user who installs skillgrid gets the pipeline, but not the learning: their project's
corrections ("no, migrations live in `db/migrations`"), gotchas (the FTS5 upsert trap they
hit at 2 a.m.), and preferences (one commit per logical change) live in their head or in
chat history. Every fresh session re-learns nothing. The repo's own `ship` reflect now
*asks* the user for experiences (Phase 0 R4) — but in a user project those answers land in
`report.md` and die with the change folder, because the **project** has no per-lesson
memory with a write path, a tier, and a review gate.

**Job to be done:** when skillgrid runs in a user project and the user corrects the agent,
or a check goes RED, the project wants that lesson captured as a **reviewed, tiered,
dated** entry in its `.skillgrid/projectwiki/` (the vault the install already creates), so
the next session — and every teammate's session — reads it before starting.

Why now: the building blocks are Phase 0's (problem-report format, sourced-learnings
discipline, the user-experience ask) and the projectwiki engine (pages, `claims[]`, lint,
index) — what's missing is the *lesson lifecycle*: counters, tiers, promotion/demotion,
and the write gate.

## Purpose & Success Criteria

- **Purpose:** a **learn-from-corrections** convention (a `skill-creator`-owned reference
  or a small cross-cutting skill — see Approaches) that any skillgrid session in *any*
  project can invoke: it turns a correction / RED signal / user-experience answer into a
  projectwiki lesson entry (typed, countered, sourced), promoted/demoted by usage, and
  surfaced to the session that needs it.
- **Success criteria:**
  1. A user correction in a session produces, after the user approves the proposed line, a
     projectwiki entry in `gotchas/` (pitfall) or `preferences/` (preference/correction)
     with: the rule (one line, general beyond the case), the category tag
     (`pitfall` | `strategy` | `correction` | `preference` | `meta-rule`), the verbatim
     source (user quote with date, or the RED output), `helpful: 0`, `harmful: 0`,
     `last_verified: <date>`, and the change id it came from.
  2. **Writes are deltas**: the lesson is appended to (or a new page in) the vault —
     never a rewrite of an existing page's other entries (the ACE context-collapse
     rule); a correction of an *existing* entry is an edit to that entry with a
     supersession note (`supersedes: <entry>`), not a silent replacement.
  3. **Tiers with movement rules** (the HOT/WARM/COLD analogue on the existing 90/180-day
     lifecycle): a lesson used 3× in 7 days is *promoted* — into a skill's reference or a
     `_shared` rule in the *library* (filed as a Phase 2 improvement proposal — the
     project→library direction of the loop); unused 30 days → demoted (marked, still
     queryable); unused 90 days → archived into the vault's archive area (never deleted,
     the lifecycle invariant).
  4. **Counters move on evidence only**: `helpful` increments when a session *cites* the
     entry to decide (the projectwiki `cite` mechanism already records citations);
     `harmful` increments when a session records the entry as wrong/stale (a
     problem-report with the entry as `signal`). No self-scoring (the repo's
     telemetry rule, applied to lessons).
  5. **Verifiable-signal-before-rule** holds at the project level too: a correction
     without a verbatim user quote (or a RED signal) is recorded as an *observation* —
     queryable, but not promotable and not surfaced in the session preamble.
  6. **Recall is scoped**: the vault's `index.md` (the engine-projected catalog) is how a
     fresh session sees lessons — no new always-loaded file (the viktor "every always-
     loaded line costs tokens" rule): lessons are *queried* (the projectwiki skill's Read
     mode already does recall-before-re-deriving) and *cited*, not injected.
- **Out of scope:** cross-project lesson sharing (one user's project lessons do not flow
  to another's — the library is the only cross-project channel, via Phase 2 proposals);
  machine-local memory (the vault is committed project state — the viktor doc's CLAUDE.
  local.md tier is deliberately not adopted: this repo commits its knowledge); the L1
  library loop (Phases 0–2); an automatic preamble injection of "all HOT lessons" (the
  index catalog is the surface); a new MCP surface (the projectwiki engine's existing
  `projectwiki_*` tools are the interface).

## Context

- **The vault (already installed per project):** `.skillgrid/projectwiki/` — page types
  incl. `gotchas/` (discovered issues / non-obvious behavior) and `preferences/` (user
  preferences, "each bullet cites the memory: observation"), `claims[]` on invariants,
  the engine (`mnemonic projectwiki ...`: query / read / cite / lint / index), `index.md`
  projection, `log.md` action log. The *shape* for lessons largely exists; what's missing
  is the lifecycle contract on top.
- **The formats (Phase 0):** the problem-report format (`signal` must be verifiable) and
  the sourced-learnings rule ("a finding with no source is a guess") — both already
  written and citable.
- **The write gate (Phase 2):** the approval-gated apply discipline (show the exact line,
  approve in the current message, backup before mutation — here the "backup" is git: the
  vault is committed; the work-unit commit names the source change).
- **The memory convention:** `_shared/rules/mnemonic-memory.md` — the `mem_*` save shape
  for SDD artifacts; this phase's lessons are *vault pages*, not `mem_save` observations
  (two different stores, the division stated in the reference so sessions don't
  double-record).
- **Upstream patterns adapted:** viktor — the five categories, the HOT/WARM/COLD tiers
  with 3×/7d promotion and 30d/90d demotion, "never delete without asking," delta writes,
  "flag both for a person" on contradiction, the worked `/learn-from-corrections`
  example; Peter Yang — the per-skill `memory.md` lesson log (here: per-project, in the
  vault, typed); Perplexity — the "two independent occurrences for creation, one grounded
  occurrence for update" bar (creation = promoting to the library; update = editing an
  entry); Shao et al. via viktor — lessons framed as *reference, not law*.
- **Contradictions:** the projectwiki lint already checks page contracts; the lesson
  contract extends it (category tag, counters, dates) — a lint extension, not a parallel
  checker (the deterministic-boundary rule).

## User Context

- **Who:** the project's developers — the people who install skillgrid and correct the
  agent in *their* repo.
- **Direct evidence:** the viktor worked example (the `npm test` vs `npm run test:unit`
  correction) is the canonical case; in-repo analogue: this repo's own `## User
  Experience` answers (Phase 0) currently have a project-level home only in the change
  folder. The user's request "ask the user for his experiences" generalizes: in a user
  project, those experiences are *the* lesson source.
- **Why it fails today:** a correction said once is honored in the current session and
  lost in the next (fresh context); the team re-pays the correction cost on every
  teammate, every session — and a wrong lesson, if it were captured without a gate, would
  be *worse* than none (the misevolution risk via viktor/OWASP).

## Pre-mortem

- **Desirability:** lessons accumulate as untrusted notes nobody reads ("a folder of
  claims"). Countermeasure: the citation counter is the trust mechanism — an entry with
  `helpful: 0` after two relevant sessions is a lint finding, and demotion retires it;
  the index catalog surfaces entries by type so recall is a query, not a rummage.
- **Feasibility:** the movement rules (3×/7d etc.) need *usage* data at the project level
  — and the Phase 1 sidecar is library-level. Countermeasure: project-level usage =
  projectwiki `cite` records + problem-report `signal` refs (both already
  engine-written); no new telemetry for v1 — the 7-day window is computed from citations,
  and if a project has too few citations to move anything, *nothing moves* (the honest
  output, not a guess).
- **Viability:** if "approve each lesson line" becomes per-session friction, sessions
  stop filing. Countermeasure: the gate is *one line, one approval* at the natural pause
  (the ship reflect already asks the user — the lesson write is the follow-through on an
  answer they just gave, not a new interruption); mid-session corrections are batched to
  the reflect/ship moment or an explicit `/record`-style user ask.

## Timebox

Target: within four weeks (2026-11-25 → 2026-12-23), after Phases 0–2 land. If exceeded:
ship the write path + lint contract (R1, R2, R3) first; tiers/counters (R4) and the
project→library promotion (R5) become the follow-up tickets. The write gate never slips
— it is the anti-poisoning floor.

## Approaches Considered

| Approach | Key trade-off | Verdict |
|----------|---------------|---------|
| A **convention + reference** owned by `skill-creator` (`references/learn-from-corrections.md`) wired into the projectwiki skill's Read/Maintain modes and `ship` reflect | No new skill (the curation-bar test: does it compose? — yes: projectwiki owns the vault, ship owns the moment, skill-creator owns the lesson lifecycle as library tooling) | **Chosen** |
| A new cross-cutting skill (`learn`) | Discoverable trigger, but it would own *nothing* (the vault is projectwiki's, the moment is ship's, the gate is Phase 2's) — a thin router is only allowed when the workflow is thick (the `_shared` thin-router rule) | Rejected |
| Per-skill `memory.md` files in user projects (Peter Yang literal) | Simple, but scatters lessons across skill dirs in the *user's* install (which they may not own) — the vault is the committed, shared, linted home | Rejected — the vault is the memory |
| CLAUDE.md-style always-loaded rules file per project | The adherence cost viktor's sources warn about (200-line cap, adherence drops); and a second SoT beside the vault | Rejected |

- **Chosen:** convention + reference, wired into existing skills; the vault is the memory.
- **Rejected by author:** a `learn` skill (thin router over three owners it doesn't own).

## Requirements

1. **R1 — Lesson page contract + lint:** the lesson entry shape is fixed (a page-type
   extension of the vault): in `gotchas/` (pitfall/strategy) or `preferences/`
   (correction/preference/meta-rule), each entry carries: one-line rule (general beyond
   the case), `category` (the five viktor tags), `source` (verbatim user quote + date, or
   RED output, or the citing problem-report id), `helpful`/`harmful` (integers, start 0),
   `last_verified` (date), `origin` (change id), `supersedes` (optional entry ref). The
   projectwiki lint gains the lesson checks (missing category / missing source /
   `harmful > helpful` without a recorded review → findings). `meta-rule` entries
   (rules about the rules) are the only lesson type that may modify *other lessons'
   handling* — and they are flagged on every review screen.
   - **Current:** `gotchas/` and `preferences/` pages exist with looser contracts
     (preferences cite the observation; gotchas have no counters).
   - **Target:** the contract in the reference + the lint extension (the engine's lint
     command reports lesson findings alongside existing page findings).
   - **Acceptance:** a lesson page missing `category` or `source` is a lint finding
     naming the page; a well-formed lesson lints clean; the vault's existing non-lesson
     pages are unaffected (the contract applies to lesson-shaped entries, identified by
     the counter fields' presence — backward compatible).
   - **Acceptance scenario:** `happy path lesson-lint-contract` → `acceptance.feature`
2. **R2 — The write path (deltas, gated):** the learn-from-corrections procedure
   (in the reference, invoked from `ship` reflect's user-experience follow-through and
   available on user ask mid-session): (a) capture the correction/signal verbatim;
   (b) generalize to one line (the Reflexion generalize step — the skill states the
   mistake/cause/correct-approach triad first, then the rule); (c) **show the exact line
   to add and wait for approval** (the viktor worked example's step 4, the Phase 2 gate
   discipline at project scale); (d) append as a new entry (or supersede, with the
   note) — never rewrite the page wholesale; (e) work-unit commit naming the origin
   change. The skip rule (viktor step 3): if the vault already says it, or the code makes
   it obvious, no entry — recorded as "already covered: <entry>" when skipped.
   - **Current:** reflect captures the user's words in the change folder; the vault's
     preferences/gotchas pages get entries only when a session happens to think of the
     projectwiki skill.
   - **Target:** the procedure + the two invocation points named in the owning skills
     (one line each in `ship` Step 2 and the projectwiki Maintain mode — pointers, not
     restatements).
   - **Acceptance:** on a seeded project vault, a user correction ("migrations live in
     `db/migrations`") produces, after approval, exactly one new entry with all contract
     fields; a correction that duplicates an existing entry produces "already covered,"
     not a second entry; a rejected proposal leaves the vault byte-identical.
   - **Acceptance scenario:** `happy path correction-becomes-entry` → `acceptance.feature`
3. **R3 — Counters on evidence only:** `helpful` increments on a projectwiki *citation*
   of the entry (the engine's `cite` records it — the counter is derived from citation
   records at index time, not hand-bumped); `harmful` increments on a problem-report
   naming the entry in its `signal`. The derivation is part of `projectwiki index`
   (the engine already rebuilds derived state — the counters are derived state, the
   engine's PW-INV-3: never hand-edited).
   - **Current:** citations exist in the engine; nothing counts them; `harmful` has no
     source at all.
   - **Target:** the derivation in the engine's index pass (or, if the engine is
     out-of-repo for this change's scope, a `scripts/`-level derivation the index
     invokes — decided at blueprint; the *contract* — counters are derived, never
     hand-written — is the requirement).
   - **Acceptance:** citing a lesson twice in a fixture project → `helpful: 2` after
     reindex; a problem-report citing it → `harmful: 1`; hand-editing a counter →
     reverted by reindex (derived state wins).
   - **Acceptance scenario:** `happy path counters-derived` → `acceptance.feature`
4. **R4 — Tiers and movement:** the lesson lifecycle on the vault: **promotion** — a
   lesson cited 3× in 7 days is *proposed* to the library (an improvement proposal per
   the Phase 0 format, `target_skill` = the governing skill, the project as origin — the
   project→library channel, gated by Phase 2 exactly like any proposal); **demotion** —
   unused 30 days → marked `dormant` (still queryable, excluded from the index catalog's
   prominent listing); **archive** — unused 90 days → moved to the vault's archive area
   (the lifecycle's never-delete rule; the move is logged in `log.md`). Contradictions:
   two lessons that conflict are *both* flagged for the human at the next review (the
   "flag both, don't pick" rule) — the review moment is the curation pass (Phase 1),
   extended to project vaults.
   - **Current:** vault pages have no movement; the 90/180 rules exist only for library
     skills.
   - **Target:** the movement rules in the reference + the curation pass (Phase 1's)
     extended to run on a project vault (the same five-step procedure, the same
     invariants — one procedure, two vaults).
   - **Acceptance:** fixture: a lesson with 3 citations in a 7-day window → a library
     proposal is drafted (not auto-applied — Phase 2's gate); a 30-day-unused lesson →
     `dormant`; a 90-day-unused lesson → archived + logged; two conflicting lessons →
     both flagged, neither deleted.
   - **Acceptance scenario:** `happy path tier-movement` → `acceptance.feature`
5. **R5 — Anti-poisoning at project scale:** the four guardrails, written in the
   reference as Red Flag rows and enforced by the write path: (a) **provenance gate** —
   every entry's `source` is the user's words or a machine signal; an *agent-generated*
   claim (its own report prose) is not a valid `source` (no self-re-ingestion — Phase 2
   R4's rule, project-scoped); (b) **verifiable-signal-before-rule** — `correction`/
   `pitfall` entries without a verbatim source are `observation`-tagged and excluded from
   promotion; (c) **reference-not-law** — the index catalog's lesson section is headed
   "Lessons (reference, not law)" (the Shao et al. framing via viktor — one line of
   engine projection, zero cost); (d) **expire unverified** — `last_verified` older than
   90 days *with* `harmful > 0` is a lint finding (the OWASP decay rule).
   - **Current:** none of the four at the project level.
   - **Target:** the four rules + the one catalog heading; a scenario eval per rule via
     the Phase 1 runner (the lesson contract is testable the way the library's is).
   - **Acceptance:** fixture project: an entry sourced from agent prose is
     `observation`-tagged by the write path (not `correction`); a stale
     `harmful > 0` entry lints; the catalog heading renders.
   - **Acceptance scenario:** `happy path anti-poisoning-gates` → `acceptance.feature`
6. **R6 — Install docs + the convention's trigger:** `INSTALL_TEMPLATE.md` (the
   install doc users get) gains the one-paragraph "your project learns" note (what gets
   recorded, where, how to approve/edit/delete — the vault is plain markdown, the user's
   files); the projectwiki skill's description is checked for trigger coverage of the
   lesson modes (its §9 rule, as in Phase 2 R6); `docs/01-skills.md`'s projectwiki row
   names the lesson lifecycle.
   - **Current:** the install template does not mention learning; the projectwiki
   description covers the vault but not lessons.
   - **Target:** the three doc edits, each a pointer to the reference (cite, don't
   restate).
   - **Acceptance:** a fresh install following `INSTALL_TEMPLATE.md` lands a project
   where the first correction produces a lesson entry without the user having to find
   the convention (the `ship` reflect wiring makes it automatic at the natural moment);
   the description trigger check passes.
   - **Acceptance scenario:** `happy path install-learns` → `acceptance.feature`

## Implementation Decisions

- **Files to create:** `skill-creator/references/learn-from-corrections.md` (the
  procedure: capture → generalize → show+approve → delta-write → commit; the movement
  rules; the four anti-poisoning Red Flags; the two-vault note on the curation pass),
  the lesson fixture project (`.agents/skills/skill-creator/evals/fixtures/` — a toy
  vault with seeded lessons/citations for R1–R5 tests).
- **Files to modify:** the projectwiki engine's lint + index (the lesson contract checks
  + counter derivation — the engine is the in-repo Go tooling per the mnemonic-core
  change specs; if the engine's scope makes that a separate PR, the *contract* still
  ships and the lint runs via a `scripts/`-level checker until the engine lands it —
  the blueprint decides the seam), `projectwiki/SKILL.md` (one line in Maintain mode:
  the lesson lifecycle pointer + the catalog heading), `ship/SKILL.md` Step 2 (one line:
  the user-experience follow-through → learn-from-corrections), `INSTALL_TEMPLATE.md`
  (one paragraph), `docs/01-skills.md` (the projectwiki row).
- **Interfaces:** the lesson entry shape (fixed, R1 — the contract is the interface);
  counters as *derived* state (never an input); the promotion artifact = a Phase 0-format
  proposal with `origin: project:<repo>` — the library's inbox already accepts it (the
  format's `evidence` field carries the project's citation records).
- **Data flow:** session correction / RED signal / reflect answer → (approve) → vault
  entry (delta) → engine index (counters derived from citations) → curation pass
  (movement: promote → library inbox / demote / archive) → next session's recall
  (projectwiki Read mode queries the catalog).
- **Error handling:** approval declined → the lesson is kept in the change folder's
  retro half (nothing lost, no vault noise); a conflicting pair → both flagged, human
  decides (never an auto-winner); counter derivation with missing citation records (old
  projects) → counters start at 0, derived from then on (no backfill guess); engine
  without the lint extension yet → the `scripts/`-level checker runs, the engine gap is
  a tracked ticket (degradation stated, not hidden).
- **Dependencies:** Phases 0+2 as stated; the projectwiki engine state as shipped by the
  in-flight mnemonic changes (coordinate at the blueprint gate — the vault contract is
  the stable surface either way).

## Shootdown Findings

**Lens status:** all seven lenses **not run** — drafted from the Phase 0 plan + the viktor
source study; Goldfish at the spec gate.

**Anchored hypothesis (as proposed, pending user confirmation):** the user-project loop
is the *vault plus a gated delta writer plus derived counters* — no new memory store, no
always-loaded file, no new skill — and the library stays the only cross-project channel
by design.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| — | (not run) | — | pending spec gate | — |

**Overall verdict:** pending — drafted, not yet shot down.

## Testing Decisions

- **What makes a good test:** the write path on a seeded fixture vault (approve/reject/
  duplicate/supersede); the counter derivation (citations → integers, reindex wins over
  hand-edits); the movement rules on dated fixtures (3×/7d, 30d, 90d, conflict pair);
  the four anti-poisoning rules as scenario evals. The *generalization quality* (the one-
  line rule being actually general) is the approval screen's job — the user is the test.
- **Modules to test:** the lint extension (lesson findings); the index derivation
  (counters); the write-path fixtures; the curation-pass extension (two-vault run).
- **Prior art:** Phase 1's curation fixture; the projectwiki engine's existing lint/
  index tests (same seam, same style); the Phase 2 scenario evals for the pass.
- **Edge cases:** a lesson cited in the same session that created it (the citation
  counts — it *was* used; the 7-day window prevents a single session from promoting,
  because promotion needs 3 citations); a vault with no `log.md` (lazy creation, the
  engine's existing behavior); promotion of a lesson whose governing skill is
  `created_by: human` (the proposal is flagged `human-skill` — Phase 2's flag, unchanged);
  a user deleting an entry by hand (git is the vault's memory — the counter derivation
  simply follows; no reconciliation needed).

## Impact on Global Docs

- `.skillgrid/projectwiki/ASSUMPTIONS.md`: VERIFIED at ship — "project lessons live in
  the project vault, are derived-countered, and promote to the library only via Phase 2
  proposals."
- `.skillgrid/projectwiki/concepts/`: `lesson-lifecycle.md` (the tiers + the four
  anti-poisoning gates as a concept).
- `.skillgrid/projectwiki/decisions/`: `NNNN-vault-as-project-memory.md` — the
  rejection of CLAUDE.md-style always-loaded files and of per-skill memory.md in user
  projects (real trade-off, hard to reverse: the install's data layout).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.85  | 0.75 | ✓      | project-level learning, named per piece |
| Boundary Clarity    | 0.80  | 0.70 | ✓      | library stays the only cross-project channel |
| Constraint Clarity  | 0.85  | 0.65 | ✓      | deltas-only, derived counters, vault-committed |
| Acceptance Criteria | 0.85  | 0.70 | ✓      | all six requirements falsifiable   |
| **Clarity**         | **0.15** | ≤0.20 | ✓ | from the plan + viktor source mapping |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | (prior session) the installed project should learn too, with the same gates | Phase 3 = L2; the vault is the memory |
| 2     | (this session) specs for all phases | convention + reference, not a new skill; counters derived |

## Open Questions & Assumptions

| Type | Item | Owner | Due |
|------|------|-------|-----|
| Question | Engine seam: does the lint/index extension land in the mnemonic engine (in-flight changes) or a `scripts/`-level checker first? | user + the mnemonic-core owners | blueprint gate |
| Question | 3×/7d promotion window — confirm the viktor numbers (they are a community-skill default, not a measured optimum). | user | spec gate |
| Assumption | The vault is committed project state (teammate-shared); machine-local notes are a non-goal for v1. | devopstales | until disproved |
| Assumption | One correction → at most one entry per session (batched at reflect/ship); mid-session writes only on explicit user ask. | devopstales | until disproved |

## Decisions (ADR)

- `NNNN-vault-as-project-memory.md` — see Impact (the layout choice is the load-bearing
  one).
- (Candidate) the derived-counters rule (counters are engine output, never input) —
  recorded as a concept page unless the engine seam question makes it contentious.

## Terms

- **Lesson** (new, project-level): a vault entry in the R1 contract. Glossary at ship.
- **Dormant** (new): the 30-day demotion state. Glossary at ship.
- **Project→library promotion** (new): the channel by which a proven project lesson
  becomes a library proposal. Glossary at ship.

## Retrospective

**Status:** pending

- **Intent met?** [filled by ship]
- **Pre-mortem score:** [filled by ship]
- **Assumptions:** [filled by ship]
- **Timebox:** [filled by ship]
- **Next time:** [filled by ship]
