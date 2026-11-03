# Agent Continuity (deep-research)

How a long-horizon experimental research run survives compaction, session
restarts, and (optionally) unattended wall-clock operation.

## The resume handle

`research-state.yaml` in the run folder is the **single source of truth for
where the run is**. Before doing anything else after a session start,
compaction, or loop tick:

1. Read `research-state.yaml` — status, current hypothesis, best value,
   outer-loop cycle count.
2. Read `findings.md` — the accumulated understanding (Current Understanding,
   Lessons and Constraints, Open Questions).
3. Read the tail of `research-log.md` — the last 3-5 entries, to know what
   was done most recently and what the next step is.
4. Continue from there. Do not re-bootstrap if state exists.

**If no state file exists**, bootstrap (see SKILL.md Step 1) and create it.

## Compaction rule

If you see a compaction message or "FIRST ACTION REQUIRED":

1. **Immediately** write the current run state to `research-state.yaml` and
   append the last actions to `research-log.md` — persist what was done before
   compaction.
2. Read the three files above to recover context.
3. Only then continue.

Without step 1, everything done before compaction is lost from the run's
durable state.

## Optional: unattended wall-clock loop

For long-horizon runs where the human is not watching, set up a wall-clock
rhythm so the run does not stop after one cycle. This is **separate from the
research loops** (inner/outer) — it is a nudge, not a phase boundary.

### Claude Code

```
/loop 20m Continue deep-research. Read research-state.yaml and findings.md.
Re-read the deep-research SKILL.md occasionally to stay aligned. Step back
and reflect holistically — is the research making real progress? Are you
deepening understanding or just running experiments? If stalling, pivot or
search literature for new ideas. Keep making research progress — never idle,
never stop. Update findings.md, research-log.md, and research-state.yaml when
there is new progress. Git commit periodically. Show the human your progress
by preparing a report in to_human/. Only when the research is truly complete,
finalize and archive.
```

### OpenCode / agent-browser / MCP

If the host supports a recurring task or heartbeat, create one bound to the
current session with the same message. Verify it is registered and enabled.
If the tick fires while mid-experiment, just continue — the tick is a nudge.

### What the loop does (each tick)

1. Read `research-state.yaml` and `findings.md` — remember where you are.
2. Check if anything is broken (failed experiment, stalled run, error).
3. If on track → keep working on whatever you were doing.
4. If stuck or something is wrong → step back, diagnose, fix, then continue.
5. Never idle. Always be making progress.

## When to stop the loop

Stop the wall-clock loop when the outer loop decides **CONCLUDE** (see the
direction rubric in SKILL.md). After concluding: write the final findings,
generate the final report, archive the run folder, and let the loop end.
