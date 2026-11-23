# mnemonic — brainstorm (pre-decision record)

**Local-first persistent memory for AI agents.** One Go binary, one SQLite file per project,
two front doors: **CLI** (humans/ops) and **MCP stdio** (agents). No server, no cloud, no
external vector database.

```
                    ┌────────────────────────────────────────────┐
                    │              adapters (1:1 seam)           │
                    │  ┌──────────────┐        ┌───────────────┐  │
                    │  │ MCP stdio    │        │ CLI (flags,   │  │
                    │  │ (~12 tools)  │        │  JSON out)    │  │
                    │  └──────┬───────┘        └───────┬───────┘  │
                    │         └───────────┬────────────┘          │
                    │             Handler (one shared brain)      │
                    ├─────────────────────┼───────────────────────┤
                    │  domains            │                       │
                    │  ┌─────────┐ ┌───────────┐ ┌─────────────┐  │
                    │  │ memory  │ │ codeindex │ │ webcache    │  │
                    │  │ observe │ │ chunk/sym │ │ TTL'd snaps │  │
                    │  │ recall  │ │ + edges   │ │             │  │
                    │  └────┬────┘ └─────┬─────┘ └──────┬──────┘  │
                    │       └────────────┼──────────────┘         │
                    │              Store (SQLite + FTS5 + vectors)│
                    └────────────────────────────────────────────┘
```

## 1. What it is / is not

**Is**: an append-oriented observation store (decisions, bugfixes, discoveries, conventions,
preferences) with sessions, hybrid recall (BM25 FTS5 + cosine vectors, RRF-fused), an
incremental code index, and a TTL'd web cache — all in one local file, queryable by a CLI and
an MCP server that share a single handler.

**Is not**: a RAG server, a chat DB, a Postgres, a vector database service, a team collab
layer, or a cloud. v1 has no network calls by default (embedder is local; API embedder is
opt-in).

## 2. What the predecessors teach

| Source | Take |
|---|---|
| `../engram` | Single-binary, zero-deps, agent-agnostic setup (`mcp` + `setup` plugins), MCP + daemon split, identity file, doctor/selftest. Its cloud tier is the part we deliberately drop. |
| `../skillgrid/mnemonic` | The data model is already proven: `sessions`/`observations` (topic_key upsert, normalized_hash dedup, revision_count, importance, scope, superseded_by), FTS5, `chunks`/`symbols`/`edges` code index, web cache, retrieval trails. Its vector layer decision (BLOB source of truth → HNSW → vec0 → brute-force, dim-strict, hash-64 floor) is sound. Its scope creep (Postgres, projectwiki, team_*, SDD coupling, ~30 code_* tools) is the thing to cut. |

**Inherit**: observations model, topic_key upsert, hash-64 floor embedder, dim-strict vectors,
degradation contract, single-handler K6 seam, project detection chain, exit codes 0/1/2.

**Cut**: Postgres + `migrations_pg`, projectwiki, `team_*` tools, SDD artifact naming, cloud
inbox/lease, dashboard viz, 30-tool code graph surface (shrink to ~5).

## 3. Storage decision: SQLite + FTS5 + vectors-in-file

The user requirement is "sqlite and vector database". The question is where the vectors live.

| Option | Verdict | Why |
|---|---|---|
| External vector DB (Qdrant/Chroma/Lance) | ✗ | Separate process + network/IPC + lifecycle. Overkill at 10³–10⁵ vectors. Violates single-binary/local-first. |
| `sqlite-vec` (vec0) via CGo (mattn + LoadExtension) | ~ | Best SQL-native ANN. But loadable C extension breaks pure-Go build; needs `.so` next to binary. |
| `sqlite-vec` pure-Go port (`asg017/sqlite-vec-go` over modernc) | ✓ default | vec0 virtual tables inside the same DB file, zero CGo. Verify API parity (dim, cosine, multi-table) in a spike before committing. |
| BLOB + Go brute-force | ✓ fallback floor | Exact, zero deps, trivially correct. 10⁵ × 384-dim cosine ≈ a few ms per query — fine at agent-memory scale. Always available regardless of which vec path is active. |

**Decision**: one SQLite file, three layers in it:

1. Plain tables = source of truth (vectors stored as `embedding BLOB` + `embedding_model` + `embedding_dim`).
2. `vec0` virtual tables (`mem_vec`, `chunk_vec`, `web_vec`) = ANN index, created lazily once the
   active embedder dim is known and stable; dropped/rebuilt on dim change (a dim change is a
   re-embed, never a silent truncate).
