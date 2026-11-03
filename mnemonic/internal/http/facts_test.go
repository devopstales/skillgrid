package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// httptestRecorder wraps a response recorder plus a decoded JSON body so the
// facts route tests can assert on the body fields directly.
type httptestRecorder struct {
	*httptest.ResponseRecorder
	out map[string]any
}

// doPost POSTs a JSON body to target and returns the recorder + decoded body.
func doPost(t *testing.T, h http.Handler, target string, body map[string]any) *httptestRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(http.MethodPost, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	out := map[string]any{}
	_ = json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&out)
	return &httptestRecorder{ResponseRecorder: rr, out: out}
}

// closeStore releases the pooled store handle for dataDir+proj so the test
// can exit without a dangling connection (mirrors the toolcalls test cleanup).
func closeStore(t *testing.T, dataDir string) {
	t.Helper()
	if st, err := store.Open(dataDir, proj); err == nil {
		st.Close()
	}
}

// seedFacts seeds n facts with a controlled importance_score for the decay /
// decay-all routes. Returns the inserted ids in order.
func seedFacts(t *testing.T, dataDir string, n int, score float64) []int64 {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("seed facts: %v", err)
	}
	defer st.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	// The fact store trails a session_events row per operation; ensure a
	// session exists so the foreign key holds for the store's Add/decay paths.
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES ('project', ?, '/tmp', ?, 'active')`, proj, now); err != nil {
		t.Fatalf("seed project session: %v", err)
	}
	var ids []int64
	for i := 0; i < n; i++ {
		res, err := st.DB.Exec(`
			INSERT INTO facts (content, importance_score, created_at, updated_at)
			VALUES (?, ?, ?, ?)`, "use bcrypt not md5 "+strconv.Itoa(i), score, now, now)
		if err != nil {
			t.Fatalf("seed fact %d: %v", i, err)
		}
		id, _ := res.LastInsertId()
		ids = append(ids, id)
	}
	return ids
}

// newFactsServer builds an isolated server + dataDir with the project store
// seeded (session + observation) so the facts routes have a live handle. It
// also seeds the 'project' session that the facts handlers default their
// session_events trail to (the FK on session_events.session_id requires it).
func newFactsServer(t *testing.T) (*Server, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("seed project session store: %v", err)
	}
	now := "2026-01-01T00:00:00Z"
	if _, err := st.DB.Exec(`INSERT OR IGNORE INTO sessions (id, project, directory, started_at, summary, status) VALUES ('project', ?, '/tmp', ?, '', 'active')`, proj, now); err != nil {
		t.Fatalf("seed project session: %v", err)
	}
	st.Close()
	s := NewServer(service.New(dataDir))
	return s, dataDir
}

func postFact(t *testing.T, s *Server, target string, body map[string]any) *httptestRecorder {
	t.Helper()
	return doPost(t, s.Handler(), target, body)
}

func TestFactsAddAndSearch(t *testing.T) {
	s, dataDir := newFactsServer(t)
	defer closeStore(t, dataDir)

	// POST /facts → 200 {id}
	res := postFact(t, s, "/facts?project="+proj, map[string]any{"content": "use bcrypt not md5"})
	if res.Code != http.StatusOK {
		t.Fatalf("add: status %d: %s", res.Code, res.Body.String())
	}
	id, _ := res.out["id"].(float64)
	if id <= 0 {
		t.Fatalf("add: expected positive id, got %v", res.out["id"])
	}

	// POST /facts/search → 200 {facts:[...]} with a match.
	res = postFact(t, s, "/facts/search?project="+proj, map[string]any{"query": "bcrypt"})
	if res.Code != http.StatusOK {
		t.Fatalf("search: status %d: %s", res.Code, res.Body.String())
	}
	facts, ok := res.out["facts"].([]any)
	if !ok || len(facts) < 1 {
		t.Fatalf("search: expected at least 1 fact, got %v", res.out["facts"])
	}

	// Empty query → 200 with an empty facts array.
	res = postFact(t, s, "/facts/search?project="+proj, map[string]any{"query": ""})
	if res.Code != http.StatusOK {
		t.Fatalf("search empty: status %d: %s", res.Code, res.Body.String())
	}
	if facts, ok = res.out["facts"].([]any); !ok || len(facts) != 0 {
		t.Fatalf("search empty: expected empty facts, got %v", res.out["facts"])
	}
}

func TestFactsAddEmptyContent400(t *testing.T) {
	s, dataDir := newFactsServer(t)
	defer closeStore(t, dataDir)

	res := postFact(t, s, "/facts?project="+proj, map[string]any{"content": ""})
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", res.Code, res.Body.String())
	}
}

func TestFactsForget(t *testing.T) {
	s, dataDir := newFactsServer(t)
	defer closeStore(t, dataDir)

	ids := seedFacts(t, dataDir, 1, 1.0)
	factID := strconv.FormatInt(ids[0], 10)

	// POST /facts/{id}/forget → 204.
	res := doPost(t, s.Handler(), "/facts/"+factID+"/forget?project="+proj, nil)
	if res.Code != http.StatusNoContent {
		t.Fatalf("forget: expected 204, got %d: %s", res.Code, res.Body.String())
	}

	// The forgotten fact is excluded from default search.
	res = postFact(t, s, "/facts/search?project="+proj, map[string]any{"query": "bcrypt"})
	if res.Code != http.StatusOK {
		t.Fatalf("search after forget: status %d: %s", res.Code, res.Body.String())
	}
	if facts, ok := res.out["facts"].([]any); !ok || len(facts) != 0 {
		t.Fatalf("search after forget: expected empty facts, got %v", res.out["facts"])
	}
}

func TestFactsForgetUnknown404(t *testing.T) {
	s, dataDir := newFactsServer(t)
	defer closeStore(t, dataDir)

	res := doPost(t, s.Handler(), "/facts/999/forget?project="+proj, nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestFactsDecay(t *testing.T) {
	s, dataDir := newFactsServer(t)
	defer closeStore(t, dataDir)

	ids := seedFacts(t, dataDir, 1, 1.0)
	factID := strconv.FormatInt(ids[0], 10)

	res := doPost(t, s.Handler(), "/facts/"+factID+"/decay?project="+proj, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("decay: status %d: %s", res.Code, res.Body.String())
	}
	score, _ := res.out["score"].(float64)
	// A fresh fact with the default zero decay rate is a no-op that still
	// returns the (unchanged) score.
	if score <= 0 {
		t.Fatalf("decay: expected score > 0, got %v", res.out["score"])
	}
}

func TestFactsDecayAll(t *testing.T) {
	s, dataDir := newFactsServer(t)
	defer closeStore(t, dataDir)

	// 3 live facts: two at 1.0 (survive), one at 0.4 (below 0.5 threshold →
	// purged after the zero-rate decay no-op).
	ids := seedFacts(t, dataDir, 2, 1.0)
	_ = ids
	low := seedFacts(t, dataDir, 1, 0.4)
	_ = low

	res := postFact(t, s, "/facts/decay-all?project="+proj, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("decay-all: status %d: %s", res.Code, res.Body.String())
	}
	decayed, _ := res.out["decayed"].(float64)
	purged, _ := res.out["purged"].(float64)
	if decayed != 3 {
		t.Fatalf("decay-all: decayed = %v, want 3", res.out["decayed"])
	}
	if purged != 1 {
		t.Fatalf("decay-all: purged = %v, want 1", res.out["purged"])
	}
}
