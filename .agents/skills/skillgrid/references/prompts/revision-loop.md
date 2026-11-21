# Phase 5: Revision Loop

Revise until two gates pass, the users' and the score's, in at most three passes per gate, and stop with a precise account when they do not.

When to load: Phase 5. Roles: orchestrator coordinates; blind reviewers, score evaluators and the fix editor do the work.

## The two gates

| Gate | Decides | Passes when |
|---|---|---|
| Review gate | whether the change holds up | the gate read in `references/roles/panel.md` gives every sampled surface a majority to keep using it, the `review.md` floor is met, and no high-severity defect remains |
| Score gate | what to fix | the calibrated floor in `references/scoring/program-score.md` reaches `genesis.quality_target` (default 8.5) and no dimension sits more than 1.0 below it |

Both must pass. Neither can be argued past: a strong score does not excuse users quitting, and happy users do not excuse a broken design.

## One pass

1. **Measure.** Run the review gate read and the score evaluation, each under `references/scoring/evaluator-protocol.md`. If the last audit was `revise`, also re-run the auditor on the units its tickets named. Save everything in `report.md` and `review.md`.
2. **Plan.** Write the revision plan in `review.md`: tickets in priority order (audit high-severity tickets, then panel two-vote defects, then the lowest score dimensions), each with its quoted passage, the smallest fix, and the preserve list from the panel's memories. Use the smallest revision class that can work: cut or merge, structural, connective, code, factual.
3. **Revise.** Hand the tickets to the fix editor (`references/specialists/fix-editor.md`). It sees tickets and passages, never scores or targets. It keeps a copy of the accepted version and writes each revision to `.skillgrid/work/<YYYY-MM-DD-topic>/attempts/<unit>/revision-rK.md`, never over the working tree.
4. **Accept or reject.** For each revised unit, give the preserved accepted version and the revision to the panel in compare mode, unlabeled. Apply the revision to the working tree only if the panel prefers it, or splits and the revision has fewer defects. Otherwise leave the unit as it is, keep the rejected revision where it is, and record why.
5. **Record.** Append the pass to `review.md`: gate results before and after, tickets closed, passages changed, strengths preserved, and the verdict `CONTINUE`, `PASSED` or `STOPPED`.
6. **Count.** Add one to `genesis.passes.review` if the review gate was failing when the pass began, and one to `genesis.passes.score` if the score gate was.

## Stop rules

- **Both gates pass:** move to Phase 6.
- **A failing gate reaches three passes:** stop. In check-in mode, show the programmer exactly what is missing (which surfaces lose users and where, which dimensions sit how far under target, which tickets resist) and offer: three more passes, accept the change as it stands, or change the plan. In autonomous mode, record the same account in `report.md` and continue to Phase 6, which reports "Gates not met".
- **No progress:** if a pass closes no high-severity ticket and moves neither gate, stop early with the same account. Revision that changes nothing measurable is churn.

## Revision rules

- Preserve what reviewers remembered unless it is structurally false.
- Do not add decorative abstraction to chase a score. Prefer a concrete seam to a generic layer, and a named error to a silent catch.
- If a module is correct but lifeless, add a real failure path, a cost, or a test that names the behavior.
- If a change cannot justify its existence, cut or merge it, and update the blueprint and the ledger.
