# When to Stop and Ask for Help

## When to Stop and Ask for Help

**Decision gates:**

| Condition | Action |
|---|---|
| State is `blocked` (missing artifacts, unsafe context) | STOP; return `blocked` with the missing items named. |
| State is `all_done` (tracker done-signals / ledger / legacy checkboxes) | Do not dispatch; return `success` with `Next: qa` — you own only up to the qa handoff. |
| `changes.<topic>.gate` is set in `state.yaml` with no user confirmation | STOP at the gate screen — the user's "Implement" is what clears it. |
| A needed edit is outside the change's declared edit roots | STOP and report the unsafe path — never edit outside the affected areas without a ruling. |
| Already in a linked worktree (and not a submodule) | Skip creation; report the existing workspace. |
| A native harness worktree tool exists | Use it — `git worktree add` with a native tool present creates phantom state. |
| A Hermes Kanban claim printed a workspace path | Use the claim's path; skip worktree creation (the claim owns placement and cleanup). |
| `testing.setup` baseline tests fail | Report and ask: proceed or investigate — do not start on a red baseline. |
| `400-line budget risk: High` / `Chained PRs recommended: Yes` / `Decision needed: Yes` with no resolved delivery path | STOP before writing code; return `blocked` naming the delivery decision required. |
| A task's declared dependency (`Needs:`) is still in flight | Do not dispatch it — dependency-gated dispatch. |
| A ticket's `Precondition:` is unmet | STOP with a checkpoint; name the unmet precondition; no partial work. |
| A one-way task's completion rests only on the implementer's report | Run the claim-check (CLAIM / EXTRACT / DOUBT / RECONCILE) before the ledger line. |
| A verify command has no `Fails-when` and no falsifiable `Then` | Not verification — flag it. |
| A worker's claims don't match disk or `git log --grep=<task-id>` | The worker is not done, whatever the message says — filesystem and git are the truth. |
| > 5 units to dispatch in parallel | STOP; ask for approval, name the count and the cap; partition into waves of ≤ 5. |
| Blocker: missing dependency, test fails, instruction unclear, plan gap | STOP executing; ask for clarification rather than guessing. |
| Verification fails repeatedly (same check, 2+ times) | STOP; name the check and the output; do not retry a third time silently. |
| Out-of-order or invalid transition (`exec` before the checklist, review with no diff, QA gate on an uncommitted tree) | One-line State Advisory, list the valid next actions, halt. Never skip ahead to "catch up". |
| On `main`/`master` without explicit user consent | STOP; branch or get consent before any implementation. |
| A delegated worker holds the path you need | Create a separate worktree — never reuse a worker's worktree. |
| The plan needs rethinking based on your feedback | Return to Step 1 (load and review the updated plan); don't force through. |
| A task is one-way (irreversible) per its ticket | Human checkpoint before it runs — the ticket's `Reversibility: one-way` says so. |

**STOP executing immediately when:**
- Hit a blocker (missing dependency, test fails, instruction unclear)
- Plan has critical gaps preventing starting
- You don't understand an instruction
- Verification fails repeatedly

**Ask for clarification rather than guessing.** Decision gates follow `_shared/craft/review-discipline.md` (artifact-first: large reports/plans go to a file, not chat; zero-tool turn-yield: end the turn with no tool calls and go idle while the user decides).

**Return to Step 1 when:** the partner updates the plan based on your feedback, or the fundamental approach needs rethinking. Don't force through blockers — stop and ask.

**Out-of-order or invalid transition** (e.g. `exec` before the checklist exists, a review with no diff, a QA gate on an uncommitted tree): do not halt silently and do not execute out of order. Open with a single-line advisory, then halt for the next command:

> **State Advisory:** Cannot `<action>` in `<state>`. Recommended: `<valid command>`.

List the valid next actions, then stop. Never skip ahead to "catch up" to the expected state.


## Remember

- Review the plan critically first
- Climb the lazy ladder before every coding task (The Lazy Discipline) — never simplify away what the ladder protects
- Follow plan steps exactly
- Don't skip verifications
- Commit after every task per `_shared/work-unit-commits` — commits are your checkpoints
- Reference skills when the plan says to
- Stop when blocked, don't guess
- Never start implementation on main/master without explicit user consent
- If `tasks.md` exists, execute tickets in wave order, not file order

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | % of executed waves whose tickets reached `done` without a re-run (no wave re-dispatched after a failed review) | `state.yaml` + ticket status history (Backlog / tracker) | should rise |
| Lagging | Post-merge rework: commits touching the change's files after the first merge that revert or re-scope an executed ticket | Git history | should fall |

