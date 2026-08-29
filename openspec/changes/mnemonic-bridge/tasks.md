# Tasks: Mnemonic Bridge

## Epic 1: GitNexus skills bridge (done)

- [x] 1-1 Copy six `gentleman-ai/.agents/skills/gitnexus-*` skills into `.agents/skills/`
- [x] 1-2 Add "Mnemonic Integration" section to `gitnexus-cli`, `gitnexus-debugging`, `gitnexus-exploring`, `gitnexus-impact-analysis`, `gitnexus-refactoring`
- [x] 1-3 Add "Working alongside Mnemonic" section to `gitnexus-guide`
- [x] 1-4 Update `.agents/AGENTS.md` GitNexus table to `.agents/skills/` paths + mnemonic-memory rows

## Epic 2: `change_summary` tool

- [ ] 2-1 Add `change_summary` handler to `internal/mnemonic/mcp/tools_*.go`
- [ ] 2-2 Implement diff digest logic (file list, symbol list, one-line summary)
- [ ] 2-3 Add `GET /changes/summary` (or `POST`) to `internal/mnemonic/http/server.go`
- [ ] 2-4 Unit tests: clean tree, uncommitted changes, explicit repo, no-git error

## Epic 3: `graph_stale` signal

- [ ] 3-1 Extend `code_status` response JSON with `graph_stale: bool`
- [ ] 3-2 Implement staleness check: compare `.gitnexus/` mtime (or registry entry) against last `code_index` run
- [ ] 3-3 Unit tests: both fresh, mnemonic fresh + graph stale, graph absent, backward-compat parse

## Epic 4: `session_diff_summary` tool

- [ ] 4-1 Add `session_diff_summary` handler to `internal/mnemonic/mcp/tools_*.go`
- [ ] 4-2 Implement Markdown diff block generator (file list, added/removed symbols, risk line)
- [ ] 4-3 Unit tests: clean tree, uncommitted changes, explicit repo, missing session

## Epic 5: Tests + docs

- [ ] 5-1 Integration test: agent calls `change_summary` → appends to `mem_session_summary` → next session reads it
- [ ] 5-2 Integration test: `code_status` returns `graph_stale` alongside `stale`
- [ ] 5-3 Update `docs/13-mnemonic.md` with the three new tools
- [ ] 5-4 Update `docs/14-gitnexus-mnemonic-bridge.md` to point at this change

## Out of scope (rides on other changes)

- `graph_search` — capability owned by `openspec/changes/mnemonic-graph/`
- `graph_cypher` — capability owned by `openspec/changes/mnemonic-cypher/`
- `sync_index` — deferred; `graph_stale` + the two existing `code_status`/`status` reads are sufficient
