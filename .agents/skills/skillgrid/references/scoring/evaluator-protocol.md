# Evaluator protocol

Who sees what, who judges whom, and how independence is graded. This file is the firewall between the work and the judgment; a run that leaks across it produces flattering scores and useless reviews.

When to load: Phase 0 (choose the setup), every review and score run, Phase 6. Role: orchestrator.

## The separation

| Role | Produces | Sees | Never sees |
|---|---|---|---|
| Producer | code, artifacts | the unit brief, the standards, the ledger | rubric, targets, scores, panel verdicts |
| Blind reviewer | `review.md`, panel cards | artifact, scenarios, standards, its card | targets, earlier scores, other reviewers |
| Score evaluator | dimension scores | artifact, rubric, scenarios | targets, panel verdicts, earlier scores |
| Auditor | audit tickets, `audit_status` | diff, scenarios, conventions, standards | blueprint targets, scores, panel |
| Orchestrator | state, check-ins | everything | — (it must not write the verdicts) |

The orchestrator may read every verdict. It must not author one. When it does, the verdict is grade C at best.

## Independence grades

| Grade | Setup | How to reach it | Weight |
|---|---|---|---|
| A | different model family, separate tool or host, no shared context | a second CLI the programmer approved (`references/hosts.md`) | full |
| B | isolated subagent of the same model, filtered packet | the host's subagent tool, packet built from files | full for the review gate, discounted for the score gate |
| C | same context, instructions separated only | no isolation available | reported, never counted as independent |

Record the grade in `changes.<slug>.genesis.independence`:

```yaml
independence:
  level: isolated-subagent
  grade: B
  writer_model: <model that produced the work>
  reviewer_model: <model family that judged it>
  isolation_check: passed
```

## The isolation check

Before the first review, prove the reviewer cannot see the forbidden material:

1. Build the packet.
2. Ask the reviewer to list every file and section it received.
3. Compare with the allow-list. Any target, score or panel card is a leak: rebuild the packet.
4. Record `passed` or `failed`. A failed check that is not fixed means the verdict is grade C.

## Calibration

A score is only comparable across runs if the evaluator is calibrated against a known artifact.

- The evaluator scores one reference artifact of known quality before the program, at the start of Phase 5, and once more in Phase 6.
- If its reference score drifts by more than 0.8 from the recorded value, the evaluator is miscalibrated. Re-brief it, re-run the reference, and discard the drifted scores.
- Calibration applies to the score gate only. The review gate reads user verdicts, not numbers.

## Rules

- One model family judging its own work is a self-review. Report it as such.
- Never tell a reviewer the target. A reviewer that knows the target grades the gap, not the artifact.
- Never average across grades. An A and a C do not make a B.
- The evaluator's rubric is the same file for every program (`references/scoring/program-score.md`). Editing the rubric mid-run is a new run.
