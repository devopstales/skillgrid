package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
)

// [door-check-code] the hypothesis falsifier: a code_index writes
// files+symbols+edges through the extended Store (SQLite, WAL), and the FTS
// floor finds a saved symbol. The v1 live probe showed the broken shape (the
// FTS leg missed the queried symbol; every graph read failed `no such
// column`). Task 1 is the stop gate: if this fails, the code-index's
// foundation (the Store extension + the single-node-table graph schema, CI-2)
// does not hold.
func TestDoorCheckCodeIndexWritesAndSearches(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	// the file (INV-6: sync_id is minted by the store on upsert)
	fid, err := st.UpsertFile(ctx, FileRow{
		Path: "internal/pay/pay.go", MtimeNS: 1, Size: 100,
		ContentHash: "abc", Language: "go",
	})
	if err != nil {
		t.Fatalf("upsert file: %v", err)
	}

	// the symbol (the single node table, CI-2)
	syms := []Symbol{{
		Name: "CapturePayment", QualifiedName: "internal/pay.CapturePayment",
		Kind: "method", Language: "go", StartLine: 480, EndLine: 520,
		ContentHash: "s1", UID: "pay.go:CapturePayment:method",
	}}
	if _, err := st.UpsertSymbols(ctx, fid, syms); err != nil {
		t.Fatalf("upsert symbols: %v", err)
	}

	// a name-only calls edge (CI-INV-3, drop-not-guess: to_id NULL + to_name)
	edges := []Edge{{Kind: "calls", ToName: "CapturePayment", Confidence: "AMBIGUOUS", Line: 79}}
	if err := st.ReplaceEdgesForFile(ctx, fid, edges); err != nil {
		t.Fatalf("replace edges: %v", err)
	}

	// the name floor must find the symbol by name (idx_symbols_name on the
	// single `symbols` table — the identifier-first lookup the FTS leg
	// (Task 5, fts_leg.go) builds on)
	rows, err := st.SymbolByName(ctx, "CapturePayment")
	if err != nil {
		t.Fatalf("symbol by name: %v", err)
	}
	if len(rows) != 1 || rows[0].UID != "pay.go:CapturePayment:method" {
		t.Fatalf("name floor broken: %+v", rows)
	}

	// the FTS floor (B1): a real FTS5 MATCH assertion, not a string check
	if err := st.ReplaceChunksForFile(ctx, fid, []Chunk{{
		Kind: "ast", StartLine: 1, EndLine: 20,
		Text: "CapturePayment trigram recall", ContentHash: "c1",
	}}); err != nil {
		t.Fatalf("replace chunks: %v", err)
	}
	if n := ftsMatchCount(t, st, "chunks_fts", "trigram"); n != 1 {
		t.Fatalf("chunks_fts MATCH must find the inserted chunk: got %d", n)
	}
	// the chunk id via the same RawQuery test seam (ChunkIDByFile was
	// test-only surface on the D2 door — review F1: off the interface)
	cid, err := rawQueryInt(t, st, `SELECT id FROM chunks WHERE file_id = ? ORDER BY id LIMIT 1`, fid)
	if err != nil {
		t.Fatalf("chunk id: %v", err)
	}
	if n := ftsMatchCount(t, st, "symbol_fts", "CapturePayment"); n != 1 {
		t.Fatalf("symbol_fts MATCH must find the indexed symbol: got %d", n)
	}

	// the single-node-table reads (the walker's surface, dialect-agnostic)
	f, err := st.FileByPath(ctx, "internal/pay/pay.go")
	if err != nil {
		t.Fatalf("file by path: %v", err)
	}
	if !regexp.MustCompile(`^file-[0-9a-f]{16}$`).MatchString(f.SyncID) {
		t.Fatalf("bad file sync id format: %q", f.SyncID)
	}
	if !regexp.MustCompile(`^sym-[0-9a-f]{16}$`).MatchString(rows[0].SyncID) {
		t.Fatalf("bad symbol sync id format: %q", rows[0].SyncID)
	}
	s, err := st.SymbolByUID(ctx, "pay.go:CapturePayment:method")
	if err != nil || s.Name != "CapturePayment" {
		t.Fatalf("symbol by uid: %+v (%v)", s, err)
	}
	byFrom, err := st.EdgesByFrom(ctx, fid, nil)
	if err != nil {
		t.Fatalf("edges by from: %v", err)
	}
	if len(byFrom) != 1 || byFrom[0].ToID != nil || byFrom[0].ToName != "CapturePayment" {
		t.Fatalf("edges by from broken: %+v", byFrom)
	}
	nameOnly, err := st.EdgesByNameOnlyToName(ctx, "CapturePayment", nil)
	if err != nil {
		t.Fatalf("edges name-only: %v", err)
	}
	if len(nameOnly) != 1 || nameOnly[0].ToID != nil {
		t.Fatalf("edges name-only broken: %+v", nameOnly)
	}
	if _, err := st.EdgesByTo(ctx, s.ID, nil); err != nil {
		t.Fatalf("edges by to: %v", err)
	}

	// the vector methods (M2: BLOB store; the ANN render is TICKET-08a's) —
	// over live rows (the FK is ON DELETE CASCADE: a cleared chunk's vector
	// is gone with it)
	if err := st.EmbedSymbol(ctx, s.ID, "hash64", []float32{0.1, 0.2, 0.3}); err != nil {
		t.Fatalf("embed symbol: %v", err)
	}
	if err := st.EmbedChunk(ctx, cid, "hash64", []float32{0.9, 0.8, 0.7}); err != nil {
		t.Fatalf("embed chunk: %v", err)
	}
	if !rawTableHasRows(t, st, "embeddings", "vector IS NOT NULL") {
		t.Fatalf("embeddings BLOB not stored")
	}
	if !rawTableHasRows(t, st, "chunk_embeddings", "vector IS NOT NULL") {
		t.Fatalf("chunk_embeddings BLOB not stored")
	}

	// target-state: replacing with an empty set clears the file's rows
	if err := st.ReplaceEdgesForFile(ctx, fid, nil); err != nil {
		t.Fatalf("replace edges empty: %v", err)
	}
	if n, err := st.EdgesByFrom(ctx, fid, nil); err != nil || len(n) != 0 {
		t.Fatalf("edges must be target-state (cleared): %+v (%v)", n, err)
	}
	if err := st.ReplaceChunksForFile(ctx, fid, nil); err != nil {
		t.Fatalf("replace chunks empty: %v", err)
	}
	if n := ftsMatchCount(t, st, "chunks_fts", "trigram"); n != 0 {
		t.Fatalf("chunks_fts must follow the target-state clear: got %d", n)
	}
	// the chunk cascade took its vector too (the FK, M10's shape)
	if rawTableHasRows(t, st, "chunk_embeddings", "vector IS NOT NULL") {
		t.Fatalf("chunk cascade must take the chunk's vector")
	}

	// the model-swap guard's DELETE (v1 indexer.go:1133-1160 shape)
	if err := st.ClearVectors(ctx); err != nil {
		t.Fatalf("clear vectors: %v", err)
	}
	if rawTableHasRows(t, st, "embeddings", "vector IS NOT NULL") {
		t.Fatalf("clear vectors must empty the symbol BLOB table")
	}

	// the trail seam (T15/INV-5 — the shared retrieval_trails table, 003)
	if err := st.RecordRetrieval(ctx, RetrievalTrail{Route: "fts", Query: "CapturePayment", Degraded: false}); err != nil {
		t.Fatalf("record retrieval: %v", err)
	}
	if !rawTableHasRows(t, st, "retrieval_trails", "route = 'fts' AND query = 'CapturePayment'") {
		t.Fatalf("retrieval_trails row not written")
	}

	// the DDL seam (T21 — the KB's table-adding surface; multi-statement)
	if err := st.ExecDDL(ctx, []string{
		"CREATE TABLE kb_pages (id INTEGER PRIMARY KEY, title TEXT)",
		"CREATE INDEX idx_kb_pages_title ON kb_pages(title)",
	}); err != nil {
		t.Fatalf("exec ddl: %v", err)
	}
	if !rawTableExists(t, st, "kb_pages") {
		t.Fatalf("ExecDDL did not create the table")
	}

	// absent-on-disk = delete (M10): the files row removed → symbols/edges cascade
	if err := st.DeleteFile(ctx, "internal/pay/pay.go"); err != nil {
		t.Fatalf("delete file: %v", err)
	}
	if _, err := st.FileByPath(ctx, "internal/pay/pay.go"); err == nil {
		t.Fatalf("delete file: the row must be gone")
	}
	if rows, _ := st.SymbolByName(ctx, "CapturePayment"); len(rows) != 0 {
		t.Fatalf("delete file: symbols must cascade: %+v", rows)
	}
}

