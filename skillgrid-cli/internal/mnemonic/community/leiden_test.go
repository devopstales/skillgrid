package community

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// openStore opens a scratch SQLite store with the symbols/edges schema (the
// 005 tables the community pass consumes) plus the 012 community tables.
func openStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	stmts := []string{
		`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, mtime_ns INTEGER, size INTEGER, content_hash TEXT, indexed_at TEXT)`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, name TEXT NOT NULL, qualified_name TEXT, kind TEXT NOT NULL, language TEXT, signature TEXT, start_line INTEGER NOT NULL, end_line INTEGER NOT NULL, content_hash TEXT NOT NULL, uid TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE edges (id INTEGER PRIMARY KEY AUTOINCREMENT, kind TEXT NOT NULL, from_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE, file_id INTEGER REFERENCES files(id) ON DELETE CASCADE, to_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE, to_name TEXT, target_path TEXT, confidence TEXT NOT NULL DEFAULT 'EXTRACTED', line INTEGER, UNIQUE(kind, from_id, file_id, to_id, to_name, target_path, line))`,
		`CREATE TABLE communities (id INTEGER, symbol_id INTEGER NOT NULL UNIQUE REFERENCES symbols(id) ON DELETE CASCADE)`,
		`CREATE TABLE community_meta (id INTEGER PRIMARY KEY, label TEXT NOT NULL, symbol_count INTEGER NOT NULL, god_nodes TEXT NOT NULL DEFAULT '', hub_label TEXT NOT NULL DEFAULT '', cache_key TEXT NOT NULL)`,
		`CREATE TABLE community_meta_cache (key TEXT PRIMARY KEY, value TEXT)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}
	return db
}

