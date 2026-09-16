# [Feature Name] — Design Briefing

> Copy this template to `.skillgrid/specs/YYYY-MM-DD-<topic>/briefing.md`.
> This is the validated design/spec. Fill every section from the
> brainstorming conversation. Every requirement MUST be falsifiable
> (Current / Target / Acceptance). Then run the spec self-review
> (placeholders, internal consistency, scope, ambiguity, and: does every
> requirement have all three fields?) before handing it to the user.
> (User preferences for spec location override this default.)

## Problem / Intent

[What problem does this feature solve, for whom, and why now. 1-3
sentences. This is the "intent" the whole spec hangs from.]

## Purpose & Success Criteria

- **Purpose:** [What the feature enables the user to do]
- **Success criteria:** [Measurable, testable conditions that define "done"]
- **Out of scope:** [What this feature explicitly does NOT do]

## Context

[Relevant existing flows, files, or constraints in the repo that this
feature must respect or integrate with. Reference `docs/PRD.md` and
`docs/ARCHITECTURE.md` sections where applicable.]

## Approaches Considered

[List the 2-3 approaches proposed during brainstorming, with trade-offs and
the chosen one + why. Keep it brief — the chosen design is below.]

- **Chosen:** [Approach + one-line rationale]
- **Rejected:** [Approach — why not]

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

## Testing Decisions

- **What makes a good test:** [Only test external behavior, not implementation
  details]
- **Modules to test:** [Which units get tests, at which seam]
- **Prior art:** [Similar tests in the codebase to model after]
- **Edge cases:** [Boundary conditions, failure modes, concurrency]

## Impact on Global Docs

[If this feature changes `docs/PRD.md` (new feature row, changed metrics) or
`docs/ARCHITECTURE.md` (new component, data store, integration), list the
exact edits here. If none, write "None".]

- `docs/PRD.md`: [edit or "None"]
- `docs/ARCHITECTURE.md`: [edit or "None"]

## Clarity Report

[Populated from the `interviewing` skill's exit check. The clarity gate must
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

[Pointers to ADRs in `.skillgrid/adr/` that constrain THIS feature — the decisions
made during the interview that clear the bar (hard to reverse + surprising +
real trade-off). List each as `.skillgrid/adr/NNNN-slug.md — one-line gist`. These
are the source of truth; do not restate them here. If no decision for this
feature cleared the ADR bar, write "None".]

## Glossary

[Pointers only. The project's canonical terms live in `.skillgrid/glossary/`
(maintained by skillgrid:architectural-decision-records during the interview). List here only
the terms THIS feature introduced or relied on, each with a link to
`.skillgrid/glossary/`, plus a one-line note if a term was sharpened or split
during this interview. Do not redefine terms here — the glossary is the single
source of truth. If the feature added no new terms, write "None".]
