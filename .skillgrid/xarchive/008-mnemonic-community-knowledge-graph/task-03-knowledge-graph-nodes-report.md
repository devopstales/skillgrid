## 03-knowledge-graph-nodes — Implementation Report

### Status: COMPLETE (03.1–03.9) + review findings addressed (commit b8a0d02)

---

### Commits

| SHA | Subject |
|-----|---------|
| `cb117c3` | feat(mnemonic): markdown doc-link extractor |
| `f5f10f2` | feat(mnemonic): config-ref extractor |
| `cfdf6f8` | feat(mnemonic): sql-schema extractor |
| `4bc0391` | feat(mnemonic): register knowledge-graph query tools |
| `aa98d1c` | feat(mnemonic): wire community/process/knowledge passes into Indexer.Run (03.8) |

---

### Per-Sub-Task Summary

#### 03.1 — Knowledge MCP/CLI tools (RED→GREEN)
- **Test**: `TestKnowledgeTools` in `tools_code_knowledge_test.go` — verifies 4 new tools (`code_docs`, `code_configs`, `code_sql_schema`, `code_sql_access`) register with distinct `code_*` names; 005 tools remain name/param-stable; bad/missing args rejected.
- **RED**: Test failed (tools not registered). **GREEN**: 4 tools registered in `server.go` (both `Start` + `NewServer`); CLI dispatch added in `code_intel.go` + `main.go`. Tool surface 67→71.
- **Files**: `tools_code_knowledge.go`, `server.go`, `server_test.go`, `code_intel.go`, `main.go`

#### 03.2 — Markdown doc-link extractor (RED→GREEN)
- **Test**: `TestDocLinks` in `knowledge_test.go` — `.md` files with `[text](./other.md)` and `[[wikilinks]]` produce `doc_nodes` + `references` edges, confidence-labeled.
- **RED**: Test failed (no extractor). **GREEN**: `doc.go` parses markdown links + wikilinks → `DocResult`/`DocLink`; `store.go` persists via `SaveDoc`. Wikilinks resolve to `<name>.md` by file convention.
- **Files**: `knowledge/doc.go`, `knowledge/store.go`

#### 03.3 — Config-ref extractor (RED→GREEN)
- **Test**: `TestConfigRefs` in `knowledge_test.go` — `.yaml`/`.toml`/`.json` produce `config_nodes` + `configures` edges; explicit refs = EXTRACTED, inferred = INFERRED.
- **RED**: Test failed. **GREEN**: `config.go` parses YAML/TOML/JSON → `ConfigResult`/`ConfigRef`; explicit CamelCase ref = EXTRACTED, convention = INFERRED, unresolvable = AMBIGUOUS (kept, not dropped).
- **Files**: `knowledge/config.go`

#### 03.4 — SQL DDL extractor (RED→GREEN)
- **Test**: `TestSqlSchema` in `knowledge_test.go` — `.sql` DDL produces `sql_schema_nodes` (tables + columns); DML references get `reads`/`writes` edges.
- **RED**: Test failed. **GREEN**: `sql.go` parses CREATE TABLE → table/column nodes; SELECT/INSERT/UPDATE/DELETE → reads/writes edges. Malformed DDL (unbalanced parens) skipped.
- **Files**: `knowledge/sql.go`

#### 03.5 — Unresolvable config ref = AMBIGUOUS (AFK)
- **Test**: `TestAmbiguousConfigRef` — unresolvable config ref produces an AMBIGUOUS edge (not dropped).
- **PASS**: Verified as part of `TestConfigRefs` + dedicated test.

#### 03.6 — Malformed doc: skip bad links, index rest (AFK)
- **Test**: `TestMalformedDoc` — malformed markdown link is skipped; valid links still indexed.
- **PASS**: Verified.

#### 03.7 — Malformed SQL: skip it, index rest (AFK)
- **Test**: `TestMalformedSQL` — malformed SQL statement (unbalanced parens) is skipped; valid DDL still indexed.
- **PASS**: Verified.

