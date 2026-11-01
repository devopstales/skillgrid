# 001 — Storage (dual-mode: SQLite | Postgres)

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `000-architecture` (INV-2, INV-4, INV-6)
**Reference models:** Engram local/cloud duality (SQLite local truth + Postgres cloud
replication); mnemonic v1 `internal/store` (embedded migrations, handle pool, WAL retry).

---

## 1. Capability

A single dialect-agnostic data layer that runs on **SQLite by default** (local, zero-ops,
one file per project) and on **PostgreSQL when a remote data source is configured**. The
same schema, the same query surface, both dialects. Local is always the source of truth
(INV-4); Postgres is replication/shared access, not a second authority.

## 2. The dialect abstraction

```
core/store
├── Store            interface — the query surface every function calls
├── SQLiteStore      implements Store (default)
└── PostgresStore    implements Store (remote)
```

**`Store` is the only thing functions see.** `internal/memory` never names
`*sql.DB`, never writes raw SQL that is dialect-specific, and never opens a connection.
It calls `Store` methods (`AddObservation`, `GetObservation`, `Search`, …). The
dialect-specific SQL lives *inside* `SQLiteStore` / `PostgresStore`.

### 2.1 The rules that make dual-mode real

| # | Rule |
|---|---|
| S-1 | **One logical schema, two physical dialects.** `002-schema` defines every table once, in dialect-neutral form. Each store renders it for its dialect. A feature that cannot be expressed in *both* dialects is not allowed. |
| S-2 | **No dialect-only features in the contract.** `Store` methods return Go types, never `*sql.Rows`. FTS, vectors, and JSON are behind `Store` methods, not raw SQL. |
| S-3 | **Local truth is enforced structurally.** In the replication path (SQLite local + Postgres remote), Postgres receives *replicated* mutations (see `01-memory/006-sync.md`); a save is never written to it directly. `PostgresStore` is the direct-PG **dialect render** — a deployment mode in which local truth *is* Postgres (one `schema_<project>` per project, same isolation as the SQLite file-per-project). It is a different deployment, not the replication path. (Amended 2026-11-01 at the memory blueprint gate; the amendment is recorded in the blueprint's Global Constraints.) |
| S-4 | **Identity is `sync_id` (INV-6).** The local rowid is dialect- and machine-local. All cross-machine identity, relations, and provenance use `sync_id`. |

## 3. FTS5 ↔ `tsvector`

Full-text search is the retrieval backbone of memory. It must exist on both dialects.

| Concern | SQLite | Postgres |
|---|---|---|
| Index | FTS5 virtual table, `tokenize='trigram'` (matches Engram) | `tsvector` column + GIN index, `to_tsvector('simple', …)` (trigram-equivalent substring recall via `pg_trgm` fallback) |
| Sync | Triggers on `observations` (insert/update/delete) keep FTS in step | `BEFORE INSERT/UPDATE/DELETE` trigger maintaining the `tsv` column |
| Rank | `bm25(fts, w_title, w_content, w_type, w_project, w_topic_key)` | `ts_rank` / `ts_rank_cd` with per-column weight classes |
| Short-term fallback | terms < 3 runes → bounded `LIKE` (Engram `hasShortFTSTerm`) | `pg_trgm` `similarity()` / `LIKE` |

**Contract:** `Store.Search(query, filters)` returns the same `SearchResult` shape on
both dialects. The ranking *weights* are identical; the ranking *mechanism* differs.
`degraded: true` is set when the FTS index is absent/being rebuilt (not when a dialect
differs).

## 4. Vector store: sqlite-vec ↔ pgvector

The `embedding` columns exist as a **reservation** (Engram does the same — columns
present, no vector search in the retrieval path yet). v2 *activates* them, but the
activation is behind the same dialect split:

| Concern | SQLite | Postgres |
|---|---|---|
| Index | `sqlite-vec` (or a pure-Go IVF/HNSW sidecar as in v1 `vec0`) | `pgvector` `vector` column + HNSW/IVFFlat index |
| Query | `Store.VectorSearch(queryVec, k, filters)` | same method, `ORDER BY embedding <-> $1 LIMIT k` |
| Embedder | the `intelligence/embedder` seam (spec `003-intelligence`) | same seam |

**Degradation (INV-3):** if the embedder is unavailable, `VectorSearch` returns empty
and sets `degraded: true`. FTS search is unaffected. A store without a vector index
still serves FTS.

## 5. The store lifecycle

- **Open:** `store.Open(dataDir, projectID)` → resolves the dialect from config
  (`006-config`), returns a `Store`. Refcounted handle pool (v1 behavior) prevents N+1
  opens of the same project.
- **Migrate:** each store runs its dialect's migration set on open. Migrations are
  **additive only** (v1 discipline): existing files keep opening. `002-schema` is the
  target state; migrations are the path to it.
- **WAL / isolation:** SQLite opens in WAL with a busy timeout + retry backoff (v1
  behavior). Postgres uses `READ COMMITTED` by default; writes are serialized through
  the memory write queue (`01-memory/001-observations.md` §write-queue).
- **Close:** refcounted; the DB closes only when the last reference returns.

## 6. Postgres reachability (resolves OPEN-4)

The remote is **a self-hosted PostgreSQL the team runs** — not a mnemonic-managed
service (no `mnemonic cloud serve`, no HTTP sync API). The store reaches it **directly
over a DSN** from config; the CLI exposes `--datasource` to override. When no remote is
configured, the store is SQLite-only and the sync journal (`01-memory/006-sync.md`) is a
no-op.

**Project isolation (the core constraint):** local is **one `.sqlite` file per project**;
remote is **one Postgres schema per project** (`schema_<project>`). The DSN selects the
*server*; the project is resolved at open and the store scopes all remote queries to that
project's schema. Projects never share a file or a schema. See `01-memory/006-sync` §2.

## 7. Interface (contract, not implementation)

```go
type Store interface {
    // observations
    AddObservation(ctx, p AddObservationParams) (AddObservationResult, error)
    GetObservation(ctx, id int64, includeHistory bool) (Observation, error)
    UpdateObservationForProject(ctx, id int64, expectedProject string, p UpdateParams) error
    DeleteObservationForProject(ctx, id int64, expectedProject string, hard bool) error
    ConsolidateObservations(ctx, p ConsolidateParams) (ConsolidateResult, error)

    // relations
    AddRelation(ctx, p AddRelationParams) (Relation, error)
    JudgeRelation(ctx, id int64, p JudgeParams) error

    // retrieval
    Search(ctx, q SearchQuery) (SearchResult, error)
    VectorSearch(ctx, q VecQuery) (VecResult, error)
    Context(ctx, q ContextQuery) (ContextResult, error)

    // sync
    EnqueueMutation(ctx, m Mutation) (seq int64, err error)
    ListPendingMutations(ctx, cursor int64, limit int) ([]Mutation, error)
    ApplyPulledMutation(ctx, m Mutation) error
    Tombstone(ctx, entity, key string) error

    // lifecycle
    Migrate(ctx) error
    Close() error
    Ping(ctx) error
}
```

The full type definitions live in `002-schema.md`.

## 8. Invariants (from `000-architecture`)

INV-2 (dual-mode), INV-4 (local truth), INV-6 (sync_id identity).

## 9. Open items

None — OPEN-4 (PG reachability) and OPEN-5 (vector store) resolved above.
