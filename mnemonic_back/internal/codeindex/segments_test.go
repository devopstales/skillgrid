package codeindex

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

// TestNameSegments locks the segment-splitting rule: underscores, case
// transitions, consecutive-uppercase runs (trailing word detaches), and
// letter<->digit boundaries. Single-char segments are kept.
func TestNameSegments(t *testing.T) {
	cases := []struct {
		name string
		want []string
	}{
		{"handleSessionCreate", []string{"handle", "session", "create"}},
		{"ParseConfig", []string{"parse", "config"}},
		{"parse_config", []string{"parse", "config"}},
		{"HTTPServer", []string{"http", "server"}},
		{"handleX", []string{"handle", "x"}},
		{"parse2", []string{"parse", "2"}},
		{"snake_case_name", []string{"snake", "case", "name"}},
		{"single", []string{"single"}},
		{"x", []string{"x"}},
		{"a1b2", []string{"a", "1", "b", "2"}},
		{"GETHandler", []string{"get", "handler"}},
	}
	for _, tc := range cases {
		if got := nameSegments(tc.name); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("nameSegments(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestIndexFillsSymbolSegments covers the segment vocabulary end-to-end: after
// indexing a file with a symbol, symbol_segments maps each of its segments to
// the symbol name, and re-indexing stays target-state (no duplicate rows).
func TestIndexFillsSymbolSegments(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "server.go"), `package main

type Server struct{}

func (s *Server) handleSessionCreate() {}
`)
	cfg := Config{
		Include:      []string{"**/*.go"},
		Exclude:      []string{"**/.git/**"},
		ChunkLines:   80,
		ChunkOverlap: 10,
	}
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	// Re-run: unchanged files are skipped, but the segment table must not
	// duplicate from a second full pass.
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
	db := idx.store.DB
	for _, seg := range []string{"handle", "session", "create"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM symbol_segments WHERE segment = ? AND symbol_name = 'handleSessionCreate'`, seg).Scan(&n); err != nil {
			t.Fatalf("count segment %q: %v", seg, err)
		}
		if n != 1 {
			t.Errorf("segment %q -> handleSessionCreate count = %d, want 1", seg, n)
		}
	}
}