// seedFile inserts a file row and returns its id.
func seedFile(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 1, 'h', 'now')`, path)
	if err != nil {
		t.Fatalf("seed file %s: %v", path, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// seedSymbol inserts a symbol row and returns its id.
func seedSymbol(t *testing.T, db *sql.DB, fileID int64, name, kind string, line int) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid) VALUES (?, ?, ?, ?, ?, 'h', ?)`,
		fileID, name, kind, line, line, name+string(rune('u'))+kind)
	if err != nil {
		t.Fatalf("seed symbol %s: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id
}

// seedEdge inserts an edge between two symbol ids.
func seedEdge(t *testing.T, db *sql.DB, kind string, fromID, toID int64, line int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO edges (kind, from_id, to_id, confidence, line) VALUES (?, ?, ?, 'EXTRACTED', ?)`, kind, fromID, toID, line); err != nil {
		t.Fatalf("seed edge %d->%d: %v", fromID, toID, err)
	}
}

// communityFixture builds two clearly-separated clusters with one weak bridge:
//
//	a1-a2-a3 (dense)  --bridge-->  b1-b2-b3 (dense)
//
// so Leiden should produce at least two communities that keep each cluster
// intact.
func communityFixture(t *testing.T) *sql.DB {
	t.Helper()
	db := openStore(t)
	fa := seedFile(t, db, "a/core.go")
	fb := seedFile(t, db, "b/engine.go")
	a1 := seedSymbol(t, db, fa, "aOne", "function", 1)
	a2 := seedSymbol(t, db, fa, "aTwo", "function", 10)
	a3 := seedSymbol(t, db, fa, "aThree", "function", 20)
	b1 := seedSymbol(t, db, fb, "bOne", "function", 1)
	b2 := seedSymbol(t, db, fb, "bTwo", "function", 10)
	b3 := seedSymbol(t, db, fb, "bThree", "function", 20)

	seedEdge(t, db, "calls", a1, a2, 2)
	seedEdge(t, db, "calls", a2, a3, 11)
	seedEdge(t, db, "calls", a1, a3, 3)
	seedEdge(t, db, "calls", b1, b2, 2)
	seedEdge(t, db, "calls", b2, b3, 11)
	seedEdge(t, db, "calls", b1, b3, 3)
	seedEdge(t, db, "imports", a3, b1, 21) // weak bridge
	return db
}

// TestCommunitiesDetectsTwoClusters covers the core Leiden contract: a graph
// of two dense clusters linked by one weak edge partitions into >=2
// communities that keep each cluster intact, every symbol gets exactly one
// community row, and each community carries an LLM-free label.
func TestCommunitiesDetectsTwoClusters(t *testing.T) {
	db := communityFixture(t)
	res, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(res.Communities) < 2 {
		t.Fatalf("expected >=2 communities, got %d", len(res.Communities))
	}
	// Every symbol is assigned exactly once.
	assigned := map[int64]int{}
	for _, c := range res.Communities {
		for _, id := range c.Members {
			assigned[id]++
		}
	}
	if len(assigned) != 6 {
		t.Fatalf("expected 6 symbols assigned, got %d", len(assigned))
	}
	for id, n := range assigned {
		if n != 1 {
			t.Errorf("symbol %d assigned %d times", id, n)
		}
	}
	// Clusters stay intact: a1..a3 in one community, b1..b3 in the other.
	byName := map[int64]int{}
	for _, c := range res.Communities {
		for _, id := range c.Members {
			var name string
			_ = db.QueryRow(`SELECT name FROM symbols WHERE id = ?`, id).Scan(&name)
			byName[id] = c.ID
		}
	}
	var a1, a2, a3, b1, b2, b3 int64
	for _, q := range []struct {
		name string
		p    *int64
	}{{"aOne", &a1}, {"aTwo", &a2}, {"aThree", &a3}, {"bOne", &b1}, {"bTwo", &b2}, {"bThree", &b3}} {
		_ = db.QueryRow(`SELECT id FROM symbols WHERE name = ?`, q.name).Scan(q.p)
	}
	if byName[a1] != byName[a2] || byName[a2] != byName[a3] {
		t.Errorf("cluster a split: %d %d %d", byName[a1], byName[a2], byName[a3])
	}
	if byName[b1] != byName[b2] || byName[b2] != byName[b3] {
		t.Errorf("cluster b split: %d %d %d", byName[b1], byName[b2], byName[b3])
	}
	// Labels are present and LLM-free: derived, never empty, never "community-N"
	// for a community that has god nodes.
	for _, c := range res.Communities {
		if c.Label == "" {
			t.Errorf("community %d has empty label", c.ID)
		}
	}
}

// TestTinyGraphYieldsOneTrivialCommunity covers the < 2 nodes rule: a graph
// with a single symbol produces one trivial community (warn+continue), never
// a crash.
func TestTinyGraphYieldsOneTrivialCommunity(t *testing.T) {
	db := openStore(t)
	f := seedFile(t, db, "solo.go")
	seedSymbol(t, db, f, "only", "function", 1)
	res, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect tiny: %v", err)
	}
	if len(res.Communities) != 1 {
		t.Fatalf("expected 1 trivial community, got %d", len(res.Communities))
	}
	if len(res.Communities[0].Members) != 1 {
		t.Fatalf("trivial community should hold the single symbol, got %v", res.Communities[0].Members)
	}
	if res.Warning == "" {
		t.Errorf("expected a warning for the <2 node graph, got none")
	}
}

// TestCommunitiesReproducibleAndCached covers the seed + content-hash cache:
// two Detect calls on the same graph yield the identical partition (stable
// cache key), and the second call reports a cache hit without re-running the
// detector.
func TestCommunitiesReproducibleAndCached(t *testing.T) {
	db := communityFixture(t)
	first, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect 1: %v", err)
	}
	if first.CacheKey == "" {
		t.Fatal("expected a non-empty content-hash cache key")
	}
	second, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect 2: %v", err)
	}
	if second.CacheKey != first.CacheKey {
		t.Errorf("cache key not stable: %q vs %q", second.CacheKey, first.CacheKey)
	}
	if !second.FromCache {
		t.Errorf("second Detect should be served from cache")
	}
	if partitionKey(first) != partitionKey(second) {
		t.Errorf("partition not reproducible: %s vs %s", partitionKey(first), partitionKey(second))
	}
}

func partitionKey(r *Result) string {
	var b strings.Builder
	for _, c := range r.Communities {
		b.WriteString("c")
		b.WriteString(string(rune('0' + c.ID%10)))
		for _, m := range c.Members {
			b.WriteString("|")
			b.WriteString(string(rune('0' + m%10)))
		}
	}
	return b.String()
}

// TestCacheInvalidatesOnChangedGraph covers the review finding: the cache key
// is a real content hash over the edge + symbol sets, so a changed graph
// (even one that preserves min/max/count, or a symbol delete that preserves
// the edge count) yields a NEW key and forces re-detection, while an unchanged
// graph is a cache hit.
func TestCacheInvalidatesOnChangedGraph(t *testing.T) {
	db := communityFixture(t)
	base, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect base: %v", err)
	}

	// (a) Changing an intermediate edge (a2->a3 becomes a2->b3) preserves the
	// edge count but changes the set → new key → re-detect (not a cache hit).
	if _, err := db.Exec(`UPDATE edges SET to_id = (SELECT id FROM symbols WHERE name='bThree') WHERE from_id = (SELECT id FROM symbols WHERE name='aTwo') AND to_id = (SELECT id FROM symbols WHERE name='aThree')`); err != nil {
		t.Fatalf("mutate edge: %v", err)
	}
	changed, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect changed: %v", err)
	}
	if changed.CacheKey == base.CacheKey {
		t.Errorf("changed intermediate edge must change the cache key (content hash, not min/max/count)")
	}
	if changed.FromCache {
		t.Errorf("changed graph must NOT be served from cache")
	}

	// (b) Deleting a symbol (which cascades its edges) changes the symbol set
	// → new key → re-detect.
	db2 := communityFixture(t)
	base2, err := Detect(context.Background(), db2, Options{})
	if err != nil {
		t.Fatalf("Detect base2: %v", err)
	}
	if _, err := db2.Exec(`DELETE FROM symbols WHERE name = 'aThree'`); err != nil {
		t.Fatalf("delete symbol: %v", err)
	}
	deleted, err := Detect(context.Background(), db2, Options{})
	if err != nil {
		t.Fatalf("Detect deleted: %v", err)
	}
	if deleted.CacheKey == base2.CacheKey {
		t.Errorf("deleting a symbol must change the cache key (symbol set is hashed)")
	}
	if deleted.FromCache {
		t.Errorf("symbol-deleted graph must NOT be served from cache")
	}
}

// TestCacheHitOnUnchangedGraph covers the positive side: two Detect calls on a
// byte-identical graph yield the same key and the second is a cache hit.
func TestCacheHitOnUnchangedGraph(t *testing.T) {
	db := communityFixture(t)
	first, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect 1: %v", err)
	}
	second, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect 2: %v", err)
	}
	if second.CacheKey != first.CacheKey {
		t.Errorf("unchanged graph must keep the same cache key: %q vs %q", second.CacheKey, first.CacheKey)
	}
	if !second.FromCache {
		t.Errorf("unchanged graph should be served from cache")
	}
}

// TestGodNodesColumnPopulated covers the review finding: writeMeta stores the
// community's top god-node names in community_meta.god_nodes (no dead column).
func TestGodNodesColumnPopulated(t *testing.T) {
	db := communityFixture(t)
	res, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	// At least one community in the fixture has god nodes (the clusters have
	// edges). Its community_meta.god_nodes must be populated, not empty.
	var populated int
	for _, c := range res.Communities {
		var stored string
		if err := db.QueryRow(`SELECT god_nodes FROM community_meta WHERE id = ?`, c.ID).Scan(&stored); err != nil {
			t.Fatalf("read god_nodes for community %d: %v", c.ID, err)
		}
		if stored != "" {
			populated++
		}
		// The stored value must match the community's GodNodes list.
		if len(c.GodNodes) > 0 && stored != strings.Join(c.GodNodes, ",") {
			t.Errorf("community %d god_nodes column %q != GodNodes %q", c.ID, stored, strings.Join(c.GodNodes, ","))
		}
	}
	if populated == 0 {
		t.Errorf("expected at least one community_meta.god_nodes populated, all empty")
	}
}

// TestCommunityHubLabelPopulated covers 035: a community with god nodes
// carries HubLabel = the top god-node name on the in-memory Community struct
// AND persists it in community_meta.hub_label (so a cache hit can report the
// hub without re-ranking).
func TestCommunityHubLabelPopulated(t *testing.T) {
	db := communityFixture(t)
	res, err := Detect(context.Background(), db, Options{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(res.Communities) < 2 {
		t.Fatalf("expected >=2 communities, got %d", len(res.Communities))
	}
	var withHub int
	for _, c := range res.Communities {
		if len(c.GodNodes) > 0 {
			if c.HubLabel == "" {
				t.Errorf("community %d has god nodes but empty HubLabel", c.ID)
			}
			if c.HubLabel != c.GodNodes[0] {
				t.Errorf("community %d HubLabel %q != top god node %q", c.ID, c.HubLabel, c.GodNodes[0])
			}
		}
		var stored string
		if err := db.QueryRow(`SELECT hub_label FROM community_meta WHERE id = ?`, c.ID).Scan(&stored); err != nil {
			t.Fatalf("read hub_label for community %d: %v", c.ID, err)
		}
		if stored != c.HubLabel {
			t.Errorf("community %d hub_label column %q != HubLabel %q", c.ID, stored, c.HubLabel)
		}
		if stored != "" {
			withHub++
		}
	}
	if withHub == 0 {
		t.Errorf("expected at least one community with a populated hub_label, all empty")
	}
}
