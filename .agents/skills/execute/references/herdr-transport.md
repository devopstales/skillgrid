# Herdr Transport (subagent mode)

When a Herdr session is available, Herdr agents are the subagent transport —
the same task loop, the same briefs and report files, a different dispatch
mechanism. Herdr gives the controller lifecycle states (`idle`, `working`,
`blocked`, `done`) and a visible agent pane instead of an opaque
in-process subagent. The installed binary is the authority for command
syntax; `herdr --skill` is the operational reference (pane topology, IDs,
read sources, safety rules). Cite it — don't restate it.

## When the transport is available

The transport is available iff:

1. `test "${HERDR_ENV:-}" = 1` — the controller runs inside a
   Herdr-managed pane (the check is in the herdr skill), AND
2. the implementer agent kind is supported by the installed binary (`herdr
   agent start --help` lists the kinds: `pi`, `claude`, `codex`, `gemini`,
   `opencode`, `copilot`, `omp`, …).

When either check fails, the in-harness subagent path is the transport —
this file adds nothing and changes nothing about the loop. "Herdr could be
installed" is not availability; the check is the pane, not the wish.

## Transport mapping

| Task-loop concept | Native in-harness subagent | Herdr transport |
|---|---|---|
| Dispatch implementer | harness subagent call (fresh context) | `pane split --current --no-focus --cwd <worktree>` → `agent start <name> --kind <KIND> --pane <id>` → `agent prompt <name> "<brief prompt>" --wait --timeout <MS>` |
| Wait for result | child result delivered to controller | `--wait` returns on the first settled `idle` / `done` / `blocked` |
| Live child (fix rounds 1–3 resume) | `write agent://<id>` (message a live subagent) | `agent prompt <name> "<findings>" --wait` — the pane is still up, context intact |
| Fresh dispatch (rounds 4–5, new model) | new subagent call | stop/replace the pane occupant, `agent start` again with the new kind/args, re-prompt |
| Blocked child (approval/question UI) | — | `agent get <name>` + `agent read <name> --source recent-unwrapped --lines 120`, inspect, ask the user before answering |
| Identity record | agent id from dispatch result | agent name + pane id, recorded in the ledger |

The name follows the pane occupant and is cleared when the agent exits —
pick a useful unique name per task (`impl-task-3`), matching
`[a-z][a-z0-9_-]{0,31}`.

## Dispatch procedure (substitutes task-loop step 1)

1. Record BASE (`git rev-parse HEAD`) — unchanged from the task loop.
2. **Create the sibling pane.** `herdr pane layout --pane "$HERDR_PANE_ID"`
   to read the caller's geometry; split a wide pane right, a narrow or tall
   pane down. Always `--cwd <worktree> --no-focus` — the worker starts in
   its worktree, and the user's focus stays on the controller. Read the
   new pane id from `.result.pane.pane_id` (parse JSON; never predict IDs).
3. **Start the agent.** `herdr agent start <name> --kind <KIND> --pane
   <pane-id>` (native agent args after `--`, e.g. the model flag from Model
   Selection). `start` returns only after the agent is detected and ready;
   on `agent_not_ready`, wait for idle before prompting — do not re-issue
   `start`.
4. **Prompt with the brief.** The prompt text is the same contract the
   native path sends (implementer-prompt.md): one line of scene, the brief
   path, the interfaces, the ambiguity resolutions, the report-file path
   and report contract. Keep it short — it is pasted into an interactive
   terminal, not a structured call. `agent prompt <name> "<prompt>" --wait
   --timeout <MS>`; the timeout is generous (the work is real: 30–60
   minutes per task is normal), and `--wait` alone is enough — do not add
   `--until`.
5. **Record identity.** Ledger line: `Task <N>: herdr <name> @ <pane-id>
   (kind <KIND>)` — fix-loop rounds 1–3 resume this agent.

## Settle, block, and stall

- **`idle` / `done`** — the agent is ready for input; the turn settled.
  Read the result the normal way: the report file on disk (the contract is
  unchanged), the ledger, `git log`. The terminal transcript is a
  convenience, not the signal.
- **`blocked`** — Herdr recognized an approval or question UI. `agent get`
  + `agent read` to see it, then ask the user (the four-things-stop-you
  rule applies to worker blocks the same way). `agent prompt` into a
  blocked agent is rejected with `agent_blocked` before any input is sent —
  inspect first.
- **`agent_prompt_stalled` / `timeout`** — the prompt may or may not have
  been delivered; a timeout does not prove it wasn't. Inspect with `agent
  get` + `agent read` before re-sending; never blindly re-submit.

## Report fallback

When the terminal transcript is too long to be the report (a long-running
agent that printed more than it returned), use the herdr skill's file
fallback: ask the agent to write its report as Markdown in the scratch
workspace (the report-file path is already in the prompt) and reply with
only the path. Do not request file output in the *initial* prompt — the
report-file contract in implementer-prompt.md already covers it.

## Fan-out (Herdr workers)

Fan-out mode is Herdr's natural shape — one worker per pane, no worktree
races by construction:

- One sibling pane per domain (same geometry rule, `--no-focus`,
  `--cwd <that worktree>`), `agent start` + `agent prompt` per pane.
  The scope gate (≤5 concurrent worktrees, waves of ≤5) is unchanged —
  five panes is also the point where the terminal stops being readable.
- The fan-out ledger is unchanged; add the agent name + pane id to each
  row. Integrate from the ledger + git, as always.
- Do not close a worker's pane before its work is merged; a closed pane's
  id is never reused, and the worker's scratch recovery
  (`checkpoint-state.js scratch-restore <id>`) needs the branch+worktree
  pair the ledger row names.

## Cleanup

A worker's pane is the controller's to close — it was created for the
task. Close it after the task completes (and its fix loop is done) and
after any scratch the worker still needs is restored. Never close a pane
the user created (herdr safety rule).
