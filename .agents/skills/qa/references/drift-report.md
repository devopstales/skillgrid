# Drift Report Workflow

The drift protocol qa's Drift mode runs (Task 31 exposes it as qa's Drift-mode
section). It compares the codebase against the documented design and reports
only actionable discrepancies. This reference set holds the protocol so that
section stays within its line budget (Size rule below).

## Invocation

- **Per-change**: `drift-report <spec-dir>` — the target is one spec dir under
  `.skillgrid/change-specs/`.
- **Project-wide**: `drift-report --project` — no target; the scan covers all
  of `.skillgrid/change-specs/` plus the repo-level design artifacts below.

## Target artifacts (R28)

The design baseline to compare the code against:

- **Per-change** (from the spec dir): `briefing.md` (requirements,
  Must-Haves), `blueprint.md` (file structure, tasks, interfaces), `adr.md`
  (in-force ADRs), `acceptance.feature` (BDD scenarios).
- **Project-wide**: all `.skillgrid/change-specs/` dirs + `AGENTS.md`
  (operational rules) + `.skillgrid/projectwiki/decisions/04-adr-*.md`
  (in-force ADRs) + the projectwiki vault pages that pin design
  (`architecture/`, `invariants/`, `workflows/`, `glossary/`).

The comparison is: **does the code match the documented design?** Not: is the
design good? (That is the Goldfish review's job, R7.)

## Sequential processing + source grounding (R29)

Process spec dirs **one at a time** (sequential, not parallel) to prevent
context overflow. For each doc, resolve referenced files, symbols, and
endpoints via code search (`grep`/`glob`/`read`). If a reference is
unresolvable (the doc names a file or symbol the codebase does not have),
record it as a **Missing Features** finding and prompt the user to add an
explicit link to the document (cheaper future scans).

Read files liberally: the report reflects the current state of the code, never
memory.

## Difference filtering (R30)

Report **only** actionable discrepancies. If the implementation matches the
design completely, output the literal `No drift detected.` — do not describe
what is working or what matches.

Be rigorous: if a design doc mentions a specific object, field, or file name
and the code does not have it, that is a reportable drift. Double-check for
coverage before clearing a doc.

## Categorize (R27)

Classify every finding into **exactly one** of the 4 categories per
`divergence-categories.md`. The categorization is fixed before review starts:
a finding carries its category into the report, and dual classification is not
allowed.

## Finding review (R33)

Findings reuse the Goldfish resolution workflow (Task 22, cited not restated:
`.agents/skills/writing-blueprints/references/goldfish-resolve.md`). The
`codebase-details` persona
(`.agents/skills/_shared/elephant-goldfish/codebase-details.md`) is the
**primary reviewer**: a fresh subagent (R36/R37) whose "verify code vs doc"
audit vectors are directly applicable.

Categorization is the R10 four actions (CONTEST / ESCALATE / PROPOSE-FIX /
DECLINE) with the same interactive gate (auto-accept / continue / interactive
break down / comment on file / custom instruction). After a finding is
resolved, the Reconcile step assigns its disposition.

## Reconcile (R32)

Two-way:

- **Code was intentional** (the design doc is stale) → update
  `briefing.md`/`blueprint.md`/ADR to reflect reality. The doc is the artifact
  that changes.
- **Design was violated** (the code is wrong) → file a Backlog task (via
  `ticketing`) or fix the code directly. The code is the artifact that
  changes.

The user decides the path per finding (interactive gate, same pattern as R10).
Protocol + dispositions: `reconciliation.md`.

## Report (R31)

Committed artifacts:

- **Per-change**: `.skillgrid/change-specs/<topic>/drift-report.md`
- **Project-wide**: `.skillgrid/drift-report-{YYYY-MM-DD}.md` (the repo's
  `.skillgrid` root, not a spec dir)

Strict format (R31, copied verbatim into the reference):

```markdown
# Drift Report — {topic | project-wide} ({YYYY-MM-DD})

## <DESIGN_DOC_NAME>
* [Divergence item — category: <R27 category> — file:line]
```

One `## <DESIGN_DOC_NAME>` section per design doc, each listing
`* [Divergence item — category: <R27 category> — file:line]` bullets (or the
literal `No drift detected.`). Each finding names its category (R27) and
includes a file:line reference where applicable; the report is committed to
the spec dir (spec-before-code). Dispositions per `reconciliation.md`;
findings write ledger rounds per
`.agents/skills/_shared/craft/review-ledger.md` (they surface at qa time via
the debt-surfacing step, Task 12).

## Size rule (R35)

This reference set keeps qa/SKILL.md's Drift-mode section under 40 lines. If
a reference grows past the size budget, split it — never inline it.
