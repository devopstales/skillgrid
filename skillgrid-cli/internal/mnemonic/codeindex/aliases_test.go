package codeindex

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/extract"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestEntityAliasLookup resolves a case-insensitive alias to its qualified
// name, and treats an unknown alias as an empty miss.
func TestEntityAliasLookup(t *testing.T) {
	const project = "alias-proj"
	st, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	if _, err := st.DB.Exec(`
		INSERT INTO entity_aliases (project, alias, qualified_name, source)
		VALUES (?, 'the renderer', 'comp_renderer.cpp', 'index')`, project); err != nil {
		t.Fatalf("insert alias: %v", err)
	}

	got, err := LookupAliases(context.Background(), st.DB, project, "The Renderer")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(got) != 1 || got[0] != "comp_renderer.cpp" {
		t.Fatalf("lookup The Renderer = %v, want [comp_renderer.cpp]", got)
	}

	got, err = LookupAliases(context.Background(), st.DB, project, "missing")
	if err != nil {
		t.Fatalf("lookup missing: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("unknown alias returned %v, want no rows", got)
	}
}

// TestEntityAliasIndexedOnWrite inserts alias rows for the symbol name and
// the qualified name when a symbol is written.
func TestEntityAliasIndexedOnWrite(t *testing.T) {
	const project = "alias-write"
	st, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	ResetFileFirstSymbol()

	tx, err := st.DB.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	var fileID int64
	if err := tx.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('comp_renderer.cpp', 1, 1, 'h', 'x')
		RETURNING id`).Scan(&fileID); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	if _, err := writeFileGraph(tx, fileID, []extract.Symbol{{
		Name:          "Render",
		QualifiedName: "comp_renderer.cpp",
		Kind:          "function",
		Language:      "cpp",
		Signature:     "void Render()",
		StartLine:     1,
		EndLine:       4,
		ContentHash:   "abc",
		UID:           "uid-renderer",
	}}, nil, nil); err != nil {
		t.Fatalf("writeFileGraph: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	rows, err := st.DB.Query(`
		SELECT alias FROM entity_aliases
		WHERE project = ? AND qualified_name = ? AND source = 'index'
		ORDER BY alias`, project, "comp_renderer.cpp")
	if err != nil {
		t.Fatalf("select aliases: %v", err)
	}
	defer rows.Close()
	var aliases []string
	for rows.Next() {
		var alias string
		if err := rows.Scan(&alias); err != nil {
			t.Fatalf("scan: %v", err)
		}
		aliases = append(aliases, alias)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	gotAlias := map[string]bool{}
	for _, a := range aliases {
		gotAlias[a] = true
	}
	if len(aliases) != 2 || !gotAlias["Render"] || !gotAlias["comp_renderer.cpp"] {
		t.Fatalf("indexed aliases = %v, want Render and comp_renderer.cpp", aliases)
	}
}

// TestEntityAliasPrepended finds a symbol by alias before FTS, and a query
// with no alias still returns ordinary FTS hits.
func TestEntityAliasPrepended(t *testing.T) {
	const project = "alias-proj"
	st, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	if _, err := st.DB.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES ('comp_renderer.cpp', 1, 1, 'h', 'x')`); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var fileID int64
	if err := st.DB.QueryRow(`SELECT id FROM files WHERE path = 'comp_renderer.cpp'`).Scan(&fileID); err != nil {
		t.Fatalf("file id: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		VALUES (?, 'zzqxonly', 'zzqxonly', 'function', 'cpp', 'void zzqxonly()', 1, 2, 'c', 'uid-zzqx')`, fileID); err != nil {
		t.Fatalf("insert symbol: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		VALUES (?, 'parseConfig', 'parseConfig', 'function', 'go', 'func parseConfig()', 3, 4, 'c2', 'uid-parse')`, fileID); err != nil {
		t.Fatalf("insert parseConfig: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO entity_aliases (project, alias, qualified_name, source)
		VALUES (?, 'the renderer', 'zzqxonly', 'index')`, project); err != nil {
		t.Fatalf("insert alias: %v", err)
	}

	hits, err := search.SymbolFTS(st.DB, "The Renderer", 10)
	if err != nil {
		t.Fatalf("symbol fts: %v", err)
	}
	if len(hits) == 0 || hits[0].QualifiedName != "zzqxonly" {
		t.Fatalf("alias search = %+v, want zzqxonly first", hits)
	}

	hits, err = search.SymbolFTS(st.DB, "parseConfig", 10)
	if err != nil {
		t.Fatalf("fts miss path: %v", err)
	}
	found := false
	for _, h := range hits {
		if h.Name == "parseConfig" {
			found = true
		}
	}
	if !found {
		t.Fatalf("query parseConfig = %+v, want the FTS hit", hits)
	}
}
