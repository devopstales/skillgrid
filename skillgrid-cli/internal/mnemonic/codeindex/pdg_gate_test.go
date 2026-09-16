package codeindex

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"strings"
	"testing"
)

// pdgCfg indexes Go/JS/TS (the 005-supported languages the fixture uses).
var pdgCfg = Config{
	Include:      []string{"**/*.go", "**/*.js", "**/*.ts", "**/*.tsx"},
	Exclude:      []string{"**/node_modules/**", "**/.git/**"},
	ChunkLines:   80,
	ChunkOverlap: 10,
}

// writePdgFixture returns a Go repo with branch/loop/return structure and a
// data flow, suitable for the per-function CFG/PDG. The content is fixed so the
// 005 baseline fingerprint is stable across runs.
func writePdgFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n\nfunc main() {\n\tn := add(1, 2)\n\tif n > 3 {\n\t\twarn(n)\n\t} else {\n\t\tok()\n\t}\n\tfor i := 0; i < n; i++ {\n\t\tlog(i)\n\t}\n\tswitch n {\n\tcase 1:\n\t\ta()\n\tdefault:\n\t\tb()\n\t}\n\treturn\n}\n\nfunc add(a, b int) int {\n\treturn a + b\n}\n\nfunc warn(n int) {\n\tm := n * 2\n\t_ = m\n}\n\nfunc ok() {}\n\nfunc log(i int) {}\n\nfunc a() {}\n\nfunc b() {}\n")
	return root
}

// pdgTablesEmpty asserts the four 011 tables carry no rows on a non---pdg
// index (created-but-empty; a store predating the migration has them absent,
// which also satisfies "no PDG/taint rows").
func pdgTablesEmpty(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"cfg_blocks", "cfg_edges", "pdg_edges", "taint_findings"} {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n)
		if err != nil {
			if strings.Contains(err.Error(), "no such table") {
				continue
			}
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Errorf("table %s has %d rows, want 0 (non---pdg index)", table, n)
		}
	}
}

// baselineFingerprint hashes the 005 common graph (files/symbols/edges/chunks)
// deterministically into a single string so a byte-for-byte comparison is one
// string compare. It hashes content only (no id/wall-clock) so the value is
// stable across independent index runs of the same fixture.
func baselineFingerprint(t *testing.T, db *sql.DB) string {
	t.Helper()
	sb := baselineFingerprintRows(t, db)
	h := fnv.New64a()
	h.Write([]byte(sb.String()))
	return fmt.Sprintf("%x", h.Sum64())
}

func baselineFingerprintRows(t *testing.T, db *sql.DB) *strings.Builder {
	t.Helper()
	var sb strings.Builder
	for _, q := range []string{
		// Content-only (no id / no wall-clock mtime_ns) so the fingerprint is
		// stable across independent index runs of the same fixture.
		`SELECT path, size, content_hash FROM files ORDER BY path`,
		`SELECT name, kind, language, COALESCE(signature,''), start_line, end_line, content_hash, uid FROM symbols ORDER BY uid`,
		`SELECT kind, COALESCE(from_id,-1), COALESCE(file_id,-1), COALESCE(to_id,-1), COALESCE(to_name,''), COALESCE(target_path,''), confidence, COALESCE(line,0) FROM edges ORDER BY kind, from_id, to_id, to_name, line`,
		`SELECT COALESCE(file_id,-1), start_line, end_line, content_hash FROM chunks ORDER BY file_id, start_line`,
	} {
		rows, err := db.Query(q)
		if err != nil {
			t.Fatalf("query %q: %v", q, err)
		}
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				t.Fatalf("scan %q: %v", q, err)
			}
			sb.WriteString(rowsString(vals))
			sb.WriteByte('\n')
		}
		rows.Close()
		sb.WriteByte('\x1e')
	}
	return &sb
}

// rowsString renders a scanned row deterministically (ints verbatim, strings
// quoted) so the fingerprint is stable and comparable.
func rowsString(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		switch x := v.(type) {
		case int64:
			parts[i] = fmt.Sprintf("%d", x)
		case []byte:
			parts[i] = `"` + string(x) + `"`
		case string:
			parts[i] = `"` + x + `"`
		case nil:
			parts[i] = "NULL"
		default:
			parts[i] = fmt.Sprintf("%v", x)
		}
	}
	return strings.Join(parts, "¦")
}

// wantPdgBaseline is the captured 005 baseline fingerprint for
// writePdgFixture under pdgCfg (005 extraction only, no opt-in pass). It is the
// byte-for-byte target for TestOptInIsolation. Regenerate with
// TestCapturePdgBaseline if the 005 extractor or fixture changes. (037:
// re-captured — AST-boundary chunking changes the chunk rows for Go fixtures,
// which is the intended semantic-tier behavior.)
var wantPdgBaseline = "c904ebd2c5b19dc7"

// TestCapturePdgBaseline prints the current 005 baseline fingerprint for the
// pdg fixture. Run it once (go test -run TestCapturePdgBaseline -v) and paste
// the value into wantPdgBaseline. It is a generator, not an assertion.
func TestCapturePdgBaseline(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := writePdgFixture(t)
	if _, err := idx.Run(context.Background(), root, pdgCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	if testing.Verbose() {
		t.Logf("ROWS:\n%s", baselineFingerprintRows(t, idx.store.DB).String())
	}
	t.Logf("wantPdgBaseline = %q", baselineFingerprint(t, idx.store.DB))
}

// TestOptInIsolation covers @step-01 (Scenario: Non-opt-in index is
// byte-for-byte unchanged): a non---pdg index (1) creates but leaves empty the
// cfg_blocks/cfg_edges/pdg_edges/taint_findings tables, and (2) produces a 005
// graph byte-for-byte identical to the pre-011 baseline captured at build time.
// The PDG/taint pass is gated on the ---pdg flag, so without it the common
// graph is untouched.
func TestOptInIsolation(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()

	root := writePdgFixture(t)
	if _, err := idx.Run(context.Background(), root, pdgCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	// (1) The 011 tables are created but empty on a non---pdg index.
	pdgTablesEmpty(t, db)

	// (2) The 005 graph is byte-for-byte the pre-011 baseline.
	got := baselineFingerprint(t, db)
	if got != wantPdgBaseline {
		t.Errorf("005 graph not byte-for-byte identical to the pre-011 baseline:\n got %s\nwant %s", got, wantPdgBaseline)
	}
}

// fnvString is a tiny FNV-1a helper (used by other pdg tests for stable
// string keys; kept here to avoid importing a hash util in every test file).
func fnvString(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
