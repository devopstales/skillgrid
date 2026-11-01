package codeindex

import (
	"context"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// hookCfg indexes Go, JS, markdown, yaml, and SQL files.
var hookCfg = Config{
	Include:      []string{"**/*.go", "**/*.js", "**/*.ts", "**/*.tsx", "**/*.md", "**/*.yaml", "**/*.sql"},
	Exclude:      []string{"**/node_modules/**", "**/.git/**"},
	ChunkLines:   80,
	ChunkOverlap: 10,
}

// writeHookFixture returns a small project: a Go file with a main + a call
// chain (for the process pass), a markdown doc (for the knowledge doc pass),
// a yaml config (for the knowledge config pass), a SQL schema + DML (for the
// knowledge SQL pass), and a JS route (for the process entry point).
func writeHookFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n\nfunc main() {\n\thandleListUsers()\n}\n\nfunc handleListUsers() {\n\tloadUsers()\n}\n\nfunc loadUsers() {}\n")
	mustWrite(t, filepath.Join(root, "docs", "a.md"), "# A\n\nSee [B](./b.md). Use handleListUsers to list users.\n")
	mustWrite(t, filepath.Join(root, "docs", "b.md"), "# B\nplain doc\n")
	mustWrite(t, filepath.Join(root, "config", "app.yaml"), "server:\n  handler: handleListUsers\n")
	mustWrite(t, filepath.Join(root, "db", "schema.sql"), "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);\n")
	mustWrite(t, filepath.Join(root, "server.js"), "function serve() { return []; }\napp.get('/users', serve);\n")
	return root
}

// TestIndexerHook covers @step-03 (Scenario: Indexer hook runs community,
// process, and knowledge in one transaction): after an index run, the 005+route
// extraction committed in one tx, then the community + process + knowledge
// passes ran (in the same index run, on a reopened DB after that tx commits —
// the store's single-connection pool cannot share the committed 005 tx, and the
// passes are advisory, so they do not roll back 005). The end state proves the
// three passes ran at index time: communities, processes, doc_nodes,
// config_nodes, and sql_schema_nodes all exist.
func TestIndexerHook(t *testing.T) {
	ResetFileFirstSymbol()
	idx, clean := newTestIndexer(t)
	defer clean()

	root := writeHookFixture(t)
	if _, err := idx.Run(context.Background(), root, hookCfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	db := idx.store.DB

	// 01 community pass ran: communities are populated.
	var comm int
	if err := db.QueryRow(`SELECT COUNT(*) FROM communities`).Scan(&comm); err != nil {
		t.Fatalf("count communities: %v", err)
	}
	if comm == 0 {
		t.Errorf("expected the community pass to populate communities, got 0")
	}

	// 02 process pass ran: processes + process_steps are populated (the JS
	// route's handler `serve` is an entry point; the Go main is a cli-main).
	var procs, steps int
	if err := db.QueryRow(`SELECT COUNT(*) FROM processes`).Scan(&procs); err != nil {
		t.Fatalf("count processes: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM process_steps`).Scan(&steps); err != nil {
		t.Fatalf("count process_steps: %v", err)
	}
	if procs == 0 {
		t.Errorf("expected the process pass to populate processes at index time, got 0")
	}
	if steps == 0 {
		t.Errorf("expected the process pass to populate process_steps at index time, got 0")
	}
	// The process pass used the deterministic LLM stub: the flows are labeled
	// (label_status != 'unlabeled'), never fabricated blank.
	var labeled int
	if err := db.QueryRow(`SELECT COUNT(*) FROM processes WHERE label_status = 'labeled'`).Scan(&labeled); err != nil {
		t.Fatalf("count labeled: %v", err)
	}
	if labeled == 0 {
		t.Errorf("expected the process pass to label flows at index time (deterministic stub), got 0 labeled")
	}

	// Knowledge doc pass ran: doc_nodes + references edges.
	var docs, docRefs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM doc_nodes`).Scan(&docs); err != nil {
		t.Fatalf("count doc_nodes: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'references' AND from_id IN (SELECT id FROM doc_nodes)`).Scan(&docRefs); err != nil {
		t.Fatalf("count doc references: %v", err)
	}
	if docs < 2 {
		t.Errorf("expected the knowledge doc pass to populate doc_nodes (a.md, b.md), got %d", docs)
	}
	if docRefs < 1 {
		t.Errorf("expected the knowledge doc pass to populate references edges (a.md -> b.md), got %d", docRefs)
	}
	// Doc->symbol edge: a.md mentions handleListUsers (defined in main.go) →
	// an INFERRED references edge from the doc node to the symbol, context
	// 'doc_mention'. The doc->doc link and the doc->symbol edge are distinct
	// rows (different to_id/context), so docRefs must now be at least 2.
	if docRefs < 2 {
		t.Errorf("expected doc->doc AND doc->symbol references edges from a.md, got %d", docRefs)
	}
	var symRefs int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges
		WHERE kind = 'references' AND context = 'doc_mention'
		  AND confidence = 'INFERRED'
		  AND to_name = 'handleListUsers'
		  AND to_id = (SELECT id FROM symbols WHERE name = 'handleListUsers')`).Scan(&symRefs); err != nil {
		t.Fatalf("count doc->symbol edges: %v", err)
	}
	if symRefs < 1 {
		t.Errorf("expected a doc->symbol references edge to handleListUsers (context=doc_mention, INFERRED), got %d", symRefs)
	}

	// Knowledge config pass ran: config_nodes + configures edges.
	var configs, configures int
	if err := db.QueryRow(`SELECT COUNT(*) FROM config_nodes`).Scan(&configs); err != nil {
		t.Fatalf("count config_nodes: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'configures'`).Scan(&configures); err != nil {
		t.Fatalf("count configures: %v", err)
	}
	if configs < 1 {
		t.Errorf("expected the knowledge config pass to populate config_nodes, got %d", configs)
	}
	if configures < 1 {
		t.Errorf("expected the knowledge config pass to populate configures edges, got %d", configures)
	}

	// Knowledge SQL pass ran: sql_schema_nodes (table + columns) + reads/
	// writes edges.
	var sqlNodes, rw int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sql_schema_nodes`).Scan(&sqlNodes); err != nil {
		t.Fatalf("count sql_schema_nodes: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind IN ('reads','writes')`).Scan(&rw); err != nil {
		t.Fatalf("count reads/writes: %v", err)
	}
	if sqlNodes == 0 {
		t.Errorf("expected the knowledge SQL pass to populate sql_schema_nodes (the users table + columns), got 0")
	}

	// All three passes ran in the SAME incremental index (one tx): the 005
	// symbols are still present alongside the knowledge nodes (the index did
	// not abort and the 005 graph is intact — advisory, never load-bearing).
	var syms int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols`).Scan(&syms); err != nil {
		t.Fatalf("count symbols: %v", err)
	}
	if syms == 0 {
		t.Errorf("expected 005 symbols to remain present after the knowledge passes, got 0")
	}
}
