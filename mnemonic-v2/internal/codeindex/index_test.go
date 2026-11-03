package codeindex

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// [door-check-code] M6 choice (a): the door exercises the REAL path —
// Index scans a 2-file temp tree, writes through the extended Store in
// per-file txs (K1), prunes orphans, and records index_freshness. At Task 1
// time the extract hook is a test double (Task 3 wires the real ladder). If
// the floor does not find the indexed symbols, the foundation does not hold.
func TestDoorCheckEndToEndThroughDriver(t *testing.T) {
	root := t.TempDir()
	writeTree := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeTree("pay.go", "package pay\n\nfunc CapturePayment() {}\n")
	writeTree("billing.go", "package billing\n\nfunc ApplyDiscount() {}\n")

	dir := t.TempDir()
	st, err := store.Open(context.Background(), dir, "proj-a", store.SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	rep, err := Index(context.Background(), root, st, IndexOpts{
		Project: "proj-a",
		ExtractFn: func(path, content string) (store.FileGraph, error) {
			return fakeExtract(path), nil
		},
	})
	if err != nil {
		t.Fatalf("index run: %v", err)
	}
	if rep.FilesIndexed != 2 {
		t.Fatalf("both files must be indexed: %+v", rep)
	}

	ctx := context.Background()
	// the name floor finds the indexed symbol
	rows, err := st.SymbolByName(ctx, "CapturePayment")
	if err != nil || len(rows) != 1 {
		t.Fatalf("driver-written symbol not findable by name: %+v (%v)", rows, err)
	}
	if !regexp.MustCompile(`^sym-[0-9a-f]{16}$`).MatchString(rows[0].SyncID) {
		t.Fatalf("driver symbol sync id format: %q", rows[0].SyncID)
	}
	// the FTS floors find what the driver wrote (B1 — live FTS, not a string)
	if n := ftsCount(t, st, "symbol_fts", "CapturePayment"); n < 1 {
		t.Fatalf("symbol_fts must match the indexed symbol name: got %d", n)
	}
	if n := ftsCount(t, st, "chunks_fts", "discount"); n < 1 {
		t.Fatalf("chunks_fts must match the indexed chunk text: got %d", n)
	}
	// the name-only edge + the unresolved ledger (CI-INV-3)
	nameOnly, err := st.EdgesByNameOnlyToName(ctx, "ApplyDiscount", nil)
	if err != nil || len(nameOnly) != 1 || nameOnly[0].ToID != nil {
		t.Fatalf("driver name-only edge broken: %+v (%v)", nameOnly, err)
	}
	// the freshness record landed (the 4-state model's data source)
	raw := st.(interface {
		RawQuery(q string, args ...any) ([][]any, error)
	})
	fresh, err := raw.RawQuery(`SELECT COUNT(*) FROM index_freshness WHERE project = 'proj-a'`)
	if err != nil {
		t.Fatalf("freshness query: %v", err)
	}
	if n, ok := fresh[0][0].(int64); !ok || n != 1 {
		t.Fatalf("index_freshness must hold the run record: %+v", fresh)
	}
}

// a second run over an unchanged tree is an idempotent no-op (T1: same
// content_hash → the graph is identical, the freshness record reflects it).
func TestIndexIdempotentReRun(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pay.go"), []byte("package pay\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	dir := t.TempDir()
	st, err := store.Open(context.Background(), dir, "proj-a", store.SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	extract := func(path, content string) (store.FileGraph, error) {
		return store.FileGraph{Symbols: []store.Symbol{
			{Name: "Pay", Kind: "function", UID: "pay.go:Pay:function"},
		}}, nil
	}
	if _, err := Index(context.Background(), root, st, IndexOpts{Project: "proj-a", ExtractFn: extract}); err != nil {
		t.Fatalf("index 1: %v", err)
	}
	rep, err := Index(context.Background(), root, st, IndexOpts{Project: "proj-a", ExtractFn: extract})
	if err != nil {
		t.Fatalf("index 2: %v", err)
	}
	if rep.FilesIndexed != 0 {
		t.Fatalf("unchanged tree must be a no-op run: %+v", rep)
	}
	rows, err := st.SymbolByName(context.Background(), "Pay")
	if err != nil || len(rows) != 1 {
		t.Fatalf("symbol must survive the idempotent re-run: %+v (%v)", rows, err)
	}
	// and a file that disappeared on disk is deleted (M10 — absent = delete)
	if err := os.Remove(filepath.Join(root, "pay.go")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	rep3, err := Index(context.Background(), root, st, IndexOpts{Project: "proj-a", ExtractFn: extract})
	if err != nil {
		t.Fatalf("index 3: %v", err)
	}
	if rep3.FilesDeleted != 1 {
		t.Fatalf("absent file must be deleted (M10): %+v", rep3)
	}
	if _, err := st.FileByPath(context.Background(), "pay.go"); err == nil {
		t.Fatalf("files row must be gone after M10 delete")
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

// fakeExtract is the Task 1 test double (Task 3 replaces it with the real
// ladder): pay.go carries the CapturePayment method + a name-only call edge
// into billing.go's ApplyDiscount.
func fakeExtract(path string) store.FileGraph {
	base := filepath.Base(path)
	switch base {
	case "pay.go":
		return store.FileGraph{
			Symbols: []store.Symbol{
				{Name: "CapturePayment", QualifiedName: "pay.CapturePayment",
					Kind: "method", Language: "go", StartLine: 3, EndLine: 3,
					ContentHash: "cp", UID: "pay.go:CapturePayment:method"},
			},
			Edges: []store.Edge{
				{Kind: "calls", ToName: "ApplyDiscount", Confidence: "AMBIGUOUS", Line: 3},
			},
			Unresolved: []store.UnresolvedRef{
				{Reference: "ApplyDiscount", Kind: "calls", Line: 3},
			},
			Chunks: []store.Chunk{
				{Kind: "ast", StartLine: 3, EndLine: 3,
					Text: "CapturePayment applies the discount then charges"},
			},
		}
	case "billing.go":
		return store.FileGraph{
			Symbols: []store.Symbol{
				{Name: "ApplyDiscount", QualifiedName: "billing.ApplyDiscount",
					Kind: "function", Language: "go", StartLine: 3, EndLine: 3,
					ContentHash: "ad", UID: "billing.go:ApplyDiscount:function"},
			},
			Chunks: []store.Chunk{
				{Kind: "ast", StartLine: 3, EndLine: 3,
					Text: "ApplyDiscount computes the discount amount"},
			},
		}
	}
	return store.FileGraph{}
}

func ftsCount(t *testing.T, st store.Store, table, term string) int {
	t.Helper()
	raw := st.(interface {
		RawQuery(q string, args ...any) ([][]any, error)
	})
	rows, err := raw.RawQuery(`SELECT COUNT(*) FROM `+table+` WHERE `+table+` MATCH ?`, term)
	if err != nil {
		t.Fatalf("fts match %s: %v", table, err)
	}
	n, ok := rows[0][0].(int64)
	if !ok {
		t.Fatalf("fts match %s: bad type %T", table, rows[0][0])
	}
	return int(n)
}

// [index-progress-callback] the driver must surface live progress during a
// run (the v1 index streamed per-file progress; v2 was silent, so a 2-minute
// re-index looked stuck). IndexOpts.Progress, when non-nil, is called once per
// indexed file with (done, total, path) — done counts the files committed this
// run, total is the eligible-file count, path is the repo-relative path just
// written. A nil callback (the default, the MCP + watch surfaces) is silent.
// Oversized / unchanged (incremental no-op) files are NOT indexed this run, so
// they do not advance `done` — the progress reflects files actually written.
func TestIndexProgressCallbackFiresPerIndexedFile(t *testing.T) {
	root := t.TempDir()
	writeTree := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeTree("pay.go", "package pay\n\nfunc CapturePayment() {}\n")
	writeTree("billing.go", "package billing\n\nfunc ApplyDiscount() {}\n")

	dir := t.TempDir()
	st, err := store.Open(context.Background(), dir, "proj-p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	var calls []struct{ done, total int; path string }
	rep, err := Index(context.Background(), root, st, IndexOpts{
		Project:   "proj-p",
		ExtractFn: func(path, content string) (store.FileGraph, error) { return fakeExtract(path), nil },
		Progress: func(done, total int, path string) {
			calls = append(calls, struct{ done, total int; path string }{done, total, path})
		},
	})
	if err != nil {
		t.Fatalf("index run: %v", err)
	}
	if rep.FilesIndexed != 2 {
		t.Fatalf("both files must be indexed: %+v", rep)
	}
	// exactly one callback per indexed file, in deterministic (sorted) order,
	// with done advancing 1..N and total == the eligible-file count (2 here).
	if len(calls) != 2 {
		t.Fatalf("Progress must fire once per indexed file: got %d calls %+v", len(calls), calls)
	}
	if calls[0].total != 2 || calls[1].total != 2 {
		t.Fatalf("Progress.total must be the eligible-file count 2: %+v", calls)
	}
	if calls[0].done != 1 || calls[1].done != 2 {
		t.Fatalf("Progress.done must advance 1,2: %+v", calls)
	}
	// deterministic walk order (sort.Strings on the rels) → billing.go then pay.go
	if calls[0].path != "billing.go" || calls[1].path != "pay.go" {
		t.Fatalf("Progress paths must follow the deterministic walk order: %+v", calls)
	}
}

// [index-progress-nil-default] a nil Progress (the default — the MCP tool and
// the watch loop do not stream) must not panic and must behave identically to
// a run with no progress wiring (the silent floor).
func TestIndexProgressNilCallbackIsSilent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pay.go"), []byte("package pay\n\nfunc CapturePayment() {}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	dir := t.TempDir()
	st, err := store.Open(context.Background(), dir, "proj-n", store.SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	// nil Progress: must not panic, must still index.
	rep, err := Index(context.Background(), root, st, IndexOpts{
		Project:   "proj-n",
		ExtractFn: func(path, content string) (store.FileGraph, error) { return fakeExtract(path), nil },
	})
	if err != nil {
		t.Fatalf("index run (nil progress): %v", err)
	}
	if rep.FilesIndexed != 1 {
		t.Fatalf("file must be indexed with nil Progress: %+v", rep)
	}
}
