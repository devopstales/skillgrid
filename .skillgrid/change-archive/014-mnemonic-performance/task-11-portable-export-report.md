# Task 11 — Portable Export — Report

## Sub-tasks implemented

### 11.1 [RED] ExportProject returns JSON with observations, graph edges, and embeddings
- Created `skillgrid-cli/internal/mnemonic/memory/export.go` with `ExportProject(ctx context.Context) (*ExportBundle, error)` on `*Service`.
- Defined `ExportBundle` struct: `Observations []ExportRecord`, `GraphEdges []EdgeRecord`, `Embeddings []EmbeddingRecord`, plus `Format`, `Project`, `ExportedAt` envelope fields.
- `ExportRecord` carries `id`, `type`, `content`, `metadata` (map with title, session_id, project, scope, visibility, status, revision_count, pinned, retrieval_usage, topic_key, source, normalized_hash, owner, tool_name, last_seen_at, expires_at, created_at, updated_at), `embeddings` (base64 map, keyed by the graph_ref symbol), and `graph_ref` (*int64, nil when unbound).
- `EdgeRecord` carries `id`, `kind`, `from_id`, `file_id`, `to_id`, `to_name`, `target_path`, `confidence`, `line`, `valid_from`, `valid_to` (temporal window from step 10).
- `EmbeddingRecord` carries `symbol_id`, `model`, `dim`, `vector` (base64), `updated_at`.
- Observations, edges, and embeddings are each queried in separate passes (no nested re-entrancy on the single-connection pool — see "Deadlock" below). The embeddings pass also attaches the per-observation `embeddings` field for each `graph_ref` binding (step 07 cross-link).

### 11.2 [RED] Export is portable — output can be imported into another instance
- `ImportProject(ctx context.Context, r io.Reader) error` on `*Service` decodes the COGX JSON and inserts records into the store in a single transaction.
- Upsert/skip strategy (documented):
  - **Files**: `INSERT OR IGNORE` on path (FK target for edges).
  - **Symbols**: `INSERT OR IGNORE` on uid (needed so `graph_ref` and embeddings resolve after import).
  - **Sessions**: `INSERT OR IGNORE` on id (FK target for observations).
  - **Observations**: `INSERT OR IGNORE` with the source row's explicit id — re-import is a no-op.
  - **Edges**: `INSERT OR IGNORE` on the composite unique constraint — temporal window preserved.
  - **Embeddings**: `INSERT OR REPLACE` keyed on `symbol_id` — latest vector wins, mirroring the codeindex upsert semantics.
- Roundtrip test (`TestExportImportRoundtrip`): exports 5 observations + 3 edges + 2 embeddings to a JSON file, imports into a fresh empty store under a different project id, verifies all records present with matching IDs and content.

### 11.3 [AFK] mem export CLI with --file and --skip-embeddings flags
- Added `export` subcommand to `mem.go` (manual switch pattern, consistent with existing subcommands).
- `--file out.json`: writes the bundle to a file instead of stdout (stdout gets only a summary line to stderr).
- `--skip-embeddings`: sets `bundle.Embeddings = bundle.Embeddings[:0]` and nils each observation's `embeddings` map before encoding.
- Streaming: `WriteBundle` (new exported function in export.go) uses `json.NewEncoder(w).Encode(bundle)` — the encoder writes incrementally to the `io.Writer`, so the JSON is never marshaled into one large `[]byte`. The bundle itself is still assembled row-by-row (one record per store row); the streaming boundary is the encoding step.

## ExportBundle JSON schema (sample)

```json
{
  "format": "cogx-v1",
  "project": "sample",
  "exported_at": "2026-09-11T20:44:54Z",
  "observations": [
    {
      "id": 1,
      "type": "decision",
      "content": "A sample observation for export demo",
      "metadata": {
        "created_at": "2026-09-11T20:44:54Z",
        "expires_at": "2026-09-18T20:44:54Z",
        "last_seen_at": "",
        "normalized_hash": "51044e3ad19e1944076089fa0c627fbbfc870f98999baffa75190f84cb3e09f1",
        "owner": "s1",
        "pinned": "false",
        "project": "sample",
        "retrieval_usage": "0",
        "revision_count": "0",
        "scope": "",
        "session_id": "s1",
        "source": "/tmp/export.go",
        "status": "active",
        "title": "Sample decision",
        "tool_name": "",
        "topic_key": "",
        "updated_at": "2026-09-11T20:44:54Z",
        "visibility": "private"
      },
      "embeddings": {
        "vector": "zczMPc3MTD6amZk+zczM..."
      },
      "graph_ref": 100
    }
  ],
  "graph_edges": [
    {
      "id": 1,
      "kind": "calls",
      "from_id": 100,
      "file_id": 500,
      "to_id": 200,
      "to_name": "expBeta",
      "target_path": "/tmp/export.go",
      "confidence": "EXTRACTED",
      "line": 1,
      "valid_from": 1789158494
    },
    {
      "id": 2,
      "kind": "references",
      "from_id": 200,
      "file_id": 500,
      "to_name": "someLib",
      "confidence": "EXTRACTED",
      "line": 2,
      "valid_from": 1789158994,
      "valid_to": 1789159484
    }
  ],
  "embeddings": [
    {
      "symbol_id": 100,
      "model": "export-model-v1",
      "dim": 4,
      "vector": "zczMPc3MTD6amZk+zczM...",
      "updated_at": "2026-01-02T00:00:00Z"
    }
  ]
}
```

