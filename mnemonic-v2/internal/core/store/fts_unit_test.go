package store

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// [fts-unit-reusable] (M11): the reusable FTS unit is PARAMETERIZED (table
// name, content columns, tokenizer) and EXECUTABLE — the KB's kb_pages_fts
// re-uses it verbatim. The contract: the DDL applies to a real temp store,
// and a table created via the unit STAYS IN SYNC (insert → MATCH finds;
// update → new term matches + old stops; delete → empty), not a string
// shape. The KB's kb_pages_fts is the canonical consumer shape.
func TestFtsUnitReusable(t *testing.T) {
	ctx := context.Background()
	// the KB shape: 4 content columns, a different table, the same unit
	ddls := FTSDDL("kb_pages_fts", "kb_pages", "id",
		[]string{"title", "body", "type", "rel_path"}, "trigram")
	if len(ddls) != 5 {
		t.Fatalf("expected 1 FTS DDL + 4 triggers, got %d", len(ddls))
	}
	for _, s := range ddls {
		if !strings.Contains(s, "kb_pages_fts") {
			t.Fatalf("DDL must name the table: %s", s)
		}
	}
	// the virtual-table statement is the FTS5 shape
	if !strings.Contains(ddls[0], "fts5(") || !strings.Contains(ddls[0], "content='kb_pages'") {
		t.Fatalf("virtual-table DDL must be external-content FTS5: %s", ddls[0])
	}
	// executable against a real store (the KB's content table fixture)
	st := openTestStore(t)
	if err := st.ExecDDL(ctx, []string{
		"CREATE TABLE kb_pages (id INTEGER PRIMARY KEY, title TEXT, body TEXT, type TEXT, rel_path TEXT)",
	}); err != nil {
		t.Fatalf("create content table: %v", err)
	}
	for _, s := range ddls {
		if err := st.ExecDDL(ctx, []string{s}); err != nil {
			t.Fatalf("FTSDDL not executable: %v\n%s", err, s)
		}
	}
	// stays in sync: insert → MATCH finds it
	if err := st.ExecDDL(ctx, []string{
		"INSERT INTO kb_pages (id, title, body, type, rel_path) VALUES (1, 'alpha trigram', 'x', 'doc', 'a.md')",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if n := ftsMatchCount(t, st, "kb_pages_fts", "trigram"); n != 1 {
		t.Fatalf("insert must sync the FTS: got %d", n)
	}
	// update → the new term matches, the old stops
	if err := st.ExecDDL(ctx, []string{
		"UPDATE kb_pages SET title='alpha bigger' WHERE id=1",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if n := ftsMatchCount(t, st, "kb_pages_fts", "bigger"); n != 1 {
		t.Fatalf("update must sync the new term: got %d", n)
	}
	if n := ftsMatchCount(t, st, "kb_pages_fts", "trigram"); n != 0 {
		t.Fatalf("update must desync the old term: got %d", n)
	}
	// delete → empty
	if err := st.ExecDDL(ctx, []string{"DELETE FROM kb_pages WHERE id=1"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if n := ftsMatchCount(t, st, "kb_pages_fts", "bigger"); n != 0 {
		t.Fatalf("delete must sync the FTS: got %d", n)
	}

	// the same unit renders the code-index shapes (chunks_fts/symbol_fts)
	chunkDDL := FTSDDL("chunks_fts", "chunks", "id", []string{"text"}, "trigram")
	if len(chunkDDL) != 5 {
		t.Fatalf("code-index chunks_fts unit: %d", len(chunkDDL))
	}
	symDDL := FTSDDL("symbol_fts", "symbols", "id",
		[]string{"name", "qualified_name", "signature", "kind", "language"}, "trigram")
	if len(symDDL) != 5 {
		t.Fatalf("code-index symbol_fts unit: %d", len(symDDL))
	}
}

// [fts-stays-in-sync-code] the code-index FTS legs (chunks_fts/symbol_fts)
// created by migration 100 stay in sync with their content tables — the
// unit's sync contract exercised on the real migration tables (the
// scenario's exact shape: trigram → update to Bigger trigrams).
func TestFTSStaysInSyncCode(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	fid, err := st.UpsertFile(ctx, FileRow{
		Path: "sync/s.go", ContentHash: "h", Language: "go",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// a chunk whose text mentions trigram recall
	if err := st.ReplaceChunksForFile(ctx, fid, []Chunk{{
		Kind: "ast", StartLine: 1, EndLine: 5,
		Text: "trigram recall beats word splitting", ContentHash: "c1",
	}}); err != nil {
		t.Fatalf("chunk: %v", err)
	}
	if n := ftsMatchCount(t, st, "chunks_fts", "trigram"); n != 1 {
		t.Fatalf("the chunk is the only hit: got %d", n)
	}
	// update the chunk text to Bigger trigrams — the BEFORE UPDATE trigger
	// must remove the old FTS row before the AFTER UPDATE indexes the new one
	cid, err := rawQueryInt(t, st, `SELECT id FROM chunks WHERE file_id = ? ORDER BY id LIMIT 1`, fid)
	if err != nil {
		t.Fatalf("chunk id: %v", err)
	}
	if err := st.ExecDDL(ctx, []string{
		"UPDATE chunks SET text='Bigger trigrams' WHERE id=" + fmt.Sprint(cid),
	}); err != nil {
		t.Fatalf("update chunk: %v", err)
	}
	if n := ftsMatchCount(t, st, "chunks_fts", "Bigger"); n != 1 {
		t.Fatalf("Bigger returns it: got %d", n)
	}
	if n := ftsMatchCount(t, st, "chunks_fts", "recall"); n != 0 {
		t.Fatalf("the old term (recall) must stop: got %d", n)
	}
	// and the symbol leg
	if _, err := st.UpsertSymbols(ctx, fid, []Symbol{
		{Name: "CapturePayment", Kind: "method", UID: "s.go:CapturePayment:method"},
	}); err != nil {
		t.Fatalf("symbol: %v", err)
	}
	if n := ftsMatchCount(t, st, "symbol_fts", "CapturePayment"); n != 1 {
		t.Fatalf("symbol_fts must find the symbol: got %d", n)
	}
}

// [short-term-like-fallback-code] a symbol named Go (a 2-rune name) must
// still be findable via the substring (LIKE) fallback below the trigram
// minimum — the memory change's short-term LIKE pattern applied to the
// code-index reads. The FTS5 trigram tokenizer cannot match < 3-rune terms,
// so a bare FTS MATCH would miss it; the fallback does the substring scan.
func TestShortTermLikeFallbackCode(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	fid, err := st.UpsertFile(ctx, FileRow{Path: "like/l.go", ContentHash: "h", Language: "go"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fid, []Symbol{
		{Name: "Go", Kind: "function", UID: "l.go:Go:function"},
		{Name: "Gopher", Kind: "function", UID: "l.go:Gopher:function"},
	}); err != nil {
		t.Fatalf("symbols: %v", err)
	}
	// the 2-rune term "go" is below the trigram minimum — a bare FTS MATCH
	// would return nothing (trigram needs 3+ runes)
	if n := ftsMatchCount(t, st, "symbol_fts", "go"); n != 0 {
		t.Fatalf("precondition: bare FTS MATCH must miss a < 3-rune term, got %d", n)
	}
	// the LIKE fallback finds the symbol named Go by substring
	if n := likeCount(t, st, "symbols", "name", "go"); n < 1 {
		t.Fatalf("LIKE fallback must find the symbol named Go: got %d", n)
	}
}

// likeCount runs the short-term substring (LIKE) fallback over a content
// column — the < 3-rune term path the memory change's searchLIKE uses. It
// is the code-index read-side shape (Task 5's fts_leg builds the fused
// leg; the parity + fallback tests pin the fallback here).
func likeCount(t *testing.T, st Store, table, column, term string) int {
	t.Helper()
	rows, err := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	}).RawQuery(`SELECT COUNT(*) FROM `+table+` WHERE `+column+` LIKE ? ESCAPE '\'`, "%"+term+"%")
	if err != nil {
		t.Fatalf("like %s.%s: %v", table, column, err)
	}
	n, ok := rows[0][0].(int64)
	if !ok {
		t.Fatalf("like %s.%s: bad type %T", table, column, rows[0][0])
	}
	return int(n)
}

