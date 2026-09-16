package hybrid

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	_ "modernc.org/sqlite"
)

// TestRankOfflineWithProvenance covers the pure RRF fusion: a hit ranked in
// all three legs outranks one ranked only in FTS, provenance flags mirror the
// rank maps, and hits absent from every leg are dropped.
func TestRankOfflineWithProvenance(t *testing.T) {
	t.Run("empty input returns nil", func(t *testing.T) {
		if got := Rank(nil, nil, nil, nil, RRFK); got != nil {
			t.Errorf("Rank(nil hits) = %v, want nil", got)
		}
		if got := Rank(map[string]Hit{}, map[string]int{"a": 0}, nil, nil, RRFK); got != nil {
			t.Errorf("Rank(empty hits) = %v, want nil", got)
		}
	})

	hits := map[string]Hit{
		"sym:all":   {Path: "all.go", Symbol: "all"},
		"sym:fts":   {Path: "fts.go", Symbol: "fts"},
		"sym:none":  {Path: "none.go", Symbol: "none"},
		"chunk:top": {Path: "top.go", Snippet: "top"},
	}
	ftsRanks := map[string]int{"sym:all": 0, "sym:fts": 1}
	sigRanks := map[string]int{"sym:all": 0, "chunk:top": 0}
	semRanks := map[string]int{"sym:all": 0}

	got := Rank(hits, ftsRanks, sigRanks, semRanks, RRFK)
	if len(got) != 3 {
		t.Fatalf("Rank returned %d hits, want 3 (none-rank hit excluded)", len(got))
	}

	byID := map[string]Hit{}
	for _, h := range got {
		byID[h.Path] = h
	}
	if _, ok := byID["none.go"]; ok {
		t.Errorf("hit absent from every rank must be excluded, got %v", got)
	}

	// All three legs: 3 * 1/(60+0+1).
	wantAll := 3.0 / float64(RRFK+1)
	if h, ok := byID["all.go"]; !ok {
		t.Fatalf("missing sym:all")
	} else {
		if h.Score != wantAll {
			t.Errorf("sym:all score = %v, want %v", h.Score, wantAll)
		}
		want := Provenance{FTS: true, Signal: true, Semantic: true}
		if h.Provenance != want {
			t.Errorf("sym:all provenance = %+v, want %+v", h.Provenance, want)
		}
	}

	// FTS only: 1 * 1/(60+1+1).
	wantFTS := 1.0 / float64(RRFK+2)
	if h, ok := byID["fts.go"]; !ok {
		t.Fatalf("missing sym:fts")
	} else {
		if h.Score != wantFTS {
			t.Errorf("sym:fts score = %v, want %v", h.Score, wantFTS)
		}
		want := Provenance{FTS: true}
		if h.Provenance != want {
			t.Errorf("sym:fts provenance = %+v, want %+v", h.Provenance, want)
		}
	}

	// Signal only: 1 * 1/(60+0+1).
	wantSig := 1.0 / float64(RRFK+1)
	if h, ok := byID["top.go"]; !ok {
		t.Fatalf("missing chunk:top")
	} else {
		if h.Score != wantSig {
			t.Errorf("chunk:top score = %v, want %v", h.Score, wantSig)
		}
		want := Provenance{Signal: true}
		if h.Provenance != want {
			t.Errorf("chunk:top provenance = %+v, want %+v", h.Provenance, want)
		}
	}

	// Strict ordering: all-legs hit first, none-rank hit never present.
	if got[0].Path != "all.go" {
		t.Errorf("top hit = %q, want all.go (3-leg fusion beats single-leg)", got[0].Path)
	}
	if got[0].Score <= wantFTS {
		t.Errorf("3-leg score %v must exceed 1-leg FTS score %v", got[0].Score, wantFTS)
	}
	// The 3-leg score (all.go) must strictly exceed the best single-leg score.
	if got[0].Score <= got[1].Score {
		t.Errorf("3-leg fusion score %v must exceed single-leg score %v: %+v", got[0].Score, got[1].Score, got)
	}
}

