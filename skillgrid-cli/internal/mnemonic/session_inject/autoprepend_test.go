package session_inject

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

func seedEndedSession(t *testing.T, svc *memory.Service, project string, events int) string {
	t.Helper()
	ctx := context.Background()
	sid := "sess-ended-1"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := svc.DB().ExecContext(ctx,
		`INSERT OR REPLACE INTO sessions (id, project, directory, started_at, ended_at, status)
		 VALUES (?,?,?,?,?,?)`,
		sid, project, t.TempDir(), now, now, "ended"); err != nil {
		t.Fatalf("insert ended session: %v", err)
	}
	for i := 1; i <= events; i++ {
		if _, err := svc.DB().ExecContext(ctx,
			`INSERT INTO session_events (session_id, project, sequence, action_type, result_status,
			   tool_name, path, "commit", timestamp) VALUES (?,?,?,?,?,?,?,?,?)`,
			sid, project, i, fmt.Sprintf("tool_call_%d", i%3), "success", "edit",
			fmt.Sprintf("src/file%d.go", i), fmt.Sprintf("%040x", i), now); err != nil {
			t.Fatalf("seed ended event %d: %v", i, err)
		}
	}
	return sid
}

func TestAutoPrepend_Resume(t *testing.T) {
	svc := newTestService(t)
	seedEndedSession(t, svc, "test-project", 15)

	got, err := AutoPrepend(context.Background(), svc, "test-project", 800)
	if err != nil {
		t.Fatalf("AutoPrepend: %v", err)
	}
	if got == "" {
		t.Fatalf("expected non-empty summary for a project with an ended session, got %q", got)
	}
	if !strings.Contains(got, "Prior Session Summary") {
		t.Errorf("summary missing header; got:\n%s", got)
	}
}

func TestAutoPrepend_FreshSession(t *testing.T) {
	svc := newTestService(t)
	seedSessionWithEvents(t, svc, 15)

	got, err := AutoPrepend(context.Background(), svc, "test-project", 800)
	if err != nil {
		t.Fatalf("AutoPrepend: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty summary for a fresh project, got:\n%s", got)
	}
}