#### 03.8 — Indexer hook (AFK)
- **Test**: `TestIndexerHook` in `indexer_hook_test.go` — after one `idx.Run`, communities, processes + process_steps, labeled processes, doc_nodes + references, config_nodes + configures, sql_schema_nodes, and 005 symbols all exist.
- **Implementation**: In `Indexer.Run`, after the 005 tx commits:
  1. Close the store's `*sql.DB` (MaxOpenConns=1 holds the WAL write lock in modernc.org/sqlite even after commit).
  2. Open a fresh `*sql.DB` with `?_pragma=foreign_keys(1)` DSN (foreign_keys is per-connection, not persistent).
  3. Run `communityPass`, `processPass`, `knowledgePass` on the fresh DB.
  4. Assign the fresh DB back to `idx.store.DB` so callers see the updated store.
- **Process LLM**: `processLLMStub` — deterministic labeler returning `"flow: <name>"`. Set via `WithProcessLLM()`. The key is the pass RUNS at index time, not that it's a live LLM.
- **Advisory**: Each pass failure warns to stderr and continues; the 005 graph + FTS floor are already committed.

---

### Deadlock Root Cause (03.8)

The store uses `sql.Open("sqlite", path)` + `SetMaxOpenConns(1)`. After a long tx commits, the modernc.org/sqlite driver's single connection holds a driver-level statement that prevents the `database/sql` pool from serving a new query on the same `*sql.DB`. A second `*sql.DB` on the same file also deadlocks because the store's open connection holds the WAL write lock.

**Fix**: Close the store's DB after the 005 tx commits. Open a fresh `*sql.DB` (no `SetMaxOpenConns(1)`) with `?_pragma=foreign_keys(1)` DSN. The fresh DB has no lingering state and no WAL lock contention (the store's DB is closed). The `foreign_keys(1)` DSN parameter ensures cascade deletes work on the fresh connection (foreign_keys is per-connection in SQLite, not persistent in the file).

Without `foreign_keys=ON` on the fresh connection, the `ON DELETE CASCADE` on `edges.from_id` doesn't fire, leaving orphan edges after symbol pruning (exposed by `TestIndexOrphanPruneNoLingering`).

---

### Test Evidence

| Suite | Command | Result |
|-------|---------|--------|
| knowledge | `go test ./internal/mnemonic/knowledge/... -count=1` | PASS (1.08s) |
| codeindex | `go test ./internal/mnemonic/codeindex/... -count=1` | PASS (6.99s) |
| mcp | `go test ./internal/mnemonic/mcp/... -count=1` | PASS (39.5s) |
| cmd/skillgrid | `go test ./cmd/skillgrid/... -count=1` | PASS (21.9s) |
| community (baseline) | `go test ./internal/mnemonic/community/... -count=1` | PASS (1.08s) |
| process (baseline) | `go test ./internal/mnemonic/process/... -count=1` | PASS (1.99s) |
| route (baseline) | `go test ./internal/mnemonic/route/... -count=1` | PASS (2.81s) |
| graph (baseline) | `go test ./internal/mnemonic/graph/... -count=1` | PASS (4.01s) |

All 8 suites green. No regressions.

---

### 03.9 — `code_path` spans code→doc→config→table (COMPLETE)

**Status**: PASS. `code_path` now additively traverses the knowledge edge kinds.

#### What `code_path` now traverses

The 005 `code_path` tool resolves its `from`/`to` as 005 symbols and calls `graph.Path` (BFS over the `symbols`/`files` tables only). Knowledge nodes (`doc_nodes` / `config_nodes` / `sql_schema_nodes`) live in separate tables, so the baseline path could never reach them.

