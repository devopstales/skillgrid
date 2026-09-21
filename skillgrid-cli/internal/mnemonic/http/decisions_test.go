package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// Decision bridge (change 2026-09-19-embed-visual-companion, P1): the agent
// posts interview questions as type=decision observations (content is a
// structured JSON payload with a convention `state` field); the dashboard lists
// them via GET /mnemonic/decisions and the user answers via
// POST /mnemonic/decisions/{id}/answer, which flips state pending→answered
// through the same version-append path mem_update uses. These are the RED
// acceptance tests for both routes (TICKET-01); the implementation is
// TICKET-02 (decisions.go + memory.UpdateContent + route registration).

// decisionSeedContent is a well-formed pending decision payload.
const decisionSeedContent = `{"question":"Which layout?","options":[{"id":"a","label":"Single column"},{"id":"b","label":"Two column"}],"recommended":"a","state":"pending"}`

// newDecisionTestServer builds a Server over a seeded store for proj plus an
// open ProjectHandle the tests use to seed rows and assert store state.
func newDecisionTestServer(t *testing.T) (*Server, *service.ProjectHandle) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, summary, status) VALUES ('s-dec', ?, '.', '2026-01-01T00:00:00Z', 'seed', 'ended')`, proj); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	st.Close()
	svc := service.New(dataDir)
	s := NewServer(svc)
	h, cleanup, err := s.openHandleFor(proj)
	if err != nil {
		t.Fatalf("open handle: %v", err)
	}
	t.Cleanup(cleanup)
	return s, h
}

// seedDecision inserts a type=decision observation directly via the store and
// returns its id. created is the RFC3339 timestamp used for created_at
// (updated_at matches), so tests control ordering deterministically.
func seedDecision(t *testing.T, h *service.ProjectHandle, topicKey, contentJSON, visibility, created string) int64 {
	t.Helper()
	db := h.Store().DB
	res, err := db.Exec(`INSERT INTO observations
		(session_id, type, title, content, project, scope, topic_key, owner, visibility, status, revision_count, normalized_hash, created_at, updated_at)
		VALUES ('s-dec', 'decision', 'Decision: `+strings.ReplaceAll(topicKey, "interview/", "")+`', ?, ?, 'project', ?, 's-dec', ?, 'active', 0, 'hash-' || ?, ?, ?)`,
		contentJSON, proj, topicKey, visibility, topicKey, created, created)
	if err != nil {
		t.Fatalf("seed decision: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seed decision last insert id: %v", err)
	}
	return id
}

// getDecisions GETs /mnemonic/decisions and decodes the rows.
func getDecisions(t *testing.T, s *Server, query string) ([]decisionRow, int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/mnemonic/decisions?project="+proj+query, nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /mnemonic/decisions: got %d: %s", w.Code, w.Body.String())
	}
	var out []decisionRow
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode decisions: %v", err)
	}
	return out, w.Code
}

// postDecisionAnswer POSTs the answer body and returns the recorder.
func postDecisionAnswer(t *testing.T, s *Server, id int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mnemonic/decisions/"+itoaStr(id)+"/answer?project="+proj, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w
}

// decisionByTopic returns the listed row with the given topic key, or nil.
func decisionByTopic(rows []decisionRow, topic string) *decisionRow {
	for i := range rows {
		if rows[i].TopicKey == topic {
			return &rows[i]
		}
	}
	return nil
}

// [decision-list-roundtrip] A pending decision posted by the agent is listed
// with its content parsed.
func TestDecisionListRoundtrip(t *testing.T) {
	s, h := newDecisionTestServer(t)
	seedDecision(t, h, "interview/demo/layout-1", decisionSeedContent, "team", "2026-01-01T00:00:00Z")

	rows, _ := getDecisions(t, s, "&state=pending")
	if len(rows) != 1 {
		t.Fatalf("want 1 pending decision, got %d: %+v", len(rows), rows)
	}
	if rows[0].Content.Question != "Which layout?" {
		t.Errorf("content not parsed: %+v", rows[0].Content)
	}
	if rows[0].Content.Recommended != "a" || rows[0].Content.State != "pending" {
		t.Errorf("fields missing: %+v", rows[0].Content)
	}
	if len(rows[0].Content.Options) != 2 || rows[0].Content.Options[1].ID != "b" {
		t.Errorf("options not parsed: %+v", rows[0].Content.Options)
	}
}

// [decision-skips-malformed] A malformed content JSON on a decision row is
// flagged (parseError) or skipped — never a 500 — and good rows still surface.
func TestDecisionSkipsMalformed(t *testing.T) {
	s, h := newDecisionTestServer(t)
	seedDecision(t, h, "interview/demo/bad", `{"question":"ok","options":[`, "team", "2026-01-01T00:00:00Z")
	seedDecision(t, h, "interview/demo/good", `{"question":"good","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:01Z")

	rows, code := getDecisions(t, s, "")
	if code != http.StatusOK {
		t.Fatalf("malformed row must not 500, got %d", code)
	}
	if decisionByTopic(rows, "interview/demo/good") == nil {
		t.Errorf("good decision missing from list: %+v", rows)
	}
}

// [decision-visible-to-reader] A team decision is visible to the dashboard
// reader; a private one is not.
func TestDecisionVisibilityGate(t *testing.T) {
	s, h := newDecisionTestServer(t)
	seedDecision(t, h, "interview/demo/team", `{"question":"t","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
	seedDecision(t, h, "interview/demo/priv", `{"question":"p","options":[{"id":"a","label":"A"}],"state":"pending"}`, "private", "2026-01-01T00:00:01Z")

	rows, _ := getDecisions(t, s, "")
	if decisionByTopic(rows, "interview/demo/team") == nil {
		t.Errorf("team decision missing from list: %+v", rows)
	}
	if d := decisionByTopic(rows, "interview/demo/priv"); d != nil {
		t.Errorf("private decision leaked to reader: %+v", d)
	}
}

// [decision-ordering] Rows are returned ascending by created_at.
func TestDecisionOrderingAscending(t *testing.T) {
	s, h := newDecisionTestServer(t)
	seedDecision(t, h, "interview/demo/q1", `{"question":"one","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
	seedDecision(t, h, "interview/demo/q2", `{"question":"two","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:01Z")
	seedDecision(t, h, "interview/demo/q3", `{"question":"three","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:02Z")

	rows, _ := getDecisions(t, s, "")
	if len(rows) != 3 {
		t.Fatalf("want 3 decisions, got %d", len(rows))
	}
	for i, want := range []string{"interview/demo/q1", "interview/demo/q2", "interview/demo/q3"} {
		if rows[i].TopicKey != want {
			t.Errorf("order[%d] = %q, want %q", i, rows[i].TopicKey, want)
		}
	}
}

// [decision-answer-records] The answer route flips state pending→answered and
// records the chosen option + note.
func TestDecisionAnswerRecords(t *testing.T) {
	s, h := newDecisionTestServer(t)
	id := seedDecision(t, h, "interview/demo/layout-1", decisionSeedContent, "team", "2026-01-01T00:00:00Z")

	w := postDecisionAnswer(t, s, id, `{"optionId":"b","note":"two column reads better"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("answer: got %d: %s", w.Code, w.Body.String())
	}

	rows, _ := getDecisions(t, s, "")
	var got *decisionRow
	for i := range rows {
		if rows[i].ID == id {
			got = &rows[i]
		}
	}
	if got == nil {
		t.Fatalf("decision not found after answer")
	}
	if got.Content.State != "answered" {
		t.Errorf("state not flipped: %q", got.Content.State)
	}
	if got.Content.AnsweredOption != "b" {
		t.Errorf("answeredOption not recorded: %+v", got.Content)
	}
	if got.Content.AnswerNote != "two column reads better" {
		t.Errorf("answerNote not recorded: %+v", got.Content)
	}
	if !strings.HasPrefix(got.Content.UpdatedBy, "user:") {
		t.Errorf("updatedBy not user:<name>: %q", got.Content.UpdatedBy)
	}
}

// [decision-answer-appends-version] R6: answering twice yields two
// observation_versions rows; the latest content carries the second answer
// (append-only history, never overwrites the history).
func TestDecisionAnswerAppendsVersion(t *testing.T) {
	s, h := newDecisionTestServer(t)
	id := seedDecision(t, h, "interview/demo/v", `{"question":"q","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
	w := postDecisionAnswer(t, s, id, `{"optionId":"a","note":"first"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("first answer: got %d: %s", w.Code, w.Body.String())
	}
	w = postDecisionAnswer(t, s, id, `{"optionId":"a","note":"second"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("second answer: got %d: %s", w.Code, w.Body.String())
	}
	var n int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM observation_versions WHERE observation_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 version rows after two answers, got %d", n)
	}
	rows, _ := getDecisions(t, s, "")
	var got *decisionRow
	for i := range rows {
		if rows[i].ID == id {
			got = &rows[i]
		}
	}
	if got == nil {
		t.Fatalf("decision not found after re-answer")
	}
	if got.Content.State != "answered" || got.Content.AnsweredOption != "a" || got.Content.AnswerNote != "second" {
		t.Errorf("latest content must carry the second answer: %+v", got.Content)
	}
}

// [decision-answer-reanswer-updates] R6: a re-answer of an answered decision
// is a 200 that records the NEW answer (option + note + updatedBy) and appends
// a version — the history is append-only, the latest content is the latest
// answer.
func TestDecisionAnswerIdempotent(t *testing.T) {
	s, h := newDecisionTestServer(t)
	id := seedDecision(t, h, "interview/demo/idem", `{"question":"q","options":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
	postDecisionAnswer(t, s, id, `{"optionId":"a","note":"first"}`)
	w := postDecisionAnswer(t, s, id, `{"optionId":"b","note":"second"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("re-answer should be 200, got %d: %s", w.Code, w.Body.String())
	}

	rows, _ := getDecisions(t, s, "")
	var got *decisionRow
	for i := range rows {
		if rows[i].ID == id {
			got = &rows[i]
		}
	}
	if got == nil {
		t.Fatalf("decision not found after re-answer")
	}
	if got.Content.State != "answered" || got.Content.AnsweredOption != "b" || got.Content.AnswerNote != "second" {
		t.Errorf("re-answer must record the new answer: %+v", got.Content)
	}
	if got.Content.UpdatedBy != "user:unknown" {
		t.Errorf("updatedBy must reflect the latest answerer: %q", got.Content.UpdatedBy)
	}
	var n int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM observation_versions WHERE observation_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if n != 2 {
		t.Errorf("re-answer must append a version, got %d rows, want 2", n)
	}
}

// [decision-answer-404] An unknown decision id is 404.
func TestDecisionAnswer404(t *testing.T) {
	s, _ := newDecisionTestServer(t)
	w := postDecisionAnswer(t, s, 999999, `{"optionId":"a"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("bad id should 404, got %d (%s)", w.Code, w.Body.String())
	}
}

// [decision-answer-409] A decision row whose content lacks a valid state
// (or is malformed) is 409 — the gate is the convention field, not status.
func TestDecisionAnswer409MissingState(t *testing.T) {
	s, h := newDecisionTestServer(t)
	noState := seedDecision(t, h, "interview/demo/nostate", `{"question":"q","options":[{"id":"a","label":"A"}]}`, "team", "2026-01-01T00:00:00Z")
	w := postDecisionAnswer(t, s, noState, `{"optionId":"a"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("missing state should 409, got %d (%s)", w.Code, w.Body.String())
	}

	badState := seedDecision(t, h, "interview/demo/badstate", `{"question":"q","options":[{"id":"a","label":"A"}],"state":"bogus"}`, "team", "2026-01-01T00:00:01Z")
	w = postDecisionAnswer(t, s, badState, `{"optionId":"a"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("invalid state should 409, got %d (%s)", w.Code, w.Body.String())
	}

	malformed := seedDecision(t, h, "interview/demo/malformed", `{"question":"q","options":[`, "team", "2026-01-01T00:00:02Z")
	w = postDecisionAnswer(t, s, malformed, `{"optionId":"a"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("malformed content should 409, got %d (%s)", w.Code, w.Body.String())
	}
}

// [decision-answer-requires-auth] With SKILLGRID_HTTP_TOKEN set, a no-auth
// answer POST is 401; with the token it succeeds.
func TestDecisionAnswerRequiresAuth(t *testing.T) {
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret-token")
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, summary, status) VALUES ('s-dec', ?, '.', '2026-01-01T00:00:00Z', 'seed', 'ended')`, proj); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	st.Close()
	svc := service.New(dataDir)
	s := NewServer(svc)
	h, cleanup, err := s.openHandleFor(proj)
	if err != nil {
		t.Fatalf("open handle: %v", err)
	}
	t.Cleanup(cleanup)
	id := seedDecision(t, h, "interview/demo/auth", `{"question":"q","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")

	w := postDecisionAnswer(t, s, id, `{"optionId":"a"}`)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no-auth answer with token set: got %d want 401", w.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/mnemonic/decisions/"+itoaStr(id)+"/answer?project="+proj, strings.NewReader(`{"optionId":"a"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("authed answer should 200, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// [decision-upsert-bumps-revision] Saving the same topic_key twice (the agent
// in-place question update, mem_save) upserts the same row: one row,
// revision_count incremented, a version appended.
func TestDecisionUpsertBumpsRevision(t *testing.T) {
	_, h := newDecisionTestServer(t)
	topic := "interview/demo/upsert"
	in := memory.SaveInput{
		SessionID: "s-dec", Type: "decision",
		Title:   "Decision: upsert",
		Content: `{"question":"q1","options":[{"id":"a","label":"A"}],"recommended":"a","state":"pending"}`,
		Scope:   "project", TopicKey: topic, Owner: "s-dec",
	}
	id1, err := h.Memory().Save(context.Background(), in)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	// session_id is an FK to sessions: the upsert keeps the same valid session.
	in2 := in
	in2.Content = `{"question":"q1 v2","options":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"recommended":"b","state":"pending"}`
	id2, err := h.Memory().Save(context.Background(), in2)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("same topic_key must upsert the same row: %d != %d", id1, id2)
	}
	var rowCount, revisionCount int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*), COALESCE(MAX(revision_count),0) FROM observations WHERE id = ?`, id1).Scan(&rowCount, &revisionCount); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("want 1 row for the topic_key, got %d", rowCount)
	}
	if revisionCount < 1 {
		t.Errorf("revision_count not incremented by upsert, got %d", revisionCount)
	}
	var versions int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM observation_versions WHERE observation_id = ?`, id1).Scan(&versions); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if versions < 1 {
		t.Errorf("upsert must append a version row, got %d", versions)
	}
}

// [decision-400-validation] A missing optionId is 400 (before the 404/409
// decision-specific checks).
func TestDecisionAnswer400MissingOption(t *testing.T) {
	s, h := newDecisionTestServer(t)
	id := seedDecision(t, h, "interview/demo/missing", decisionSeedContent, "team", "2026-01-01T00:00:00Z")
	w := postDecisionAnswer(t, s, id, `{"note":"no option"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing optionId should 400, got %d (%s)", w.Code, w.Body.String())
	}
}
