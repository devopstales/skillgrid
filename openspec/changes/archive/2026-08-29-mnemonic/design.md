## Context

Skillgrid's default profile currently bundles Engram (memory) + GitNexus (code structure) + ccc (semantic search). These are external binaries with separate install, MCP namespace, and index lifecycle. MemIndex replaces them with a single Go binary using SQLite + FTS5, exposing two transports (MCP stdio for agent tools, HTTP for plugin hooks) over one service layer.

The design is informed by: Engram (memory protocol), Hermes Memory (layered scopes, bi-temporal), OpenClaw (incremental hash indexing), Neuledge (local-first web cache), srclight (hybrid FTS + embeddings).

## Goals / Non-Goals

**Goals:**
- One binary, one MCP, YAML-driven install
- Memory protocol compatible with existing `engram-memory` skill (tool rename only)
- Local, offline, portable — one `.sqlite` file per project
- Incremental re-index uses mtime + content hash
- Web research results cacheable with TTL + dedup
- OpenCode/Kilo plugins for session lifecycle and compaction recovery
- Cursor MCP + rule fallback (no TypeScript plugin surface)

**Non-Goals:**
- Call graphs, blast-radius, coordinated rename (GitNexus territory — opt-in overlay)
- Cloud sync / team replication
- Replacing Context7/Exa/DeepWiki MCP servers (they remain live fetchers)
- Vector embeddings / Ollama dependency (v2)
- Indexing node_modules, .git, build artifacts
- Auto-ingest raw tool-call firehose

## Decisions

### 1. MemIndex inside skillgrid-cli vs external binary

**Decision:** Build MemIndex inside `skillgrid-cli` as `internal/memindex/` packages.

**Alternatives considered:**
- Keep Engram + GitNexus (status quo) — rejected: two MCP servers, two index lifecycles, external binaries
- Separate MemIndex binary — rejected: violates single-binary reproducibility goal

**Rationale:** Aligns with skillgrid's "config + one binary" thesis. Go + SQLite (modernc.org/sqlite) matches Engram's zero-dependency ethos.

### 2. Two transports (MCP + HTTP) over one service layer

**Decision:** Expose MCP stdio (`skillgrid mcp`) and HTTP REST (`skillgrid serve`) over a shared `internal/memindex/service` layer.

**Alternatives considered:**
- MCP only — rejected: OpenCode/Kilo plugins need HTTP for session lifecycle hooks
- HTTP only — rejected: Cursor and other agents expect MCP for tool calls

**Rationale:** Same split as Engram (`engram mcp` + `engram serve`). No duplicated business logic.

### 3. Memory taxonomy (Hermes-inspired)

**Decision:** Observations use `type` + `scope` for consistent routing. Bi-temporal evolution (v1.1) via `observation_history`.

**Alternatives considered:**
- Flat observation list — rejected: no way to distinguish standing rules from session notes
- Separate tables per type — rejected: too many tables, harder to search across

**Rationale:** Hermes model proven in OpenCode plugin. SQLite remains canonical.

### 4. Web cache keys and TTL

**Decision:** Dedup via normalized `cache_key` per source. Default TTL: Context7 30d, Exa 7d, DeepWiki 14d, fetch 7d, manual none.

**Alternatives considered:**
- Single TTL for all sources — rejected: library docs change slower than web content
- No TTL — rejected: stale docs mislead agents

**Rationale:** Neuledge model. `web_cache_lookup` returns `stale: true` when expired so agents re-fetch.

## Risks / Trade-offs

- **MCP lib maturity in Go** -> Use `github.com/mark3labs/mcp-go` (same as Engram)
- **FTS trigram index size** -> Exclude large/generated paths; cap file size
- **Agents confuse mem vs code** -> Explicit tool descriptions + AGENTS.md rule block
- **Web cache grows unbounded** -> TTL per source + 256KB entry cap
- **Stale cached docs mislead agent** -> `web_cache_lookup` returns `stale: true`
- **SQLite lock contention** -> WAL + busy timeout; single `skillgrid serve` process
- **Port 7438 conflict** -> Configurable `SKILLGRID_MEMINDEX_PORT`
- **Zombie `skillgrid serve` processes** -> Plugin health-check reuse; document `pkill`

## Migration Plan

1. Implement `internal/memindex/service` + `store` (SQLite schema, migrations)
2. Implement `memory` layer (observations, FTS, `mem_*` tools)
3. Implement `codeindex` layer (incremental scan, `code_*` tools)
4. Implement `webcache` layer (TTL, dedup, `web_*` tools)
5. Implement `mcp` transport (`skillgrid mcp`)
6. Implement `http` transport (`skillgrid serve`)
7. Implement `agent-plugins` (OpenCode/Kilo/Cursor)
8. Fork `engram-memory` skill → `memindex-memory`
9. Update `config.d/indexing.yaml` default profile

**Rollback:** Remove `skillgrid-memindex` MCP entry from `mcp.yaml`, re-add Engram/GitNexus entries, delete `~/.skillgrid/memindex/`.

## Decisions (continued)

### 5. AGENTS.md protocol injection

**Decision:** Inject the Mnemonic Memory Protocol into AGENTS.md between managed markers (`<!-- BEGIN MNEMONIC MEMORY PROTOCOL -->` / `<!-- END MNEMONIC MEMORY PROTOCOL -->`), matching the engram protocol block pattern.

**Alternatives considered:**
- Plugin-only injection — rejected: agents without plugin support (Cursor, Codex, plain Claude) would miss the protocol
- Separate `MNEMONIC.md` file — rejected: fragments the memory protocol across files, harder to discover

**Rationale:** AGENTS.md is the universal agent instruction file. The marker pattern allows idempotent setup and preserves the block during `project-context` refreshes.

### 6. Skill integration

**Decision:** Integrate mnemonic tool references into existing skillgrid skills (brainstorming, openspec-apply-change, openspec-explore, project-context, spec-as-source) rather than creating a separate skill per integration point.

**Alternatives considered:**
- Standalone `mnemonic-skills` skill — rejected: fragments discovery, skills are loaded individually
- Only update `memindex-memory` — rejected: other skills (brainstorming, apply-change) also benefit from memory/code search

**Rationale:** Each skill already has a "context gathering" or "exploration" phase where `mem_search` / `code_search` references fit naturally.

## Open Questions

1. **Profile migration:** Default new installs to `memindex` or keep Engram until v1 ships?
2. **Chunk size:** 80-line default vs AST-aware chunks in v1.1?
3. **Docs zone:** Index `docs/superpowers/` in v1.1 for IDD/BDD agent workflows?
4. **Auto-capture:** Gryph policy template to call `web_cache_save` after exa/context7 MCP tools — ship in v1 or v1.1?
5. **Portable `.db` export:** Commit team web cache for pinned Context7 versions — v2?
