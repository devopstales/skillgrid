package knowledge

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// openKnowledgeStore opens a scratch SQLite store with the 005
// symbols/edges/files tables plus the 015 knowledge tables — the full surface
// the knowledge extractors read from / write to.
func openKnowledgeStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	stmts := []string{
		`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, mtime_ns INTEGER, size INTEGER, content_hash TEXT, indexed_at TEXT)`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, name TEXT NOT NULL, qualified_name TEXT, kind TEXT NOT NULL, language TEXT, signature TEXT, start_line INTEGER NOT NULL, end_line INTEGER NOT NULL, content_hash TEXT NOT NULL, uid TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE edges (id INTEGER PRIMARY KEY AUTOINCREMENT, kind TEXT NOT NULL, from_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE, file_id INTEGER REFERENCES files(id) ON DELETE CASCADE, to_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE, to_name TEXT, target_path TEXT, confidence TEXT NOT NULL DEFAULT 'EXTRACTED', context TEXT NOT NULL DEFAULT '', confidence_score REAL NOT NULL DEFAULT 1.0, line INTEGER, valid_from INTEGER NOT NULL DEFAULT 0, UNIQUE(kind, from_id, file_id, to_id, to_name, target_path, line))`,
		`CREATE TABLE doc_nodes (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, title TEXT NOT NULL, path TEXT NOT NULL)`,
		`CREATE TABLE config_nodes (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, title TEXT NOT NULL, path TEXT NOT NULL)`,
		`CREATE TABLE sql_schema_nodes (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, table_name TEXT NOT NULL, column_name TEXT, kind TEXT NOT NULL, path TEXT NOT NULL)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}
	return db
}

