# [Feature Name] — Design Briefing (requirements & intent)

> **STATUS:** `draft` (YYYY-MM-DD)
> **Author:** [resolved per SKILL.md § Author resolution: `git config user.name` → `GIT_AUTHOR_NAME` → ask in chat; never a placeholder]
> **Intent frozen:** no (YYYY-MM-DD) ← set to "yes (YYYY-MM-DD)" when the user approves the briefing (the approval gate); after that, the frozen block below is read-only for agent steps

> Copy this template to `.skillgrid/change-specs/YYYY-MM-DD-<topic>/briefing.md`.
> **Role:** this is the *what* — the validated design/spec: problem, intent,
> requirements, success criteria. The companion `blueprint.md` is the *how*
> (implementation plan). The filename pair is load-bearing: `brainstorming` and
> `resume` key phase state off which of the two files exist. Do not rename.
> This is the validated design/spec. Fill every section from the
> brainstorming conversation. Every requirement MUST be falsifiable
> (Current / Target / Acceptance). Then run the spec self-review
> (placeholders, internal consistency, scope, ambiguity, and: does every
> requirement have all three fields?) before handing it to the user.
> (User preferences for spec location override this default.)
>
> **Frozen intent:** the `## Problem / Intent` and `## Purpose & Success
> Criteria` sections — including **Out of scope** — form the frozen block
> (per `_shared/craft/review-ledger.md` § Frozen intent). After the
> `Intent frozen: yes` line is set, no agent step (blueprint, slicing, execute,
> test, ship) may modify those sections. A finding whose fix requires it is an
> **intent_gap** — HALT and ask the human; on renegotiation, the human edits
> the sections, re-dates the freeze line, and downstream artifacts are
> re-derived. "Out of scope" changes are scope renegotiations, never silent
> edits.

## Problem / Intent

[What problem does this feature solve, for whom, and why now. 1-3
sentences. This is the "intent" the whole spec hangs from.]

## Purpose & Success Criteria

- **Purpose:** [What the feature enables the user to do]
- **Success criteria:** [Measurable, testable conditions that define "done"]
- **Out of scope:** [What this feature explicitly does NOT do]

## Context

[Relevant existing flows, files, or constraints in the repo that this
 feature must respect or integrate with. Reference `.skillgrid/projectwiki/ASSUMPTIONS.md`
(VERIFIED facts + `§ LOCKED` ADR index) and `AGENTS.md` (§ Rules — the
operational rules) and `.skillgrid/projectwiki/ARCHITECTURE.md` sections where applicable.]

## Approaches Considered

[List the 2-3 approaches proposed during brainstorming as a comparison table
(columns: approach, key trade-off, verdict), then the chosen one + why. Keep
it brief — the chosen design is below.]

| Approach | Key trade-off | Verdict |
|----------|---------------|---------|
| [approach] | [the trade-off that decides it] | [chosen / rejected] |

- **Chosen:** [Approach + one-line rationale]
- **Rejected by author:** [Approach — reason, attributed to the author, not to a maintainer]

## Requirements

[Every requirement MUST be falsifiable — you can write a test or check that
proves it was met or not. Vague requirements like "improve performance" are
not allowed. Each requirement needs all three fields:]

1. **[Short label]:** [Specific, testable statement.]
   - **Current:** [What exists or does NOT exist today in the codebase]
   - **Target:** [What it should become — "X becomes Y", not "improve X"]
   - **Acceptance:** [Concrete pass/fail check — how a verifier confirms this was met]
   - **Acceptance scenario:** `happy path <scenario-name>` → `acceptance.feature` (BDD is always on)

2. **[Short label]:** [Specific, testable statement.]
   - **Current:** [What exists today]
   - **Target:** [Concrete change]
   - **Acceptance:** [Pass/fail check]
   - **Acceptance scenario:** `happy path <scenario-name>` → `acceptance.feature` (BDD is always on)

[Continue for all requirements. The `acceptance.feature` file is the single
source of truth for executable scenarios — this pointer is a cross-reference only.]

## Implementation Decisions

[Technical decisions that constrain the implementation. This is WHERE the
technical planning lives:]

- **Modules to build/modify:** [Which units are new, which are changed]
- **Interfaces:** [Exact signatures, schemas, or API contracts that other
  modules or callers depend on. No file paths (they go stale) — use module
  names and type shapes.]
