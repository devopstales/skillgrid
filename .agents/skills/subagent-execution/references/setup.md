# Setup

Loaded on demand from `subagent-execution` SKILL.md before dispatching Task 1:
workspace creation, the progress ledger, the commit checkpoint, and the
pre-flight conflict scan. The Status Guard and the gate subsections stay in the
main file; this is the detailed setup procedure.

## Isolated workspace

Ensure the work happens in an isolated workspace: use
`skillgrid:isolated-workspace` to create one or verify the existing one. Never
start implementation on a main/master branch without your human partner's
explicit consent.

## The progress ledger

Conversation memory does not survive compaction. In real sessions, controllers
that lost their place have re-dispatched entire completed task sequences — the
single most expensive failure observed. Track progress in a ledger file, not
only in todos.

- Each plan owns a workspace: at skill start, run this skill's
  `scripts/sdd-workspace PLAN_FILE` — it prints the plan's git-ignored
  directory (`<repo-root>/.skillgrid/sdd/<plan-basename>/`), home to every
  artifact for THIS plan: ledger, briefs, reports, review packages. Another
  plan's directory is never yours to read or write.
- Check for this plan's ledger at `<workspace>/progress.md`. If its first line
  names your plan file, tasks with a `Task <N>: complete` line are DONE — do
  not re-dispatch them; resume at the first task without one. A task whose last
  line is a fix round is mid-loop: resume the loop at the next round. A ledger
  whose first line names a different plan file — or a stray ledger at the old
  flat path `.skillgrid/sdd/progress.md` — is another plan's progress: leave it
  in place and start your own, fresh.
- Create the ledger with its identity as the first line:
  `# SDD ledger — plan: <plan file path>`.
- The ledger is your recovery map: the commits it names exist in git even when
  your context no longer remembers creating them. After compaction, trust the
  ledger and `git log` over your own recollection.
- **Mnemonic mirror:** when `mnemonic.enabled: true`,
  `mem_save(topic_key: skillgrid/<topic>/execution-progress, ...)` the ledger's
  tail (current task + last ruling) after each task completion and each ruling
  — upsert, never duplicate. The in-repo ledger stays the source of truth; the
  store is the index for a fresh session that hasn't found the workspace yet.
- On resume, also read the commit checkpoint per `skillgrid:work-unit-commits`:
  `bash .agents/hooks/checkpoint-state.sh restore`. It prints
  `.skillgrid/sdd/checkpoint.json` (derived from `git log -1` + the last
  `[skillgrid-context]` block) plus live git state. If its `remaining` names a
  partial unit, finish that unit before dispatching the next task. The ledger
  tells you *which* tasks are done; the checkpoint tells you *where in the code*
  the last unit left off — use both.
- `git clean -fdx` will destroy the workspace (it's git-ignored scratch); if
  that happens, recover from `git log`.

## Read the plan

Read the plan once, note its context and Global Constraints, and create a todo
per task. If a `tasks.md` exists alongside the plan (produced by
`skillgrid:slicing`), read it instead: tickets replace tasks as the dispatch
unit, and execution follows the wave order. If the plan names a Spec, read that
too: the spec is the authority the plan argues from, and conflicts inside the
plan resolve against it. A plan with no reachable spec gets a ledger note
saying so — rulings made without one are provisional.

## Pre-flight conflict scan

Before dispatching Task 1, scan the plan once for conflicts, writing down what
you checked as you check it:

- tasks that contradict each other or the plan's Global Constraints
- anything the plan explicitly mandates that the review rubric treats as a
  defect (a test that asserts nothing, verbatim duplication of a logic block)

The scan's output is a table, not a verdict. One row for every pair of tasks
that share a file or an interface: the two tasks, what one produces against
what the other consumes, and what you found. One row for every task: whether
its own text agrees with itself — the tests it specifies against the code it
specifies, the files it creates against the files it later touches. "The scan
is clean" without those rows is not a scan you ran.

Write the table to the ledger. Rule on everything you find before execution
begins — each finding against the plan text that mandates it — and record each
ruling in the ledger. If the scan is clean, proceed without comment. Rule on
each conflict it surfaces — the spec is the binding authority, the plan is its
argument — record the ruling beside its row, and dispatch Task 1. The review
loop remains the net for conflicts that only emerge from implementation.

If the blueprint has no `## Plan Review` section, note it in your pre-flight
report: "Blueprint was written before plan review was introduced — proceeding
without the structural gate." Do not block; the execution review loop still
catches what the structural gate would have.
