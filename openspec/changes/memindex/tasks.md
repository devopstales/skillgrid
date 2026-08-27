# Tasks: MemIndex

## Epic 1: Storage layer

- [ ] 1-1 Create `internal/memindex/store/` with SQLite schema (sessions, observations, files, chunks, web_cache, meta)
- [ ] 1-2 Implement FTS5 virtual tables and sync triggers
- [ ] 1-3 Implement migrations and schema versioning
- [ ] 1-4 Implement project resolution (git remote URL normalization, fallback)

## Epic 2: Memory layer

- [ ] 2-1 Create `internal/memindex/memory/` with observation CRUD
- [ ] 2-2 Implement `mem_save`, `mem_search`, `mem_context`, `mem_get_observation`
- [ ] 2-3 Implement `mem_session_start`, `mem_session_end`, `mem_session_summary`
- [ ] 2-4 Implement `mem_suggest_topic_key`

## Epic 3: Code index layer

- [ ] 3-1 Create `internal/memindex/codeindex/` with file scanner and chunker
- [ ] 3-2 Implement incremental indexing (mtime + content hash)
- [ ] 3-3 Implement `code_status`, `code_index`, `code_search`, `code_read`
- [ ] 3-4 Implement exclude globs and file size cap

## Epic 4: Web cache layer

- [ ] 4-1 Create `internal/memindex/webcache/` with save/lookup/search
- [ ] 4-2 Implement cache key normalization per source
- [ ] 4-3 Implement TTL per source and stale detection
- [ ] 4-4 Implement 256KB entry cap

## Epic 5: MCP transport

- [ ] 5-1 Create `internal/memindex/mcp/` with tool handlers
- [ ] 5-2 Implement `cmd/mcp.go` (`skillgrid mcp` command)
- [ ] 5-3 Register all `mem_*`, `code_*`, `web_*` tools

## Epic 6: HTTP transport

- [ ] 6-1 Create `internal/memindex/http/` with REST handlers
- [ ] 6-2 Implement `cmd/serve.go` (`skillgrid serve` command)
- [ ] 6-3 Implement session, memory, code, web cache endpoints
- [ ] 6-4 Implement `GET /health` and `GET /context`

## Epic 7: Agent plugins

- [ ] 7-1 Create `plugins/memindex/shared/memory-protocol.md`
- [ ] 7-2 Create `plugins/memindex/opencode/memindex.ts`
- [ ] 7-3 Create `plugins/memindex/cursor/memindex.mdc`
- [ ] 7-4 Implement `skillgrid setup opencode|kilocode|cursor` commands

## Epic 8: CLI integration

- [ ] 8-1 Implement `cmd/index.go` (`skillgrid index` command)
- [ ] 8-2 Implement `config.d/indexing.yaml` default profile
- [ ] 8-3 Fork `engram-memory` skill → `memindex-memory`

## Epic 9: Testing

- [ ] 9-1 Unit tests for store migrations and FTS sync
- [ ] 9-2 Unit tests for memory CRUD and search
- [ ] 9-3 Unit tests for code indexing (cold/warm/incremental)
- [ ] 9-4 Unit tests for web cache (dedup, TTL, stale)
- [ ] 9-5 Unit tests for HTTP handlers
- [ ] 9-6 Unit tests for MCP tool dispatch
- [ ] 9-7 Integration test: `skillgrid mcp` + `skillgrid serve` share one SQLite store

## Epic 10: Validation

- [ ] 10-1 Run `go build ./... && go test ./...` — all green
- [ ] 10-2 Run `openspec validate memindex --type change --strict` before archive
- [ ] 10-3 Archive via `openspec archive memindex`
