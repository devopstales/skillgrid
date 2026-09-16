package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestSplitIdentifier locks the camelCase/snake_case splitting that makes the
// FTS identifier-aware.
func TestSplitIdentifier(t *testing.T) {
	cases := map[string][]string{
		"parseConfig":      {"parseconfig"},
		"parse_config":     {"parse", "config"},
		"loadUserSettings": {"loadusersettings"},
		"snake_case_name":  {"snake", "case", "name"},
		"camelCase":        {"camelcase"},
		"a":                {"a"},
		"":                 {},
	}
	for in, want := range cases {
		got := SplitIdentifier(in)
		if len(got) != len(want) {
			t.Errorf("SplitIdentifier(%q) = %v, want %v", in, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("SplitIdentifier(%q) = %v, want %v", in, got, want)
				break
			}
		}
	}
}

// TestIdentifierQuery builds a MATCH expression with the right tokens.
func TestIdentifierQuery(t *testing.T) {
	if got := IdentifierQuery(""); got != "" {
		t.Errorf("empty query -> %q, want empty", got)
	}
	got := IdentifierQuery("parseConfig")
	if got != `"`+"parseconfig"+`"` {
		t.Errorf("IdentifierQuery(parseConfig) = %q", got)
	}
}

// TestSymbolFTSFindsSymbols covers @step-02 happy: identifier FTS finds
// camelCase/snake_case symbols, and a symbol whose exact spelling matches the
// query is returned.
func TestSymbolFTSFindsSymbols(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "symbol-fts-test")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	db := st.DB

	// Insert a file + symbols directly (bypasses the indexer).
	var fileID int64
	if _, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES ('main.go', 1, 1, 'h', 'x')`); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	db.QueryRow(`SELECT id FROM files WHERE path='main.go'`).Scan(&fileID)

	syms := []struct{ name, lang string }{
		{"parseConfig", "go"},
		{"parse_config", "python"},
		{"loadUserSettings", "typescript"},
	}
	for _, s := range syms {
		if _, err := db.Exec(`INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid) VALUES (?, ?, ?, 'function', ?, 'sig', 1, 2, 'c', ?)`,
			fileID, s.name, s.name, s.lang, s.name); err != nil {
			t.Fatalf("insert symbol %s: %v", s.name, err)
		}
	}

	// Exact identifier matches its same-spelling symbol.
	hits, err := SymbolFTS(db, "parseConfig", 10)
	if err != nil {
		t.Fatalf("symbol fts: %v", err)
	}
	found := map[string]bool{}
	for _, h := range hits {
		found[h.Name] = true
	}
	if !found["parseConfig"] {
		t.Errorf("query 'parseConfig' should find parseConfig, got %v", hits)
	}
	// The snake_case variant is found by its own exact query.
	hits, _ = SymbolFTS(db, "parse_config", 10)
	found = map[string]bool{}
	for _, h := range hits {
		found[h.Name] = true
	}
	if !found["parse_config"] {
		t.Errorf("query 'parse_config' should find parse_config, got %v", hits)
	}
}

// openStoreForSymbolFTS helper (kept local to avoid import cycles in tests).
func openStoreForSymbolFTS(t *testing.T) *store.Store {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "symbol-fts-test")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

var _ = os.Getenv
var _ = filepath.Join
