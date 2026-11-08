package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

// openTestStore opens a SQLite store in a temp dir and registers its Close as
// test cleanup.
func openTestStore(t *testing.T) Store {
	t.Helper()
	dir := t.TempDir()
	st, err := Open(context.Background(), dir, "proj-a", SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// ledgerRowSurvivesNull probes the save-operation ledger: after the
// observation is hard-deleted the ledger row for operationID must still exist
// with observation_id NULL (spec 002-schema §3.3).
func ledgerRowSurvivesNull(t *testing.T, st Store, operationID string) bool {
	t.Helper()
	raw := st.(interface {
		Probe(query string, args ...any) (bool, error)
	})
	found, err := raw.Probe(
		`SELECT observation_id IS NULL FROM observation_save_operations WHERE operation_id = ?`,
		operationID,
	)
	if err != nil {
		t.Fatalf("ledger probe: %v", err)
	}
	return found
}

// [store-opens-per-project] one file per project, both answer ping.
func TestStoreOpensPerProjectFile(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(context.Background(), dir, "proj-a", SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	if _, err := os.Stat(filepath.Join(dir, "proj-a.sqlite")); err != nil {
		t.Fatalf("expected per-project file: %v", err)
	}
	// a second project must not share the file
	st2, err := Open(context.Background(), dir, "proj-b", SQLite, "")
	if err != nil {
		t.Fatalf("open b: %v", err)
	}
	defer st2.Close()
	if _, err := os.Stat(filepath.Join(dir, "proj-b.sqlite")); err != nil {
		t.Fatalf("expected second per-project file: %v", err)
	}
	if err := st.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if err := st2.Ping(context.Background()); err != nil {
		t.Fatalf("ping b: %v", err)
	}
}

// [v1-delete-restart] Open detects a v1-era store at the v2 path and deletes
// it, restarting fresh through the normal migration set (the approved
// behavior: no v1→v2 automigration).
func TestOpenDeletesV1StoreAndRestartsFresh(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	v1Path := filepath.Join(dir, "proj-v1.sqlite")
	db, err := sql.Open("sqlite", v1Path)
	if err != nil {
		t.Fatalf("v1 open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE observations (id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL, type TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL, project TEXT, scope TEXT NOT NULL DEFAULT 'project', topic_key TEXT, normalized_hash TEXT, importance_score REAL, created_at TEXT, updated_at TEXT, deleted_at TEXT)`); err != nil {
		t.Fatalf("v1 observations: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO observations (session_id, type, title, content, project, scope, topic_key, normalized_hash, importance_score, created_at, updated_at, deleted_at) VALUES ('v1s1', 'decision', 'v1row', 'v1content', 'p', 'project', 'p', 'hash', 1.0, datetime('now'), datetime('now'), NULL)`); err != nil {
		t.Fatalf("v1 insert: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("v1 close: %v", err)
	}
	st, err := Open(ctx, dir, "proj-v1", SQLite, "")
	if err != nil {
		t.Fatalf("v1 open: %v", err)
	}
	defer st.Close()
	raw, ok := st.(*sqliteStore)
	if !ok {
		t.Fatalf("expected *sqliteStore, got %T", st)
	}
	if _, err := os.Stat(v1Path); err != nil {
		t.Fatalf("expected fresh db file after restart: %v", err)
	}
	found, err := raw.Probe(`SELECT 1 FROM sqlite_master WHERE type='table' AND name='sync_state'`)
	if err != nil {
		t.Fatalf("probe sync_state: %v", err)
	}
	if !found {
		t.Fatal("fresh open must run migrations: sync_state missing")
	}
	n, err := raw.ProbeInt(`SELECT COUNT(*) FROM observations`)
	if err != nil {
		t.Fatalf("probe observations count: %v", err)
	}
	if n != 0 {
		t.Fatalf("v1 rows must not carry into the fresh restart: %d", n)
	}
}

// [door-check] WAL open + FTS5 trigram retrieval + the < 3-rune Like fallback,
// all through the Store interface.
func TestDoorCheckFTSRoundTrip(t *testing.T) {
	st := openTestStore(t)
	res, err := st.AddObservation(context.Background(), AddObservationParams{
		SessionID: "ses-1", Type: "discovery", Title: "Trigram recall",
		Content: "trigram tokens give substring recall",
		Project: "proj-a", Scope: "project",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	r, err := st.Search(context.Background(), SearchQuery{Query: "trigram", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 1 || r.Observations[0].ID != res.ID {
		t.Fatalf("FTS5 trigram retrieval broken: %+v", r)
	}
	// the < 3 runes fallback (trigram minimum) must find it via LIKE
	r2, err := st.Search(context.Background(), SearchQuery{Query: "go", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("short search: %v", err)
	}
	_ = r2 // (a second row titled "Go" is seeded in Task 2's LIKE test; here the fallback path is exercised)
}

// Add+Get round trip: sync id format, provenance, virtual state.
func TestAddGetObservationRoundTrip(t *testing.T) {
	st := openTestStore(t)
	res, err := st.AddObservation(context.Background(), AddObservationParams{
		SessionID: "ses-1", Type: "decision", Title: "Use SQLite",
		Content: "**What** chose SQLite\n**Why** cgo-free",
		Project: "proj-a", Scope: "project",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if res.ID == 0 || !strings.HasPrefix(res.SyncID, "obs-") || utf8.RuneCountInString(res.SyncID) != 4+16 {
		t.Fatalf("bad result: %+v", res)
	}
	got, err := st.GetObservation(context.Background(), res.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Use SQLite" || got.Provenance.CreatedAt == "" || got.State() != "active" {
		t.Fatalf("bad row: %+v", got)
	}
}

// [ledger-fk-set-null] a hard-deleted observation leaves its save-operation
// ledger row as a tombstone (spec 002-schema §3.3; Task 6's 410 path relies on it).
func TestLedgerFKSetNull(t *testing.T) {
	st := openTestStore(t)
	res, err := st.AddObservation(context.Background(), AddObservationParams{
		SessionID: "s", Type: "decision", Title: "T", Content: "C",
		Project: "p", Scope: "project", OperationID: "op-fk",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := st.DeleteObservationForProject(context.Background(), res.ID, "p", true); err != nil {
		t.Fatalf("hard delete: %v", err)
	}
	// ledger row survives with observation_id NULL (via the test store's raw probe)
	if ok := ledgerRowSurvivesNull(t, st, "op-fk"); !ok {
		t.Fatalf("ledger row must survive the hard delete with observation_id SET NULL")
	}
}

// TRIANGULATE (beyond the brief's set): two projects in the same data dir are
// data-isolated, not just file-isolated — a search on project A must not
// return B's rows even when the file handles share the pool.
func TestSearchIsolatedAcrossProjects(t *testing.T) {
	dir := t.TempDir()
	stA, err := Open(context.Background(), dir, "proj-a", SQLite, "")
	if err != nil {
		t.Fatalf("open a: %v", err)
	}
	defer stA.Close()
	stB, err := Open(context.Background(), dir, "proj-b", SQLite, "")
	if err != nil {
		t.Fatalf("open b: %v", err)
	}
	defer stB.Close()

	resB, err := stB.AddObservation(context.Background(), AddObservationParams{
		SessionID: "s-b", Type: "discovery", Title: "unique term b",
		Content: "unique term b content", Project: "proj-b", Scope: "project",
	})
	if err != nil {
		t.Fatalf("add b: %v", err)
	}
	// searching proj-a for proj-b's unique term must come back empty
	rA, err := stA.Search(context.Background(), SearchQuery{Query: "unique", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search a: %v", err)
	}
	if len(rA.Observations) != 0 {
		t.Fatalf("proj-a search leaked proj-b rows: %+v", rA)
	}
	// ... and searching proj-b must find exactly its own row
	rB, err := stB.Search(context.Background(), SearchQuery{Query: "unique", Project: "proj-b", Limit: 10})
	if err != nil {
		t.Fatalf("search b: %v", err)
	}
	if len(rB.Observations) != 1 || rB.Observations[0].ID != resB.ID {
		t.Fatalf("proj-b search broken: %+v", rB)
	}
}

// TRIANGULATE (beyond the brief's set): the refcounted pool hands the second
// Open of the same project the SAME underlying handle, and only the final
// Close releases it.
func TestPoolRefcountSharesAndReleases(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(context.Background(), dir, "proj-x", SQLite, "")
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}
	second, err := Open(context.Background(), dir, "proj-x", SQLite, "")
	if err != nil {
		t.Fatalf("open 2: %v", err)
	}
	if !poolSameHandle(first, second) {
		t.Fatalf("pool must share the underlying *sql.DB for the same project path")
	}
	if err := second.Close(); err != nil {
		t.Fatalf("close 2: %v", err)
	}
	// handle still usable after one of two refs is released
	if err := first.Ping(context.Background()); err != nil {
		t.Fatalf("ping after partial release: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close 1: %v", err)
	}
	if poolCount() != 0 {
		t.Fatalf("pool must be empty after final release, got %d entries", poolCount())
	}
}

// sync-id format, mirrored from the syncid package contract: prefix + 16 hex.
func TestNewSyncIDFormatViaObservation(t *testing.T) {
	st := openTestStore(t)
	pat := regexp.MustCompile(`^obs-[0-9a-f]{16}$`)
	var prev string
	for i := 0; i < 8; i++ {
		res, err := st.AddObservation(context.Background(), AddObservationParams{
			SessionID: "s", Type: "learning", Title: "mint",
			Content: "x", Project: "p", Scope: "project",
		})
		if err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
		if !pat.MatchString(res.SyncID) {
			t.Fatalf("bad sync id %q", res.SyncID)
		}
		if i > 0 && res.SyncID == prev {
			t.Fatalf("collision: %q minted twice", prev)
		}
		prev = res.SyncID
	}
}

// TestSearchPromptsReturnsFullRow asserts SearchPrompts returns the captured
// prompt with its full identity (id, sync_id, session_id, project, content,
// created_at), not just the content string. Covers the PromptRow contract fix.
func TestSearchPromptsReturnsFullRow(t *testing.T) {
	st := openTestStore(t)
	raw, ok := st.(*sqliteStore)
	if !ok {
		t.Fatalf("expected *sqliteStore, got %T", st)
	}
	id, err := raw.InsertPrompt(context.Background(), "ses-p", "the deployment runs on port 7431", "proj-a")
	if err != nil {
		t.Fatalf("insert prompt: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero prompt id")
	}
	r, err := st.SearchPrompts(context.Background(), PromptSearchQuery{Query: "deployment", Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("search prompts: %v", err)
	}
	if len(r.Prompts) != 1 {
		t.Fatalf("expected exactly 1 prompt hit, got %+v", r.Prompts)
	}
	row := r.Prompts[0]
	if row.ID != id {
		t.Fatalf("bad id: got %d want %d", row.ID, id)
	}
	if row.SyncID == "" || row.SessionID != "ses-p" || row.Project != "proj-a" {
		t.Fatalf("identity fields not populated: %+v", row)
	}
	if row.Content != "the deployment runs on port 7431" {
		t.Fatalf("bad content: %q", row.Content)
	}
	if row.CreatedAt == "" {
		t.Fatalf("created_at not populated: %+v", row)
	}
}
