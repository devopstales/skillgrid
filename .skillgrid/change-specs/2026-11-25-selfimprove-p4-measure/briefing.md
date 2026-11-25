# Self-Improving Loop — Phase 4: Prove it (measurement) — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-25)
> **Author:** devopstales
> **Intent frozen:** no (pending) ← the agent sets this to "yes (YYYY-MM-DD)" once the STATUS banner above shows `approved` (the user's hand-edit IS the approval — the agent never writes `approved` itself); after that, the frozen block below is read-only for agent steps

> **Phase 4 of the self-improving-skills plan.** Depends on Phases 0–3 (the loop exists
> end-to-end). This phase is the loop's *scoreboard*: the metrics that answer the
> question the whole plan exists to answer — **is the library actually getting better?**
> — plus the before/after pilot on this repo that gives the first data points.

## Problem / Intent

A self-improving loop with no measurement is a belief. The sources the user cited are
consistent on what "better" means and how to measure it cheaply (viktor's measurement
section, Perplexity's randomized test): **repeat-error rate** (does a previously diagnosed
failure reappear?), **task/eval success on a fixed set with the skill off vs on**, and
**cost per task** (tokens/steps). This repo already mandates per-skill measurement
(`_shared/craft/measurement.md`: one leading + one lagging indicator, a real artifact as
data source, explicit direction) — but the *library-level* metrics the loop itself needs
(did the loop reduce recurrences? are applied edits not degrading evals?) have no home,
no data source, and no cadence.

**Job to be done:** when the maintainer asks "is the self-improvement working?", they want
a number with a direction — from artifacts the loop already writes — not an impression.

Why now: after Phases 0–3 every signal the metrics need is being recorded (diagnosis codes
on proposals, replay matrices, eval runs, apply/reject dispositions, user-experience
blocks). Measurement is the last stage of the Perplexity loop ("future sessions provide
evidence for the next round" — the round only closes if someone scores it).

## Purpose & Success Criteria

- **Purpose:** a **library scoreboard**: a deterministic report (a script with an exit
  code, run by the curation pass and on demand) that computes the loop's metrics from the
  loop's own artifacts, with a leading + lagging pair per `_shared/craft/measurement.md`,
  and a **before/after pilot** on this repo's existing eval corpus that produces the
  first baseline.
- **Success criteria:**
  1. **Repeat-error rate (the lagging headline):** every proposal carries a
     *diagnosis code* (a short stable slug naming the failure class — e.g.
     `missing-approval-gate`, `stale-snippet`, `trigger-miss` — the Phase 0 format gains
     one field: `diagnosis_code`); the scoreboard computes, over a window, the share of
     new proposals whose `diagnosis_code` matches an *already-applied* proposal's code
     (a recurrence — the same failure class was fixed and came back) vs. new codes.
     Direction: should **fall** as the loop matures. A recurrence with the applied
     fix's commit named is a **loop bug** (the fix didn't hold — the eval that proved it
     is wrong), surfaced as a first-class finding, not averaged away.
  2. **Eval pass rate on a fixed corpus (the leading indicator):** the scoreboard runs
     `run-evals.mjs` (Phase 1) over the repo's full existing scenario corpus
     (11 skills with evals; `execute`/`brainstorm` with `scenarios.yaml`) in a scheduled
     pass and reports the pass rate; direction: should **hold** (a drop after an applied
     edit is the regression signal the paired-replay gate is supposed to catch — a drop
     *between* passes means the gate was bypassed or the corpus changed; the report
     distinguishes "corpus changed" from "skill changed" by diffing the corpus files).
     This is the SAGE-style "skill on vs off / before vs after" made local and cheap: the
     corpus is fixed, the model is the harness's, the comparison is over time.
  3. **Loop throughput (a health indicator, not a vanity metric):** proposals filed /
     applied / rejected / abstained per window + median time from `proposed` to
     disposition + the accept rate (applied / (applied + rejected)). Direction: accept
     rate should **hold or rise** (a falling accept rate = the pass is proposing junk —
     the Phase 2 pre-mortem's distrust scenario, now measurable); throughput itself has
     *no* direction (more is not better).
  4. **The report is an artifact, not a chat reply:** `scoreboard.mjs` writes
     `.skillgrid/skill-improvements/scoreboard.md` (committed; one run = one dated
     section, append-only like `log.md`), and the curation pass (Phase 1) includes the
     scoreboard run in its five-step procedure (step 1.5 — status now includes the
     metrics). Exit code: 0 = computed; 1 = a loop-bug recurrence found (the report is
     still written — the exit code *is* the alarm).
  5. **The pilot baseline:** a fixed task set — the repo's existing acceptance scenarios
     + a small set of real change types (documented in the change folder) — is run
     with the improvement machinery **off** (the pre-Phase-2 tree) and **on** (current
     tree), same model, order varied once (the PATH-Bench caveat via viktor: task order
     can change rankings — one order swap, not a study). The result is the first
     before/after row in the scoreboard, with the honest caveat that one repo's data is
     a baseline, not a claim.
- **Out of scope:** per-user or cross-project metrics (this repo is the instrumented
  library; user projects get the scoreboard as an installable script but no aggregate
  telemetry — no data leaves the project, consistent with Phase 3's provenance stance);
  model-quality tracking (model swaps are a separate confound — the report stamps the
  model name per run so a model change is visible as a line break, not investigated
  here); dashboards/UI (a dated markdown section is the surface; the `/plans` endpoint
  already renders spec state — the scoreboard is a file); A/B randomization at
  Perplexity's scale (this repo has one "user"; the pilot's off/on comparison is the
  honest local analogue).

## Context

- **The measurement contract (already the rule):** `_shared/craft/measurement.md` —
  two indicators (leading + lagging), a real artifact as data source ("'feels faster' is
  not a data source"), explicit direction, cheap enough to run often, one skill its own
  table. This phase is that contract applied to the *library as a system*, and it
  extends the contract with the diagnosis-code convention (a new durable rule, recorded
  in the craft doc, not invented in the report).
- **The data sources (all written by Phases 0–3):** proposals + `diagnosis_code` +
  `status`/`disposition` timestamps (Phase 0 format, extended by R1); replay matrices
  (`run-evals.mjs --json`, Phase 1); the scenario corpus (existing, 11 skills); the
  sidecar (Phase 1); apply/reject commits (the work-unit commit body naming `PROP-<n>`,
  Phase 2); lesson counters (Phase 3 — the project-level analog; the library scoreboard
  reads the *library* inbox, project vaults are out of scope per the line above).
- **The cadence owner (already exists):** the curation pass (Phase 1) — the scoreboard
  is its step 1.5; on-demand runs are the maintainer's.
- **Upstream evidence adapted:** viktor — the three metrics (repeat-error rate with
  diagnosis codes; task success on repeated runs; tokens/steps) + the "keep the skill
  only if the repeat-error rate falls without task success dropping" decision rule + the
  baseline caveat ("the baseline determines the headline"); Perplexity — the measured
  outcomes (tool-failure −11%, dissatisfaction −7.6%) as the *shape* of the claim this
  repo's data will eventually support, not a borrowed number; SAGE via viktor — the
  skill-on/off ablation as the pilot's design.
- **Repo state:** the eval corpus is the pilot's asset; `scripts/` already hosts
  repo-level tooling (`state-lock.mjs`, `test-hooks.mjs` — the script+test pairing is the
  convention).

## User Context

- **Who:** the maintainer, at the scheduled curation moment (the scoreboard arrives with
  the status table) and at any "is this working?" question.
- **Direct evidence:** the user asked for the plan *from* five sources, three of which
  lead with measurement (Perplexity's experiment, viktor's metric section, SAGE) — the
  ask "check the skills and plan how we can make the self improving" implicitly includes
  "and show me it improved." A loop that cannot answer that is a feature; one that can
  is a practice.
- **Why it fails today:** without the scoreboard, the accept-rate/decay signals
  (Phases 2–3's pre-mortems: "the pass proposes junk," "lessons accumulate unread") are
  *felt* but not *seen* — and a felt signal is exactly what the measurement contract
  forbids as a data source.

## Pre-mortem

- **Desirability:** the scoreboard becomes a ritual nobody reads — a dated section
  appended forever. Countermeasure: the exit-code alarm (a loop-bug recurrence fails the
  run — the curation pass *must* surface it) and the one-line verdict at the top of each
  section ("repeat-error rate: 0/7 new proposals recurrent; eval pass rate 94%→96%;
  accept rate 3/4") — the detail is below the line, the line is the deliverable.
- **Feasibility:** diagnosis codes need *discipline* at filing time (a bad code kills the
  recurrence match). Countermeasure: the code is a **closed-ish set with a slug rule** —
  the Phase 0 `check-proposal.mjs` validates the code's shape (kebab, ≤40 chars), and the
  curation pass reviews code *consistency* (two codes naming the same class → merge
  finding); matching is exact-slug, not semantic (no embedding guesswork in v1 — the
  mnemonic embedder exists but exact-slug is the honest v1; semantic matching is an
  explicit follow-up if codes fragment).
- **Viability:** the pilot (R5) costs model runs. Countermeasure: the pilot is a *one-
  time* baseline (the corpus is small — 11 skills' evals, a handful of scenarios), run
  once off and once on, with the model stamped; it is not a recurring experiment (the
  recurring indicator is the pass rate in R2, which the curation pass runs anyway).

## Timebox

Target: within five weeks (2026-11-25 → 2026-12-30), after Phases 0–3 land. If exceeded:
ship R1 (the code + match) + R4 (the report/cadence) first; the pilot (R5) becomes a
standalone follow-up with the same spec (it is self-contained). R2 (the pass-rate run)
never slips — it is the leading indicator the whole plan leans on.

## Approaches Considered

| Approach | Key trade-off | Verdict |
|----------|---------------|---------|
| A **deterministic scoreboard script** (`scoreboard.mjs` in `scripts/`) + the curation pass running it | A new repo-level script (joins `state-lock.mjs` etc.); the model is not involved in computing | **Chosen** — the metrics are *counts and diffs over the loop's artifacts*; a model computing them would be self-grading (the telemetry rule) |
| An LLM-judged "health review" of the loop (a Goldfish lens over the scoreboard data) | Judgment on *interpretation*, but a judge that also *computes* is the self-scoring trap | Rejected as the computer; **kept as an optional interpretation addendum** in the curation pass (a lens reads the report and comments — the number never comes from it) |
| Per-skill metrics only (each skill's measurement.md table, aggregated by hand) | No new tooling, but the *loop's* metrics (recurrence, accept rate) are cross-skill by definition — hand aggregation is where the signal dies | Rejected — per-skill tables stay as they are; the loop gets its own scoreboard |
| A dashboard in the `/plans` endpoint | Visible, but the endpoint parses spec state; a second data model for metrics is a dependency creation | Rejected — a committed markdown artifact is the surface (the repo's docs-first rule) |

- **Chosen:** `scoreboard.mjs` + curation-pass cadence + the one-time pilot.
- **Rejected by author:** the LLM-computed health review (self-scoring).

## Requirements

1. **R1 — Diagnosis codes on proposals:** the Phase 0 proposal format gains
   `diagnosis_code` (required, kebab-case slug ≤40 chars, naming the failure class in
   one noun phrase — the *class*, not the instance: `trigger-miss` not
   "the debug skill didn't fire on Tuesday"); `check-proposal.mjs` validates the shape;
   the curation pass reviews code consistency (duplicate-class codes → a merge finding
   with the two slugs named). Existing proposals (filed before this phase) get codes at
   their next curation review (backfilled, never fabricated — an uncodeable proposal is
   `diagnosis_code: uncoded`, which the scoreboard counts separately and the pass
   chases).
   - **Current:** proposals carry `signal_class` (the *provenance* of the evidence) but
     no failure-*class* code — recurrence cannot be matched.
   - **Target:** the field + validation + the consistency review in the curation pass
     procedure (one step, one Red Flag row: "two codes, one class").
   - **Acceptance:** a proposal with a malformed code fails `check-proposal.mjs` with the
     rule named; a fixture inbox with two same-class codes produces a merge finding in
     the curation output.
   - **Acceptance scenario:** `happy path diagnosis-code-validated` → `acceptance.feature`
2. **R2 — Scoreboard computation:** `scripts/scoreboard.mjs [--window N-days] [--json]`
   reads the inbox (+ `diagnosis_code`, statuses, timestamps), the last replay
   matrices, and the scenario corpus, and writes the dated section to
   `.skillgrid/skill-improvements/scoreboard.md`: the one-line verdict, then
   (a) **repeat-error rate** — new proposals in the window, matched against applied
     codes; recurrences named with the original fix's commit (loop-bug findings listed
     individually); (b) **eval pass rate** — the corpus run's result, previous value,
     delta, and the "corpus changed vs skill changed" distinction (corpus files' git
     diff in the window); (c) **loop throughput** — filed/applied/rejected/abstained,
     median proposed→disposition days, accept rate; (d) **model stamp** per run. Exit 0
   = computed; exit 1 = ≥1 loop-bug recurrence (the report is written either way).
   - **Current:** no library-level metric exists; per-skill tables only.
   - **Target:** the script + `test-scoreboard.mjs` (fixture inbox + fixture matrices
     covering: a recurrence, a new code, an `uncoded`, a corpus-change window, an empty
     window — the empty window reports "no activity," exit 0, not a silent file).
   - **Acceptance:** the fixtures produce the expected numbers (each hand-verifiable
     against the fixture); the exit code flips exactly when a recurrence is present;
     the written section renders the one-line verdict with all three metrics.
   - **Acceptance scenario:** `happy path scoreboard-computes` → `acceptance.feature`
3. **R3 — The corpus run is scheduled, not ad hoc:** the curation pass (Phase 1's
   procedure, edited by this phase) runs `scoreboard.mjs` as step 1.5 — after the status
   table, before candidate review — so every curation sitting sees the metrics *and* the
   status in one artifact; the corpus run (R2b) uses `run-evals.mjs` over the full
   scenario corpus with the model stamped; a pass rate drop triggers the "corpus changed
   vs skill changed" check *in the pass* (not just the report) — a skill change with a
   drop is a named finding against the edit's proposal (the gate's job, surfaced late is
   still surfaced).
   - **Current:** the curation pass (Phase 1) has no metrics step.
   - **Target:** the procedure edit (one step + the drop-check rule) + the pass's eval
     extended (the seeded fixture now includes a proposal whose applied edit dropped a
     fixture scenario → the pass names it).
   - **Acceptance:** the fixture pass output includes the scoreboard section and the
     named drop finding; the real curation pass run (this change's own ship) contains a
     real scoreboard section.
   - **Acceptance scenario:** `happy path curation-sees-metrics` → `acceptance.feature`
4. **R4 — The craft doc gains the code convention:** `_shared/craft/measurement.md`
   gains a short "Diagnosis codes" section (the slug rule, the class-not-instance rule,
   the exact-match v1 stance + the semantic-matching follow-up note, the `uncoded`
   state) — the contract, cited by the scoreboard script, `check-proposal.mjs`, and the
   curation pass (cite, don't restate); the per-skill two-indicator contract is
   untouched (this phase adds the *loop-level* pair: leading = eval pass rate on the
   fixed corpus; lagging = repeat-error rate — declared in the doc as the library's own
   table, satisfying the contract's "one skill, its own table" at the library level).
   - **Current:** the doc covers per-skill tables only.
   - **Target:** the section + the loop-level table (two rows: leading/lagging, data
     source = the scoreboard's own artifacts, direction = hold / fall).
   - **Acceptance:** the doc's loop-level table names the scoreboard as data source
     (a real artifact, per the contract) and the directions; no per-skill table is
     modified.
   - **Acceptance scenario:** `happy path loop-level-table` → `acceptance.feature`
5. **R5 — The before/after pilot (one-time baseline):** a documented task set = the
   full existing scenario corpus + the 11 skills' `evals.json` trigger cases (the
   corpus is the fixed set); run once against the pre-improvement-machinery tree (the
   commit before Phase 0's first merge — the "off" state: the loop's edits absent) and
   once against the current tree ("on"), same model (stamped), and once more with the
   scenario order swapped (the PATH-Bench caveat, one swap); results → the first
   "pilot" section of `scoreboard.md` with the honest framing: one repo, one maintainer,
   a baseline not a claim; the decision rule (viktor's) is recorded: *the loop is kept
   and scheduled only if the repeat-error rate falls without the pass rate dropping* —
   evaluated at the first real curation window after ship (owner: the maintainer, due:
   first scheduled pass).
   - **Current:** no off/on comparison exists for the machinery.
   - **Target:** the pilot run + the section + the decision rule with its owner/due.
   - **Acceptance:** the section contains both runs (model stamps present), the order
     swap's delta, and the decision rule verbatim with owner + due; the numbers are the
     runs' actual outputs (no hand adjustment — a corrected run is a new dated row).
   - **Acceptance scenario:** `happy path pilot-baseline-recorded` → `acceptance.feature`
6. **R6 — Docs close the loop:** `docs/04-lifecycle.md`'s "Improvement loop" section
   (Phase 2's) gains the measurement line (the two indicators + the cadence — the loop
   is now described end to end: evidence → reflection → verification → activation →
   *measurement* → next round, the Perplexity fifth arrow made explicit);
   `docs/05-commands.md` (the commands doc) lists `scoreboard.mjs` with its invocation.
   - **Current:** the loop doc has no measurement stage.
   - **Target:** both docs updated; the loop description is complete (a reader can name
   all five stages and the artifact of each).
   - **Acceptance:** a reader following `docs/04-lifecycle.md` can run every named
     stage's invocation from a clean shell (each executed once during this change's
     test phase).
   - **Acceptance scenario:** `happy path loop-docs-complete` → `acceptance.feature`

## Implementation Decisions

- **Files to create:** `scripts/scoreboard.mjs` + `scripts/test-scoreboard.mjs` (the
  repo-level script home, matching `state-lock.mjs`/`test-hooks.mjs`),
  `.skillgrid/skill-improvements/scoreboard.md` (created by the first run).
- **Files to modify:** `.skillgrid/skill-improvements/README.md` (the `diagnosis_code`
  field + the code rules pointer), `check-proposal.mjs` + its test (the shape check),
  `skill-creator/references/curation-pass.md` (step 1.5 + the drop-check + the code-
  consistency review), `_shared/craft/measurement.md` (the codes section + the
  loop-level table), `docs/04-lifecycle.md` + `docs/05-commands.md` (R6), the curation
  pass's fixture/eval (the drop-finding case).
- **Interfaces:** `scoreboard.mjs` input = the inbox files + git (corpus diff, applied-
  commit lookup) + the last `run-evals.mjs --json` output (the pass runs it fresh in
  step 1.5 — the script does not re-run the corpus itself: it *reports* the run the pass
  just did — the division keeps the model-bound part in the runner, as in Phase 1).
  Output: the dated markdown section + `--json` for machine use. The one-line verdict
  format is fixed (greppable across dated sections — the trend is a `grep` over
  `scoreboard.md`).
- **Data flow:** curation pass → `run-evals.mjs` (fresh corpus run, model stamped) →
  `scoreboard.mjs` (computes from inbox + that run + git) → `scoreboard.md` dated
  section (committed) → the pass's candidate review reads the recurrences (a loop-bug
  recurrence is a candidate with the fix's proposal named) → next round.
- **Error handling:** empty window → "no activity" section, exit 0; a proposal with
  `uncoded` → counted separately, listed (the pass chases it); the corpus run fails
  mid-window (a broken scenario file) → the section records "corpus run failed:
  <scenario_id>" and the pass rate is the last good value, flagged (the run's failure is
  itself a finding — a scenario that can't run is a corpus bug); a recurrence whose
  original fix has no resolvable commit (pre-git history, a manual edit) → the
  loop-bug finding says "fix origin unresolvable" (still surfaced, still exit 1 — an
  unexplained recurrence is the worst kind).
- **Dependencies:** Phases 0–3 as stated. The pilot (R5) additionally needs the
  pre-Phase-0 commit to still be reachable (it is — this repo's history; if the pilot
  runs long after a rebase, the "off" tree is the recorded commit SHA, not "main~N").

## Shootdown Findings

**Lens status:** all seven lenses **not run** — drafted from the Phase 0 plan + the
measurement sources; Goldfish at the spec gate.

**Anchored hypothesis (as proposed, pending user confirmation):** the loop's fifth
stage is *counting over its own artifacts* — a deterministic scoreboard with a leading
pair, a recurrence alarm, and a one-time pilot baseline — and the loop is declared
working only when the repeat-error rate falls without the pass rate dropping.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| — | (not run) | — | pending spec gate | — |

**Overall verdict:** pending — drafted, not yet shot down.

## Testing Decisions

- **What makes a good test:** the computation, hand-verifiable fixtures (each metric
  recomputable by eye from the fixture inbox); the exit-code contract; the verdict-line
  format stability (a grep over two dated sections extracts both numbers — the trend
  must be machine-readable). No model in `test-scoreboard.mjs` (the corpus run is
  Phase 1's runner's job; the scoreboard *reports* it — the test injects a fixture run
  output).
- **Modules to test:** `scoreboard.mjs` (recurrence match, new code, `uncoded`, corpus-
  change window, empty window, loop-bug exit code); the curation-pass extension (the
  drop-finding fixture); `check-proposal.mjs` (the code shape).
- **Prior art:** `scripts/test-state-lock.mjs` (repo-level script+test pairing);
  Phase 2's seeded fixture discipline; the `log.md` rotation precedent (the dated
  sections grow — the doc names a 500-entry-style rotation if the file ever outgrows
  comfortable size; not pre-built).
- **Edge cases:** two proposals, same code, one applied one rejected (the recurrence
  only counts against *applied* fixes — a rejected proposal's code is not a "fixed"
  class); a code backfilled at curation (its timestamp is the *filing* time, not the
  backfill time — the window math uses filing); the order-swap run disagreeing with the
  primary run (both recorded; the section says "order-sensitive: see both" — the
  PATH-Bench caveat in practice, not a failure); a model swap inside the window (the
  stamp shows it; the section splits the window at the swap line).

## Impact on Global Docs

- `.skillgrid/projectwiki/ASSUMPTIONS.md`: VERIFIED at ship — "the loop's headline
  metric is the repeat-error rate over diagnosis codes; the loop-level indicator pair
  is leading=eval pass rate, lagging=repeat-error rate."
- `.skillgrid/projectwiki/concepts/`: `diagnosis-code.md` (the class-not-instance rule +
  exact-match v1) — the concept behind R1/R4.
- `.skillgrid/projectwiki/decisions/`: `NNNN-scoreboard-deterministic.md` if the
  LLM-interpretation question is contentious at the gate (the Approach rejection is the
  candidate; default: concept page, no ADR).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.85  | 0.75 | ✓      | "is it working?" answered with a number |
| Boundary Clarity    | 0.85  | 0.70 | ✓      | library-level only; no dashboard; one-time pilot |
| Constraint Clarity  | 0.85  | 0.65 | ✓      | deterministic computation, fixed corpus, model stamped |
| Acceptance Criteria | 0.85  | 0.70 | ✓      | all six requirements falsifiable   |
| **Clarity**         | **0.15** | ≤0.20 | ✓ | from the plan + the sources' metric sections |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | (prior session) the loop must prove itself with cheap, artifact-based metrics | Phase 4 = scoreboard + pilot; the decision rule is the viktor one |
| 2     | (this session) specs for all phases | deterministic script, not LLM-computed; exit code is the alarm |

## Open Questions & Assumptions

| Type | Item | Owner | Due |
|------|------|-------|-----|
| Question | Window default for the scoreboard (7d / 30d)? Default: 30d (matches the demotion tier; a 7d window chatters). | user | spec gate |
| Question | Semantic code matching (embeddings via the mnemonic embedder) as a follow-up — confirm it stays out of v1. | user | blueprint gate |
| Assumption | The pilot's "off" tree = the commit before Phase 0's first merge (recorded SHA at pilot time). | devopstales | until disproved |
| Assumption | `scoreboard.md` is committed (the trend is a git-diffable artifact) — same stance as the sidecar. | devopstales | until disproved |

## Decisions (ADR)

None clears the bar by default (the deterministic-scoreboard choice is recorded as a
concept page; an ADR is filed only if the interpretation-lens question is contentious at
the gate — see Impact).

## Terms

- **Diagnosis code** (new): the stable failure-class slug on a proposal. Glossary at
  ship (the craft doc is the contract).
- **Loop-bug recurrence** (new): a recurrence against an *applied* fix — the fix or its
  proving eval is wrong. Glossary at ship.
- **Scoreboard** (new): the dated, deterministic loop-metrics artifact. Glossary at ship.

## Retrospective

**Status:** pending

- **Intent met?** [filled by ship]
- **Pre-mortem score:** [filled by ship]
- **Assumptions:** [filled by ship]
- **Timebox:** [filled by ship]
- **Next time:** [filled by ship]