**Additive extension** (no 005 rewrite):
- **`internal/mnemonic/graph/span.go` (new)** — a unified-node-space BFS:
  - `spanNode` / `spanRef` — a node is a 005 symbol **or** a knowledge node (doc/config/table/column), identified by `(kind, id)` (the kind disambiguates id collisions across tables).
  - `resolveSpanNode` / `resolveSpanEndpoint` — resolve a name or an edge endpoint id into the unified space (symbol first, then knowledge node). Endpoint id ambiguity (a symbol id can numerically equal a knowledge-node id) is resolved by the **edge kind**: `reads`/`writes` probe `table` first, `configures` probes `config` first, `references` probes `doc` first, and 005 code edges probe `symbol` first (baseline unchanged).
  - `fetchSpanEdges` — loads edges touching a node (`WHERE from_id=? OR to_id=?`), drains the rows **before** resolving endpoints (a single-connection store cannot run two queries at once with a result set open — this was the deadlock), then resolves each endpoint into the unified space. Name-only edges that resolve to nothing are skipped (no fabricated hop, no un-terminable BFS).
  - `PathSpan(ctx, db, a Symbol, bName)` — BFS from a code symbol to a target name (symbol or knowledge node). Undirected (a `config->code` edge is walkable from `code` back to `config`).
  - `PathSpanFromName(ctx, db, aName, bName)` — same, but the source is also resolved by name, so a span can **start** from a doc/config/table (used for doc→code→table).
  - `PathResult` gained a `Reason` field (for not-found without a graph-stops frontier).
- **`internal/mnemonic/mcp/tools_code_graph.go`** — `mcpCodePath` (the `code_path` handler): when both endpoints resolve as symbols it uses `graph.Path` (byte-identical 005 baseline); when either is not a symbol it falls back to `graph.PathSpanFromName`. Additive — code-to-code paths unchanged.

#### TestCodePathSpan RED→GREEN

- **Test**: `internal/mnemonic/knowledge/codpath_span_test.go` — seeds a code symbol `loadUsers`, a config (`app.yaml` -configures-> code), a table (`users`, code -reads-> table), and a doc (`docs/a.md` -references-> code). Asserts:
  - `PathSpanFromName("docs/a.md", "users")` returns a **multi-hop** path (doc→code→table) traversing **≥2 knowledge edge kinds** (`references` + `reads`) and reaching the table node.
  - `PathSpan(code, "config/app.yaml")` reaches the config via a `configures` hop.
- **RED**: `go test ./internal/mnemonic/knowledge/... -run CodePathSpan -count=1` → `undefined: graph.PathSpan` / `undefined: graph.PathSpanFromName` (the span functions did not exist).
- **GREEN**: `--- PASS: TestCodePathSpan (0.02s)`.

> Fixture note: the 005 `edges` table shares one integer space for `from_id`/`to_id`, but the 015 knowledge tables autoincrement independently. A cold fixture gives every node id 1, so `WHERE from_id=1 OR to_id=1` over-matches. The test bumps each table's autoincrement with throwaway rows so the real nodes land on distinct ids (code=2, doc=3, config=4, table=5), mirroring a real store where symbol and knowledge-node ids do not all collide.

#### Full relevant suite + 005 baseline

```
$ go test ./internal/mnemonic/knowledge/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1
ok  github.com/.../internal/mnemonic/knowledge  1.002s
ok  github.com/.../internal/mnemonic/codeindex   5.451s
ok  github.com/.../internal/mnemonic/mcp        29.672s
ok  github.com/.../cmd/skillgrid                17.738s
```

005 code-to-code path baseline (graph `Path`, `Resolve`, `fetchEdges`, neighbor views, impact) — unchanged and green:
```
$ go test ./internal/mnemonic/community/... ./internal/mnemonic/process/... ./internal/mnemonic/route/... ./internal/mnemonic/graph/... ./internal/mnemonic/service/... -count=1
ok  community   0.946s
ok  process     1.664s
ok  route       2.900s
ok  graph       2.592s
ok  service    19.107s
```

The brief's run command `go test ./internal/mnemonic/mcp/... ./internal/mnemonic/knowledge/... -run CodePathSpan -count=1` → PASS (the span test lives in `knowledge`; the mcp package's `TestKnowledgeTools` locks the `code_path` tool surface and passes).

---

### Files Changed

