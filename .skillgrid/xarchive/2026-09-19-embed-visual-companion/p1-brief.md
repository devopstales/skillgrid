### TICKET-01 — RED: decision list + answer acceptance tests
- **Tracker ID:** TASK-007

- **Scope:** Write the failing acceptance tests (no implementation yet) for the two backend routes: a `GET /mnemonic/decisions` list test (filters `type=decision` + `content.state=pending`, orders ascending by `created_at`) and a `POST /mnemonic/decisions/{id}/answer` test (state flip `pending`→`answered`, appends `observation_versions`, bumps `revision_count`, `actor=user:<name>`, idempotent no-op on re-answer, `404` on bad id, `409` when `content.state` is missing/invalid).
- **Acceptance:** `go test -count=1 ./internal/mnemonic/http/... -run 'TestDecision'` fails (routes not yet registered / `UpdateContent` undefined) with the exact expected assertions; test file compiles against the proposed `memory.UpdateContent` signature (add a forward declaration or interface stub if needed so it compiles RED, not a build error).
- **SATISFIES:** Scenario "agent posts a pending decision, dashboard lists it" + Scenario "user answers, agent reads it back, decision is versioned" (RED state).
- **Files:** `internal/mnemonic/http/decisions_test.go` (new).
- **Size:** ~150 (S)
- **Blocks:** TICKET-02
- **Blocked by:** none
- **Fails-when:** `go test` exits non-zero on the decision tests (expected RED) but the package still compiles.

### TICKET-02 — `GET /mnemonic/decisions` + `POST /mnemonic/decisions/{id}/answer`
- **Tracker ID:** TASK-008

- **Scope:** Implement the read route (filter `type=decision` + `content.state=pending` from `GET /mnemonic/memories` output, ascending by `created_at`, return `{decisions, total}`) and the write route (parse `{answer, rationale?, answerer?}`, require `content.state` present + valid, set `state="answered"` + `answer` + `rationale`, `actor=user:<name>` via `UpdateContent`, `404` bad id, `409` missing/invalid `content.state`, idempotent no-op when already answered). Add `UpdateContent` to `memory` with version append. Register both routes in `server.go` before `registerUIRoutes()`.
- **Acceptance:** TICKET-01 tests now pass; a live `skillgrid serve` + curl round-trip: seed a `type=decision`/`state=pending` observation via `mem_save`, `GET /mnemonic/decisions` lists it, `POST .../answer` flips it, re-`GET` shows `state=answered` with the answer, `observation_versions` gained a row, re-`POST` is a no-op, bad id → `404`, memory without `content.state` → `409`.
- **SATISFIES:** Scenario "agent posts a pending decision, dashboard lists it" + Scenario "user answers, agent reads it back, decision is versioned" (GREEN).
- **Files:** `internal/mnemonic/http/decisions.go` (new), `internal/mnemonic/http/decisions_test.go` (fill in), `internal/mnemonic/memory/governance.go` (add `UpdateContent` + tests), `internal/mnemonic/http/server.go` (register routes).
- **Size:** ~450 (M)
- **Blocks:** TICKET-04
- **Blocked by:** TICKET-01
- **Fails-when:** `go test -count=1 ./internal/mnemonic/http/...` exits non-zero.
### Task 1: Decision content convention + `GET /mnemonic/decisions` read route (DOOR CHECK)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/decisions.go`
- Create: `skillgrid-cli/internal/mnemonic/http/decisions_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (register the two routes)

**Interfaces:**
- Consumes: `projectFromRequest(r)`, `s.openHandleFor(projectID)`, `writeJSON`, `writeError` (existing). `h.Store().DB` for the query.
- Produces:
  - `type decisionOption struct { ID string; Label string; Rationale string }` (json `id`,`label`,`rationale`)
  - `type decisionContent struct { Question string; Options []decisionOption; Recommended string; State string; AnsweredOption string; AnswerNote string; UpdatedBy string }` (json tags)
  - `type decisionRow struct { ID int64; TopicKey string; Title string; CreatedAt string; UpdatedAt string; Content decisionContent; ParseError bool }`
  - `func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request)` — `GET /mnemonic/decisions?project=…&state=pending|answered|all&topic=…`
  - `func (s *Server) registerDecisionRoutes()`
  - `func parseDecisionContent(raw string) (decisionContent, bool)`

**SATISFIES:** scenarios `decision-list-roundtrip` + `decision-skips-malformed` (see `acceptance.feature`).

- [ ] **Step 1: Write the failing test**

```go
// decisions_test.go
package http

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// seedDecision inserts a type=decision observation directly via the store.
func seedDecision(t *testing.T, h interface{ Store() interface{ DB *sql.DB } }, topicKey, contentJSON, visibility string) int64 {
	t.Helper()
	db := h.Store().DB
	id, err := db.Exec(`INSERT INTO observations
		(session_id, type, title, content, project, scope, topic_key, visibility, status, created_at, updated_at)
		VALUES ('s-dec','decision','Q','?','testproj','project','?','?','active',?,?,?)`,
		contentJSON, topicKey, visibility,
		time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339)).RowsAffected()
	// (adapt the Exec to the package's actual store handle shape; see mnemonic_files.go)
	return id
}

func TestDecisionListRoundtrip(t *testing.T) {
	s, h := newDecisionTestServer(t) // builds a Server + open handle for project 'testproj'
	content := `{"question":"Which layout?","options":[{"id":"a","label":"Single column"},{"id":"b","label":"Two column"}],"recommended":"a","state":"pending"}`
	seedDecision(t, h, "interview/demo/layout-1", content, "team")

	req := httptest.NewRequest("GET", "/mnemonic/decisions?project=testproj&state=pending", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("GET decisions: got %d: %s", w.Code, w.Body.String()) }
	var out []decisionRow
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil { t.Fatalf("decode: %v", err) }
	if len(out) != 1 { t.Fatalf("want 1 decision, got %d", len(out)) }
	if out[0].Content.Question != "Which layout?" { t.Errorf("content not parsed: %+v", out[0].Content) }
	if out[0].Content.Recommended != "a" || out[0].Content.State != "pending" { t.Errorf("fields missing: %+v", out[0].Content) }
}

