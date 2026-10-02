package memory

import (
	"context"
	"testing"
	"time"
)

func TestCheckpointState(t *testing.T) {
	ctx := context.Background()
	st, err := openStoreFor(t, "checkpoint-proj")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	svc := New(st, "checkpoint-proj")
	sid := "sess-checkpoint-1"
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES (?, 'checkpoint-proj', '/tmp', '2026-01-01T00:00:00Z', 'active')`, sid); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	for i := 0; i < 6; i++ {
		ts := time.Date(2026, 1, 1, 12, 0, i, 0, time.UTC).Format(time.RFC3339)
		if _, err := st.DB.ExecContext(ctx,
			`INSERT INTO session_events (session_id, project, sequence, action_type, result_status, timestamp)
			 VALUES (?, 'checkpoint-proj', ?, 'tool_call', 'success', ?)`,
			sid, i, ts,
		); err != nil {
			t.Fatalf("insert event %d: %v", i, err)
		}
	}

	st0, err := svc.CheckpointState(ctx, sid)
	if err != nil {
		t.Fatalf("CheckpointState: %v", err)
	}
	if !st0.Exists {
		t.Fatal("expected session to exist")
	}
	if st0.SessionID != sid {
		t.Errorf("SessionID = %q, want %q", st0.SessionID, sid)
	}
	if st0.EventsSinceWrite != 6 {
		t.Errorf("EventsSinceWrite = %d, want 6", st0.EventsSinceWrite)
	}

	if err := svc.SessionSummary(ctx, sid, "## Goal\ndone"); err != nil {
		t.Fatalf("SessionSummary: %v", err)
	}
	st1, err := svc.CheckpointState(ctx, sid)
	if err != nil {
		t.Fatalf("CheckpointState after summary: %v", err)
	}
	if st1.EventsSinceWrite != 0 {
		t.Errorf("EventsSinceWrite after summary = %d, want 0", st1.EventsSinceWrite)
	}
	if st1.LastWriteAt.IsZero() {
		t.Error("expected LastWriteAt set after SessionSummary")
	}

	claimAt := time.Date(2026, 2, 1, 15, 4, 5, 0, time.UTC)
	if err := svc.ClaimCheckpoint(ctx, sid, claimAt); err != nil {
		t.Fatalf("ClaimCheckpoint: %v", err)
	}
	st2, err := svc.CheckpointState(ctx, sid)
	if err != nil {
		t.Fatalf("CheckpointState after claim: %v", err)
	}
	if !st2.LastClaimedAt.Equal(claimAt) {
		t.Errorf("LastClaimedAt = %v, want %v", st2.LastClaimedAt, claimAt)
	}
}

func TestCheckpointState_UnknownSession(t *testing.T) {
	ctx := context.Background()
	st, err := openStoreFor(t, "checkpoint-unknown")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	svc := New(st, "checkpoint-unknown")

	st0, err := svc.CheckpointState(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("CheckpointState unknown: %v", err)
	}
	if st0.Exists {
		t.Error("expected Exists false for unknown session")
	}
}
