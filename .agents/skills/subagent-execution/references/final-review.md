# Final Review

Loaded on demand from `subagent-execution` SKILL.md once every task is complete
and `skillgrid:qa` has passed — the whole-branch review that precedes ship.

The final whole-branch review gets a package too: run
`scripts/review-package PLAN_FILE MERGE_BASE HEAD` (MERGE_BASE = the commit the
branch started from, e.g. `git merge-base main HEAD`) and include the printed
path in the final review dispatch, so the reviewer reads one file instead of
re-deriving the branch diff with git commands. Dispatch on the most capable
available model (see `model-selection.md`) using
`skillgrid:requesting-code-review`'s two-axis templates, in parallel:
[code-reviewer.md](../../requesting-code-review/references/code-reviewer.md)
(Standards — glossary, in-force ADRs, smell baseline) and
[spec-reviewer.md](../../requesting-code-review/references/spec-reviewer.md)
(Spec — per-scenario BDD, missing/partial, scope creep). Present the two
reports side by side without merging or reranking. Point them at the ledger's
deferred-minor and parked lines so they can triage which must be fixed before
merge.

**Escalate for large or high-risk branches:** if the whole-branch diff is large
(50+ changed lines) or high-risk (auth, data migration, money, concurrency,
public API), escalate the final review from the two-axis pass to
`skillgrid:parallel-code-review` — it fans out specialist reviewers (Standards,
Spec, edge cases, verification gaps, security, red team) in parallel, then
triages and deduplicates their findings. The per-task reviews stay the
lightweight two-axis pass; only the final whole-branch review escalates.

If the final whole-branch review returns findings, dispatch ONE fix subagent
with the complete findings list — not one fixer per finding. Per-finding
fixers each rebuild context and re-run suites; a real session's final-review
fix wave cost more than all its tasks combined. Then run exactly one scoped
re-review of the fix wave (`scripts/review-package PLAN_FILE FIX_BASE HEAD`
over the fix range, [re-review-prompt.md](re-review-prompt.md)). Adjudicate
any residual findings as in the task loop's breaker: park with rulings, or rule
on the load-bearing ones and ledger what you decided. Only the four breaker
classes stop you here. There is no second fix wave — residual load-bearing
findings surface to your human partner when ship presents the options.

Per-task fix loops follow `skillgrid:receiving-code-review` triage rules
(fix now / defer / human look / noise) applied through the ledger's
park-and-rule mechanism.
