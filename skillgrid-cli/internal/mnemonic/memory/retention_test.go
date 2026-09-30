package memory

import (
	"context"
	"testing"
	"time"
)

// seedEventsWithAges inserts one sessions row (satisfying the session_events
// FK) then one session_events row per age (in days ago) with the given
// project. The session_start event that SessionStart appends itself is NOT
// backdated, so it stays recent and is never pruned — the seeded rows are the
// only ones whose age the tests control.
func seedEventsWithAges(t *testing.T, svc *Service, projectID string, agesDays []int) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	sid, err := svc.SessionStart(ctx, dir, "retention-seed")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	// SessionStart already appended a session_start row at sequence 0 (recent,
	// never pruned); seed the crafted-age rows from sequence 1 onward.
	for i, age := range agesDays {
		ts := time.Now().UTC().AddDate(0, 0, -age).Format(time.RFC3339)
		if _, err := svc.store.DB.ExecContext(ctx,
			`INSERT INTO session_events (session_id, project, sequence, action_type, result_status, timestamp)
			 VALUES (?, ?, ?, 'tool_call', 'success', ?)`,
			sid, projectID, i+1, ts,
		); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
}

// TestPruneOldEvents_PrunesOld (BDD retention-prunes-old): 5 events 100 days
// old + 5 recent → prune 90d → 5 deleted, 5 recent remain, and no remaining
// event is older than 90 days.
func TestPruneOldEvents_PrunesOld(t *testing.T) {
	_, svc := newTestStore(t, "retproj")
	seedEventsWithAges(t, svc, "retproj", []int{100, 100, 100, 100, 100, 1, 1, 1, 1, 1})
	ctx := context.Background()

	deleted, err := svc.PruneOldEvents(ctx, 90)
	if err != nil {
		t.Fatalf("PruneOldEvents: %v", err)
	}
	if deleted != 5 {
		t.Errorf("deleted = %d, want 5", deleted)
	}
	// BDD "no remaining event is older than 90 days": read back every row and
	// assert its timestamp is within the retention window.
	var oldest string
	if err := svc.store.DB.QueryRowContext(ctx,
		`SELECT MIN(timestamp) FROM session_events`).Scan(&oldest); err != nil {
		t.Fatalf("oldest timestamp: %v", err)
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	oldestTS, err := time.Parse(time.RFC3339, oldest)
	if err != nil {
		t.Fatalf("parse oldest timestamp %q: %v", oldest, err)
	}
	if oldestTS.Before(cutoff) {
		t.Errorf("event older than 90d survived: %s", oldest)
	}
}

// TestPruneOldEvents_KeepsRecent (BDD retention-keeps-recent): all recent →
// 0 deleted.
func TestPruneOldEvents_KeepsRecent(t *testing.T) {
	_, svc := newTestStore(t, "retrecent")
	seedEventsWithAges(t, svc, "retrecent", []int{1, 2, 3})
	ctx := context.Background()

	deleted, err := svc.PruneOldEvents(ctx, 90)
	if err != nil {
		t.Fatalf("PruneOldEvents: %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted = %d, want 0", deleted)
	}
}

// TestPruneOldEvents_BestEffort (BDD retention-best-effort): a DB failure
// (closed store) returns an error and does not panic. Retention is not a
// session-critical path — the caller logs and continues.
func TestPruneOldEvents_BestEffort(t *testing.T) {
	st, svc := newTestStore(t, "retbest")
	seedEventsWithAges(t, svc, "retbest", []int{100, 1})
	// Close the underlying DB so the prune's DELETE fails. newTestStore's
	// cleanup also closes the store; that double-close is a no-op.
	_ = st.Close()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PruneOldEvents panicked (best-effort must not): %v", r)
		}
	}()
	deleted, err := svc.PruneOldEvents(context.Background(), 90)
	if err == nil {
		t.Fatalf("PruneOldEvents on a closed DB: got no error, deleted=%d", deleted)
	}
}