// the extension is additive: the memory change's observation methods still
// work on the same store, and the code-index tables coexist in the same DB.
func TestStoreExtensionAdditive(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	res, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "decision", Title: "The additive extension keeps memory working",
		Content: "The memory observation survives the code-index tables",
		Project: "proj-a", Scope: "project",
	})
	if err != nil || res.SyncID == "" {
		t.Fatalf("memory AddObservation broken by the extension: %v", err)
	}
	if !regexp.MustCompile(`^obs-[0-9a-f]{16}$`).MatchString(res.SyncID) {
		t.Fatalf("memory sync id format changed: %q", res.SyncID)
	}
	// and the code-index tables coexist in the same database
	if _, err := st.UpsertFile(ctx, FileRow{Path: "a.go", ContentHash: "x", Language: "go"}); err != nil {
		t.Fatalf("code-index table on the same store: %v", err)
	}
	// the memory search still answers (the FTS trigger set is untouched)
	r, err := st.Search(ctx, SearchQuery{Query: "additive", Project: "proj-a", Limit: 5})
	if err != nil {
		t.Fatalf("memory search: %v", err)
	}
	if len(r.Observations) != 1 || r.Observations[0].ID != res.ID {
		t.Fatalf("memory search broken: %+v", r)
	}
}

// [sync-id-format-files-symbols] the file/symbol sync-id mints follow INV-6
// (prefix + exactly 16 hex) and are unique within a burst.
func TestCodeIndexSyncIDMintFormat(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	wantFile := regexp.MustCompile(`^file-[0-9a-f]{16}$`)
	wantSym := regexp.MustCompile(`^sym-[0-9a-f]{16}$`)
	seenFile, seenSym := map[string]bool{}, map[string]bool{}
	for i := 0; i < 32; i++ {
		if _, err := st.UpsertFile(ctx, FileRow{
			Path: fmt.Sprintf("burst/f%d.go", i), ContentHash: fmt.Sprintf("h%d", i),
			Language: "go",
		}); err != nil {
			t.Fatalf("upsert file %d: %v", i, err)
		}
	}
	for i := 0; i < 32; i++ {
		f, err := st.FileByPath(ctx, fmt.Sprintf("burst/f%d.go", i))
		if err != nil {
			t.Fatalf("file %d: %v", i, err)
		}
		if !wantFile.MatchString(f.SyncID) || seenFile[f.SyncID] {
			t.Fatalf("file sync id %q (i=%d): format or duplicate broken", f.SyncID, i)
		}
		seenFile[f.SyncID] = true
	}
	// symbols on the way out: upsert a burst, read every row back
	if fid, err := st.UpsertFile(ctx, FileRow{Path: "burst/sym.go", ContentHash: "hs", Language: "go"}); err != nil {
		t.Fatalf("upsert sym file: %v", err)
	} else {
		burst := make([]Symbol, 32)
		for i := range burst {
			burst[i] = Symbol{
				Name: fmt.Sprintf("Sym%d", i), Kind: "function", Language: "go",
				UID: fmt.Sprintf("sym.go:Sym%d:function", i),
			}
		}
		if _, err := st.UpsertSymbols(ctx, fid, burst); err != nil {
			t.Fatalf("upsert symbol burst: %v", err)
		}
	}
	for i := 0; i < 32; i++ {
		rows, err := st.SymbolByName(ctx, fmt.Sprintf("Sym%d", i))
		if err != nil || len(rows) != 1 {
			t.Fatalf("symbol read %d: %+v (%v)", i, rows, err)
		}
		if !wantSym.MatchString(rows[0].SyncID) || seenSym[rows[0].SyncID] {
			t.Fatalf("symbol sync id %q (i=%d): format or duplicate broken", rows[0].SyncID, i)
		}
		seenSym[rows[0].SyncID] = true
	}
}