func TestDecisionSkipsMalformed(t *testing.T) {
	s, h := newDecisionTestServer(t)
	seedDecision(t, h, "interview/demo/bad", `{"question":"ok","options":[`, "team") // truncated JSON
	seedDecision(t, h, "interview/demo/good", `{"question":"good","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team")

	req := httptest.NewRequest("GET", "/mnemonic/decisions?project=testproj", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("malformed row must not 500, got %d", w.Code) }
	var out []decisionRow
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	// The good row is present; the bad row is either skipped or flagged, but the handler survived.
	found := false
	for _, d := range out { if d.Content.Question == "good" { found = true } }
	if !found { t.Errorf("good decision missing from list: %+v", out) }
}

func TestDecisionVisibilityGate(t *testing.T) {
	s, h := newDecisionTestServer(t)
	seedDecision(t, h, "interview/demo/team", `{"question":"t","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team")
	seedDecision(t, h, "interview/demo/priv", `{"question":"p","options":[{"id":"a","label":"A"}],"state":"pending"}`, "private")
	req := httptest.NewRequest("GET", "/mnemonic/decisions?project=testproj", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	var out []decisionRow
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	for _, d := range out {
		if d.TopicKey == "interview/demo/priv" { t.Errorf("private decision leaked to reader") }
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestDecision' -v`
Expected: FAIL — `handleDecisions`, `decisionRow`, etc. undefined (compile error).

- [ ] **Step 3: Write minimal implementation**

```go
// decisions.go
package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

type decisionOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Rationale string `json:"rationale,omitempty"`
}

type decisionContent struct {
	Question        string           `json:"question"`
	Options         []decisionOption `json:"options"`
	Recommended     string           `json:"recommended,omitempty"`
	State           string           `json:"state"` // pending | answered | superseded
	AnsweredOption  string           `json:"answeredOption,omitempty"`
	AnswerNote      string           `json:"answerNote,omitempty"`
	UpdatedBy       string           `json:"updatedBy,omitempty"`
}

type decisionRow struct {
	ID         int64           `json:"id"`
	TopicKey   string          `json:"topicKey"`
	Title      string          `json:"title"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Visibility string          `json:"visibility"`
	Content    decisionContent `json:"content"`
	ParseError bool            `json:"parseError,omitempty"`
}

func parseDecisionContent(raw string) (decisionContent, bool) {
	var c decisionContent
	if err := json.Unmarshal([]byte(raw), &c); err != nil || c.Question == "" {
		return decisionContent{}, false
	}
	return c, true
}

func (s *Server) registerDecisionRoutes() {
	s.mux.HandleFunc("GET /mnemonic/decisions", s.handleDecisions)
	s.mux.HandleFunc("POST /mnemonic/decisions/{id}/answer", s.requireWriteAuth(s.handleDecisionAnswer))
}

func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil { writeError(w, http.StatusBadRequest, err.Error()); return }
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil { writeError(w, http.StatusNotFound, err.Error()); return }
	defer cleanup()

	wantState := r.URL.Query().Get("state")    // pending | answered | superseded | ""(all)
	wantTopic := r.URL.Query().Get("topic")     // optional topic_key prefix filter

	rows, err := h.Store().DB.Query(r.Context(), `
		SELECT id, topic_key, title, content, visibility, created_at, updated_at
		FROM observations
		WHERE project = ? AND type = 'decision' AND deleted_at IS NULL
		  AND (visibility IN ('team','restricted','agent') OR visibility IS NULL)
		ORDER BY updated_at DESC, id DESC
		LIMIT 200`, projectID)
	if err != nil { writeError(w, http.StatusInternalServerError, err.Error()); return }
	defer rows.Close()

	out := []decisionRow{}
	for rows.Next() {
		var d decisionRow
		var content string
		if err := rows.Scan(&d.ID, &d.TopicKey, &d.Title, &content, &d.Visibility, &d.CreatedAt, &d.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if c, ok := parseDecisionContent(content); ok {
			d.Content = c
		} else {
			d.ParseError = true
		}
		// filters
		if wantTopic != "" && !strings.HasPrefix(d.TopicKey, wantTopic) { continue }
		if wantState != "" && d.Content.State != wantState && !d.ParseError { continue }
		out = append(out, d)
	}
	if err := rows.Err(); err != nil { writeError(w, http.StatusInternalServerError, err.Error()); return }
	writeJSON(w, http.StatusOK, out)
}
```

Note: the visibility filter matches how the memories list already exposes rows to the reader; confirm the exact predicate against `handleMnemonicMemories` in `mnemonic_files.go` and reuse it verbatim (the `private` exclusion above is the expected behavior — the dashboard reader is a team-scoped identity). Wire `s.registerDecisionRoutes()` into `registerRoutes()` next to `s.registerGraphRoutes()`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestDecision' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/decisions.go skillgrid-cli/internal/mnemonic/http/decisions_test.go skillgrid-cli/internal/mnemonic/http/server.go
git commit -m "feat(decisions): read route + content convention for the Mnemonic decision bridge

[skillgrid-context]
Change: 2026-09-19-embed-visual-companion
Decisions: companion = view over type=decision observations; polling transport; state is a content field (status is reserved)
"
```

---