**Created:**
- `skillgrid-cli/internal/mnemonic/knowledge/doc.go` — markdown link/wikilink parser
- `skillgrid-cli/internal/mnemonic/knowledge/config.go` — YAML/TOML/JSON config ref parser
- `skillgrid-cli/internal/mnemonic/knowledge/sql.go` — SQL DDL/DML parser
- `skillgrid-cli/internal/mnemonic/knowledge/store.go` — knowledge node persistence seam
- `skillgrid-cli/internal/mnemonic/knowledge/passes.go` — pass runner (RunPasses)
- `skillgrid-cli/internal/mnemonic/knowledge/knowledge_test.go` — knowledge extractor tests
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_knowledge.go` — 4 knowledge MCP tools
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_knowledge_test.go` — TestKnowledgeTools
- `skillgrid-cli/internal/mnemonic/codeindex/indexer_hook_test.go` — TestIndexerHook
- `skillgrid-cli/internal/mnemonic/store/migrations/015_knowledge_graph.sql` — doc/config/sql node tables
- `skillgrid-cli/internal/mnemonic/graph/span.go` — 03.9 unified-node-space BFS (PathSpan, PathSpanFromName)
- `skillgrid-cli/internal/mnemonic/knowledge/codpath_span_test.go` — TestCodePathSpan

