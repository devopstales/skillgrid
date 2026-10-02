# Execution coordination lives in the SDD ledger; Mnemonic stores an index

---
status: "accepted"
supersedes: none
date: 2026-10-02
---

## Context and Problem Statement

A live run has two places that look like a team queue. `.skillgrid/sdd/<plan>/` holds `progress.md` or `parallel-ledger.md`: task status, `Owns:`, `Needs:`, rulings, and gate results. Mnemonic also has a hybrid teams plane (`teams`, `tasks`, `reviews`, plus `team_spawn_task` / `agent_pull_next_task` and the other agent tools) whose markdown sits under `.skillgrid/files/`. The execution skills never call those tools. Cursor workers are push-dispatched by a parent. They do not sit in a pull loop. Leaving both as owners means a later session cannot tell which status is real.

## Considered Options

- Move dispatch, claim, and review state into the Mnemonic teams tables and `.skillgrid/files/`
- Keep only the SDD ledger and write nothing to Mnemonic during execution
- Keep the SDD ledger as the execution record and upsert a short Mnemonic index that points at it

## Decision Outcome

Chosen option: "Keep the SDD ledger as the execution record and upsert a short Mnemonic index that points at it", because the parent re-reads the full ledger after compaction, and a Mnemonic observation is a searchable pointer. The teams tables stay in the engine. SDD execution does not spawn, claim, or complete work through them.

The record for a run is `.skillgrid/sdd/<plan>/progress.md`. A parallel wave uses `.skillgrid/sdd/<plan>/parallel-ledger.md` for the wave rows and still mirrors through the same observation. When `mnemonic.enabled` is true, each completed task or wave upserts `skillgrid/{YYYY-MM-DD-<topic>}/execution-progress`. The body names the ledger path and the current tail (active task or wave, last ruling). It does not copy per-agent rows or the review diffs. When the file and the observation disagree, the file wins.

### Consequences

- Good, because recovery after compaction is a file read plus `git log`, and a fresh session can still find the ledger path from one observation.
- Bad, because the teams MCP and HTTP routes remain registered and look like a second queue. A caller that uses them for an SDD run splits status from the ledger. `git clean -fdx` deletes the ledger; the index then points at a missing file and git history is the fallback.