func seedFile(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 1, 'h', 'now')`, path)
	if err != nil {
		t.Fatalf("seed file %s: %v", path, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedSymbol(t *testing.T, db *sql.DB, fileID int64, name, kind string, line int) int64 {
	t.Helper()
	return seedSymbolUID(t, db, fileID, name, kind, name+kind, line)
}

// seedSymbolUID is seedSymbol with an explicit uid, for fixtures that need
// two same-named symbols (a cross-package collision) with distinct uids.
func seedSymbolUID(t *testing.T, db *sql.DB, fileID int64, name, kind, uid string, line int) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid) VALUES (?, ?, ?, ?, ?, 'h', ?)`,
		fileID, name, kind, line, line, uid)
	if err != nil {
		t.Fatalf("seed symbol %s: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// countTable returns the row count of a table.
func countTable(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestDocLinks covers @step-03 (Scenario: Markdown links and wikilinks become
// references edges): .md files with [text](./other.md) and [[wikilinks]]
// produce doc_nodes + references edges between doc nodes, each
// confidence-labeled.
func TestDocLinks(t *testing.T) {
	db := openKnowledgeStore(t)
	ctx := context.Background()

	// Two docs: a.md links to b.md (markdown link) and to a wiki (wikilink).
	fA := seedFile(t, db, "docs/a.md")
	fB := seedFile(t, db, "docs/b.md")
	seedFile(t, db, "docs/wiki.md")

	aDoc := ExtractDoc("docs/a.md", []byte("# A\n\nSee [B](./b.md) and [[wiki]].\n"))
	if !aDoc.IsDoc {
		t.Fatalf("a.md should be recognized as a doc")
	}
	if aDoc.Title != "A" {
		t.Errorf("doc title = %q, want A (first H1)", aDoc.Title)
	}
	if len(aDoc.Links) != 2 {
		t.Fatalf("expected 2 links (md + wiki), got %d: %+v", len(aDoc.Links), aDoc.Links)
	}

	// Persist the doc nodes + references edges via the store API.
	store := &Store{db: db}
	for _, f := range []struct {
		path string
		doc  *DocResult
	}{
		{"docs/a.md", aDoc},
		{"docs/b.md", ExtractDoc("docs/b.md", []byte("# B\nplain doc\n"))},
		{"docs/wiki.md", ExtractDoc("docs/wiki.md", []byte("# Wiki\ncontent\n"))},
	} {
		if _, err := store.SaveDoc(ctx, f.path, f.doc); err != nil {
			t.Fatalf("SaveDoc %s: %v", f.path, err)
		}
	}

	// doc_nodes exist for the three docs.
	if n := countTable(t, db, "doc_nodes"); n != 3 {
		t.Errorf("expected 3 doc_nodes, got %d", n)
	}

	// references edges: a.md -> b.md (EXTRACTED) and a.md -> wiki.md
	// (INFERRED), both resolvable between doc nodes.
	var refs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'references'`).Scan(&refs); err != nil {
		t.Fatalf("count references: %v", err)
	}
	if refs != 2 {
		t.Errorf("expected 2 references edges, got %d", refs)
	}

	// The markdown link is EXTRACTED, the wikilink is INFERRED.
	var mdConf, wikiConf string
	err := db.QueryRow(`
		SELECT e.confidence FROM edges e
		JOIN doc_nodes dn ON dn.id = e.from_id
		WHERE e.kind = 'references' AND e.target_path = 'docs/b.md'`).Scan(&mdConf)
	if err != nil || mdConf != ConfidenceExtracted {
		t.Errorf("a.md -> b.md references confidence = %q (err %v), want EXTRACTED", mdConf, err)
	}
	err = db.QueryRow(`
		SELECT e.confidence FROM edges e
		JOIN doc_nodes dn ON dn.id = e.from_id
		WHERE e.kind = 'references' AND e.target_path = 'docs/wiki.md'`).Scan(&wikiConf)
	if err != nil || wikiConf != ConfidenceInferred {
		t.Errorf("a.md -> wiki.md references confidence = %q (err %v), want INFERRED", wikiConf, err)
	}

	// Every references edge from a doc node is confidence-labeled.
	var unlabeled int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'references' AND (confidence IS NULL OR confidence = '')`).Scan(&unlabeled); err != nil {
		t.Fatalf("count unlabeled: %v", err)
	}
	if unlabeled != 0 {
		t.Errorf("expected all doc references edges labeled, got %d unlabeled", unlabeled)
	}

	_ = fA
	_ = fB
}

// TestMalformedDoc covers @step-03 (Scenario: Malformed doc file falls back
// and indexes the rest): a doc with unparseable links among valid docs skips
// the bad links and indexes the rest; the index does not abort.
func TestMalformedDoc(t *testing.T) {
	db := openKnowledgeStore(t)
	ctx := context.Background()
	store := &Store{db: db}

	// A doc with a malformed link (no closing paren) next to a valid one.
	bad := "# Mix\n\nGood [B](./b.md)\nbroken [C](./c.md\n"
	seedFile(t, db, "docs/bad.md")
	seedFile(t, db, "docs/b.md")
	res := ExtractDoc("docs/bad.md", []byte(bad))
	if !res.IsDoc {
		t.Fatalf("bad.md should be a doc")
	}
	if _, err := store.SaveDoc(ctx, "docs/bad.md", res); err != nil {
		t.Fatalf("SaveDoc bad.md: %v", err)
	}
	if _, err := store.SaveDoc(ctx, "docs/b.md", ExtractDoc("docs/b.md", []byte("# B\n"))); err != nil {
		t.Fatalf("SaveDoc b.md: %v", err)
	}

	// The valid link (b.md) is indexed; the malformed link (c.md) is skipped,
	// not fatal. The doc node + its good reference edge exist.
	if n := countTable(t, db, "doc_nodes"); n != 2 {
		t.Errorf("expected 2 doc_nodes (bad.md, b.md), got %d", n)
	}
	var refs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'references'`).Scan(&refs); err != nil {
		t.Fatalf("count references: %v", err)
	}
	if refs != 1 {
		t.Errorf("expected 1 references edge (the valid b.md link; the malformed c.md link skipped), got %d", refs)
	}
}

// TestConfigRefs covers @step-03 (Scenario: Config references become
// configures edges): .yaml/.toml/.json files produce config_nodes +
// configures edges to the code they configure; explicit syntax is EXTRACTED,
// resolved-but-inferred refs are INFERRED.
func TestConfigRefs(t *testing.T) {
	db := openKnowledgeStore(t)
	ctx := context.Background()
	store := &Store{db: db}

	// A Go file with an explicit struct tag referencing a config key.
	fGo := seedFile(t, db, "app/server.go")
	sym := seedSymbol(t, db, fGo, "serverConfig", "struct", 5)

	// A YAML config whose value explicitly names the code symbol (EXTRACTED).
	fYaml := seedFile(t, db, "config/app.yaml")
	yamlRes := ExtractConfig("config/app.yaml", []byte("server:\n  handler: serverConfig\n"))
	if !yamlRes.IsConfig {
		t.Fatalf("app.yaml should be recognized as a config")
	}
	if len(yamlRes.Refs) != 1 {
		t.Fatalf("expected 1 config ref, got %d: %+v", len(yamlRes.Refs), yamlRes.Refs)
	}
	if yamlRes.Refs[0].Confidence != ConfidenceExtracted {
		t.Errorf("explicit yaml ref confidence = %q, want EXTRACTED", yamlRes.Refs[0].Confidence)
	}
	if n, err := store.SaveConfig(ctx, "config/app.yaml", yamlRes); err != nil || n != 1 {
		t.Fatalf("SaveConfig yaml = (%d, %v), want (1, nil)", n, err)
	}

	// A JSON config whose value names the symbol (EXTRACTED).
	fJSON := seedFile(t, db, "config/app.json")
	jsonRes := ExtractConfig("config/app.json", []byte("{\n  \"service\": \"serverConfig\"\n}"))
	if len(jsonRes.Refs) != 1 || jsonRes.Refs[0].Value != "serverConfig" {
		t.Fatalf("json refs = %+v, want 1 ref to serverConfig", jsonRes.Refs)
	}
	if _, err := store.SaveConfig(ctx, "config/app.json", jsonRes); err != nil {
		t.Fatalf("SaveConfig json: %v", err)
	}

	// A TOML config (INFERRED: the key->symbol link is a convention).
	fToml := seedFile(t, db, "config/app.toml")
	tomlRes := ExtractConfig("config/app.toml", []byte("name = \"serverConfig\"\n"))
	if len(tomlRes.Refs) != 1 || tomlRes.Refs[0].Value != "serverConfig" {
		t.Fatalf("toml refs = %+v, want 1 ref to serverConfig", tomlRes.Refs)
	}
	if _, err := store.SaveConfig(ctx, "config/app.toml", tomlRes); err != nil {
		t.Fatalf("SaveConfig toml: %v", err)
	}

	// config_nodes exist for the three configs.
	if n := countTable(t, db, "config_nodes"); n != 3 {
		t.Errorf("expected 3 config_nodes, got %d", n)
	}

	// configures edges resolve to the code symbol (by id), labeled.
	var cfg int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'configures' AND to_id = ?`, sym).Scan(&cfg); err != nil {
		t.Fatalf("count configures: %v", err)
	}
	if cfg != 3 {
		t.Errorf("expected 3 configures edges to serverConfig, got %d", cfg)
	}
	// The explicit yaml/json refs are EXTRACTED.
	var extracted int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'configures' AND confidence = 'EXTRACTED'`).Scan(&extracted); err != nil {
		t.Fatalf("count extracted: %v", err)
	}
	if extracted < 2 {
		t.Errorf("expected >=2 EXTRACTED configures edges (yaml + json), got %d", extracted)
	}
	_ = fYaml
	_ = fJSON
	_ = fToml
}

// TestAmbiguousConfigRef covers @step-03 (Scenario: Unresolvable config ref is
// ambiguous not dropped): a config reference that resolves to no known symbol
// is kept and marked AMBIGUOUS, not silently dropped.
func TestAmbiguousConfigRef(t *testing.T) {
	db := openKnowledgeStore(t)
	ctx := context.Background()
	store := &Store{db: db}

	seedFile(t, db, "config/app.yaml")
	res := ExtractConfig("config/app.yaml", []byte("service:\n  handler: ghostHandler\n"))
	if _, err := store.SaveConfig(ctx, "config/app.yaml", res); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// The unresolvable ref is KEPT as an AMBIGUOUS edge (to_id null, to_name
	// the literal ref), not dropped.
	var ambiguous int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'configures' AND confidence = 'AMBIGUOUS' AND to_id IS NULL AND to_name = 'ghostHandler'`).Scan(&ambiguous); err != nil {
		t.Fatalf("count ambiguous: %v", err)
	}
	if ambiguous != 1 {
		t.Errorf("expected 1 AMBIGUOUS configures edge for the unresolvable ref (not dropped), got %d", ambiguous)
	}
}

// TestAmbiguousConfigRefMultipleSymbols covers the cross-package name
// collision: a config reference whose name matches MORE THAN ONE symbol is
// stored AMBIGUOUS (to_id null), not silently bound to the lowest-id symbol as
// EXTRACTED. A name match of exactly one symbol stays EXTRACTED.
func TestAmbiguousConfigRefMultipleSymbols(t *testing.T) {
	db := openKnowledgeStore(t)
	ctx := context.Background()
	store := &Store{db: db}

	// Two distinct symbols in distinct packages share the name `loadUsers` —
	// a cross-package collision (different uid, same name). seedSymbol derives
	// uid from name+kind (which would collide), so insert directly with
	// distinct uids.
	fA := seedFile(t, db, "pkgA/load.go")
	fB := seedFile(t, db, "pkgB/load.go")
	seedSymbolUID(t, db, fA, "loadUsers", "function", "pkgA.loadUsers", 3)
	seedSymbolUID(t, db, fB, "loadUsers", "function", "pkgB.loadUsers", 7)

	seedFile(t, db, "config/app.yaml")
	res := ExtractConfig("config/app.yaml", []byte("service:\n  handler: loadUsers\n"))
	if len(res.Refs) != 1 || res.Refs[0].Value != "loadUsers" {
		t.Fatalf("expected 1 ref to loadUsers, got %+v", res.Refs)
	}
	if _, err := store.SaveConfig(ctx, "config/app.yaml", res); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	// The ref matched >1 symbol → AMBIGUOUS (to_id null), NOT EXTRACTED bound
	// to the lowest-id loadUsers.
	var ambiguous, extracted int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'configures' AND to_name = 'loadUsers' AND confidence = 'AMBIGUOUS' AND to_id IS NULL`).Scan(&ambiguous); err != nil {
		t.Fatalf("count ambiguous: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'configures' AND to_name = 'loadUsers' AND confidence = 'EXTRACTED' AND to_id IS NOT NULL`).Scan(&extracted); err != nil {
		t.Fatalf("count extracted: %v", err)
	}
	if ambiguous != 1 {
		t.Errorf("expected 1 AMBIGUOUS configures edge for the colliding name (not bound to the lowest id), got %d", ambiguous)
	}
	if extracted != 0 {
		t.Errorf("expected 0 EXTRACTED configures edges for the colliding name, got %d", extracted)
	}

	// Contrast: a ref matching exactly one symbol stays EXTRACTED (bound by id).
	fC := seedFile(t, db, "pkgC/single.go")
	sole := seedSymbol(t, db, fC, "uniqueHandler", "function", 2)
	seedFile(t, db, "config/one.yaml")
	res2 := ExtractConfig("config/one.yaml", []byte("service:\n  handler: uniqueHandler\n"))
	if _, err := store.SaveConfig(ctx, "config/one.yaml", res2); err != nil {
		t.Fatalf("SaveConfig one: %v", err)
	}
	var soleExtracted int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'configures' AND to_id = ? AND confidence = 'EXTRACTED'`, sole).Scan(&soleExtracted); err != nil {
		t.Fatalf("count sole extracted: %v", err)
	}
	if soleExtracted != 1 {
		t.Errorf("expected 1 EXTRACTED configures edge for the unique ref, got %d", soleExtracted)
	}
}

// TestSqlSchema covers @step-03 (Scenario: SQL DDL becomes table and column
// nodes with reads and writes): .sql DDL produces sql_schema_nodes (tables +
// columns) and code that references them gets reads/writes edges,
// confidence-labeled.
func TestSqlSchema(t *testing.T) {
	db := openKnowledgeStore(t)
	ctx := context.Background()
	store := &Store{db: db}

	// A SQL schema file with DDL for `users` and `orders` tables.
	seedFile(t, db, "db/schema.sql")
	schema := `
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  name TEXT,
  email TEXT UNIQUE
);
CREATE TABLE orders (
  id INTEGER PRIMARY KEY,
  user_id INTEGER,
  total REAL
);
`
	schemaRes := ExtractSQL("db/schema.sql", []byte(schema))
	if n, err := store.SaveSchema(ctx, "db/schema.sql", schemaRes); err != nil || n != 8 {
		t.Fatalf("SaveSchema = (%d, %v), want (8, nil) [2 tables + 6 columns]", n, err)
	}

	// sql_schema_nodes: 2 tables + 6 columns = 8 nodes.
	if n := countTable(t, db, "sql_schema_nodes"); n != 8 {
		t.Errorf("expected 8 sql_schema_nodes, got %d", n)
	}
	var tables, cols int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sql_schema_nodes WHERE kind = 'table'`).Scan(&tables); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sql_schema_nodes WHERE kind = 'column'`).Scan(&cols); err != nil {
		t.Fatalf("count columns: %v", err)
	}
	if tables != 2 || cols != 6 {
		t.Errorf("expected 2 tables + 6 columns, got %d tables + %d columns", tables, cols)
	}

	// Code that reads/writes the tables gets reads/writes edges (EXTRACTED).
	// Two separate code files: one that reads `users`, one that writes
	// `orders`. Each file's first symbol is the edge source.
	fRead := seedFile(t, db, "app/users.go")
	seedSymbol(t, db, fRead, "loadUsers", "function", 10)
	fWrite := seedFile(t, db, "app/orders.go")
	seedSymbol(t, db, fWrite, "createOrder", "function", 5)

	readRes := ExtractSQL("app/users.go", []byte("SELECT id, name FROM users;\n"))
	if n, err := store.SaveSchema(ctx, "app/users.go", readRes); err != nil || n != 1 {
		t.Fatalf("SaveSchema read = (%d, %v), want (1, nil)", n, err)
	}
	writeRes := ExtractSQL("app/orders.go", []byte("INSERT INTO orders (id, user_id, total) VALUES (1, 2, 3.5);\n"))
	if n, err := store.SaveSchema(ctx, "app/orders.go", writeRes); err != nil || n != 1 {
		t.Fatalf("SaveSchema write = (%d, %v), want (1, nil)", n, err)
	}
	var reads, writes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'reads'`).Scan(&reads); err != nil {
		t.Fatalf("count reads: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'writes'`).Scan(&writes); err != nil {
		t.Fatalf("count writes: %v", err)
	}
	if reads != 1 || writes != 1 {
		t.Errorf("expected 1 reads + 1 writes, got %d reads + %d writes", reads, writes)
	}
	// The reads edge is from loadUsers to the users table; writes from
	// createOrder to orders.
	var readFrom, writeTo string
	if err := db.QueryRow(`
		SELECT s.name FROM edges e
		JOIN symbols s ON s.id = e.from_id
		JOIN sql_schema_nodes tn ON tn.id = e.to_id
		WHERE e.kind = 'reads' AND tn.table_name = 'users'`).Scan(&readFrom); err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := db.QueryRow(`
		SELECT s.name FROM edges e
		JOIN symbols s ON s.id = e.from_id
		JOIN sql_schema_nodes tn ON tn.id = e.to_id
		WHERE e.kind = 'writes' AND tn.table_name = 'orders'`).Scan(&writeTo); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if readFrom != "loadUsers" || writeTo != "createOrder" {
		t.Errorf("reads from %q (want loadUsers), writes from %q (want createOrder)", readFrom, writeTo)
	}
	// Every reads/writes edge is confidence-labeled EXTRACTED.
	var nonExtracted int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind IN ('reads','writes') AND confidence != 'EXTRACTED'`).Scan(&nonExtracted); err != nil {
		t.Fatalf("count non-extracted: %v", err)
	}
	if nonExtracted != 0 {
		t.Errorf("expected all reads/writes edges EXTRACTED, got %d non-EXTRACTED", nonExtracted)
	}
}

// TestMalformedSQL covers @step-03 (Scenario: Malformed SQL statement is
// skipped and the rest is indexed): a SQL file with a statement that fails to
// parse among valid DDL skips that statement and indexes the rest; the index
// does not abort.
func TestMalformedSQL(t *testing.T) {
	db := openKnowledgeStore(t)
	ctx := context.Background()
	store := &Store{db: db}

	fid := seedFile(t, db, "db/schema.sql")
	seedSymbol(t, db, fid, "sym", "function", 1)
	// A valid CREATE TABLE, a malformed CREATE TABLE (unbalanced parens), and
	// a valid DML statement (an INSERT that names a table). The malformed DDL
	// is skipped; the rest (the valid DDL table + the DML access) is indexed.
	sql := `
CREATE TABLE good1 (id INTEGER PRIMARY KEY);
CREATE TABLE broken (id INTEGER PRIMARY KEY
INSERT INTO good1 (id) VALUES (1);
`
	res := ExtractSQL("db/schema.sql", []byte(sql))
	if _, err := store.SaveSchema(ctx, "db/schema.sql", res); err != nil {
		t.Fatalf("SaveSchema: %v", err)
	}
	// The valid table is indexed; the broken one is skipped.
	var good1, broken int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sql_schema_nodes WHERE table_name = 'good1' AND kind = 'table'`).Scan(&good1); err != nil {
		t.Fatalf("count good1: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sql_schema_nodes WHERE table_name = 'broken' AND kind = 'table'`).Scan(&broken); err != nil {
		t.Fatalf("count broken: %v", err)
	}
	if good1 != 1 {
		t.Errorf("expected the valid 'good1' table to be indexed, got %d", good1)
	}
	if broken != 0 {
		t.Errorf("expected the malformed 'broken' table to be skipped, got %d", broken)
	}
	// The valid DML (the INSERT) is still indexed as a writes edge — the
	// malformed statement did not abort the rest.
	var writes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'writes'`).Scan(&writes); err != nil {
		t.Fatalf("count writes: %v", err)
	}
	if writes != 1 {
		t.Errorf("expected the valid INSERT to be indexed as a writes edge (the index did not abort), got %d", writes)
	}
}

// TestSQLOnlyForSQLFiles covers @step-03 (isSQL false positives): only .sql
// files are parsed for SQL DDL/DML. A .go (or other non-.sql) file whose
// contents merely contain the words SELECT / INSERT / UPDATE / DELETE FROM
// must NOT be parsed into spurious table nodes.
func TestSQLOnlyForSQLFiles(t *testing.T) {
	// Unit: isSQL is true only for .sql / .SQL, regardless of contents.
	cases := []struct {
		path string
		want bool
	}{
		{"db/schema.sql", true},
		{"db/SCHEMA.SQL", true},
		{"app/users.go", false},
		{"docs/a.md", false},
		{"config/app.yaml", false},
	}
	for _, c := range cases {
		// Contents contain SQL keywords to prove the extension (not the
		// contents) drives the decision.
		got := isSQL(c.path, []byte("SELECT id FROM users; INSERT INTO x;"))
		if got != c.want {
			t.Errorf("isSQL(%q) = %v, want %v", c.path, got, c.want)
		}
	}

	// Integration: a .go file containing SELECT in a string/comment produces
	// no table nodes (whereas a .sql file with the same DDL does).
	db := openKnowledgeStore(t)
	ctx := context.Background()
	store := &Store{db: db}
	fGo := seedFile(t, db, "app/users.go")
	seedSymbol(t, db, fGo, "loadUsers", "function", 10)
	// A .go file with SELECT in a string and a comment — SQL keywords but not
	// a .sql file.
	goSrc := "package app\n\nvar q = `SELECT id, name FROM users;`\n\n// UPDATE the users table via SELECT\nfunc loadUsers() {}\n"
	files := []FileInput{{Path: "app/users.go", Contents: []byte(goSrc)}}
	if _, err := RunPasses(ctx, store, files); err != nil {
		t.Fatalf("RunPasses: %v", err)
	}
	if n := countTable(t, db, "sql_schema_nodes"); n != 0 {
		t.Errorf("expected 0 sql_schema_nodes from a .go file with SELECT keywords, got %d", n)
	}
	var reads int
	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE kind = 'reads'`).Scan(&reads); err != nil {
		t.Fatalf("count reads: %v", err)
	}
	if reads != 0 {
		t.Errorf("expected 0 reads edges from a .go file with SELECT keywords, got %d", reads)
	}

	// Contrast: the SAME DDL in a .sql file IS parsed (the .sql behavior is
	// kept).
	fSQL := seedFile(t, db, "db/schema.sql")
	sqlFiles := []FileInput{{Path: "db/schema.sql", Contents: []byte("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);\n")}}
	if _, err := RunPasses(ctx, store, sqlFiles); err != nil {
		t.Fatalf("RunPasses sql: %v", err)
	}
	if n := countTable(t, db, "sql_schema_nodes"); n == 0 {
		t.Errorf("expected the .sql file to produce sql_schema_nodes, got 0")
	}
	_ = fSQL
}
