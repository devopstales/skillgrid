package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedSessionsActivityStore seeds two sessions with distinct observations so
// the session-scoped activity endpoint has rows to filter.
func seedSessionsActivityStore(t *testing.T, dataDir string) {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	seed := `
		INSERT INTO sessions (id, project, directory, started_at, ended_at, summary, status, title)
		VALUES ('sa-sess-A', 'http-test', '.', '2026-09-10T09:00:00Z', NULL, '## Goal A', 'active', 'Session A'),
		       ('sa-sess-B', 'http-test', '.', '2026-09-10T10:00:00Z', NULL, '## Goal B', 'active', 'Session B');

		INSERT INTO observations (session_id, type, title, content, project, scope, source, created_at, updated_at, normalized_hash)
		VALUES ('sa-sess-A', 'decision', 'A-chose-sqlite', 'a', 'http-test', 'project', 'agent', '2026-09-10T09:05:00Z', '2026-09-10T09:05:00Z', 'sa-h1'),
		       ('sa-sess-A', 'bugfix', 'A-fixed-login', 'a', 'http-test', 'project', 'agent', '2026-09-10T09:15:00Z', '2026-09-10T09:15:00Z', 'sa-h2'),
		       ('sa-sess-B', 'discovery', 'B-found-n1', 'b', 'http-test', 'project', 'passive', '2026-09-10T10:05:00Z', '2026-09-10T10:05:00Z', 'sa-h3');
	`
	if _, err := st.DB.Exec(seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func newSessionsActivityServer(t *testing.T) http.Handler {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedSessionsActivityStore(t, dataDir)
	return NewServer(service.New(dataDir)).Handler()
}

// TestSessionScopedActivity verifies GET /sessions/{id}/activity?limit=N returns
// only the requested session's events (server-side scoping), in the same
// {project, events, limit} shape as /activity/events.
func TestSessionScopedActivity(t *testing.T) {
	h := newSessionsActivityServer(t)

	// Scoped to sa-sess-A → only that session's 2 events, newest first.
	rr := doGet(t, h, "/sessions/sa-sess-A/activity?project="+proj+"&limit=50")
	if rr.Code != http.StatusOK {
		t.Fatalf("scoped: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var em map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &em); err != nil {
		t.Fatalf("scoped: unmarshal: %v", err)
	}
	evs, _ := em["events"].([]any)
	if len(evs) != 2 {
		t.Fatalf("scoped sa-sess-A: got %d events, want 2; body=%s", len(evs), rr.Body.String())
	}
	for _, raw := range evs {
		ev := raw.(map[string]any)
		if ev["sessionId"] != "sa-sess-A" {
			t.Errorf("scoped event leaked sessionId=%v, want sa-sess-A", ev["sessionId"])
		}
	}
	if evs[0].(map[string]any)["summary"] != "A-fixed-login" {
		t.Errorf("scoped[0].summary = %v, want 'A-fixed-login' (newest first)", evs[0].(map[string]any)["summary"])
	}

	// Scoped to sa-sess-B → only its 1 event.
	rr = doGet(t, h, "/sessions/sa-sess-B/activity?project="+proj+"&limit=50")
	_ = json.Unmarshal(rr.Body.Bytes(), &em)
	evs, _ = em["events"].([]any)
	if len(evs) != 1 {
		t.Fatalf("scoped sa-sess-B: got %d events, want 1", len(evs))
	}
	if evs[0].(map[string]any)["summary"] != "B-found-n1" {
		t.Errorf("scoped sa-sess-B[0].summary = %v, want 'B-found-n1'", evs[0].(map[string]any)["summary"])
	}

	// A session with no observations → 200 with an empty events array (not 404,
	// matching the /activity/events unknown-project convention).
	rr = doGet(t, h, "/sessions/sa-empty/activity?project="+proj+"&limit=50")
	if rr.Code != http.StatusOK {
		t.Fatalf("empty session: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &em)
	if evs, _ = em["events"].([]any); len(evs) != 0 {
		t.Errorf("empty session: got %d events, want 0", len(evs))
	}
}
