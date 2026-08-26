# Skillgrid Mnemonic — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a built-in SQLite memory + code FTS + web research cache to skillgrid-cli, exposed as **`skillgrid mcp` (stdio)** + **`skillgrid serve` (HTTP)** for OpenCode/Kilo plugins, and **`skillgrid index`**.

**Architecture:** Pure Go under `internal/memindex/`. Shared **`service/`** layer; thin **MCP** and **HTTP** adapters. HTTP on `127.0.0.1:7438` for plugin session/compaction hooks ([Engram `engram serve` pattern](https://github.com/Gentleman-Programming/engram/blob/main/docs/PLUGINS.md)). Agents use MCP; OpenCode/Kilo plugins use HTTP + MCP.

**Tech Stack:** Go 1.22+, modernc.org/sqlite, github.com/mark3labs/mcp-go, gopkg.in/yaml.v3 (existing).

**Spec:** [2026-08-26-skillgrid-mnemonic-design.md](../specs/2026-08-26-skillgrid-mnemonic-design.md)

## Development status

> **Last updated:** 2026-08-26

| Phase | Plan tasks | Status | Git |
|-------|------------|--------|-----|
| **v1** | 1–18, 10–16, 12 | **COMPLETE** | branch `feat/skillgrid-mnemonic` @ `254fed5` |
| v1.1 | 19–22, 24–26 | not started | — |
| v2 | 23 | not started | — |

**Delivered (v1):** `internal/mnemonic/` — SQLite store, memory + sessions, incremental code index + FTS, web cache, MCP tools (`mem_*`, `code_*`, `web_*`), HTTP `skillgrid serve`, CLI `index` / `web` / `setup`, OpenCode/Kilo plugin + Cursor rule, `config.d/indexing.yaml`, install gating on `profile: mnemonic`.

**Manual verification:** `skillgrid index` (111 files / 304 chunks), Kilo MCP connected, HTTP `/health` ok, `setup kilocode` → `kilo.jsonc` + AGENTS.md.

**Merge:** PR `release/2` ← `feat/skillgrid-mnemonic` — [open compare](https://github.com/devopstales/skillgrid/compare/release/2...feat/skillgrid-mnemonic?expand=1)

**Cutover (post-merge):** set `indexing.profile: mnemonic`; uncomment `skillgrid-mnemonic` in `config.d/mcp.yaml`; demote Engram to opt-in.

## Global Constraints

- All MemIndex code lives in `skillgrid-cli/` — no separate binary, no Node/Python runtime.
- Data directory default: `~/.skillgrid/memindex/` (override via `SKILLGRID_MEMINDEX_DATA_DIR`).
- WAL SQLite; foreign keys ON; migrations versioned in `internal/memindex/store/migrations/`.
- Dual transport v1: **MCP stdio** (`skillgrid mcp`) + **HTTP** (`skillgrid serve`, default `127.0.0.1:7438`).
- `SKILLGRID_MEMINDEX_PORT` default `7438` (Engram uses 7437 — avoid collision when co-installed).
- `SKILLGRID_MEMINDEX_URL` for plugin base URL (default `http://127.0.0.1:7438`).
- All business logic in `internal/memindex/service/` — MCP and HTTP are thin adapters only.
- HTTP binds `127.0.0.1` only; optional `SKILLGRID_HTTP_TOKEN` bearer auth (unset = open localhost, Engram-style).
- OpenCode/Kilo plugins auto-start `skillgrid serve` when `GET /health` fails.
- Cursor: MCP + rule only — no HTTP plugin.
- **MCP tool responses:** return raw JSON arrays/objects only — no preamble phrases (`Found N results`, `Index status:`, etc.). Inspired by [open-codebase-index](https://github.com/Helweg/open-codebase-index) token optimization.

## Design influences

| Source | What we adopt | When |
|--------|---------------|------|
| [Engram](https://github.com/Gentleman-Programming/engram) | Memory protocol, MCP+HTTP split, session lifecycle | v1 |
| [OpenClaw / srclight](https://www.pingcap.com/blog/local-first-rag-using-sqlite-ai-agent-memory-openclaw/) | Incremental hash indexing, FTS tokenizers, RRF hybrid | v1 FTS; v2 embeddings |
| [open-codebase-index (OCBI)](https://github.com/Helweg/open-codebase-index) | Tiered code tools, semantic chunks, branch catalog, health check, tool routing table | v1.1 code layer |
| [Neuledge](https://neuledge.com/blog/2026-02-19/local-first-documentation-for-ai) | Web cache TTL, offline research recall | v1 |
| [MemWeave](https://levelup.gitconnected.com/memweave-zero-infra-ai-agent-memory-with-markdown-and-sqlite-no-vector-database-required-cf3869efc840) | Temporal decay on session logs, optional markdown mirror | v1.1 memory search |
| [Hermes Memory](https://github.com/realchendahuang/opencode-hermes-memory) | Layered scopes/types, bi-temporal replace+history, proactive plugin injection | v1 protocol types; v1.1 plugin + tools |

**Explicit non-goals from OCBI:** Node/Rust NAPI runtime, embeddings required for v1, in-process call graph (`call_graph` / `pr_impact` → GitNexus opt-in).

**Explicit non-goals from Hermes:** Markdown-as-sole-truth (SQLite remains canonical; optional export only), plugin-only tools without MCP (`mem_*` names stay Engram-compatible), OpenCode-internal LLM idle learning loop in Go core (optional plugin-only behavior).

**OCBI positioning:** Reference implementation for code-layer v1.1+; teams needing OCBI-level semantic code search today can run OCBI alongside MemIndex (`mem_*`/`web_*` = MemIndex; semantic code = OCBI or GitNexus).

**Hermes positioning:** Reference for **proactive OpenCode plugin hooks** beyond Engram parity; Hermes can coexist for markdown-first users (`memory_*` plugin tools vs MemIndex MCP).

---

## File structure

| File | Action | Responsibility |
|------|--------|----------------|
| `skillgrid-cli/go.mod` | Modify | Add modernc.org/sqlite, mcp-go |
| `skillgrid-cli/cmd/main.go` | Modify | Register `mcp`, `serve`, `index`, `setup` subcommands |
| `skillgrid-cli/cmd/mcp.go` | Create | `skillgrid mcp` stdio server entry |
| `skillgrid-cli/cmd/serve.go` | Create | `skillgrid serve` HTTP API |
| `skillgrid-cli/cmd/index.go` | Create | `skillgrid index`, `--status` |
| `skillgrid-cli/internal/memindex/service/service.go` | Create | Facade over memory/code/web packages |
| `skillgrid-cli/internal/memindex/http/server.go` | Create | REST routes → service |
| `skillgrid-cli/internal/memindex/http/server_test.go` | Create | httptest for /health, /sessions, /search |
| `skillgrid-cli/internal/memindex/store/store.go` | Create | DB open, migrations, project paths |
| `skillgrid-cli/internal/memindex/store/migrations/001_initial.sql` | Create | Schema from spec |
| `skillgrid-cli/internal/memindex/store/store_test.go` | Create | Migration + CRUD smoke |
| `skillgrid-cli/internal/memindex/project/resolve.go` | Create | Git remote / cwd project id |
| `skillgrid-cli/internal/memindex/memory/service.go` | Create | Save, search, context, summary |
| `skillgrid-cli/internal/memindex/memory/service_test.go` | Create | FTS round-trip tests |
| `skillgrid-cli/internal/memindex/codeindex/scanner.go` | Create | Walk, hash, exclude globs |
| `skillgrid-cli/internal/memindex/codeindex/chunker.go` | Create | Line-bounded chunks |
| `skillgrid-cli/internal/memindex/codeindex/indexer.go` | Create | Incremental upsert |
| `skillgrid-cli/internal/memindex/codeindex/indexer_test.go` | Create | mtime/hash skip tests |
| `skillgrid-cli/internal/memindex/search/fts.go` | Create | Code + memory FTS queries |
| `skillgrid-cli/internal/memindex/mcp/server.go` | Create | Tool registration + handlers |
| `skillgrid-cli/internal/memindex/mcp/tools_memory.go` | Create | mem_* handlers → service |
| `skillgrid-cli/internal/memindex/mcp/tools_code.go` | Create | code_* handlers → service |
| `skillgrid-cli/internal/memindex/mcp/response.go` | Create | Raw JSON formatter (no preamble) |
| `skillgrid-cli/internal/memindex/mcp/tools_web.go` | Create | web_* handlers → service |
| `skillgrid-cli/cmd/web.go` | Create | `skillgrid web search|status` CLI |
| `skillgrid-cli/internal/memindex/setup/setup.go` | Create | Agent setup orchestrator |
| `skillgrid-cli/internal/memindex/setup/opencode.go` | Create | opencode.json + plugin copy |
| `skillgrid-cli/internal/memindex/setup/kilocode.go` | Create | kilo MCP + AGENTS.md + bridge copy |
| `skillgrid-cli/internal/memindex/setup/cursor.go` | Create | mcp.json + memindex.mdc |
| `skillgrid-cli/internal/memindex/setup/setup_test.go` | Create | Idempotent upsert tests |
| `skillgrid-cli/cmd/setup.go` | Create | `skillgrid setup <agent>` CLI |
| `plugins/memindex/opencode/memindex.ts` | Create | OpenCode plugin (Engram v1 hooks) |
| `plugins/memindex/opencode/hooks.ts` | Create | Hermes-style proactive hooks (v1.1) |
| `plugins/memindex/shared/injection.ts` | Create | Auto-inject, error prefetch, correction detect |
| `plugins/memindex/cursor/memindex.mdc` | Create | Cursor rule template |
| `plugins/memindex/shared/memory-protocol.md` | Create | Protocol source text |
| `skillgrid-cli/cmd/steps.go` | Modify | Install step: call setup per agent |
| `skillgrid-cli/cmd/install.go` | Modify | Gate setup on indexing.profile |
| `skillgrid-cli/internal/memindex/config/load.go` | Create | indexing.yaml loader |
| `config.d/indexing.yaml` | Create | Default memindex profile |
| `docs/07-plugins.md` | Modify | MemIndex plugin section (OpenCode/Kilo/Cursor) |
| `docs/04-mcp-servers.md` | Modify | MemIndex MCP docs |
| `docs/03-config-reference.md` | Modify | indexing.yaml schema |
| `docs/02-usage.md` | Modify | index lifecycle + agent rules |

---

### Task 1: Dependencies and command scaffold

**Files:**
- Modify: `skillgrid-cli/go.mod`, `skillgrid-cli/go.sum`
- Modify: `skillgrid-cli/cmd/main.go`
- Create: `skillgrid-cli/cmd/mcp.go`, `skillgrid-cli/cmd/index.go`
- Test: `skillgrid-cli/cmd/main_test.go`

**Interfaces:**
- Produces: `runMCP()`, `runServe(args []string) error`, `runIndex(args []string) error` stubs

- [ ] **Step 1: Add dependencies**

```bash
cd skillgrid-cli && go get modernc.org/sqlite@v1.45.0 github.com/mark3labs/mcp-go@v0.44.0
```

- [ ] **Step 2: Extend main.go switch**

Add cases `"mcp"` → `runMCP()`, `"serve"` → `runServe(rest)`, `"index"` → `runIndex(rest)`, and `"setup"` → `runSetup(rest)`.

- [ ] **Step 3: Stub commands**

`cmd/mcp.go`:

```go
package main

func runMCP() {
    // TODO Task 8: memindex/mcp.Start()
    println("skillgrid mcp: not yet implemented")
}
```

`cmd/index.go`:

```go
package main

func runIndex(args []string) error {
    // TODO Task 6: incremental index
    println("skillgrid index: not yet implemented")
    return nil
}
```

- [ ] **Step 4: Update main_test.go**

Assert `skillgrid mcp` and `skillgrid index --help` exit 0 once help text added to `printUsage()`.

- [ ] **Step 5: Verify**

Run: `go build ./... && go test ./cmd/...`
Expected: PASS

---

### Task 2: SQLite store and migrations

**Files:**
- Create: `internal/memindex/store/store.go`
- Create: `internal/memindex/store/migrations/001_initial.sql`
- Create: `internal/memindex/store/store_test.go`

**Interfaces:**
- Produces: `type Store struct`, `func Open(dataDir, projectID string) (*Store, error)`, `func (s *Store) Close() error`

- [ ] **Step 1: Write failing test**

```go
func TestStoreMigrations(t *testing.T) {
    dir := t.TempDir()
    s, err := store.Open(dir, "test-project")
    if err != nil { t.Fatal(err) }
    defer s.Close()
    var v int
    if err := s.DB.QueryRow("SELECT schema_version FROM index_meta WHERE key='schema_version'").Scan(&v); err != nil {
        t.Fatal(err)
    }
    if v != 1 { t.Fatalf("schema_version=%d", v) }
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/memindex/store/... -v -run TestStoreMigrations`

- [ ] **Step 3: Implement 001_initial.sql**

Tables: `sessions`, `observations`, `observations_fts` (+ triggers), `files`, `chunks`, `chunks_fts` (+ triggers), **`web_cache`, `web_cache_fts` (+ triggers)**, `index_meta`. Set `schema_version=1`.

FTS5 for chunks uses `tokenize='trigram'`; observations uses default porter.

- [ ] **Step 4: Implement store.Open**

- Path: `{dataDir}/{projectID}.sqlite`
- Enable WAL, FK, busy timeout 5000ms
- Run embedded migration SQL on first open

- [ ] **Step 5: Run test — expect PASS**

---

### Task 3: Project resolution

**Files:**
- Create: `internal/memindex/project/resolve.go`
- Create: `internal/memindex/project/resolve_test.go`

**Interfaces:**
- Produces: `func Resolve(cwd string) (projectID string, err error)`

- [ ] **Step 1: Test with temp git repo**

```go
func TestResolveFromGitRemote(t *testing.T) {
    // init temp repo with origin url → expect stable normalized id
}
```

- [ ] **Step 2: Implement**

Priority: `.skillgrid/config.json` `project` field → `git remote get-url origin` normalized → `{basename}-{hash(cwd)}`.

- [ ] **Step 3: Verify**

Run: `go test ./internal/memindex/project/... -v`

---

### Task 4: Memory service (save + search)

**Files:**
- Create: `internal/memindex/memory/service.go`
- Create: `internal/memindex/memory/service_test.go`

**Interfaces:**
- Produces:
  - `func (s *Service) Save(ctx, SaveInput) (observationID int64, error)`
  - `func (s *Service) Search(ctx, query string, matchMode string, limit int) ([]Observation, error)`
  - `func (s *Service) Get(ctx, id int64) (Observation, error)`

- [ ] **Step 1: Failing test — save then FTS search**

```go
func TestMemorySaveSearch(t *testing.T) {
    // Save observation type=decision title="Chose SQLite"
    // Search "SQLite" → 1 hit
}
```

- [ ] **Step 2: Implement Save**

Fields: title, type, content, scope, topic_key, session_id. Compute `normalized_hash` for dedup within 24h window (simplified: hash title+content+type).

Topic_key upsert: update existing row in same project+scope+topic_key.

- [ ] **Step 3: Implement Search**

Use `observations_fts MATCH ?` with `matchMode` all (AND) vs any (OR). Join observations for metadata. Limit default 20.

- [ ] **Step 4: Verify**

Run: `go test ./internal/memindex/memory/... -v`

---

### Task 5: Session summary + context

**Files:**
- Modify: `internal/memindex/memory/service.go`
- Modify: `internal/memindex/memory/service_test.go`

**Interfaces:**
- Produces:
  - `func (s *Service) SessionSummary(ctx, sessionID, summary string) error`
  - `func (s *Service) SessionStart(ctx, directory string) (sessionID string, error)`
  - `func (s *Service) SessionEnd(ctx, sessionID, summary string) error`
  - `func (s *Service) RecentContext(ctx, limit int) ([]Session, error)`

- [ ] **Step 1: Test session lifecycle**

Create session via `SessionStart` → `SessionSummary` → appears in `RecentContext`.

- [ ] **Step 2: Implement SessionStart / SessionEnd**

Insert/update `sessions` row with project from cwd resolution.

- [ ] **Step 3: Implement SessionSummary — update sessions.summary, ended_at**

- [ ] **Step 4: Implement RecentContext — last N sessions with summaries**

- [ ] **Step 5: Verify tests PASS**

---

### Task 6: Code indexer (incremental)

**Files:**
- Create: `internal/memindex/codeindex/scanner.go`
- Create: `internal/memindex/codeindex/chunker.go`
- Create: `internal/memindex/codeindex/indexer.go`
- Create: `internal/memindex/codeindex/indexer_test.go`

**Interfaces:**
- Produces: `func (idx *Indexer) Run(ctx, root string, cfg Config) (Stats, error)`

- [ ] **Step 1: Test hash skip**

Create file, index once, index again unchanged → `FilesSkipped=1`, `FilesIndexed=0`.

- [ ] **Step 2: Implement scanner**

Walk root; apply include/exclude glob matching; skip files >512KB; compute SHA256 content hash.

- [ ] **Step 3: Implement chunker**

Split by `chunk_lines` / `chunk_overlap` from config (defaults 80/10).

- [ ] **Step 4: Implement indexer**

Upsert `files`; replace `chunks` + FTS rows on change; delete chunks when file removed.

- [ ] **Step 5: Wire `skillgrid index`**

`cmd/index.go` calls `project.Resolve`, `config.Load`, `indexer.Run`.

- [ ] **Step 6: Verify**

Run: `go test ./internal/memindex/codeindex/... -v`
Manual: `go run ./cmd/.. index --status` on skillgrid-cli repo (after build).

---

### Task 7: Code FTS search

**Files:**
- Create: `internal/memindex/search/fts.go`
- Create: `internal/memindex/search/fts_test.go`

**Interfaces:**
- Produces: `func CodeSearch(db, query string, limit int) ([]CodeHit, error)`
  - `CodeHit`: Path, StartLine, EndLine, Snippet, Score

- [ ] **Step 1: Test search finds known string in indexed fixture**

- [ ] **Step 2: Implement BM25-style rank via FTS5 `bm25(chunks_fts)`**

- [ ] **Step 3: Verify tests PASS**

---

### Task 8: MCP server + memory tools

**Files:**
- Create: `internal/memindex/mcp/server.go`
- Create: `internal/memindex/mcp/tools_memory.go`
- Modify: `skillgrid-cli/cmd/mcp.go`

**Interfaces:**
- Produces: `func Start() error` — blocks on stdio MCP loop

- [ ] **Step 1: Register tools**

`mem_save`, `mem_search`, `mem_context`, `mem_get_observation`, `mem_session_start`, `mem_session_end`, `mem_session_summary`, `mem_suggest_topic_key`

Tool descriptions copy Engram semantics from spec (What/Why/Where/Learned).

- [ ] **Step 2: Implement handlers**

Resolve project from `os.Getwd()` per call; open Store per project.

- [ ] **Step 3: Raw tool responses (OCBI convention)**

Create `mcp/response.go` — all handlers return structured JSON only (arrays/objects). **No** summary strings like `Found 3 results` or `Index status:` in tool text. Add test asserting handler output has no leading prose.

- [ ] **Step 4: Wire cmd/mcp.go**

```go
func runMCP() {
    if err := mcp.Start(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

- [ ] **Step 5: Manual smoke**

Run: `echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | go run . mcp` (or use MCP inspector) — expect tool list.

---

### Task 9: MCP code tools

**Files:**
- Create: `internal/memindex/mcp/tools_code.go`
- Modify: `internal/memindex/mcp/server.go`

**Interfaces:**
- Produces: handlers for `code_status`, `code_index`, `code_search`, `code_read`

- [ ] **Step 1: Implement code_status** — return counts + last indexed timestamp

- [ ] **Step 2: Implement code_index** — invoke Indexer.Run on cwd git root

- [ ] **Step 3: Implement code_search** — delegate to search.CodeSearch

- [ ] **Step 4: Implement code_read** — fetch chunk or line range from store

- [ ] **Step 5: Integration test**

Test: index fixture repo in temp dir → MCP code_search returns expected path.

- [ ] **Step 6: Document v1 code tool ladder in tool descriptions**

v1 ships `code_status` → `code_search` → `code_read`. v1.1 adds low-token `code_peek` / `code_context` (Task 19). Descriptions should steer agents: check status when stale; search before grepping large unknown areas; read only after search narrows location.

---

### Task 17: Web cache service + MCP tools

**Files:**
- Create: `internal/memindex/webcache/service.go`
- Create: `internal/memindex/webcache/service_test.go`
- Create: `internal/memindex/webcache/keys.go` — cache_key normalization per source
- Create: `internal/memindex/mcp/tools_web.go`
- Create: `skillgrid-cli/cmd/web.go`
- Modify: `skillgrid-cli/cmd/main.go` — `web` subcommand

**Interfaces:**
- Produces:
  - `func (s *Service) Save(ctx, SaveWebInput) (id int64, error)`
  - `func (s *Service) Lookup(ctx, LookupInput) (LookupResult, error)` — hit | miss | stale
  - `func (s *Service) Search(ctx, query string, source string, freshOnly bool, limit int) ([]WebHit, error)`
  - `func (s *Service) Get(ctx, id int64) (WebEntry, error)`

- [ ] **Step 1: Failing test — save Context7-shaped entry then lookup**

```go
func TestWebCacheContext7RoundTrip(t *testing.T) {
    // Save source=context7 library_id=/vercel/next.js version=v15 query=middleware
    // Lookup same keys → hit, fresh=true
    // Search "middleware" → 1 result
}
```

- [ ] **Step 2: Implement cache_key + TTL from indexing.yaml**

Sources: `context7`, `exa`, `deepwiki`, `fetch`, `manual`. Enforce 256KB max body.

- [ ] **Step 3: Implement Save with upsert on (project, source, cache_key)**

- [ ] **Step 4: Implement Lookup** — return `{status: "hit"|"miss"|"stale", id, fetched_at, expires_at}`

- [ ] **Step 5: Implement Search** — FTS5 on `web_cache_fts`; optional `fresh_only` filters `expires_at > now`

- [ ] **Step 6: Register MCP tools**

`web_cache_lookup`, `web_cache_save`, `web_cache_search`, `web_cache_get`, `web_cache_status`

- [ ] **Step 7: Wire `skillgrid web search` and `skillgrid web status`**

- [ ] **Step 8: Verify**

Run: `go test ./internal/memindex/webcache/... -v`

---

### Task 18: Shared service facade + HTTP server

**Files:**
- Create: `internal/memindex/service/service.go`
- Create: `internal/memindex/http/server.go`
- Create: `internal/memindex/http/routes_memory.go`
- Create: `internal/memindex/http/routes_code.go`
- Create: `internal/memindex/http/routes_web.go`
- Create: `internal/memindex/http/server_test.go`
- Create: `skillgrid-cli/cmd/serve.go`
- Modify: `skillgrid-cli/internal/memindex/mcp/*.go` — delegate to `service` (refactor after Tasks 4–8, 17)

**Interfaces:**
- Produces: `type Service struct` with methods matching spec HTTP + MCP operations
- Produces: `func StartHTTP(addr string, svc *Service) error`
- Produces: `runServe(args []string) error` — default port from `SKILLGRID_MEMINDEX_PORT` or `7438`

- [ ] **Step 1: Implement Service facade**

Wrap `memory`, `codeindex`, `webcache` packages — single entry for MCP and HTTP.

- [ ] **Step 2: Failing HTTP test**

```go
func TestHTTPSessionsAndSearch(t *testing.T) {
    // POST /sessions {directory: tempRepo}
    // POST /observations {...}
    // GET /search?q=keyword → 1 hit
    // GET /health → ok
}
```

- [ ] **Step 3: Implement routes per spec**

Minimum v1: `/health`, `/sessions`, `/sessions/{id}/end`, `/context`, `/observations`, `/search`, `/code/status`, `/code/search`, `/web/lookup`, `/web/cache`, `/web/search`, `/web/status`

- [ ] **Step 4: Wire `skillgrid serve`**

Bind `127.0.0.1:<port>` only; read `SKILLGRID_HTTP_TOKEN` for optional bearer auth on write routes.

- [ ] **Step 5: Refactor MCP handlers to call `service`**

Ensure MCP `mem_save` and HTTP `POST /observations` produce identical DB rows.

- [ ] **Step 6: Verify**

Run: `go test ./internal/memindex/http/... -v`
Manual: `skillgrid serve &` → `curl http://127.0.0.1:7438/health`

---

### Task 10: config.d + docs + install profile

**Files:**
- Create: `config.d/indexing.yaml`
- Modify: `config.d/mcp.yaml`
- Modify: `docs/03-config-reference.md`, `docs/04-mcp-servers.md`, `docs/02-usage.md`

- [ ] **Step 1: Add indexing.yaml**

```yaml
profile: memindex
memindex:
  include: ["**/*.go", "**/*.ts", "**/*.tsx", "**/*.md"]
  exclude: ["**/node_modules/**", "**/.git/**", "**/dist/**"]
  chunk_lines: 80
  chunk_overlap: 10
  embeddings: false
  web_cache:
    enabled: true
    max_entry_bytes: 262144
    ttl:
      context7: 720h
      exa: 168h
      deepwiki: 336h
      fetch: 168h
      manual: 0
    sources: [context7, exa, deepwiki, fetch, manual]
  http:
    enabled: true
    host: 127.0.0.1
    port: 7438
    auto_start: plugin
  # v1.1+ (see Tasks 19–22)
  watch: false                 # file watcher; debounced re-index
  branch_aware: true           # branch catalog (Task 21)
  search:
    fusion_strategy: rrf       # v2 when embeddings: true (Task 23)
  plugin:
    # v1.1 Hermes-style proactive hooks (Task 26)
    standing_inject: true      # type=standing always in system prompt
    auto_inject: true          # chat.message → GET /search
    inject_score_min: 0.4
    inject_max_per_turn: 2
    error_prefetch: true       # tool.execute.after bash failure
    correction_detect: true    # rule-based on chat.message
    compaction_flush: true     # summarize + save before GET /context
    idle_review: false         # optional LLM review on session.idle (plugin-only)
gitnexus:
  enabled: false
```

- [ ] **Step 2: Add mcp.yaml entry** (commented alternative to engram until cutover)

```yaml
  skillgrid-memindex:
    type: local
    command: [skillgrid, mcp]
```

- [ ] **Step 3: Document agent rule**

Add to `docs/02-usage.md`:

> `mem_*` = decisions/history; `code_*` = repo text search; **`web_*` = cache Context7/Exa/DeepWiki/fetch before re-querying**; GitNexus opt-in for impact graphs.

- [ ] **Step 4: Document index lifecycle**

`skillgrid index` after clone; re-run when stale; `code_status` from agent.

- [ ] **Step 5: Update superpowers README track index**

Add Wave 3.5 or note MemIndex as alternative to 3.2 GitNexus-only track.

---

### Task 11: Skills fork (memindex-memory)

**Files:**
- Create: `config.d/skills/memindex-memory/SKILL.md` (from engram-memory template)
- Modify: `config.d/skills.yaml` — add skill; mark engram skills opt-in
- Modify: `docs/05-skills.md` — list memindex-memory in bootstrap section

- [ ] **Step 1: Copy engram-memory skill; replace server references**

- [ ] **Step 2: Add `code_*` and `web_*` usage section**

When to call `code_search` vs `mem_search` vs `web_cache_lookup` / `web_cache_search`. Include OCBI ladder: `code_status` → (v1.1) `code_peek`/`code_context` → `code_search` → `code_read`; grep for exact identifiers.

- [ ] **Step 3: Update exa-search and mcp-deepwiki skills** — add post-fetch `web_cache_save` step

- [ ] **Step 4: Update 05-skills.md rollout table**

---

### Task 13: Memory protocol + plugin assets

**Files:**
- Create: `plugins/memindex/shared/memory-protocol.md`
- Create: `plugins/memindex/cursor/memindex.mdc` (body includes protocol via setup render)
- Create: `config.d/rules/memindex-memory-protocol.md` (optional symlink/copy for AGENTS.md merge)

**Interfaces:**
- Produces: `func ProtocolMarkdown() string` reading embedded or repo-relative `memory-protocol.md`

- [ ] **Step 1: Write memory-protocol.md**

Fork `config.d/rules/engram-memory-protocol.md`; add sections:

- **CODE SEARCH** — when to call `code_search` / `code_status` vs grep
- **WEB RESEARCH CACHE** — lookup → remote MCP → save workflow for context7/exa/deepwiki/fetch
- **SESSION START** — plugin calls `mem_session_start`
- Keep Engram-compatible `mem_save` / `mem_session_summary` / compaction blocks
- **MEMORY TAXONOMY** (Hermes-inspired) — document `type` + `scope` conventions:

| `type` | `scope` | Purpose | Retrieval |
|--------|---------|---------|-----------|
| `standing` | `user` or `project` | Hard rules — always injected | Plugin L0, not search-ranked |
| `preference` | `user` | User workflow prefs | `mem_search` |
| `convention` | `project` | Project norms | `mem_search` |
| `decision` | `project` | Architecture choices (Engram default) | `mem_search` |
| `bugfix` / `lesson` | `project` | Failures and fixes | `mem_search` + error prefetch |
| `correction` | `project` | User corrections | Instant save via plugin |
| `discovery` | `project` or `global` | Tool/env quirks | `mem_search` |
| `session_log` | `project` | Dated session notes | `mem_search` with decay (Task 24) |

- **TOOL ROUTING TABLE** (OCBI-style) — paste into protocol:

| Need | Tool |
|------|------|
| Decisions, bugs, session history | `mem_search` / `mem_context` |
| Route a repo question cheaply (v1.1) | `code_context` |
| Likely files/symbols, minimal tokens (v1.1) | `code_peek` |
| Full matching source in repo | `code_search` → `code_read` |
| Exact identifier / exhaustive text | grep (not MemIndex) |
| Cached research before remote MCP | `web_cache_lookup` → `web_cache_search` |
| Callers, blast radius, rename | GitNexus (opt-in) |

- [ ] **Step 2: Write cursor/memindex.mdc template**

Frontmatter: `alwaysApply: true`, `description: Skillgrid MemIndex protocol`

- [ ] **Step 3: Verify protocol mentions both mem_* and code_***

---

### Task 14: OpenCode + Kilo TypeScript plugin (HTTP + MCP)

**Files:**
- Create: `plugins/memindex/opencode/memindex.ts`
- Create: `plugins/memindex/shared/http-client.ts` — fetch wrapper for `SKILLGRID_MEMINDEX_URL`
- Test: manual — load in OpenCode after setup

Reference: [Engram OpenCode plugin](https://github.com/Gentleman-Programming/engram/blob/main/plugin/opencode/engram.ts) — **use HTTP for plugin hooks**, MCP remains in `opencode.json` for agent tool calls.

**v1 hooks (Engram parity):** auto-start, system transform, session start, compaction inject, code status nudge.

**v1.1 hooks (Hermes parity — Task 26):** auto-inject, error prefetch, correction detect, compaction flush, standing L0.

- [ ] **Step 1: Implement ensureServer()**

```typescript
async function ensureServer(baseUrl: string): Promise<void> {
  try {
    const r = await fetch(`${baseUrl}/health`);
    if (r.ok) return;
  } catch { /* spawn */ }
  spawn("skillgrid", ["serve"], { detached: true, stdio: "ignore" });
  await waitForHealth(baseUrl, 5000);
}
```

- [ ] **Step 2: Session lifecycle via HTTP**

`POST /sessions` on plugin init; store `session_id` in plugin state.

- [ ] **Step 3: chat.system.transform** — inject memory-protocol.md

- [ ] **Step 4: Compaction hook** — `GET /context` + inject `mem_session_summary` reminder

- [ ] **Step 5: GET /code/status** nudge on session start

- [ ] **Step 6: Kilo uses same plugin** (copied via setup bridge) — verify HTTP base URL from env or default `7438`

- [ ] **Step 7: Document manual test**

OpenCode: plugin starts serve → `/health` ok → session created → MCP tools still work for agent chat.

---

### Task 15: `skillgrid setup` command (OpenCode, Kilo, Cursor)

**Files:**
- Create: `skillgrid-cli/cmd/setup.go`
- Create: `skillgrid-cli/internal/memindex/setup/setup.go`
- Create: `skillgrid-cli/internal/memindex/setup/opencode.go`
- Create: `skillgrid-cli/internal/memindex/setup/kilocode.go`
- Create: `skillgrid-cli/internal/memindex/setup/cursor.go`
- Create: `skillgrid-cli/internal/memindex/setup/setup_test.go`
- Modify: `skillgrid-cli/cmd/main.go` — register `setup` subcommand

**Interfaces:**
- Produces:
  - `func RunSetup(agent string, repoRoot string, dryRun bool) error`
  - `func SetupOpenCode(home, repoRoot string, dryRun bool) error`
  - `func SetupKiloCode(home, repoRoot string, dryRun bool) error`
  - `func SetupCursor(home string, dryRun bool) error`

- [ ] **Step 1: Write failing idempotency test**

Run `SetupOpenCode` twice → second run does not duplicate `mcp.skillgrid-memindex` or `plugin` entries.

- [ ] **Step 2: Implement SetupOpenCode**

- Copy `plugins/memindex/opencode/memindex.ts` → `~/.config/opencode/plugins/memindex.ts`
- Upsert `mcp.skillgrid-memindex` in `opencode.json` (preserve JSONC comments via read-modify-write or existing jsonc helper)
- Append `memindex.ts` path to `plugin` array if absent

- [ ] **Step 3: Implement SetupKiloCode**

- Upsert MCP in `~/.config/kilo/opencode.json`
- Write/replace marker block in `~/.config/kilo/AGENTS.md` between `BEGIN SKILLGRID MEMINDEX` / `END`
- **Bridge copy** (first-write-wins):
  - `~/.config/opencode/plugins/memindex.ts` → `~/.config/kilo/plugins/memindex.ts`
  - optional `tui.json` copy (same as engram in `steps.go:193-200`)

- [ ] **Step 4: Implement SetupCursor**

- Upsert `~/.cursor/mcp.json` `mcpServers.skillgrid-memindex`
- Write `~/.cursor/rules/memindex.mdc` from template + protocol body

- [ ] **Step 5: Wire CLI**

```bash
skillgrid setup opencode
skillgrid setup kilocode
skillgrid setup cursor
```

- [ ] **Step 6: Run setup tests**

Run: `go test ./internal/memindex/setup/... -v`

---

### Task 16: Install flow integration + docs

**Files:**
- Modify: `skillgrid-cli/cmd/steps.go` — `installMemIndexPlugins(agents, baseDir, dryRun)`
- Modify: `skillgrid-cli/cmd/install.go` — call after superpowers when `indexing.profile == memindex`
- Modify: `docs/07-plugins.md` — MemIndex section parallel to engram
- Modify: `docs/01-installation.md` — step 5 mention setup commands

- [ ] **Step 1: Implement installMemIndexPlugins**

```go
// Gated: only when config.d/indexing.yaml profile is memindex
if hasAgent(agents, "opencode") { setup.RunSetup("opencode", repoRoot, dryRun) }
if hasAgent(agents, "kilo")      { setup.RunSetup("kilocode", repoRoot, dryRun) }
if hasAgent(agents, "cursor")    { setup.RunSetup("cursor", repoRoot, dryRun) }
```

Dry-run: log `[dry-run] skillgrid setup opencode` etc.

- [ ] **Step 2: Test install dry-run output**

Run: `go test ./cmd/... -run MemIndexSetup`

- [ ] **Step 3: Document three-agent transport table**

| Agent | MCP | HTTP plugin |
|-------|-----|---------------|
| OpenCode | `skillgrid mcp` in opencode.json | `memindex.ts` → `skillgrid serve` |
| Kilo Code | same | copied plugin + AGENTS.md marker |
| Cursor | `skillgrid mcp` in mcp.json | rule only (no HTTP) |

- [ ] **Step 4: Cross-check mcp.yaml server id matches setup writes**

---

### Task 12: End-to-end verification

- [ ] **Step 1: Full test suite**

Run: `cd skillgrid-cli && go build ./... && go test ./...`

- [ ] **Step 2: Index skillgrid-cli repo**

Run: `./bin/skillgrid index` from repo root

- [ ] **Step 3: MCP manual test**

Run: `code_search` for `"MergeMCP"` or `"memindex"` — expect hits in `skillgrid-cli/`. Verify tool output is raw JSON (no `Found N results` preamble).

- [ ] **Step 4: Memory round-trip**

Via MCP: `mem_save` → restart mcp process → `mem_search` → hit.

- [ ] **Step 5: Web cache round-trip**

Via MCP: `web_cache_save` (source=context7, query=test) → `web_cache_lookup` → hit → `web_cache_search` → hit.

- [ ] **Step 6: add-mcp sync dry-run** (if Wave 1.2 complete)

Confirm `skillgrid-memindex` appears in planned upserts.

- [ ] **Step 7: Agent plugin + HTTP smoke**

Run: `skillgrid setup opencode` then start OpenCode.

Verify:
- Plugin auto-starts `skillgrid serve` (or reuses running instance)
- `curl http://127.0.0.1:7438/health` returns ok
- `POST /sessions` creates session (via plugin or manual curl)
- MCP `mem_save` still works during agent chat
- Kilo bridge: `memindex.ts` copied; same HTTP behavior

---

## Post-v1 tasks (OCBI + MemWeave + Hermes learnings)

### Task 19: Tiered code tools — `code_peek` + `code_context` (v1.1)

**Reference:** [OCBI tool ladder](https://github.com/Helweg/open-codebase-index#recommended-workflow) — `codebase_context` / `codebase_peek` before full search.

**Files:**
- Modify: `internal/memindex/search/fts.go`, `internal/memindex/mcp/tools_code.go`, `internal/memindex/http/routes_code.go`
- Modify: `plugins/memindex/shared/memory-protocol.md`

**Interfaces:**
- `code_peek(query, limit)` → `[{path, start_line, end_line, symbol?, score}]` — no full chunk bodies
- `code_context(query, limit)` → bounded evidence pack: top paths + symbol names + one-line snippets (token budget ~2k)

- [ ] **Step 1: Failing tests** — peek returns paths without bodies; context respects token cap

- [ ] **Step 2: Implement peek/context in search layer** — reuse FTS ranks; strip bodies in formatter

- [ ] **Step 3: Register MCP + HTTP routes** — `GET /code/peek`, `GET /code/context`

- [ ] **Step 4: Update memory-protocol routing table** — mark tools as available

- [ ] **Step 5: Verify** — agent smoke: peek → search → read uses fewer tokens than search-only

---

### Task 20: Tree-sitter semantic chunking + `symbols_fts` (v1.1)

**Reference:** [OCBI indexing flow](https://github.com/Helweg/open-codebase-index/blob/main/ARCHITECTURE.md#indexing-flow) — parse → semantic chunk → store.

**Files:**
- Create: `internal/memindex/codeindex/parser/` (Go tree-sitter bindings or exec helper — spike first)
- Create: `internal/memindex/store/migrations/002_symbols.sql`
- Modify: `internal/memindex/codeindex/chunker.go`, `indexer.go`

**Scope:** Go + TypeScript/JavaScript first (skillgrid-cli targets). Line chunks remain fallback when parser unsupported.

- [ ] **Step 1: Spike** — index `skillgrid-cli/**/*.go`; compare chunk count vs line chunker

- [ ] **Step 2: Migration 002** — populate `symbols_fts` from parsed defs (func, type, method)

- [ ] **Step 3: Semantic chunker** — split at function/class boundaries + docstring overlap (OCBI pattern)

- [ ] **Step 4: `code_definition` MCP tool (v1.1)** — lookup symbol by name → authoritative path + line range

- [ ] **Step 5: Re-index skillgrid-cli** — verify `code_search` for `"Resolve"` hits `project/resolve.go` function block

---

### Task 21: Branch catalog + branch-scoped search (v1.1)

**Reference:** [OCBI branch-aware indexing](https://github.com/Helweg/open-codebase-index/blob/main/ARCHITECTURE.md#why-branch-aware-indexing) — content-hash dedup + branch membership catalog.

**Files:**
- Create: `internal/memindex/store/migrations/003_branch_catalog.sql`
- Create: `internal/memindex/codeindex/branch.go`
- Modify: `indexer.go`, `search/fts.go`

**Schema:**
```sql
branch_catalog(branch TEXT, chunk_id INTEGER, PRIMARY KEY(branch, chunk_id))
```

- [ ] **Step 1: On index** — record current git branch; upsert chunk IDs into `branch_catalog`

- [ ] **Step 2: Content-hash dedup** — same `content_hash` across branches reuses chunk row (OCBI dedup model)

- [ ] **Step 3: Search filter** — default `code_search` scopes to active branch; optional `all_branches=true`

- [ ] **Step 4: Test** — two branches, shared file unchanged → warm index skips re-chunk; branch-only file scoped correctly

---

### Task 22: `code_health` + optional file watch (v1.1)

**Reference:** OCBI `index_health_check`, file watcher with debounce.

**Files:**
- Create: `internal/memindex/codeindex/health.go`
- Modify: `config.d/indexing.yaml`, `internal/memindex/mcp/tools_code.go`
- Optional: `internal/memindex/codeindex/watcher.go` (off by default)

- [ ] **Step 1: Implement `code_health`** — schema version match, orphaned FTS rows, stale `index_meta`, last successful index time

- [ ] **Step 2: MCP + HTTP** — `code_health` tool; `GET /code/health`

- [ ] **Step 3: Plugin nudge** — OpenCode plugin calls `/code/health` on session start when `/code/status` reports stale

- [ ] **Step 4: Optional watch** — `indexing.watch: false` default; when true, debounced 500ms re-index on save (document goroutine lifecycle; no duplicate `skillgrid serve`)

---

### Task 23: Hybrid embeddings + eval harness (v2)

**Reference:** OCBI `fusionStrategy: "rrf"` default; [evaluation harness](https://github.com/Helweg/open-codebase-index/tree/main/benchmarks).

**Files:**
- Create: `internal/memindex/store/migrations/004_embeddings.sql`
- Create: `internal/memindex/search/hybrid.go` — RRF merge (k=60)
- Create: `internal/memindex/embeddings/ollama.go`
- Create: `skillgrid-cli/internal/memindex/eval/` — golden queries fixture

- [ ] **Step 1: Schema** — `chunks.embedding_json BLOB`; optional sqlite-vec virtual table

- [ ] **Step 2: Ollama provider** — auto-detect like OCBI (`embeddingProvider: auto` → Ollama first)

- [ ] **Step 3: RRF fusion** — default `fusionStrategy: rrf` in `indexing.yaml`; FTS + vector candidates

- [ ] **Step 4: Golden eval** — 15–20 queries on skillgrid-cli with expected paths; `go test ./internal/memindex/eval/...`

- [ ] **Step 5: Document** — embeddings optional; v1 FTS remains valid when `embeddings: false`

---

### Task 24: Memory search quality — temporal decay (v1.1, MemWeave)

**Reference:** [MemWeave temporal decay](https://levelup.gitconnected.com/memweave-zero-infra-ai-agent-memory-with-markdown-and-sqlite-no-vector-database-required-cf3869efc840) — evergreen vs dated observations.

**Files:**
- Modify: `internal/memindex/memory/service.go`, `observations` schema or type metadata
- Modify: `mem_search` ranking

- [ ] **Step 1: Observation classes** — `type=evergreen` (no decay) vs session logs / dated notes (decay)

- [ ] **Step 2: `mem_search` post-rank** — optional `decay_half_life_days` (default 30); exponential multiplier on FTS score

- [ ] **Step 3: Test** — book-club-style fixture: recent session log outranks stale semantic match for “most recent decision” queries

- [ ] **Step 4: Optional markdown export** — `skillgrid mem export` → `memory/` dir for git audit (deferred detail in spec open questions)

---

### Task 25: Bi-temporal memory — `mem_replace` + `mem_history` (v1.1, Hermes)

**Reference:** [Hermes bi-temporal evolution](https://github.com/realchendahuang/opencode-hermes-memory) — `memory_replace` moves superseded entries to history; active retrieval sees current truth only.

**Files:**
- Create: `internal/memindex/store/migrations/005_observation_history.sql`
- Modify: `internal/memindex/memory/service.go`, `internal/memindex/mcp/tools_memory.go`
- Modify: `internal/memindex/http/routes_memory.go`

**Schema:**
```sql
observation_history(
  id INTEGER PK,
  observation_id INTEGER,
  superseded_by INTEGER,
  title TEXT, content TEXT, type TEXT, scope TEXT, topic_key TEXT,
  superseded_at TEXT
)
```

**Interfaces:**
- `mem_replace(topic_key, ...)` — upsert active row; append prior version to `observation_history`; exclude history from default `mem_search`
- `mem_history(topic_key | id)` — return supersession chain

- [ ] **Step 1: Failing tests** — replace twice → search returns latest only; history has 2 prior versions

- [ ] **Step 2: Implement Replace + History in memory service**

- [ ] **Step 3: Register MCP tools** — `mem_replace`, `mem_history` (keep `mem_save` for new keys)

- [ ] **Step 4: HTTP routes** — `PUT /observations/{topic_key}`, `GET /observations/history`

- [ ] **Step 5: Update memindex-memory skill** — use `mem_replace` when correcting prior decisions

---

### Task 26: Hermes proactive plugin hooks (v1.1)

**Reference:** [Hermes event hooks](https://github.com/realchendahuang/opencode-hermes-memory#event-hooks) — auto-injection, error prefetch, correction detection, compaction flush.

**Files:**
- Create: `plugins/memindex/opencode/hooks.ts`
- Create: `plugins/memindex/shared/injection.ts`
- Modify: `plugins/memindex/opencode/memindex.ts` — register hooks when `indexing.yaml` plugin flags enabled
- Modify: `internal/memindex/http/routes_memory.go` — `GET /search` accepts `type`, `scope`, `min_score` for plugin use

- [ ] **Step 1: L0 standing inject** — on `chat.system.transform`, fetch `type=standing` for user+project via HTTP; prepend before protocol (Hermes `STANDING.md` equivalent)

- [ ] **Step 2: Auto-inject on `chat.message`** — `GET /search?q=<user message>&min_score=0.4&limit=2`; dedupe per session; inject as system/user context block (max 2/turn)

- [ ] **Step 3: Correction detect** — rule-based patterns on user message (`no, actually`, `that's wrong`, `use X not Y`) → immediate `POST /observations` with `type=correction`

- [ ] **Step 4: Error prefetch on `tool.execute.after`** — if bash/shell tool exit ≠ 0 → `GET /search?q=<cmd>&type=bugfix,lesson&limit=2`; inject on next turn

- [ ] **Step 5: Compaction flush** — on `experimental.session.compacting`: prompt agent (via injected reminder) to `mem_session_summary` + save key lessons **before** `GET /context` inject (Hermes flush review)

- [ ] **Step 6: Optional `session.idle` review** — off by default (`idle_review: false`); when enabled, rate-limited reminder to call `mem_session_summary` (no Go LLM — plugin nudge only)

- [ ] **Step 7: Kilo bridge** — same hooks in copied plugin; test auto-inject + error prefetch

- [ ] **Step 8: Document coexistence** — if user also runs Hermes plugin, warn about duplicate memory systems in `docs/07-plugins.md`

---

## Spec coverage checklist

| Spec section | Task |
|--------------|------|
| SQLite schema | Task 2 |
| Project resolution | Task 3 |
| Memory layer (Engram-compatible) | Tasks 4–5, 8 |
| Incremental code index | Task 6 |
| Code FTS search | Task 7 |
| **Web cache layer** | Task 17 |
| **HTTP API (`skillgrid serve`)** | **Task 18** |
| **Shared service layer** | Task 18 |
| MCP stdio server | Tasks 8–9, 17–18 |
| CLI `skillgrid index` | Task 6 |
| config.d indexing.yaml | Task 10 |
| Agent skills | Task 11 |
| Memory protocol assets | Task 13 |
| OpenCode/Kilo HTTP plugin | Task 14 |
| `skillgrid setup` (opencode/kilocode/cursor) | Task 15 |
| Install flow plugin wiring | Task 16 |
| Raw MCP responses (no preamble) | Task 8 |
| OCBI tool routing table | Task 13 |
| Success criteria | Task 12 |
| **Tiered code tools (peek/context)** | **Task 19** |
| **Tree-sitter + symbols_fts** | **Task 20** |
| **Branch catalog** | **Task 21** |
| **code_health + watch** | **Task 22** |
| **Hybrid RRF + eval harness** | **Task 23** |
| **Memory temporal decay** | **Task 24** |
| **Memory taxonomy (types/scopes)** | **Task 13** |
| **Bi-temporal replace + history** | **Task 25** |
| **Hermes proactive plugin hooks** | **Task 26** |

## Deferred (post-v1.1 plan)

- v1.1 (remaining): `mem_capture_passive`, unified `search` (mem+code+web), Gryph auto-capture hook for exa/context7 → `web_cache_save`, optional markdown memory export, optional `session.idle` LLM review (plugin-only, off by default)
- v2: Task 23 (embeddings + RRF + golden eval)
- v3: GitNexus companion or graph layer evaluation — do **not** rebuild OCBI `call_graph` / `pr_impact` in MemIndex

---

## Execution handoff

**Plan complete:** `docs/superpowers/plans/2026-08-26-skillgrid-memindex.md`

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks
2. **Inline Execution** — execute tasks in this session via executing-plans with checkpoints

Which approach?
