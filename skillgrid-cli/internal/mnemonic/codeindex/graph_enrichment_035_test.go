package codeindex

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/extract"
)

// TestDynamicImportEdgeKind covers 035's dynamic_import kind at the extractor
// (where the import() conversion happens): a TS import() call is a dynamic
// module load → a 'dynamic_import' edge whose target is the module path
// (import('./mod') -> './mod'), and a static Go import declaration → an
// 'imports' edge (the gotreesitter lib surfaces Go import declarations but not
// JS/TS import statements, so the static side is demonstrated in Go). The
// indexer-level dangling-edge prune drops name-only import edges (no matching
// symbol), so this asserts on the extractor directly.
func TestDynamicImportEdgeKind(t *testing.T) {
	ex := extract.Default()

	tsG, err := ex.ExtractFile("main.ts", []byte(
		"export async function load() { const m = await import('./mod'); return m; }\n"))
	if err != nil {
		t.Fatalf("extract ts: %v", err)
	}
	var dynCount int
	var dynTarget string
	for _, e := range tsG.Edges {
		if e.Kind == "dynamic_import" {
			dynCount++
			dynTarget = e.TargetPath
		}
	}
	if dynCount < 1 {
		t.Errorf("expected >=1 dynamic_import edge from the TS import() call, got %d", dynCount)
	}
	if dynTarget != "./mod" {
		t.Errorf("expected the dynamic import target to be the module path './mod', got %q", dynTarget)
	}

	goG, err := ex.ExtractFile("main.go", []byte(
		"package main\n\nimport \"math\"\n\nfunc main() { math.MaxInt }\n"))
	if err != nil {
		t.Fatalf("extract go: %v", err)
	}
	var statCount int
	for _, e := range goG.Edges {
		if e.Kind == "imports" {
			statCount++
		}
	}
	if statCount < 1 {
		t.Errorf("expected >=1 imports edge from the static Go import, got %d", statCount)
	}
}

// TestEdgeContextAndConfidenceScorePersisted covers 035: an EXTRACTED call
// edge written by the indexer persists confidence_score=1.0 (the numeric form
// of its categorical label) into the new edges columns.
func TestEdgeContextAndConfidenceScorePersisted(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()

	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n\nfunc main() {\n\tgreet()\n}\n\nfunc greet() {}\n")
	if _, err := idx.Run(context.Background(), root, testCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	var conf, ctx string
	var score float64
	if err := db.QueryRow(`SELECT confidence, context, confidence_score FROM edges WHERE kind = 'calls' AND to_name = 'greet' LIMIT 1`).
		Scan(&conf, &ctx, &score); err != nil {
		t.Fatalf("read call edge: %v", err)
	}
	if conf != "EXTRACTED" {
		t.Errorf("expected call edge confidence EXTRACTED, got %q", conf)
	}
	if score != 1.0 {
		t.Errorf("expected EXTRACTED confidence_score 1.0, got %v", score)
	}
	_ = ctx // '' by default for call edges; the column exists (the scan proved it)
}

// TestAstHashStructureChange covers 035's structure hash: files.ast_hash is
// non-empty after indexing, and editing a symbol's declaration (a comment above
// greet() shifts its span, changing its UID) CHANGES ast_hash — structure UIDs
// are hashed, so a real structure change is detected. A pure comment-inside-
// body edit (no span/line change) would keep the same hash.
func TestAstHashStructureChange(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()

	root := t.TempDir()
	p := filepath.Join(root, "main.go")
	mustWrite(t, p, "package main\n\nfunc main() {\n\tgreet()\n}\n\nfunc greet() {}\n")
	if _, err := idx.Run(context.Background(), root, testCfg); err != nil {
		t.Fatalf("first run: %v", err)
	}
	hash1 := astHashFor(t, idx.store.DB, "main.go")
	if hash1 == "" {
		t.Fatalf("expected a non-empty files.ast_hash after indexing")
	}

	// A comment above greet() shifts its span (start_line 7 -> 8), changing its
	// UID — a real structure change. The content bytes also differ, so the file
	// is re-indexed.
	mustWrite(t, p, "package main\n\nfunc main() {\n\tgreet()\n}\n\n// greet says hi\nfunc greet() {}\n")
	if err := os.Chtimes(p, nowPlus(), nowPlus()); err != nil {
		t.Fatalf("touch: %v", err)
	}
	stats, err := idx.Run(context.Background(), root, testCfg)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if stats.FilesIndexed != 1 {
		t.Fatalf("expected the edited file to be re-indexed, got %d indexed", stats.FilesIndexed)
	}
	hash2 := astHashFor(t, idx.store.DB, "main.go")
	if hash2 == hash1 {
		t.Errorf("ast_hash should change when a symbol's span/UID changes: %q", hash1)
	}
}

// astHashFor reads files.ast_hash for a path.
func astHashFor(t *testing.T, db *sql.DB, path string) string {
	t.Helper()
	var h string
	if err := db.QueryRow(`SELECT ast_hash FROM files WHERE path = ?`, path).Scan(&h); err != nil {
		t.Fatalf("read ast_hash for %s: %v", path, err)
	}
	return h
}

// nowPlus returns a time strictly after the first run's recorded mtime.
func nowPlus() time.Time {
	return time.Now().Add(2 * time.Second)
}