## Anti-Patterns

| Anti-pattern | Symptom | Fix |
|---|---|---|
| Skipping the plan review | Execution starts before the plan is critically reviewed | A gap caught in review is one line; caught in execution it is a rework of every task that depended on it. |
| Testing at the end | No failing test before code; a red gate at the end can't tell you which task broke | Failing test exists before code, per task — verifications are per-task. |
| Gate-less "small" task | A requirement with no gate before its code | Small tasks are where a missing gate silently skips the acceptance scenario — gates are per requirement. |
| Zone-guard fixup "later" | A commit stages both `.skillgrid/change-specs/` and code | The `pre-commit` hook blocks it; "later" means a commit that was already wrong — one commit each, spec before code. |
| "Close enough" on spec gaps | Spec gaps found in review, work marked done anyway | Not done — fix or hit the cap and adjudicate; those are the only exits. |
| Controller fixing inline | The controller edits the code itself instead of dispatching | Controller fixes pollute your context and skip review — resume the implementer (subagent mode). |
| Round past the cap | "One more round will converge" after the cap | Past the cap, rounds don't converge — the failure is structural; adjudicate and route. |
| Early adjudication | Ruling a finding in round 2 to end the loop | Adjudicate only at the cap — the loop runs its rounds; the breaker is the only exit without a fix. |
| Wandering re-review | A re-review reports new findings on untouched code | Scoped re-reviews verify fixes and cannot wander — new findings go to the ledger, not the loop. |
| Silent finding discard | "Obviously wrong, I'll drop it" with no ledger entry | Every ruling is a ledger entry — silent discards are forbidden. |
| Re-review skipped on a small fix | A fix lands unreviewed | Unreviewed fixes are how regressions land — every round ends with a scoped re-review. |
| Shipping on minor review findings | Open findings at ship | A review only closes clean or escalated (cap per `_shared/planning/rigor-tiers.md`). |
| Ledger skipped | No `progress.md` row per task; a re-dispatched completed sequence after compaction | The ledger is what survives compaction — bookkeeping is not overhead. |
| Worker-spawned reviewer | The implementer dispatches its own reviewer | A duplicate seat reviewing the same diff — the task review is the gate; flag it as a defect. |
| Eyeballed worktree state | "Obviously not in a worktree" — no detection run | Run the Step 0 detection — harness-created isolation and submodules both fool eyeballing. |
| `git worktree add` with a native tool present | Phantom state the harness can't see or manage | A native tool (e.g. `EnterWorktree`) owns placement, branching, and cleanup — use it. |
| Unchecked worktree directory | The whole tree committed into the repo | `git check-ignore` before creation — an unignored worktree dir commits everything. |
| Deferred baseline tests | Work starts on a workspace whose tests were never run | A dirty baseline makes every later failure ambiguous — run now; proceeding past failures is the human's call. |
| Back-to-back dispatches | Dispatches spread across multiple responses | One dispatch per response = sequential execution — multiple dispatches in the same response are what run in parallel. |
| Sequential "because risky" | Genuinely independent domains run one at a time | Sequential costs the whole wall-clock time for zero safety gain — the risk is bad scoping, not concurrency. |
| Blind fan-out merge | Results merged without checking each agent's output | Agents make systematic errors — integrate from the ledger + git, review each summary and diff. |
| "Two tasks, parallel is overkill" | Two independent investigations run sequentially | Dispatch cost is near zero; the benefit scales with independence, not count. |
| Shared fan-out checkout | One worktree for all the agents | Shared checkouts race on the index and produce unmergeable diffs — one worktree per agent, always. |
| Speculative robustness | "I'll add the robust version to be safe" | A new dependency with a bug surface — ship the lazy version, question it in the same response, let the user ask for the full version. |
| "Lazy means untested" | A non-trivial change done with its happy-path gate unmet or unrun | The check rule: the gate is green, freshly run — lazy shortens the code, not the check. |
| Small-diff-first | Understanding skipped to ship a smaller diff | The ladder runs after you understand, never instead of — a small diff in the wrong place is a second bug. |

## Red Flags

