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
| QA report | `skillgrid/{YYYY-MM-DD-<topic>}/qa-report` |
| Review report | `skillgrid/{YYYY-MM-DD-<topic>}/review-report` |

## Recovery Ladder

```
1. mem_context(limit: 5)                    # fast: recent session summaries
2. mem_search(query: "<topic>")             # FTS5 over observations
3. mem_get_observation(id)                  # full untruncated body (previews are 300 chars)
```

**Critical**: `mem_search` returns 300-char previews. A preview of a 2000-char blueprint loses most of it. Always `mem_get_observation(id)` before relying on it as an acceptance criterion or constraint.

## Session Protocol

```
1. mem_session_start(title: "skillgrid/<goal>")   # once per agent session
2. ... work (mem_save as needed) ...
3. mem_session_summary(session_id, summary)       # before saying "done"
4. mem_session_end(session_id, summary)
```

After compaction / "FIRST ACTION REQUIRED": FIRST call `mem_session_summary` with the compacted content, then `mem_context`, only then continue.
