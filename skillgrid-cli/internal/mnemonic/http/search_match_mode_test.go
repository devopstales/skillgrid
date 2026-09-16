package http

import (
	"net/http"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestSearchMatchModeTrigram is 26.2 [RED] — the GET /search endpoint accepts
// the `match_mode` query parameter (a 014 step-02 additive parameter) and
// routes it to the scoped search. The endpoint must: (1) accept a trigram
// match_mode without error and return the observations shape (the param is
// wired through SearchWithScope, not rejected); (2) return the seeded row for a
// phrase match_mode (default any); (3) return 400 when no project is supplied
// (the existing contract, unchanged). This pins the new parameter at the HTTP
// boundary the web dashboard + API consumers hit, not just the service seam.
func TestSearchMatchModeTrigram(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	seedHTTPStore(t, dataDir, "search-trigram-proj")
	h := NewServer(svc).Handler()

	// (1) trigram match_mode is accepted (200) and returns the observations
	// shape. The seeded content is porter-tokenized, so a trigram query for an
	// absent term yields 0 hits — the contract is that the param is accepted,
	// not that it matches.
	rr, out := do(t, h, http.MethodGet, "/search?project=search-trigram-proj&query=zzqqxy&match_mode=trigram", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("trigram search expected 200, got %d (body %s)", rr.Code, rr.Body.String())
	}
	if _, ok := out["observations"]; !ok {
		t.Fatalf("trigram search response missing 'observations' key: %v", out)
	}

	// (2) The default (any/phrase) match_mode returns the seeded row — proves
	// the endpoint is wired and the seed is reachable.
	rr2, out2 := do(t, h, http.MethodGet, "/search?project=search-trigram-proj&query=obs-body", nil)
	if rr2.Code != http.StatusOK {
		t.Fatalf("phrase search expected 200, got %d (body %s)", rr2.Code, rr2.Body.String())
	}
	obs, _ := out2["observations"].([]any)
	if len(obs) == 0 {
		t.Fatalf("phrase search returned 0 hits for the seeded term; want the seeded row")
	}

	// (3) No project → 400 (the existing contract, unchanged by the new param).
	rr3, _ := do(t, h, http.MethodGet, "/search?query=obs-body&match_mode=trigram", nil)
	if rr3.Code != http.StatusBadRequest {
		t.Fatalf("search without project expected 400, got %d", rr3.Code)
	}
}

// seedHTTPStore opens the store for the given project and inserts a session +
// observation so the /search endpoint has a row to match.
func seedHTTPStore(t *testing.T, dataDir, project string) {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("seed store: %v", err)
	}
	defer st.Close()
	now := "2026-01-01T00:00:00Z"
	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, summary, status) VALUES ('s1', ?, '/tmp', ?, '## Goal\nseed', 'ended')`, project, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO observations (session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at) VALUES ('s1','decision','search-trig-title','obs-body', ?, 'project','h','0',?,?)`, project, now, now); err != nil {
		t.Fatalf("seed observation: %v", err)
	}
}
