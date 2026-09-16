package http

import (
	"net/http"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// These tests pin the single-open contract for the HTTP API: every
// single-project route opens its project store exactly once per request —
// the route opens a *service.ProjectHandle via s.svc.Open / OpenForDirectory
// and performs the domain op through it (h.Memory()/h.Web()/h.Store()),
// matching the MCP lifecycle from step 02.
//
// store.OpenCount() counts every store.Open call in the process, so a route
// that legitimately opens one store must report 1, and a double-open (route
// open + facade re-open) would report 2.

// TestObservationRoutesOpenStoreOnce proves the observation create/recent
// path opens its project store exactly once per request (Scenario:
// "Observation routes open store once"). A double-open (route open #1 +
// facade openProject #2) would make the count 2.
func TestObservationRoutesOpenStoreOnce(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	// Write path: POST /observations.
	store.ResetOpenCount()
	rr, out := do(t, h, http.MethodPost, "/observations?project="+proj, map[string]any{
		"session_id": "s1",
		"type":       "decision",
		"title":      "single open probe",
		"content":    "created over http, single-open probe",
		"scope":      "project",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /observations: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if id, ok := out["id"].(float64); !ok || id <= 0 {
		t.Fatalf("expected positive id, got %v", out["id"])
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("POST /observations opened the store %d times, want exactly 1 (double-open)", got)
	}

	// Read path: GET /observations/recent.
	store.ResetOpenCount()
	rr, out = do(t, h, http.MethodGet, "/observations/recent?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /observations/recent: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := out["sessions"].([]any); !ok {
		t.Fatalf("expected sessions array, got %v", out["sessions"])
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("GET /observations/recent opened the store %d times, want exactly 1 (double-open)", got)
	}
}

// TestObservationListRouteOpenStoreOnce is a second single-open probe on the
// GET /observations route (memory.Recent through the handle).
func TestObservationListRouteOpenStoreOnce(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	store.ResetOpenCount()
	rr, out := do(t, h, http.MethodGet, "/observations?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /observations: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	obs, ok := out["observations"].([]any)
	if !ok || len(obs) != 1 {
		t.Fatalf("expected one observation, got %v", out["observations"])
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("GET /observations opened the store %d times, want exactly 1 (double-open)", got)
	}
}

// singleOpenRoutes is the matrix of rewired single-project routes (Scenario:
// "Single-project HTTP uses handle"). Each must open the project store exactly
// once and return its unchanged JSON shape.
func TestSingleProjectRoutesUseHandle(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	seedSecond(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	cases := []struct {
		method string
		target string
		body   any
		key    string
	}{
		{http.MethodGet, "/observations/recent?project=" + proj, nil, "sessions"},
		{http.MethodGet, "/observations?project=" + proj, nil, "observations"},
		{http.MethodGet, "/context?project=" + proj, nil, "sessions"},
		{http.MethodGet, "/memory/status?project=" + proj, nil, "observation_count"},
		{http.MethodGet, "/memory/last-save-at?project=" + proj, nil, "last_save_at"},
		{http.MethodGet, "/memory/doctor?project=" + proj, nil, "schema_version"},
		{http.MethodGet, "/memory/timeline?project=" + proj + "&id=1&window=1h", nil, "anchor_id"},
		{http.MethodGet, "/memory/reviews?project=" + proj, nil, "due"},
		{http.MethodGet, "/search?project=" + proj + "&query=obs", nil, "observations"},
		{http.MethodGet, "/relations/1?project=" + proj, nil, "relations"},
		{http.MethodGet, "/relations?project=" + proj + "&src_id=1&dst_id=2", nil, "relations"},
		{http.MethodGet, "/code/status?project=" + proj, nil, "file_count"},
		{http.MethodGet, "/code/files?project=" + proj, nil, "files"},
		{http.MethodGet, "/code/search?project=" + proj + "&query=x", nil, "hits"},
		{http.MethodGet, "/web/lookup?project=" + proj + "&source=context7&library_id=/o&p&query=q", nil, "status"},
		{http.MethodGet, "/web/search?project=" + proj + "&query=x", nil, "entries"},
		{http.MethodGet, "/web/status?project=" + proj, nil, "total_entries"},
		{http.MethodGet, "/sessions/s1?project=" + proj, nil, "id"},
		{http.MethodPatch, "/memory/observations/1?project=" + proj, map[string]any{"content": "updated via single-open probe"}, "updated"},
		{http.MethodPost, "/memory/reviews/1?project=" + proj, nil, "marked_reviewed"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.target, func(t *testing.T) {
			store.ResetOpenCount()
			rr, out := do(t, h, c.method, c.target, c.body)
			if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
				t.Fatalf("expected 200/201, got %d: %s", rr.Code, rr.Body.String())
			}
			if _, ok := out[c.key]; !ok {
				t.Errorf("expected %q key in response, got %v", c.key, out)
			}
			if got := store.OpenCount(); got != 1 {
				t.Errorf("opened the store %d times, want exactly 1 (double-open)", got)
			}
		})
	}
}

// TestSessionEndOpenStoreOnce proves the session-end route (the
// projectID-scoped variant of the directory-rooted session lifecycle) opens
// the project store exactly once.
func TestSessionEndOpenStoreOnce(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	store.ResetOpenCount()
	rr, _ := do(t, h, http.MethodPost, "/sessions/s1/end?project="+proj, map[string]any{"summary": "ended"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("POST /sessions/{id}/end opened the store %d times, want exactly 1 (double-open)", got)
	}
}

// TestMigrateMergeStayOnRoot proves the cross-store migrate/merge routes keep
// working through the root Service (Scenario: "Migrate and merge stay on root")
// and that a single-project route still opens exactly once afterward.
func TestMigrateMergeStayOnRoot(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	// Merge a legacy project into the seeded one (alias recorded even with 0
	// rows moved) — the root cross-store path must run.
	rr, out := do(t, h, http.MethodPost, "/projects/merge", map[string]any{
		"source": "proj-legacy", "canonical": proj,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /projects/merge: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if out["merged"] != true || out["alias_recorded"] != true {
		t.Fatalf("expected merged+alias_recorded true, got %v", out)
	}

	// Migrate is a no-op for a missing source store (root path, 0 moved).
	rr, out = do(t, h, http.MethodPost, "/projects/migrate", map[string]any{
		"old_project": "proj-legacy", "new_project": proj,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /projects/migrate: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if out["rows_moved"] != float64(0) {
		t.Fatalf("expected 0 rows moved, got %v", out["rows_moved"])
	}

	// Single-project route after the cross-store ops: still opens once.
	store.ResetOpenCount()
	rr, out = do(t, h, http.MethodGet, "/observations?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /observations: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := out["observations"].([]any); !ok {
		t.Fatalf("expected observations array, got %v", out["observations"])
	}
	if got := store.OpenCount(); got != 1 {
		t.Errorf("GET /observations after migrate/merge opened the store %d times, want exactly 1", got)
	}
}