Notes:
- `graph_edges[].valid_to` is omitted when NULL (active edge); present as a UNIX second when the edge is expired.
- `observations[].embeddings` is omitted (`omitempty`) when the observation has no `graph_ref` or the symbol has no bridge row.
- `graph_edges[].to_id`, `file_id`, `target_path` are omitted (`omitempty`) when NULL.

## Embeddings base64 encoding/decoding
- **Encode**: `base64.StdEncoding.EncodeToString(blob)` where `blob` is the raw `vector` BLOB (little-endian float32, per the step 07 convention).
- **Decode**: `base64.StdEncoding.DecodeString(record.Vector)` on import, stored directly back into the `embeddings` table.

## ImportProject upsert/skip strategy
- Observations: `INSERT OR IGNORE` with explicit source id — idempotent re-import.
- Edges: `INSERT OR IGNORE` on the composite unique constraint — no duplicates, window preserved.
- Embeddings: `INSERT OR REPLACE` on `symbol_id` — latest vector wins.
- Files/symbols/sessions: `INSERT OR IGNORE` on their unique keys — FK targets ensured.

## Streaming
`WriteBundle` uses `json.NewEncoder(w).Encode(bundle)` where `w` is an `io.Writer` (stdout or `*os.File`). The encoder writes incrementally; no `json.Marshal` → `[]byte` → `Write` pipeline. The bundle is assembled row-by-row in memory (one record per store row); the encoding step is the streaming boundary.

## CLI flags
- `mem export` — stream bundle to stdout.
- `mem export --file out.json` — write to file; stdout gets only a summary line (stderr).
- `mem export --skip-embeddings` — `bundle.Embeddings` emptied and per-observation `embeddings` nilled before encoding.
- `--project` and `--dir` are the standard mem flags.

## Deadlock fix (single-connection pool)
The store's pool is `SetMaxOpenConns(1)` (store.go:178). The original `ExportProject` ran a nested `QueryRow` per observation (to fetch its `graph_ref` embedding) while the outer observation `rows` were still open — with one connection, the outer query held the connection and the nested query blocked forever. Fix: fetch all embeddings in a SEPARATE pass (after closing the observation rows), then attach them to the matching observations by `graph_ref` index. The test fixture (`openExportTestStore`) also opens a multi-connection pool over the same SQLite file to mirror the `graph.temporal_test` pattern, so the unit tests don't depend on the pool size.

## RED→GREEN evidence

### 11.1 (TestExportProjectStructure)
- RED: `ExportProject`/`ExportRecord`/`EdgeRecord`/`EmbeddingRecord` undefined (build failure).
- GREEN: `ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  1.363s`

### 11.2 (TestExportImportRoundtrip)
- RED: same build failure (ImportProject undefined).
- GREEN: `ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  1.363s`

### 11.3 (TestMemExportCLI)
- RED: `mem export` → "unknown mem command" (exit 2).
- GREEN: `ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  11.195s`

## Full test suites

### memory package
```
$ go test ./internal/mnemonic/memory/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  12.911s
```

### cmd/skillgrid package
```
$ go test ./cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  52.555s
```

### Baseline packages
```
$ go test ./internal/mnemonic/store/... ./internal/mnemonic/service/... \
  ./internal/mnemonic/codeindex/... ./internal/mnemonic/route/... \
  ./internal/mnemonic/affected/... ./internal/mnemonic/community/... \
  ./internal/mnemonic/pdg/... -count=1

ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store       5.344s
FAIL github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service   47.404s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex 16.002s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route     2.137s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected 12.995s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community 4.272s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg      6.418s
```

**Pre-existing failure**: `TestReindexStructuralIsEmbedderFree` (service) fails with `sql: database is closed` on the clean tree (verified via `git stash`) — not caused by this change.

## Commits
- `365b9c5` feat(mnemonic): COGX-inspired portable JSON export with import roundtrip
- `dc514cb` feat(mnemonic): add mem export CLI with file and skip-embeddings flags

## Concerns
- **Bundle assembled in memory**: the bundle struct (all observations, edges, embeddings) is held in memory before encoding. For very large stores (100k+ observations), this could be a memory concern. A fully streaming implementation would encode each section incrementally (write the JSON prefix, stream observations, write edges, write embeddings, close the array) but is more complex. The current approach streams the ENCODING (no giant `[]byte`), which is the requirement's stated boundary.
- **Pre-existing test failure**: `TestReindexStructuralIsEmbedderFree` fails on the clean tree (unrelated to this change).
