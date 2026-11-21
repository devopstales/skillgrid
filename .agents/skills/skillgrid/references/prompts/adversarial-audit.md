# Phase 4: Adversarial Audit

Break the system's bias toward approving itself before anything is scored: one skeptical, read-only audit of the whole change.

When to load: Phase 4, and in Phase 5 when a pass needs a fresh audit. Role: orchestrator runs the auditor.

## Run the auditor

1. Build the auditor's packet exactly as `references/roles/auditor.md` lists: the changed code, the plan files, and the pattern standards. No targets, no scores, no panel verdicts.
2. Run it at the strongest independence level recorded in `changes.<slug>.genesis.independence` (`references/scoring/evaluator-protocol.md`).
3. Save its report verbatim to `report.md` (QA half) and `review.md`. A read-only auditor returns the report; you save it.

## Act on the verdict

The report ends with one `audit_status` line.

- `audit_status: pass`: set the gate to passed and move to Phase 5, which still runs its two gates.
- `audit_status: revise`: set the gate to `revise` and move to Phase 5 with the tickets as its first work.
- `audit_status: major_rewrite`: stop and check in, even in autonomous mode. Show the failing passes and offer three paths: re-architect the affected part (back to Phase 2 for those units), revise anyway, or stop. Record the programmer's choice in `genesis.decisions`.

A strong panel result never overrides the audit. The audit judges the change as a whole; the panel judges whether users keep using it.
