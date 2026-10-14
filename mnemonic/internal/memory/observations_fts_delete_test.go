package memory

import (
	"context"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestFtsDeleteSurvivingRowsStayIndexed guards SEC1: the FTS removal in
// ftsDeleteObs and the INSTEAD OF DELETE trigger must re-derive the index
// from the live observations_core table, not run the FTS5 'rebuild' command
// against the FTS table's own rows. If a surviving core row has no FTS row
// (e.g. a seed that failed mid-trigger), a delete-then-'rebuild' would drop
// that row's index entries and make it invisible to every MATCH path until
// the next full re-sync.
func TestFtsDeleteSurvivingRowsStayIndexed(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir(), "fts-delete-test")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	db := st.DB
	if _, err := db.ExecContext(ctx, `
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-fts-delete', 'fts-delete-test', '/tmp/fts-delete', ?, 'active')`,
		time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	seed := func(title, content string) (int64, error) {
		return txInsertSplitObservation(ctx, db, splitInsertArgs{
			SessionID: "sess-fts-delete",
			Type:      "decision",
			Title:     title,
			Content:   content,
			Project:   "fts-delete-test",
			Scope:     "project",
			Hash:      title,
			Now:       time.Now().UTC().Format(time.RFC3339),
			Source:    "agent",
			Owner:     "agent",
		})
	}
	count := func(term string) int {
		var n int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, term,
		).Scan(&n); err != nil {
			t.Fatalf("match %q: %v", term, err)
		}
		return n
	}

	keeperID, err := seed("keepmarker zebra observation", "uniquekeeper body")
	if err != nil {
		t.Fatalf("seed keeper: %v", err)
	}
	victimID, err := seed("victimmarker observation", "uniquevictim body")
	if err != nil {
		t.Fatalf("seed victim: %v", err)
	}
	if n := count("keepmarker"); n != 1 {
		t.Fatalf("precondition: keepmarker matches = %d, want 1", n)
	}

	// Break the invariant the 'rebuild' command depended on: a surviving core
	// row with no FTS row.
	if _, err := db.ExecContext(ctx, `DELETE FROM observations_fts WHERE rowid = ?`, keeperID); err != nil {
		t.Fatalf("drop keeper FTS row: %v", err)
	}

	if err := ftsDeleteObs(ctx, db, victimID, "x", "y", "z", "p"); err != nil {
		t.Fatalf("ftsDeleteObs: %v", err)
	}

	// The fix: the rebuild SELECTs observations_core, so the keeper's entry
	// is re-derived even though its FTS row was missing.
	if n := count("keepmarker"); n != 1 {
		t.Errorf("keeper invisible after ftsDeleteObs (rebuild-from-own-rows bug): keepmarker matches = %d, want 1", n)
	}
	if n := count("victimmarker"); n != 0 {
		t.Errorf("victim still indexed after ftsDeleteObs: victimmarker matches = %d, want 0", n)
	}
	var present int64
	if err := db.QueryRowContext(ctx, `SELECT 1 FROM observations_core WHERE id = ?`, keeperID).Scan(&present); err != nil {
		t.Errorf("keeper core row missing: %v", err)
	}
}