3. Brute-force scan = guaranteed path when vec tables are absent, dim-locked, or the vec
   extension fails to load. Search always runs both legs when available and fuses (RRF) with FTS5;
   when the vector leg is absent the result carries `degraded: true` — never silent.

SQLite build: `modernc.org/sqlite` (pure Go) → static binary, no CGo, no system libs.
WAL mode, `busy_timeout=5000`, `PRAGMA synchronous=NORMAL`. Multi-agent concurrency is
local-scale; WAL + busy timeout is the whole story (no leases in v1).

## 4. Embedding: three tiers, model-stamped

```
hash-64   → deterministic, offline, zero-dep floor. 64-dim token hash bag.
            Lets the vector leg + RRF + trails exist before any model is present.
            Degraded=true until upgraded.
local     → small ONNX sentence-embedder (e.g. MiniLM-class, 384-dim), model file
            under ~/.mnemonic/models/. Offline.
api       → OpenAI-compatible /embeddings (MNEMONIC_API_BASE/_MODEL/_KEY).
```

- `Embedder` interface: `Embed(ctx, texts) ([][]float32, error)`, `Model()`, `Dimensions()`.
  Nil = absent (leg skipped, `degraded`), erroring = one bounded retry then floor — never block
  a save on embeddings.
- Every vector row carries `model` + `dim`; mixing models in one table is a re-embed, enforced
  dim-strict.
- Tier selection: local when available → API when configured → hash floor otherwise.

## 5. Data model (single file, additive migrations, `schema_version` in `meta`)

```sql
-- memory domain
sessions(id, project, title, dir, started_at, ended_at, summary)
observations(
  id PK, session_id FK, type,          -- decision|bugfix|discovery|pattern|
                                       -- preference|convention|config|learning|architecture
  title, content, scope,               -- project|user|global
  topic_key,                           -- upsert key (same topic_key+scope ⇒ UPDATE, revision++)
  normalized_hash,                     -- dedup floor (content-hash, LLM-adjudicated later)
  revision_count, duplicate_count, last_seen_at,
  pinned, superseded_by, review_after, expires_at,
  importance REAL DEFAULT 0.5, retrieval_usage INTEGER,
  embedding BLOB, embedding_model, embedding_dim, embedding_at
)
obs_fts   -- FTS5 external-content over observations(title, content)
obs_links(observations id↔id, kind)    -- supports|contradicts|relates  (typed edges, sparse)
mem_vec   -- vec0 (rowid→observations.id)

-- ops
meta(key PK, value)                    -- schema_version, created_at, embedder state
trails(id, at, query, route, legs, hits_json, scope, project)  -- retrieval audit
```

`project` column exists in every domain table even inside a per-project file (cheap, makes
`--all` and export semantics uniform).

(Code-domain tables — files/chunks/symbols/edges + their FTS/vec — ship in a later tool; the
schema is designed to admit them without migration pain.)

## 6. Recall pipeline

```
query → scope filter (project|user|global|all)
     → legs:  FTS5 BM25          (always)
              vec0 KNN / brute   (when vectors exist for the active dim)
     → RRF fuse → boosts: recency, importance, pinned
     → top-K, each hit stamped: id, title, snippet, source, leg provenance
     → trail row written (audit + later eval)
```

- Previews truncated (~200 chars); full body only via `mem_get(id)` — context-budget discipline.
- Provenance is explicit: `route` (which leg(s) matched) + `degraded` (vector leg absent).

## 7. Interfaces — one handler, two adapters

**K6 seam (inherited)**: CLI and MCP call the *same* `Handler` methods and render the *same*
JSON envelope; parity tests assert byte-identical payloads. Neither adapter may contain logic.

Envelope: `{"ok":true,"code":"ok","data":{...},"degraded":false,"elapsed_ms":n}` — error
carries `code` (stable string) + message. Exit codes: 0 ok · 1 handler error · 2 usage.

### MCP tools (v1: 12)

