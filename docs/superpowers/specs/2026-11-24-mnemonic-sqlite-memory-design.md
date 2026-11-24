# Mnemonic Local-First SQLite AI Memory — Design Specification

- **Date**: 2026-11-24
- **Status**: Approved for Implementation Planning
- **Repository**: `skillgrid-mnamonic` (`mnemonic` binary)
- **Lineage**: Synthesized from `skillgrid-v2/mnemonic` and `engram`

---

## 1. Purpose & Goals

Mnemonic is a **strictly local-first, zero-CGo Go persistent memory engine for AI agents**, backed exclusively by SQLite (`~/.skillgrid/mnemonic/global.db` + `~/.skillgrid/mnemonic/<repo>.db`).

It combines the strongest engineering qualities of its two predecessors while eliminating their architectural debt:

- **From `engram`**:
  - Operational SQLite hardening (`WAL`, `BEGIN IMMEDIATE`, `FileControlPersistWAL`, NFS/SMB refusal, inode generation fencing).
  - Token-efficient MCP progressive disclosure (`WithDeferLoading(true)` + 3-layer `PreviewSearch -> timeline -> get_observation` retrieval).
  - Replay-safe save idempotency (`save_operations`), `topic_key` versioned upserts (`observation_versions`), `<private>` redaction, pairwise conflict surfacing, and an interactive Bubbletea TUI.
- **From `skillgrid-v2/mnemonic`**:
  - Strict shared-service handler parity across CLI and MCP.
  - Hybrid Reciprocal Rank Fusion (`RRF`, $K=60$) combining FTS5 trigram BM25 with vector KNN (`modernc.org/sqlite/vec` `vec0`) and composite scoring (`pinned`, `importance`, `retrieval_usage`, recency decay, conflict penalties).
  - Deterministic offline degradation plus bounded session and retrieval trails.
- **New Capabilities & Debt Eliminated**:
  - **First-class local ONNX CPU embeddings** (quantized `BAAI/bge-small-en-v1.5`, 384-dim, zero CGo) as the default semantic tier, paired with optional external `Ollama` / `OpenAI`-compatible HTTP embeddings and optional LLM write-time semantic dedup/consolidation.
  - **Isolated Dual-Tier Local Storage**: `~/.skillgrid/mnemonic/global.db` (`user` & `global` memories + project catalog) + `~/.skillgrid/mnemonic/<repo>.db` (`project` memories), eliminating cross-repo write contention while keeping cross-scope recall fast.
  - **Shared Local Daemon + Seamless In-Process Fallback**: Auto-spawned background daemon (`~/.skillgrid/mnemonic/run/daemon.sock` + authenticated loopback HTTP) holds **1x** ONNX model in RAM across all IDE sessions, with automatic in-process fallback for restricted sandboxes.
  - **Modular Subsystem Architecture (`mnemonic memory ...`)**: Built around a reusable `internal/platform` kernel and pluggable subsystem contracts so future pillars (e.g., `mnemonic code ...`, `mnemonic wiki ...`) attach without refactoring core packages or breaking CLI/MCP/daemon namespaces.
  - **Zero Sync/Cloud/Dual-DB Bloat**: No PostgreSQL dialect seams (`RawQuery` type assertions), no cloud outbox/tombstone/chunk replication tables, no 15k-LOC monolithic files, and zero stub MCP tools.

### Non-Goals (Current Scope)

1. Cloud sync, PostgreSQL storage, or Git chunk replication.
2. AST symbol/codebase indexing (`code_*`) and Markdown knowledge vault synthesis (`projectwiki_*`) — explicitly deferred to future sibling subsystems under the extensible registry.
3. Embedded browser SPA dashboard (the Bubbletea TUI and CLI provide local inspection).

---

## 2. System Architecture & Extensibility Boundaries

### 2.1 Package Layout

Every Go source file must remain focused on a single responsibility (target `<600` LOC per file; split by aggregate rather than growing monoliths).