- Implementation started before the acceptance scenario was confirmed RED
- A commit stages both `.skillgrid/change-specs/` and code (zone guard violation)
- Task marked completed with no captured test output or exit code
- A task marked complete on the implementer's return message alone — the claimed artifact or commit was not spot-checked on disk / in git
- QA gate hit FAIL and the fix was applied without a failing test first
- Ticket status jumped past `review` to `done` without a clean review
- Work started on main/master without explicit user consent
- Edits landed in the main checkout instead of the isolated workspace
- Baseline tests not verified clean before starting work (a dirty baseline makes every later failure ambiguous)
- Existing isolation not detected and a duplicate worktree was created
- `git worktree add` used even though a native worktree tool was available
- Worktree directory not checked with `git check-ignore` before creation
- Fan-out agents shared a single checkout instead of one worktree each
- Two fan-out agents edited the same file or shared resource (no disjoint scope was set)
- Fan-out dispatches spread across multiple responses instead of one (running sequentially)
- The fan-out ledger is missing or stale — integration relies on in-context summaries that may have compacted
- Full test suite not re-run after all fan-out agents returned
- A fan-out dispatched with \> 5 concurrent worktrees without a recorded user approval
- An implementer was dispatched without a complete task brief (no `task-brief` path, or the brief path is missing from the prompt)
- A subagent's output was trusted without parent re-verification — the `#### Gates` oracles were not re-run before marking the task complete
- The task loop re-dispatched the same task after round 5 without the breaker adjudicating — the cap was exceeded silently
- A task reviewer was dispatched without a diff file, or without all three inputs (brief, report, review package)
- The QA gate was skipped or its findings were not triaged before the final review ran
- A ticket without a complete Work Unit Evidence row was marked complete
- A `use case` / interface / factory / config block for one implementation or a value that never changes
- A new dependency added for what a few lines of stdlib or a native platform feature can do
- A `ponytail:`-marked corner with no named ceiling and upgrade path
- A non-trivial change declared done while its happy-path gate is unmet or unrun

## Verification

- [ ] The plan (or `tasks.md`) was read and critically reviewed; concerns raised and resolved before execution started
- [ ] Every task's gate is green: `CHECK:`+`EXPECT:` ran (or `manual` executed, `ABANDON` surfaced as a handoff) with captured output
- [ ] Test runner from `testing.runner` exits 0; build from `commands.build` and lint from `commands.lint` are clean
- [ ] Every ticket has a complete Work Unit Evidence row (focused test, runtime harness, rollback boundary)
- [ ] The lazy ladder was climbed before every coding task: the chosen rung is the minimum that satisfies the constraints, real corners carry a `ponytail:` ceiling comment, and nothing protected by "When NOT to be lazy" was simplified away
- [ ] The QA gate returned PASS or WAIVED with the waiver recorded; FAIL findings were fixed with a failing test first
- [ ] Subagent mode: every dispatched implementer received a complete task brief (the `task-brief` output path was in the dispatch prompt), and every subagent's `#### Gates` oracles were re-run by the parent before the task was marked complete
- [ ] The final review is clean or capped (per `_shared/planning/rigor-tiers.md`, fix loop cap); open findings logged with rulings
- [ ] Subagent mode: the task loop reached a terminal state — no task stuck cycling past round 5 without a recorded breaker adjudication
- [ ] If `ticketing.enabled: true`, every ticket's status reflects its real state (`ready` → `in-progress` → `review` → `done`)
- [ ] Step 0 workspace checks: existing isolation detected (submodule guard included), worktree directory confirmed ignored, clean baseline verified (test runner exit 0, `git status` clean)
- [ ] Fan-out (when used): each agent ran in its own worktree, all dispatches issued in a single response, the fan-out ledger has a row per agent, every agent summary and diff was reviewed for conflicts, and the full suite was re-run with the result in the ledger
- [ ] `state.yaml` carries `changes.<topic>.phase: execution`, the workspace is deleted after a clean review, and `skillgrid:ship` was invoked

## Output Contract

The deliverables are the executed work, the ledger, and the handoff state:

- **Workspace:** the isolated worktree (path + branch named), or the explicit
  "work in place" decision.
- **Progress ledger** at `<workspace>/progress.md` — one row per task:
  `Task N: Owns: ... / Needs: ... / complete` with captured gate output; the
  fan-out ledger rows when fan-out was used.
- **Work Unit Evidence** per ticket — focused test command + result, runtime
  harness + result (or `N/A` + reason), rollback boundary (per
  `_shared/rules/verification-ladder.md`).
- **QA gate result** — PASS / WAIVED (waiver recorded) / FAIL (findings fixed
  with a failing test first).
- **Ticket statuses** (when `ticketing.enabled: true`) — every ticket at its
  real state, `review` → `done` only on a clean review.
- **`state.yaml`** — `changes.<topic>.phase: execution`; the qa handoff is the
  terminal state (`Next: qa`), not a done-claim.

