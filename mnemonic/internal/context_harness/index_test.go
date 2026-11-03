package context_harness

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// openCtxStore opens a fresh store under a temp dir for the context harness
// tests (mirrors the migrate_053/054 test idiom).
func openCtxStore(t *testing.T, project string) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// indexedCount returns the number of indexed_files rows for project.
func indexedCount(t *testing.T, st *store.Store, project string) int {
	t.Helper()
	var n int
	if err := st.DB.QueryRow(`SELECT count(*) FROM indexed_files WHERE project = ?`, project).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// TestIndexFile_ChunksAndUpserts writes a temp file of ~5KB of distinct
// line-separated content, indexes it, and asserts: (1) a positive chunk count
// matching the row count, (2) FTS MATCH finds a token from the file, (3)
// re-indexing the same file is idempotent (row count does not grow).
func TestIndexFile_ChunksAndUpserts(t *testing.T) {
	st := openCtxStore(t, "testproj")

	// Build ~5KB of distinct, line-separated content. Line i mentions a
	// distinctive token (chunktoken-<i>) so FTS can find any line by token.
	var sb []string
	total := 0
	for i := 0; total < 5200; i++ {
		line := fmt.Sprintf("chunktoken-%04d line %d with some filler text to pad the chunk body", i, i)
		sb = append(sb, line)
		total += len(line) + 1
	}
	path := filepath.Join(t.TempDir(), "sampledoc.md")
	data := ""
	for _, l := range sb {
		data += l + "\n"
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	n, err := IndexFile(st.DB, "testproj", path)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if n <= 0 {
		t.Fatalf("expected >0 chunks, got %d", n)
	}
	if got := indexedCount(t, st, "testproj"); got != n {
		t.Fatalf("expected %d indexed_files rows, got %d", n, got)
	}

	// FTS MATCH on a distinctive token from the file must find it.
	// Note: the porter tokenizer splits on hyphens, so "chunktoken-0007" is
	// stored as two tokens "chunktoken" and "0007". We match the "chunktoken"
	// stem which appears in every line.
	var c int
	if err := st.DB.QueryRow(`SELECT count(*) FROM indexed_files_fts WHERE indexed_files_fts MATCH ?`, `"chunktoken"`).Scan(&c); err != nil {
		t.Fatalf("fts match: %v", err)
	}
	if c == 0 {
		t.Fatalf("expected >0 fts hits for %q, got 0", "chunktoken")
	}

	// Idempotent upsert: re-indexing the same file must not grow the row count.
	n2, err := IndexFile(st.DB, "testproj", path)
	if err != nil {
		t.Fatalf("re-index: %v", err)
	}
	if n2 != n {
		t.Fatalf("re-index returned %d chunks, want %d (same content)", n2, n)
	}
	if got := indexedCount(t, st, "testproj"); got != n {
		t.Fatalf("re-index grew row count to %d, want %d (idempotent upsert expected)", got, n)
	}
}

// TestPurgeExpiredIndexed seeds one indexed_files row 8 days old and one from
// now, purges with a 7-day TTL, and asserts the old row is gone, the new one
// remains, and the returned count is 1.
func TestPurgeExpiredIndexed(t *testing.T) {
	st := openCtxStore(t, "testproj")
	now := time.Now()

	oldTS := now.Add(-8 * 24 * time.Hour).UTC().Format(time.RFC3339)
	newTS := now.UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, start_line, end_line, content, created_at)
		VALUES ('testproj', 'oldfile.go', 'file', 'go', 1, 10, 'old content', ?)`, oldTS); err != nil {
		t.Fatalf("insert old: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, start_line, end_line, content, created_at)
		VALUES ('testproj', 'newfile.go', 'file', 'go', 1, 10, 'new content', ?)`, newTS); err != nil {
		t.Fatalf("insert new: %v", err)
	}

	deleted, err := PurgeExpiredIndexed(st.DB, "testproj", now, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 purged row, got %d", deleted)
	}
	var oldN, newN int
	if err := st.DB.QueryRow(`SELECT count(*) FROM indexed_files WHERE name = 'oldfile.go'`).Scan(&oldN); err != nil {
		t.Fatalf("count old: %v", err)
	}
	if oldN != 0 {
		t.Fatalf("old row should be gone, count=%d", oldN)
	}
	if err := st.DB.QueryRow(`SELECT count(*) FROM indexed_files WHERE name = 'newfile.go'`).Scan(&newN); err != nil {
		t.Fatalf("count new: %v", err)
	}
	if newN != 1 {
		t.Fatalf("new row should remain, count=%d", newN)
	}
}

