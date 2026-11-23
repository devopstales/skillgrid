# Section-by-Section Drafting (Track C)

How every blueprint (including the next change's) gets built: drafted
section by section, gated per section and per task, reconciled against
the briefing before commit. Replaces one-shot drafting with a single
end-of-file review — the failure mode Track C exists to kill.

## Ingestion (R18)

Before writing any section:

1. read `briefing.md` in full
2. read `findings.md` if present
3. read `adr.md` if present
4. present a 2–3 sentence summary of the briefing's key decisions to the
   user
5. confirm the draft starts from the right context

Light-weight — the briefing already exists; the step makes ingestion
explicit and adds the confirmation gate.

## Hypothesis validation (R26)

After ingestion, before the first byte of `blueprint.md`:

1. state the plan-level hypothesis you are about to encode — the claim
   (the approach, the key technical assumptions, the first-wave build
   shape), derived from the briefing and findings
2. name the falsifiers — the observable results that would prove the plan
   wrong
3. name the thinning — the smallest slice that would test the claim
4. put each to the user as a decision through the structured answer tool
   (per `skillgrid:questioning` frontier rounds: options with trade-offs,
   one at a time, each with your recommendation)
5. the user accepts, overrides, or amends; the answers are the input to
   the Hypothesis section
6. ADR capture — in the same step, every validated decision that clears
   the ADR-sparingly test (`_shared/adr-records/README.md`) is written to
   `{projectwiki}/decisions/NNNN-slug.md` (monotonic number,
   frontmatter, body per the config's `adr_style` template) plus its
   pointer row in the `## LOCKED` ADR index in
   `{projectwiki}/ASSUMPTIONS.md`; a decision that contradicts an in-force
   ADR gets a new superseding ADR (`supersedes:` frontmatter, the prior
   file untouched — IRON RULE) and the prior index row flips to
   "In force: no"

The hypothesis the user has not seen is an unvalidated assumption in the
plan, not a decision. R18 confirms the user and you share the *briefing's*
context; R26 puts the *plan's* core claim to them — and records its
durable decisions as ADRs — before the file exists.

## Section order (R19)

The drafting order: Phase 1 (Research) → Header → Hypothesis →
Key Decisions → Must-Haves → Validation → Threat Matrix (if applicable)
→ Failure Handling (if applicable) → File Structure → Global
Constraints → Rollout & Rollback → Tasks 1..N → Open Questions → Phase
3 (Project Wiki) → Phase 4 (Prototype, if the user opted in) →
Self-Review → Phase 5 (Commit).

Phase 1 runs before the first section (it feeds the draft); Phase 3 and
Phase 4 run after the Tasks section (they persist and de-risk what the
draft pins down) and before Self-Review / Commit. The `blueprint.md`
sections below are the ones drafted and gated in the file; the phases are
working steps with their own gates (research plan approval, page-home
confirmation + no-new-findings, prototype opt-in + hypothesis nod + verdict
report).

One line per blueprint.md section, with its content contract (point at the
SKILL.md section — cite, don't restate):

- **Header** — the plan header block; contract in `SKILL.md` →
  "Plan header and task structure" + `templates/blueprint.md`
- **Hypothesis** — `SKILL.md` → "Hypothesis"
- **Key Decisions** — the `## Key Decisions` block in
  `templates/blueprint.md` (rows sourced from the briefing's
  `Approaches Considered` + `Decisions (ADR)` — R23: cite, don't re-derive)
- **Must-Haves** — `SKILL.md` → "Must-Haves (Goal-Backward Verification)"
- **Validation** — `SKILL.md` → "Validation Matrix" (one row per in-scope
  requirement: happy / edge / failure / target / check / falsifier)
- **Threat Matrix (if applicable)** — `SKILL.md` → "Threat Matrix"
- **Failure Handling (if applicable)** — the `## Failure Handling` block in
  `templates/blueprint.md` (applicable rows only; omitted if the change
  adds no failure path)
- **File Structure** — `SKILL.md` → "File Structure"
- **Global Constraints** — the header scaffold's `## Global Constraints`
  block in `templates/blueprint.md` (values copied verbatim from the
  spec; inherited rules cited, not restated, per
  `_shared/craft/cite-dont-restate.md`)
- **Rollout & Rollback** — the `## Rollout & Rollback` block in
  `templates/blueprint.md` (one rollback row per one-way-door decision)
- **Tasks 1..N** — `SKILL.md` → "Task Right-Sizing" +
  `references/task-structure.md` (Files / Interfaces / SATISFIES / Steps
  with code)
- **Open Questions** — the `## Open Questions` block in
  `templates/blueprint.md` (home for the R21 "addition" items; owner +
  due, or "None")
- **Self-Review** — `blueprint-plan-spec.md` → "Self-Review" (two mechanical
  checks: Spec Reconciliation per `## Spec Reconciliation (R21)` below, and
  Placeholder scan per `task-structure.md` → "No Placeholders"). Quality
  judgments are the Goldfish pipeline's job.
- **Commit (Phase 5)** — reconcile against the briefing, re-run the final
  Self-Review, then commit the change folder before advancing (spec zone
  rule)

The phase working steps, with their own gates:

- **Phase 1 — Research (delegated to `skillgrid:research`)** (before the first
  section) — `SKILL.md` → "Phase 1 — Research (delegated)". Gate: research
  plan approval (fan-out) or the handoff report (single-pass). Output: the
  `## Research:` section in `findings.md`.
- **Phase 3 — Project Wiki** (after Tasks, before Self-Review) — `SKILL.md` →
  "Phase 3 — Project Wiki". Gate: page-home confirmation + the
  `projectwiki index` → `validate` → `lint` → `status` no-new-findings gate.
  Output: durable pages in `.skillgrid/projectwiki/`.
- **Phase 4 — Prototype (delegated to `skillgrid:prototype`, optional)** (after
  Phase 3, before Self-Review) — `SKILL.md` → "Phase 4 — Prototype
  (delegated, optional)". Gate: the user's opt-in (skip to Phase 5 if opted
  out), the hypothesis nod, and the verdict report. Output: the prototype
  directory + the `## Prototype:` section in `findings.md`; the feedback loop
  re-runs the affected blueprint.md sections.

This is the order sections are drafted and gated; the final document's
layout follows `templates/blueprint.md`.

## Per-section gate (R19)

Artifact-first (R25): write the section to `blueprint.md` (the file is
the deliverable — never paste 200 lines into chat). Then present in the
same chat turn, in this strict order:

**First: the visible chat summary** — before any question tool call:

- **Section link:** the path to `blueprint.md`, visible, before the
  decision question (e.g. "Drafted **Key Decisions** in
  [blueprint.md](.skillgrid/change-specs/2026-11-23-foo/blueprint.md)").
  No empty chat bubbles: the question gate is never the first thing the
  user sees.
- **The section's concept (1–2 sentences, plain English):** what this
  section is for and how it fits the blueprint — so the user can judge
  the section against the plan, not just the section in isolation.
- **The 2–3 key decisions in it, each presented for judgment, not as
  facts:**
  - **Mechanism** — how the decision works (1 line, no jargon, no
    "robust"/"seamless" glaze).
  - **Advantages / disadvantages** — at least the disadvantages, and the
    advantages when they are not self-evident. A decision with only an
    advantage list is a recommendation, not a decision point.
  - **When it fails** — the conditions under which the decision is wrong
    (per the stress-test default: lead with what is wrong, missing, or
    risky — `_shared/craft/review-discipline.md`).
- **Open choices in the section:** any load-bearing choice in the section
  with ≥2 defensible options goes to the user (owner-decides-architecture,
  R22) using the R22 **Option table** — up to three options, each row:
  Mechanism / Advantages / Disadvantages / Fails-when (the table format
  lives in `blueprint-plan-spec.md` § Owed-Decision Gate, "Presenting the
  options (R22)" — cited, not restated). No ranking, no default, no pick.
  Mechanical choices with one defensible answer (naming, formatting) are
  decided silently.

**Second: the single-select approval gate** — same turn, strictly one
question, zero tool calls after it (turn ends and waits):

- **"Approve [Section] and proceed"**
- **"Request revisions to [Section]"**

**Empty write-in guard:** if "Request revisions" is chosen with no
write-in text and no inline comments on `blueprint.md`, do not loop or
fail — ask in chat what to adjust and yield the turn.

The user can revisit any earlier section at any point.

### Gate presentation — paired example

The format is a checkable contract (per
`_shared/craft/review-discipline.md` paired-example format):

**❌ the failure** — a summary that tells the user what was decided:

> Drafted **Storage** in blueprint.md. Decisions: (1) Postgres for the
> event log, (2) 30-day retention, (3) JSONB payload column.
> Approve and proceed?

**✅ the fix** — the same section, presented for judgment:

> Drafted **Storage** in
> [blueprint.md](.skillgrid/change-specs/2026-11-23-events/blueprint.md).
> This section is the event log's storage model — everything downstream
> (replay, audit, retention) hangs off it.
>
> 1. **Postgres for the event log.** Mechanism: one table, append-only,
>    partitioned by month.
>    Advantage: no new service, transactions with the rest of the app.
>    Disadvantage: append-only at high write rate pressures the vacuum
>    horizon; no cheap replay to a second store.
>    Fails when: event volume > ~1M rows/day, or when a consumer needs
>    the log outside Postgres.
> 2. **30-day retention.** Mechanism: cron drops partitions older than
>    30 days. Disadvantage: the audit trail disappears with it — the
>    briefing's audit requirement assumed 90 days. Fails when: compliance
>    audit needs a 2nd look back.
> 3. **JSONB payload column.** Mechanism: schema-less payload, queried by
>    index on `event_type`. Disadvantage: no schema enforcement at write
>    time — a bad producer corrupts rows silently. Fails when: consumers
>    need field-level guarantees.
>
> Approve **Storage** and proceed to **Validation**? / Request revisions?

**The one-line test:** could the user approve the section without knowing
what any of the decisions costs? If yes, the gate is a rubber stamp —
present the mechanism, the disadvantage, and the failure condition for
each decision.

## Per-task gate (R20)

Each task in the Tasks section gets its own approval gate (the section
gate at task granularity): the agent writes Task 1, presents the gate,
the user approves, then the agent writes Task 2. The task template
(Files / Interfaces / SATISFIES / Steps with code) is structurally
unchanged — only the approval cadence changes. This is the
highest-value addition of Track C: it catches a wrong interface, a
missing file, or a bad test strategy *before* the agent writes 15 more
tasks.

The gate summary follows the R19 presentation format: the section link,
what the task does (1–2 sentences), and each design choice the task makes
(interface, test strategy, file layout) with mechanism / disadvantage /
failure condition — presented for judgment, not as facts.

## Spec Reconciliation (R21)

The full protocol for the Self-Review check Task 27 adds to SKILL.md:
re-read `briefing.md` and compare it against the blueprint.

- **Contradiction:** the briefing decided a question and the blueprint
  decides it differently — report every one, quoting the briefing and
  the blueprint.
- **Addition:** the blueprint decides a question the briefing did not
  raise — report it only if it changes the architecture, an external
  interface, or a dependency.

List contradictions first. Do NOT change the blueprint to match the
briefing on your own: the blueprint can be right, because drafting
reads code that the brainstorm did not. If nothing diverges, say so in
one line. This is the check "no Goldfish can do, because no Goldfish
reads the brief."

## The 8 drafting disciplines

Track C's eight disciplines ARE the eight Track C requirements, ported
from EGM's `egm-draft`, plus R26 (hypothesis validation — the plan's core
claim put to the user before the file exists). One line each: what it is +
where it lives (this file: R18–R21, R26 protocol; SKILL.md sections:
R22–R25 per Task 27). This section is an index, not a restatement.

1. **Explicit ingestion (R18)** — read the briefing, confirm context
   before drafting; protocol: `## Ingestion (R18)` (this file)
1. **Hypothesis validation (R26)** — the plan's claim, falsifiers, and
   thinning put to the user as decisions before the first byte of
   `blueprint.md`; protocol: `## Hypothesis validation (R26)` (this file)
2. **Section-by-section drafting (R19)** — fixed drafting order,
   per-section gate; protocol: `## Section order (R19)` + `## Per-section
   gate (R19)` (this file)
3. **Per-task gate (R20)** — approval at task granularity; protocol:
   `## Per-task gate (R20)` (this file)
4. **Spec reconciliation (R21)** — Self-Review check against the
   briefing; protocol: `## Spec Reconciliation (R21)` (this file)
5. **Owner-decides-architecture (R22)** — architecture questions go to
   the user, not the agent; SKILL.md section per Task 27 (Owed-Decision
   Gate)
6. **Alternatives-from-briefing (R23)** — alternatives come from the
   briefing's decision record; SKILL.md section per Task 27
7. **Anti-sycophancy + plain-English (R24)** — drafting tone and
   discipline; SKILL.md section per Task 27 (Tone & Discipline)
8. **Artifact-first delivery (R25)** — the file is the deliverable,
   chat carries the summary; SKILL.md section per Task 27
