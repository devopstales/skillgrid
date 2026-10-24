// GET /memory/global HTTP seam (change 2026-10-24-global-llm-wiki,
// TICKET-05): the OpenCode plugin's frozen-snapshot source.
//
// RED notes (the failures that prove each test exercises new code):
//   - TestGlobalMemoryPrefetchRoute: 404 — the route does not exist yet.
package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// TestGlobalMemoryPrefetchRoute: GET /memory/global renders the __global__
// store via facts.RenderFrozenSnapshot (both section headers, the seeded
// fact) and returns HTTP 200.
func TestGlobalMemoryPrefetchRoute(t *testing.T) {
	s, dataDir := newFactsServer(t)
	defer closeStore(t, dataDir)

	// Seed one live fact directly into the __global__ store (bypassing the
	// session trail: the route is a read path).
	svc := service.New(dataDir)
	gh, cleanup, err := svc.OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal: %v", err)
	}
	if _, err := gh.Store().DB.Exec(
		`INSERT INTO facts (content, category, created_at, updated_at)
		 VALUES (?, ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		"global prefetch probe fact", facts.CategoryMemory,
	); err != nil {
		t.Fatalf("seed global fact: %v", err)
	}
	cleanup()

	var out struct {
		Snapshot string `json:"snapshot"`
	}
	status := func() int {
		req := httptest.NewRequest(http.MethodGet, "/memory/global", nil)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("unmarshal: %v (%s)", err, rec.Body.String())
		}
		return rec.Code
	}()
	if status != http.StatusOK {
		t.Fatalf("status %d, want 200", status)
	}
	if !strings.Contains(out.Snapshot, "GLOBAL MEMORY") || !strings.Contains(out.Snapshot, "USER PROFILE") {
		t.Errorf("snapshot missing a section header:\n%s", out.Snapshot)
	}
	if !strings.Contains(out.Snapshot, "global prefetch probe fact") {
		t.Errorf("snapshot missing the seeded fact:\n%s", out.Snapshot)
	}
}

// TestGlobalMemoryPrefetchEmptyStore: an empty __global__ store renders both
// headers at 0% with no entries (HTTP 200).
func TestGlobalMemoryPrefetchEmptyStore(t *testing.T) {
	s, _ := newFactsServer(t)

	req := httptest.NewRequest(http.MethodGet, "/memory/global", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Snapshot string `json:"snapshot"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v (%s)", err, rec.Body.String())
	}
	if !strings.Contains(out.Snapshot, "GLOBAL MEMORY") || !strings.Contains(out.Snapshot, "USER PROFILE") {
		t.Errorf("empty-store snapshot missing a section header:\n%s", out.Snapshot)
	}
	if !strings.Contains(out.Snapshot, "0%") {
		t.Errorf("empty-store snapshot must show 0%% usage:\n%s", out.Snapshot)
	}
}
