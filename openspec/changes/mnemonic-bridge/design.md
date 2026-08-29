# Mnemonic Bridge — Design

## Division of labor

| System           | Owns                              | Does NOT own                           |
|------------------|-----------------------------------|----------------------------------------|
| GitNexus         | Graph snapshot, relationships     | History, web cache, code text search   |
| Mnemonic `mem_*` | Observations, sessions, web cache | Graph, callers, impact, taint          |
| Mnemonic `code_*`| BM25 chunk search, index health   | Relationship queries, process grouping |
| New `change_summary` | Git-diff digest               | Graph analysis (delegate to GitNexus)  |
| New `graph_stale`   | Single staleness read         | Full graph rebuild (delegate to GitNexus) |
| New `session_diff_summary` | Diff → session body   | Working-tree writes (read-only)         |

The three new tools are *bridges*, not competitors: they glue the two systems
together at the workflow level without duplicating either one's core job.

## `change_summary`

Input: cwd (default) or explicit `repo` path.
Output: JSON with `files`, `symbols`, `summary` (one-line natural language).

Implementation: reuse the existing `git diff --name-only` + symbol-pipeline
output shape from GitNexus `detect_changes`. No new dependencies.

Why not extend `detect_changes` instead? Because `detect_changes` lives in
GitNexus (KuzuDB) and requires a running MCP server. `change_summary` must
work in the Mnemonic session-close path even when GitNexus is not installed.

## `graph_stale` on `code_status`

Current `code_status` response:
```json
{"file_count": 123, "chunk_count": 456, "last_indexed": "...", "stale": false}
```

Extended response:
```json
{"file_count": 123, "chunk_count": 456, "last_indexed": "...", "stale": false, "graph_stale": false}
```

`graph_stale` is computed by checking the mtime of `.gitnexus/` (or the
GitNexus registry entry for this repo) against the last `code_index` run.
No graph query required — it is a file-system staleness check, cheap.

## `session_diff_summary`

Input: `session_id` (required), optional `repo` path.
Output: appends a `## Diff Summary` block to the existing session summary
body, or returns the block as a string for the agent to fold in.

The block shape:
```
## Diff Summary
- Changed files: 3 (src/auth/login.ts, src/auth/middleware.ts, src/api/routes.ts)
- Added symbols: 2 (authenticateUser, TokenRefresh)
- Removed symbols: 1 (validateUser)
- Risk: MEDIUM (2 direct callers per mnemonic-memory note)
```

The agent calls this after `detect_changes` (GitNexus) or after a manual
`git diff`, then calls `mem_session_summary` with the enriched body. The
tool does NOT write the summary itself — it only produces the diff block so
the agent can decide what to include.

## Skills bridge

The six GitNexus skills already carry "Mnemonic Integration" sections. Each
section maps one GitNexus step to one Mnemonic call. The join point table
lives in `gitnexus-guide/SKILL.md` ("Working alongside Mnemonic"). No code
changes are needed to sustain this — it is a documentation + discipline
addition, captured here as the `mnemonic-bridge-skills` done-capability.

## What we explicitly do NOT build

- No `mem_graph` tool family duplicating GitNexus `query`/`context`/`impact`.
- No Cypher-aware `code_search` — BM25 is its value.
- No `sync_index` orchestrator — `graph_stale` plus the two existing
  `code_status`/`status` reads is sufficient and keeps each system sovereign.
- `graph_search` and `graph_cypher` are out of scope — they ride on
  `mnemonic-graph` and `mnemonic-cypher` respectively.
