package store

import (
	"context"
	"testing"
)

// TestPostgresCodeDialectRender verifies the PG code-index render (TICKET-08c,
// the DoD gate — one real PG run before ship; the PG-skip risk note M8). It
// skips without MNEMONIC_PG_TEST_DSN (gate G-15); with the DSN it proves:
//   - the 7 content + 2 FTS (tsvector) tables from an empty schema (the
//     same-number mirror of Task 1's migration);
//   - SymbolByName works on PG (the dialect-agnostic walker, INV-2);
//   - the FTS leg on PG: the tsvector triggers index a symbol + chunk, the
//     tsvector MATCH finds the symbol;
//   - the under-prefix on PG: a files.path prefix narrows the FTS leg
//     (the dialect-specific prefix query);
//   - the vector methods: EmbedSymbol/EmbedChunk write the pgvector BLOB,
//     ClearVectors deletes;
//   - the ON CONFLICT upsert: a repeat UpsertFile with the same path → no
//     error, the row is updated.
func TestPostgresCodeDialectRender(t *testing.T) {
	st := openPGTestStore(t)
	ctx := context.Background()

	// The 7 content + 2 FTS tables from an empty schema (the migration
	// applied at Open — the same-number mirror of Task 1's 100_code_index.sql).
	// A read against a code-index table proves the table exists (the runner
	// applied the migration).
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("re-migrate (idempotent): %v", err)
	}

	// The code-index reads run on PG (the dialect-agnostic walker, INV-2).
	// A miss returns an empty slice (no rows), not an error.
	if _, err := st.SymbolByName(ctx, "x"); err != nil {
		t.Fatalf("PG SymbolByName (empty store): %v", err)
	}

	// UpsertFile (the ON CONFLICT upsert — the path is the stable key).
	fid, err := st.UpsertFile(ctx, FileRow{
		Path: "a.go", MtimeNS: 1, Size: 10, ContentHash: "c", Language: "go",
	})
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	if fid == 0 {
		t.Fatalf("UpsertFile must return the file rowid, got 0")
	}

	// A repeat UpsertFile with the same path → no error, the row is updated
	// (the ON CONFLICT DO UPDATE — not a unique-violation error).
	fid2, err := st.UpsertFile(ctx, FileRow{
		Path: "a.go", MtimeNS: 2, Size: 11, ContentHash: "c2", Language: "go",
	})
	if err != nil {
		t.Fatalf("repeat UpsertFile (ON CONFLICT): %v", err)
	}
	if fid2 != fid {
		t.Fatalf("a repeat UpsertFile by path must return the same rowid, got %d != %d", fid2, fid)
	}

	// UpsertSymbols (the tsvector trigger indexes the new row).
	n, err := st.UpsertSymbols(ctx, fid, []Symbol{{
		Name: "WireGitHooks", QualifiedName: "x.WireGitHooks", Kind: "function",
		Language: "go", UID: "a.go:WireGitHooks:function",
	}})
	if err != nil {
		t.Fatalf("UpsertSymbols: %v", err)
	}
	if n != 1 {
		t.Fatalf("UpsertSymbols must report 1 upserted, got %d", n)
	}

	// SymbolByName works on PG (the dialect-agnostic walker, INV-2).
	syms, err := st.SymbolByName(ctx, "WireGitHooks")
	if err != nil {
		t.Fatalf("PG SymbolByName: %v", err)
	}
	if len(syms) != 1 || syms[0].UID != "a.go:WireGitHooks:function" {
		t.Fatalf("PG SymbolByName must find the indexed symbol, got %+v", syms)
	}

	// The FTS leg on PG: the tsvector trigger indexed the symbol; the
	// tsvector MATCH finds it (the PG FTS render works).
	if n := pgFTSMatchCount(t, st, "symbols", "wiregithooks"); n < 1 {
		t.Fatalf("PG tsvector FTS must find the indexed symbol: got %d", n)
	}

	// The chunks leg: insert a chunk, the tsvector trigger indexes it, the
	// tsvector MATCH finds the chunk.
	if err := st.ReplaceChunksForFile(ctx, fid, []Chunk{{
		Kind: "ast", StartLine: 1, EndLine: 5, Text: "wire the git hooks", ContentHash: "c1",
	}}); err != nil {
		t.Fatalf("ReplaceChunksForFile: %v", err)
	}
	if n := pgFTSMatchCount(t, st, "chunks", "hooks"); n < 1 {
		t.Fatalf("PG tsvector FTS must find the indexed chunk: got %d", n)
	}

	// The under-prefix on PG: a files.path prefix narrows the FTS leg (the
	// dialect-specific prefix query — the FTS leg JOINs files and filters
	// by path). A non-matching prefix → 0.
	if n := pgFTSMatchCountUnder(t, st, "chunks", "hooks", "a.go"); n < 1 {
		t.Fatalf("PG under-prefix (matching) must narrow the FTS leg: got %d", n)
	}
	if n := pgFTSMatchCountUnder(t, st, "chunks", "hooks", "b.go"); n != 0 {
		t.Fatalf("PG under-prefix (non-matching) must exclude the FTS leg: got %d", n)
	}

	// The vector methods: EmbedSymbol/EmbedChunk write the pgvector BLOB,
	// ClearVectors deletes.
	vec := []float32{0.1, 0.2, 0.3}
	if err := st.EmbedSymbol(ctx, syms[0].ID, "test-model", vec); err != nil {
		t.Fatalf("EmbedSymbol (pgvector BLOB): %v", err)
	}
	// The chunk id (the chunk row the ReplaceChunksForFile inserted).
	chunkID, err := pgChunkIDForFile(t, st, fid)
	if err != nil {
		t.Fatalf("chunk id: %v", err)
	}
	if err := st.EmbedChunk(ctx, chunkID, "test-model", vec); err != nil {
		t.Fatalf("EmbedChunk (pgvector BLOB): %v", err)
	}
	// The BLOB rows exist (the vector was written).
	if n := pgVectorCount(t, st, "embeddings", "symbol_id", syms[0].ID); n != 1 {
		t.Fatalf("EmbedSymbol must write the pgvector BLOB: got %d embeddings rows", n)
	}
	if n := pgVectorCount(t, st, "chunk_embeddings", "chunk_id", chunkID); n != 1 {
		t.Fatalf("EmbedChunk must write the pgvector BLOB: got %d chunk_embeddings rows", n)
	}
	// ClearVectors deletes both BLOB tables (the model-swap guard's DELETE).
	if err := st.ClearVectors(ctx); err != nil {
		t.Fatalf("ClearVectors: %v", err)
	}
	if n := pgVectorCount(t, st, "embeddings", "symbol_id", syms[0].ID); n != 0 {
		t.Fatalf("ClearVectors must delete the embeddings BLOB rows: got %d", n)
	}
	if n := pgVectorCount(t, st, "chunk_embeddings", "chunk_id", chunkID); n != 0 {
		t.Fatalf("ClearVectors must delete the chunk_embeddings BLOB rows: got %d", n)
	}

	// The per-file tx (K1): Begin composes the five per-file writes over one
	// tx; a rollback undoes them (the idempotency claim rests on it).
	tx, err := st.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if _, err := tx.UpsertFile(ctx, FileRow{Path: "tx.go", MtimeNS: 1, Size: 1, ContentHash: "c", Language: "go"}); err != nil {
		t.Fatalf("tx.UpsertFile: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("tx.Rollback: %v", err)
	}
	// The rolled-back file is absent (the tx did not commit).
	if _, err := st.FileByPath(ctx, "tx.go"); err == nil {
		t.Fatalf("a rolled-back tx.UpsertFile must not persist the file")
	}
}