**Modified:**
- `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` — 03.8 hook (passes + processLLMStub)
- `skillgrid-cli/internal/mnemonic/mcp/server.go` — register 4 knowledge tools
- `skillgrid-cli/internal/mnemonic/mcp/server_test.go` — expect 71 tools
- `skillgrid-cli/cmd/skillgrid/code_intel.go` — CLI dispatch for knowledge subcommands
- `skillgrid-cli/cmd/skillgrid/main.go` — CLI wiring
- `skillgrid-cli/internal/mnemonic/graph/graph.go` — `PathResult` gains `Reason` field (03.9)
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_graph.go` — `mcpCodePath` falls back to `PathSpanFromName` for knowledge endpoints (03.9)

---

### Review Findings — Step 03 (commit `b8a0d02`)

Addressed four step-03 review findings (1 IMPORTANT, 3 MINOR).

#### 1. (IMPORTANT) 03.8 "same transaction" claim was false

The `indexer.go` comment (and the `TestIndexerHook` / brief scenario wording) claimed the community + process + knowledge passes ran in the **same incremental tx** as 005. That is not what ships: the 005+route extraction commits in one tx, and the passes run **after** that commit, on a freshly reopened `*sql.DB` (`passDB`), because modernc.org/sqlite deadlocks the `MaxOpenConns(1)` store pool once the committed tx holds the WAL write lock. The passes are advisory (warn-and-continue, never load-bearing) and do **not** roll back the committed 005 extraction.

**Fix (honest framing, no re-architecture):** the close/reopen workaround behavior is unchanged (real modernc constraint). Only the comments now state what actually happens.

- `indexer.go` `Run` — comment before the close/reopen now reads:
  > Before: "The store's single-connection *sql.DB (MaxOpenConns=1) holds the WAL write lock even after the tx commits… Close it so a fresh *sql.DB can acquire the lock for the passes." (implied shared context)
  > After: "The 005 + route extraction (above) ran in one transaction, now committed. The community + process + knowledge passes run AFTER that commit, NOT in the same tx… The passes are advisory, not transactional with 005: they do not roll back the committed 005 extraction…"
- `indexer.go` `communityPass` — comment now reads:
  > Before: "on the pass DB (opened **before the main tx**, so its sub-txs do not deadlock the store's single-connection pool). The community rows land in the same WAL database as the 005 extraction; the main tx's Commit finalizes the incremental index."
  > After: "on the pass DB (a fresh *sql.DB opened **after the 005 tx commits** — the store's single-connection pool deadlocks once the committed tx holds the write lock, so the passes cannot share that tx). The community rows land in the same WAL database as the 005 extraction, but are committed independently: the pass is advisory and does not roll back the 005 extraction."
- `knowledge/store.go` `Store` doc + `NewStore` — now describe the store as backed by the **pass DB** (fresh *sql.DB after the 005 tx commits), advisory, not the 005 tx.
- `TestIndexerHook` comment — now says the 005+route tx commits first, then the advisory passes run on the reopened DB (same index run, not same tx), and the end state proves the three passes ran.

`tasks.md` / acceptance feature were NOT rewritten (controller's job) — only code + test comments.

#### 2. (MINOR) `SaveConfig` cross-package name collision

`SaveConfig` resolved config refs by `name` with `matches == 1`; a ref matching **>1** symbol (a cross-package `loadUsers` collision) was surfaced **AMBIGUOUS** (to_id null) rather than EXTRACTED, and the code now documents that intent explicitly (matches > 1 → AMBIGUOUS, to_id null, not silently bound to the lowest id).

- `knowledge/store.go` `SaveConfig` — comment on the AMBIGUOUS branch now states matches > 1 is a cross-package collision surfaced AMBIGUOUS.
- **Test**: `TestAmbiguousConfigRefMultipleSymbols` in `knowledge_test.go` — seeds two same-named `loadUsers` symbols (distinct uid, different packages) and asserts a config ref to `loadUsers` is stored **AMBIGUOUS** (to_id null), **0** EXTRACTED edges to a resolved id; contrast asserts a ref to a unique name stays **EXTRACTED** (bound by id). New helper `seedSymbolUID` (explicit uid) added since `seedSymbol` derives uid from name+kind (would collide).

#### 3. (MINOR) `isSQL` false positives

`isSQL` scanned **every** file (incl. `.go`/`.md`) for `SELECT`/`INSERT`/`UPDATE`/`DELETE FROM` before deciding to parse, so Go code or prose containing those words produced spurious table refs.

- `knowledge/passes.go` `isSQL` — now returns true **only** for `.sql` / `.SQL` files (the DDL/DML parse is restricted to `.sql`; the `.sql` behavior is kept). Removed the now-dead `containsWord`/`isIdentByte` helpers and the `strings` import.
- **Test**: `TestSQLOnlyForSQLFiles` in `knowledge_test.go` — unit: `isSQL` is true only for `.sql`/`.SQL` regardless of contents; integration: a `.go` file containing `SELECT` in a string/comment yields **0** `sql_schema_nodes` and **0** reads edges (via `RunPasses`), while a `.sql` file with the same DDL still parses.

#### 4. (MINOR) `selectIsRead` dead prefix check

`knowledge/sql.go` `selectIsRead` had `strings.HasPrefix(upper, "SELECT") || strings.Contains(upper, "SELECT")` — the prefix check was dead (Contains subsumes it). Removed the `HasPrefix`, kept `Contains`. No behavior change.

---

### Review-findings test run

```
$ go test ./internal/mnemonic/knowledge/... ./internal/mnemonic/codeindex/... \
    ./internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/knowledge   1.141s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex   5.568s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp         31.991s
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid                 19.710s
```

(graph suite also re-verified: `ok …/internal/mnemonic/graph 1.054s`.)

**Commit**: `b8a0d02` fix(mnemonic): honest 03.8 tx framing + config/SQL extractor correctness

**Files changed (b8a0d02)**:
- `skillgrid-cli/internal/mnemonic/codeindex/indexer.go` — honest 03.8 tx comments (Run + communityPass)
- `skillgrid-cli/internal/mnemonic/codeindex/indexer_hook_test.go` — TestIndexerHook comment
- `skillgrid-cli/internal/mnemonic/knowledge/store.go` — Store/NewStore doc + SaveConfig AMBIGUOUS-on-multiple comment
- `skillgrid-cli/internal/mnemonic/knowledge/passes.go` — isSQL .sql-only (+ dead helpers removed)
- `skillgrid-cli/internal/mnemonic/knowledge/sql.go` — selectIsRead dead HasPrefix removed
- `skillgrid-cli/internal/mnemonic/knowledge/knowledge_test.go` — TestAmbiguousConfigRefMultipleSymbols, TestSQLOnlyForSQLFiles, seedSymbolUID helper
