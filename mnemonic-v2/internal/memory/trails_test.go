package memory

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// [save-publishes-event] exactly one save event per committed save, carrying
// the step-10 payload (brief §3; spec 01-memory/001 §5).
func TestSavePublishesEvent(t *testing.T) {
	svc, bus := newTestServiceWithBus(t)
	var got []SaveEvent
	bus.Subscribe(func(e SaveEvent) { got = append(got, e) })
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision",
		Title: "T", Content: "C", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("exactly one event expected, got %d", len(got))
	}
	e := got[0]
	if e.SyncID != r.SyncID || e.Project != "p" || e.Scope != "project" ||
		e.Type != "decision" || e.TopicKey != "auth/model" {
		t.Fatalf("event payload wrong: %+v", e)
	}
	if e.Importance <= 0 {
		t.Fatalf("importance must be the positive step-6 value, got %v", e.Importance)
	}
	if e.Entity != "" {
		t.Fatalf("entity is filled in Task 12; this task must leave it empty, got %q", e.Entity)
	}
	// nil LLM → the save degraded (INV-3), the event carries the flag.
	if !e.Degraded || !r.Degraded {
		t.Fatalf("degraded flag must be on with no LLM seam: result=%+v event=%+v", r, e)
	}
}

// [trail-appended-on-save] the trail consumer appends one entry per committed
// save (v1 session_events pattern: per-session 0-based, monotonically
// increasing sequence, timestamped).
func TestTrailAppendedOnSave(t *testing.T) {
	svc, _ := newTestServiceWithBus(t)
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning",
		Title: "T1", Content: "c1", Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	rows := trailRows(t, svc.st, "s")
	if len(rows) != 1 {
		t.Fatalf("one trail entry expected, got %d (%+v)", len(rows), rows)
	}
	if rows[0].sequence != 0 {
		t.Fatalf("first entry sequence must be 0 (v1 0-based), got %d", rows[0].sequence)
	}
	if rows[0].timestamp == "" {
		t.Fatalf("entry must carry a timestamp")
	}
}

// [missing-session-auto-register] a save into an unknown session registers a
// shared-ownership session row for the resolved project, then appends the
// trail entry under that session (user-approved gate decision; the
// pipeline's step-2 policy, the gate's CheckSession).
func TestMissingSessionAutoRegistersOnSave(t *testing.T) {
	svc, _ := newTestServiceWithBus(t)
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "new-sess", Type: "discovery",
		Title: "T", Content: "C", Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save into unknown session must succeed: %v", err)
	}
	info := sessionInfoOf(t, svc.st, "new-sess")
	if info.ownership != "shared" || info.project != "p" {
		t.Fatalf("auto-register must be shared-ownership for the resolved project: %+v", info)
	}
	rows := trailRows(t, svc.st, "new-sess")
	if len(rows) != 1 || rows[0].sequence != 0 {
		t.Fatalf("trail must land under the auto-registered session: %+v", rows)
	}
}

// TRIANGULATE (publish): a second, different save to the same session emits
// a second event with its own sync id — the bus contract is per-save, not
// one-shot.
func TestSavePublishesOneEventPerSave(t *testing.T) {
	svc, bus := newTestServiceWithBus(t)
	var got []SaveEvent
	bus.Subscribe(func(e SaveEvent) { got = append(got, e) })
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning",
		Title: "A", Content: "aa", Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save 1: %v", err)
	}
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision",
		Title: "B", Content: "bb", Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save 2: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("two saves → two events, got %d", len(got))
	}
	if got[0].SyncID == got[1].SyncID {
		t.Fatalf("events must carry distinct sync ids: %+v", got)
	}
}

// TRIANGULATE (trail): three saves into the same session append three trail
// entries with monotonically increasing sequences (v1's sequence guard).
func TestTrailSequenceIncrementsAcrossSaves(t *testing.T) {
	svc, _ := newTestServiceWithBus(t)
	for i, title := range []string{"X1", "X2", "X3"} {
		if _, err := svc.Save(ctx, SaveRequest{SessionID: "seq", Type: "pattern",
			Title: title, Content: "c" + title, Project: "p", Scope: "project"}); err != nil {
			t.Fatalf("save %d: %v", i+1, err)
		}
	}
	rows := trailRows(t, svc.st, "seq")
	if len(rows) != 3 {
		t.Fatalf("three saves → three trail entries, got %d", len(rows))
	}
	for i, row := range rows {
		if row.sequence != i {
			t.Fatalf("sequence must increment by 1 per save (0-based), got %d at index %d", row.sequence, i)
		}
	}
}

