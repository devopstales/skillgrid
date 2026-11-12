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

## Section order (R19)

The drafting order, verbatim from the spec: Header → Hypothesis →
Must-Haves → Threat Matrix (if applicable) → File Structure → Global
Constraints → Tasks 1..N → Self-Review → Commit.

One line per section, with its content contract (point at the SKILL.md
section — cite, don't restate):

- **Header** — the plan header block; contract in `SKILL.md` →
  "Plan header and task structure" + `templates/blueprint.md`
- **Hypothesis** — `SKILL.md` → "Hypothesis"
- **Must-Haves** — `SKILL.md` → "Must-Haves (Goal-Backward Verification)"
- **Threat Matrix (if applicable)** — `SKILL.md` → "Threat Matrix"
- **File Structure** — `SKILL.md` → "File Structure"
- **Global Constraints** — the header scaffold's `## Global Constraints`
  block in `templates/blueprint.md` (values copied verbatim from the
  spec; inherited rules cited, not restated, per
  `_shared/craft/cite-dont-restate.md`)
- **Tasks 1..N** — `SKILL.md` → "Task Right-Sizing" +
  `references/task-structure.md` (Files / Interfaces / SATISFIES / Steps
  with code)
- **Self-Review** — `SKILL.md` → "Self-Review" (including the Spec
  Reconciliation check, `## Spec Reconciliation (R21)` below)
- **Commit** — commit the change folder before advancing (spec zone rule)

This is the order sections are drafted and gated; the final document's
layout follows `templates/blueprint.md`.

## Per-section gate (R19)

Artifact-first (R25): write the section to `blueprint.md` (the file is
the deliverable — never paste 200 lines into chat). Then present in the
same chat turn: which section was drafted, the 2–3 key decisions in it
(one line each), and the path to `blueprint.md` — the chat summary
precedes the gate (no empty chat bubbles). Then the single-select
approval gate:

- **"Approve [Section] and proceed"**
- **"Request revisions to [Section]"**

The user can revisit any earlier section at any point.

## Per-task gate (R20)

Each task in the Tasks section gets its own approval gate (the section
gate at task granularity): the agent writes Task 1, presents the gate,
the user approves, then the agent writes Task 2. The task template
(Files / Interfaces / SATISFIES / Steps with code) is structurally
unchanged — only the approval cadence changes. This is the
highest-value addition of Track C: it catches a wrong interface, a
missing file, or a bad test strategy *before* the agent writes 15 more
tasks.

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
from EGM's `egm-draft`. One line each: what it is + where it lives (this
file: R18–R21 protocol; SKILL.md sections: R22–R25 per Task 27). This
section is an index, not a restatement.

1. **Explicit ingestion (R18)** — read the briefing, confirm context
   before drafting; protocol: `## Ingestion (R18)` (this file)
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
