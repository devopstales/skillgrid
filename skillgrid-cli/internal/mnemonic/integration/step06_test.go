package integration

import (
	"context"
	stdhttp "net/http"
	"testing"

	mnemonichttp "github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// step06Server builds a dashboard HTTP handler backed by a seeded memory
// store for the Sessions entry routes (P6).
func step06Server(t *testing.T) (stdhttp.Handler, string, *service.Service, string, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	workspace := seedWorkspace(t)
	svc := service.New(dataDir)
	ctx := context.Background()
	sessID, projectID, err := svc.SessionStart(ctx, workspace, "step06 base")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	return mnemonichttp.NewServer(svc).Handler(), projectID, svc, sessID, workspace
}

// TestStep06_SessionList: GET /sessions returns every session for the project
// (id, title, started_at, status), newest first.
func TestStep06_SessionList(t *testing.T) {
	h, projectID, svc, baseID, workspace := step06Server(t)
	q := "?project=" + projectID
	ctx := context.Background()

	// A second session in the SAME workspace (same project) with a distinct
	// title — a session in a different directory would resolve to a different
	// project store and would not appear in this project's list.
	second, proj2, err := svc.SessionStart(ctx, workspace, "step06 second")
	if err != nil {
		t.Fatalf("second session start: %v", err)
	}
	if proj2 != projectID {
		t.Fatalf("second session resolved to project %q, want %q", proj2, projectID)
	}

	rr, out := doHTTP(t, h, stdhttp.MethodGet, "/sessions"+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("GET /sessions: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	list, _ := out["sessions"].([]any)
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d: %v", len(list), out)
	}
	ids := map[string]bool{}
	for _, item := range list {
		m, _ := item.(map[string]any)
		if m == nil {
			t.Fatalf("session item is not an object: %v", item)
		}
		id, _ := m["id"].(string)
		if id == "" {
			t.Errorf("session item missing id: %v", m)
		}
		ids[id] = true
		for _, field := range []string{"title", "started_at", "status"} {
			if m[field] == nil {
				t.Errorf("session item missing %q: %v", field, m)
			}
		}
	}
	if !ids[baseID] || !ids[second] {
		t.Errorf("expected session ids %q and %q in list, got %v", baseID, second, ids)
	}
}

// TestStep06_SessionSummary: GET /sessions/{id}/summary returns the stored
// summary for a session; an unknown session → 404.
func TestStep06_SessionSummary(t *testing.T) {
	h, projectID, svc, sessID, _ := step06Server(t)
	q := "?project=" + projectID
	ctx := context.Background()

	// End the session with a summary (the write path records sessions.summary).
	if err := handleFor(t, svc, projectID).Memory().SessionEnd(ctx, sessID, "## Goal\nstep06 wrap-up"); err != nil {
		t.Fatalf("session end: %v", err)
	}

	rr, out := doHTTP(t, h, stdhttp.MethodGet, "/sessions/"+sessID+"/summary"+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("GET /sessions/{id}/summary: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if s, _ := out["summary"].(string); s != "## Goal\nstep06 wrap-up" {
		t.Errorf("summary = %q, want the stored summary", s)
	}
	if id, _ := out["id"].(string); id != sessID {
		t.Errorf("id = %q, want %q", id, sessID)
	}
	if st, _ := out["status"].(string); st != "ended" {
		t.Errorf("status = %q, want ended", st)
	}

	// Unknown session → 404.
	rr, _ = doHTTP(t, h, stdhttp.MethodGet, "/sessions/nope/summary"+q, nil)
	if rr.Code != stdhttp.StatusNotFound {
		t.Fatalf("unknown session: expected 404, got %d", rr.Code)
	}
}
