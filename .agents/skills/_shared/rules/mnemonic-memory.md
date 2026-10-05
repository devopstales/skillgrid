# Mnemonic Memory Convention (shared across all Skillgrid skills)

Save shape, session protocol, and topic-key rules for all Mnemonic memory saves.

## Save Shape

```
mem_save(
  title:      "skillgrid/{topic}/{artifact}",   # MUST equal topic_key exactly
  topic_key:  "skillgrid/{topic}/{artifact}",   # same as title
  type:       "architecture",                    # always architecture for SDD artifacts
  scope:      "project",                         # always project
  session_id: "{sid}",                           # active session
  content:    "{full markdown content}"
)
```

- `title == topic_key` exactly.
- `scope: "project"` always.
- Pass the active `session_id`.
- There is **no** `project:` parameter and **no** `capture_prompt` field in the Mnemonic schema — omit both.
- `topic_key` upserts — saving the same `topic_key` again replaces the observation in place.

## Artifact Naming

When working a change `{YYYY-MM-DD-<topic>}`:

| Artifact | topic_key |
|---|---|
| Briefing | `skillgrid/{YYYY-MM-DD-<topic>}/briefing` |
| Blueprint | `skillgrid/{YYYY-MM-DD-<topic>}/blueprint` |
| Tasks | `skillgrid/{YYYY-MM-DD-<topic>}/tasks` |
| Execution progress | `skillgrid/{YYYY-MM-DD-<topic>}/execution-progress` |
| Report | `skillgrid/{YYYY-MM-DD-<topic>}/report` — one observation, upserted: qa saves the QA half, reflect upserts the full report (retro half) |
| Review report | `skillgrid/{YYYY-MM-DD-<topic>}/review-report` |

### Execution progress body

ADR-0020. The Execution Ledger under `.skillgrid/sdd/` is the record. This observation is the index. Upsert it after each completed task or wave. Do not copy the ledger, per-agent rows, or review diffs. Do not spawn or claim SDD work through `team_spawn_task` / `agent_pull_next_task`.

```
change: {YYYY-MM-DD-<topic>}
ledger: .skillgrid/sdd/<plan>/progress.md
wave_ledger: .skillgrid/sdd/<plan>/parallel-ledger.md
tail: {active task or wave, and the last ruling}
```

Omit `wave_ledger` when the run has no parallel wave. When the file and this observation disagree, the file wins.

## Recovery Ladder

```
1. mem_context(limit: 5)                    # fast: recent session summaries
1.5. mem_inject_session(query: "<topic>")   # deeper hybrid retrieval when context is insufficient (Layer 2)
2. mem_search(query: "<topic>")             # FTS5 over observations
3. mem_get_observation(id)                  # full untruncated body (previews are 300 chars)
```

**Critical**: `mem_search` returns 300-char previews. A preview of a 2000-char blueprint loses most of it. Always `mem_get_observation(id)` before relying on it as an acceptance criterion or constraint. `mem_inject_session` items are 200-char snippets — same rule: `mem_get_observation(id)` for full content.

## Session Protocol

```
1. mem_session_start(title: "skillgrid/<goal>")   # once per agent session
2. ... work (mem_save as needed) ...
3. mem_session_summary(session_id, summary)       # before saying "done"
4. mem_session_end(session_id, summary)
```

After compaction / "FIRST ACTION REQUIRED": FIRST call `mem_session_summary` with the compacted content, then `mem_context`, only then continue.

## Session summary structure

`mem_session_summary` takes a structured body with **exactly six sections** — not five. Every section must be filled (use "None" where genuinely empty, never omit a heading):

1. **Goal** — what we were working on this session.
2. **Instructions** — user preferences or constraints discovered (skip if none).
3. **Discoveries** — technical findings, gotchas, non-obvious learnings.
4. **Accomplished** — completed items with key details.
5. **Next Steps** — what remains to be done, for the next session.
6. **Relevant Files** — paths and what each does or what changed.

This is the single source of truth for the shape — skills that close a session (mnemonic, reflect) reference it rather than restating it.