- **Data flow:** [Request/response or event paths, where data is stored,
  where it's transformed]
- **Error handling:** [Failure modes, how each is surfaced, edge cases]
- **Dependencies:** [Accept dependencies, don't create them. Return results,
  don't produce side effects. One clear purpose per unit.]
- **Prototype snippets** (optional): [If a prototype produced a snippet that
  encodes a decision more precisely than prose can (state machine, schema,
  type shape), inline it here and note it came from a prototype. Trim to the
  decision-rich parts.]

## Shootdown Findings

[The Goldfish shootdown record — the seven fresh lenses (zoom-out,
design-critic, comprehension, implementation-readiness, codebase-details,
structural, inventive-catalyst) ran in parallel against the anchored hypothesis
after the codebase feasibility check and before the ADR manifest, aggregated by
the synthesizer (per `references/shootdown.md` in the `brainstorm` skill).
`briefing.md` is the single source of truth (ADR-0013): no separate
`concepts_brief.md`. A lens that is `not run` caps its contribution at "not
verified," not "clean." Bounded/prototype changes carry an explicit
`Skipped: bounded/prototype path` note here instead of a table.]

**Lens status:** [one line per lens — completed / not run, all seven named; a
`not run` lens is named, never silent.]

**Anchored hypothesis:** [2-3 sentences, as confirmed by the user at the
readiness gate.]

**Disposition options (one per 🔴 finding, user selects):**
- `resolved` — the design is revised to address the finding; loop back to the feasibility check.
- `accepted-with-note` — the risk is recorded as a known risk in this section; the spec proceeds.
- `downgraded-to-non-goal` — the finding is moved to the spec's Out of Scope.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| 🔴 Blocker | [lens name] | [finding] | [resolved / accepted-with-note / downgraded-to-non-goal] | [the note] |
| 🟡 Major | [lens name] | [finding] | [recorded — does not block] | [the note] |
| 🟢 Nit | [lens name] | [finding] | [recorded — does not block] | [the note] |

**Overall verdict:** [survived / survived-with-notes / needs-revision]

Gate: all 🔴 Blocker findings dispositioned before the spec is written — an
undispositioned blocker blocks the write.

## Testing Decisions

- **What makes a good test:** [Only test external behavior, not implementation
  details]
- **Modules to test:** [Which units get tests, at which seam]
- **Prior art:** [Similar tests in the codebase to model after]
- **Edge cases:** [Boundary conditions, failure modes, concurrency]

## Impact on Global Docs

[If this feature changes `.skillgrid/projectwiki/reference/00-prd.md` (scope, feature, metric),
`.skillgrid/projectwiki/ASSUMPTIONS.md` (a new VERIFIED/INFERRED fact, a new ADR path row),
or `.skillgrid/projectwiki/ARCHITECTURE.md` (new component, data store, integration), list
the exact edits here. If none, write "None".]

- `.skillgrid/projectwiki/reference/00-prd.md`: [edit or "None"]
- `.skillgrid/projectwiki/ASSUMPTIONS.md`: [edit or "None"]
- `.skillgrid/projectwiki/ARCHITECTURE.md`: [edit or "None"]

## Clarity Report

[Populated from the exit check in `references/questioning.md`. The clarity gate must
pass before this spec is written: clarity ≤ 0.20 AND all dimensions ≥ their
minimums. If a dimension is below its minimum, it is marked ⚠ and the planner
treats it as an assumption.]

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        |       | 0.75 |        |                                    |
| Boundary Clarity    |       | 0.70 |        |                                    |
| Constraint Clarity  |       | 0.65 |        |                                    |
| Acceptance Criteria |       | 0.70 |        |                                    |
| **Clarity**         |       | ≤0.20|        |                                    |

**Interview log:** [Key decisions made during the interview. Format: round → question → answer → decision locked.]

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | [what was asked]        | [what was decided]                 |
| 2     | [what was asked]        | [what was decided]                 |

## Open Questions & Assumptions

- **Question:** [Any unresolved question]
- **Assumption:** [Any assumption made — including any ⚠ dimension from the Clarity Report]

## Decisions (ADR)

[Pointers to ADR files (`.skillgrid/projectwiki/decisions/04-adr-NNNN-slug.md`) that constrain THIS feature —
the decisions made during the interview that clear the bar (hard to reverse +
surprising + real trade-off). List each as the Record path plus a one-line gist.
These are the source of truth; do not restate them here. If no decision for this
feature cleared the ADR bar, write "None".]

## Terms

[Pointers only. The project's canonical terms live in `.skillgrid/projectwiki/glossary/`
(`01-business-terms.md` / `02-technical-terms.md`, maintained per `_shared/adr-records/` during the interview). List here only
the terms THIS feature introduced or relied on, each with a link to the terms
file, plus a one-line note if a term was sharpened or split
during this interview. Do not redefine terms here — the terms files are the single
source of truth. If the feature added no new terms, write "None".]