// TRIANGULATE (auto-register): a save into an EXPLICIT pre-existing session
// appends its trail entry without touching the registered row (no re-register,
// no re-stamp of started_at — the user-approved no-re-register case).
func TestSaveIntoExistingSessionDoesNotReRegister(t *testing.T) {
	svc, _ := newTestServiceWithBus(t)
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "pre", Type: "learning",
		Title: "T", Content: "C", Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save 1: %v", err)
	}
	info1 := sessionInfoOf(t, svc.st, "pre")
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "pre", Type: "config",
		Title: "T2", Content: "C2", Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save 2: %v", err)
	}
	info2 := sessionInfoOf(t, svc.st, "pre")
	if info1.start != info2.start || info1.project != info2.project {
		t.Fatalf("second save must not re-register the row: %+v → %+v", info1, info2)
	}
	rows := trailRows(t, svc.st, "pre")
	if len(rows) != 2 {
		t.Fatalf("both saves must append trail entries: %+v", rows)
	}
}

// TRIANGULATE (wiring): a replay that returns early (identical operation id,
// same fingerprint) is NOT a new committed save — the step-10 publish and the
// trail append both run exactly once, on the original commit.
func TestReplayDoesNotRepublishOrRetrail(t *testing.T) {
	svc, bus := newTestServiceWithBus(t)
	var got []SaveEvent
	bus.Subscribe(func(e SaveEvent) { got = append(got, e) })
	req := SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C",
		Project: "p", Scope: "project", OperationID: "op-1"}
	r1, err := svc.Save(ctx, req)
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	r2, err := svc.Save(ctx, req)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !r2.Replayed || r2.SyncID != r1.SyncID {
		t.Fatalf("identical replay must return the committed id: %+v vs %+v", r2, r1)
	}
	if len(got) != 1 {
		t.Fatalf("replay is not a new commit → exactly one event, got %d", len(got))
	}
	rows := trailRows(t, svc.st, "s")
	if len(rows) != 1 {
		t.Fatalf("replay must not append a second trail entry, got %d", len(rows))
	}
}

// TRIANGULATE (isolation): a panicking trail consumer never fails the save
// (spec 01-memory/001 §5 — the bus's failure isolation, now exercised through
// the real save path).
func TestPanickingConsumerNeverFailsSave(t *testing.T) {
	svc, bus := newTestServiceWithBus(t)
	bus.Subscribe(func(e SaveEvent) { panic("trail consumer boom") })
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "learning",
		Title: "T", Content: "C", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("save must survive a panicking consumer: %v", err)
	}
	if r.ID == 0 {
		t.Fatalf("save result lost: %+v", r)
	}
}

// ── assertion helpers (test-only; the RawQuery/RawExec read seams) ─────────

// trailEntry is one row of a session's trail (the assertion surface).
type trailEntry struct {
	sequence  int
	timestamp string
}

// trailRows reads a session's trail entries in sequence order (the
// assertion surface for the trail consumer).
func trailRows(t *testing.T, st store.Store, sessionID string) []trailEntry {
	t.Helper()
	rows, err := rawQuery(t, st,
		`SELECT sequence, timestamp FROM session_trails WHERE session_id = ? ORDER BY sequence`, sessionID)
	if err != nil {
		t.Fatalf("trail read: %v", err)
	}
	out := make([]trailEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, trailEntry{sequence: intVal(r, 0), timestamp: strVal(r, 1)})
	}
	return out
}

// sessionsRow is a sessions row read for the auto-register assertions.
type sessionsRow struct {
	ownership string
	project   string
	start     string
}

// sessionInfoOf reads the sessions row for a session id (ownership + project
// + the original started_at, so a re-register would be visible as a changed
// start stamp).
func sessionInfoOf(t *testing.T, st store.Store, sessionID string) sessionsRow {
	t.Helper()
	rows, err := rawQuery(t, st,
		`SELECT ownership_mode, ifnull(project,''), ifnull(started_at,'') FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		t.Fatalf("session read: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly one sessions row for %q, got %d", sessionID, len(rows))
	}
	return sessionsRow{
		ownership: strVal(rows[0], 0),
		project:   strVal(rows[0], 1),
		start:     strVal(rows[0], 2),
	}
}

// rawQuery narrows the store to the RawQuery read seam (the SQLite store's
// [][]any rows; a store without it is a test bug).
func rawQuery(t *testing.T, st store.Store, query string, args ...any) ([][]any, error) {
	t.Helper()
	p, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		t.Fatal("store lacks the RawQuery test seam")
	}
	return p.RawQuery(query, args...)
}
