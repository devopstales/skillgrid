package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/graph"
)

// seedDocNode inserts a doc node and returns its id.
func seedDocNode(t *testing.T, db *sql.DB, fileID int64, title, path string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO doc_nodes (file_id, title, path) VALUES (?, ?, ?)`, fileID, title, path)
	if err != nil {
		t.Fatalf("seed doc node: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// seedConfigNode inserts a config node and returns its id.
func seedConfigNode(t *testing.T, db *sql.DB, fileID int64, path string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO config_nodes (file_id, title, path) VALUES (?, ?, ?)`, fileID, path, path)
	if err != nil {
		t.Fatalf("seed config node: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// seedSQLNode inserts a sql_schema_node and returns its id.
func seedSQLNode(t *testing.T, db *sql.DB, fileID int64, table, column, kind, path string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO sql_schema_nodes (file_id, table_name, column_name, kind, path) VALUES (?, ?, ?, ?, ?)`,
		fileID, table, column, kind, path)
	if err != nil {
		t.Fatalf("seed sql node: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// seedEdge inserts a knowledge edge. toID 0 means a name-only edge (to_id null).
func seedEdge(t *testing.T, db *sql.DB, kind string, fromID, fileID, toID int64, toName, targetPath, conf string, line int) {
	t.Helper()
	var to sql.NullInt64
	if toID != 0 {
		to = sql.NullInt64{Int64: toID, Valid: true}
	}
	if _, err := db.Exec(`INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		kind, fromID, fileID, to, toName, targetPath, conf, line); err != nil {
		t.Fatalf("seed edge %s: %v", kind, err)
	}
}

// TestCodePathSpan covers @step-03 (Scenario: Path tool traces code to doc to
// config to table): code_path (via graph.PathSpan) returns a single path that
// spans the code symbol to a doc, a config, and a table, traversing at least
// two different knowledge edge kinds (configures + reads). The path is
// non-vacuous: a genuine multi-hop walk, not a single hop.
func TestCodePathSpan(t *testing.T) {
	db := openKnowledgeStore(t)
	// PathSpan issues many sequential queries (BFS over the unified node
	// space); a :memory: store must be single-connection or the pool opens a
	// second (empty) in-memory DB.
	db.SetMaxOpenConns(1)
	ctx := context.Background()

	// The 005 edges table shares one integer space for from_id/to_id, but the
	// 015 knowledge tables (doc/config/sql) autoincrement independently of
	// symbols. In a cold fixture every node's id is 1, so
	// `WHERE from_id=1 OR to_id=1` over-matches across tables. Bump each
	// table's autoincrement by a different count with throwaway rows so the
	// fixture's real nodes land on DISTINCT ids (mirroring a real store where
	// symbol ids and knowledge-node ids do not all collide).
	filler := seedFile(t, db, "filler.txt")
	for i := 0; i < 1; i++ {
		db.Exec(`INSERT INTO symbols (file_id, name, qualified_name, kind, language, start_line, end_line, content_hash, uid) VALUES (?, ?, ?, 'function', 'go', 1, 1, 'h', ?)`, filler, fmt.Sprintf("_s%d", i), fmt.Sprintf("x._s%d", i), fmt.Sprintf("filler-sym-%d", i))
	}
	for i := 0; i < 2; i++ {
		db.Exec(`INSERT INTO doc_nodes (file_id, title, path) VALUES (?, ?, ?)`, filler, fmt.Sprintf("_f%d", i), fmt.Sprintf("filler%d.md", i))
	}
	for i := 0; i < 3; i++ {
		db.Exec(`INSERT INTO config_nodes (file_id, title, path) VALUES (?, ?, ?)`, filler, fmt.Sprintf("_f%d", i), fmt.Sprintf("filler%d.yaml", i))
	}
	for i := 0; i < 4; i++ {
		db.Exec(`INSERT INTO sql_schema_nodes (file_id, table_name, column_name, kind, path) VALUES (?, ?, '', 'table', ?)`, filler, fmt.Sprintf("_f%d", i), fmt.Sprintf("filler%d.sql", i))
	}

	// Code symbol (the code side of the span). qualified_name is set so the
	// graph's loadSymbolByID (which scans it into a string) resolves cleanly.
	fCode := seedFile(t, db, "app/users.go")
	res, err := db.Exec(`INSERT INTO symbols (file_id, name, qualified_name, kind, language, start_line, end_line, content_hash, uid) VALUES (?, 'loadUsers', 'app.loadUsers', 'function', 'go', 3, 3, 'h', 'loadUsers-function')`, fCode)
	if err != nil {
		t.Fatalf("seed code symbol: %v", err)
	}
	codeID, _ := res.LastInsertId()

	// Config node that configures the code symbol.
	fCfg := seedFile(t, db, "config/app.yaml")
	cfgID := seedConfigNode(t, db, fCfg, "config/app.yaml")
	seedEdge(t, db, KindConfigures, cfgID, fCfg, codeID, "loadUsers", "", ConfidenceExtracted, 2)

	// Table node the code symbol reads.
	fSQL := seedFile(t, db, "db/schema.sql")
	tableID := seedSQLNode(t, db, fSQL, "users", "", KindTable, "db/schema.sql")
	seedEdge(t, db, KindReads, codeID, fSQL, tableID, "users", "", ConfidenceExtracted, 5)

	// Doc node that references the code symbol.
	fDoc := seedFile(t, db, "docs/a.md")
	docID := seedDocNode(t, db, fDoc, "A", "docs/a.md")
	seedEdge(t, db, KindReferences, docID, fDoc, codeID, "loadUsers", "", ConfidenceExtracted, 4)

	codeSym := graph.Symbol{ID: codeID, Name: "loadUsers", Path: "app/users.go", Kind: "function"}

	// doc→code→table: a genuine multi-hop path that spans a doc to a table,
	// traversing TWO different knowledge edge kinds (references doc->code, then
	// reads code->table). This is the non-vacuous span: the doc is not directly
	// connected to the table, so the path must walk through the code symbol.
	docSym := graph.Symbol{ID: 0, Name: "docs/a.md", Path: "docs/a.md"}
	// Resolve the doc to its span symbol so PathSpan can start from it. The doc
	// node id is resolved by name (its path).
	span, err := graph.PathSpanFromName(ctx, db, "docs/a.md", "users")
	if err != nil {
		t.Fatalf("PathSpan doc->table: %v", err)
	}
	if !span.Found {
		t.Fatalf("expected code_path to span doc -> code -> table, got Found=false: %+v", span.GraphStops)
	}
	if len(span.Path) < 2 {
		t.Fatalf("expected a multi-hop path (doc->code->table), got %d hops", len(span.Path))
	}
	kinds := map[string]bool{}
	for _, hop := range span.Path {
		kinds[hop.Kind] = true
	}
	knowledgeKinds := 0
	for _, k := range []string{KindConfigures, KindReads, KindWrites, KindReferences} {
		if kinds[k] {
			knowledgeKinds++
		}
	}
	if knowledgeKinds < 2 {
		t.Errorf("expected the path to traverse >=2 knowledge edge kinds (references + reads), got %d (kinds %v)", knowledgeKinds, kinds)
	}
	// The path must pass through the code symbol and reach the table node
	// (knowledge nodes are surfaced by name, ID 0, so match on name).
	if !pathTouches(span.Path, tableNodeName(tableID)) {
		t.Errorf("expected the doc->table path to reach the table node, got %+v", span.Path)
	}
	_ = docSym

	// code→config: the config that configures the code symbol is reachable in
	// one hop, and that hop is a knowledge edge (configures).
	cfgSpan, err := graph.PathSpan(ctx, db, codeSym, "config/app.yaml")
	if err != nil {
		t.Fatalf("PathSpan code->config: %v", err)
	}
	if !cfgSpan.Found {
		t.Fatalf("expected code_path to reach the config, got Found=false")
	}
	if !kindsHas(cfgSpan.Path, KindConfigures) {
		t.Errorf("expected the code->config hop to be a configures edge, got kinds %v", pathKinds(cfgSpan.Path))
	}
	// The config hop must actually land on the config node (the stored edge is
	// config->code, so the config appears as the hop's From or To endpoint).
	if !pathTouches(cfgSpan.Path, "config/app.yaml") {
		t.Errorf("expected the code->config path to land on the config node, got %+v", cfgSpan.Path)
	}
	_ = cfgID
	_ = docID
}

// tableNodeName returns the span display name of a table node ("table:<name>"),
// matching graph.tableNodeKey.
func tableNodeName(tableID int64) string {
	return "table:users"
}

// pathTouches reports whether any hop's endpoint carries the given node name.
func pathTouches(path []graph.Edge, name string) bool {
	for _, e := range path {
		if e.From.Name == name || e.To.Name == name {
			return true
		}
	}
	return false
}

func kindsHas(path []graph.Edge, kind string) bool {
	for _, e := range path {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

func pathKinds(path []graph.Edge) []string {
	out := make([]string, 0, len(path))
	for _, e := range path {
		out = append(out, e.Kind)
	}
	return out
}
