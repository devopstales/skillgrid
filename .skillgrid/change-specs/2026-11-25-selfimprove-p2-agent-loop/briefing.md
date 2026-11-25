# Self-Improving Loop — Phase 2: The improvement agent loop — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-25)
> **Author:** devopstales
> **Intent frozen:** no (pending) ← the agent sets this to "yes (YYYY-MM-DD)" once the STATUS banner above shows `approved` (the user's hand-edit IS the approval — the agent never writes `approved` itself); after that, the frozen block below is read-only for agent steps

> **Phase 2 of the self-improving-skills plan.** Depends on Phases 0 and 1 (inbox + problem
> reports + sidecar + `run-evals.mjs --replay` + curation pass). This phase is the loop's
> *brain*: a scheduled, human-triggered **improvement pass** that turns filed proposals and
> telemetry into tested skill edits — gated by recurrence, paired replay, deterministic
> checks, and an explicit user approval before anything lands.

## Problem / Intent

Phases 0–1 made evidence *arrive* (inbox, problem reports, user-experience blocks) and
checks *mechanical* (sidecar, replay, curation). What remains is the **reflection +
activation** half of the loop: something must read the evidence, decide which failures
justify an edit, draft that edit in the skill-creator discipline (Scope Bar + "match the form
to the failure"), verify it against the old version, and apply it only on human approval.
Today that is a human running `skill-creator` by hand — correct, but not a loop: it runs
only when the maintainer remembers to.

**Job to be done:** when open proposals exist (or a curation pass finds one worth
chasing), the maintainer wants a single invocation that produces a *verified, approvable*
edit per skill — with the evidence, the replay matrix, and the checks all attached — so the
only human act left is the approval (or the reject, with a recorded reason).

Why now: every input this phase consumes exists after Phase 1; the phase is the difference
between "a library that records its failures" and "a library that fixes them."

## Purpose & Success Criteria

- **Purpose:** a `skill-creator` mode — the **Improvement Pass** — that per invocation:
  consumes inbox proposals + sidecar + recent `## User Experience` / problem-report
  evidence; passes each candidate through the recurrence gate; drafts at most **one edit
  per skill per run** in the skill-creator form-matched style; verifies it (paired replay +
  the three deterministic checks + a Goldfish sample); and applies it only via the approval
  gate — as a work-unit commit (or draft PR) with backup + version bump.
- **Success criteria:**
  1. One invocation on a seeded inbox produces, per qualifying proposal: a draft edit file
     (the exact replacement text) + a replay matrix (revised passes all scenarios; original
     fails ≥1; controls unchanged) + check results (`check-frontmatter`, size budget,
     10-rule audit) + a one-paragraph evidence summary (what failed, what the edit is,
     why this form).
  2. Non-qualifying candidates are **abstained** with a recorded reason (recurrence not met;
     no verifiable signal; form unmatchable) — an abstention is a first-class output, not a
     silence.
  3. Nothing is applied without the approval gate: the pass presents the EGM screen (Done /
     Assumed / Next + reply menu) and the user's approval in the *current* message is the
     trigger; the apply step re-verifies the snippet is still exact-once before editing
     (stale → the proposal is reported stale, not force-applied).
  4. Applied edits: backup tarball of the skill dir *before* the edit (the never-delete
     invariant), `metadata.version` MINOR bump, `last_activity_at` reset semantics via the
     sidecar (the patch itself bumps it — the hook does this), proposal `status: applied`
     with `backup_path` + commit ref, and the work-unit commit body naming the proposal id
     (rollback = one `git revert` + the tarball).
  5. **Library cap:** the count of `created_by: agent` skills is bounded (default 5); at
     the cap, the pass may *update* agent-created skills but a *new* agent-created skill
     requires retiring one (named in the proposal) — the Perplexity retire-to-add rule.
     Human-created skills are never retired by the pass (provenance scope).
  6. The pass is **deterministic in its decisions**: given the same inputs, the gate
     outcomes (qualify / abstain) are recomputable — the gate logic is a script
     (`decide-proposals.mjs`) with unit tests; the model only does what it is good at
     (drafting the edit in the matched form). The deterministic-boundary rule, applied to
     the loop itself.
- **Out of scope:** *new* skill creation from raw sessions (the pass edits existing skills;
  creating a skill still goes through the full `skill-creator` four phases + a change spec —
  the pass may *propose* creation, which then gets a normal change); the user-project (L2)
  mode (Phase 3); metrics (Phase 4); auto-approval of any kind (the gate is permanent —
  the whole point of the sources' safety analysis is that the review gate is the
  mitigation); any daemon (the pass is a human-triggered invocation, like the curation pass).

## Context

- **Consumes (all shipped by Phases 0–1):** `.skillgrid/skill-improvements/` +
  `check-proposal.mjs`; `.agents/skills/.usage.json` (the hook);
  `run-evals.mjs --replay` + its `--json` matrix; the curation pass (the improvement pass
  is its *sibling*, not a replacement: curation manages *lifecycle state*, improvement
  manages *content*).
- **The discipline the pass drafts in:** `skill-creator` — Scope Bar (observed failure +
  composes-or-new), "match the form to the failure" table (failure shape → fix form:
  script / legwork / leading word / CRITICAL block / Anti-Patterns row / Red Flag /
  description / Output Contract), RED-GREEN gate, 10-rule audit, size budget. The pass does
  not restate this — it is a *mode* that walks the existing procedure on a proposal
  instead of an interview.
- **The isolation contract for the judge stage:** `_shared/craft/goldfish-protocol.md`
  (R36 freshness STOP, R37 four-part isolation) — the proposal's draft edit gets a
  fresh-context review by the Goldfish lens (the "LLM judge" stage), never by the same
  context that drafted it.
- **Upstream patterns adapted:**
  - BerriAI — the proposal's exact-once `original_snippet` + apply-after-confirmation +
    draft-PR option (already in the Phase 0 format; this phase *executes* it).
  - Perplexity Brain — one change per skill per run; abstain when evidence is thin;
    paired replay with held-out failure scenarios the reviser never sees (our analogue:
    the scenario that was RED is the held-out failure — the drafter sees it as a test, the
    replay proves it); the cap + retire-to-add.
  - viktor — recurrence gate (2–3× for a *new* rule; 1 verifiable failure suffices for an
    *edit*); verifiable signal before any rule; no self-re-ingestion (see R5).
- **Repo state:** `skill-creator` is the `external` size tier (90,000 B ceiling) — the new
  mode must fit or push its tail into `references/` (the skill's own pruning rule).

## User Context

- **Who:** the maintainer, at the scheduled curation moment (weekly/monthly per the doc) or
  on-demand after a cluster of failures.
- **Direct evidence:** the user's own instruction sequence — first "check the skills and
  plan how we can make the self improving," then "write change-specs for all phases" — is
  exactly the manual loop this phase automates: read evidence, decide, draft, verify,
  apply. The user does not want to stop doing it; they want the *decision surface* to be a
  short approval screen, not a research project.
- **Why it fails today:** between "proposal filed" (Phase 0) and "skill edited" (manual
  `skill-creator` session) there is a gap where proposals rot — the `status: proposed`
  state has no consumer with a schedule.

## Pre-mortem

- **Desirability:** the pass proposes edits the maintainer always rejects — the loop
  "works" and earns distrust. Countermeasure: the abstention discipline (thin evidence →
  abstain, recorded) biases the pass toward *fewer, stronger* proposals; the approval
  screen shows the replay matrix so a reject is one line against visible evidence, and
  rejects accumulate as data (Phase 4 measures the accept rate).
- **Feasibility:** paired replay needs a model per run — a 5-proposal pass may take
  minutes and tokens. Countermeasure: the pass is *scheduled*, not in-line; the replay
  matrix is the only model-bound part (drafting is one model call per proposal); a
  `--fast` mode skips replay for prose-only edits with the skip recorded (replay is
  mandatory for edits to skills with `scenarios.yaml`).
- **Viability:** if the pass edits a skill the maintainer is mid-change on, the
  exact-once snippet check fights concurrent work. Countermeasure: the pass refuses to
  draft against a skill whose `SKILL.md` has uncommitted changes (reported, deferred to
  the next run) — the worktree-aware repo already has the machinery to detect this.

## Timebox

Target: within three weeks (2026-11-25 → 2026-12-16). If exceeded: ship the gate script
(R1) + the draft/verify pipeline (R2) with *manual* application (the EGM screen + human
applies via `skill-creator`); the automated apply (R3) + cap (R5) become the follow-up
tickets of the same change. The recurrence gate never slips — it is the loop's filter.

## Approaches Considered

| Approach | Key trade-off | Verdict |
|----------|---------------|---------|
| A **mode** of `skill-creator` (Improvement Pass, `references/improvement-pass.md` + gate script) | `skill-creator` grows toward its tier ceiling; two owners of "library tooling" collapse into one | **Chosen** — the pass *is* skill-creator walking its own procedure on proposals; one owner, one discipline |
| A new `skill-tuner` meta-skill | Clean separation, but restates the four phases / form table (the repo's DR Y rule forbids); a second meta-skill is a second convention | Rejected |
| A background "Dream agent" (Perplexity's fleet) | The doc's no-daemon invariant + this repo's human-gate culture say scheduled | Rejected as a daemon; the *scheduled invocation* is the Dream agent, triggered by a human (or a cron that *presents*, never applies) |
| Let the curation pass also improve content | One invocation does two jobs; the curation pass's output (state transitions) and the improvement pass's output (content edits) have different risk profiles and different gates | Rejected — siblings, not one |

- **Chosen:** `skill-creator` Improvement Pass mode + `decide-proposals.mjs` gate script.
- **Rejected by author:** `skill-tuner` (DR Y; a second meta-skill is a second convention).

## Requirements

1. **R1 — Recurrence + signal gate (deterministic):**
   `skill-creator/scripts/decide-proposals.mjs` reads the inbox (`status: proposed|deferred`)
   + the sidecar and emits, per proposal: `qualify` | `abstain(<reason>)`. Rules (encoded,
   unit-tested): (a) an *edit* to an existing skill qualifies with ≥1 verifiable signal
   (a `test`/`eval` signal class with RED output, a `correction` with a verbatim user
   quote, or a problem report with a verifiable `signal`); (b) a *creation* proposal
   qualifies only with ≥2 independent observed instances (the skill-creator Scope Bar's
   "observed failure," made numeric, per the Perplexity two-occurrence rule); (c) an
   `observation`-only proposal abstains (`no verifiable signal`); (d) a proposal whose
   `original_snippet` no longer matches exactly once abstains (`stale`); (e) `created_by:
   human` targets qualify for *edits* but the proposal is flagged `human-skill` (the
   approval screen shows the flag; the pass never lowers the bar for them). The script is
   pure (input → JSON decisions); the model never overrides an abstention in the same run
   (an override is a human act, recorded as a new proposal with `recurrence: override`).
   - **Current:** no gate; every proposal is equally "next."
   - **Target:** the script + `test-decide-proposals.mjs` (fixture inbox covering a–e).
   - **Acceptance:** the fixture inbox produces the expected qualify/abstain table with
     reasons; changing one fixture field flips exactly the expected decision.
   - **Acceptance scenario:** `happy path gate-qualifies-on-verifiable-signal` → `acceptance.feature`
2. **R2 — Draft + verify pipeline (one edit per skill per run):** the Improvement Pass
   mode (`skill-creator/references/improvement-pass.md` + a SKILL.md section): for each
   qualifying proposal, **grouped by target skill** (one edit per skill per run — if two
   proposals target the same skill, the pass drafts the one edit that closes both, or
   abstains the weaker with the reason `same-skill-budget`); the draft (a) restates the
   failure class from the evidence, (b) picks the fix form from the skill-creator
   "match the form to the failure" table, (c) writes the replacement text for the exact
   snippet, (d) runs RED-GREEN: the held-out scenario (the one that was RED) must fail on
   the old text and pass on the new — via `run-evals.mjs --replay` — plus all existing
   scenarios unchanged; (e) runs the three deterministic checks (`check-frontmatter`,
   size budget, 10-rule audit) on the post-edit tree; (f) dispatches the Goldfish lens
   (R37 isolation, the draft is the artifact under review) as the LLM-judge stage; (g)
   writes the draft to `.skillgrid/skill-improvements/PROP-<n>.draft.md` (beside the
   proposal, not in the skill dir — the skill is untouched until apply). Output per
   proposal: the draft + replay matrix + check results + judge notes + a one-paragraph
   evidence summary.
   - **Current:** the same steps exist as `skill-creator`'s manual procedure; nothing runs
     them on a proposal.
   - **Target:** the reference + the mode wired to R1's output; the pass's own eval
     (a seeded fixture library + inbox) exercises one full qualify→draft→verify cycle.
   - **Acceptance:** on the fixture, the produced `PROP-<n>.draft.md` carries all five
     artifacts (draft, matrix, checks, judge notes, summary); a proposal whose held-out
     scenario does not go RED on the old version is reported `replay-inconclusive` (the
     scenario was not a real failure — the evidence is wrong), not applied.
   - **Acceptance scenario:** `happy path draft-verified-against-replay` → `acceptance.feature`
3. **R3 — Approval-gated apply:** the pass ends with the EGM screen (the repo's
   user-gate format: Done / Assumed / Next + reply menu), one section per draft: the
   evidence summary, the diff, the matrix, the checks, the `human-skill` flag where set.
   The user's approval in the current message triggers apply **per proposal** (the user
   can approve one and reject another — the menu says so). Apply step, in order: re-check
   `original_snippet` exact-once against the *current* tree (stale → reported, not
   applied); backup tarball of the skill dir → `.skillgrid/artifacts/` (path recorded in
   the proposal's `backup_path`); write the edit; bump `metadata.version` MINOR; set
   proposal `status: applied` + the commit ref after the work-unit commit; the commit
   body names `PROP-<n>` (the work-unit commit contract). Reject → `status: rejected` +
   the user's reason in `disposition_note`. A draft-PR variant (push branch + PR, BerriAI
   style) is the option when the maintainer is not at the machine to commit — same gates,
   different sink; the PR body is the EGM screen content.
   - **Current:** no apply path; the proposal format has the fields but no executor.
   - **Target:** the apply sub-procedure in the reference + the EGM screen format for the
     pass (a canonical payload example, per the EGM gate rules in
     `skill-creator/references/interaction-gates.md`).
   - **Acceptance:** on the fixture, approving one of two drafts applies exactly one
     (version bumped, backup exists, proposal `applied` with `backup_path`, commit names
     the proposal); the other stays `proposed` until its own approval; a stale snippet
     (fixture: the target file edited after drafting) is reported stale and untouched.
   - **Acceptance scenario:** `happy path approval-applies-one-rejects-the-other` → `acceptance.feature`
4. **R4 — No self-re-ingestion (anti-poisoning):** the pass's *evidence inputs* are the
   inbox, the sidecar, problem reports, and `## User Experience` blocks — and **not** the
   pass's own prior drafts/reports or the agent-written parts of `report.md` retro halves
   (which are agent self-assessment). A proposal may cite a prior *applied* proposal's
   commit as evidence (that is a verifiable artifact), but never a prior *draft* or an
   unapplied abstention. The rule is written in the reference as a Red Flag row: "the
   pass justifying an edit with its own previous output" (the OWASP self-re-ingestion
   line via the viktor source).
   - **Current:** no such rule exists; the first pass runs on human-filed evidence, the
   second would run partly on its own.
   - **Target:** the rule in the reference + a scenario eval (the pass given a fixture
     inbox whose only "evidence" is a prior draft abstains with `self-re-ingestion`).
   - **Acceptance:** the scenario eval is RED without the rule, GREEN with it (the
     repo's own scenario-evals RED-GREEN discipline, applied to the pass itself).
   - **Acceptance scenario:** `happy path self-reingestion-abstained` → `acceptance.feature`
5. **R5 — Library cap on agent-created skills:** the pass computes `count(created_by:
   agent, state: active)` against the cap (default **5**, configurable in
   `.skillgrid/config.yaml` under a new `lifecycle: { agent_skill_cap: 5 }` key — the
   config already owns per-project policy). Below the cap: creation proposals proceed
   (into a normal change spec, R2's grouping note). At the cap: a creation proposal
   carries a required `retires: <skill>` field naming an *agent-created* skill to
   archive (backup + transition per the lifecycle rules) in the same apply; the retire
   is shown on the EGM screen with the justification. Human-created skills are never a
   `retires` value.
   - **Current:** no cap; the curation pass archives on age, not count.
   - **Target:** the config key (documented in `docs/04-lifecycle.md`'s new
     "Improvement loop" note), the count in the pass's status output, the retire-to-add
     rule in the reference.
   - **Acceptance:** fixture library at the cap → a creation proposal without `retires`
     is reported `cap-reached` (abstain, not silent fail); with `retires` naming a
     human skill → rejected with the provenance rule named; with a valid agent-skill
     retire → both transitions happen with backups, count returns to the cap.
   - **Acceptance scenario:** `happy path cap-enforced-retire-to-add` → `acceptance.feature`
6. **R6 — Docs + the pass's own evals:** `docs/04-lifecycle.md` gains the
   "Improvement loop" section (the four stages mapped to the shipped pieces: evidence =
   inbox/sidecar/UX blocks, reflection = the pass, verification = replay+checks+judge,
   activation = the gate + apply; the daemon prohibition restated); `skill-creator`'s
   description is checked for trigger coverage ("improvement pass" wording, per its own
   §9 description-optimization rule); the pass's scenario evals (R2, R4) live in
   `skill-creator/evals/`.
   - **Current:** the doc describes the loop aspirationally; the skill's description does
     not name an improvement pass.
   - **Target:** the doc is true (every named piece exists and is invocable as written);
     the description's 20-query trigger check (10 should-trigger incl. "run the
     improvement pass," 10 near-miss should-not) passes.
   - **Acceptance:** the doc's section names only shipped invocations (each run once);
     the trigger check report is in the change folder.
   - **Acceptance scenario:** `happy path docs-map-the-loop` → `acceptance.feature`

## Implementation Decisions

- **Files to create:** `skill-creator/references/improvement-pass.md` (the procedure:
  gate → group → draft → verify → screen → apply; the EGM payload example; the Red Flag
  rows incl. self-re-ingestion), `skill-creator/scripts/decide-proposals.mjs` +
  `test-decide-proposals.mjs`, fixture library for the pass's eval
  (`.agents/skills/skill-creator/evals/fixtures/` — toy skills + a toy inbox, the same
  seeded-fixture pattern as Phase 1's curation-pass test), the pass's scenarios in
  `skill-creator/evals/scenarios.yaml` (new file for that skill).
- **Files to modify:** `skill-creator/SKILL.md` (Contents + Improvement Pass section;
  description trigger wording; size-budget check — the mode's tail stays in the
  reference), `.skillgrid/config.yaml` (the `lifecycle.agent_skill_cap` key, documented),
  `docs/04-lifecycle.md` (the loop section), `.skillgrid/skill-improvements/README.md`
  (the `status` lifecycle gains the pass's transitions: `approved`/`applied` written by
  the pass's apply step — the README already reserves these values; this makes them true).
- **Interfaces:** `decide-proposals.mjs` output: `{ proposals: [{ id, target_skill,
  decision: qualify|abstain, reason?, flags?: [human-skill, cap-reached] }] }` (pure,
  no I/O beyond reading). The draft artifact `PROP-<n>.draft.md` sections (fixed,
  greppable): `## Evidence Summary` / `## Failure Class & Fix Form` / `## Draft Edit` /
  `## Replay Matrix` / `## Checks` / `## Judge Notes`. The EGM screen reuses the
  existing user-gate contract (one screen, Done/Assumed/Next + menu) — no new format.
- **Data flow:** `decide-proposals.mjs` (gate) → per-skill groups → model drafts (one call
  per group) → `run-evals.mjs --replay` (held-out + controls) → three checks → Goldfish
  lens (fresh context, R37) → `PROP-<n>.draft.md` → EGM screen → user approval (per
  proposal) → re-verify snippet → backup → edit + bump → work-unit commit → proposal
  `applied`.
- **Error handling:** replay-inconclusive (held-out scenario wasn't actually RED) →
  reported, proposal stays `proposed` with the note; judge FAIL → the draft is not
  screened (the pass records the judge's named defect; a re-draft is the next run's job —
  no fix-loop within one run: one run, one shot, then schedule); apply-time staleness →
  `deferred` + reason; a run with zero qualifying proposals → the screen says "nothing
  qualifies" with the abstention table (a run is still a run — the abstentions are the
  output).
- **Dependencies:** Phases 0+1 as stated. The model call for drafting/replay uses the
  harness's model (the runner's turn-executor interface from Phase 1); the Goldfish
  dispatch uses the existing subagent mechanism (R36 STOP if absent — the pass degrades to
  *no judge stage* and says so, per the protocol's non-degradation rule: a same-context
  "review" is not a review).

## Shootdown Findings

**Lens status:** all seven lenses **not run** — drafted from the Phase 0 plan + the source
study; Goldfish at the spec gate.

**Anchored hypothesis (as proposed, pending user confirmation):** the loop's missing half
is *deciding* — and a decision that is gate-scripted (deterministic) + form-matched
(discipline) + replay-proven (evidence) + approval-gated (safety) is one the maintainer
will actually trust to run on a schedule.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| — | (not run) | — | pending spec gate | — |

**Overall verdict:** pending — drafted, not yet shot down.

## Testing Decisions

- **What makes a good test:** the gate script's decision table (pure, exhaustive fixture
  inbox); the apply step's ordering invariants (backup-before-edit, re-verify-before-edit,
  bump-on-edit) on a temp fixture library; the two scenario evals (R2, R4) via the Phase 1
  runner. Model-bound behavior (draft *quality*) is judged by the Goldfish lens + the
  maintainer, not unit-tested — the unit tests pin the *process*, the lens pins the
  *product*.
- **Modules to test:** `decide-proposals.mjs` (rules a–e + cap interaction); the apply
  sub-procedure (a `test-improvement-apply.mjs`-style fixture: approve-one/reject-other,
  stale snippet, cap-reached); the pass's scenarios (`scenarios.yaml` for
  `skill-creator`).
- **Prior art:** Phase 1's seeded-fixture curation test; `execute`'s
  `evals/scenarios.yaml` (the scenario format in production use); `ship-gates.mjs`
  (gate-script shape: read state → emit decisions, no side effects).
- **Edge cases:** two proposals, same skill (grouping, `same-skill-budget`); a proposal
  arriving mid-run (the run snapshots the inbox at start — new files are next run);
  approval of a draft whose *sibling* was just applied (re-verify runs against the new
  tree — the sibling may have shifted the snippet; exact-once re-check catches it); cap
  reached *and* a retire's backup fails (the apply aborts before any mutation — the
  backup is the first mutation step); Goldfish absent (R36 STOP: judge stage reported
  missing, the screen flags it, approval still possible — the human decides).

## Impact on Global Docs

- `.skillgrid/projectwiki/ASSUMPTIONS.md`: VERIFIED at ship — "the improvement pass is a
  `skill-creator` mode; the gate is deterministic; the approval gate is permanent; agent-
  created skills are capped."
- `.skillgrid/projectwiki/concepts/`: `improvement-loop.md` — the four stages mapped to
  shipped pieces (the loop is a new durable concept of the repo).
- `.skillgrid/projectwiki/decisions/`: `NNNN-improvement-pass-as-skill-creator-mode.md`
  (the Approach rejection of `skill-tuner` clears the ADR bar: real trade-off, hard to
  reverse — a second meta-skill would be a permanent convention).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.90  | 0.75 | ✓      | the loop's brain, named per stage  |
| Boundary Clarity    | 0.85  | 0.70 | ✓      | creation→change spec; L2→Phase 3; no auto-approve |
| Constraint Clarity  | 0.85  | 0.65 | ✓      | one-edit-per-skill, gate-scripted, no-daemon, cap |
| Acceptance Criteria | 0.85  | 0.70 | ✓      | all six requirements falsifiable   |
| **Clarity**         | **0.14** | ≤0.20 | ✓ | from the plan + Phases 0–1 contracts |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | (prior session) the agent loop needs evidence, replay, and an approval gate | Phase 2 = the pass; gate is a script |
| 2     | (this session) specs for all phases | pass is a `skill-creator` mode; cap default 5; retire-to-add |

## Open Questions & Assumptions

| Type | Item | Owner | Due |
|------|------|-------|-----|
| Question | Cap default 5 — confirm the number (the Perplexity figure is 10 *per user* auto-created skills; this repo's library is 19 hand-written, so 5 agent-created is the conservative reading). | user | spec gate |
| Question | Draft-PR sink: needs the repo's remote + a token (BerriAI's env-var pattern). Available in this environment? If not, the PR variant is specified but untested (local apply is the tested path). | user | blueprint gate |
| Assumption | One model call per skill-group for drafting is sufficient (no multi-draft competition in v1 — the Goldfish judge replaces the "design it twice" luxury). | devopstales | until disproved |
| Assumption | The pass snapshots the inbox at run start (no mid-run proposals). | devopstales | until disproved |

## Decisions (ADR)

- `NNNN-improvement-pass-as-skill-creator-mode.md` — the mode-over-new-skill choice
  (rationale in Approaches; the DR-Y argument).
- (Candidate) `NNNN-deterministic-gate-for-improvement.md` — the gate is a script, the
  model never overrides in-run: clears the bar if the spec gate confirms "override is a
  human act, recorded as a new proposal" (that is the load-bearing, surprising part).

## Terms

- **Improvement Pass** (new): the `skill-creator` mode that turns qualifying proposals
  into verified, approvable skill edits. Glossary at ship.
- **Retire-to-add** (new, from Perplexity): at the cap, a new agent-created skill requires
  archiving one. Glossary at ship.
- **Replay-inconclusive** (new): the held-out scenario was not actually RED on the old
  version — the evidence, not the edit, is in question. Glossary at ship.

## Retrospective

**Status:** pending

- **Intent met?** [filled by ship]
- **Pre-mortem score:** [filled by ship]
- **Assumptions:** [filled by ship]
- **Timebox:** [filled by ship]
- **Next time:** [filled by ship]
