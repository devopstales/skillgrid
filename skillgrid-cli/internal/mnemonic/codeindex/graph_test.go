package codeindex

import (
	"database/sql"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedTemporalStore opens a fresh store and plants one file + two symbols
// (so the edges' from_id / file_id FKs resolve), returning the store plus the
// ids. Each test seeds its own edges on top.
func seedTemporalStore(t *testing.T) (*store.Store, int64, int64, int64) {
	t.Helper()
	st, dataDir, clean := openStoreFor(t)
	t.Cleanup(clean)
	_ = dataDir
	db := st.DB

	var fileID int64
	if err := db.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('temporal.go', 1, 100, 'h', '2026-01-01T00:00:00Z')
		RETURNING id`).Scan(&fileID); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var fromID, toID int64
	if err := db.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, 'alpha', 'function', 1, 5, 'c1', 'uid-alpha')
		RETURNING id`, fileID).Scan(&fromID); err != nil {
		t.Fatalf("insert from symbol: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, 'beta', 'function', 6, 10, 'c2', 'uid-beta')
		RETURNING id`, fileID).Scan(&toID); err != nil {
		t.Fatalf("insert to symbol: %v", err)
	}
	return st, fileID, fromID, toID
}

// seedEdge inserts one edge with explicit temporal bounds, returning its id.
// validTo = 0 means NULL (active).
func seedEdge(t *testing.T, st *store.Store, fileID, fromID, toID int64, kind string, validFrom, validTo int64) int64 {
	t.Helper()
	db := st.DB
	var to sql.NullInt64
	to.Int64 = toID
	to.Valid = true
	var vt sql.NullInt64
	if validTo != 0 {
		vt.Int64 = validTo
		vt.Valid = true
	}
	var id int64
	if err := db.QueryRow(`
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from, valid_to)
		VALUES (?, ?, ?, ?, ?, NULL, 'EXTRACTED', 1, ?, ?)
		RETURNING id`, kind, fromID, fileID, to, "beta", validFrom, vt).Scan(&id); err != nil {
		t.Fatalf("insert edge: %v", err)
	}
	return id
}

// TestEdgeValidFromSetOnCreate is 10.1 [RED] — a newly observed relationship
// carries valid_from ~= now and valid_to NULL (active), and stays visible in
// the normal (current-state) query path.
func TestEdgeValidFromSetOnCreate(t *testing.T) {
	st, fileID, fromID, toID := seedTemporalStore(t)
	before := time.Now().Unix()
	id := seedEdge(t, st, fileID, fromID, toID, "calls", before, 0)
	after := time.Now().Unix()

	var vf int64
	var vt sql.NullInt64
	if err := st.DB.QueryRow(`SELECT valid_from, valid_to FROM edges WHERE id = ?`, id).Scan(&vf, &vt); err != nil {
		t.Fatalf("read edge: %v", err)
	}
	if vt.Valid {
		t.Fatalf("valid_to must be NULL for a fresh edge, got %d", vt.Int64)
	}
	if vf < before || vf > after {
		t.Fatalf("valid_from %d not within [%d, %d]", vf, before, after)
	}

	// Visible in the normal (current-state) query.
	edges, err := QueryEdges(st)
	if err != nil {
		t.Fatalf("QueryEdges: %v", err)
	}
	found := false
	for _, e := range edges {
		if e.ID == id {
			found = true
			if e.Status != "active" {
				t.Errorf("fresh edge status = %q, want active", e.Status)
			}
		}
	}
	if !found {
		t.Errorf("fresh edge not visible in normal query")
	}
}

// TestExpiredEdgesHiddenFromQueries is 10.2 [RED] — an expired edge
// (valid_to in the past) is hidden from the current-state query but present in
// the history query; an active edge (valid_to NULL) is visible in both.
func TestExpiredEdgesHiddenFromQueries(t *testing.T) {
	st, fileID, fromID, toID := seedTemporalStore(t)
	now := time.Now().Unix()

	// Expired: observed long ago, ended in the past.
	expiredID := seedEdge(t, st, fileID, fromID, toID, "calls", now-10000, now-100)
	// Active: observed long ago, never ended.
	activeID := seedEdge(t, st, fileID, fromID, toID, "imports", now-10000, 0)
	// Pending: not yet active (valid_from in the future).
	pendingID := seedEdge(t, st, fileID, fromID, toID, "references", now+10000, 0)

	// Normal (current-state) query: only the active edge.
	cur, err := QueryEdges(st)
	if err != nil {
		t.Fatalf("QueryEdges: %v", err)
	}
	wantCur := map[int64]bool{activeID: true}
	seen := map[int64]bool{}
	for _, e := range cur {
		seen[e.ID] = true
	}
	if !seen[activeID] {
		t.Errorf("active edge missing from current-state query")
	}
	if seen[expiredID] {
		t.Errorf("expired edge leaked into current-state query")
	}
	if seen[pendingID] {
		t.Errorf("pending edge leaked into current-state query")
	}
	_ = wantCur

	// History query: all three.
	hist, err := QueryEdgesWithHistory(st)
	if err != nil {
		t.Fatalf("QueryEdgesWithHistory: %v", err)
	}
	histSeen := map[int64]bool{}
	status := map[int64]string{}
	for _, e := range hist {
		histSeen[e.ID] = true
		status[e.ID] = e.Status
	}
	for _, id := range []int64{expiredID, activeID, pendingID} {
		if !histSeen[id] {
			t.Errorf("edge %d missing from history query", id)
		}
	}
	if status[expiredID] != "expired" {
		t.Errorf("expired edge status = %q, want expired", status[expiredID])
	}
	if status[activeID] != "active" {
		t.Errorf("active edge status = %q, want active", status[activeID])
	}
	if status[pendingID] != "pending" {
		t.Errorf("pending edge status = %q, want pending", status[pendingID])
	}
}

// TestBackfilledEdgesRemainActive verifies the migration backfill: existing
// edges (valid_from = 0) pass the valid_from <= now filter and stay active, so
// pre-migration rows are not hidden by the temporal filter.
func TestBackfilledEdgesRemainActive(t *testing.T) {
	st, fileID, fromID, toID := seedTemporalStore(t)
	// valid_from = 0 (the backfill value for pre-migration rows).
	id := seedEdge(t, st, fileID, fromID, toID, "calls", 0, 0)
	edges, err := QueryEdges(st)
	if err != nil {
		t.Fatalf("QueryEdges: %v", err)
	}
	found := false
	for _, e := range edges {
		if e.ID == id {
			found = true
			if e.Status != "active" {
				t.Errorf("backfilled edge status = %q, want active", e.Status)
			}
		}
	}
	if !found {
		t.Errorf("backfilled (valid_from=0) edge hidden from current-state query")
	}
}