// pgFTSMatchCount runs the PG tsvector FTS leg over table and returns the
// row count for query (the dialect-specific FTS render: tsvector MATCH with
// the 'simple' config, the trigram analog). It uses the store's RawQuery
// seam (the dialect-agnostic test surface).
func pgFTSMatchCount(t *testing.T, st Store, table, query string) int {
	t.Helper()
	raw, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		t.Fatalf("store %T does not expose the RawQuery seam", st)
	}
	rows, err := raw.RawQuery(
		`SELECT count(*) FROM `+table+` WHERE tsv @@ plainto_tsquery('simple', $1)`, query)
	if err != nil {
		t.Fatalf("PG FTS MATCH (%s): %v", table, err)
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return 0
	}
	n, ok := rows[0][0].(int64)
	if !ok {
		// lib/pq returns COUNT(*) as int64; a different type is a drift.
		t.Fatalf("PG FTS MATCH count type %T", rows[0][0])
	}
	return int(n)
}

// pgFTSMatchCountUnder runs the PG tsvector FTS leg narrowed by a files.path
// prefix (the dialect-specific prefix query — the under? surface: the FTS
// leg JOINs the files table and filters by path). It returns the row count.
func pgFTSMatchCountUnder(t *testing.T, st Store, table, query, pathPrefix string) int {
	t.Helper()
	raw, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		t.Fatalf("store %T does not expose the RawQuery seam", st)
	}
	// The under-prefix: the FTS leg (tsvector MATCH) JOINs the files table
	// and narrows by the path prefix. The dialect-specific prefix query is
	// the ILIKE (the searchLIKEPG pattern — case-insensitive substring),
	// not a LIKE (the SQLite leg's prefix).
	q := `SELECT count(*) FROM ` + table + ` c
		JOIN files f ON f.id = c.file_id
		WHERE c.tsv @@ plainto_tsquery('simple', $1)
		  AND f.path ILIKE $2`
	rows, err := raw.RawQuery(q, query, pathPrefix+"%")
	if err != nil {
		t.Fatalf("PG under-prefix (%s %q): %v", table, pathPrefix, err)
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return 0
	}
	n, ok := rows[0][0].(int64)
	if !ok {
		t.Fatalf("PG under-prefix count type %T", rows[0][0])
	}
	return int(n)
}

// pgChunkIDForFile returns the first chunk rowid for a file (the vector
// write's target — the chunk the ReplaceChunksForFile inserted).
func pgChunkIDForFile(t *testing.T, st Store, fileID int64) (int64, error) {
	t.Helper()
	raw, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return 0, nil
	}
	rows, err := raw.RawQuery(`SELECT id FROM chunks WHERE file_id = $1 ORDER BY id LIMIT 1`, fileID)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return 0, nil
	}
	id, ok := rows[0][0].(int64)
	if !ok {
		return 0, nil
	}
	return id, nil
}

// pgVectorCount counts the vector BLOB rows in table keyed by (keyCol, val).
func pgVectorCount(t *testing.T, st Store, table, keyCol string, val int64) int {
	t.Helper()
	raw, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		t.Fatalf("store %T does not expose the RawQuery seam", st)
	}
	rows, err := raw.RawQuery(`SELECT count(*) FROM `+table+` WHERE `+keyCol+` = $1`, val)
	if err != nil {
		t.Fatalf("PG vector count (%s): %v", table, err)
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return 0
	}
	n, ok := rows[0][0].(int64)
	if !ok {
		t.Fatalf("PG vector count type %T", rows[0][0])
	}
	return int(n)
}