// TestCtxQuery_Provenance confirms CtxQuery carries per-leg provenance: a
// sandbox-only hit has Legs=["sandbox"] with Path=tool name; an indexed-only
// hit has Legs=["indexed"] with Path=file name and StartLine/EndLine from the
// row; a cross-leg token (same candidate name matched by both legs) carries
// Legs containing both "sandbox" and "indexed".
func TestCtxQuery_Provenance(t *testing.T) {
	st := openCtxStore(t, "testproj")

	// Indexed leg: a file whose content carries both the indexed-only token
	// and the cross-leg token. start_line/end_line are set so provenance can
	// be asserted from the row.
	if _, err := st.DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, start_line, end_line, content, created_at)
		VALUES ('testproj', 'provenancefile.go', 'file', 'go', 10, 22, 'provenance-indexed-token and crossleg-token', datetime('now'))`); err != nil {
		t.Fatalf("insert indexed: %v", err)
	}
	// Sandbox leg: a tool whose name IS the cross-leg candidate and whose
	// output carries both the sandbox-only token and the cross-leg token.
	insertToolOutput(t, st, "crossleg", "provenance-sandbox-token and crossleg-token")

	// Indexed-only token: must yield Legs=["indexed"], Path=file, lines 10-22.
	rs, err := CtxQuery(st.DB, "provenance-indexed-token", 20)
	if err != nil {
		t.Fatalf("query indexed-only: %v", err)
	}
	if len(rs) != 1 {
		t.Fatalf("expected 1 result, got %d", len(rs))
	}
	r := rs[0]
	if len(r.Legs) != 1 || r.Legs[0] != sourceIndexed {
		t.Fatalf("indexed-only Legs = %v, want [indexed]", r.Legs)
	}
	if r.Path != "provenancefile.go" {
		t.Fatalf("indexed-only Path = %q, want provenancefile.go", r.Path)
	}
	if r.StartLine != 10 || r.EndLine != 22 {
		t.Fatalf("indexed-only lines = %d-%d, want 10-22", r.StartLine, r.EndLine)
	}
	if r.Source != sourceIndexed {
		t.Fatalf("indexed-only Source = %q, want indexed", r.Source)
	}

	// Sandbox-only token: must yield Legs=["sandbox"], Path=tool name.
	rs, err = CtxQuery(st.DB, "provenance-sandbox-token", 20)
	if err != nil {
		t.Fatalf("query sandbox-only: %v", err)
	}
	if len(rs) != 1 {
		t.Fatalf("expected 1 result, got %d", len(rs))
	}
	r = rs[0]
	if len(r.Legs) != 1 || r.Legs[0] != sourceSandbox {
		t.Fatalf("sandbox-only Legs = %v, want [sandbox]", r.Legs)
	}
	if r.Path != "crossleg" {
		t.Fatalf("sandbox-only Path = %q, want crossleg (tool name)", r.Path)
	}
	if r.Source != sourceSandbox {
		t.Fatalf("sandbox-only Source = %q, want sandbox", r.Source)
	}

	// Cross-leg token: the sandbox tool is named "crossleg" and the indexed
	// file content also contains "crossleg-token" (but the file NAME is
	// provenancefile.go, so the fusion keys differ — the cross-leg fusion
	// happens on candidate identity, i.e. name. To exercise a true cross-leg
	// hit we need the SAME candidate key matched by both legs. The sandbox
	// tool name is "crossleg"; insert an indexed row named "crossleg" too.
	if _, err := st.DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, start_line, end_line, content, created_at)
		VALUES ('testproj', 'crossleg', 'file', 'go', 5, 9, 'crossleg-token indexed side', datetime('now'))`); err != nil {
		t.Fatalf("insert indexed crossleg: %v", err)
	}
	rs, err = CtxQuery(st.DB, "crossleg", 20)
	if err != nil {
		t.Fatalf("query crossleg: %v", err)
	}
	// Find the candidate named "crossleg" (the fused cross-leg hit).
	var got *CtxResult
	for i := range rs {
		if rs[i].Name == "crossleg" {
			got = &rs[i]
		}
	}
	if got == nil {
		t.Fatalf("no fused result named crossleg, got %+v", rs)
	}
	hasSandbox, hasIndexed := false, false
	for _, l := range got.Legs {
		if l == sourceSandbox {
			hasSandbox = true
		}
		if l == sourceIndexed {
			hasIndexed = true
		}
	}
	if !hasSandbox || !hasIndexed {
		t.Fatalf("cross-leg Legs = %v, want both sandbox and indexed", got.Legs)
	}
}