```text
cmd/mnemonic/
└── main.go                       # Root CLI entry: dispatches subsystems + platform commands
internal/
├── platform/                     # Shared foundation for `memory` and future subsystems
│   ├── config/                   # ~/.skillgrid/mnemonic/config.yaml + .mnemonic.yaml + env/flag loader
│   ├── project/                  # Canonical repo slug resolver, alias lookup, reserved-name guard
│   ├── sqlitepool/               # Ref-counted DB handles (1 writer + N readers), WAL, inode fence, writeQueue
│   ├── migrate/                  # Per-subsystem migration runner (`schema_migrations(subsystem, version)`)
│   ├── intelligence/             # Embedder (ONNX CPU + HTTP `/v1/embeddings`), LLM client, circuit breaker
│   ├── ipc/                      # Envelope DTOs, error codes, Unix socket + bearer-authed HTTP server/client
│   ├── daemon/                   # Background lifecycle (pidfile, token, auto-spawn, idle reaper, workers)
│   ├── tuishell/                 # Top-level Bubbletea frame & subsystem tab router
│   └── registry/                 # Pluggable `Subsystem` contract registration
└── memory/                       # The `memory` subsystem module
    ├── api/                      # `Service` interface, domain models, filter/request/response types
    ├── store/                    # Pure SQLite repository (observations, sessions, prompts, relations, vectors, trails)
    │   └── migrations/           # `001_init.sql` (subsystem = "memory")
    ├── engine/                   # `Local` implementation of `api.Service` (pipeline, RRF search, judging, consolidate)
    ├── client/                   # `DaemonClient` implementation of `api.Service` over IPC (`daemon.sock` / HTTP)
    ├── resolver/                 # `ResolveBackend`: connects/auto-spawns daemon or falls back to `engine.Local`
    ├── mcp/                      # `mnemonic memory mcp` stdio server (`mcp-go`, progressive disclosure)
    ├── http/                     # `/v1/memory/*` JSON handlers mounted on daemon mux
    ├── cli/                      # `mnemonic memory ...` subcommands
    └── tui/                      # Memory TUI screens registered into `tuishell`
```

### 2.2 Extensible Subsystem Contract (`platform/registry`)

Future subsystems (`code`, `wiki`, etc.) implement:

```go
type Subsystem interface {
    Name() string                                      // e.g. "memory"
    Migrations() fs.FS                                 // tracked under schema_migrations.subsystem = Name()
    MountHTTP(mux *http.ServeMux, rt *Runtime) error   // mounts under /v1/<name>/*
    CLICommand(rt *Bootstrap) *cli.Command             // mounts `mnemonic <name> ...` (incl. `mnemonic <name> mcp`)
    BackgroundWorkers(rt *Runtime) []Worker            // daemon background tasks (e.g. vector backfill)
    TUITab(rt *Bootstrap) tuishell.Tab                 // mounts `[Memory]` tab in `mnemonic tui`
}
```

### 2.3 Runtime Topology & Backend Resolution

```text
┌──────────────────────┐  ┌──────────────────────┐  ┌──────────────────────┐
│ mnemonic memory mcp  │  │ mnemonic memory ...  │  │     mnemonic tui     │
│   (IDE session 1..N) │  │     (CLI / hooks)    │  │  (Bubbletea browser) │
└──────────┬───────────┘  └──────────┬───────────┘  └──────────┬───────────┘
           │                         │                         │
           └─────────────────────────┼─────────────────────────┘
                                     ▼
                       memory/resolver.ResolveBackend()
                                     │
            ┌────────────────────────┴────────────────────────┐
            │ Normal Desktop Mode                             │ --standalone OR sandbox blocks sockets
            ▼                                                 ▼
┌───────────────────────────┐                     ┌───────────────────────────┐
│ memory/client.DaemonClient│                     │    memory/engine.Local    │
│  Unix: run/daemon.sock    │                     │   (In-process fallback)   │
└─────────────┬─────────────┘                     └─────────────┬─────────────┘
              ▼                                                 │
┌───────────────────────────────────────────┐                   │
│             mnemonic daemon               │                   │
│  • 1x Shared ONNX CPU Model in RAM        │                   │
│  • Background Vector Backfill & Pruning   │                   │
│  • Delegates to memory/engine.Local       │                   │
└─────────────────────┬─────────────────────┘                   │
                      └────────────────────┬────────────────────┘
                                           ▼
                      ┌─────────────────────────────────────────────┐
                      │             platform/sqlitepool             │
                      │ ~/.skillgrid/mnemonic/global.db + <repo>.db │
                      │  (1 Writer + N WAL Readers + Inode Fence)   │
                      └─────────────────────────────────────────────┘
```

1. Unless `--standalone` (`MNEMONIC_STANDALONE=1`) is set, `ResolveBackend` dials `~/.skillgrid/mnemonic/run/daemon.sock`.
2. If the socket is absent or unresponsive, `ResolveBackend` acquires `~/.skillgrid/mnemonic/run/spawn.lock` and spawns `mnemonic daemon run --idle-timeout=30m` detached, polling readiness for up to `1.5s`.
3. If process spawning or Unix socket binding fails (e.g., restricted container sandbox), `ResolveBackend` logs a structured diagnostic note to stderr and returns `memory/engine.Local` directly in-process.

---

## 3. Filesystem Layout, Storage Pool & SQLite Schema

### 3.1 Data Directory Layout (`~/.skillgrid/mnemonic/`)

Overridable via `MNEMONIC_DATA_DIR` or `--data-dir`:

```text
~/.skillgrid/mnemonic/
├── config.yaml                  # Global configuration
├── global.db                    # Project registry/aliases + all `user` & `global` scope memories
├── <repo>.db                    # One isolated SQLite DB per canonical repo slug (`project` scope)
├── models/                      # Verified ONNX models + tokenizers
│   └── bge-small-en-v1.5/
│       ├── model_quantized.onnx
│       └── tokenizer.json
└── run/                         # 0700 directory
    ├── daemon.sock              # 0600 Unix domain socket
    ├── daemon.pid               # Daemon PID + startup nonce
    ├── daemon.token             # 0600 random 256-bit hex bearer token for loopback HTTP
    └── daemon.log               # Rotating daemon log (max 10 MiB x 3)
```

### 3.2 Canonical Project Resolution & Reserved Names

Project slug `<repo>` is resolved in priority order:
1. Explicit `--project` / MCP `project` argument.
2. `MNEMONIC_PROJECT` environment variable.
3. `.mnemonic.yaml` `project:` key at repository root.
4. Alias match in `global.db` `project_aliases` (by normalized CWD or Git remote URL).
5. Normalized Git remote `owner-repo` or repository root directory basename (NFC-normalized, lowercase `[a-z0-9._-]`).

**Reserved Filename Guard**: Slugs matching `global`, `config`, `models`, `run`, `tmp`, or `wal` are automatically prefixed as `repo-<slug>` (e.g., `repo-global.db`) so a repository named `global` can never collide with `global.db`.

### 3.3 Scope Routing & Multi-DB Federation

| Operation | `scope = "project"` (default) | `scope = "user"` or `"global"` | Unscoped Read (`scope = ""`) | `all_projects = true` Read |
|---|---|---|---|---|
| **Target DB(s)** | `~/.skillgrid/mnemonic/<repo>.db` (+ touch `projects` row in `global.db`) | `~/.skillgrid/mnemonic/global.db` | Parallel read of `<repo>.db` + `global.db` | Parallel read of `global.db` + all existing `<repo>.db` in `projects` (bounded semaphore = 8) |

Each observation carries a globally unique **UUIDv7 `uid`**, guaranteeing collision-free IDs across `global.db` and all `<repo>.db` files without cross-database sequence coordination. For ID-targeted operations (`get`, `update`, `pin`, `unpin`, `delete`, `timeline`), if `scope` is omitted, the engine probes the active `<repo>.db` first and falls back to `global.db` via `idx_obs_uid`.

### 3.4 SQLite Pool & Operational Hardening (`platform/sqlitepool`)

- **Driver**: Pure-Go `modernc.org/sqlite` + blank import `_ "modernc.org/sqlite/vec"` (registers `vec0`, `vec_f32`, `vec_distance_cosine`).
- **Dual Handle per `.db` File**:
  - **Writer `*sql.DB`**: `SetMaxOpenConns(1)`, DSN `_txlock=immediate`, guarded by a per-DB buffered Go `writeQueue` (capacity 64) with a 10s context timeout so concurrent goroutines never trigger `SQLITE_BUSY_SNAPSHOT`.
  - **Reader `*sql.DB`**: `SetMaxOpenConns(min(8, NumCPU))`, `PRAGMA query_only = 1` for lock-free concurrent WAL reads.
- **PRAGMAs applied on every connection**:
  - `PRAGMA journal_mode = WAL;` (plus `FileControlPersistWAL("main", 1)`).
  - `PRAGMA synchronous = NORMAL;`
  - `PRAGMA busy_timeout = 5000;`
  - `PRAGMA journal_size_limit = 67108864;` (64 MiB).
  - `PRAGMA foreign_keys = ON;`
  - `PRAGMA temp_store = MEMORY;`
- **Filesystem & Inode Guards**:
  - **Network FS Refusal**: Statfs check on startup rejects `nfs`, `smb`, `cifs`, and `afs` mounts with an explicit actionable error instructing the user to point `MNEMONIC_DATA_DIR` to local disk.
  - **Inode Generation Fence**: Every pooled entry records `os.FileInfo` (`dev`, `ino`) of `<file>.db` on open. Before checkout, `os.Stat` + `os.SameFile` verifies identity; if a user externally deletes or replaces `<repo>.db`, the pool atomically drains and closes the stale handles and opens a fresh generation without restarting the daemon.
  - **LRU Idle Eviction**: Pooled `<repo>.db` handles unused for >15 minutes close automatically (`global.db` stays pinned while daemon runs).

### 3.5 SQLite DDL Specification

#### Platform Tables

Applied to **every** DB (`global.db` and `<repo>.db`):

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    subsystem  TEXT NOT NULL,
    version    INTEGER NOT NULL,
    applied_at TEXT NOT NULL,
    PRIMARY KEY (subsystem, version)
);
```

Applied to **`global.db` only** (`subsystem = "platform"`, version `1`):

```sql
CREATE TABLE IF NOT EXISTS projects (
    slug           TEXT PRIMARY KEY,
    db_file        TEXT NOT NULL UNIQUE,
    root_path      TEXT NOT NULL DEFAULT '',
    remote_url     TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL,
    last_active_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS project_aliases (
    kind       TEXT NOT NULL CHECK (kind IN ('cwd', 'remote', 'key')),
    alias_key  TEXT NOT NULL,
    slug       TEXT NOT NULL REFERENCES projects(slug) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY (kind, alias_key)
);
CREATE INDEX IF NOT EXISTS idx_project_aliases_slug ON project_aliases(slug);
```

#### Memory Subsystem Tables (`subsystem = "memory"`, version `1`)

Applied identically to `global.db` and every `<repo>.db`:

```sql
-- 1. Sessions (combines lifecycle, structured 6-section summary, and finalization state)
CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,
    project      TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    directory    TEXT NOT NULL DEFAULT '',
    started_at   TEXT NOT NULL,
    ended_at     TEXT,
    summary_json TEXT,             -- JSON: {goal, instructions, discoveries, accomplished, next_steps, relevant_files, text}
    finalized_at TEXT,             -- Set atomically by mem_session_finalize (idempotent on retry)
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
CREATE INDEX idx_sessions_project_started ON sessions(project, started_at DESC);

-- 2. Core Observations (lean metadata + text; NO raw float32 BLOBs on this table)
CREATE TABLE observations (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    uid               TEXT NOT NULL UNIQUE,           -- UUIDv7 (agent-visible canonical ID)
    session_id        TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    project           TEXT NOT NULL,
    scope             TEXT NOT NULL DEFAULT 'project' CHECK (scope IN ('project', 'user', 'global')),
    type              TEXT NOT NULL CHECK (type IN (
                          'decision', 'architecture', 'bugfix', 'pattern',
                          'config', 'discovery', 'learning', 'preference', 'convention'
                      )),
    title             TEXT NOT NULL,
    content           TEXT NOT NULL,
    topic_key         TEXT,
    tool_name         TEXT NOT NULL DEFAULT '',
    normalized_hash   TEXT NOT NULL,
    revision_count    INTEGER NOT NULL DEFAULT 1,
    duplicate_count   INTEGER NOT NULL DEFAULT 1,
    pinned            INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1)),
    importance        REAL NOT NULL DEFAULT 0.5,
    retrieval_usage   INTEGER NOT NULL DEFAULT 0,
    superseded_by_uid TEXT,
    review_after      TEXT,
    expires_at        TEXT,
    provenance_json   TEXT NOT NULL DEFAULT '{}',
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    last_seen_at      TEXT NOT NULL,
    deleted_at        TEXT
);
CREATE INDEX idx_obs_uid          ON observations(uid);
CREATE INDEX idx_obs_topic        ON observations(topic_key, scope, updated_at DESC) WHERE deleted_at IS NULL AND topic_key IS NOT NULL;
CREATE INDEX idx_obs_dedupe       ON observations(normalized_hash, scope, type, title, last_seen_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_obs_active_time  ON observations(pinned DESC, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_obs_session      ON observations(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_obs_review       ON observations(review_after) WHERE deleted_at IS NULL AND review_after IS NOT NULL;
CREATE INDEX idx_obs_expires      ON observations(expires_at) WHERE deleted_at IS NULL AND expires_at IS NOT NULL;

-- 3. Immutable Version History (created before topic_key upserts or explicit mem_update)
CREATE TABLE observation_versions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
    version        INTEGER NOT NULL,
    title          TEXT NOT NULL,
    content        TEXT NOT NULL,
    session_id     TEXT,
    created_at     TEXT NOT NULL,
    UNIQUE (observation_id, version)
);

-- 4. Replay-Safe Idempotency Ledger
CREATE TABLE save_operations (
    operation_id   TEXT PRIMARY KEY,
    fingerprint    TEXT NOT NULL,  -- SHA-256 of canonicalized save payload
    observation_id INTEGER REFERENCES observations(id) ON DELETE SET NULL,
    observation_uid TEXT NOT NULL,
    created_at     TEXT NOT NULL
);
CREATE INDEX idx_save_ops_created ON save_operations(created_at);

-- 5. User Prompts
CREATE TABLE user_prompts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    uid             TEXT NOT NULL UNIQUE,
    session_id      TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    project         TEXT NOT NULL,
    content         TEXT NOT NULL,
    normalized_hash TEXT NOT NULL,
    created_at      TEXT NOT NULL
);
CREATE INDEX idx_prompts_session_created ON user_prompts(session_id, created_at DESC);
CREATE INDEX idx_prompts_dedupe          ON user_prompts(session_id, normalized_hash, created_at DESC);

-- 6. Pairwise Conflict & Semantic Relations (within the same DB)
CREATE TABLE memory_relations (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    source_uid      TEXT NOT NULL REFERENCES observations(uid) ON DELETE CASCADE,
    target_uid      TEXT NOT NULL REFERENCES observations(uid) ON DELETE CASCADE,
    relation        TEXT NOT NULL DEFAULT 'pending' CHECK (relation IN (
                        'pending', 'related', 'compatible', 'scoped',
                        'supersedes', 'conflicts_with', 'not_conflict'
                    )),
    judgment_status TEXT NOT NULL DEFAULT 'pending' CHECK (judgment_status IN ('pending', 'judged', 'superseded')),
    confidence      REAL,
    reason          TEXT NOT NULL DEFAULT '',
    evidence        TEXT NOT NULL DEFAULT '',
    marked_by_kind  TEXT NOT NULL DEFAULT 'system',  -- 'system' | 'agent' | 'user' | 'llm'
    marked_by_model TEXT NOT NULL DEFAULT '',
    session_id      TEXT,
    superseded_at   TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    UNIQUE (source_uid, target_uid)
);
CREATE INDEX idx_memrel_source  ON memory_relations(source_uid) WHERE judgment_status != 'superseded';
CREATE INDEX idx_memrel_target  ON memory_relations(target_uid) WHERE judgment_status != 'superseded';
CREATE INDEX idx_memrel_pending ON memory_relations(judgment_status, created_at DESC) WHERE judgment_status = 'pending';

-- 7. Bounded Audit & Retrieval Trails
CREATE TABLE session_trails (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL,     -- 'save' | 'revise' | 'duplicate' | 'judge' | 'consolidate' | 'delete'
    target_uid  TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX idx_session_trails_sid ON session_trails(session_id, id ASC);

CREATE TABLE retrieval_trails (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    route      TEXT NOT NULL,      -- 'search' | 'lookup' | 'context' | 'timeline'
    query      TEXT NOT NULL,
    filters    TEXT NOT NULL DEFAULT '{}',
    hit_uids   TEXT NOT NULL DEFAULT '[]',
    degraded   INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_retrieval_trails_created ON retrieval_trails(created_at DESC);
```

#### FTS5 Lexical Tables & Corruption-Safe Triggers

```sql
CREATE VIRTUAL TABLE observations_fts USING fts5(
    title,
    content,
    topic_key,
    type,
    tool_name,
    tokenize = 'trigram remove_diacritics 1',
    content = 'observations',
    content_rowid = 'id'
);

CREATE TRIGGER obs_fts_ai AFTER INSERT ON observations WHEN new.deleted_at IS NULL BEGIN
    INSERT INTO observations_fts(rowid, title, content, topic_key, type, tool_name)
    VALUES (new.id, new.title, new.content, COALESCE(new.topic_key, ''), new.type, new.tool_name);
END;

CREATE TRIGGER obs_fts_ad AFTER DELETE ON observations BEGIN
    DELETE FROM observations_fts WHERE rowid = old.id;
END;

CREATE TRIGGER obs_fts_au AFTER UPDATE ON observations BEGIN
    DELETE FROM observations_fts WHERE rowid = old.id;
    INSERT INTO observations_fts(rowid, title, content, topic_key, type, tool_name)
    SELECT new.id, new.title, new.content, COALESCE(new.topic_key, ''), new.type, new.tool_name
    WHERE new.deleted_at IS NULL;
END;

CREATE VIRTUAL TABLE prompts_fts USING fts5(
    content,
    tokenize = 'trigram remove_diacritics 1',
    content = 'user_prompts',
    content_rowid = 'id'
);

CREATE TRIGGER prompts_fts_ai AFTER INSERT ON user_prompts BEGIN
    INSERT INTO prompts_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER prompts_fts_ad AFTER DELETE ON user_prompts BEGIN
    DELETE FROM prompts_fts WHERE rowid = old.id;
END;

CREATE TRIGGER prompts_fts_au AFTER UPDATE ON user_prompts BEGIN
    DELETE FROM prompts_fts WHERE rowid = old.id;
    INSERT INTO prompts_fts(rowid, content) VALUES (new.id, new.content);
END;
```

#### 3-Table Vector Subsystem (`modernc.org/sqlite/vec`)

```sql
CREATE TABLE embedding_meta (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    provider   TEXT NOT NULL,      -- 'onnx' | 'ollama' | 'openai'
    model      TEXT NOT NULL,      -- e.g. 'BAAI/bge-small-en-v1.5'
    dimension  INTEGER NOT NULL CHECK (dimension > 0),
    updated_at TEXT NOT NULL
);

CREATE TABLE observation_embeddings (
    observation_id INTEGER PRIMARY KEY REFERENCES observations(id) ON DELETE CASCADE,
    content_hash   TEXT NOT NULL,  -- SHA-256(title || "\n" || content)
    model          TEXT NOT NULL,
    dimension      INTEGER NOT NULL,
    vector         BLOB NOT NULL,  -- L2-normalized little-endian float32[dimension]
    created_at     TEXT NOT NULL
);

-- Dynamically created/recreated to match embedding_meta.dimension (default 384):
CREATE VIRTUAL TABLE IF NOT EXISTS observations_vec USING vec0(
    observation_id INTEGER PRIMARY KEY,
    embedding      float[384]
);
```

- **Vector Lifecycle Rules**:
  1. `vec0` does not support `UPSERT`; updating or replacing an observation's vector always executes `DELETE FROM observations_vec WHERE observation_id = ?` then `INSERT INTO observations_vec(observation_id, embedding) VALUES (?, vec_f32(?))` inside the writer transaction.
  2. Soft-deleting (`deleted_at IS NOT NULL`) or expiring an observation deletes its row from `observations_vec` and `observation_embeddings`.
  3. On startup or `mnemonic models use`, if `embedding_meta.dimension != configured.dimension`, the store drops and recreates `observations_vec` at `float[<new_dim>]`, updates `embedding_meta`, and marks existing `observation_embeddings` for incremental background re-embedding.

---

## 4. Intelligence Providers, Write Pipeline & Hybrid Retrieval

### 4.1 Intelligence Providers (`platform/intelligence`)

#### Configuration (`~/.skillgrid/mnemonic/config.yaml`)

```yaml
embedding:
  provider: onnx                        # onnx | ollama | openai | none
  model: BAAI/bge-small-en-v1.5         # 384-dim quantized INT8/FP32 ONNX default
  dimension: 384
  endpoint: ""                          # e.g. http://127.0.0.1:11434/v1 for ollama/openai
  api_key_env: ""                       # e.g. OPENAI_API_KEY
  batch_size: 16
  timeout_ms: 3000

llm:
  enabled: false                        # Strictly opt-in
  endpoint: http://127.0.0.1:11434/v1
  model: qwen2.5:7b
  api_key_env: ""
  write_dedup: false                    # Optional LLM 4-way write deduplication
  timeout_ms: 8000
```

#### ONNX CPU Engine

- Implemented via `knights-analytics/hugot` using its **pure-Go `gomlx` CPU backend** by default so `CGO_ENABLED=0 go build` produces a static, cross-platform single binary on macOS (`arm64`/`amd64`) and Linux (`amd64`/`arm64`).
- Supports optional dynamic `purego` `libonnxruntime` acceleration when present on the host without requiring CGo at build time.
- Prefixes asymmetric BGE queries automatically (`"Represent this sentence for searching relevant passages: "` on queries when model family is `bge-*`) and L2-normalizes all output vectors.
- Model weights are stored in `~/.skillgrid/mnemonic/models/<slug>/` with pinned SHA-256 checksums; `mnemonic models pull` downloads explicitly, or the daemon performs a non-blocking background pull on first start while serving FTS5 searches with `degraded: true` until ready.

#### Degradation Circuit Breaker

- `provider == "none"` or model not yet downloaded: `0` retries, immediate deterministic fallback (`degraded: true`).
- Transient HTTP error on `ollama`/`openai`: `1` fast retry (250 ms), then opens a 30-second cooldown circuit so tool latencies never stall an AI agent turn.

### 4.2 Write Pipeline (`mem_save`)

```text
Input -> Validate & Redact <private> -> Check save_operations (Idempotency)
      -> Deterministic Dedup (topic_key Revise / 24h Hash Duplicate)
      -> [Optional] LLM 4-Way Dedup (create | duplicate | revise | supersede)
      -> Conflict Candidate Scan (Lexical >=2 sig terms OR Vector cos >= 0.82, max 3)
      -> Single Writer TXN (observations + versions + relations + save_operations + vec0 + trail)
      -> Envelope {uid, verdict, revision_count, judgment_required, candidates[], degraded}
```

1. **Privacy & Bounds**: Regex replaces `<private>[\s\S]*?</private>` with `[REDACTED]` before hashing, embedding, or storage. Truncates content cleanly at 32 KiB UTF-8 rune boundary.
2. **Idempotency (`operation_id`)**:
   - Matching `operation_id` + matching `fingerprint` -> returns stored `observation_uid` (`replayed: true`).
   - Matching `operation_id` + differing `fingerprint` -> returns `FINGERPRINT_CONFLICT` (`409`).
   - Matching `operation_id` where `observation_id IS NULL` (hard-deleted) -> returns `TOMBSTONE_GONE` (`410`).
3. **Deterministic Dedup Floor**:
   - Active observation with identical `(topic_key, scope)` -> snapshots current `(title, content)` to `observation_versions(version = revision_count)`, updates row in place, increments `revision_count`.
   - Active observation with identical `(normalized_hash, scope, type, title)` within `24h` -> increments `duplicate_count` and updates `last_seen_at` (`verdict: "duplicate"`).
4. **Optional LLM Semantic Dedup** (`llm.enabled && llm.write_dedup`): Evaluates top-10 lexical/vector candidates; permits `duplicate`, `revise`, `supersede`, or `create`. Returned target `uid` must exist in the candidate set or falls back to `create`.
5. **Review Horizon (`review_after`)**: Defaults automatically by `type` (`90d` for `config`/`preference`, `180d` for `discovery`/`pattern`/`decision`/`learning`/`convention`, `365d` for `architecture`/`bugfix`).
6. **Non-Blocking Embedding**: Embeds `(title + "\n" + content)` before opening the SQLite write transaction (never holding the DB write lock during CPU/HTTP inference!). If embedding fails or times out, the observation commits without a vector and the daemon's background backfill worker embeds it asynchronously.

### 4.3 Hybrid Search & Composite Ranking (`mem_search`)

1. **Exact `topic_key` Fast-Path**: If query matches an active `topic_key` exactly, prepends it at synthetic top score.
2. **Lexical Leg**:
   - Tokenizes query; if all terms are $\ge 3$ runes, executes `observations_fts` `MATCH` (`match_mode = "any"` joins with `OR`, `"all"` joins with `AND`), ranked by `bm25(observations_fts, 5.0, 1.0, 3.0, 0.5, 0.5)`.
   - If any term has $< 3$ runes, executes a parameterized `LIKE ... ESCAPE '\'` query across `title`, `topic_key`, and `content` (capped at 16 terms) so short tokens (`go`, `db`, `ui`, `id`) still match reliably.
3. **Vector Leg**: Embeds query once, runs `observations_vec` KNN ($k = 2 \times \text{limit}$, min $20$) against each target DB (`<repo>.db` + `global.db`), falling back to in-Go SIMD-friendly dot-product scan over `observation_embeddings` if `vec0` is rebuilding.
4. **Reciprocal Rank Fusion & Composite Score**:
   $$\text{RRF}(d) = \frac{1}{60 + r_{\text{lex}}(d)} + \frac{1}{60 + r_{\text{vec}}(d)}$$
   $$\text{Score}(d) = \text{RRF}(d) + 0.15 \cdot \mathbb{I}[\text{pinned}] + 0.06 \cdot \text{importance} + 0.04 \cdot \frac{\ln(1 + \text{usage})}{5} + \frac{0.06}{1 + \text{age\_days}/30} + 0.04 \cdot \frac{n_{\text{rev}}}{n_{\text{rev}} + 4} - \text{ConflictPenalty}(d)$$
   where $\text{ConflictPenalty}(d) = 0.20$ if `superseded_by_uid IS NOT NULL`, or $0.10$ if an active `conflicts_with` relation exists.
5. **Post-Search Side Effects**: Async write task increments `retrieval_usage`, updates `importance` EMA, and appends one row to `retrieval_trails` (never blocking the read response). `mem_lookup` does **not** mutate `retrieval_usage`.

### 4.4 Conflict Resolution & Single-Transaction Consolidation

- **`mem_judge` / `mem_judge_many` (1–32 items)**: Keyed by `judgment_id` (`memory_relations.id`) + target DB scope; updates `memory_relations` (`judgment_status = 'judged'`, `relation`, `reason`, `evidence`, `confidence`, `marked_by_model`). When `relation == 'supersedes'`, sets `superseded_by_uid` on the target observation in the same transaction.
- **`mem_compare`**: Deterministically compares (or invokes LLM when `llm.enabled`) two `uid`s in the same DB and records a judged relation idempotently.
- **`mem_consolidate`**:
  - `confirm = false`: Computes `fingerprint = sha256(sorted_uids + titles + contents + revision_counts)` and returns a merge preview (optionally drafted by LLM if `new_content` is omitted and `llm.enabled`).
  - `confirm = true`: Recomputes `fingerprint` inside a **single `BEGIN IMMEDIATE` transaction**; on mismatch returns `409 FINGERPRINT_CONFLICT`. Otherwise inserts the merged observation, sets `deleted_at` + `superseded_by_uid = survivor.uid` on all source observations, removes source vectors from `observations_vec`, and records `session_trails`.

---

## 5. Transport Surfaces: MCP, Daemon, CLI & TUI

### 5.1 MCP Server (`mnemonic memory mcp --profile agent|admin|all`)

Built on `github.com/mark3labs/mcp-go` over stdio. Strict JSON Schema validation rejects unknown arguments (`additionalProperties: false`). Every registered tool is 100% implemented (zero stubs).

| Tier | Loading | Tools |
|---|---|---|
| **Core Agent (12)** | Eager | `mem_session_start`, `mem_session_summary`, `mem_session_end`, `mem_save`, `mem_search`, `mem_get_observation`, `mem_lookup`, `mem_context`, `mem_save_prompt`, `mem_judge`, `mem_judge_many`, `mem_current_project` |
| **Extended Agent (12)** | `WithDeferLoading(true)` | `mem_timeline`, `mem_search_prompts`, `mem_update`, `mem_pin`, `mem_unpin`, `mem_suggest_topic_key`, `mem_capture_passive`, `mem_compare`, `mem_consolidate`, `mem_session_finalize`, `mem_review`, `mem_list_projects` |
| **Admin (4)** (`admin` / `all`) | `WithDeferLoading(true)` | `mem_delete`, `mem_stats`, `mem_doctor`, `mem_backfill_embeddings` |

- **3-Layer Retrieval Contract**:
  - `mem_search`: Returns `uid`, `title`, `type`, `scope`, `project`, `topic_key`, `snippet` (300 runes max), `score`, `pinned`, `relations[]`, `token_estimate`.
  - `mem_timeline`: Chronological neighbor observations (`before`/`after` window around `uid` or within `session_id`).
  - `mem_get_observation`: Full untruncated `content`, metadata, active relations, and `versions[]` when `include_history = true`.
- **`mem_context`**: Returns pinned memories ($\le 20$) + recent session summaries + recent observations within a configurable byte budget (`16 KiB` default, `64 KiB` max), merging `<repo>.db` and `global.db`.
- **`mem_capture_passive`**: Parses `## Key Learnings:` bulleted/numbered items from task output, infers observation `type` (`bugfix`, `decision`, `discovery`, `pattern`, `learning`), and routes each item through `Save`.

### 5.2 Local Daemon (`mnemonic daemon`)

- **Endpoints**:
  - Unix socket: `~/.skillgrid/mnemonic/run/daemon.sock` (`0600` file permissions; OS filesystem permissions enforce local-user auth).
  - Optional TCP HTTP (`daemon.http_enabled: true`, default `127.0.0.1:7438`): Enforces `Authorization: Bearer <token>` matching `~/.skillgrid/mnemonic/run/daemon.token` (`0600`) on all routes except `GET /v1/health` (which returns only `{"status":"ok","version":"..."}`).
- **Routes**:
  - `GET /v1/health`, `GET /v1/capabilities`
  - `POST /v1/memory/sessions/{start|summary|end|finalize}`, `GET /v1/memory/sessions`
  - `POST /v1/memory/observations`, `PATCH /v1/memory/observations/{uid}`, `DELETE /v1/memory/observations/{uid}`, `GET /v1/memory/observations/{uid}`
  - `POST /v1/memory/search`, `POST /v1/memory/lookup`, `GET /v1/memory/context`, `GET /v1/memory/timeline`
  - `POST /v1/memory/prompts`, `POST /v1/memory/prompts/search`, `POST /v1/memory/capture-passive`
  - `POST /v1/memory/relations/{judge|judge-many|compare|consolidate}`, `GET /v1/memory/review`
  - `GET /v1/memory/projects`, `GET /v1/memory/stats`, `POST /v1/memory/backfill`, `GET /v1/doctor`
- **Background Workers**:
  1. **Embedding Backfill Worker**: Runs on startup, on dirty signal, and every `60s` to embed unindexed or model-stale observations in batches of 16.
  2. **Maintenance Worker**: Runs every `15m` to soft-expire rows past `expires_at`, prune `save_operations` older than `30d`, and cap `retrieval_trails` at `10,000` rows per DB.
  3. **Idle Reaper**: When started with `--idle-timeout=30m` (auto-spawn mode), cleanly checkpoints WAL (`PRAGMA wal_checkpoint(TRUNCATE)`) and exits after 30 minutes with zero active requests.

### 5.3 CLI Reference

```text
mnemonic memory save --title "<text>" --type <type> --content "<body|@file|->" [--topic-key <key>] [--scope project|user|global] [--session <id>] [--op-id <id>]
mnemonic memory update <uid> [--title "<text>"] [--content "<body>"] [--type <type>] [--topic-key <key>]
mnemonic memory search "<query>" [--scope project|user|global] [--type <type>] [--match any|all] [--all-projects] [--limit 10] [--json]
mnemonic memory get <uid> [--history] [--json]
mnemonic memory lookup [--uid <uid> | --topic-key <key> | --phrase "<text>"] [--scope project|user|global] [--json]
mnemonic memory context [--budget 16384] [--older] [--json]
mnemonic memory timeline <uid> [--window 5] [--json]
mnemonic memory prompt <save|search> [--session <id>] "<text-or-query>" [--limit 10] [--json]
mnemonic memory capture-passive --session <id> [--file <path> | --stdin]
mnemonic memory pin <uid> | mnemonic memory unpin <uid>
mnemonic memory delete <uid> [--hard]
mnemonic memory review [--limit 20] [--json]
mnemonic memory stats [--all-projects] [--json]
mnemonic memory backfill [--all-projects]
mnemonic memory session <start|summary|end|finalize|list> [--id <id>] [--title "<text>"] [--json]
mnemonic memory conflicts <list|judge|compare|consolidate> [--json]
mnemonic memory mcp [--profile agent|admin|all] [--standalone]

mnemonic daemon <start|stop|status|restart|logs> [--http] [--idle-timeout 30m]
mnemonic models <status|pull|use> [model-name]
mnemonic tui [--project <slug>] [--standalone]
mnemonic doctor [--repair] [--json]
mnemonic version
```

### 5.4 Interactive TUI (`mnemonic tui`)

Built with `charmbracelet/bubbletea`, `bubbles`, and `lipgloss` inside `platform/tuishell`:

1. **Top Tab Bar**: `[1: Memory]` (extensible for future `[2: Code]`, `[3: Wiki]`) + active DB badge (`<repo>.db + global.db` or `global.db`).
2. **Views**:
   - **Dashboard / Search (`/`)**: Live hybrid/lexical query bar, filter chips (`scope`, `type`, `pinned`), compact ranked list showing RRF/BM25/Vec indicators.
   - **Observation Inspector (`Enter`)**: Full markdown content, provenance, `topic_key`, usage/importance metrics, linked relations, and `v` toggle for side-by-side `observation_versions` history diffing + `p` pin/unpin.
   - **Sessions (`s`)**: Chronological session list -> 6-section summary & `session_trails` timeline.
   - **Conflicts Queue (`c`)**: Pending `memory_relations` side-by-side comparison with single-keystroke verdicts (`compatible`, `related`, `supersedes`, `conflicts_with`, `not_conflict`).
   - **Projects (`P`)**: Switch active `<repo>.db` or inspect `global.db` stats.
3. **Status Footer**: Daemon mode (`daemon.sock` vs `standalone`), active embedder (`ONNX bge-small-en-v1.5 [ready]` or `[degraded: lexical-only]`), and pending backfill count.

---

## 6. Error Handling & Resilience Contract

All domain errors carry a canonical `Code` mapped consistently across transports:

| Error Code | CLI Exit | HTTP Status | MCP Envelope | Trigger |
|---|---|---|---|---|
| `VALIDATION_ERROR` | `2` | `400` | `isError: true` | Missing required fields, invalid `type`/`scope`, unknown schema key |
| `UNAUTHORIZED` | `3` | `401` | `isError: true` | Missing/invalid HTTP bearer token |
| `NOT_FOUND` | `4` | `404` | `isError: true` | Observation `uid`, `session_id`, or `relation_id` not found |
| `FINGERPRINT_CONFLICT` | `5` | `409` | `isError: true` | `operation_id` replay mismatch or `mem_consolidate` stale fingerprint |
| `TOMBSTONE_GONE` | `6` | `410` | `isError: true` | `operation_id` refers to a hard-deleted observation |
| `UNSUPPORTED_FS` | `7` | `500` | `isError: true` | Data directory resides on NFS/SMB/CIFS network filesystem |
| `STORAGE_CORRUPT` | `8` | `500` | `isError: true` | SQLite integrity/FTS failure (`mnemonic doctor --repair` rebuilds FTS & `vec0`) |

- **`mnemonic doctor --repair`** performs non-destructive self-healing: runs `PRAGMA integrity_check`, rebuilds `observations_fts` and `prompts_fts` (`INSERT INTO ... VALUES('rebuild')`), drops/rebuilds `observations_vec` from `observation_embeddings` (or schedules re-embedding), cleans stale `daemon.sock`/`daemon.pid`, and verifies file permissions (`0700`/`0600`).

---

## 7. Verification & Testing Strategy

1. **SQLite Store & Concurrency Suite (`internal/platform/sqlitepool`, `internal/memory/store`)**:
   - Parallel 20-goroutine writer + 20-goroutine WAL reader stress test asserting zero `SQLITE_BUSY` errors.
   - External file unlink/recreate mid-flight verifying inode generation fence auto-recovery.
   - Reserved slug collision test (`global` -> `repo-global.db`).
   - FTS5 trigram search + `<3`-rune `LIKE` fallback + soft-delete trigger removal test.
   - `vec0` KNN search, soft-delete exclusion, and 384-dim -> 768-dim model-swap `observations_vec` rebuild test.
2. **Shared Service Parity Suite (`internal/memory/api/compliance`)**:
   - Parameterized contract suite covering idempotency (`200`/`409`/`410`), `<private>` redaction, `topic_key` revisions + `observation_versions`, 24h hash dedup, RRF hybrid ranking across `<repo>.db` + `global.db`, conflict surfacing + `mem_judge_many`, and atomic single-tx `mem_consolidate`.
   - Executed identically against **both** `engine.Local` (standalone) and `client.DaemonClient` (over Unix socket and bearer-authed HTTP).
3. **MCP Protocol & Progressive Disclosure Suite (`internal/memory/mcp`)**:
   - Verifies eager vs `WithDeferLoading(true)` tool metadata, `--profile agent` vs `admin` filtering, `additionalProperties: false` rejection, and end-to-end stdio JSON-RPC session flows.
4. **ONNX CPU & Degradation Integration Suite (`internal/platform/intelligence`)**:
   - Deterministic mock embedder/LLM unit tests + circuit-breaker timeout tests verifying writes succeed and searches return `degraded: true` when offline.
   - Live quantized ONNX CPU (`bge-small-en-v1.5`) test verifying semantic recall on paraphrased queries with zero lexical overlap (skippable under `go test -short` if model cache is absent).
5. **TUI Model Tests (`internal/memory/tui`)**:
   - Headless Bubbletea state-transition tests (`teatest` / `Update` message sequences) for search, version diff toggle, project switching, and conflict verdict submission.
