# Program score

The rubric for the score gate, and the Phase 6 report. Eight dimensions, each scored 1–10 against a named artifact, then combined with the floor rule.

When to load: Phase 5 (score gate), Phase 6 (final report). Role: score evaluator.

## Dimensions

| # | Dimension | 9–10 | 7–8 | 5–6 | 1–4 | Evidence |
|---|---|---|---|---|---|---|
| 1 | **Spec fit** | every acceptance scenario passes and the edge cases the scenario implies are tested | scenarios pass; one implied edge case untested | scenarios pass only on the happy path | scenarios fail or are asserted loosely | `acceptance.feature`, test output |
| 2 | **Test evidence** | tests name behavior, fail for the right reason, and the command output is recorded | tests cover the main paths with recorded output | tests exist but assert little | no meaningful tests | test files, commands, output |
| 3 | **Code health** | small functions, one job each, names carry meaning, duplication is deliberate | readable, a few long functions | works but hard to change | tangled, duplicated, fragile | `references/patterns/code-patterns.md` |
| 4 | **Security** | trust boundaries named, inputs validated, secrets handled, applicable threat rows closed | the main risks handled | one open medium risk | an open high risk | threat matrix, audit tickets |
| 5 | **Review axes** | every review axis passes with quoted evidence | one axis at warning | two axes at warning | an axis at fail | `review.md` |
| 6 | **Operability** | one-command build and run, config precedence documented, actionable errors, logs that identify the request | build and run work; errors mostly actionable | setup needs undocumented steps | does not build from a clean checkout | `README.md`, build output |
| 7 | **Change safety** | rollback documented and tested, migration reversible, scope within plan | rollback documented | rollback assumed | no rollback path | blueprint, `report.md` |
| 8 | **Handoff** | README, changelog, PR body and ADRs let a new maintainer start without asking | the main documents exist | partial docs | nothing a newcomer can use | `README.md`, `CHANGELOG.md`, ADRs |

## Combining

1. Score each dimension with a one-line justification and the evidence it rests on. A dimension without evidence scores at most 4.
2. Weight: Spec fit 2, Test evidence 2, Code health 1.5, Security 1.5, Review axes 1, Operability 1, Change safety 0.5, Handoff 0.5.
3. The **calibrated score** is the weighted mean, adjusted by the calibration rule in `references/scoring/evaluator-protocol.md`.
4. Apply the **floor rule** from `skillgrid:_shared/verification/floor.md`: the program is judged by its weakest load-bearing dimension, not its mean. A program with a 9 mean and a 4 in Security is not an 8.5 program.

## Gates

- **Score gate** passes when the calibrated score reaches `changes.<slug>.genesis.quality_target` (default 8.5) and no dimension sits more than 1.0 below it.
- **Floor** is `met` when every load-bearing dimension is at or above target, `met-with-fixes` when the low dimensions have accepted fixes, `not-met` otherwise.
- Report the score with its grade. A grade C score is reported and labelled, never used to pass the gate on its own.

## Phase 6 report

`report.md` ends with:

```text
Completeness: <complete | incomplete — what is missing>
Review: <floor met | met-with-fixes | not met>
Score: <n.n> / target <n.n> (grade <A|B|C>)
Gates: QA <PASS|CONCERNS|FAIL|WAIVED> · Scope <PASS|FLAG|BLOCK> · Ship <decision>
Independence: <grade> — <level>
```

A run that did not reach its target says so plainly and lists the dimensions that held it back. "Complete" and "good" are different claims; report both.