| Group | Tool | Notes |
|---|---|---|
| memory | `mem_save` | session_id required; topic_key upsert; `<private>` stripped |
| | `mem_search` | query, scope, all_projects, match_mode |
| | `mem_get` | full untruncated body |
| | `mem_context` | Layer-1: recent sessions + top observations (cheap) |
| | `mem_inject` | Layer-2: hybrid retrieval rendered to a token-budgeted block |
| sessions | `mem_session_start` / `mem_session_end` / `mem_session_summary` | six-section summary contract |
| | `mem_delete`, `mem_pin`, `mem_stats` | |
| ops | `doctor` | store, vec ext, embedder, migrations — one report |

### CLI

```
mnemonic init
mnemonic save|search|get|context|inject|stats|delete|pin
mnemonic session start|end|summary
mnemonic doctor | export [--out DIR]
mnemonic mcp [--debug]          # stdio MCP server
mnemonic serve [--port 7438]    # v1.1, optional HTTP + read-only dashboard
```

- Project resolution (inherited chain): `--project` > `MNEMONIC_PROJECT` > git toplevel >
  directory basename. `--all` spans every file in the data dir.
- CLI output is the raw envelope JSON (scriptable); `--pretty` for humans. No table rendering
  in v1 — agents parse JSON, humans pipe to `jq`.

## 8. Layout & env

```
~/.mnemonic/                    # $MNEMONIC_HOME
  <project-slug>.sqlite         # one portable brain per project (WAL sidecars alongside)
  models/<model>.onnx           # local embedder (opt-in)
  config.toml                   # [embedder] provider=..., [index] dirs include/exclude
```

Env: `MNEMONIC_HOME`, `MNEMONIC_PROJECT`, `MNEMONIC_EMBEDDER` (hash|local|api),
`MNEMONIC_API_BASE`, `MNEMONIC_API_MODEL`, `MNEMONIC_API_KEY`.

**Per-project file vs one global DB** — decision: **per-project file**, matching both
predecessors: a project's brain is portable (copy the file, copy the repo) and a project's
crash can't corrupt another's. `--all` recall = open every file in the dir; at 10–50 projects
that is a few ms of extra open/scan, no federation machinery.

## 9. Opt-in intelligence (LLM curator)

Off by default (v1 is network-free). When an API is configured, the same `llm.Client` seam
provides: importance scoring, dedup adjudication (hash-collision ties), session-summary
drafting, and `mnemonic consolidate` (merge same-topic revisions, supersede). Every curated
field writes provenance; every LLM path degrades to the deterministic floor on error
(`degrade` package, one bounded retry) — an LLM timeout never blocks a save or a search.

## 10. Phasing

| Phase | Ships |
|---|---|
| **v0.1 core** | store + migrations, observations model, FTS5, save/search/get/context/sessions, MCP(7) + CLI, hash-64 vectors, doctor, stats, trails, parity tests |
| **v0.2 hybrid** | `local` ONNX embedder + `api` embedder, vec0 tables (pure-Go sqlite-vec, spike-gated), RRF fusion, `mem_inject` |
| **v0.3 code** | tree-sitter extractors (go/ts/py first), files/chunks/symbols/edges, incremental index + freshness, code_status/index/search/read/map |
| **v0.4 web** | web cache with per-source TTLs + caps, web_* tools |
| **v1.0** | LLM curator, export (markdown + JSON), `serve`, consolidate, migration importer for `skillgrid/mnemonic` stores (sync_id-stable), release |

Repo: new sibling `~/AI/mnemonic`, module `github.com/devopstales/mnemonic`, binary `mnemonic`,
Go 1.26, MIT.

## 11. Open questions (defaults chosen — challenge me)

1. **vec0 pure-Go port maturity** — spike first (`-tags`-free or build-tagged?). Default: build
   tag `sqlitevec`; brute force is the floor so a failed spike only defers speed, not function.
2. **ONNX default at v1.0** — needs ~90 MB model file. Default: bundled by default, `hash` if
   `MNEMONIC_EMBEDDER=hash` or `--no-models`.
3. **`serve` in v1?** Default: v1.1. MCP+CLI already cover agents and humans.
4. **Old-store importer scope** — full (chunks/symbols/edges too) or memory-only? Default:
   memory-only at v1.0, code reindexed from disk anyway.
5. **Name collision with installed `skillgrid:mnemonic` skill** — the skill in this ecosystem
   points at the `skillgrid` binary's `mcp` subcommand. Default: the new binary is standalone;
   skillgrid later re-exposes it via a thin `skillgrid mcp` passthrough or the skill flips to
   `mnemonic mcp` directly.
