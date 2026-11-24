# Mnemonic Phase 2: Local-First Code Indexing & Static Intelligence — Design Specification

- **Date**: 2026-11-24
- **Status**: Draft — Pending User Spec Review
- **Repository**: `skillgrid-mnamonic` (`mnemonic` binary)
- **Prerequisite**: `docs/superpowers/specs/2026-11-24-mnemonic-sqlite-memory-design.md` (Phase 1 Platform & Memory Subsystem)
- **Lineage**: Synthesized from `skillgrid-v2/mnemonic`, [CodeGraph](https://github.com/colbymchenry/codegraph), [GitNexus](https://github.com/abhigyanpatwari/GitNexus), and [Graphify](https://github.com/Graphify-Labs/graphify)

---

## 1. Purpose, Goals & Lineage

Phase 2 introduces the **`mnemonic code` subsystem** (`internal/code`), attaching cleanly onto Phase 1's `internal/platform` foundation (`CGO_ENABLED=0` pure Go, `modernc.org/sqlite` + `vec0`, shared `mnemonic daemon` with `--standalone` fallback, `hugot` 384-dim `bge-small-en-v1.5` ONNX CPU embeddings, and `platform/registry.Subsystem`).

It stores the complete code graph, lexical/segment/vector indexes, architectural topology, framework contracts, and intra-procedural CFG/PDG/taint summaries directly inside the project's existing **`~/.mnemonic/<repo>.db`** (`subsystem = "code"`), enabling zero-overhead SQL joins between durable `memory` observations and live code symbols.

### 1.1 Architectural Synthesis

- **From [CodeGraph](https://github.com/colbymchenry/codegraph)**:
  - `code_name_segments WITHOUT ROWID` camelCase/PascalCase/snake_case vocabulary splitting so natural-language prose queries hit multi-word identifiers missed by standard FTS5 tokenizers.
  - `code_unresolved_refs` (`pending` vs `failed` + partial indexes on `name_tail` and `module_spec`) with incoming cross-file edge resurrection before file updates/deletes, enabling incremental cross-file healing without full rebuilds.
  - `WalCheckpointValve` + cooperative writer batching for high-throughput SQLite WAL ingestion.
  - Personalized PageRank / Random Walk with Restart (`RWR`, $\alpha = 0.25$, 25 iterations) + proportional `<= 24 KiB` whole-symbol contiguous cluster budgeting + `validateAnswerFiles` live staleness verification.
- **From [GitNexus](https://github.com/abhigyanpatwari/GitNexus) & [Graphify](https://github.com/Graphify-Labs/graphify)**:
  - Confidence-tiered edges (`LSP_RESOLVED`, `EXTRACTED`, `SYNTHESIZED`, `INFERRED`, `AMBIGUOUS`) with weakest-hop path confidence reporting.
  - Description-forward symbol embedding projection (`"<kind>: <qualified_name>\nSig: <signature>\nDoc: <docstring>\nPath: <tail>"`) paired with Reciprocal Rank Fusion (`RRF`, $K = 60$).
  - Deterministic Leiden community clustering (`seed = 0xc0de`) with hub-symbol labeling, cross-community `god_nodes`, Tarjan `cycles`, and ranked entry-point DFS execution `flows`.
  - Framework contract synthesis (HTTP/gRPC routes, SQL/ORM tables, pub-sub topics, MCP/CLI tools, Markdown/config sections), intra-procedural `BasicBlock` CFG/PDG/taint summaries, and 2-step `plan_token`-guarded `code_rename`.
- **From `skillgrid-v2/mnemonic` (Retained Strengths & Eliminated Defects)**:
  - Retains pure-Go `gotreesitter` 3-tier extraction (`Rich` -> `Map` -> `Regex`) with a panic-recovering grammar circuit breaker and shared CLI/MCP/Daemon parity.
  - Structurally eliminates every defect discovered in `skillgrid-v2`:

| `skillgrid-v2` Defect | Phase 2 Structural Remediation |
|---|---|
| `edges.from_id = fileID` FK corruption on file-scope refs | Every indexed file creates a real `kind = 'file'` root row in `code_symbols`; enclosing-span stack attributes every ref to its innermost symbol or the file's root `code_symbols.id`. |
| Local 1-based slice index written as `edges.to_id` | Workers emit string `uid`s and `raw_target`s only; numeric `code_symbols.id` FKs bind strictly after `INSERT ... RETURNING id` / `uid` lookup inside SQLite. |
| Pathless `symbolUID` collisions across files | `uid = sha256(repo_rel_path || "\x00" || kind || "\x00" || qualified_name || "\x00" || sig_hash)`. |
| `semanticLeg` brute-force scan bypassing `vec0` + NULL language filter | Unified `search` queries `code_symbols_vec` KNN (`vec0`) joined directly to `code_symbols.language` (falling back to SIMD-friendly float32 scan only during model swap). |
| Missing `.gitignore` filtering & non-recursive `fsnotify` | `scan` parses nested `.gitignore`, `.git/info/exclude`, and `.mnemonicignore`; recursive `Watcher` subscribes all non-ignored directories and newly created subdirectories dynamically. |
| CWD vs repo-root bugs in `read` and structural `grep` | Every filesystem read confines paths via `filepath.EvalSymlinks` strictly against `code_index_meta.active_root`; process `os.Getwd()` is never called in `internal/code`. |
| `mnemonic code watch` exiting immediately | CLI `watch` blocks on `ctx.Done()` (`SIGINT`/`SIGTERM`) and streams structured incremental index events. |
| 39 bloated MCP tools (incl. 7 stubs & unimplemented LSP) | Progressive disclosure (**7 Eager Core** + **6 Deferred Extended** + **3 Admin**), registered strictly as each sub-phase (`2A`–`2D`) ships. Zero stubs at every commit. |

### 1.2 Non-Goals

1. Cloud or PostgreSQL storage backends, multi-tenant remote indexing, or Git LFS binary blob parsing.
2. Whole-program inter-procedural symbolic execution / SMT solver verification (Phase 2D provides deterministic intra-procedural CFG/CDG/reaching-defs + bottom-up bitmask taint summaries across the call graph).
3. Automatic unprompted installation of third-party language server binaries (`--lsp` / `code_lsp_sync` uses binaries already present on `PATH` or configured in `config.yaml`, degrading cleanly when absent).

---

## 2. System Architecture, Package Layout & Phased Delivery

### 2.1 Package Layout (`< 600` LOC per file invariant)

```text
internal/
├── platform/
│   └── sqlitepool/               # Extended with PriorityWrite fairness + WalCheckpointValve
└── code/                         # Phase 2 `code` subsystem (`Name() = "code"`)
    ├── api/                      # `Service` interface, domain DTOs, capability flags, compliance suite
    ├── store/                    # SQLite repository (`<repo>.db` only) + `migrations/001..004.sql`
    ├── scan/                     # Gitignore-aware walker, generated/test classifier, recursive watcher
    ├── extract/                  # 3-tier `gotreesitter` worker pool, span stack, chunker, stamp
    ├── resolve/                  # Resurrection engine + 6 Tier-1 workspace manifest/import resolvers
    ├── csr/                      # Generation-fenced in-RAM CSR graph + RWR, Leiden, SCC, Dijkstra, flows
    ├── contracts/                # Phase 2C route, RPC, ORM/SQL, pub-sub, and doc/config synthesizers
    ├── analysis/                 # Phase 2D BasicBlock CFG, Lengauer-Tarjan CDG, reaching defs, taint
    ├── lsp/                      # Phase 2D JSON-RPC 2.0 stdio client pool + 5m idle subprocess reaper
    ├── search/                   # 3-way RRF + CSR RWR + <=24 KiB cluster allocator + staleness verifier
    ├── rename/                   # 2-step `plan_token` refactor planner & atomic file applier
    ├── engine/                   # `Local` orchestrator (cooperative DAG, memory<->code linker)
    ├── client/                   # `DaemonClient` over `daemon.sock` / bearer HTTP
    ├── resolver/                 # `ResolveBackend` (daemon auto-spawn or `engine.Local` fallback)
    ├── mcp/                      # `mnemonic code mcp` stdio server (`mcp-go`, progressive disclosure)
    ├── http/                     # `/v1/code/*` JSON handlers mounted on daemon mux
    ├── cli/                      # `mnemonic code ...` subcommands
    └── tui/                      # `[2: Code]` Bubbletea tab registered into `platform/tuishell`
```

### 2.2 Platform Seam Extensions (`platform/sqlitepool` & `platform/migrate`)

Because `subsystem = "code"` shares `~/.mnemonic/<repo>.db` with `subsystem = "memory"`, two minimal additive safeguards are built into `internal/platform`:

1. **Target Scope Filter on `Subsystem`**: `Subsystem` adds `AppliesToGlobalDB() bool` (`true` for `memory`, `false` for `code` so `global.db` never creates empty `code_*` tables).
2. **Two-Lane Priority `writeQueue` & `WalCheckpointValve` in `sqlitepool`**:
   - Interactive writes (`PriorityHigh`, used by `mem_save`, `mem_judge`, `mem_session_*`, `code_rename`) drain ahead of background/bulk writes (`PriorityBatch`, used by `code index` 50-file batches and vector backfill). Between every batch transaction, the indexer yields to any queued `PriorityHigh` operations so `p99 mem_save` latency stays `< 25 ms` during a full repository index.
   - During bulk `code index` runs, `WalCheckpointValve` sets `PRAGMA wal_autocheckpoint = 0` on the writer so batch commits become sequential WAL appends. It tracks WAL byte growth since the last full backfill (`soft = 256 MiB`, `hard = 512 MiB`, `cap = 1024 MiB`), running off-thread `PRAGMA wal_checkpoint(PASSIVE)` at `soft`, briefly pausing batch submission for `RESTART` at `hard`, and restoring `wal_autocheckpoint = 1000` + `TRUNCATE` upon run completion.

### 2.3 Four Sequential Vertical Slices (`2A -> 2B -> 2C -> 2D`)

The schema and `api.Service` boundaries are locked upfront in this specification; implementation proceeds across **4 shippable sub-plans**, where every slice ships 100% tested features and registers **only** implemented MCP/CLI surfaces (zero stubs):

| Slice | Name | Core Deliverables | MCP Tools Unlocked |
|---|---|---|---|
| **Phase 2A** | Core Graph, Incremental Sync & Hybrid RWR Retrieval | `001_code_core.sql`, gitignore scanner + recursive watcher, 3-tier `gotreesitter` pool (30+ langs), 6 Tier-1 manifest/cross-file resolvers + `unresolved_refs` resurrection, FTS5 + `code_name_segments` + async ONNX `vec0`, CSR PageRank (`RWR`) + `<=24 KiB` `code_explore`, `code_read`, `code_search`, `code_impact`, `code_affected`, `code_neighbors`, 2-step `code_rename`, `observation_code_links`, CLI/Daemon/TUI | Eager: `code_explore`, `code_read`, `code_impact`, `code_affected`, `code_status`, `code_index`; Deferred: `code_search`, `code_neighbors`, `code_rename`; Admin: `code_watch`, `code_backfill_embeddings` |
| **Phase 2B** | Graph Topology, Communities & Execution Flows | `002_code_topology.sql`, deterministic Leiden community detection + hub PageRank labeling, cross-community `god_nodes`, Tarjan `cycles`, weakest-hop Dijkstra `shortest_path`, ranked entry-point DFS `flows` (3–10 steps), flow blast radius in `code_affected`, TUI Topology view | Eager: **`code_topology`** (`overview`, `communities`, `god_nodes`, `flows`, `cycles`, `path`) |
| **Phase 2C** | Framework Synthesis & Cross-Boundary Contracts | `003_code_contracts.sql`, AST contract synthesizers across all 6 Tier-1 ecosystems (HTTP/gRPC routes, SQL/ORM tables, pub-sub topics, MCP/CLI tools) + Markdown heading sections & YAML/TOML/JSON config keys, contract edges in CSR RWR & `code_impact` | Deferred: **`code_contracts`** (`routes`, `queries`, `events`, `tools`, `docs`, `configs`) |
| **Phase 2D** | Intra-Procedural CFG/PDG/Taint & Optional LSP Pass | `004_code_analysis.sql`, `BasicBlock` CFG, Lengauer-Tarjan post-dominator CDG, reaching definitions, bottom-up bitmask `code_taint_summaries` + source->sink taint traces, JSON-RPC stdio LSP pool (`gopls`, `typescript-language-server`, `pyright-langserver`, `rust-analyzer`, `jdtls`, `csharp-ls`) upgrading edges to `LSP_RESOLVED` | Deferred: **`code_pdg`**, **`code_taint`**; Admin: **`code_lsp_sync`** |

---

## 3. SQLite Schema Specification (`subsystem = "code"`, `<repo>.db`)

### 3.1 Migration `001_code_core.sql` (Phase 2A)

```sql
-- 1. Singleton Index State & Generation Counters
CREATE TABLE code_index_meta (
    id                  INTEGER PRIMARY KEY CHECK (id = 1),
    active_root         TEXT NOT NULL DEFAULT '',  -- Canonical realpath of last synced checkout/worktree
    head_commit         TEXT NOT NULL DEFAULT '',  -- Git HEAD SHA at last index
    extractor_stamp     TEXT NOT NULL DEFAULT '',  -- Hex FNV-1a/SHA-256 of grammar + chunker + resolver rules
    workspace_hash      TEXT NOT NULL DEFAULT '',  -- Hash of manifest files (go.mod, tsconfig.json, etc.)
    graph_generation    INTEGER NOT NULL DEFAULT 0,-- Incremented on any symbol/edge mutation (invalidates CSR)
    topology_generation INTEGER NOT NULL DEFAULT 0,-- Last graph_generation for which 2B topology ran
    capabilities_json   TEXT NOT NULL DEFAULT '{}',-- {"core":true,"vectors":true,"topology":false,...}
    files_count         INTEGER NOT NULL DEFAULT 0,
    symbols_count       INTEGER NOT NULL DEFAULT 0,
    edges_count         INTEGER NOT NULL DEFAULT 0,
    unresolved_count    INTEGER NOT NULL DEFAULT 0,
    last_indexed_at     TEXT,
    updated_at          TEXT NOT NULL
);

-- 2. Indexed Repository Files
CREATE TABLE code_files (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    path          TEXT NOT NULL UNIQUE,            -- Repo-relative slash-normalized path (never leading '/')
    language      TEXT NOT NULL,
    tier          TEXT NOT NULL CHECK (tier IN ('rich', 'map', 'regex', 'doc')),
    size          INTEGER NOT NULL,
    mtime_ns      INTEGER NOT NULL,
    content_hash  TEXT NOT NULL,                   -- SHA-256 of raw file bytes
    ast_hash      TEXT NOT NULL,                   -- SHA-256 of normalized exported symbol signatures
    is_generated  INTEGER NOT NULL DEFAULT 0 CHECK (is_generated IN (0, 1)),
    is_test       INTEGER NOT NULL DEFAULT 0 CHECK (is_test IN (0, 1)),
    parse_errors  INTEGER NOT NULL DEFAULT 0,
    indexed_at    TEXT NOT NULL
);
CREATE INDEX idx_code_files_lang ON code_files(language);
CREATE INDEX idx_code_files_gen  ON code_files(is_generated) WHERE is_generated = 1;

-- 3. Code Symbols (includes synthetic kind='file' root node per file)
CREATE TABLE code_symbols (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    uid            TEXT NOT NULL UNIQUE,           -- sha256(path || '\0' || kind || '\0' || qualified_name || '\0' || sig_hash)
    file_id        INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    parent_id      INTEGER REFERENCES code_symbols(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    qualified_name TEXT NOT NULL,
    kind           TEXT NOT NULL CHECK (kind IN (
                       'file', 'module', 'namespace', 'package', 'class', 'struct',
                       'interface', 'trait', 'enum', 'enum_member', 'type_alias',
                       'function', 'method', 'constructor', 'property', 'field',
                       'constant', 'variable', 'macro', 'section', 'config_key'
                   )),
    signature      TEXT NOT NULL DEFAULT '',
    docstring      TEXT NOT NULL DEFAULT '',
    return_type    TEXT NOT NULL DEFAULT '',
    visibility     TEXT NOT NULL DEFAULT 'internal' CHECK (visibility IN ('public', 'protected', 'internal', 'private')),
    is_exported    INTEGER NOT NULL DEFAULT 0 CHECK (is_exported IN (0, 1)),
    start_line     INTEGER NOT NULL,
    start_col      INTEGER NOT NULL DEFAULT 0,
    end_line       INTEGER NOT NULL,
    end_col        INTEGER NOT NULL DEFAULT 0,
    body_hash      TEXT NOT NULL,
    community_id   INTEGER,                        -- Populated in Phase 2B
    pagerank       REAL NOT NULL DEFAULT 0.0
);
CREATE INDEX idx_code_symbols_file       ON code_symbols(file_id, start_line);
CREATE INDEX idx_code_symbols_name       ON code_symbols(name);
CREATE INDEX idx_code_symbols_qname      ON code_symbols(qualified_name);
CREATE INDEX idx_code_symbols_exported   ON code_symbols(name, file_id) WHERE is_exported = 1;
CREATE INDEX idx_code_symbols_community  ON code_symbols(community_id) WHERE community_id IS NOT NULL;

-- 4. AST-Aligned Retrieval Chunks
CREATE TABLE code_chunks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id      INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    symbol_id    INTEGER REFERENCES code_symbols(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('ast', 'skeleton', 'window', 'doc_section')),
    start_line   INTEGER NOT NULL,
    end_line     INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    text         TEXT NOT NULL
);
CREATE INDEX idx_code_chunks_file ON code_chunks(file_id, start_line);

-- 5. Resolved Directed Code Graph Edges
CREATE TABLE code_edges (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    from_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    to_symbol_id   INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    file_id        INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    kind           TEXT NOT NULL CHECK (kind IN (
                       'contains', 'imports', 're_exports', 'calls', 'references',
                       'extends', 'implements', 'overrides', 'type_of', 'returns',
                       'instantiates', 'decorates'
                   )),
    confidence     TEXT NOT NULL DEFAULT 'EXTRACTED' CHECK (confidence IN (
                       'LSP_RESOLVED', 'EXTRACTED', 'SYNTHESIZED', 'INFERRED', 'AMBIGUOUS'
                   )),
    line           INTEGER,
    col            INTEGER
);
CREATE UNIQUE INDEX idx_code_edges_dedup ON code_edges(
    from_symbol_id, to_symbol_id, kind, IFNULL(line, -1), IFNULL(col, -1)
);
CREATE INDEX idx_code_edges_to   ON code_edges(to_symbol_id, kind);
CREATE INDEX idx_code_edges_file ON code_edges(file_id);

-- 6. Unresolved Cross-File References (Incremental Healing Queue)
CREATE TABLE code_unresolved_refs (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id        INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    from_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    raw_target     TEXT NOT NULL,                  -- Full qualifier expression e.g. 'pkg.Client.Do'
    name_tail      TEXT NOT NULL,                  -- Terminal identifier e.g. 'Do'
    module_spec    TEXT NOT NULL DEFAULT '',       -- Import specifier e.g. './auth' or 'github.com/x/y'
    receiver_hint  TEXT NOT NULL DEFAULT '',       -- Inferred receiver/type token when available
    kind           TEXT NOT NULL,
    line           INTEGER NOT NULL,
    col            INTEGER NOT NULL DEFAULT 0,
    status         TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'failed'))
);
CREATE INDEX idx_unresolved_pending       ON code_unresolved_refs(status, file_id) WHERE status = 'pending';
CREATE INDEX idx_unresolved_failed_tail   ON code_unresolved_refs(name_tail) WHERE status = 'failed';
CREATE INDEX idx_unresolved_failed_module ON code_unresolved_refs(module_spec) WHERE status = 'failed' AND module_spec != '';

-- 7. Identifier Segment Vocabulary (camelCase / PascalCase / snake_case -> prose matching)
CREATE TABLE code_name_segments (
    segment   TEXT NOT NULL,
    symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    PRIMARY KEY (segment, symbol_id)
) WITHOUT ROWID;
CREATE INDEX idx_code_segments_sym ON code_name_segments(symbol_id);

-- 8. FTS5 Lexical Tables (Symbol Metadata + Trigram Source Chunks)
CREATE VIRTUAL TABLE code_symbols_fts USING fts5(
    name,
    qualified_name,
    signature,
    docstring,
    tokenize = 'unicode61 remove_diacritics 1',
    content = 'code_symbols',
    content_rowid = 'id'
);

CREATE TRIGGER code_sym_fts_ai AFTER INSERT ON code_symbols BEGIN
    INSERT INTO code_symbols_fts(rowid, name, qualified_name, signature, docstring)
    VALUES (new.id, new.name, new.qualified_name, new.signature, new.docstring);
END;
CREATE TRIGGER code_sym_fts_ad AFTER DELETE ON code_symbols BEGIN
    INSERT INTO code_symbols_fts(code_symbols_fts, rowid, name, qualified_name, signature, docstring)
    VALUES ('delete', old.id, old.name, old.qualified_name, old.signature, old.docstring);
END;
CREATE TRIGGER code_sym_fts_au AFTER UPDATE OF name, qualified_name, signature, docstring ON code_symbols BEGIN
    INSERT INTO code_symbols_fts(code_symbols_fts, rowid, name, qualified_name, signature, docstring)
    VALUES ('delete', old.id, old.name, old.qualified_name, old.signature, old.docstring);
    INSERT INTO code_symbols_fts(rowid, name, qualified_name, signature, docstring)
    VALUES (new.id, new.name, new.qualified_name, new.signature, new.docstring);
END;

CREATE VIRTUAL TABLE code_chunks_fts USING fts5(
    text,
    tokenize = 'trigram',
    content = 'code_chunks',
    content_rowid = 'id'
);

CREATE TRIGGER code_chunks_fts_ai AFTER INSERT ON code_chunks BEGIN
    INSERT INTO code_chunks_fts(rowid, text) VALUES (new.id, new.text);
END;
CREATE TRIGGER code_chunks_fts_ad AFTER DELETE ON code_chunks BEGIN
    INSERT INTO code_chunks_fts(code_chunks_fts, rowid, text) VALUES ('delete', old.id, old.text);
END;
CREATE TRIGGER code_chunks_fts_au AFTER UPDATE OF text ON code_chunks BEGIN
    INSERT INTO code_chunks_fts(code_chunks_fts, rowid, text) VALUES ('delete', old.id, old.text);
    INSERT INTO code_chunks_fts(rowid, text) VALUES (new.id, new.text);
END;

-- 9. Description-Forward Symbol Embeddings (shares Phase 1 `embedding_meta`)
CREATE TABLE code_symbol_embeddings (
    symbol_id    INTEGER PRIMARY KEY REFERENCES code_symbols(id) ON DELETE CASCADE,
    content_hash TEXT NOT NULL,                    -- SHA-256 of description-forward projection text
    model        TEXT NOT NULL,
    dimension    INTEGER NOT NULL,
    vector       BLOB NOT NULL,                    -- L2-normalized float32[dimension]
    created_at   TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS code_symbols_vec USING vec0(
    symbol_id INTEGER PRIMARY KEY,
    embedding float[384]
);
-- Lifecycle rule (matches Phase 1 `observations_vec`): `vec0` updates execute
-- DELETE + INSERT inside the writer TXN; when `embedding_meta.dimension` changes,
-- `code_symbols_vec` is dropped and recreated at `float[<new_dim>]` and stale rows
-- in `code_symbol_embeddings` are re-embedded asynchronously by the backfill worker.

-- 10. Memory <-> Code Cross-Links (joins Phase 1 `observations` with live code)
CREATE TABLE observation_code_links (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_uid TEXT NOT NULL REFERENCES observations(uid) ON DELETE CASCADE,
    file_path       TEXT NOT NULL,
    symbol_uid      TEXT,
    link_kind       TEXT NOT NULL DEFAULT 'mentions' CHECK (link_kind IN ('mentions', 'governs', 'bugfix_in')),
    created_at      TEXT NOT NULL
);
CREATE UNIQUE INDEX idx_obs_code_links_dedup ON observation_code_links(
    observation_uid, file_path, IFNULL(symbol_uid, '')
);
CREATE INDEX idx_obs_code_links_file ON observation_code_links(file_path);
CREATE INDEX idx_obs_code_links_sym  ON observation_code_links(symbol_uid) WHERE symbol_uid IS NOT NULL;
```

### 3.2 Migration `002_code_topology.sql` (Phase 2B)

```sql
CREATE TABLE code_communities (
    id            INTEGER PRIMARY KEY,
    label         TEXT NOT NULL,                   -- Hub-derived label e.g. 'sqlitepool / Pool.Checkout'
    hub_symbol_id INTEGER REFERENCES code_symbols(id) ON DELETE SET NULL,
    size          INTEGER NOT NULL,
    cohesion      REAL NOT NULL,                   -- Internal edges / total incident edges
    god_score_max REAL NOT NULL DEFAULT 0.0,
    updated_at    TEXT NOT NULL
);

CREATE TABLE code_flows (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    uid                TEXT NOT NULL UNIQUE,       -- Deterministic hash of step symbol UIDs
    name               TEXT NOT NULL,              -- e.g. 'HandleExplore -> HybridSearch -> RunRWR'
    entry_symbol_id    INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    terminal_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    step_count          INTEGER NOT NULL,
    min_confidence      TEXT NOT NULL CHECK (min_confidence IN (
                            'LSP_RESOLVED', 'EXTRACTED', 'SYNTHESIZED', 'INFERRED', 'AMBIGUOUS'
                        )),                         -- Weakest hop confidence along the trace
    communities_spanned INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX idx_code_flows_entry ON code_flows(entry_symbol_id);

CREATE TABLE code_flow_steps (
    flow_id   INTEGER NOT NULL REFERENCES code_flows(id) ON DELETE CASCADE,
    step_idx  INTEGER NOT NULL,
    symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    edge_kind TEXT NOT NULL,
    PRIMARY KEY (flow_id, step_idx)
);
CREATE INDEX idx_code_flow_steps_sym ON code_flow_steps(symbol_id);
```

### 3.3 Migration `003_code_contracts.sql` (Phase 2C)

```sql
CREATE TABLE code_contracts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    kind          TEXT NOT NULL CHECK (kind IN (
                      'http_route', 'grpc_method', 'sql_table', 'orm_model',
                      'event_topic', 'mcp_tool', 'cli_command', 'doc_section', 'config_key'
                  )),
    canonical_key TEXT NOT NULL,                   -- e.g. 'POST /v1/memory/observations' or 'table:observations'
    method        TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    UNIQUE (kind, canonical_key)
);

CREATE TABLE code_contract_edges (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id   INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    contract_id INTEGER NOT NULL REFERENCES code_contracts(id) ON DELETE CASCADE,
    file_id     INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN (
                    'handles', 'calls_client', 'reads_table', 'writes_table',
                    'publishes', 'subscribes', 'registers_tool', 'documents', 'reads_config'
                )),
    confidence  TEXT NOT NULL DEFAULT 'SYNTHESIZED' CHECK (confidence IN (
                    'LSP_RESOLVED', 'EXTRACTED', 'SYNTHESIZED', 'INFERRED', 'AMBIGUOUS'
                )),
    line        INTEGER NOT NULL DEFAULT 0,
    UNIQUE (symbol_id, contract_id, role, line)
);
CREATE INDEX idx_contract_edges_contract ON code_contract_edges(contract_id, role);
CREATE INDEX idx_contract_edges_symbol   ON code_contract_edges(symbol_id);
```

### 3.4 Migration `004_code_analysis.sql` (Phase 2D)

```sql
CREATE TABLE code_cfg_blocks (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id  INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    block_idx  INTEGER NOT NULL,                   -- 0 = entry, N-1 = exit
    kind       TEXT NOT NULL CHECK (kind IN ('entry', 'body', 'branch', 'loop_head', 'return', 'throw', 'exit')),
    start_line INTEGER NOT NULL,
    end_line   INTEGER NOT NULL,
    defs_json  TEXT NOT NULL DEFAULT '[]',         -- Local variable names assigned in block
    uses_json  TEXT NOT NULL DEFAULT '[]',         -- Local variable names read in block
    UNIQUE (symbol_id, block_idx)
);

CREATE TABLE code_pdg_edges (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id  INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    from_block INTEGER NOT NULL,
    to_block   INTEGER NOT NULL,
    kind       TEXT NOT NULL CHECK (kind IN ('cfg_next', 'cfg_true', 'cfg_false', 'cdg', 'post_dom', 'reaching_def', 'taint_local', 'sanitized')),
    var_name   TEXT NOT NULL DEFAULT '',
    line       INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_pdg_edges_symbol ON code_pdg_edges(symbol_id, kind);

CREATE TABLE code_taint_summaries (
    symbol_id            INTEGER PRIMARY KEY REFERENCES code_symbols(id) ON DELETE CASCADE,
    param_count          INTEGER NOT NULL DEFAULT 0,
    param_to_return_mask INTEGER NOT NULL DEFAULT 0,-- Bit i = 1 iff param i flows unsanitized to return
    sources_json         TEXT NOT NULL DEFAULT '[]',-- [{kind:'http_input'|'rpc_arg'|'cli_arg'|'env_var'|'file_read', block_idx, line, expr}]
    sinks_json           TEXT NOT NULL DEFAULT '[]',-- [{kind:'sql_exec'|'cmd_exec'|'fs_path'|'html_out'|'ssrf_url', param_mask, sanitized, line, expr}]
    sanitizers_json      TEXT NOT NULL DEFAULT '[]',
    updated_at           TEXT NOT NULL
);
```

---

## 4. Cooperative Ingestion Pipeline, Resolution & Deep Analysis

### 4.1 File Discovery, Classification & Worktree Switching (`internal/code/scan`)

1. **Ignore Hierarchy**: Evaluates `.git/info/exclude`, root + nested directory `.gitignore` files, `.mnemonicignore`, and hardcoded directory exclusions (`.git`, `node_modules`, `vendor`, `dist`, `build`, `target`, `.next`, `.venv`, `__pycache__`). Rejects symlinks escaping `active_root`, files `> 1 MiB`, and files containing NUL bytes in the first `8 KiB`.
2. **Generated & Test Tagging**:
   - `is_generated = 1`: Matches path patterns (`*.pb.go`, `*_gen.go`, `*.generated.ts`, `*.min.js`, `dist/**`) or first-10-line header banners (`Code generated by`, `DO NOT EDIT`, `@generated`).
   - `is_test = 1`: Matches `*_test.go`, `test_*.py`, `*_test.py`, `*.test.{ts,tsx,js,jsx}`, `*.spec.{ts,tsx,js,jsx}`, `*Test.java`, `*Tests.cs`, or `tests/**` / `__tests__/**`.
3. **Incremental Delta & Worktree Reconcile**:
   - Fast-skips any file where `path`, `size`, and `mtime_ns` match `code_files` (unless `extractor_stamp` changed or `--full` is set). When `mtime_ns` changed, computes `SHA-256`; if `content_hash` is identical, updates `mtime_ns` only.
   - If the caller's git toplevel (`active_root`) differs from `code_index_meta.active_root` (e.g., switching git worktrees of the same repo), `scan` sets the new `active_root` and runs the exact same hash-based delta sync — rewriting only the files that actually differ between the two worktrees.

### 4.2 3-Tier `gotreesitter` Extraction & Chunking (`internal/code/extract`)

1. **Language Ladder (`30+` languages, zero CGo)**:
   - **Tier 1 (`rich`, 6 ecosystems / 8 grammars)**: `go`, `typescript`, `tsx`, `javascript`, `jsx`, `python`, `rust`, `java`, `csharp` — full AST symbol spans, parameter/return types, visibility/modifiers, docstrings, scoped imports/re-exports, receiver/member call sites, heritage (`extends`/`implements`), contract sites (2C), and statement-level `BasicBlock` builders (2D).
   - **Tier 2 (`map`, 22 grammars)**: `c`, `cpp`, `ruby`, `php`, `kotlin`, `swift`, `scala`, `dart`, `lua`, `bash`, `sql`, `hcl`, `elixir`, `erlang`, `haskell`, `ocaml`, `zig`, `css`, `scss`, `html`, `protobuf`, `graphql` — declarative node-type AST extraction for definitions, imports, and calls.
   - **Tier 3 (`doc` & `regex`)**: Markdown (`#` heading hierarchy -> `section` symbols & `doc_section` chunks), YAML/TOML/JSON (`config_key` top-level keys), and panic-safe regex fallback.
2. **Self-Healing Worker Pool & Span Stack**:
   - `GOMAXPROCS` parser pool wraps every parse in `recover()`. `MaxRespawns = 3` per worker; `TripAfter = 3` panics on the same language trips a per-run circuit breaker demoting that grammar to `regex` without aborting the index run.
   - A lexical interval stack tracks the innermost enclosing `code_symbols` span at every AST node so every call/reference/import binds deterministically to its enclosing callable/type `uid` or the file's synthetic `kind = 'file'` symbol `uid`.
3. **Chunking & Segment Splitting**:
   - Callables `<= 1,200` chars become a single `ast` chunk; larger callables split on statement boundaries (`80` lines, `10` lines overlap) into `window` chunks; classes/structs/interfaces emit a body-stripped `skeleton` chunk (`signature + method/field list`).
   - Identifier splitter segments `parseHTTPRequestURL` -> `["parse", "http", "request", "url"]` (`>= 2` chars, lowercased, deduplicated per symbol) into `code_name_segments`.

### 4.3 Cooperative Writer Batching & Incremental Cross-File Resolution (`internal/code/resolve`)

```text
Modified/Deleted Paths
  -> Batch TXN (50 files/tx, PriorityBatch + WalCheckpointValve):
       1. Resurrect incoming cross-file `code_edges` targeting modified/deleted files -> `code_unresolved_refs(status='pending')`
       2. Delete removed `code_files` (ON DELETE CASCADE cleans symbols, chunks, outgoing edges, segments, embeddings, PDG)
       3. Upsert modified `code_files`, `code_symbols`, `code_chunks`, `code_name_segments`, intra-file `code_edges`, new `pending` refs
       4. Re-open `status='failed'` refs whose `name_tail` matches newly exported symbol names OR when workspace manifests changed
       5. Yield `writeQueue` to any pending `PriorityHigh` (`mem_save`) operations
  -> Cross-File Resolution Pass (chunked 500 refs/tx):
       6. Resolve `pending` refs via 6 Tier-1 Workspace Resolvers -> insert `code_edges` (`EXTRACTED`/`INFERRED`/`AMBIGUOUS`) or mark `failed`
       7. Increment `code_index_meta.graph_generation` + link `observation_code_links`
```

**Tier-1 Workspace Manifest & Resolution Rules**:
- **Go**: Parses `go.work` + `go.mod` module paths; maps package import paths to repo directories; resolves package-qualified calls, struct method receivers, and structural interface satisfaction (`implements` with `INFERRED` confidence).
- **TypeScript / JavaScript**: Parses `tsconfig.json` (`baseUrl`, `paths`, project references) + `package.json` (`workspaces`, `exports`); follows `index.{ts,tsx,js}` barrel `export *` / named re-exports cleanly across cycles.
- **Python**: Parses `pyproject.toml` (`tool.setuptools`, `tool.poetry`, `src/` layout); resolves absolute & relative dotted imports (`.`, `..`) and `__init__.py` `__all__` re-exports.
- **Rust**: Parses Cargo workspace `Cargo.toml` members; resolves `mod`, `use crate::...`, `super::...`, `impl Trait for Type`, and inherent `impl Type` methods.
- **Java**: Parses Maven `pom.xml` / Gradle `build.gradle(.kts)` source roots; resolves `package` + `import` declarations, wildcard package imports, class inheritance, and `implements`.
- **C#**: Parses `*.sln` / `*.csproj` namespaces; resolves `using` directives, partial classes across files, and interface implementations.

### 4.4 CSR Graph Projection & Phase 2B Topology (`internal/code/csr`)

`csr.Graph` loads `code_symbols` (`id`, `kind`, `file_id`, `is_generated`, `is_test`) and `code_edges` (`from`, `to`, `kind`, `confidence`) into flat `[]uint32` forward and reverse CSR arrays (`~6 MiB` RAM per `100,000` edges), cached in `engine.Local` keyed by `code_index_meta.graph_generation`.

1. **Confidence Weights**: `LSP_RESOLVED = 1.00`, `EXTRACTED = 0.95`, `SYNTHESIZED = 0.85`, `INFERRED = 0.65`, `AMBIGUOUS = 0.30`.
2. **Deterministic Leiden Clustering (`2B`)**: Runs modularity + refinement passes (`seed = 0xc0de`, resolution $\gamma = 1.0$, excluding `AMBIGUOUS` edges when graph `> 10,000` nodes). Names each community `<top_dir>/<hub_symbol_name>` using its highest global PageRank member and persists `code_communities` + `code_symbols.community_id`.
3. **Architectural `god_nodes` & `cycles` (`2B`)**:
   - `god_score(u) = log(1 + in_deg(u)) * log(1 + out_deg(u)) * (1 + cross_community_ratio(u))` (supports `exclude_generated` and `exclude_tests` by default).
   - Tarjan's strongly connected components algorithm detects both symbol-level call cycles and file/package-level import cycles.
4. **Weakest-Hop `shortest_path` & Execution `flows` (`2B`)**:
   - Dijkstra over `-ln(confidence_weight)` finds the highest-confidence shortest path between any two symbols/files (`directed` or `undirected`), reporting every hop and `min_confidence`.
   - Entry-point scorer ranks `main()`, HTTP/gRPC/MCP/CLI handlers, and uncalled exported facade functions -> runs bounded DFS (`3..10` steps along `calls`/`implements`/`handles` edges) -> deduplicates traces round-robin by terminal node into `code_flows` + `code_flow_steps`.

### 4.5 Phase 2C Framework & Contract Synthesis (`internal/code/contracts`)

AST pattern rules over parsed Tier-1 trees emit normalized `code_contracts` and `code_contract_edges` (`SYNTHESIZED`):

| Ecosystem | HTTP / gRPC Routes | SQL / ORM Tables | Events / Tools / Configs |
|---|---|---|---|
| **Go** | `net/http` `ServeMux`, `chi`, `gin`, `echo`, `fiber`, `grpc.ServiceDesc` | `database/sql` SQL literals, `sqlc`, `gorm` models | `mcp-go` `NewTool`, `cobra`/CLI commands, `os.Getenv` |
| **TS / JS** | Next.js App/Pages Router, Express, Fastify, Hono, NestJS (`@Controller`/`@Get`), `fetch`/`axios` client URLs | Prisma schema & client, Drizzle `pgTable`/`sqliteTable`, TypeORM `@Entity` | MCP SDK `server.tool`, EventEmitter / Kafka / BullMQ topics, `process.env` |
| **Python** | FastAPI (`@router.get`), Flask (`@app.route`), Django `urlpatterns` | SQLAlchemy `__tablename__`, Django `models.Model`, raw SQL strings | FastMCP `@mcp.tool`, Celery `@shared_task`, `os.environ` |
| **Rust** | `axum::Router::route`, `actix-web` (`#[get]`, `#[post]`), `tonic` gRPC | `sqlx::query!`, Diesel `table!` | `clap` subcommands, `std::env::var` |
| **Java** | Spring MVC (`@RequestMapping`, `@GetMapping`, `@PostMapping`), JAX-RS | JPA `@Entity` / `@Table`, Spring Data `@Query` | `@KafkaListener` / `@EventListener`, `@Value("${...}")` |
| **C#** | ASP.NET Core Minimal APIs (`MapGet`/`MapPost`) & `[Route]`/`[HttpGet]` | Entity Framework `DbSet<T>` / `[Table]` | MediatR `INotificationHandler`, `IConfiguration["..."]` |

Client `fetch` URLs matching backend `http_route` canonical keys automatically bridge frontend callers to backend handlers across languages inside a monorepo!

### 4.6 Phase 2D Intra-Procedural CFG/PDG/Taint & Optional LSP Pass (`internal/code/analysis`, `internal/code/lsp`)

1. **Intra-Procedural CFG, CDG & Reaching Definitions**:
   - Statement walker partitions each Tier-1 function/method AST into `code_cfg_blocks` (`entry`, `body`, `branch`, `loop_head`, `return`, `throw`, `exit`) with `cfg_next`/`cfg_true`/`cfg_false` edges.
   - Lengauer-Tarjan post-dominator tree on the reverse CFG computes control-dependence edges (`cdg`). Iterative bitset dataflow computes `reaching_def` edges `(def_block -> use_block, var_name)`.
2. **Bottom-Up Inter-Procedural Taint Summaries**:
   - Processes Tarjan call-graph SCCs bottom-up (fixed-point iteration within recursive SCCs, max 4 passes).
   - For each callable, tracks parameter bitmasks (`uint64` for up to 64 params) and built-in taint sources (`http_input`, `rpc_arg`, `cli_arg`, `env_var`, `file_read`) through assignments and calls (`param_to_return_mask`), clearing taint bits when values pass through recognized sanitizers (parameterized SQL placeholders, `filepath.Clean` + prefix confine, `html.EscapeString`, `url.QueryEscape`, allowlist parsers) before reaching sinks (`sql_exec`, `cmd_exec`, `fs_path`, `html_out`, `ssrf_url`).
   - `code_taint` reconstructs concrete multi-function `source -> call_chain -> sink` traces on demand with file/line evidence.
3. **Optional JSON-RPC 2.0 Stdio LSP Precision Pass (`--lsp` / `code_lsp_sync`)**:
   - Supported servers: `gopls` (Go), `typescript-language-server --stdio` (TS/JS), `pyright-langserver --stdio` (Python), `rust-analyzer` (Rust), `jdtls` (Java), `csharp-ls` (C#).
   - Queries `textDocument/definition` and `textDocument/implementation` **only** for `code_unresolved_refs` and `INFERRED`/`AMBIGUOUS` call/interface edges (bounded concurrency = 8, per-request timeout `2s`), promoting verified matches to `code_edges(confidence = 'LSP_RESOLVED')`. Daemon reaps idle LSP subprocesses after `5m` of inactivity.

---

## 5. Hybrid RWR Retrieval (`code_explore`) & Safe Refactoring

### 5.1 `code_explore` Ranking & Budget Allocation

1. **Seed Selection (4-Signal RRF, $K = 60$, top 60 seeds)**:
   - **Symbol FTS Leg**: `code_symbols_fts` BM25 `(name=20, qualified_name=8, signature=3, docstring=1)`.
   - **Identifier Segment Leg**: Tokenizes natural-language query words, joins `code_name_segments`, and scores symbols by `(matched_query_terms / total_query_terms)^2 * idf_sum`.
   - **Chunk Trigram Leg**: `code_chunks_fts` (`>= 3` runes) mapped to owning `symbol_id`.
   - **Semantic Vector Leg**: Asymmetric 384-dim ONNX query embedding against `code_symbols_vec` KNN ($k = 60$); skipped cleanly with `degraded: true` when `embedding.provider == "none"` or backfill is in progress.
2. **Structural Expansion (CSR Personalized PageRank)**:
   - Runs 25 power iterations ($\alpha = 0.25$ restart probability onto RRF seed mass) across `calls`, `implements`, `extends`, `type_of`, `returns`, and `code_contract_edges`.
   - Applies rank multipliers (`0.30x` when `is_generated = 1`, `0.50x` when `is_test = 1` unless the query explicitly mentions `test`/`spec`/`mock`) to **both** initial seed priors and final RWR scores.
   - Rescues referenced parameter/return struct/interface definitions (`type_of`/`returns`) of top-5 callables when their standalone score fell below the cutoff.
3. **Proportional `<= 24 KiB` Whole-Symbol Cluster Allocator**:
   - Applies a relative cliff (`>= 15%` of top symbol weight, max 12 primary symbols), groups selected symbols by `file_path`, and merges adjacent/overlapping symbol spans (`gap <= 8` lines) into contiguous line clusters — **never truncating a selected function mid-body** (oversize functions switch automatically to `skeleton` + matched hunk window).
   - Allocates byte budget (`default 24,576` bytes, max `70%` to any single file), reads live UTF-8 source from `active_root`, redacts `<private>...</private>` and high-entropy credential literals, and attaches:
     - Outgoing/incoming call & contract trail headers per cluster.
     - **`validateAnswerFiles` Freshness Check** (`<= 250 ms`, max 200 files / `8 MiB`): compares live disk `(size, mtime_ns, sha256)` against `code_files.content_hash` and stamps `stale_files[]` if edited since last index.
     - **Linked Memories**: Joins `observation_code_links` + `observations` (`deleted_at IS NULL`) to append compact `[Memory: <type>] <title> (uid)` annotations for touched files/symbols.

### 5.2 Safe Two-Step `code_rename`

1. **Step 1 (`dry_run = true`, default)**: Resolves target `symbol_uid`, gathers declaration + all `LSP_RESOLVED` / `EXTRACTED` references across `code_files` (flagging any `INFERRED` / `AMBIGUOUS` sites separately as `unverified_candidates[]` without auto-editing them), verifies lexical identifier boundaries on disk, builds a unified diff preview, and computes `plan_token = sha256(symbol_uid || new_name || sorted(file_path + content_hash))`.
2. **Step 2 (`dry_run = false`, `plan_token` required)**: Re-hashes every participating file on disk; if any file drifted since preview, aborts with `409 FINGERPRINT_CONFLICT` (`PLAN_STALE`) and zero disk mutations. Otherwise writes temp files in each target directory, `fsync`s, atomically renames over targets, and triggers immediate incremental re-indexing of the touched files.

---

## 6. Transport Surfaces: MCP, Daemon, CLI & TUI

### 6.1 MCP Server (`mnemonic code mcp --profile agent|admin|all`)

Strict `draft-07` JSON Schema (`additionalProperties: false`). Tools register dynamically only when their slice capability is active (**zero stubs**):

| Tier | Slice | Tool | Purpose & Modes |
|---|---|---|---|
| **Eager Core (7)** | 2A | `code_explore` | Budgeted (`<=24 KiB`) RRF + PageRank clustered source, call trails, staleness check & linked memories |
| **Eager Core** | 2A | `code_read` | Repo-root-confined reader (`mode`: `slice` \| `skeleton` \| `symbol`) with neighbor IDs & linked memories |
| **Eager Core** | 2A | `code_impact` | Reverse BFS blast radius (`depth 1..10`, `min_confidence`), grouped into `will_break` (hop 1) vs `likely_affected` |
| **Eager Core** | 2A | `code_affected` | Maps `git diff` hunks (`working` \| `staged` \| `base_ref`) to modified symbols, impacted callers, affected tests & 2B `flows` |
| **Eager Core** | 2B | `code_topology` | Architectural graph queries (`mode`: `overview` \| `communities` \| `god_nodes` \| `flows` \| `cycles` \| `path`) |
| **Eager Core** | 2A | `code_status` | Freshness (`not_indexed` \| `fresh` \| `stale` \| `corrupt`), dirty file counts, active capabilities, embedding & LSP status |
| **Eager Core** | 2A | `code_index` | Triggers incremental or `--full` sync (with optional `lsp: true` in 2D) |
| **Extended (`DeferLoading`) (6)** | 2A | `code_search` | Compact ranked symbol/chunk hits (`mode`: `hybrid` \| `lexical` \| `semantic` \| `structural` AST pattern) |
| **Extended (`DeferLoading`)** | 2A | `code_neighbors` | Direct 1-hop structural queries (`mode`: `callers` \| `callees` \| `implementors` \| `hierarchy` \| `tests`) |
| **Extended (`DeferLoading`)** | 2A | `code_rename` | 2-step safe symbol rename (`dry_run` preview + `plan_token` atomic apply) |
| **Extended (`DeferLoading`)** | 2C | `code_contracts` | Cross-boundary queries (`mode`: `routes` \| `queries` \| `events` \| `tools` \| `docs` \| `configs`) |
| **Extended (`DeferLoading`)** | 2D | `code_pdg` | Intra-procedural CFG, control dependence (`cdg`), and `reaching_def` slice for a symbol |
| **Extended (`DeferLoading`)** | 2D | `code_taint` | Source-to-sink taint paths across call chains (`sink_kind` filter, `include_sanitized`) |
| **Admin (`DeferLoading`) (3)** | 2A | `code_watch` | Manage/inspect daemon recursive directory watcher for active project |
| **Admin (`DeferLoading`)** | 2A | `code_backfill_embeddings` | Force symbol ONNX `vec0` embedding backfill |
| **Admin (`DeferLoading`)** | 2D | `code_lsp_sync` | Run targeted stdio LSP edge-promotion pass and report upgraded edge counts |

### 6.2 Daemon Routes (`/v1/code/*`) & Background Workers

- **HTTP / Unix Socket Endpoints**:
  - `GET /v1/code/status`, `POST /v1/code/index`, `POST /v1/code/explore`, `POST /v1/code/read`, `POST /v1/code/search`
  - `POST /v1/code/impact`, `POST /v1/code/affected`, `POST /v1/code/neighbors`, `POST /v1/code/rename`
  - `POST /v1/code/topology` (2B), `POST /v1/code/contracts` (2C), `POST /v1/code/pdg` (2D), `POST /v1/code/taint` (2D)
  - `POST /v1/code/watch`, `POST /v1/code/backfill`, `POST /v1/code/lsp-sync`
- **Daemon Background Workers (`Subsystem.BackgroundWorkers`)**:
  1. **Recursive FS Watcher**: Subscribes non-ignored subdirectories of active watched projects (`1.5s` debounce), dynamically adding newly created directories and triggering incremental delta syncs.
  2. **Symbol Embedding Backfill Worker**: Batches (`32` symbols/step) unembedded or hash-changed `code_symbols` through `platform/intelligence` ONNX/HTTP embedder without blocking interactive queries.
  3. **LSP Subprocess Reaper (2D)**: Terminates idle stdio language servers after `5m` of inactivity or on daemon shutdown.

### 6.3 CLI Reference & `[2: Code]` TUI Tab

```text
mnemonic code index [--full] [--lsp] [--json]
mnemonic code status [--json]
mnemonic code explore "<query>" [--budget 24576] [--include-tests] [--json]
mnemonic code read <path> [--symbol <name-or-uid>] [--lines <start:end>] [--mode slice|skeleton|symbol] [--json]
mnemonic code search "<query-or-ast-pattern>" [--mode hybrid|lexical|semantic|structural] [--lang <lang>] [--limit 20] [--json]
mnemonic code impact <symbol-or-path> [--depth 5] [--min-confidence INFERRED] [--json]
mnemonic code affected [--staged | --base <git-ref>] [--depth 5] [--json]
mnemonic code neighbors <symbol> --mode callers|callees|implementors|hierarchy|tests [--json]
mnemonic code topology --mode overview|communities|god_nodes|flows|cycles|path [--from <a> --to <b>] [--json]
mnemonic code contracts --mode routes|queries|events|tools|docs|configs [--query "<filter>"] [--json]
mnemonic code pdg <symbol> [--json]
mnemonic code taint [--symbol <sym>] [--sink sql_exec|cmd_exec|fs_path|html_out|ssrf_url] [--json]
mnemonic code rename <symbol> <new-name> [--apply --token <plan-token>] [--json]
mnemonic code watch [--debounce 1500ms]          # Blocks until SIGINT/SIGTERM, streaming JSON/text sync events
mnemonic code backfill [--json]
mnemonic code lsp-sync [--lang <lang>] [--json]
mnemonic code mcp [--profile agent|admin|all] [--standalone]
```

**`mnemonic tui` `[2: Code]` Tab**:
- **Explore / Search Pane (`/`)**: Live RRF + PageRank query bar, language/confidence chips, ranked symbols with community & PageRank badges.
- **Symbol & Source Inspector (`Enter`)**: Syntax-highlighted source slice, callers/callees tree, contract badges, and linked Phase 1 `observations` (press `m` to jump directly to `[1: Memory]` inspector).
- **Topology View (`t`)**: Leiden communities list, `god_nodes` leaderboard, and step-by-step execution `flows` viewer.
- **Contracts & Taint View (`x`)**: Filterable route/table catalog and interactive `source -> sink` taint trace inspector.

---

## 7. Error Handling, Self-Healing & Verification Strategy

### 7.1 Additional Canonical Error Codes

Extends Phase 1's error table (`VALIDATION_ERROR=2`, `UNAUTHORIZED=3`, `NOT_FOUND=4`, `FINGERPRINT_CONFLICT=5`, `UNSUPPORTED_FS=7`, `STORAGE_CORRUPT=8`):

| Error Code | CLI Exit | HTTP Status | MCP Envelope | Trigger |
|---|---|---|---|---|
| `FINGERPRINT_CONFLICT` (`PLAN_STALE`) | `5` | `409` | `isError: true` | `code_rename --apply` file hash drifted from `plan_token` |
| `PATH_OUTSIDE_ROOT` | `9` | `403` | `isError: true` | Requested path or symlink target escapes `code_index_meta.active_root` |
| `INDEX_BUSY` | `10` | `429` | `isError: true` | Concurrent explicit `code_index --full` already running on the same `<repo>.db` |
| `LSP_UNAVAILABLE` | `11` | `503` | `isError: true` | Explicit `code_lsp_sync` called when no matching LSP binary is installed (`code_index --lsp` logs warning and degrades instead of failing) |

- **`mnemonic doctor --repair` Extensions**: Verifies `code_symbols` / `code_edges` FK integrity, rebuilds `code_symbols_fts` and `code_chunks_fts` (`INSERT INTO ... VALUES('rebuild')`), regenerates `code_name_segments` from `code_symbols`, rebuilds `code_symbols_vec` from `code_symbol_embeddings`, and bumps `graph_generation` to force CSR & 2B topology refresh.

### 7.2 Verification & Compliance Requirements

1. **Zero-Mock-Extractor Compliance Rule (`internal/code/api/compliance`)**:
   - Integration tests **must** execute real `gotreesitter` extractions across multi-file polyglot fixture repositories covering all **6 Tier-1 ecosystems** (`Go`, `TS/TSX`, `Python`, `Rust`, `Java`, `C#`) against **both** `engine.Local` (standalone) and `client.DaemonClient` (Unix socket + HTTP). Mock extractors are permitted only for parser panic/circuit-breaker fault-injection tests.
2. **Incremental Resurrection Property Tests (`internal/code/resolve`)**:
   - Modifies, renames, moves, and deletes exported functions, barrel `index.ts` / `__init__.py` files, and workspace manifests across random edit sequences, asserting that the incremental SQLite state (`code_symbols`, `code_edges`, `code_unresolved_refs`) is **100% isomorphic** to a clean `code index --full` build from scratch.
3. **Priority-Fairness & WAL Valve Stress Benchmark (`internal/platform/sqlitepool`)**:
   - Runs a full 5,000-file repository index concurrently with 10 goroutines issuing `mem_save` + `code_explore` calls, asserting zero `SQLITE_BUSY` errors, bounded WAL size (`<= 512 MiB`), and `p99 mem_save < 25 ms`.
4. **CWD Independence & Symlink Escape Suite (`internal/code/scan`, `internal/code/search`)**:
   - Executes `code_index`, `code_explore`, `code_read`, `code_search --mode structural`, and `code_rename` while process `os.Getwd()` is `/tmp` (outside the repository) and asserts symlink escapes return `PATH_OUTSIDE_ROOT`.
5. **Sub-Phase Precision Suites**:
   - **2B**: Deterministic Leiden repeatability check (identical partition across 10 runs on shuffled edge insertion orders), Tarjan cycle accuracy, and weakest-hop path confidence tests.
   - **2C**: Cross-language `TS fetch('/v1/orders') -> Go/Python/Java/Rust/C# POST /v1/orders` route bridging + ORM table read/write verification.
   - **2D**: Seeded SQLi / command-injection / path-traversal vulnerable vs sanitized fixtures verifying `code_taint` recalls true unsanitized flows with zero false positives on sanitized variants, plus live `gopls` stdio LSP upgrade test (skippable only if `gopls` absent).

### 7.3 Implementation Plan Sequence

Implementation is decomposed into four sequential, self-contained plans under `docs/superpowers/plans/`:

1. `docs/superpowers/plans/2026-11-24-mnemonic-phase2a-core-code-graph.md` — Platform `WalCheckpointValve`/priority queue, `001_code_core.sql`, scanner/watcher, 3-tier `gotreesitter` extractor, 6 Tier-1 resolvers + incremental resurrection, FTS5/segments/ONNX `vec0`, CSR PageRank `code_explore`, `code_read`/`search`/`impact`/`affected`/`neighbors`/`rename`, `observation_code_links`, CLI/MCP/Daemon/TUI.
2. `docs/superpowers/plans/2026-11-24-mnemonic-phase2b-graph-topology-flows.md` — `002_code_topology.sql`, deterministic Leiden clustering + hub labels, `god_nodes`, Tarjan `cycles`, weakest-hop `shortest_path`, ranked entry-point DFS `flows`, Eager Core `code_topology`, flow blast radius in `code_affected`, and TUI Topology view.
3. `docs/superpowers/plans/2026-11-24-mnemonic-phase2c-framework-contracts.md` — `003_code_contracts.sql`, 6-ecosystem HTTP/gRPC route, SQL/ORM table, pub-sub event, MCP/CLI tool, and doc/config synthesizers, cross-language client->route bridging, and `code_contracts`.
4. `docs/superpowers/plans/2026-11-24-mnemonic-phase2d-pdg-taint-lsp.md` — `004_code_analysis.sql`, intra-procedural `BasicBlock` CFG, Lengauer-Tarjan CDG, reaching definitions, bottom-up bitmask `code_taint_summaries` + `code_taint` / `code_pdg`, and JSON-RPC 2.0 stdio LSP pool (`code_lsp_sync`).

