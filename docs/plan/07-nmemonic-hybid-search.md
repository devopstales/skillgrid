# Mnemonic Enhancement Proposal: Hybrid Code Intelligence Platform

## Goal

Transform Mnemonic from a "better grep" (fixed 80-line chunks + trigram FTS) into a full-featured code intelligence platform, integrating:

1. Identifier-aware FTS5 (camelCase/snake_case tokenization)
2. Call graph with symbols + qualified names + rich relation taxonomy
3. Hybrid search: FTS5 + deterministic similarity signals + optional dense embeddings
4. Code analysis: community detection, dead code, git-diff impact
5. 42 MCP tools across 7 tiers (matching srclight's comprehensive surface)
6. 30+ languages via tree-sitter (per-language extractor + per-file fallback)
7. Pluggable embedding providers: bundled (nomic-embed-code), Ollama, external

All layers sit on top of Mnemonic's existing per-project SQLite store, CGo-free (`modernc.org/sqlite`), incremental content-hash syncing, and MCP server.

---

## 1. Reference Systems & What We Learn From Each

| System | Key Lesson |
|---|---|
| **srclight** (42 tools, 11 langs) | 42 MCP tools across 7 tiers; hybrid RRF search; strict argument validation; multi-repo workspaces; index freshness stamps; document extraction (10 formats) |
| **codegraph** (Rust, 20+ langs) | Rust kernel for 20+ languages; byte-for-byte identical graphs; per-file fallback; adaptive worker sizing; file watcher (FSEvents/inotify); surgical context |
| **codebase-memory-mcp** (C, 158 langs) | 158 languages; 11-signal hybrid semantic scorer; bundled nomic-embed-code (no API key); camel/snake FTS; Louvain community detection; Cypher-like queries; git-diff impact; dead code; cross-service HTTP linking |

---

## 2. Mnemonic SQLite Schema (Additive — all new tables)

### 2.1 Symbols (qualified-name key, per-language extractor)

```sql
CREATE TABLE symbols (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    qualified_name TEXT NOT NULL,          -- canonical, disambiguated identity
    name TEXT NOT NULL,                    -- short name (for display)
    kind TEXT NOT NULL,                    -- function, method, class, interface, enum, type, constant, route, module, file
    start_line INTEGER,
    end_line INTEGER,
    signature TEXT,                        -- function signature (lightweight)
    profile BLOB,                          -- per-symbol computed signals (see §4)
    content_hash TEXT,                     -- for freshness
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_symbols_path ON symbols(path);
CREATE INDEX idx_symbols_qualified_name ON symbols(qualified_name);
CREATE INDEX idx_symbols_kind ON symbols(kind);
```

### 2.2 Edges (rich relation taxonomy + confidence labels)

```sql
CREATE TABLE edges (
    id TEXT PRIMARY KEY,
    src_symbol TEXT NOT NULL,
    dst_symbol TEXT NOT NULL,
    relation TEXT NOT NULL,                -- CALLS, CALL_REFERENCE, USAGE, IMPORTS,
                                           -- DEFINES, IMPLEMENTS, INHERITS, DEFINED_IN,
                                           -- HANDLES, DATA_FLOWS, EMITS, LISTENS_ON,
                                           -- SIMILAR_TO, SEMANTICALLY_RELATED, MEMBER_OF
    confidence TEXT NOT NULL,              -- EXTRACTED | INFERRED | AMBIGUOUS
    properties BLOB,                       -- relation-specific data (arg→param mapping, etc.)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_edges_src ON edges(src_symbol);
CREATE INDEX idx_edges_dst ON edges(dst_symbol);
CREATE INDEX idx_edges_relation ON edges(relation);
```

### 2.3 Identifier-Aware FTS (camelCase/snake_case tokenizer)

```sql
-- Custom FTS5 tokenizer with cbm_camel_split behaviour
CREATE VIRTUAL TABLE symbol_fts USING fts5(
    name,
    qualified_name,
    signature,
    tokenize = 'unicode61',
    content = symbols
);

-- Trigram FTS for code search (existing, augmented with identifier tokens)
CREATE VIRTUAL TABLE code_fts USING fts5(
    content,
    tokenize = 'trigram',
    content = chunks
);
```

### 2.4 Semantic Vectors (pluggable embedding providers)

```sql
CREATE TABLE embeddings (
    id TEXT PRIMARY KEY,
    unit_id TEXT NOT NULL,                 -- symbol_id or chunk_id
    unit_type TEXT NOT NULL,               -- 'symbol' | 'chunk'
    model TEXT NOT NULL,                   -- e.g., 'nomic-embed-code', 'ollama/nomic-embed-text', 'openai/text-embedding-3-small'
    dimension INTEGER NOT NULL,
    vector BLOB NOT NULL,                  -- float32 or int8 array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_embeddings_unit ON embeddings(unit_id, unit_type);
CREATE INDEX idx_embeddings_model ON embeddings(model);

-- Embedding provider metadata (self-describing, drift-safe)
CREATE TABLE embed_meta (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,                -- 'bundled' | 'ollama' | 'openai' | 'gemini' | 'bedrock'
    model TEXT NOT NULL,
    dimension INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2.5 MinHash LSH (near-clone detection, deterministic)

```sql
CREATE TABLE lsh_buckets (
    symbol_id TEXT NOT NULL,
    bucket INTEGER NOT NULL,
    FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);
CREATE INDEX idx_lsh_buckets ON lsh_buckets(bucket);
```

### 2.6 Community Detection (Louvain/Leiden)

```sql
CREATE TABLE communities (
    symbol_id TEXT NOT NULL,
    cluster INTEGER NOT NULL,
    FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);
CREATE INDEX idx_communities_cluster ON communities(cluster);
```

### 2.7 Freshness & Observability

```sql
-- Staleness tracking (existing content_hash + mtime_ns extended)
CREATE TABLE index_freshness (
    path TEXT PRIMARY KEY,
    content_hash TEXT NOT NULL,
    mtime_ns INTEGER NOT NULL,
    indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    stale BOOLEAN DEFAULT 0
);

-- Retrieval trails (observability)
CREATE TABLE retrieval_trails (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    query TEXT NOT NULL,
    tool TEXT NOT NULL,
    params TEXT,                           -- JSON
    results_count INTEGER,
    latency_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 3. Per-Language Extractor Interface (30+ languages via tree-sitter)

### 3.1 Extractor Interface (Go)

```go
type Extractor interface {
    // Extract symbols and edges from a single file
    Extract(path string, content []byte) (*FileGraph, error)

    // Language name (for logging/dispatch)
    Language() string

    // File patterns to match
    Patterns() []string
}

type FileGraph struct {
    Symbols []Symbol
    Edges   []Edge
}

type Symbol struct {
    Name           string
    QualifiedName  string   // e.g., "github.com/user/repo/pkg.Function"
    Kind           string   // function, method, class, interface, enum, type, constant, route, module
    StartLine      int
    EndLine        int
    Signature      string
    Profile        []byte   // computed signals (see §4)
}

type Edge struct {
    Src        string   // qualified_name
    Dst        string   // qualified_name
    Relation   string   // CALLS, IMPORTS, INHERITS, etc.
    Confidence string   // EXTRACTED | INFERRED | AMBIGUOUS
    Properties []byte   // optional relation-specific data
}
```

### 3.2 Language Support (30+ languages, Tier 1 rollout)

| Tier | Languages | Status |
|---|---|---|
| **Tier 1 (MVP)** | Go, TS/TSX | Full extractor + resolver, per-file fallback |
| **Tier 2** | Python, Java, C#, PHP, JavaScript | Extractors + reference resolution |
| **Tier 3** | C, C++, Rust, Kotlin, Swift, Dart, Ruby, Scala | Extractors only (basic resolution) |
| **Tier 4** | Lua, R, Erlang, Solidity, Terraform, Svelte, Vue, HTML, CSS, Markdown, YAML, TOML, JSON, Protobuf | Community contributions welcome |

### 3.3 Per-File Fallback (codegraph pattern)

- If a file fails to parse (syntax errors, unsupported features, etc.):
  - Log the error (WARN level)
  - Fall back to regex-based extraction for that file
  - Continue indexing (never abort the entire run)
- This ensures one malformed file never breaks the whole index

### 3.4 Incremental Indexing (existing content_hash + mtime_ns guard)

- Scan files, skip those with unchanged `content_hash` + `mtime_ns`
- Re-extract only changed files
- Prune symbols/edges owned by deleted files
- Parallel extraction + in-memory buffer → single write transaction

---

## 4. Hybrid Search: FTS5 + Deterministic Signals + Dense Vectors

### 4.1 Deterministic Similarity Signals (no backend required)

Per symbol, at index time, compute and store in `symbols.profile`:

| Signal | Description |
|---|---|
| MinHash | Fingerprint + LSH buckets for near-clone detection |
| AST profile | Control-flow shape, expression mix, literal density |
| Approx data-flow | Parameter→return / parameter→condition vectors |
| Halstead-lite | Operator/operand metrics (volume, difficulty, effort) |
| TF-IDF (local) | Term frequencies within the symbol |
| API/Type sig | API endpoint / type / decorator signature vectors |

These provide "find similar code" functionality even with no embedding backend.

### 4.2 Dense Embeddings (pluggable, tiered)

| Provider | Description |
|---|---|
| **BUNDLED** (default) | nomic-embed-code (768d int8) compiled into the binary — ~40K tokens, no API key, no Ollama, no Docker |
| **OLLAMA** | Local models: nomic-embed-text, qwen3-embedding, etc. Configurable via `OLLAMA_BASE_URL` env var |
| **EXTERNAL** | OpenAI (text-embedding-3-small/large), Gemini, Bedrock. Configurable via API keys + base URLs |

- Embeddings stored as compact int8 BLOB with dimension metadata
- Model/dimension switch → auto-triggers vector re-index (graceful, non-destructive)
- In-process top-k cosine scan per project (sqlite-vec optional for large corpora)

### 4.3 Fusion = Multi-Signal Scoring (11 signals, codebase-memory-mcp pattern)

```text
search_score = w1 * BM25(fts) + w2 * cosine(embedding) + w3 * minhash_sim +
               w4 * ast_sim + w5 * dataflow_sim + w6 * halstead_sim +
               w7 * tfidf_sim + w8 * api_sig_sim + w9 * module_proximity +
               w10 * graph_diffusion(neighbour_score)
```

- Default weights: sensible defaults (RRF for v1, tuned weights for v2)
- Per-hit provenance: each signal's sub-score returned for transparency
- Graph diffusion: blend a node's score with its neighbours' scores
- Module proximity: boost code in the same directory/package

---

## 5. MCP Tools (42 tools across 7 tiers — matching srclight's surface)

### 5.1 Tier 1: Instant Orientation (6 tools)

| Tool | What it does |
|---|---|
| `codebase_map()` | Full project overview — call first every session |
| `search_symbols(q)` | Search across symbol names, code, and docs |
| `get_symbol(name)` | Full source code + metadata for a symbol |
| `get_signature(name)` | Just the signature (lightweight) |
| `symbols_in_file(path)` | Table of contents for a file |
| `list_projects()` | All projects in workspace with stats |

### 5.2 Tier 2: Relationship Graph (6 tools)

| Tool | What it does |
|---|---|
| `get_callers(name)` | Who calls this symbol? |
| `get_callees(name)` | What does this symbol call? |
| `get_dependents(name, transitive)` | Blast radius — what breaks if I change this? |
| `get_implementors(interface)` | All classes implementing an interface |
| `get_tests_for(name)` | Test functions covering a symbol |
| `get_type_hierarchy(name)` | Inheritance tree (base classes + subclasses) |

### 5.2b Tier 2b: Community & Impact Analysis (5 tools)

| Tool | What it does |
|---|---|
| `get_communities(project)` | Auto-detected functional module clusters (Louvain algorithm) |
| `get_community(name, project)` | Which community a symbol belongs to, with all co-members |
| `get_execution_flows(project)` | Traced execution paths from entry points through the call graph |
| `get_impact(name, project)` | Blast radius + risk level (LOW/MEDIUM/HIGH/CRITICAL) |
| `detect_changes(project, ref?)` | Map git diff to affected symbols — aggregate blast radius of your edits |

### 5.3 Tier 3: Git Change Intelligence (5 tools)

| Tool | What it does |
|---|---|
| `blame_symbol(name)` | Who changed this, when, and why |
| `recent_changes(n)` | Commit feed (cross-project in workspace) |
| `git_hotspots(n, since)` | Most frequently changed files (bug magnets) |
| `whats_changed()` | Uncommitted work in progress |
| `changes_to(name)` | Commit history for a symbol's file |

### 5.4 Tier 4: Build & Config (3 tools)

| Tool | What it does |
|---|---|
| `get_build_targets()` | CMake/.csproj/npm targets with dependencies |
| `get_platform_variants(name)` | `#ifdef` platform guards around a symbol |
| `platform_conditionals()` | All platform-conditional code blocks |

### 5.5 Tier 5: Semantic Search (3 tools)

| Tool | What it does |
|---|---|
| `semantic_search(q)` | Find code by meaning (natural language) |
| `hybrid_search(q)` | Best of both: keyword + semantic with RRF fusion |
| `embedding_status()` | Embedding coverage and model info |

### 5.6 Tier 6: Meta & Server (6 tools)

| Tool | What it does |
|---|---|
| `index_status()` | Index freshness and stats |
| `reindex()` | Trigger incremental re-index |
| `embedding_health()` | Check if the embedding provider is reachable |
| `setup_guide()` | Structured setup instructions for agents/users |
| `server_stats()` | Server uptime and process info |
| `restart_server()` | Request server restart (SSE only) |

### 5.7 Tier 7: Document Extraction (8 tools)

| Tool | What it does |
|---|---|
| `extract_pdf(path)` | Extract text, headings, tables from PDF |
| `extract_docx(path)` | Extract text, headings, tables from DOCX |
| `extract_xlsx(path)` | Extract tables, sheets from XLSX |
| `extract_html(path)` | Extract text, headings from HTML |
| `extract_csv(path)` | Extract rows, columns from CSV/TSV |
| `extract_email(path)` | Extract subject, body, attachments from `.eml` |
| `extract_image(path)` | OCR text from image (tesseract optional) |
| `extract_text(path)` | Extract plain text with heading detection |

---

## 6. Index Freshness & Auto-Sync

### 6.1 Staleness Banner (srclight pattern)

Every symbol/graph result carries `index_freshness`:

- `"verified-fresh"` — files behind the answer are byte-identical to indexed
- `{"stale": ["path1", "path2"]}` — list of stale files (agent should re-read)

`check_freshness(paths?)` probes any paths on demand (unchanged files cost one stat).
`index_status` reports whole-index `checked`/`stale_count`.

### 6.2 Debounced Watcher (codegraph pattern)

- `skillgrid serve` spawns a `fsnotify` watcher
- Native OS events: FSEvents (macOS) / inotify (Linux) / ReadDirectoryChangesW (Windows)
- Debounce: 300ms after a file save → auto-reindex changed files
- Incremental: only re-extract changed files; update symbols/edges atomically

### 6.3 Shareable Graph Artifact (optional, Tier 3)

- Compact, `VACUUM INTO`-ed, zstd-compressed export: `graph.sqlite.zst`
- `merge=ours` gitattributes line for team sharing
- Two-tier: fast for watcher writes, best for explicit `skillgrid export-graph`
- Bootstrap-import + incremental-fill on fresh clone (mirrors codegraph's team artifact)

---

## 7. Implementation Roadmap (12 weeks)

### Week 1-2 — Foundation & Schema

- [ ] Enable tree-sitter-Go bindings + tree-sitter-ts bindings
- [ ] Run all SQL migrations (symbols, edges, FTS, embeddings, communities, trails)
- [ ] Write extractor interface + Go extractor (Tier 1)
- [ ] Write TS/TSX extractor (Tier 1)
- [ ] Write incremental indexer (content_hash + mtime_ns guard)
- [ ] Write parallel extraction + in-memory buffer → single write tx

### Week 3-4 — Identifier-Aware FTS + Basic Search

- [ ] Implement camelCase/snake_case tokenizer (cbm_camel_split behaviour)
- [ ] Implement `code_fts` with identifier tokens
- [ ] Implement `search_symbols` (FTS-only v1)
- [ ] Implement `symbols_in_file`, `list_projects`
- [ ] Implement `get_symbol`, `get_signature`

### Week 5-6 — Call Graph + Traversal Tools

- [ ] Implement edge resolution (import-aware, type-inferred)
- [ ] Implement `get_callers`, `get_callees`
- [ ] Implement `get_dependents` (transitive closure)
- [ ] Implement `get_implementors`, `get_type_hierarchy`
- [ ] Implement `get_tests_for`
- [ ] Add confidence labels: `EXTRACTED | INFERRED | AMBIGUOUS`

### Week 7-8 — Hybrid Search (Deterministic Signals + Embeddings)

- [ ] Implement deterministic signals: MinHash+LSH, AST profile, data-flow, Halstead-lite
- [ ] Implement bundled embedding provider (nomic-embed-code) via ONNX runtime
- [ ] Implement Ollama embedding provider (OLLAMA_BASE_URL)
- [ ] Implement external embedding provider (OpenAI/Gemini/Bedrock)
- [ ] Implement `hybrid_search` (RRF fusion v1)
- [ ] Implement `semantic_search`
- [ ] Add `embedding_status`, `embedding_health`

### Week 9 — Community Detection + Analysis

- [ ] Implement Louvain/Leiden community detection over call edges
- [ ] Implement `get_communities`, `get_community`
- [ ] Implement `get_execution_flows`
- [ ] Implement `get_impact` (blast radius + risk classification)
- [ ] Implement `detect_changes` (git diff → affected symbols)
- [ ] Implement dead code detection (`get_dead_code`)

### Week 10 — Git Intelligence + Build/Config

- [ ] Implement `blame_symbol` (git blame integration)
- [ ] Implement `recent_changes` (commit feed)
- [ ] Implement `git_hotspots` (most frequently changed files)
- [ ] Implement `whats_changed` (uncommitted work)
- [ ] Implement `changes_to` (commit history)
- [ ] Implement `get_build_targets` (CMake/npm/.csproj)

### Week 11 — Document Extraction + Freshness

- [ ] Implement document extractors: PDF (pdfium), DOCX, XLSX, HTML, CSV, email, images
- [ ] Implement OCR support: PaddleOCR (scanned PDFs), tesseract (images)
- [ ] Implement `check_freshness`, `index_status`
- [ ] Implement debounced fsnotify watcher
- [ ] Implement staleness banners on all search/read/traversal tools

### Week 12 — CLI, Observability & Testing

- [ ] Build all CLI commands (42 tools surface)
- [ ] Implement `retrieval_trails` logging for all tools
- [ ] Implement strict argument validation (srclight pattern: `additionalProperties: false`)
- [ ] End-to-end test: full workflow across all 7 tiers
- [ ] Write documentation + agent guidance (tool selection guide, session protocol)

---

## 8. CLI Commands (expanded from existing skillgrid CLI)

```bash
# Tier 1: Orientation
skillgrid codebase-map
skillgrid search-symbols "main" --kind function --limit 10
skillgrid get-symbol "RefreshSessions"
skillgrid get-signature "RefreshSessions"
skillgrid symbols-in-file src/main.go
skillgrid list-projects

# Tier 2: Graph
skillgrid get-callers "RefreshSessions" --depth 2
skillgrid get-callees "RefreshSessions"
skillgrid get-dependents "RefreshSessions" --transitive
skillgrid get-implementors "Authenticator"
skillgrid get-tests-for "RefreshSessions"
skillgrid get-type-hierarchy "Authenticator"

# Tier 2b: Community & Impact
skillgrid get-communities --project myproject
skillgrid get-community "RefreshSessions" --project myproject
skillgrid get-execution-flows --project myproject
skillgrid get-impact "RefreshSessions" --project myproject
skillgrid detect-changes --project myproject

# Tier 3: Git
skillgrid blame-symbol "RefreshSessions"
skillgrid recent-changes 10
skillgrid git-hotspots 20 --since "2 weeks ago"
skillgrid whats-changed
skillgrid changes-to "RefreshSessions"

# Tier 4: Build
skillgrid get-build-targets
skillgrid get-platform-variants "RefreshSessions"
skillgrid platform-conditionals

# Tier 5: Semantic
skillgrid semantic-search "dictionary lookup"
skillgrid hybrid-search "authenticate user"
skillgrid embedding-status

# Tier 6: Meta
skillgrid index-status
skillgrid reindex
skillgrid embedding-health
skillgrid setup-guide
skillgrid server-stats

# Tier 7: Documents
skillgrid extract-pdf document.pdf
skillgrid extract-docx document.docx
skillgrid extract-xlsx data.xlsx
skillgrid extract-html page.html
skillgrid extract-csv data.csv
skillgrid extract-email message.eml
skillgrid extract-image screenshot.png
skillgrid extract-text README.md
```

---

## 9. Key Benefits

- [x] **42 MCP tools across 7 tiers** — comprehensive code intelligence surface
- [x] **30+ languages** via tree-sitter (per-language extractor + per-file fallback)
- [x] **Identifier-aware FTS5** (camelCase/snake_case tokenization)
- [x] **Call graph** with symbols + qualified names + rich relation taxonomy
- [x] **Confidence labels:** `EXTRACTED | INFERRED | AMBIGUOUS` (trust transparency)
- [x] **Hybrid search:** FTS5 + 11 deterministic signals + dense embeddings
- [x] **Pluggable embedding providers:** bundled (nomic-embed-code), Ollama, external
- [x] **Community detection:** Louvain/Leiden → `MEMBER_OF` edges
- [x] **Dead code detection:** zero callers (excluding entry points)
- [x] **Git-diff impact mapping:** uncommitted changes → affected symbols + risk
- [x] **Index freshness:** staleness banners on every result
- [x] **Debounced file watcher:** auto-sync on save (FSEvents/inotify)
- [x] **Strict argument validation:** unknown args rejected with error
- [x] **Incremental indexing:** `content_hash` + `mtime_ns` guard
- [x] **No external dependencies for core graph:** tree-sitter + SQLite only
- [x] **Git-friendly:** all L2 documents are plain text (commit to repo)
- [x] **Team sharing:** optional compressed graph artifact (`.zst`, `merge=ours`)

---

## 10. Architecture Diagram

```text
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              MCP CLIENTS                                        │
│     (Claude Code, Cursor, OpenCode, skillgrid CLI, etc.)                        │
└─────────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         Mnemonic MCP SERVER (Go)                                │
│                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────────┐
│  │  42 MCP TOOLS (7 Tiers)                                                       │
│  │  T1: Orientation (6)   │ T2: Graph (6)     │ T2b: Community/Impact (5)        │
│  │  T3: Git (5)           │ T4: Build/Config(3) │ T5: Semantic (3)               │
│  │  T6: Meta/Server (6)   │ T7: Documents (8)  │                                 │
│  └───────────────────────────────────────────────────────────────────────────────┘
│                                      │                                          │
│  ┌───────────────────────────────────┼───────────────────────────────────────────┐
│  │  ┌─────────────────┐  ┌───────────┴──┐  ┌──────────────────────────────────┐  │
│  │  │ Extractor       │  │ Resolver     │  │  Hybrid Search Engine             │  │
│  │  │ Interface       │  │ (import-     │  │  - FTS5 (identifier-aware)        │  │
│  │  │ (30+ langs)     │  │  aware)      │  │  - 11 deterministic signals       │  │
│  │  │ - tree-sitter   │  │ - type-      │  │  - Dense embeddings (pluggable)   │  │
│  │  │ - per-file       │  │   inferred   │  │  - Graph diffusion + module prox  │  │
│  │  │   fallback       │  │ - confidence │  │  - RRF fusion                     │  │
│  │  └─────────────────┘  │   labels     │  └──────────────────────────────────┘  │
│  │                       └──────────────┘  │                                     │
│  │  ┌───────────────────────────────────────────────────────────────────────────┐ │
│  │  │  Analysis Engine                                                          │ │
│  │  │  - Louvain/Leiden community detection  - Dead code detection              │ │
│  │  │  - Git-diff impact mapping             - Execution flows                  │ │
│  │  └───────────────────────────────────────────────────────────────────────────┘ │
│  │  ┌───────────────────────────────────────────────────────────────────────────┐ │
│  │  │  Freshness & Watcher                                                      │ │
│  │  │  - Staleness banners (every result)  - fsnotify (FSEvents/inotify)        │ │
│  │  │  - Debounced auto-reindex            - content_hash + mtime_ns            │ │
│  │  └───────────────────────────────────────────────────────────────────────────┘ │
│  └─────────────────────────────────────────────────────────────────────────────────┘
└─────────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    skillgrid.db (SQLite - Mnemonic)                             │
│                                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ symbols  │  │ edges    │  │ symbol_  │  │ code_fts │  │ embeddings       │  │
│  │          │  │          │  │ fts      │  │ (trigram)│  │ (BLOB vectors)   │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ lsh_     │  │ commun-  │  │ index_   │  │retrieval │  │ embed_meta       │  │
│  │ buckets  │  │ ities    │  │ freshness│  │ trails   │  │                  │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    Filesystem ($PROJECT/.skillgrid/files/)                      │
│                                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  ┌──────────────────┐ │
│  │ tasks/   │  │ memories/│  │ skills/  │  │ knowledge/ │  │ graph.sqlite.zst │ │
│  │ (L2)     │  │ (L2)     │  │ (L2)     │  │ (L2)       │  │ (optional share) │ │
│  └──────────┘  └──────────┘  └──────────┘  └────────────┘  └──────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 11. Conclusion

This proposal upgrades Mnemonic from a "better grep" to a complete code intelligence platform, matching srclight's 42-tool surface while incorporating the best lessons from codegraph (Rust kernel, per-file fallback, watcher) and codebase-memory-mcp (158 languages, 11-signal hybrid scorer, bundled embeddings).

The implementation is incremental and additive:

- All new tables are additive (no schema breaks)
- Existing `code_search` keeps its name/signature (additive fields only)
- New tools are registered alongside existing ones
- Per-language extractors can be added gradually (Tier 1 → Tier 4)
- Embedding providers are pluggable (bundled default, Ollama, external)

The result is a 100% local, dependency-light, CGo-free code intelligence system that agents can query with 42 precise tools instead of dozens of grep/read cycles.