// the K1 single per-file tx surface: the driver composes the five per-file
// writes inside ONE tx (all commit or all roll back).
func TestCodeIndexTxnSurface(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	tx, err := st.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.UpsertFile(ctx, FileRow{Path: "tx/f.go", ContentHash: "h", Language: "go"}); err != nil {
		t.Fatalf("tx upsert file: %v", err)
	}
	// a failure in the middle must roll the file upsert back too
	if err := tx.ReplaceEdgesForFile(ctx, 999999, []Edge{{Kind: "calls", ToName: "x"}}); err == nil {
		t.Fatalf("tx replace edges for a missing file: expected an FK error")
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if _, err := st.FileByPath(ctx, "tx/f.go"); err == nil {
		t.Fatalf("rollback must undo the file upsert")
	}
	// and the commit path
	tx2, err := st.Begin(ctx)
	if err != nil {
		t.Fatalf("begin 2: %v", err)
	}
	if _, err := tx2.UpsertFile(ctx, FileRow{Path: "tx/g.go", ContentHash: "h", Language: "go"}); err != nil {
		t.Fatalf("tx2 upsert: %v", err)
	}
	if err := tx2.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := st.FileByPath(ctx, "tx/g.go"); err != nil {
		t.Fatalf("commit must keep the file upsert: %v", err)
	}
}

// the driver-side orphan prune (002 §9: a file whose symbols no longer exist
// is pruned by the uid set the extract declared).
func TestPruneOrphanSymbols(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	fid, err := st.UpsertFile(ctx, FileRow{Path: "prune/p.go", ContentHash: "h", Language: "go"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fid, []Symbol{
		{Name: "Keep", Kind: "function", UID: "p.go:Keep:function"},
		{Name: "Gone", Kind: "function", UID: "p.go:Gone:function"},
	}); err != nil {
		t.Fatalf("upsert syms: %v", err)
	}
	if err := st.PruneOrphanSymbols(ctx, []string{"p.go:Keep:function"}); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if rows, _ := st.SymbolByName(ctx, "Gone"); len(rows) != 0 {
		t.Fatalf("pruned symbol must be gone: %+v", rows)
	}
	if rows, _ := st.SymbolByName(ctx, "Keep"); len(rows) != 1 {
		t.Fatalf("kept symbol must survive: %+v", rows)
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

// ftsMatchCount runs a real FTS5 MATCH against the named table and returns
// the hit count (the FTS floor assertion — not a string check).
func ftsMatchCount(t *testing.T, st Store, table, term string) int {
	t.Helper()
	rows, err := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	}).RawQuery(`SELECT COUNT(*) FROM `+table+` WHERE `+table+` MATCH ?`, term)
	if err != nil {
		t.Fatalf("fts match %s: %v", table, err)
	}
	n, ok := rows[0][0].(int64)
	if !ok {
		t.Fatalf("fts match %s: bad count type %T", table, rows[0][0])
	}
	return int(n)
}

// rawQueryInt reads a single int off the RawQuery test seam (the
// ChunkIDByFile replacement — review F1: test-only reads stay on the seam,
// off the D2 interface).
func rawQueryInt(t *testing.T, st Store, q string, args ...any) (int64, error) {
	t.Helper()
	rows, err := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	}).RawQuery(q, args...)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, errors.New("rawQueryInt: no rows")
	}
	n, ok := rows[0][0].(int64)
	if !ok {
		return 0, fmt.Errorf("rawQueryInt: bad type %T", rows[0][0])
	}
	return n, nil
}

// rawTableHasRows probes the raw table (the SQLite store's Probe seam).
func rawTableHasRows(t *testing.T, st Store, table, where string) bool {
	t.Helper()
	raw := st.(interface {
		Probe(query string, args ...any) (bool, error)
	})
	found, err := raw.Probe(`SELECT EXISTS(SELECT 1 FROM ` + table + ` WHERE ` + where + `)`)
	if err != nil {
		t.Fatalf("probe %s: %v", table, err)
	}
	return found
}

// rawTableExists probes the catalog for a table (a just-created, still-empty
// table: EXISTS-over-rows is false until the first row, so the DDL seam
// asserts the catalog, not the row count).
func rawTableExists(t *testing.T, st Store, table string) bool {
	t.Helper()
	raw := st.(interface {
		Probe(query string, args ...any) (bool, error)
	})
	found, err := raw.Probe(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE name = ` + quoteLit(table) + `)`)
	if err != nil {
		t.Fatalf("catalog probe %s: %v", table, err)
	}
	return found
}

func quoteLit(s string) string { return "'" + s + "'" }
