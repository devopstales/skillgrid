# Cite-Don't-Restate Convention (shared across all Skillgrid skills)

When a downstream artifact (a spec, blueprint, task, or report) references a
decision, a locked constraint, or a prior ADR, **cite it by its identifier — do
not restate its wording.** Restating is where drift creeps in: a locked
constraint copied into three blueprints, then edited in one, silently diverges
from the other two. A citation cannot diverge — it points at the one source
that still has authority.

## The Rule

1. **Cite, don't copy.** When a blueprint's Global Constraints, a task's
   rationale, or a report's decision references a decision that already has an
   identifier, write the reference, not the text:
   - ADR: `per 04-adr-0008` (or `per ADR-0008`)
   - Locked constraint: `per 05-locked-constraints §<section-or-bullet>`
   - Prior change: `per .skillgrid/archive/2026-09-21-session-events-layer`
2. **Restate only when the citation is not enough to act.** If a reader must
   *do* something with the constraint (not just know it exists), you may
   restate the operative clause — but keep it to the clause, and still cite the
   source. A one-line "consequence for this change" is a restatement that
   earns its place; a paragraph paraphrase of the ADR's rationale does not.
3. **One authority.** The cited artifact is the source of truth. If the cited
   ADR is superseded, the citation goes with it (cite the in-force ADR, per
   `03-adr-index.md`). Never cite a superseded ADR as current authority.
4. **No citation, no claim.** A downstream artifact that asserts a constraint
   or decision it did not earn in this change (not in its own briefing/blueprint)
   MUST cite where it came from. An uncited inherited decision is an
   assumption — it gets the `ASSUMED` treatment (decision debt, WARNING) until
   sourced.

## Why

This is the "cite the predicate by ID, never paraphrase" discipline, applied
to our ADR + locked-constraint model. The ADRs and `05-locked-constraints.md`
are already the single authority; the failure is not in the authority, it is in
the *downstream paraphrase* that quietly disagrees with it. A citation makes
the downstream artifact a *pointer*, so the authority stays singular.

## Where It Applies

- `writing-blueprints` — Global Constraints and task rationale that inherit a
  locked constraint or prior ADR decision: cite it.
- `slicing` — a ticket's rationale that rests on a blueprint constraint or ADR:
  cite it.
- `qa` / `reflect` — a finding or decision that invokes a constraint: cite the
  ADR / constraint / archive entry, not a paraphrase.

## Red Flags

- A locked constraint paraphrased in a blueprint without a `per 05-locked-constraints …` citation.
- Two blueprints restating the same ADR's decision with slightly different wording.
- A report citing a superseded ADR as current authority.
- A downstream artifact asserting a decision it never earned, with no citation
  and no `ASSUMED` flag.