const hybridTestSchema = `
CREATE TABLE IF NOT EXISTS files (id INTEGER PRIMARY KEY, path TEXT UNIQUE, mtime_ns INTEGER, size INTEGER, content_hash TEXT, indexed_at TEXT);
CREATE TABLE IF NOT EXISTS chunks (id INTEGER PRIMARY KEY, file_id INTEGER, start_line INTEGER, end_line INTEGER, text TEXT, content_hash TEXT);
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(text, content='chunks', content_rowid='id', tokenize='unicode61');
CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN INSERT INTO chunks_fts(rowid, text) VALUES (new.id, new.text); END;
CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN INSERT INTO chunks_fts(chunks_fts, rowid, text) VALUES('delete', old.id, old.text); END;
CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN INSERT INTO chunks_fts(chunks_fts, rowid, text) VALUES('delete', old.id, old.text); INSERT INTO chunks_fts(rowid, text) VALUES (new.id, new.text); END;
CREATE TABLE IF NOT EXISTS symbols (id INTEGER PRIMARY KEY, file_id INTEGER, name TEXT, qualified_name TEXT, kind TEXT, language TEXT, signature TEXT, start_line INTEGER, end_line INTEGER, content_hash TEXT, uid TEXT UNIQUE);
CREATE VIRTUAL TABLE IF NOT EXISTS symbol_fts USING fts5(name, qualified_name, content='symbols', content_rowid='id', tokenize='unicode61');
CREATE TRIGGER IF NOT EXISTS symbols_ai AFTER INSERT ON symbols BEGIN INSERT INTO symbol_fts(rowid, name, qualified_name) VALUES (new.id, new.name, new.qualified_name); END;
CREATE TRIGGER IF NOT EXISTS symbols_ad AFTER DELETE ON symbols BEGIN INSERT INTO symbol_fts(symbol_fts, rowid, name, qualified_name) VALUES('delete', old.id, old.name, old.qualified_name); END;
CREATE TRIGGER IF NOT EXISTS symbols_au AFTER UPDATE ON symbols BEGIN INSERT INTO symbol_fts(symbol_fts, rowid, name, qualified_name) VALUES('delete', old.id, old.name, old.qualified_name); INSERT INTO symbol_fts(rowid, name, qualified_name) VALUES (new.id, new.name, new.qualified_name); END;
CREATE TABLE IF NOT EXISTS embeddings (id INTEGER PRIMARY KEY, symbol_id INTEGER, vector BLOB, model TEXT, dim INTEGER NOT NULL DEFAULT 0, language TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS chunk_embeddings (chunk_id INTEGER PRIMARY KEY, vector BLOB, model TEXT, dim INTEGER NOT NULL DEFAULT 0, language TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS embed_meta (key TEXT PRIMARY KEY, value TEXT);
CREATE TABLE IF NOT EXISTS edges (id INTEGER PRIMARY KEY, kind TEXT, from_id INTEGER, file_id INTEGER, to_id INTEGER, to_name TEXT, target_path TEXT, confidence TEXT, line INTEGER);
CREATE TABLE IF NOT EXISTS rationale (id INTEGER PRIMARY KEY, symbol_id INTEGER, text TEXT, kind TEXT, line INTEGER);
CREATE TABLE IF NOT EXISTS lsh_buckets (id INTEGER PRIMARY KEY, symbol_id INTEGER, bucket TEXT);
CREATE TABLE IF NOT EXISTS index_freshness (id INTEGER PRIMARY KEY, project_id TEXT, last_indexed TEXT);
`

// openHybridTestDB returns a CGo-free in-memory SQLite with the hybrid search
// schema. A unique per-test shared-cache URI keeps the pool connections on one
// database; MaxOpenConns(1) avoids shared-cache busy-wait deadlocks.
func openHybridTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:hybridtest_%s?cache=shared&mode=memory", t.Name())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(hybridTestSchema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return db
}

func insertHybridFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES ('main.go', 1, 1, 'h1', 'x')`,
		`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES ('other.go', 1, 1, 'h2', 'x')`,
		`INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid) VALUES (1, 'parseConfig', 'parseConfig', 'function', 'go', 'func parseConfig() int', 10, 20, 'c1', 'parseConfig')`,
		`INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid) VALUES (2, 'loadUserSettings', 'loadUserSettings', 'function', 'go', 'func loadUserSettings() string', 30, 40, 'c2', 'loadUserSettings')`,
		`INSERT INTO chunks (file_id, start_line, end_line, text, content_hash) VALUES (1, 1, 5, 'parseConfig call site in main', 'k1')`,
		`INSERT INTO chunks (file_id, start_line, end_line, text, content_hash) VALUES (2, 1, 5, 'loadUserSettings call site in cli', 'k2')`,
	}
	// A chunk embedding on the first file's chunk (file 1, go) so the
	// chunk-vector leg has a row to rank and the language filter can scope it.
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("insert fixture: %v", err)
		}
	}
}

func seedHybridEmbeddings(t *testing.T, db *sql.DB, emb embedder.Embedder, query string) {
	t.Helper()
	ctx := context.Background()
	var symbolIDs []int64
	rows, err := db.Query(`SELECT id, name FROM symbols`)
	if err != nil {
		t.Fatalf("select symbols: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("scan symbol: %v", err)
		}
		symbolIDs = append(symbolIDs, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate symbols: %v", err)
	}
	for _, id := range symbolIDs {
		var name string
		if err := db.QueryRow(`SELECT name FROM symbols WHERE id = ?`, id).Scan(&name); err != nil {
			t.Fatalf("select symbol name: %v", err)
		}
		vec, err := emb.Embed(ctx, name+" "+query)
		if err != nil {
			t.Fatalf("embed %s: %v", name, err)
		}
		if _, err := db.Exec(`INSERT INTO embeddings (symbol_id, vector, model) VALUES (?, ?, ?)`, id, memory.EncodeVector(vec), emb.Model()); err != nil {
			t.Fatalf("insert embedding: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, id := range symbolIDs {
			if _, err := db.Exec(`DELETE FROM embeddings WHERE symbol_id = ?`, id); err != nil {
				t.Errorf("cleanup embedding: %v", err)
			}
		}
	})
}

// TestSearchDegenerateDegrades covers the FTS floor: a working embedder adds
// the semantic leg, a nil embedder degrades to FTS + signals without error,
// and empty-query / nil-db are hard errors.
func TestSearchDegenerateDegrades(t *testing.T) {
	ctx := context.Background()

	t.Run("nil db errors", func(t *testing.T) {
		if _, err := Search(ctx, nil, "parseConfig", Options{}); err == nil {
			t.Error("Search(nil db) should error")
		}
	})

	t.Run("empty query errors", func(t *testing.T) {
		db := openHybridTestDB(t)
		insertHybridFixture(t, db)
		if _, err := Search(ctx, db, "", Options{Embedder: embedder.NewHash(64)}); err == nil {
			t.Error("Search(\"\") should error")
		}
	})

	t.Run("working hash embedder adds semantic leg", func(t *testing.T) {
		db := openHybridTestDB(t)
		insertHybridFixture(t, db)
		emb := embedder.NewHash(64)
		seedHybridEmbeddings(t, db, emb, "parseConfig")

		res, err := Search(ctx, db, "parseConfig", Options{Embedder: emb})
		if err != nil {
			t.Fatalf("Search with hash embedder: %v", err)
		}
		if len(res.Hits) == 0 {
			t.Fatalf("expected hits, got none: %+v", res)
		}
		if !contains(res.Legs, "semantic") {
			t.Errorf("legs = %v, want semantic", res.Legs)
		}
		// The symbol-named hit must carry provenance from the legs it
		// participated in (FTS identifier match + signal overlap).
		var found bool
		for _, h := range res.Hits {
			if h.Symbol == "parseConfig" {
				found = true
				if !h.Provenance.FTS && !h.Provenance.Signal {
					t.Errorf("parseConfig provenance = %+v, want FTS or Signal set", h.Provenance)
				}
				if h.Provenance.Semantic && h.Provenance.Sim == 0 {
					t.Errorf("semantic hit must carry a similarity, got 0")
				}
			}
		}
		if !found {
			t.Errorf("hits do not include parseConfig: %+v", res.Hits)
		}
	})

	t.Run("nil embedder degrades to FTS floor", func(t *testing.T) {
		db := openHybridTestDB(t)
		insertHybridFixture(t, db)

		res, err := Search(ctx, db, "parseConfig", Options{Embedder: nil})
		if err != nil {
			t.Fatalf("Search with nil embedder must degrade, got error: %v", err)
		}
		if len(res.Hits) == 0 {
			t.Fatalf("expected FTS floor hits, got none: %+v", res)
		}
		if contains(res.Legs, "semantic") {
			t.Errorf("legs = %v, must not include semantic without an embedder", res.Legs)
		}
		var found bool
		for _, h := range res.Hits {
			if h.Symbol == "parseConfig" && h.Provenance.FTS {
				found = true
			}
		}
		if !found {
			t.Errorf("FTS floor should rank parseConfig with FTS provenance: %+v", res.Hits)
		}
	})
}

// TestSearchNamesSymbol covers the identifier-aware symbol leg: a symbol in
// the DB is returned with its Symbol field set, and in semantic mode the
// vector leg still surfaces symbol-named results.
func TestSearchNamesSymbol(t *testing.T) {
	ctx := context.Background()
	db := openHybridTestDB(t)
	insertHybridFixture(t, db)
	emb := embedder.NewHash(64)
	seedHybridEmbeddings(t, db, emb, "parseConfig")

	t.Run("default mode names the symbol", func(t *testing.T) {
		res, err := Search(ctx, db, "parseConfig", Options{Embedder: emb})
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		var found bool
		for _, h := range res.Hits {
			if h.Symbol == "parseConfig" {
				found = true
				if h.Path != "main.go" || h.StartLine != 10 || h.EndLine != 20 {
					t.Errorf("parseConfig hit = %+v, want main.go:10-20", h)
				}
				if h.Kind != "function" {
					t.Errorf("parseConfig kind = %q, want function", h.Kind)
				}
			}
		}
		if !found {
			t.Fatalf("hits do not name parseConfig: %+v", res.Hits)
		}
	})

	t.Run("semantic mode still returns symbol-named results", func(t *testing.T) {
		res, err := Search(ctx, db, "parseConfig", Options{Semantic: true, Embedder: emb})
		if err != nil {
			t.Fatalf("Search semantic: %v", err)
		}
		if len(res.Hits) == 0 {
			t.Fatalf("semantic mode returned no hits: %+v", res)
		}
		if !contains(res.Legs, "semantic") {
			t.Errorf("legs = %v, want semantic", res.Legs)
		}
		var found bool
		for _, h := range res.Hits {
			if h.Symbol != "" {
				found = true
				if !h.Provenance.Semantic {
					t.Errorf("semantic-mode hit %q has no semantic provenance: %+v", h.Symbol, h.Provenance)
				}
			}
		}
		if !found {
			t.Fatalf("semantic mode returned no symbol-named results: %+v", res.Hits)
		}
	})
}

// TestSearchLanguageFilterAndChunkLeg covers 037: (1) a language-scoped
// semantic search filters on the embeddings/chunk_embeddings language column
// (exact index-level filter, the cocoindex partition key), and (2) the
// chunk-vector leg surfaces a chunk-level semantic hit.
func TestSearchLanguageFilterAndChunkLeg(t *testing.T) {
	ctx := context.Background()
	db := openHybridTestDB(t)
	insertHybridFixture(t, db)
	emb := embedder.NewHash(64)

	// Give file 1's symbol a "go" language + a chunk embedding, so the
	// language filter and chunk leg have rows to operate on.
	if _, err := db.Exec(`UPDATE embeddings SET language='go' WHERE 1=0`); err != nil {
		t.Fatalf("touch embeddings.language: %v", err)
	}
	var symID int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name='parseConfig'`).Scan(&symID); err != nil {
		t.Fatalf("symbol id: %v", err)
	}
	vec, err := emb.Embed(ctx, "parseConfig settings")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO embeddings (symbol_id, vector, model, dim, language) VALUES (?, ?, ?, ?, 'go')`, symID, memory.EncodeVector(vec), emb.Model(), emb.Dimension()); err != nil {
		t.Fatalf("seed embedding: %v", err)
	}
	var chunkID int64
	if err := db.QueryRow(`SELECT id FROM chunks WHERE file_id=1`).Scan(&chunkID); err != nil {
		t.Fatalf("chunk id: %v", err)
	}
	cvec, err := emb.Embed(ctx, "parseConfig call site in main")
	if err != nil {
		t.Fatalf("embed chunk: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO chunk_embeddings (chunk_id, vector, model, dim, language) VALUES (?, ?, ?, ?, 'go')`, chunkID, memory.EncodeVector(cvec), emb.Model(), emb.Dimension()); err != nil {
		t.Fatalf("seed chunk embedding: %v", err)
	}

	t.Run("language filter scopes the semantic leg", func(t *testing.T) {
		ResetVectorCacheForTest()
		// A non-matching language must yield no semantic hits for the go row.
		res, err := Search(ctx, db, "parseConfig settings", Options{Semantic: true, Language: "python", Embedder: emb})
		if err != nil {
			t.Fatalf("Search python: %v", err)
		}
		for _, h := range res.Hits {
			if h.Symbol == "parseConfig" || h.Kind == "chunk" {
				t.Errorf("language=python should exclude the go rows, got %+v", h)
			}
		}
		// The matching language surfaces the go symbol.
		resGo, err := Search(ctx, db, "parseConfig settings", Options{Semantic: true, Language: "go", Embedder: emb})
		if err != nil {
			t.Fatalf("Search go: %v", err)
		}
		var found bool
		for _, h := range resGo.Hits {
			if h.Symbol == "parseConfig" {
				found = true
			}
		}
		if !found {
			t.Errorf("language=go should surface the go symbol, got %+v", resGo.Hits)
		}
	})

	t.Run("chunk-vector leg surfaces a chunk hit", func(t *testing.T) {
		ResetVectorCacheForTest()
		// A distinctive chunk whose text shares tokens with the query makes it
		// top the chunk-vector leg (the hash embedder scores by token overlap).
		// A distinctive chunk + query (no token overlap with any symbol) makes
		// the chunk the only semantic hit, proving the chunk-vector leg fires.
		var cid int64
		if _, err := db.Exec(`INSERT INTO chunks (file_id, start_line, end_line, text, content_hash) VALUES (1, 30, 34, 'zzzwidget qqqthing rrrstuff', 'k3')`); err != nil {
			t.Fatalf("insert chunk: %v", err)
		}
		if err := db.QueryRow(`SELECT id FROM chunks WHERE content_hash='k3'`).Scan(&cid); err != nil {
			t.Fatalf("chunk id: %v", err)
		}
		zvec, err := emb.Embed(ctx, "zzzwidget qqqthing rrrstuff")
		if err != nil {
			t.Fatalf("embed chunk: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO chunk_embeddings (chunk_id, vector, model, dim, language) VALUES (?, ?, ?, ?, 'go')`, cid, memory.EncodeVector(zvec), emb.Model(), emb.Dimension()); err != nil {
			t.Fatalf("seed chunk embedding: %v", err)
		}
		res, err := Search(ctx, db, "zzzwidget qqqthing rrrstuff", Options{Semantic: true, Language: "go", Embedder: emb})
		if err != nil {
			t.Fatalf("Search chunk: %v", err)
		}
		var foundChunk bool
		for _, h := range res.Hits {
			if h.Kind == "chunk" {
				foundChunk = true
			}
		}
		if !foundChunk {
			t.Errorf("chunk-vector leg should surface a chunk hit, got %+v", res.Hits)
		}
	})
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
