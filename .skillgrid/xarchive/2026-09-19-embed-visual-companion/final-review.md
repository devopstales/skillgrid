# Review package: e490ed1..8148b5d

## Commits
8148b5d docs(http): openapi + user guide for decision companion (TICKET-07)
49b3a25 test(http): RED round-trip guard for decision MCP read-back (TICKET-07)
1086a2b feat(http): prototype serve route (.skillgrid/prototype) + sandboxed iframe in decision card (TICKET-06)
4c884d2 test(http): RED acceptance tests for prototype serve + decision card iframe (TICKET-05)
76ec335 feat(ui): decisions view with answer round-trip (TICKET-04)
b7e39a4 test(ui): RED acceptance tests for decisions view (TICKET-03)
7cc50f4 fix(http): re-answer appends a version instead of no-op (R6)
8bc12bf feat(http): decision bridge read+answer routes (TICKET-02)
c3b07ca test(http): RED acceptance tests for decision bridge routes (TICKET-01)

## Files changed
 docs/user-guide/10-decision-companion.md           | 180 ++++++
 skillgrid-cli/internal/mnemonic/http/decisions.go  | 259 ++++++++
 .../internal/mnemonic/http/decisions_test.go       | 566 +++++++++++++++++
 .../internal/mnemonic/http/openapi_phase8_test.go  |  90 +++
 skillgrid-cli/internal/mnemonic/http/prototype.go  |  64 ++
 .../internal/mnemonic/http/prototype_test.go       |  92 +++
 skillgrid-cli/internal/mnemonic/http/server.go     |  10 +
 .../internal/mnemonic/http/ui/openapi.yaml         | 202 +++++-
 .../internal/mnemonic/memory/governance.go         |  45 ++
 skillgrid-ui/package-lock.json                     | 681 +++++++++++++++++++++
 skillgrid-ui/package.json                          |   3 +
 skillgrid-ui/src/app.tsx                           |  14 +
 skillgrid-ui/src/components/layout/AppLayout.tsx   |   2 +
 .../src/features/decisions/DecisionCard.test.tsx   |  78 +++
 .../src/features/decisions/DecisionCard.tsx        | 171 ++++++
 .../src/features/decisions/DecisionsPage.test.tsx  |  84 +++
 .../src/features/decisions/DecisionsPage.tsx       |  86 +++
 skillgrid-ui/src/features/decisions/api.ts         |  78 +++
 skillgrid-ui/src/vite-env.d.ts                     |   1 +
 skillgrid-ui/vite.config.ts                        |  10 +
 20 files changed, 2714 insertions(+), 2 deletions(-)

## Diff
diff --git a/docs/user-guide/10-decision-companion.md b/docs/user-guide/10-decision-companion.md
new file mode 100644
index 0000000..8ac47a8
--- /dev/null
+++ b/docs/user-guide/10-decision-companion.md
@@ -0,0 +1,180 @@
+# Decision companion — the Mnemonic decision bridge
+
+The decision companion is a **view over `type: decision` Mnemonic observations**.
+The agent posts interview questions; the dashboard renders them; the user
+answers; the agent reads the answer back. No WebSocket, no second process, no
+new store — **polling is the transport**. Everything is durable in the Mnemonic
+SQLite store (see [Serve — the web dashboard](./09-serve-dashboard.md) for the
+UI that renders it).
+
+This page is the **agent-side convention**: the content schema, the topic-key
+and visibility rules, the poll loop, the answer semantics, and the
+throwaway-prototype serving convention.
+
+---
+
+## The decision content schema (JSON in `observations.content`)
+
+A decision is an ordinary observation whose `content` column is a structured
+JSON payload. The schema:
+
+```json
+{
+  "question": "Which layout for the homepage?",
+  "options": [
+    { "id": "a", "label": "Single column", "rationale": "cleanest reading" },
+    { "id": "b", "label": "Two column" }
+  ],
+  "recommended": "a",
+  "state": "pending"
+}
+```
+
+Fields:
+
+| Field | Required | Meaning |
+|-------|----------|---------|
+| `question` | yes | The question, in domain language. A missing/blank question makes the row malformed (flagged `parseError`, never a 500). |
+| `options[]` | yes (≥1) | Each: `id` (stable, `a`/`b`/…), `label`, optional `rationale`. |
+| `recommended` | no | The id of the option the agent recommends (highlighted in the UI). |
+| `state` | yes | `pending` \| `answered` \| `superseded`. **This is a content field, not the governance `status`** — `status` is reserved for `active` \| `superseded` \| `archived`. |
+| `answeredOption` | set by answer | The id of the option the user chose. |
+| `answerNote` | set by answer | The user's answer note. |
+| `updatedBy` | set by answer | `agent`, or `user:<name>` after an answer. |
+| `visual` | no | A prototype id `<topic>/<variant>.html` served from `.skillgrid/prototype/` (see [Prototypes](#prototypes-the-visual-function)). When present, the decision card renders it in a sandboxed iframe. |
+
+After a user answers, the content carries the answer inline:
+
+```json
+{
+  "question": "Which layout for the homepage?",
+  "options": [ { "id": "a", "label": "Single column" }, { "id": "b", "label": "Two column" } ],
+  "recommended": "a",
+  "state": "answered",
+  "answeredOption": "b",
+  "answerNote": "two column reads better",
+  "updatedBy": "user:paladm"
+}
+```
+
+---
+
+## The topic-key and visibility convention
+
+- **`type: decision`** — the observation type. The dashboard and the list route
+  select on this.
+- **`topic_key: interview/<slug>/<question-id>`** — a stable, hierarchical key.
+  `<slug>` is the interview (e.g. `homepage-layout`), `<question-id>` is one
+  question within it (e.g. `layout-1`). Reusing the same `topic_key` **upserts**
+  the row (an in-place question update; it bumps the revision and appends a
+  version).
+- **`visibility: team`** — the companion's default. A fresh `mem_save` is
+  `private` (governance default); the frozen `mem_save` tool has no visibility
+  argument, so the agent widens it with `mem_share` (target `team`) after
+  saving. The dashboard reader is a team-scoped identity: **private decisions
+  are never listed**.
+
+---
+
+## The loop (polling, not realtime)
+
+1. **Agent → user.** `mem_save` the decision (`type: decision`,
+   `topic_key: interview/<slug>/<question-id>`, `content: {… state: "pending"}`),
+   then `mem_share` it to `team`. Re-saving the same `topic_key` updates the
+   question in place.
+2. **Dashboard.** Polls `GET /mnemonic/decisions?state=pending` and renders the
+   cards (question, options, the recommended highlight, and the visual when
+   present).
+3. **User → agent.** Click an option (optionally with a note) →
+   `POST /mnemonic/decisions/{id}/answer` with body `{"optionId","note"}`. This
+   is **write-gated** (Bearer `SKILLGRID_HTTP_TOKEN`); it sets `state: "answered"`,
+   records the choice, and **appends a version** (durable, auditable).
+4. **Agent.** Polls `mem_search` (its own frozen MCP surface) on a query
+   consistent with the decision — a word of the question or the content — and
+   proceeds when the returned content has `state == "answered"`.
+
+There is no push: both sides poll. The round trip is guard-tested
+(`TestDecisionRoundTrip` in `skillgrid-cli/internal/mnemonic/http/`): the
+dashboard list and the agent's `mem_search` agree on the seeded decision, and
+a *fresh* `mem_search` (a new service, simulating the agent polling later in a
+resumed session) sees the answered state and the chosen option.
+
+---
+
+## Answer semantics
+
+- **Pending → answered.** The first answer flips the convention `state` and
+  records `answeredOption` / `answerNote` / `updatedBy` (`user:<name>`).
+- **Re-answer appends a version.** Answering an already-answered decision is a
+  `200`: the state stays `answered`, the *new* answer is recorded, and another
+  `observation_versions` row is appended. The history is **append-only** — the
+  **latest content carries the latest answer**, and prior answers are recoverable
+  from the version history (governance). This is the same version-append path
+  `mem_update` uses.
+- **Superseded is agent-owned.** The `pending → superseded` transition is made by
+  the agent (e.g. the question was answered elsewhere). The answer gate does not
+  apply to a superseded decision.
+
+Error contract for `POST /mnemonic/decisions/{id}/answer`:
+
+| Status | When |
+|--------|------|
+| `200` | Answered (or re-answered). Body: `{"ok": true, "state": "answered"}`. |
+| `400` | Missing/invalid `project` or `id` (non-integer), malformed body, or missing `optionId`. |
+| `401` | Write-gate: no Bearer token while `SKILLGRID_HTTP_TOKEN` is set. |
+| `404` | Unknown id (no such decision row in the project). |
+| `409` | Content has no valid `state` (missing, malformed, or unrecognized), or the decision is `superseded`. |
+| `422` | `optionId` is not in the decision's `options`. |
+
+`GET /mnemonic/decisions` takes `?state=pending|answered|superseded|all` (default
+all) and `?topic=` (a `topic_key` prefix filter). It never 500s on a malformed
+row — the row is returned with `parseError: true`.
+
+---
+
+## Prototypes (the visual function)
+
+When a question is clearer **shown** than told, the agent writes throwaway
+standalone interactive HTML to
+
+```
+.skillgrid/prototype/<topic>/<variant>.html
+```
+
+and sets `content.visual` to `<topic>/<variant>.html`. The dashboard serves it at
+`GET /prototype/<topic>/<variant>.html` (sandboxed to `.skillgrid/prototype/`,
+traversal-guarded) and renders it in a **sandboxed iframe**
+(`sandbox="allow-scripts"`, no `allow-same-origin`) inside the decision card.
+
+The prototype is **throwaway by design** — the durable record is the decision
+observation and its version history, not the HTML. It is a **distinct subtree**
+from the Phase 7 `.stitch/` prototypes gallery (`GET /prototypes/{id}`):
+
+| | Decision companion | Phase 7 gallery |
+|---|--------------------|-----------------|
+| Served at | `GET /prototype/{id...}` | `GET /prototypes/{id}` |
+| Sandbox root | `.skillgrid/prototype/` | `.stitch/` |
+| Purpose | Throwaway interview visuals | Reusable design prototypes |
+
+The `sketch` skill's default output location is `sketch.dir`
+(`{specs_root}/{topic}/sketches/`). For the companion flow, **point `sketch` at
+`.skillgrid/prototype/<topic>/`** (set `sketch.dir` in `.skillgrid/config.yaml`)
+so the variants land where the dashboard can serve them; the companion then
+reads the variant id back into `content.visual`.
+
+---
+
+## Durability
+
+Everything is in the Mnemonic SQLite store: it survives a server restart, is
+versioned (`observation_versions`), owned, and governed (`visibility`,
+`acl_grants`). A resumed session reads the same decisions — the agent's poll
+(`mem_search`) and the dashboard's poll (`GET /mnemonic/decisions`) see the same
+rows, so the companion survives a process boundary with no extra state.
+
+---
+
+## Next step
+
+- [Serve — the web dashboard](./09-serve-dashboard.md) — the UI that renders the decision cards.
+- [Memory and indexing](./05-memory-and-indexing.md) — the store the bridge reads and writes.
diff --git a/skillgrid-cli/internal/mnemonic/http/decisions.go b/skillgrid-cli/internal/mnemonic/http/decisions.go
new file mode 100644
index 0000000..efd4f57
--- /dev/null
+++ b/skillgrid-cli/internal/mnemonic/http/decisions.go
@@ -0,0 +1,259 @@
+package http
+
+import (
+	"encoding/json"
+	"net/http"
+	"strconv"
+	"strings"
+)
+
+// Phase 8 — Mnemonic decision bridge (change 2026-09-19-embed-visual-companion).
+// The agent posts interview questions as type=decision observations (content is
+// a structured JSON payload with a convention `state` field); the dashboard
+// lists them via GET /mnemonic/decisions and the user answers via
+// POST /mnemonic/decisions/{id}/answer, which flips state pending→answered
+// through the same version-append path mem_update uses (memory.UpdateContent).
+// No new table, no migration: decisions are a convention over observations.
+
+// decisionState is the convention approval gate inside content. The row's
+// `status` column is closed to active|superseded|archived (governance
+// lifecycle), so pending/answered lives here, in the JSON content.
+const (
+	decisionStatePending     = "pending"
+	decisionStateAnswered    = "answered"
+	decisionStateSuperseded  = "superseded"
+)
+
+// decisionOption is one lettered option in a decision payload.
+type decisionOption struct {
+	ID        string `json:"id"`
+	Label     string `json:"label"`
+	Rationale string `json:"rationale,omitempty"`
+}
+
+// decisionContent is the structured decision payload stored in
+// observations.content. A malformed payload on a type=decision row is a data
+// problem, never a 500: it is flagged (ParseError), never fatal.
+type decisionContent struct {
+	Question       string           `json:"question"`
+	Options        []decisionOption `json:"options"`
+	Recommended    string           `json:"recommended,omitempty"`
+	State          string           `json:"state"` // pending | answered | superseded
+	AnsweredOption string           `json:"answeredOption,omitempty"`
+	AnswerNote     string           `json:"answerNote,omitempty"`
+	UpdatedBy      string           `json:"updatedBy,omitempty"`
+	Visual         string           `json:"visual,omitempty"`
+}
+
+// decisionRow is one row in the GET /mnemonic/decisions payload.
+type decisionRow struct {
+	ID         int64           `json:"id"`
+	TopicKey   string          `json:"topicKey"`
+	Title      string          `json:"title"`
+	CreatedAt  string          `json:"createdAt"`
+	UpdatedAt  string          `json:"updatedAt"`
+	Visibility string          `json:"visibility"`
+	Content    decisionContent `json:"content"`
+	ParseError bool            `json:"parseError,omitempty"`
+}
+
+// parseDecisionContent parses a decision content payload. ok=false means the
+// row's content is unreadable as a decision (invalid JSON or missing
+// question): the caller flags the row, it never 500s.
+func parseDecisionContent(raw string) (decisionContent, bool) {
+	var c decisionContent
+	if err := json.Unmarshal([]byte(raw), &c); err != nil || strings.TrimSpace(c.Question) == "" {
+		return decisionContent{}, false
+	}
+	return c, true
+}
+
+func (s *Server) registerDecisionRoutes() {
+	s.mux.HandleFunc("GET /mnemonic/decisions", s.handleDecisions)
+	s.mux.HandleFunc("POST /mnemonic/decisions/{id}/answer", s.requireWriteAuth(s.handleDecisionAnswer))
+}
+
+// handleDecisions returns the project's decision observations: type='decision'
+// rows, not deleted, excluding private rows (the dashboard reader is a
+// team-scoped identity, mirroring the memories list). `?state=` filters the
+// convention state (pending|answered|superseded; default all), `?topic=` an
+// optional topic_key prefix filter. Rows are ordered ascending by created_at.
+// A row with malformed content is still returned, flagged parseError.
+func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
+	projectID, err := projectFromRequest(r)
+	if err != nil {
+		writeError(w, http.StatusBadRequest, err.Error())
+		return
+	}
+	h, cleanup, err := s.openHandleFor(projectID)
+	if err != nil {
+		writeError(w, http.StatusNotFound, err.Error())
+		return
+	}
+	defer cleanup()
+
+	wantState := r.URL.Query().Get("state")
+	wantTopic := r.URL.Query().Get("topic")
+
+	rows, err := h.Store().DB.QueryContext(r.Context(), `
+		SELECT id, COALESCE(topic_key,''), COALESCE(title,''), COALESCE(content,''),
+		       COALESCE(visibility,''), COALESCE(created_at,''), COALESCE(updated_at,'')
+		FROM observations
+		WHERE project = ? AND type = 'decision' AND deleted_at IS NULL
+		  AND COALESCE(visibility,'') != 'private'
+		ORDER BY created_at ASC, id ASC
+		LIMIT 200`, projectID)
+	if err != nil {
+		writeError(w, http.StatusInternalServerError, err.Error())
+		return
+	}
+	defer rows.Close()
+
+	out := []decisionRow{}
+	for rows.Next() {
+		var d decisionRow
+		var content string
+		if err := rows.Scan(&d.ID, &d.TopicKey, &d.Title, &content, &d.Visibility, &d.CreatedAt, &d.UpdatedAt); err != nil {
+			writeError(w, http.StatusInternalServerError, err.Error())
+			return
+		}
+		if c, ok := parseDecisionContent(content); ok {
+			d.Content = c
+		} else {
+			d.ParseError = true
+		}
+		if wantTopic != "" && !strings.HasPrefix(d.TopicKey, wantTopic) {
+			continue
+		}
+		if wantState != "" && wantState != "all" && !d.ParseError && d.Content.State != wantState {
+			continue
+		}
+		out = append(out, d)
+	}
+	if err := rows.Err(); err != nil {
+		writeError(w, http.StatusInternalServerError, err.Error())
+		return
+	}
+	writeJSON(w, http.StatusOK, out)
+}
+
+// decisionAnswerBody is the R5 answer contract: {optionId, note}.
+type decisionAnswerBody struct {
+	OptionID string `json:"optionId"`
+	Note     string `json:"note"`
+}
+
+// handleDecisionAnswer flips a decision's convention state pending→answered:
+// it records the chosen option + note + updatedBy (user:<name>) in the
+// content and persists it through memory.UpdateContent, so an
+// observation_versions row is appended and revision_count/updated_at bumped —
+// the same governed path mem_update uses. Re-answering an answered decision
+// applies the same update path again (state stays answered, the new answer is
+// recorded, another version row is appended): append-only history, the latest
+// content carries the latest answer. A bad id is 404; a row whose content has
+// no valid state (missing, malformed, or unrecognized) is 409.
+func (s *Server) handleDecisionAnswer(w http.ResponseWriter, r *http.Request) {
+	projectID, err := projectFromRequest(r)
+	if err != nil {
+		writeError(w, http.StatusBadRequest, err.Error())
+		return
+	}
+	id64, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
+	if err != nil {
+		writeError(w, http.StatusBadRequest, "id must be an integer")
+		return
+	}
+	var in decisionAnswerBody
+	if err := decodeJSON(r, &in); err != nil {
+		writeError(w, http.StatusBadRequest, err.Error())
+		return
+	}
+	if strings.TrimSpace(in.OptionID) == "" {
+		writeError(w, http.StatusBadRequest, "optionId is required")
+		return
+	}
+
+	h, cleanup, err := s.openHandleFor(projectID)
+	if err != nil {
+		writeError(w, http.StatusNotFound, err.Error())
+		return
+	}
+	defer cleanup()
+
+	var (
+		content  string
+		curState string
+	)
+	err = h.Store().DB.QueryRowContext(r.Context(),
+		`SELECT COALESCE(content,''), COALESCE(status,'')
+		 FROM observations
+		 WHERE id = ? AND project = ? AND type = 'decision' AND deleted_at IS NULL`,
+		id64, projectID).Scan(&content, &curState)
+	if err != nil {
+		writeError(w, http.StatusNotFound, "no such decision")
+		return
+	}
+	c, ok := parseDecisionContent(content)
+	if !ok {
+		writeError(w, http.StatusConflict, "decision content has no valid state")
+		return
+	}
+	switch c.State {
+	case decisionStatePending, decisionStateAnswered:
+		// pending → answered flip, or re-answer of an answered decision:
+		// the same update path below (state stays answered on re-answer, the
+		// new answer is recorded, a version row is appended).
+	case decisionStateSuperseded:
+		// The agent owns the pending→superseded transition; the answer gate
+		// does not apply.
+		writeError(w, http.StatusConflict, "decision is superseded")
+		return
+	default:
+		_ = curState
+		writeError(w, http.StatusConflict, "decision content has no valid state")
+		return
+	}
+
+	if !decisionOptionExists(c, in.OptionID) {
+		writeError(w, http.StatusUnprocessableEntity, "optionId not in this decision's options")
+		return
+	}
+
+	name := strings.TrimSpace(r.URL.Query().Get("answerer"))
+	if name == "" {
+		name = "unknown"
+	}
+	c.State = decisionStateAnswered
+	c.AnsweredOption = in.OptionID
+	c.AnswerNote = in.Note
+	c.UpdatedBy = "user:" + name
+	newContent, err := json.Marshal(c)
+	if err != nil {
+		writeError(w, http.StatusInternalServerError, err.Error())
+		return
+	}
+	if err := h.Memory().UpdateContent(r.Context(), id64, string(newContent)); err != nil {
+		if strings.Contains(err.Error(), "not found") {
+			writeError(w, http.StatusNotFound, err.Error())
+			return
+		}
+		writeError(w, http.StatusInternalServerError, err.Error())
+		return
+	}
+	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "state": decisionStateAnswered})
+}
+
+// decisionOptionExists reports whether the option id is one of the decision's
+// options (an empty option list accepts any id — a payload that parsed but
+// carries no options is still answerable by convention).
+func decisionOptionExists(c decisionContent, optionID string) bool {
+	if len(c.Options) == 0 {
+		return true
+	}
+	for _, o := range c.Options {
+		if o.ID == optionID {
+			return true
+		}
+	}
+	return false
+}
diff --git a/skillgrid-cli/internal/mnemonic/http/decisions_test.go b/skillgrid-cli/internal/mnemonic/http/decisions_test.go
new file mode 100644
index 0000000..1dace79
--- /dev/null
+++ b/skillgrid-cli/internal/mnemonic/http/decisions_test.go
@@ -0,0 +1,566 @@
+package http
+
+import (
+	"context"
+	"encoding/json"
+	"net/http"
+	"net/http/httptest"
+	"strings"
+	"testing"
+
+	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
+	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
+	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
+)
+
+// Decision bridge (change 2026-09-19-embed-visual-companion, P1): the agent
+// posts interview questions as type=decision observations (content is a
+// structured JSON payload with a convention `state` field); the dashboard lists
+// them via GET /mnemonic/decisions and the user answers via
+// POST /mnemonic/decisions/{id}/answer, which flips state pending→answered
+// through the same version-append path mem_update uses. These are the RED
+// acceptance tests for both routes (TICKET-01); the implementation is
+// TICKET-02 (decisions.go + memory.UpdateContent + route registration).
+
+// decisionSeedContent is a well-formed pending decision payload.
+const decisionSeedContent = `{"question":"Which layout?","options":[{"id":"a","label":"Single column"},{"id":"b","label":"Two column"}],"recommended":"a","state":"pending"}`
+
+// newDecisionTestServer builds a Server over a seeded store for proj plus an
+// open ProjectHandle the tests use to seed rows and assert store state.
+func newDecisionTestServer(t *testing.T) (*Server, *service.ProjectHandle) {
+	t.Helper()
+	dataDir := t.TempDir()
+	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
+	st, err := store.Open(dataDir, proj)
+	if err != nil {
+		t.Fatalf("open store: %v", err)
+	}
+	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, summary, status) VALUES ('s-dec', ?, '.', '2026-01-01T00:00:00Z', 'seed', 'ended')`, proj); err != nil {
+		t.Fatalf("seed session: %v", err)
+	}
+	st.Close()
+	svc := service.New(dataDir)
+	s := NewServer(svc)
+	h, cleanup, err := s.openHandleFor(proj)
+	if err != nil {
+		t.Fatalf("open handle: %v", err)
+	}
+	t.Cleanup(cleanup)
+	return s, h
+}
+
+// seedDecision inserts a type=decision observation directly via the store and
+// returns its id. created is the RFC3339 timestamp used for created_at
+// (updated_at matches), so tests control ordering deterministically.
+func seedDecision(t *testing.T, h *service.ProjectHandle, topicKey, contentJSON, visibility, created string) int64 {
+	t.Helper()
+	db := h.Store().DB
+	res, err := db.Exec(`INSERT INTO observations
+		(session_id, type, title, content, project, scope, topic_key, owner, visibility, status, revision_count, normalized_hash, created_at, updated_at)
+		VALUES ('s-dec', 'decision', 'Decision: `+strings.ReplaceAll(topicKey, "interview/", "")+`', ?, ?, 'project', ?, 's-dec', ?, 'active', 0, 'hash-' || ?, ?, ?)`,
+		contentJSON, proj, topicKey, visibility, topicKey, created, created)
+	if err != nil {
+		t.Fatalf("seed decision: %v", err)
+	}
+	id, err := res.LastInsertId()
+	if err != nil {
+		t.Fatalf("seed decision last insert id: %v", err)
+	}
+	return id
+}
+
+// getDecisions GETs /mnemonic/decisions and decodes the rows.
+func getDecisions(t *testing.T, s *Server, query string) ([]decisionRow, int) {
+	t.Helper()
+	req := httptest.NewRequest(http.MethodGet, "/mnemonic/decisions?project="+proj+query, nil)
+	w := httptest.NewRecorder()
+	s.Handler().ServeHTTP(w, req)
+	if w.Code != http.StatusOK {
+		t.Fatalf("GET /mnemonic/decisions: got %d: %s", w.Code, w.Body.String())
+	}
+	var out []decisionRow
+	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
+		t.Fatalf("decode decisions: %v", err)
+	}
+	return out, w.Code
+}
+
+// postDecisionAnswer POSTs the answer body and returns the recorder.
+func postDecisionAnswer(t *testing.T, s *Server, id int64, body string) *httptest.ResponseRecorder {
+	t.Helper()
+	req := httptest.NewRequest(http.MethodPost, "/mnemonic/decisions/"+itoaStr(id)+"/answer?project="+proj, strings.NewReader(body))
+	req.Header.Set("Content-Type", "application/json")
+	w := httptest.NewRecorder()
+	s.Handler().ServeHTTP(w, req)
+	return w
+}
+
+// decisionByTopic returns the listed row with the given topic key, or nil.
+func decisionByTopic(rows []decisionRow, topic string) *decisionRow {
+	for i := range rows {
+		if rows[i].TopicKey == topic {
+			return &rows[i]
+		}
+	}
+	return nil
+}
+
+// [decision-list-roundtrip] A pending decision posted by the agent is listed
+// with its content parsed.
+func TestDecisionListRoundtrip(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	seedDecision(t, h, "interview/demo/layout-1", decisionSeedContent, "team", "2026-01-01T00:00:00Z")
+
+	rows, _ := getDecisions(t, s, "&state=pending")
+	if len(rows) != 1 {
+		t.Fatalf("want 1 pending decision, got %d: %+v", len(rows), rows)
+	}
+	if rows[0].Content.Question != "Which layout?" {
+		t.Errorf("content not parsed: %+v", rows[0].Content)
+	}
+	if rows[0].Content.Recommended != "a" || rows[0].Content.State != "pending" {
+		t.Errorf("fields missing: %+v", rows[0].Content)
+	}
+	if len(rows[0].Content.Options) != 2 || rows[0].Content.Options[1].ID != "b" {
+		t.Errorf("options not parsed: %+v", rows[0].Content.Options)
+	}
+}
+
+// [decision-skips-malformed] A malformed content JSON on a decision row is
+// flagged (parseError) or skipped — never a 500 — and good rows still surface.
+func TestDecisionSkipsMalformed(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	seedDecision(t, h, "interview/demo/bad", `{"question":"ok","options":[`, "team", "2026-01-01T00:00:00Z")
+	seedDecision(t, h, "interview/demo/good", `{"question":"good","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:01Z")
+
+	rows, code := getDecisions(t, s, "")
+	if code != http.StatusOK {
+		t.Fatalf("malformed row must not 500, got %d", code)
+	}
+	if decisionByTopic(rows, "interview/demo/good") == nil {
+		t.Errorf("good decision missing from list: %+v", rows)
+	}
+}
+
+// [decision-visible-to-reader] A team decision is visible to the dashboard
+// reader; a private one is not.
+func TestDecisionVisibilityGate(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	seedDecision(t, h, "interview/demo/team", `{"question":"t","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
+	seedDecision(t, h, "interview/demo/priv", `{"question":"p","options":[{"id":"a","label":"A"}],"state":"pending"}`, "private", "2026-01-01T00:00:01Z")
+
+	rows, _ := getDecisions(t, s, "")
+	if decisionByTopic(rows, "interview/demo/team") == nil {
+		t.Errorf("team decision missing from list: %+v", rows)
+	}
+	if d := decisionByTopic(rows, "interview/demo/priv"); d != nil {
+		t.Errorf("private decision leaked to reader: %+v", d)
+	}
+}
+
+// [decision-ordering] Rows are returned ascending by created_at.
+func TestDecisionOrderingAscending(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	seedDecision(t, h, "interview/demo/q1", `{"question":"one","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
+	seedDecision(t, h, "interview/demo/q2", `{"question":"two","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:01Z")
+	seedDecision(t, h, "interview/demo/q3", `{"question":"three","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:02Z")
+
+	rows, _ := getDecisions(t, s, "")
+	if len(rows) != 3 {
+		t.Fatalf("want 3 decisions, got %d", len(rows))
+	}
+	for i, want := range []string{"interview/demo/q1", "interview/demo/q2", "interview/demo/q3"} {
+		if rows[i].TopicKey != want {
+			t.Errorf("order[%d] = %q, want %q", i, rows[i].TopicKey, want)
+		}
+	}
+}
+
+// [decision-answer-records] The answer route flips state pending→answered and
+// records the chosen option + note.
+func TestDecisionAnswerRecords(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	id := seedDecision(t, h, "interview/demo/layout-1", decisionSeedContent, "team", "2026-01-01T00:00:00Z")
+
+	w := postDecisionAnswer(t, s, id, `{"optionId":"b","note":"two column reads better"}`)
+	if w.Code != http.StatusOK {
+		t.Fatalf("answer: got %d: %s", w.Code, w.Body.String())
+	}
+
+	rows, _ := getDecisions(t, s, "")
+	var got *decisionRow
+	for i := range rows {
+		if rows[i].ID == id {
+			got = &rows[i]
+		}
+	}
+	if got == nil {
+		t.Fatalf("decision not found after answer")
+	}
+	if got.Content.State != "answered" {
+		t.Errorf("state not flipped: %q", got.Content.State)
+	}
+	if got.Content.AnsweredOption != "b" {
+		t.Errorf("answeredOption not recorded: %+v", got.Content)
+	}
+	if got.Content.AnswerNote != "two column reads better" {
+		t.Errorf("answerNote not recorded: %+v", got.Content)
+	}
+	if !strings.HasPrefix(got.Content.UpdatedBy, "user:") {
+		t.Errorf("updatedBy not user:<name>: %q", got.Content.UpdatedBy)
+	}
+}
+
+// [decision-answer-appends-version] R6: answering twice yields two
+// observation_versions rows; the latest content carries the second answer
+// (append-only history, never overwrites the history).
+func TestDecisionAnswerAppendsVersion(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	id := seedDecision(t, h, "interview/demo/v", `{"question":"q","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
+	w := postDecisionAnswer(t, s, id, `{"optionId":"a","note":"first"}`)
+	if w.Code != http.StatusOK {
+		t.Fatalf("first answer: got %d: %s", w.Code, w.Body.String())
+	}
+	w = postDecisionAnswer(t, s, id, `{"optionId":"a","note":"second"}`)
+	if w.Code != http.StatusOK {
+		t.Fatalf("second answer: got %d: %s", w.Code, w.Body.String())
+	}
+	var n int
+	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM observation_versions WHERE observation_id = ?`, id).Scan(&n); err != nil {
+		t.Fatalf("count versions: %v", err)
+	}
+	if n != 2 {
+		t.Errorf("expected 2 version rows after two answers, got %d", n)
+	}
+	rows, _ := getDecisions(t, s, "")
+	var got *decisionRow
+	for i := range rows {
+		if rows[i].ID == id {
+			got = &rows[i]
+		}
+	}
+	if got == nil {
+		t.Fatalf("decision not found after re-answer")
+	}
+	if got.Content.State != "answered" || got.Content.AnsweredOption != "a" || got.Content.AnswerNote != "second" {
+		t.Errorf("latest content must carry the second answer: %+v", got.Content)
+	}
+}
+
+// [decision-answer-reanswer-updates] R6: a re-answer of an answered decision
+// is a 200 that records the NEW answer (option + note + updatedBy) and appends
+// a version — the history is append-only, the latest content is the latest
+// answer.
+func TestDecisionAnswerIdempotent(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	id := seedDecision(t, h, "interview/demo/idem", `{"question":"q","options":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
+	postDecisionAnswer(t, s, id, `{"optionId":"a","note":"first"}`)
+	w := postDecisionAnswer(t, s, id, `{"optionId":"b","note":"second"}`)
+	if w.Code != http.StatusOK {
+		t.Fatalf("re-answer should be 200, got %d: %s", w.Code, w.Body.String())
+	}
+
+	rows, _ := getDecisions(t, s, "")
+	var got *decisionRow
+	for i := range rows {
+		if rows[i].ID == id {
+			got = &rows[i]
+		}
+	}
+	if got == nil {
+		t.Fatalf("decision not found after re-answer")
+	}
+	if got.Content.State != "answered" || got.Content.AnsweredOption != "b" || got.Content.AnswerNote != "second" {
+		t.Errorf("re-answer must record the new answer: %+v", got.Content)
+	}
+	if got.Content.UpdatedBy != "user:unknown" {
+		t.Errorf("updatedBy must reflect the latest answerer: %q", got.Content.UpdatedBy)
+	}
+	var n int
+	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM observation_versions WHERE observation_id = ?`, id).Scan(&n); err != nil {
+		t.Fatalf("count versions: %v", err)
+	}
+	if n != 2 {
+		t.Errorf("re-answer must append a version, got %d rows, want 2", n)
+	}
+}
+
+// [decision-answer-404] An unknown decision id is 404.
+func TestDecisionAnswer404(t *testing.T) {
+	s, _ := newDecisionTestServer(t)
+	w := postDecisionAnswer(t, s, 999999, `{"optionId":"a"}`)
+	if w.Code != http.StatusNotFound {
+		t.Errorf("bad id should 404, got %d (%s)", w.Code, w.Body.String())
+	}
+}
+
+// [decision-answer-409] A decision row whose content lacks a valid state
+// (or is malformed) is 409 — the gate is the convention field, not status.
+func TestDecisionAnswer409MissingState(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	noState := seedDecision(t, h, "interview/demo/nostate", `{"question":"q","options":[{"id":"a","label":"A"}]}`, "team", "2026-01-01T00:00:00Z")
+	w := postDecisionAnswer(t, s, noState, `{"optionId":"a"}`)
+	if w.Code != http.StatusConflict {
+		t.Errorf("missing state should 409, got %d (%s)", w.Code, w.Body.String())
+	}
+
+	badState := seedDecision(t, h, "interview/demo/badstate", `{"question":"q","options":[{"id":"a","label":"A"}],"state":"bogus"}`, "team", "2026-01-01T00:00:01Z")
+	w = postDecisionAnswer(t, s, badState, `{"optionId":"a"}`)
+	if w.Code != http.StatusConflict {
+		t.Errorf("invalid state should 409, got %d (%s)", w.Code, w.Body.String())
+	}
+
+	malformed := seedDecision(t, h, "interview/demo/malformed", `{"question":"q","options":[`, "team", "2026-01-01T00:00:02Z")
+	w = postDecisionAnswer(t, s, malformed, `{"optionId":"a"}`)
+	if w.Code != http.StatusConflict {
+		t.Errorf("malformed content should 409, got %d (%s)", w.Code, w.Body.String())
+	}
+}
+
+// [decision-answer-requires-auth] With SKILLGRID_HTTP_TOKEN set, a no-auth
+// answer POST is 401; with the token it succeeds.
+func TestDecisionAnswerRequiresAuth(t *testing.T) {
+	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret-token")
+	dataDir := t.TempDir()
+	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
+	st, err := store.Open(dataDir, proj)
+	if err != nil {
+		t.Fatalf("open store: %v", err)
+	}
+	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, summary, status) VALUES ('s-dec', ?, '.', '2026-01-01T00:00:00Z', 'seed', 'ended')`, proj); err != nil {
+		t.Fatalf("seed session: %v", err)
+	}
+	st.Close()
+	svc := service.New(dataDir)
+	s := NewServer(svc)
+	h, cleanup, err := s.openHandleFor(proj)
+	if err != nil {
+		t.Fatalf("open handle: %v", err)
+	}
+	t.Cleanup(cleanup)
+	id := seedDecision(t, h, "interview/demo/auth", `{"question":"q","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team", "2026-01-01T00:00:00Z")
+
+	w := postDecisionAnswer(t, s, id, `{"optionId":"a"}`)
+	if w.Code != http.StatusUnauthorized {
+		t.Errorf("no-auth answer with token set: got %d want 401", w.Code)
+	}
+
+	req := httptest.NewRequest(http.MethodPost, "/mnemonic/decisions/"+itoaStr(id)+"/answer?project="+proj, strings.NewReader(`{"optionId":"a"}`))
+	req.Header.Set("Content-Type", "application/json")
+	req.Header.Set("Authorization", "Bearer secret-token")
+	rr := httptest.NewRecorder()
+	s.Handler().ServeHTTP(rr, req)
+	if rr.Code != http.StatusOK {
+		t.Errorf("authed answer should 200, got %d (%s)", rr.Code, rr.Body.String())
+	}
+}
+
+// [decision-upsert-bumps-revision] Saving the same topic_key twice (the agent
+// in-place question update, mem_save) upserts the same row: one row,
+// revision_count incremented, a version appended.
+func TestDecisionUpsertBumpsRevision(t *testing.T) {
+	_, h := newDecisionTestServer(t)
+	topic := "interview/demo/upsert"
+	in := memory.SaveInput{
+		SessionID: "s-dec", Type: "decision",
+		Title:   "Decision: upsert",
+		Content: `{"question":"q1","options":[{"id":"a","label":"A"}],"recommended":"a","state":"pending"}`,
+		Scope:   "project", TopicKey: topic, Owner: "s-dec",
+	}
+	id1, err := h.Memory().Save(context.Background(), in)
+	if err != nil {
+		t.Fatalf("first save: %v", err)
+	}
+	// session_id is an FK to sessions: the upsert keeps the same valid session.
+	in2 := in
+	in2.Content = `{"question":"q1 v2","options":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"recommended":"b","state":"pending"}`
+	id2, err := h.Memory().Save(context.Background(), in2)
+	if err != nil {
+		t.Fatalf("second save: %v", err)
+	}
+	if id1 != id2 {
+		t.Fatalf("same topic_key must upsert the same row: %d != %d", id1, id2)
+	}
+	var rowCount, revisionCount int
+	if err := h.Store().DB.QueryRow(`SELECT COUNT(*), COALESCE(MAX(revision_count),0) FROM observations WHERE id = ?`, id1).Scan(&rowCount, &revisionCount); err != nil {
+		t.Fatalf("read row: %v", err)
+	}
+	if rowCount != 1 {
+		t.Errorf("want 1 row for the topic_key, got %d", rowCount)
+	}
+	if revisionCount < 1 {
+		t.Errorf("revision_count not incremented by upsert, got %d", revisionCount)
+	}
+	var versions int
+	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM observation_versions WHERE observation_id = ?`, id1).Scan(&versions); err != nil {
+		t.Fatalf("count versions: %v", err)
+	}
+	if versions < 1 {
+		t.Errorf("upsert must append a version row, got %d", versions)
+	}
+}
+
+// [decision-400-validation] A missing optionId is 400 (before the 404/409
+// decision-specific checks).
+func TestDecisionAnswer400MissingOption(t *testing.T) {
+	s, h := newDecisionTestServer(t)
+	id := seedDecision(t, h, "interview/demo/missing", decisionSeedContent, "team", "2026-01-01T00:00:00Z")
+	w := postDecisionAnswer(t, s, id, `{"note":"no option"}`)
+	if w.Code != http.StatusBadRequest {
+		t.Errorf("missing optionId should 400, got %d (%s)", w.Code, w.Body.String())
+	}
+}
+
+// [decision-roundtrip] The MCP round-trip guard (TICKET-07, change
+// 2026-09-19-embed-visual-companion): the agent posts a decision via mem_save
+// (type=decision, topic_key interview/<slug>/<question-id>, visibility team);
+// the dashboard lists it via GET /mnemonic/decisions; the agent's own
+// mem_search (SearchOwnerScoped — the same service the MCP mem_search tool
+// runs) finds it with a query consistent with the list; the user answers via
+// POST /mnemonic/decisions/{id}/answer; and a FRESH mem_search (a new service
+// over the same data dir, simulating the agent polling later in a new session)
+// sees content.state=answered + the answered option. This proves the
+// agent-side read-back works with the frozen MCP surface.
+func TestDecisionRoundTrip(t *testing.T) {
+	ctx := context.Background()
+	dataDir := t.TempDir()
+	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
+	topic := "interview/demo/roundtrip-1"
+
+	// Agent session (the writer identity), seeded directly into the proj store
+	// — the same fixture shape newDecisionTestServer uses, so the session id
+	// is a valid FK target for observations.session_id in proj. The session id
+	// is also the default owner of the saved row, so the same agent reads it
+	// back under per-owner visibility enforcement.
+	sid := "s-roundtrip"
+	{
+		st, err := store.Open(dataDir, proj)
+		if err != nil {
+			t.Fatalf("open store: %v", err)
+		}
+		if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, summary, status) VALUES ('s-roundtrip', ?, '.', '2026-01-01T00:00:00Z', 'seed', 'ended')`, proj); err != nil {
+			t.Fatalf("seed session: %v", err)
+		}
+		st.Close()
+	}
+
+	// 1. Agent → user: mem_save the pending decision (same payload shape the
+	// agent posts; owner defaults to the session id, so the same agent can
+	// read it back under per-owner visibility enforcement). The frozen mem_save
+	// has no visibility arg, so the companion's visibility=team convention is
+	// applied via mem_share (the only path that leaves private).
+	svc0 := service.New(dataDir)
+	h0, cleanup0, err := svc0.Open(proj)
+	if err != nil {
+		t.Fatalf("open for save: %v", err)
+	}
+	id, err := h0.Memory().Save(ctx, memory.SaveInput{
+		SessionID: sid,
+		Type:      "decision",
+		Title:     "Decision: roundtrip layout",
+		Content:   decisionSeedContent,
+		Scope:     "project",
+		TopicKey:  topic,
+	})
+	if err != nil {
+		cleanup0()
+		t.Fatalf("mem_save decision: %v", err)
+	}
+	if err := h0.Memory().Share(ctx, id, memory.ShareInput{Visibility: "team"}); err != nil {
+		cleanup0()
+		t.Fatalf("mem_share team: %v", err)
+	}
+	cleanup0()
+
+	// 2. Dashboard side: GET /mnemonic/decisions lists exactly the seeded
+	// decision, with its content parsed.
+	s := NewServer(service.New(dataDir))
+	rows, _ := getDecisions(t, s, "&state=pending")
+	if len(rows) != 1 {
+		t.Fatalf("want exactly 1 listed decision, got %d: %+v", len(rows), rows)
+	}
+	row := rows[0]
+	if row.TopicKey != topic {
+		t.Fatalf("listed row topic = %q, want %q", row.TopicKey, topic)
+	}
+	if row.Content.State != "pending" || row.Content.Question != "Which layout?" {
+		t.Errorf("listed content not parsed: %+v", row.Content)
+	}
+	if len(rows) != 1 || row.ParseError {
+		t.Fatalf("list must be exactly the seeded decision, unparsed: %+v", rows)
+	}
+
+	// 3. Agent-side mem_search (the same service the MCP mem_search tool
+	// uses — SearchOwnerScoped with the agent as reader) finds the decision
+	// with a query consistent with the list (a word of the listed content).
+	hits, err := searchDecisions(t, dataDir, sid, "layout")
+	if err != nil {
+		t.Fatalf("agent mem_search: %v", err)
+	}
+	if len(hits) != 1 || hits[0].ID != row.ID {
+		t.Fatalf("mem_search must surface the listed decision, got %+v", hits)
+	}
+	if hits[0].TopicKey != topic {
+		t.Errorf("mem_search hit topic = %q, want %q (consistent with the list)", hits[0].TopicKey, topic)
+	}
+
+	// 4. User → agent: the answer flips the convention state.
+	w := postDecisionAnswer(t, s, row.ID, `{"optionId":"b","note":"two column reads better"}`)
+	if w.Code != http.StatusOK {
+		t.Fatalf("answer: got %d: %s", w.Code, w.Body.String())
+	}
+
+	// 5. FRESH agent mem_search (a NEW service over the same data dir — the
+	// agent polling later in a resumed session, same query as step 3) sees the
+	// answered state and the chosen option in the content.
+	fresh, err := searchDecisions(t, dataDir, sid, "layout")
+	if err != nil {
+		t.Fatalf("fresh mem_search: %v", err)
+	}
+	if len(fresh) != 1 {
+		t.Fatalf("fresh mem_search must surface the decision, got %+v", fresh)
+	}
+	var content struct {
+		State          string `json:"state"`
+		AnsweredOption string `json:"answeredOption"`
+		AnswerNote     string `json:"answerNote"`
+		UpdatedBy      string `json:"updatedBy"`
+	}
+	if err := json.Unmarshal([]byte(fresh[0].Content), &content); err != nil {
+		t.Fatalf("fresh content is not decision JSON: %v (%q)", err, fresh[0].Content)
+	}
+	if content.State != "answered" {
+		t.Errorf("fresh mem_search must see state=answered, got %q", content.State)
+	}
+	if content.AnsweredOption != "b" {
+		t.Errorf("fresh mem_search must see the answered option, got %q", content.AnsweredOption)
+	}
+	if !strings.HasPrefix(content.UpdatedBy, "user:") {
+		t.Errorf("fresh content updatedBy must be user:<name>, got %q", content.UpdatedBy)
+	}
+}
+
+// searchDecisions runs the same search service the MCP mem_search tool uses
+// (Service.SearchOwnerScoped, the per-owner-enforced FTS path) as a fresh
+// service over dataDir, reading as the given owner, and returns only the
+// type=decision hits — the agent-side read-back leg of the round trip.
+func searchDecisions(t *testing.T, dataDir, readerOwner, query string) ([]memory.Observation, error) {
+	t.Helper()
+	svc := service.New(dataDir)
+	h, cleanup, err := svc.Open(proj)
+	if err != nil {
+		t.Fatalf("open service for search: %v", err)
+	}
+	defer cleanup()
+	hits, err := h.Memory().SearchOwnerScoped(context.Background(), readerOwner, "", query, "any", "", 20)
+	if err != nil {
+		return nil, err
+	}
+	var out []memory.Observation
+	for _, o := range hits {
+		if o.Type == "decision" {
+			out = append(out, o)
+		}
+	}
+	return out, nil
+}
diff --git a/skillgrid-cli/internal/mnemonic/http/openapi_phase8_test.go b/skillgrid-cli/internal/mnemonic/http/openapi_phase8_test.go
new file mode 100644
index 0000000..5b2f6d9
--- /dev/null
+++ b/skillgrid-cli/internal/mnemonic/http/openapi_phase8_test.go
@@ -0,0 +1,90 @@
+package http
+
+import (
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+
+	"gopkg.in/yaml.v3"
+)
+
+// TestPhase8_OpenAPI asserts openapi.yaml documents the Phase 8 decision-bridge
+// routes (change 2026-09-19-embed-visual-companion) and the /prototype serve
+// route, and that the spec still parses as YAML with no duplicate keys. The
+// earlier phases' content is asserted by TestPhase6/7_OpenAPI (substring
+// only); this is the first test that also parses the doc, which catches the
+// plain-scalar indent/colon bugs the substring checks never see.
+func TestPhase8_OpenAPI(t *testing.T) {
+	wd, err := os.Getwd()
+	if err != nil {
+		t.Fatalf("Getwd: %v", err)
+	}
+	path := filepath.Join(wd, "ui", "openapi.yaml")
+	data, err := os.ReadFile(path)
+	if err != nil {
+		t.Fatalf("read openapi.yaml: %v", err)
+	}
+	text := string(data)
+	for _, want := range []string{
+		"/mnemonic/decisions:",
+		"/mnemonic/decisions/{id}/answer:",
+		"/prototype/{id...}:",
+		"operationId: mnemonicDecisions",
+		"operationId: mnemonicDecisionAnswer",
+		"operationId: prototypeDecision",
+		"DecisionRow", "DecisionContent", "DecisionOption", "DecisionAnswerInput",
+		"mnemonic-decisions",
+		"pending", "answered", "superseded", // the state filter values
+		"409", "404", "400", // the answer route error semantics
+	} {
+		if !strings.Contains(text, want) {
+			t.Errorf("openapi.yaml missing Phase 8 content: %q", want)
+		}
+	}
+
+	// Parse the doc and assert no duplicate keys at the three maps the spec
+	// is built from (a duplicate silently drops a path/schema from the served
+	// spec — the substring checks can't catch it).
+	var node yaml.Node
+	if err := yaml.Unmarshal(data, &node); err != nil {
+		t.Fatalf("openapi.yaml does not parse: %v", err)
+	}
+	root := &node
+	if node.Kind == yaml.DocumentNode {
+		root = node.Content[0]
+	}
+	assertNoDupKeys(t, "top-level", root)
+	for i := 0; i < len(root.Content)-1; i += 2 {
+		switch root.Content[i].Value {
+		case "paths":
+			assertNoDupKeys(t, "paths", root.Content[i+1])
+		case "components":
+			c := root.Content[i+1]
+			for j := 0; j < len(c.Content)-1; j += 2 {
+				if c.Content[j].Value == "schemas" {
+					assertNoDupKeys(t, "schemas", c.Content[j+1])
+				}
+			}
+		}
+	}
+}
+
+// assertNoDupKeys fatals on any repeated key in a mapping node.
+func assertNoDupKeys(t *testing.T, label string, m *yaml.Node) {
+	t.Helper()
+	seen := map[string]int{}
+	var dups []string
+	for i := 0; i < len(m.Content)-1; i += 2 {
+		k := m.Content[i].Value
+		seen[k]++
+	}
+	for k, n := range seen {
+		if n > 1 {
+			dups = append(dups, k)
+		}
+	}
+	if len(dups) > 0 {
+		t.Fatalf("openapi.yaml %s has duplicate keys: %v", label, dups)
+	}
+}
diff --git a/skillgrid-cli/internal/mnemonic/http/prototype.go b/skillgrid-cli/internal/mnemonic/http/prototype.go
new file mode 100644
index 0000000..1429705
--- /dev/null
+++ b/skillgrid-cli/internal/mnemonic/http/prototype.go
@@ -0,0 +1,64 @@
+package http
+
+import (
+	"net/http"
+	"os"
+	"path/filepath"
+	"strings"
+)
+
+// prototypeRoot returns the .skillgrid/prototype/ directory (under sddRoot,
+// the repo root). This is the on-disk store for the companion's throwaway
+// interactive prototypes (agent-authored HTML), separate from .stitch/.
+func prototypeRoot() string { return filepath.Join(sddRoot(), ".skillgrid", "prototype") }
+
+// prototypeFile resolves a prototype id to an absolute path sandboxed to
+// .skillgrid/prototype/ — the same path-traversal guard shape as stitchFile
+// (prototypes.go). ok=false for empty/absolute ids, '..' segments, or any
+// path that escapes the root after filepath.Clean.
+func prototypeFile(id string) (string, bool) {
+	id = strings.TrimSpace(id)
+	if id == "" {
+		return "", false
+	}
+	// Reject absolute ids (leading / or a Windows drive) up front.
+	if filepath.IsAbs(id) || id[0] == '/' || (len(id) >= 2 && id[1] == ':') {
+		return "", false
+	}
+	root := prototypeRoot()
+	full := filepath.Join(root, filepath.FromSlash(id))
+	// Clean both and ensure the resolved path stays under root.
+	cleanFull := filepath.Clean(full)
+	cleanRoot := filepath.Clean(root)
+	rel, err := filepath.Rel(cleanRoot, cleanFull)
+	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
+		return "", false
+	}
+	return cleanFull, true
+}
+
+// handlePrototypeDecision serves GET /prototype/{id...} — the HTML of one
+// companion prototype, sandboxed to .skillgrid/prototype/. Traversal/absolute
+// ids → 400; unknown → 404. Rendered client-side in a sandboxed iframe
+// (sandbox="allow-scripts", no allow-same-origin).
+func (s *Server) handlePrototypeDecision(w http.ResponseWriter, r *http.Request) {
+	id := r.PathValue("id")
+	full, ok := prototypeFile(id)
+	if !ok {
+		writeError(w, http.StatusBadRequest, "invalid prototype id: "+id)
+		return
+	}
+	data, err := os.ReadFile(full)
+	if err != nil {
+		if os.IsNotExist(err) {
+			writeError(w, http.StatusNotFound, "unknown prototype: "+id)
+			return
+		}
+		writeError(w, http.StatusInternalServerError, err.Error())
+		return
+	}
+	w.Header().Set("Content-Type", "text/html; charset=utf-8")
+	w.Header().Set("X-Content-Type-Options", "nosniff")
+	w.WriteHeader(http.StatusOK)
+	_, _ = w.Write(data)
+}
diff --git a/skillgrid-cli/internal/mnemonic/http/prototype_test.go b/skillgrid-cli/internal/mnemonic/http/prototype_test.go
new file mode 100644
index 0000000..63fa676
--- /dev/null
+++ b/skillgrid-cli/internal/mnemonic/http/prototype_test.go
@@ -0,0 +1,92 @@
+package http
+
+import (
+	"net/http"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+
+	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
+)
+
+// newProtoServer builds a server whose sddRoot points at a temp repo with a
+// .skillgrid/prototype/ tree (the companion's throwaway HTML prototypes,
+// distinct from the .stitch/ design prototypes) plus a sibling file outside
+// the root that must NOT be reachable (path-traversal threat).
+func newProtoServer(t *testing.T) http.Handler {
+	t.Helper()
+	repo := t.TempDir()
+	proto := filepath.Join(repo, ".skillgrid", "prototype", "demo")
+	if err := os.MkdirAll(proto, 0o755); err != nil {
+		t.Fatalf("mkdir prototype: %v", err)
+	}
+	if err := os.WriteFile(filepath.Join(proto, "a.html"),
+		[]byte("<html><body><h1>variant A</h1></body></html>"), 0o644); err != nil {
+		t.Fatalf("write a.html: %v", err)
+	}
+	// A secret file OUTSIDE .skillgrid/prototype/ that must not be servable.
+	if err := os.WriteFile(filepath.Join(repo, "secret.html"),
+		[]byte("<html><body>SECRET</body></html>"), 0o644); err != nil {
+		t.Fatalf("write secret: %v", err)
+	}
+
+	t.Setenv("SKILLGRID_DOCS_CWD", repo)
+	dataDir := t.TempDir()
+	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
+	svc := service.New(dataDir)
+	return NewServer(svc).Handler()
+}
+
+func TestPrototypeServeAndTraversal(t *testing.T) {
+	h := newProtoServer(t)
+
+	// --- happy path: serves the seeded file as HTML ---
+	rr := doGet(t, h, "/prototype/demo/a.html")
+	if rr.Code != http.StatusOK {
+		t.Fatalf("/prototype/demo/a.html: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
+	}
+	if !strings.Contains(rr.Body.String(), "variant A") {
+		t.Errorf("content = %q, want 'variant A'", rr.Body.String())
+	}
+	ct := rr.Header().Get("Content-Type")
+	if !strings.Contains(ct, "text/html") {
+		t.Errorf("Content-Type = %q, want text/html", ct)
+	}
+	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
+		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
+	}
+
+	// --- traversal / absolute ids → 400 ---
+	// Bare `..` ids are rewritten by the mux's clean-path redirect before
+	// reaching the handler (307 to a path outside the /prototype subtree);
+	// %-encoded forms reach the handler and must be 400. A successful
+	// response must never be the secret (no escape past the root either way).
+	wants := map[string]int{
+		"307": http.StatusTemporaryRedirect,
+		"400": http.StatusBadRequest,
+		"404": http.StatusNotFound,
+	}
+	for _, tc := range []struct{ id, want string }{
+		{"../a.html", "307"},
+		{"..%2F..%2Fa.html", "400"},
+		{"%2e%2e/secret.html", "400"},
+		{"%2Fetc%2Fpasswd", "400"},
+		{"demo/../../secret.html", "307"},
+		{"demo/../secret.html", "307"},
+	} {
+		rr = doGet(t, h, "/prototype/"+tc.id)
+		if rr.Code != wants[tc.want] {
+			t.Errorf("id %q: status = %d, want %s; body=%s", tc.id, rr.Code, tc.want, rr.Body.String())
+		}
+		if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "SECRET") {
+			t.Errorf("id %q: served SECRET (escaped .skillgrid/prototype/)", tc.id)
+		}
+	}
+
+	// --- unknown file → 404 ---
+	rr = doGet(t, h, "/prototype/demo/nope.html")
+	if rr.Code != http.StatusNotFound {
+		t.Errorf("unknown: status = %d, want 404; body=%s", rr.Code, rr.Body.String())
+	}
+}
diff --git a/skillgrid-cli/internal/mnemonic/http/server.go b/skillgrid-cli/internal/mnemonic/http/server.go
index 1cdccd6..9e06c83 100644
--- a/skillgrid-cli/internal/mnemonic/http/server.go
+++ b/skillgrid-cli/internal/mnemonic/http/server.go
@@ -140,20 +140,21 @@ func (s *Server) registerRoutes() {
 	s.mux.HandleFunc("GET /web/entry/{id}", s.handleWebEntry)
 	s.mux.HandleFunc("GET /web/status", s.handleWebStatus)
 
 	s.mux.HandleFunc("GET /projects", s.handleProjects)
 	s.mux.HandleFunc("GET /project/current", s.handleProjectCurrent)
 
 	s.registerTeamsRoutes()
 	s.registerTrackerRoutes()
 	s.registerDocsRoutes()
 	s.registerGraphRoutes()
+	s.registerDecisionRoutes()
 	s.registerUIRoutes()
 }
 
 // registerGraphRoutes mounts the Phase 4 mnemonic vector-graph routes. The API
 // lives under the /mnemonic/graph/ subtree (data + nodes) so the bare
 // /mnemonic/graph path stays free for the SPA client router. The other
 // mnemonic client paths also get explicit SPA-shell routes so the API-prefix
 // 404 catch-all doesn't swallow them.
 func (s *Server) registerGraphRoutes() {
 	s.mux.HandleFunc("GET /mnemonic/graph/data", s.handleMnemonicGraph)
@@ -173,39 +174,48 @@ func (s *Server) registerGraphRoutes() {
 	s.mux.HandleFunc("GET /mnemonic/audit", s.handleMnemonicAudit)
 	s.mux.HandleFunc("GET /mnemonic/search", s.handleMnemonicListOrShell(s.handleMnemonicSearch))
 
 	// Phase 6 activity routes (agent event feed + live SSE stream + stats).
 	// /activity is the SPA route, so the list endpoints share it via
 	// shellOrJSON (browser → shell, API client Accept: application/json → JSON).
 	s.mux.HandleFunc("GET /activity", s.shellOrJSON(s.handleMnemonicActivityEvents))
 	s.mux.HandleFunc("GET /activity/events", s.handleMnemonicActivityEvents)
 	s.mux.HandleFunc("GET /activity/stats", s.handleMnemonicActivityStats)
 	s.mux.HandleFunc("GET /activity/stream", s.handleMnemonicActivityStream)
+	// Handoff Hub (change 015): change-snapshot log + one-call hub status.
+	s.mux.HandleFunc("GET /activity/snapshots", s.handleHandoffSnapshots)
+	s.mux.HandleFunc("GET /handoff/status", s.handleHandoffStatus)
 
 	// Phase 6 plans/specs routes (SDD aggregation from .skillgrid/*).
 	s.mux.HandleFunc("GET /plans", s.shellOrJSON(s.handlePlans))
 	s.mux.HandleFunc("GET /plans/{id}", s.handlePlanDetail)
 	s.mux.HandleFunc("GET /specs", s.handleSpecs)
 	s.mux.HandleFunc("GET /specs/{path...}", s.handleSpecContent)
 
 	// Phase 6 git routes (read-only commit log / diff / file-history / blame).
 	s.mux.HandleFunc("GET /git", s.shellOrJSON(s.handleGitCommits))
 	s.mux.HandleFunc("GET /git/commits", s.handleGitCommits)
 	s.mux.HandleFunc("GET /git/commits/{sha}", s.handleGitCommit)
 	s.mux.HandleFunc("GET /git/diff/{sha}", s.handleGitDiff)
 	s.mux.HandleFunc("GET /git/file-history", s.handleGitFileHistory)
 	s.mux.HandleFunc("GET /git/blame", s.handleGitBlame)
 
 	// Phase 7 prototypes routes (sandboxed .stitch/ gallery readers).
 	s.mux.HandleFunc("GET /prototypes", s.shellOrJSON(s.handlePrototypes))
 	s.mux.HandleFunc("GET /prototypes/{id...}", s.handlePrototype)
 
+	// Visual companion prototype serve route (sandboxed .skillgrid/prototype/
+	// readers). A distinct subtree from the plural /prototypes above;
+	// registered here (before registerUIRoutes) so the SPA fallback can't
+	// swallow it.
+	s.mux.HandleFunc("GET /prototype/{id...}", s.handlePrototypeDecision)
+
 	// The SPA shell uses base './', so on a /mnemonic/... path its module
 	// script is requested at /mnemonic/assets/... — serve the embedded build
 	// assets there (the root /assets/ route covers the top-level roots).
 	s.mux.HandleFunc("GET /mnemonic/assets/{rest...}", s.handleUIAsset)
 	// The bare client paths stay SPA shells for the client router. The Phase 5
 	// API routes above own the exact /mnemonic/{memories,sessions,search}
 	// paths, so only the paths without an API handler keep an explicit shell.
 	for _, p := range []string{"/mnemonic/graph", "/mnemonic/files"} {
 		p := p
 		s.mux.HandleFunc("GET "+p, s.handleShellPage)
diff --git a/skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml b/skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml
index a389b5d..3fce255 100644
--- a/skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml
+++ b/skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml
@@ -18,20 +18,21 @@ tags:
   - name: code
   - name: web
   - name: tracker
   - name: ui
   - name: mnemonic-graph
   - name: mnemonic-files
   - name: mnemonic-memories
   - name: mnemonic-sessions
   - name: mnemonic-audit
   - name: mnemonic-search
+  - name: mnemonic-decisions
 paths:
   /health:
     get:
       tags: [health]
       operationId: health
       summary: Health probe
       responses:
         "200":
           description: Service is up
           content:
@@ -2308,20 +2309,140 @@ paths:
               schema:
                 type: object
                 properties:
                   project: { type: string }
                   results:
                     type: array
                     items: { $ref: "#/components/schemas/SearchResult" }
                   total: { type: integer }
         "400": { $ref: "#/components/responses/Bad" }
         "500": { $ref: "#/components/responses/Bad" }
+  # ── Phase 8: Mnemonic decision bridge (change 2026-09-19-embed-visual-companion) ──
+  /mnemonic/decisions:
+    get:
+      tags: [mnemonic-decisions]
+      operationId: mnemonicDecisions
+      summary: List the project's decision observations (the visual companion)
+      description: |-
+        Returns the project's type=decision observations (not deleted, never
+        private) with their content parsed into a structured payload, ordered
+        ascending by created_at (capped at 200 rows). `state` filters the
+        convention state inside the content (pending|answered|superseded;
+        `all` or omitted = every state); a row whose content is malformed is
+        still returned, flagged parseError, and is only excluded by a
+        specific state filter. `topic` is an optional topic_key prefix filter
+        (the convention is interview/<slug>/<question-id>). The row's
+        `status` column is governance (active|superseded|archived); pending /
+        answered lives in the content JSON.
+      parameters:
+        - name: project
+          in: query
+          required: true
+          schema: { type: string }
+          example: my-project
+        - name: state
+          in: query
+          required: false
+          description: Filter on the convention state inside the content.
+          schema: { type: string, enum: [pending, answered, superseded, all] }
+        - name: topic
+          in: query
+          required: false
+          description: topic_key prefix filter (e.g. interview/<slug>/)
+          schema: { type: string }
+      responses:
+        "200":
+          description: Decision list (empty array when none match)
+          content:
+            application/json:
+              schema:
+                type: array
+                items: { $ref: "#/components/schemas/DecisionRow" }
+              example:
+                - id: 12
+                  topicKey: interview/demo/layout-1
+                  title: "Decision: layout-1"
+                  createdAt: "2026-09-19T00:00:00Z"
+                  updatedAt: "2026-09-19T00:00:00Z"
+                  visibility: team
+                  content:
+                    question: Which layout?
+                    options:
+                      - id: a
+                        label: Single column
+                      - id: b
+                        label: Two column
+                    recommended: a
+                    state: pending
+        "400":
+          $ref: "#/components/responses/Bad"
+        "404":
+          $ref: "#/components/responses/Bad"
+        "500":
+          $ref: "#/components/responses/Internal"
+  /mnemonic/decisions/{id}/answer:
+    post:
+      tags: [mnemonic-decisions]
+      operationId: mnemonicDecisionAnswer
+      summary: Answer a decision (write-gated)
+      description: |-
+        Write-gated (Bearer token, SKILLGRID_HTTP_TOKEN) answer: flips the
+        decision's convention state pending→answered, records the chosen
+        option + note + updatedBy (user:<name>) in the content, and persists
+        it through the same version-append path mem_update uses — an
+        observation_versions row is appended and revision_count/updated_at
+        bumped (durable, auditable). Re-answering an answered decision is a
+        200: the state stays answered, the new answer is recorded, and another
+        version is appended (the latest content carries the latest answer).
+        400 for a missing/invalid project or id, a malformed body, or a
+        missing optionId; 401 without the bearer token; 404 for an unknown id
+        (or non-decision row); 409 for content with no valid state (missing,
+        malformed, or unrecognized) or a superseded decision; 422 for an
+        optionId not in the decision's options.
+      parameters:
+        - name: id
+          in: path
+          required: true
+          schema: { type: string, description: Decision observation id (an integer) }
+        - name: project
+          in: query
+          required: true
+          schema: { type: string }
+        - name: answerer
+          in: query
+          required: false
+          description: Answerer name recorded as updatedBy (user:<name>; default "unknown")
+          schema: { type: string }
+      requestBody:
+        required: true
+        content:
+          application/json:
+            schema: { $ref: "#/components/schemas/DecisionAnswerInput" }
+            example: { optionId: b, note: two column reads better }
+      responses:
+        "200":
+          description: Answered
+          content:
+            application/json:
+              schema:
+                type: object
+                required: [ok, state]
+                properties:
+                  ok: { type: boolean }
+                  state: { type: string, enum: [answered] }
+              example: { ok: true, state: answered }
+        "400": { $ref: "#/components/responses/Bad" }
+        "401": { $ref: "#/components/responses/Bad" }
+        "404": { $ref: "#/components/responses/Bad" }
+        "409": { $ref: "#/components/responses/Bad" }
+        "422": { $ref: "#/components/responses/Bad" }
+        "500": { $ref: "#/components/responses/Internal" }
   /swagger-ui:
     get:
       tags: [ui]
       operationId: swaggerUI
       summary: Swagger UI (interactive API explorer)
       responses:
         "200":
           description: HTML page
           content:
             text/html: {}
@@ -2638,21 +2759,22 @@ paths:
         "404":
           description: Blame error / unknown path
           content:
             application/json:
               schema: { type: object, properties: { error: { type: string } } }
   /prototypes:
     get:
       tags: [prototypes]
       operationId: prototypesList
       summary: List .stitch/ design prototypes (sandboxed)
-      description: Returns the prototype .html files under .stitch/ as
+      description: |-
+        Returns the prototype .html files under .stitch/ as
         relative paths. Served as the SPA shell to a browser (Accept: text/html),
         as JSON to the API client (Accept: application/json).
       responses:
         "200":
           description: Prototype list
           content:
             application/json:
               schema:
                 type: object
                 properties:
@@ -2665,21 +2787,22 @@ paths:
         "404":
           description: No .stitch/ directory
           content:
             application/json:
               schema: { type: object, properties: { error: { type: string } } }
   /prototypes/{id}:
     get:
       tags: [prototypes]
       operationId: prototypeGet
       summary: Get one prototype HTML (sandboxed to .stitch/)
-      description: Serves the HTML of one prototype. Traversal/absolute ids
+      description: |-
+        Serves the HTML of one prototype. Traversal/absolute ids
         (.., leading /) → 400; unknown → 404. The file is sandboxed to .stitch/
         (Phase 7 path-traversal threat model) and served with nosniff.
       parameters:
         - name: id
           in: path
           required: true
           description: Prototype path relative to .stitch/
           schema: { type: string }
       responses:
         "200":
@@ -2691,20 +2814,57 @@ paths:
         "400":
           description: Invalid (traversal/absolute) id
           content:
             application/json:
               schema: { type: object, properties: { error: { type: string } } }
         "404":
           description: Unknown prototype
           content:
             application/json:
               schema: { type: object, properties: { error: { type: string } } }
+  /prototype/{id...}:
+    get:
+      tags: [mnemonic-decisions]
+      operationId: prototypeDecision
+      summary: Serve one decision-companion prototype (sandboxed to .skillgrid/prototype)
+      description: |-
+        Serves the HTML of one visual-companion prototype, sandboxed to
+        .skillgrid/prototype/ (a distinct subtree from the plural /prototypes
+        above, which is sandboxed to .stitch/). Traversal/absolute ids
+        (.., leading /) → 400; unknown → 404. Served with nosniff; rendered
+        client-side in a sandboxed iframe (sandbox="allow-scripts", no
+        allow-same-origin) inside the decision card. Throwaway by design —
+        the durable record is the decision observation + its version history.
+      parameters:
+        - name: id
+          in: path
+          required: true
+          description: Prototype path relative to .skillgrid/prototype/ (e.g. <topic>/<variant>.html)
+          schema: { type: string }
+      responses:
+        "200":
+          description: Prototype HTML
+          content:
+            text/html:
+              schema: { type: string }
+              example: "<html><body><h1>Variant A</h1></body></html>"
+        "400":
+          description: Invalid (traversal/absolute) id
+          content:
+            application/json:
+              schema: { type: object, properties: { error: { type: string } } }
+        "404":
+          description: Unknown prototype
+          content:
+            application/json:
+              schema: { type: object, properties: { error: { type: string } } }
+        "500": { $ref: "#/components/responses/Internal" }
 components:
   responses:
     Bad:
       description: Bad request
       content:
         application/json:
           schema:
             $ref: "#/components/schemas/HTTPError"
     Internal:
       description: Server error
@@ -2845,20 +3005,58 @@ components:
       properties:
         id: { type: integer }
         title: { type: string }
         type: { type: string }
         scope: { type: string }
         topic_key: { type: string }
         source: { type: string }
         preview: { type: string }
         created_at: { type: string }
         relevance: { type: number, description: 0..1 rank-derived score }
+    DecisionOption:
+      type: object
+      properties:
+        id: { type: string, description: "Stable option id (a/b/…)" }
+        label: { type: string }
+        rationale: { type: string }
+    DecisionContent:
+      type: object
+      required: [question, state]
+      properties:
+        question: { type: string, description: "The question, in domain language" }
+        options:
+          type: array
+          items: { $ref: "#/components/schemas/DecisionOption" }
+        recommended: { type: string, description: "Id of the agent-recommended option (highlighted in the UI)" }
+        state: { type: string, enum: [pending, answered, superseded], description: "The convention state (a content field, not the governance status)" }
+        answeredOption: { type: string, description: "Id of the option the user chose (set by the answer)" }
+        answerNote: { type: string, description: "The user's answer note (set by the answer)" }
+        updatedBy: { type: string, description: "Last writer, agent, or user:<name> after an answer" }
+        visual: { type: string, description: "Prototype id <topic>/<variant>.html served from .skillgrid/prototype/ (GET /prototype/…); rendered in a sandboxed iframe when present" }
+    DecisionRow:
+      type: object
+      required: [id, topicKey, title, createdAt, updatedAt, visibility, content]
+      properties:
+        id: { type: integer, format: int64 }
+        topicKey: { type: string, description: "Convention topic_key interview/<slug>/<question-id>" }
+        title: { type: string }
+        createdAt: { type: string }
+        updatedAt: { type: string }
+        visibility: { type: string }
+        content: { $ref: "#/components/schemas/DecisionContent" }
+        parseError: { type: boolean, description: "True when the row's content is not valid decision JSON (the row is still returned, never a 500)" }
+    DecisionAnswerInput:
+      type: object
+      required: [optionId]
+      properties:
+        optionId: { type: string, description: "Id of the option the user chose" }
+        note: { type: string, description: "Optional answer note" }
     DocChange:
       type: object
       required: [name, status]
       properties:
         name:
           type: string
           description: Change directory name (NNN-slug)
           example: 009-web-admin-dashboard
         status:
           type: string
diff --git a/skillgrid-cli/internal/mnemonic/memory/governance.go b/skillgrid-cli/internal/mnemonic/memory/governance.go
index ba5da49..31b02a7 100644
--- a/skillgrid-cli/internal/mnemonic/memory/governance.go
+++ b/skillgrid-cli/internal/mnemonic/memory/governance.go
@@ -156,20 +156,65 @@ func (s *Service) AppendVersion(ctx context.Context, id int64, revision int) err
 	}
 	now := time.Now().UTC().Format(time.RFC3339)
 	if _, err := s.store.DB.ExecContext(ctx, `
 		INSERT INTO observation_versions (observation_id, revision, content, created_at)
 		VALUES (?, ?, ?, ?)`, id, revision, content, now); err != nil {
 		return fmt.Errorf("append version: %w", err)
 	}
 	return nil
 }
 
+// UpdateContent sets an observation's content in place through the same
+// governance path mem_update uses: the prior state is captured into
+// observation_versions (tagged with the revision being reached) before the
+// rewrite, and the update bumps revision_count and updated_at. It is the
+// content-only companion to SetStatus/SetVisibility, for callers that must
+// rewrite content as a governed, append-only change (the decision bridge's
+// answer route). An unknown id is an error; the observation is left unchanged.
+func (s *Service) UpdateContent(ctx context.Context, id int64, newContent string) error {
+	if s == nil || s.store == nil || s.store.DB == nil {
+		return errors.New("memory service not initialized")
+	}
+	if strings.TrimSpace(newContent) == "" {
+		return errors.New("content is required")
+	}
+	// Capture the pre-update state (tagged with the revision about to be
+	// reached) before the UPDATE below rewrites content and advances
+	// revision_count — the same append, never-overwrite rule as mem_update.
+	var curRev int
+	if err := s.store.DB.QueryRowContext(ctx, `
+		SELECT COALESCE(revision_count,0) FROM observations
+		WHERE id = ? AND project = ? AND deleted_at IS NULL`, id, s.projectID).Scan(&curRev); err != nil {
+		if errors.Is(err, sql.ErrNoRows) {
+			return fmt.Errorf("observation %d not found", id)
+		}
+		return fmt.Errorf("load revision: %w", err)
+	}
+	if appErr := s.AppendVersion(ctx, id, curRev+1); appErr != nil {
+		return fmt.Errorf("append version: %w", appErr)
+	}
+	now := time.Now().UTC().Format(time.RFC3339)
+	res, err := s.store.DB.ExecContext(ctx, `
+		UPDATE observations SET content = ?,
+			revision_count = COALESCE(revision_count,0) + 1,
+			updated_at = ?
+		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
+		newContent, now, id, s.projectID)
+	if err != nil {
+		return fmt.Errorf("update content: %w", err)
+	}
+	if n, _ := res.RowsAffected(); n == 0 {
+		return fmt.Errorf("observation %d not found", id)
+	}
+	return nil
+}
+
 // SetStatus sets an observation's lifecycle status explicitly. It is never
 // inferred from content: the caller decides. An unknown status is rejected.
 func (s *Service) SetStatus(ctx context.Context, id int64, status string) error {
 	if s == nil || s.store == nil || s.store.DB == nil {
 		return errors.New("memory service not initialized")
 	}
 	status = strings.TrimSpace(status)
 	if !IsValidStatus(status) {
 		return fmt.Errorf("invalid status %q (valid: %s|superseded|archived)", status, StatusActive)
 	}
diff --git a/skillgrid-ui/package-lock.json b/skillgrid-ui/package-lock.json
index 0fd2aae..026776b 100644
--- a/skillgrid-ui/package-lock.json
+++ b/skillgrid-ui/package-lock.json
@@ -22,42 +22,98 @@
         "mermaid": "^12.0.0",
         "react": "^19.1.0",
         "react-dom": "^19.1.0",
         "react-markdown": "^10.1.0",
         "rehype-sanitize": "^6.0.0",
         "remark-gfm": "^4.0.1",
         "sigma": "^3.0.3"
       },
       "devDependencies": {
         "@tailwindcss/vite": "^4.1.11",
+        "@testing-library/dom": "^10.4.2",
+        "@testing-library/react": "^16.3.3",
         "@types/react": "^19.1.8",
         "@types/react-dom": "^19.1.6",
         "@vitejs/plugin-react": "^4.6.0",
+        "jsdom": "^30.1.0",
         "tailwindcss": "^4.1.11",
         "typescript": "~5.8.3",
         "vite": "^6.3.5",
         "vitest": "^5.0.1"
       }
     },
     "node_modules/@antfu/install-pkg": {
       "version": "2.1.0",
       "resolved": "https://registry.npmjs.org/@antfu/install-pkg/-/install-pkg-2.1.0.tgz",
       "integrity": "sha512-sdg9NxU3zR4Mnawfbc/x6GB5Wf17WYud5qOuEuxXjaKpYpMkISSJEjItGebXJ2bQ4DIcly4NYH23mtkGJjvKUw==",
       "license": "MIT",
       "dependencies": {
         "package-manager-detector": "^1.8.0",
         "tinyexec": "^1.3.1"
       },
       "funding": {
         "url": "https://github.com/sponsors/antfu"
       }
     },
+    "node_modules/@asamuzakjp/css-color": {
+      "version": "7.0.1",
+      "resolved": "https://registry.npmjs.org/@asamuzakjp/css-color/-/css-color-7.0.1.tgz",
+      "integrity": "sha512-C9duntabagkBZ1LebM7FKmphR4Q1pBclLxVbZETQV0akkFjV0ooFxo8FlvAQyKj9F6l8rEnmidgcTlpyxFYizg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@csstools/css-calc": "^3.4.0",
+        "@csstools/css-color-parser": "^4.2.3",
+        "@csstools/css-parser-algorithms": "^4.0.0",
+        "@csstools/css-tokenizer": "^4.0.1",
+        "lru-cache": "^11.5.3"
+      },
+      "engines": {
+        "node": "^22.22.2 || ^24.15.0 || >=26.0.0"
+      }
+    },
+    "node_modules/@asamuzakjp/css-color/node_modules/lru-cache": {
+      "version": "11.5.3",
+      "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-11.5.3.tgz",
+      "integrity": "sha512-U4N8FgzmWxc8k1VH8Kr6lQg18U7Fjvby6wXHVRX/ZZ7IwWbRMgrRbP0Wrb5q5NVinryp4SQampHKdvtecItxUg==",
+      "dev": true,
+      "license": "BlueOak-1.0.0",
+      "engines": {
+        "node": "20 || >=22"
+      }
+    },
+    "node_modules/@asamuzakjp/dom-selector": {
+      "version": "9.2.1",
+      "resolved": "https://registry.npmjs.org/@asamuzakjp/dom-selector/-/dom-selector-9.2.1.tgz",
+      "integrity": "sha512-NT4s3yZLjovPpliRpTvdsdzyPjqRqiCZj9MxnarBihbaY5MbAG7DWAJcrLlhmC7xKTNCXsTELcqB8YsqpLnUFQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "bidi-js": "^1.1.0",
+        "css-tree": "^3.2.1",
+        "is-potential-custom-element-name": "^1.0.1",
+        "lru-cache": "^11.5.3"
+      },
+      "engines": {
+        "node": "^22.22.2 || ^24.15.0 || >=26.0.0"
+      }
+    },
+    "node_modules/@asamuzakjp/dom-selector/node_modules/lru-cache": {
+      "version": "11.5.3",
+      "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-11.5.3.tgz",
+      "integrity": "sha512-U4N8FgzmWxc8k1VH8Kr6lQg18U7Fjvby6wXHVRX/ZZ7IwWbRMgrRbP0Wrb5q5NVinryp4SQampHKdvtecItxUg==",
+      "dev": true,
+      "license": "BlueOak-1.0.0",
+      "engines": {
+        "node": "20 || >=22"
+      }
+    },
     "node_modules/@babel/code-frame": {
       "version": "7.29.7",
       "resolved": "https://registry.npmjs.org/@babel/code-frame/-/code-frame-7.29.7.tgz",
       "integrity": "sha512-Aup7aUOfpbAUg2ROOJN6Iw5f9DMBlzu0mIkm/malLQFN/YQgO48wCj0Kxa3sEHJvPVFg7siR+qRInwXd2qhQKw==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "@babel/helper-validator-identifier": "^7.29.7",
         "js-tokens": "^4.0.0",
         "picocolors": "^1.1.1"
@@ -278,20 +334,30 @@
       "dependencies": {
         "@babel/helper-plugin-utils": "^7.29.7"
       },
       "engines": {
         "node": ">=6.9.0"
       },
       "peerDependencies": {
         "@babel/core": "^7.0.0-0"
       }
     },
+    "node_modules/@babel/runtime": {
+      "version": "7.29.7",
+      "resolved": "https://registry.npmjs.org/@babel/runtime/-/runtime-7.29.7.tgz",
+      "integrity": "sha512-Nq8OhGWiZIZGV6hLHoyAKLLcJihP/xFeBMGJoUrxTX2psI8dCifzLhZISFb+VWS3wFMRDmCGw5R+dOySCqPLhw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6.9.0"
+      }
+    },
     "node_modules/@babel/template": {
       "version": "7.29.7",
       "resolved": "https://registry.npmjs.org/@babel/template/-/template-7.29.7.tgz",
       "integrity": "sha512-puq+Gf35oI24FeN11LkoUQFqv9uwNeWpxXZi/Ji3rRIoKAzKnxRaZ+Gkj0vKS9ZCiTESfng1N9LyOyXvo+m+Gg==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "@babel/code-frame": "^7.29.7",
         "@babel/parser": "^7.29.7",
         "@babel/types": "^7.29.7"
@@ -332,20 +398,33 @@
       "engines": {
         "node": ">=6.9.0"
       }
     },
     "node_modules/@braintree/sanitize-url": {
       "version": "7.1.2",
       "resolved": "https://registry.npmjs.org/@braintree/sanitize-url/-/sanitize-url-7.1.2.tgz",
       "integrity": "sha512-jigsZK+sMF/cuiB7sERuo9V7N9jx+dhmHHnQyDSVdpZwVutaBu7WvNYqMDLSgFgfB30n452TP3vjDAvFC973mA==",
       "license": "MIT"
     },
+    "node_modules/@bramus/specificity": {
+      "version": "2.4.2",
+      "resolved": "https://registry.npmjs.org/@bramus/specificity/-/specificity-2.4.2.tgz",
+      "integrity": "sha512-ctxtJ/eA+t+6q2++vj5j7FYX3nRu311q1wfYH3xjlLOsczhlhxAg2FWNUXhpGvAw3BWo1xBcvOV6/YLc2r5FJw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "css-tree": "^3.0.0"
+      },
+      "bin": {
+        "specificity": "bin/cli.js"
+      }
+    },
     "node_modules/@chevrotain/cst-dts-gen": {
       "version": "11.1.2",
       "resolved": "https://registry.npmjs.org/@chevrotain/cst-dts-gen/-/cst-dts-gen-11.1.2.tgz",
       "integrity": "sha512-XTsjvDVB5nDZBQB8o0o/0ozNelQtn2KrUVteIHSlPd2VAV2utEb6JzyCJaJ8tGxACR4RiBNWy5uYUHX2eji88Q==",
       "license": "Apache-2.0",
       "dependencies": {
         "@chevrotain/gast": "11.1.2",
         "@chevrotain/types": "11.1.2",
         "lodash-es": "4.17.23"
       }
@@ -371,20 +450,160 @@
       "resolved": "https://registry.npmjs.org/@chevrotain/types/-/types-11.1.2.tgz",
       "integrity": "sha512-U+HFai5+zmJCkK86QsaJtoITlboZHBqrVketcO2ROv865xfCMSFpELQoz1GkX5GzME8pTa+3kbKrZHQtI0gdbw==",
       "license": "Apache-2.0"
     },
     "node_modules/@chevrotain/utils": {
       "version": "11.1.2",
       "resolved": "https://registry.npmjs.org/@chevrotain/utils/-/utils-11.1.2.tgz",
       "integrity": "sha512-4mudFAQ6H+MqBTfqLmU7G1ZwRzCLfJEooL/fsF6rCX5eePMbGhoy5n4g+G4vlh2muDcsCTJtL+uKbOzWxs5LHA==",
       "license": "Apache-2.0"
     },
+    "node_modules/@csstools/color-helpers": {
+      "version": "6.1.1",
+      "resolved": "https://registry.npmjs.org/@csstools/color-helpers/-/color-helpers-6.1.1.tgz",
+      "integrity": "sha512-gLNsunvwf3mCi5u5o46/Z/JcJMnhbHSaZ69rkgPzNM3J4s8hWwpPUQB6/tt0EDFyCiWzxANlx+2LJwpYj4zS1w==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT-0",
+      "engines": {
+        "node": ">=20.19.0"
+      }
+    },
+    "node_modules/@csstools/css-calc": {
+      "version": "3.4.0",
+      "resolved": "https://registry.npmjs.org/@csstools/css-calc/-/css-calc-3.4.0.tgz",
+      "integrity": "sha512-XQKj5B7QiZcHiegCOCAzcAOJdhGgWOHbbu62h5e5mkHnn8lWcfiJhllkqWmxu5zWR9jucPHuo1iTB56P033hcg==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT",
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "peerDependencies": {
+        "@csstools/css-parser-algorithms": "^4.0.0",
+        "@csstools/css-tokenizer": "^4.0.0"
+      }
+    },
+    "node_modules/@csstools/css-color-parser": {
+      "version": "4.2.3",
+      "resolved": "https://registry.npmjs.org/@csstools/css-color-parser/-/css-color-parser-4.2.3.tgz",
+      "integrity": "sha512-y4LpL+lmpuyKDiEFq2PnZUVFdAjsoB/qQJod79yLNokXyW7jewi+/WJ69EfItj8A2unWtxXnGjw6LYXgXu5ZjA==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT",
+      "dependencies": {
+        "@csstools/color-helpers": "^6.1.1",
+        "@csstools/css-calc": "^3.4.0"
+      },
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "peerDependencies": {
+        "@csstools/css-parser-algorithms": "^4.0.0",
+        "@csstools/css-tokenizer": "^4.0.0"
+      }
+    },
+    "node_modules/@csstools/css-parser-algorithms": {
+      "version": "4.0.0",
+      "resolved": "https://registry.npmjs.org/@csstools/css-parser-algorithms/-/css-parser-algorithms-4.0.0.tgz",
+      "integrity": "sha512-+B87qS7fIG3L5h3qwJ/IFbjoVoOe/bpOdh9hAjXbvx0o8ImEmUsGXN0inFOnk2ChCFgqkkGFQ+TpM5rbhkKe4w==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT",
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "peerDependencies": {
+        "@csstools/css-tokenizer": "^4.0.0"
+      }
+    },
+    "node_modules/@csstools/css-syntax-patches-for-csstree": {
+      "version": "1.1.14",
+      "resolved": "https://registry.npmjs.org/@csstools/css-syntax-patches-for-csstree/-/css-syntax-patches-for-csstree-1.1.14.tgz",
+      "integrity": "sha512-HpbVXyrofRXpHpgkNIjU/3EWR4WJvOkO3emNK/L6X/mTJU7bGUI3AkkpoTNXznQLp0KRjLHELTGeKI5dIkI9JQ==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT-0",
+      "peerDependencies": {
+        "css-tree": "^3.2.1"
+      },
+      "peerDependenciesMeta": {
+        "css-tree": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/@csstools/css-tokenizer": {
+      "version": "4.0.1",
+      "resolved": "https://registry.npmjs.org/@csstools/css-tokenizer/-/css-tokenizer-4.0.1.tgz",
+      "integrity": "sha512-bPlN9S9O1A0euCpEWE4qnvB5YDuyYVsUTrxSgmAM1Is0j4tICHoVyOVAXfWMP/kS9ZrjvyIXWV2PmomiAXXqOw==",
+      "dev": true,
+      "funding": [
+        {
+          "type": "github",
+          "url": "https://github.com/sponsors/csstools"
+        },
+        {
+          "type": "opencollective",
+          "url": "https://opencollective.com/csstools"
+        }
+      ],
+      "license": "MIT",
+      "engines": {
+        "node": ">=20.19.0"
+      }
+    },
     "node_modules/@dnd-kit/accessibility": {
       "version": "3.1.1",
       "resolved": "https://registry.npmjs.org/@dnd-kit/accessibility/-/accessibility-3.1.1.tgz",
       "integrity": "sha512-2P+YgaXF+gRsIihwwY1gCsQSYnu9Zyj2py8kY5fFvUM1qm2WA2u639R6YNVfU4GWr+ZM5mqEsfHZZLoRONbemw==",
       "license": "MIT",
       "dependencies": {
         "tslib": "^2.0.0"
       },
       "peerDependencies": {
         "react": ">=16.8.0"
@@ -866,20 +1085,38 @@
       "dev": true,
       "license": "MIT",
       "optional": true,
       "os": [
         "win32"
       ],
       "engines": {
         "node": ">=18"
       }
     },
+    "node_modules/@exodus/bytes": {
+      "version": "1.15.1",
+      "resolved": "https://registry.npmjs.org/@exodus/bytes/-/bytes-1.15.1.tgz",
+      "integrity": "sha512-S6mL0yNB/Abt9Ei4tq8gDhcczc4S3+vQ4ra7vxnAf+YHC02srtqxKKZghx2Dq6p0e66THKwR6r8N6P95wEty7Q==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": "^20.19.0 || ^22.12.0 || >=24.0.0"
+      },
+      "peerDependencies": {
+        "@noble/hashes": "^1.8.0 || ^2.0.0"
+      },
+      "peerDependenciesMeta": {
+        "@noble/hashes": {
+          "optional": true
+        }
+      }
+    },
     "node_modules/@iconify/types": {
       "version": "2.0.0",
       "resolved": "https://registry.npmjs.org/@iconify/types/-/types-2.0.0.tgz",
       "integrity": "sha512-+wluvCrRhXrhyOmRDJ3q8mux9JkKy5SJ/v8ol2tu4FVjyYvtEzkc/3pK15ET6RKg4b4w4BmTk1+gsCUhf21Ykg==",
       "license": "MIT"
     },
     "node_modules/@iconify/utils": {
       "version": "3.1.7",
       "resolved": "https://registry.npmjs.org/@iconify/utils/-/utils-3.1.7.tgz",
       "integrity": "sha512-JZHlwdID+dy+lTgbYC8NEC4zeugqeYsc6jewvzb4c58kHauJn+X7rNwQjxz5p2qSjqaEeQoLkCIQ9v/H4PK0/w==",
@@ -1739,20 +1976,75 @@
     "node_modules/@tanstack/store": {
       "version": "0.11.1",
       "resolved": "https://registry.npmjs.org/@tanstack/store/-/store-0.11.1.tgz",
       "integrity": "sha512-mzTOBhypOuDJAy/D8n2MfUZ1HFkXnmSETviRyhqEC8LUE7/IZQExOTxMANj3KjTofYTkFNpBY67qaVrT41YccA==",
       "license": "MIT",
       "funding": {
         "type": "github",
         "url": "https://github.com/sponsors/tannerlinsley"
       }
     },
+    "node_modules/@testing-library/dom": {
+      "version": "10.4.2",
+      "resolved": "https://registry.npmjs.org/@testing-library/dom/-/dom-10.4.2.tgz",
+      "integrity": "sha512-yzr2S9HyAIdhz2/6qHgbs665Q7PKVcDF05vsOlHPxG1mo36gKVesdYVeDLnXgfjJ03CrKRk08knc6+E/9m8v2Q==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/code-frame": "^7.10.4",
+        "@babel/runtime": "^7.12.5",
+        "@types/aria-query": "^5.0.1",
+        "aria-query": "5.3.0",
+        "dom-accessibility-api": "^0.5.9",
+        "lz-string": "^1.5.0",
+        "picocolors": "1.1.1",
+        "pretty-format": "^27.0.2"
+      },
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/@testing-library/react": {
+      "version": "16.3.3",
+      "resolved": "https://registry.npmjs.org/@testing-library/react/-/react-16.3.3.tgz",
+      "integrity": "sha512-Uo193NgQbPMz6lrrhtRQQFcMC6Re/ELLFbbuVL30WDlZxlpZf9/lMHTAVxPRLw1q1iu9OJmR1c2BLiENRstdBg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@babel/runtime": "^7.12.5"
+      },
+      "engines": {
+        "node": ">=18"
+      },
+      "peerDependencies": {
+        "@testing-library/dom": "^10.0.0",
+        "@types/react": "^18.0.0 || ^19.0.0",
+        "@types/react-dom": "^18.0.0 || ^19.0.0",
+        "react": "^18.0.0 || ^19.0.0",
+        "react-dom": "^18.0.0 || ^19.0.0"
+      },
+      "peerDependenciesMeta": {
+        "@types/react": {
+          "optional": true
+        },
+        "@types/react-dom": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/@types/aria-query": {
+      "version": "5.0.4",
+      "resolved": "https://registry.npmjs.org/@types/aria-query/-/aria-query-5.0.4.tgz",
+      "integrity": "sha512-rfT93uj5s0PRL7EzccGMs3brplhcrghnDoV26NqKhCAS1hVo+WdNsPvE/yb6ilfr5hi2MEk6d5EWJTKdxg8jVw==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/@types/babel__core": {
       "version": "7.20.5",
       "resolved": "https://registry.npmjs.org/@types/babel__core/-/babel__core-7.20.5.tgz",
       "integrity": "sha512-qoQprZvz5wQFJwMDqeseRXWv3rqMvhgpbXFfVyWhbx9X47POIA6i/+dXefEmZKoAgOaTdaIgNSMqMIU61yRyzA==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "@babel/parser": "^7.20.7",
         "@babel/types": "^7.20.7",
         "@types/babel__generator": "*",
@@ -2234,20 +2526,53 @@
     "node_modules/@vitest/spy": {
       "version": "5.0.1",
       "resolved": "https://registry.npmjs.org/@vitest/spy/-/spy-5.0.1.tgz",
       "integrity": "sha512-rbto/mF/SGERxEgYOek7Xm6B9b+y+mVoo+f4b2LymYO8zM1b7uB5nHuhVMTP2hxdzgxvGiZYGxGIaMvL5y180Q==",
       "dev": true,
       "license": "MIT",
       "funding": {
         "url": "https://opencollective.com/vitest"
       }
     },
+    "node_modules/ansi-regex": {
+      "version": "5.0.1",
+      "resolved": "https://registry.npmjs.org/ansi-regex/-/ansi-regex-5.0.1.tgz",
+      "integrity": "sha512-quJQXlTSUGL2LH9SUXo8VwsY4soanhgo6LNSm84E1LBcE8s3O0wpdiRzyR9z/ZZJMlMWv37qOOb9pdJlMUEKFQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=8"
+      }
+    },
+    "node_modules/ansi-styles": {
+      "version": "5.2.0",
+      "resolved": "https://registry.npmjs.org/ansi-styles/-/ansi-styles-5.2.0.tgz",
+      "integrity": "sha512-Cxwpt2SfTzTtXcfOlzGEee8O+c+MmUgGrNiBcXnuWxuFJHe6a5Hz7qwhwe5OgaSYI0IJvkLqWX1ASG+cJOkEiA==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=10"
+      },
+      "funding": {
+        "url": "https://github.com/chalk/ansi-styles?sponsor=1"
+      }
+    },
+    "node_modules/aria-query": {
+      "version": "5.3.0",
+      "resolved": "https://registry.npmjs.org/aria-query/-/aria-query-5.3.0.tgz",
+      "integrity": "sha512-b0P0sZPKtyu8HkeRAfCq0IfURZK+SuwMjY1UXGBU27wpAiTwQAIlq56IbIO+ytk/JjS1fMR14ee5WBBfKi5J6A==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "dependencies": {
+        "dequal": "^2.0.3"
+      }
+    },
     "node_modules/assertion-error": {
       "version": "2.0.1",
       "resolved": "https://registry.npmjs.org/assertion-error/-/assertion-error-2.0.1.tgz",
       "integrity": "sha512-Izi8RQcffqCeNVgFigKli1ssklIbpHnCYc6AknXGYoB6grJqyeby7jv12JUQgmTAnIDnbck1uxksT4dzN3PWBA==",
       "dev": true,
       "license": "MIT",
       "engines": {
         "node": ">=12"
       }
     },
@@ -2267,20 +2592,30 @@
       "integrity": "sha512-hYrgxie335U08WqICoGqKRzV1HFXv6zdxwJE4ekCb80CM9a0SVVsN4QPwT67RraRo+9h8IATk6uxHJw7QSkdOg==",
       "dev": true,
       "license": "Apache-2.0",
       "bin": {
         "baseline-browser-mapping": "dist/cli.cjs"
       },
       "engines": {
         "node": ">=6.0.0"
       }
     },
+    "node_modules/bidi-js": {
+      "version": "1.1.0",
+      "resolved": "https://registry.npmjs.org/bidi-js/-/bidi-js-1.1.0.tgz",
+      "integrity": "sha512-fX1Onk0tdVPC7obPWB5EbJ1z7NVhLq4m2xZLq2YXBkxzMXIGRpNMU88n0EPgWseKl12J7zXs7qrDxPK4sRs2fg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "require-from-string": "^2.0.2"
+      }
+    },
     "node_modules/browserslist": {
       "version": "4.29.0",
       "resolved": "https://registry.npmjs.org/browserslist/-/browserslist-4.29.0.tgz",
       "integrity": "sha512-3GSvyjvDI4Dur1Meg2BekJquu5uF+9R9a1+5M1Mde192eZoXbeXjzgOsgqPS2V8D5wrrip0gR5Hf/GhWQ9ZzaA==",
       "dev": true,
       "funding": [
         {
           "type": "opencollective",
           "url": "https://opencollective.com/browserslist"
         },
@@ -2437,20 +2772,34 @@
     },
     "node_modules/cose-base": {
       "version": "1.0.3",
       "resolved": "https://registry.npmjs.org/cose-base/-/cose-base-1.0.3.tgz",
       "integrity": "sha512-s9whTXInMSgAp/NVXVNuVxVKzGH2qck3aQlVHxDCdAEPgtMKwc4Wq6/QKhgdEdgbLSi9rBTAcPoRa6JpiG4ksg==",
       "license": "MIT",
       "dependencies": {
         "layout-base": "^1.0.0"
       }
     },
+    "node_modules/css-tree": {
+      "version": "3.2.1",
+      "resolved": "https://registry.npmjs.org/css-tree/-/css-tree-3.2.1.tgz",
+      "integrity": "sha512-X7sjQzceUhu1u7Y/ylrRZFU2FS6LRiFVp6rKLPg23y3x3c3DOKAwuXGDp+PAGjh6CSnCjYeAul8pcT8bAl+lSA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "mdn-data": "2.27.1",
+        "source-map-js": "^1.2.1"
+      },
+      "engines": {
+        "node": "^10 || ^12.20.0 || ^14.13.0 || >=15.0.0"
+      }
+    },
     "node_modules/csstype": {
       "version": "3.2.3",
       "resolved": "https://registry.npmjs.org/csstype/-/csstype-3.2.3.tgz",
       "integrity": "sha512-z1HGKcYy2xA8AGQfwrn0PAy+PB7X/GSj3UVJW9qKyn43xWa+gl5nXmU4qqLMRzWVLFC8KusUX8T/0kCiOYpAIQ==",
       "license": "MIT"
     },
     "node_modules/cytoscape": {
       "version": "3.34.3",
       "resolved": "https://registry.npmjs.org/cytoscape/-/cytoscape-3.34.3.tgz",
       "integrity": "sha512-yfYGhRcGAntq6YBD583j4n0Eg3jIxvWmZtz/5uz9UYkeIStSlMxuUja+ec5j3iBD8nv1rwaOAYMW09tBdkSeaQ==",
@@ -2942,20 +3291,49 @@
     "node_modules/dagre-d3-es": {
       "version": "7.0.14",
       "resolved": "https://registry.npmjs.org/dagre-d3-es/-/dagre-d3-es-7.0.14.tgz",
       "integrity": "sha512-P4rFMVq9ESWqmOgK+dlXvOtLwYg0i7u0HBGJER0LZDJT2VHIPAMZ/riPxqJceWMStH5+E61QxFra9kIS3AqdMg==",
       "license": "MIT",
       "dependencies": {
         "d3": "^7.9.0",
         "lodash-es": "^4.17.21"
       }
     },
+    "node_modules/data-urls": {
+      "version": "7.0.0",
+      "resolved": "https://registry.npmjs.org/data-urls/-/data-urls-7.0.0.tgz",
+      "integrity": "sha512-23XHcCF+coGYevirZceTVD7NdJOqVn+49IHyxgszm+JIiHLoB2TkmPtsYkNWT1pvRSGkc35L6NHs0yHkN2SumA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "whatwg-mimetype": "^5.0.0",
+        "whatwg-url": "^16.0.0"
+      },
+      "engines": {
+        "node": "^20.19.0 || ^22.12.0 || >=24.0.0"
+      }
+    },
+    "node_modules/data-urls/node_modules/whatwg-url": {
+      "version": "16.0.1",
+      "resolved": "https://registry.npmjs.org/whatwg-url/-/whatwg-url-16.0.1.tgz",
+      "integrity": "sha512-1to4zXBxmXHV3IiSSEInrreIlu02vUOvrhxJJH5vcxYTBDAx51cqZiKdyTxlecdKNSjj8EcxGBxNf6Vg+945gw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@exodus/bytes": "^1.11.0",
+        "tr46": "^6.0.0",
+        "webidl-conversions": "^8.0.1"
+      },
+      "engines": {
+        "node": "^20.19.0 || ^22.12.0 || >=24.0.0"
+      }
+    },
     "node_modules/dayjs": {
       "version": "1.11.23",
       "resolved": "https://registry.npmjs.org/dayjs/-/dayjs-1.11.23.tgz",
       "integrity": "sha512-QDTCU0M0MxR3hQfnlDJfwekQiaanm1ubOD231u73WBckQ/fsamwRLiE2GBz6D3a/xF1NgfiDLJjXBa1hYOYTtQ==",
       "license": "MIT"
     },
     "node_modules/debug": {
       "version": "4.4.3",
       "resolved": "https://registry.npmjs.org/debug/-/debug-4.4.3.tgz",
       "integrity": "sha512-RGwwWnwQvkVfavKVt22FGLw+xYSdzARwm0ru6DhTVA3umU5hZc28V3kO4stgYryrTlLpuvgI9GiijltAjNbcqA==",
@@ -2965,20 +3343,27 @@
       },
       "engines": {
         "node": ">=6.0"
       },
       "peerDependenciesMeta": {
         "supports-color": {
           "optional": true
         }
       }
     },
+    "node_modules/decimal.js": {
+      "version": "10.6.0",
+      "resolved": "https://registry.npmjs.org/decimal.js/-/decimal.js-10.6.0.tgz",
+      "integrity": "sha512-YpgQiITW3JXGntzdUmyUR1V812Hn8T1YVXhCu+wO3OpS4eU9l4YdD3qjyiKdV6mvV29zapkMeD390UVEf2lkUg==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/decode-named-character-reference": {
       "version": "1.3.0",
       "resolved": "https://registry.npmjs.org/decode-named-character-reference/-/decode-named-character-reference-1.3.0.tgz",
       "integrity": "sha512-GtpQYB283KrPp6nRw50q3U9/VfOutZOe103qlN7BPP6Ad27xYnOIWv4lPzo8HCAL+mMZofJ9KEy30fq6MfaK6Q==",
       "license": "MIT",
       "dependencies": {
         "character-entities": "^2.0.0"
       },
       "funding": {
         "type": "github",
@@ -3019,20 +3404,27 @@
       "integrity": "sha512-RWmIqhcFf1lRYBvNmr7qTNuyCt/7/ns2jbpp1+PalgE/rDQcBT0fioSMUpJ93irlUhC5hrg4cYqe6U+0ImW0rA==",
       "license": "MIT",
       "dependencies": {
         "dequal": "^2.0.0"
       },
       "funding": {
         "type": "github",
         "url": "https://github.com/sponsors/wooorm"
       }
     },
+    "node_modules/dom-accessibility-api": {
+      "version": "0.5.16",
+      "resolved": "https://registry.npmjs.org/dom-accessibility-api/-/dom-accessibility-api-0.5.16.tgz",
+      "integrity": "sha512-X7BJ2yElsnOJ30pZF4uIIDfBEVgF4XEBxL9Bxhy6dnrm5hkzqmsWHGTiHqRiITNhMyFLyAiWndIJP7Z1NTteDg==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/dompurify": {
       "version": "3.4.15",
       "resolved": "https://registry.npmjs.org/dompurify/-/dompurify-3.4.15.tgz",
       "integrity": "sha512-EUBjM+B+lkDE41iE82DDSCfkoPGfXx8IxFxPMjNzm/Uk4xDet77rTN9wqlxlVg71kK7XGuUMv6wUxJUwwv+Xyw==",
       "license": "(MPL-2.0 OR Apache-2.0)",
       "optionalDependencies": {
         "@types/trusted-types": "^2.0.7"
       }
     },
     "node_modules/electron-to-chromium": {
@@ -3055,20 +3447,33 @@
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "graceful-fs": "^4.2.4",
         "tapable": "^2.3.3"
       },
       "engines": {
         "node": ">=10.13.0"
       }
     },
+    "node_modules/entities": {
+      "version": "8.1.0",
+      "resolved": "https://registry.npmjs.org/entities/-/entities-8.1.0.tgz",
+      "integrity": "sha512-kxL7msIffSuh9aaFAMD7rxAIuTRMAHMeBtgHW2yUdWw732ZNh4MehkF2gdjvtdmikkaIP9bFDDJOPlsvm7avrA==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=20.19.0"
+      },
+      "funding": {
+        "url": "https://github.com/fb55/entities?sponsor=1"
+      }
+    },
     "node_modules/es-module-lexer": {
       "version": "2.3.2",
       "resolved": "https://registry.npmjs.org/es-module-lexer/-/es-module-lexer-2.3.2.tgz",
       "integrity": "sha512-poHGpORABojJJucnV9KbOavETW8lBVnphkW77ER5/BQ5Fz7oXSoCNek7IH3vR5nRjdsEz926ibFYX8KtLQmdyw==",
       "dev": true,
       "license": "MIT"
     },
     "node_modules/es-toolkit": {
       "version": "1.52.0",
       "resolved": "https://registry.npmjs.org/es-toolkit/-/es-toolkit-1.52.0.tgz",
@@ -3361,20 +3766,33 @@
       "integrity": "sha512-88JUN06ipLwsnv+dVn+OIYOvAuvBMy/Qoi6O7mQHxdPXpjy+Cd6xRkWwux7DKO+4sYILtLBRIKgsdpS2gQc7qw==",
       "license": "MIT",
       "dependencies": {
         "@types/hast": "^3.0.0"
       },
       "funding": {
         "type": "opencollective",
         "url": "https://opencollective.com/unified"
       }
     },
+    "node_modules/html-encoding-sniffer": {
+      "version": "6.0.0",
+      "resolved": "https://registry.npmjs.org/html-encoding-sniffer/-/html-encoding-sniffer-6.0.0.tgz",
+      "integrity": "sha512-CV9TW3Y3f8/wT0BRFc1/KAVQ3TUHiXmaAb6VW9vtiMFf7SLoMd1PdAc4W3KFOFETBJUb90KatHqlsZMWV+R9Gg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@exodus/bytes": "^1.6.0"
+      },
+      "engines": {
+        "node": "^20.19.0 || ^22.12.0 || >=24.0.0"
+      }
+    },
     "node_modules/html-url-attributes": {
       "version": "3.0.1",
       "resolved": "https://registry.npmjs.org/html-url-attributes/-/html-url-attributes-3.0.1.tgz",
       "integrity": "sha512-ol6UPyBWqsrO6EJySPz2O7ZSr856WDrEzM5zMqp+FJJLGMW35cLYmmZnl0vztAZxRUoNZJFTCohfjuIJ8I4QBQ==",
       "license": "MIT",
       "funding": {
         "type": "opencollective",
         "url": "https://opencollective.com/unified"
       }
     },
@@ -3464,20 +3882,27 @@
       "resolved": "https://registry.npmjs.org/is-plain-obj/-/is-plain-obj-4.1.0.tgz",
       "integrity": "sha512-+Pgi+vMuUNkJyExiMBt5IlFoMyKnr5zhJ4Uspz58WOhBF5QoIZkFyNHIbBAtHwzVAgk5RtndVNsDRN61/mmDqg==",
       "license": "MIT",
       "engines": {
         "node": ">=12"
       },
       "funding": {
         "url": "https://github.com/sponsors/sindresorhus"
       }
     },
+    "node_modules/is-potential-custom-element-name": {
+      "version": "1.0.1",
+      "resolved": "https://registry.npmjs.org/is-potential-custom-element-name/-/is-potential-custom-element-name-1.0.1.tgz",
+      "integrity": "sha512-bCYeRA2rVibKZd+s2625gGnGF/t7DSqDs4dP7CrLA1m7jKWz6pps0LpYLJN8Q64HtmPKJ1hrN3nzPNKFEKOUiQ==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/isbot": {
       "version": "5.2.2",
       "resolved": "https://registry.npmjs.org/isbot/-/isbot-5.2.2.tgz",
       "integrity": "sha512-iQcBXcd+Rv/pkubRyGh2utW2j1oPG5hZY6TUhVPpqK4G+o3IbxpJNx04hgksjc/N7GK5pEorUxDeg31cFgEk/w==",
       "license": "Unlicense",
       "engines": {
         "node": ">=18"
       }
     },
     "node_modules/jiti": {
@@ -3490,20 +3915,70 @@
         "jiti": "lib/jiti-cli.mjs"
       }
     },
     "node_modules/js-tokens": {
       "version": "4.0.0",
       "resolved": "https://registry.npmjs.org/js-tokens/-/js-tokens-4.0.0.tgz",
       "integrity": "sha512-RdJUflcE3cUzKiMqQgsCu06FPu9UdIJO0beYbPhHN4k6apgJtifcoCtT9bcxOpYBtpD2kCM6Sbzg4CausW/PKQ==",
       "dev": true,
       "license": "MIT"
     },
+    "node_modules/jsdom": {
+      "version": "30.1.0",
+      "resolved": "https://registry.npmjs.org/jsdom/-/jsdom-30.1.0.tgz",
+      "integrity": "sha512-h/Q+H+KmlGf49ITw3TrykSVbg04S75S2O7ZtJc9gMGOWBrQRZmx5AJIKLtJ01e6Kszl+/tpFthCVqxNifxhoqg==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@asamuzakjp/css-color": "^7.0.0",
+        "@asamuzakjp/dom-selector": "^9.1.2",
+        "@bramus/specificity": "^2.4.2",
+        "@csstools/css-syntax-patches-for-csstree": "^1.1.13",
+        "@exodus/bytes": "^1.15.1",
+        "css-tree": "^3.2.1",
+        "data-urls": "^7.0.0",
+        "decimal.js": "^10.6.0",
+        "html-encoding-sniffer": "^6.0.0",
+        "is-potential-custom-element-name": "^1.0.1",
+        "lru-cache": "^11.5.2",
+        "parse5": "^8.0.1",
+        "saxes": "^6.0.0",
+        "tough-cookie": "^6.0.2",
+        "undici": "^8.10.2",
+        "w3c-xmlserializer": "^5.0.0",
+        "webidl-conversions": "^8.0.1",
+        "whatwg-mimetype": "^5.0.0",
+        "whatwg-url": "^17.1.1",
+        "xml-name-validator": "^5.0.0"
+      },
+      "engines": {
+        "node": "^22.22.2 || ^24.15.0 || >=26.0.0"
+      },
+      "peerDependencies": {
+        "canvas": "^3.2.3"
+      },
+      "peerDependenciesMeta": {
+        "canvas": {
+          "optional": true
+        }
+      }
+    },
+    "node_modules/jsdom/node_modules/lru-cache": {
+      "version": "11.5.3",
+      "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-11.5.3.tgz",
+      "integrity": "sha512-U4N8FgzmWxc8k1VH8Kr6lQg18U7Fjvby6wXHVRX/ZZ7IwWbRMgrRbP0Wrb5q5NVinryp4SQampHKdvtecItxUg==",
+      "dev": true,
+      "license": "BlueOak-1.0.0",
+      "engines": {
+        "node": "20 || >=22"
+      }
+    },
     "node_modules/jsesc": {
       "version": "3.1.0",
       "resolved": "https://registry.npmjs.org/jsesc/-/jsesc-3.1.0.tgz",
       "integrity": "sha512-/sM3dO2FOzXjKQhJuo0Q173wf2KOo8t4I8vHy6lF9poUp7bKT0/NHE8fPX23PwfhnykfqnC2xRxOnVw5XuGIaA==",
       "dev": true,
       "license": "MIT",
       "bin": {
         "jsesc": "bin/jsesc"
       },
       "engines": {
@@ -3851,20 +4326,30 @@
     "node_modules/lru-cache": {
       "version": "5.1.1",
       "resolved": "https://registry.npmjs.org/lru-cache/-/lru-cache-5.1.1.tgz",
       "integrity": "sha512-KpNARQA3Iwv+jTA0utUVVbrh+Jlrr1Fv0e56GGzAFOXN7dk/FviaDW8LHmK52DlcH4WP2n6gI8vN1aesBFgo9w==",
       "dev": true,
       "license": "ISC",
       "dependencies": {
         "yallist": "^3.0.2"
       }
     },
+    "node_modules/lz-string": {
+      "version": "1.5.0",
+      "resolved": "https://registry.npmjs.org/lz-string/-/lz-string-1.5.0.tgz",
+      "integrity": "sha512-h5bgJWpxJNswbU7qCrV0tIKQCaS3blPDrqKWx+QxzuzL1zGUzij9XCWLrSLsJPu5t+eWA/ycetzYAO5IOMcWAQ==",
+      "dev": true,
+      "license": "MIT",
+      "bin": {
+        "lz-string": "bin/bin.js"
+      }
+    },
     "node_modules/magic-string": {
       "version": "0.30.21",
       "resolved": "https://registry.npmjs.org/magic-string/-/magic-string-0.30.21.tgz",
       "integrity": "sha512-vd2F4YUyEXKGcLHoq+TEyCjxueSeHnFxyyjNp80yg0XV4vUhnDer/lvvlqM/arB5bXQN5K2/3oinyCRyx8T2CQ==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "@jridgewell/sourcemap-codec": "^1.5.5"
       }
     },
@@ -4153,20 +4638,27 @@
       "integrity": "sha512-0H44vDimn51F0YwvxSJSm0eCDOJTRlmN0R1yBh4HLj9wiV1Dn0QoXGbvFAWj2hSItVTlCmBF1hqKlIyUBVFLPg==",
       "license": "MIT",
       "dependencies": {
         "@types/mdast": "^4.0.0"
       },
       "funding": {
         "type": "opencollective",
         "url": "https://opencollective.com/unified"
       }
     },
+    "node_modules/mdn-data": {
+      "version": "2.27.1",
+      "resolved": "https://registry.npmjs.org/mdn-data/-/mdn-data-2.27.1.tgz",
+      "integrity": "sha512-9Yubnt3e8A0OKwxYSXyhLymGW4sCufcLG6VdiDdUGVkPhpqLxlvP5vl1983gQjJl3tqbrM731mjaZaP68AgosQ==",
+      "dev": true,
+      "license": "CC0-1.0"
+    },
     "node_modules/mermaid": {
       "version": "12.0.0",
       "resolved": "https://registry.npmjs.org/mermaid/-/mermaid-12.0.0.tgz",
       "integrity": "sha512-/wQXC9iBxoGV8p3erbvaXs9h77VyLDBH6GdayVjj3hEcSQhFU4N1WUhUppotCEqlIxI2pRMwjwBSwTB1MfZBgQ==",
       "license": "MIT",
       "dependencies": {
         "@braintree/sanitize-url": "^7.1.2",
         "@iconify/utils": "^3.0.2",
         "@mermaid-js/parser": "^2.0.0",
         "@types/d3": "^7.4.3",
@@ -4854,20 +5346,33 @@
         "type": "github",
         "url": "https://github.com/sponsors/wooorm"
       }
     },
     "node_modules/parse-entities/node_modules/@types/unist": {
       "version": "2.0.11",
       "resolved": "https://registry.npmjs.org/@types/unist/-/unist-2.0.11.tgz",
       "integrity": "sha512-CmBKiL6NNo/OqgmMn95Fk9Whlp2mtvIv+KNpQKN2F4SjvrEesubTRWGYSg+BnWZOnlCaSTU1sMpsBOzgbYhnsA==",
       "license": "MIT"
     },
+    "node_modules/parse5": {
+      "version": "8.0.1",
+      "resolved": "https://registry.npmjs.org/parse5/-/parse5-8.0.1.tgz",
+      "integrity": "sha512-z1e/HMG90obSGeidlli3hj7cbocou0/wa5HacvI3ASx34PecNjNQeaHNo5WIZpWofN9kgkqV1q5YvXe3F0FoPw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "entities": "^8.0.0"
+      },
+      "funding": {
+        "url": "https://github.com/inikulin/parse5?sponsor=1"
+      }
+    },
     "node_modules/path-data-parser": {
       "version": "0.1.0",
       "resolved": "https://registry.npmjs.org/path-data-parser/-/path-data-parser-0.1.0.tgz",
       "integrity": "sha512-NOnmBpt5Y2RWbuv0LMzsayp3lVylAHLPUTut412ZA3l+C4uw4ZVkQbjShYCQ8TCpUMdPapr4YjUqLYD6v68j+w==",
       "license": "MIT"
     },
     "node_modules/picocolors": {
       "version": "1.1.1",
       "resolved": "https://registry.npmjs.org/picocolors/-/picocolors-1.1.1.tgz",
       "integrity": "sha512-xceH2snhtb5M9liqDsmEw56le376mTZkEX/jEb/RxNFyegNul7eNslCXP9FDj/Lcu0X8KEyMceP2ntpaHrDEVA==",
@@ -4925,30 +5430,55 @@
       "license": "MIT",
       "dependencies": {
         "nanoid": "^3.3.18",
         "picocolors": "^1.1.1",
         "source-map-js": "^1.2.1"
       },
       "engines": {
         "node": "^10 || ^12 || >=14"
       }
     },
+    "node_modules/pretty-format": {
+      "version": "27.5.1",
+      "resolved": "https://registry.npmjs.org/pretty-format/-/pretty-format-27.5.1.tgz",
+      "integrity": "sha512-Qb1gy5OrP5+zDf2Bvnzdl3jsTf1qXVMazbvCoKhtKqVs4/YK4ozX4gKQJJVyNe+cajNPn0KoC0MC3FUmaHWEmQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "ansi-regex": "^5.0.1",
+        "ansi-styles": "^5.0.0",
+        "react-is": "^17.0.1"
+      },
+      "engines": {
+        "node": "^10.13.0 || ^12.13.0 || ^14.15.0 || >=15.0.0"
+      }
+    },
     "node_modules/property-information": {
       "version": "7.2.0",
       "resolved": "https://registry.npmjs.org/property-information/-/property-information-7.2.0.tgz",
       "integrity": "sha512-IAtzIB6sUiWaJYrX9smp3V46pBGbBeLFRGdh25kg1334VcBlD8HzhPeNIWQH9zhGmo2itIe25EHt9dQP7G5hmg==",
       "license": "MIT",
       "funding": {
         "type": "github",
         "url": "https://github.com/sponsors/wooorm"
       }
     },
+    "node_modules/punycode": {
+      "version": "2.3.1",
+      "resolved": "https://registry.npmjs.org/punycode/-/punycode-2.3.1.tgz",
+      "integrity": "sha512-vYt7UD1U9Wg6138shLtLOvdAu+8DsC/ilFtEVHcH+wydcSpNE20AfSOduf6MkRFahL5FY7X1oU7nKVZFtfq8Fg==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=6"
+      }
+    },
     "node_modules/react": {
       "version": "19.3.0",
       "resolved": "https://registry.npmjs.org/react/-/react-19.3.0.tgz",
       "integrity": "sha512-E8LUcbtBWt20bbl2YoHfx4ZDBdxVTfOKtCZn9cDSJ4l6/nuoApcpIBcj47t2wZoVX8g2ZHuMHbiShgCR1T5Sog==",
       "license": "MIT",
       "engines": {
         "node": ">=0.10.0"
       }
     },
     "node_modules/react-dom": {
@@ -4956,20 +5486,27 @@
       "resolved": "https://registry.npmjs.org/react-dom/-/react-dom-19.3.0.tgz",
       "integrity": "sha512-JDk8dgif51OjFoDE70+OT9ICyYr+69HlmihNwp1+Nsfbna3t5sIiCa9ZJktDmQ4/1b/rn26hIAR2uYXDMr5r0Q==",
       "license": "MIT",
       "dependencies": {
         "scheduler": "^0.28.0"
       },
       "peerDependencies": {
         "react": "^19.3.0"
       }
     },
+    "node_modules/react-is": {
+      "version": "17.0.2",
+      "resolved": "https://registry.npmjs.org/react-is/-/react-is-17.0.2.tgz",
+      "integrity": "sha512-w2GsyukL62IJnlaff/nRegPQR94C/XXamvMWmSHRJ4y7Ts/4ocGRmTHvOs8PSE6pB3dWOrD/nueuU5sduBsQ4w==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/react-markdown": {
       "version": "10.1.0",
       "resolved": "https://registry.npmjs.org/react-markdown/-/react-markdown-10.1.0.tgz",
       "integrity": "sha512-qKxVopLT/TyA6BX3Ue5NwabOsAzm0Q7kAPwq6L+wWDwisYs7R8vZ0nRXqq6rkueboxpkjvLGU9fWifiX/ZZFxQ==",
       "license": "MIT",
       "dependencies": {
         "@types/hast": "^3.0.0",
         "@types/mdast": "^4.0.0",
         "devlop": "^1.0.0",
         "hast-util-to-jsx-runtime": "^2.0.0",
@@ -5073,20 +5610,30 @@
       "dependencies": {
         "@types/mdast": "^4.0.0",
         "mdast-util-to-markdown": "^2.0.0",
         "unified": "^11.0.0"
       },
       "funding": {
         "type": "opencollective",
         "url": "https://opencollective.com/unified"
       }
     },
+    "node_modules/require-from-string": {
+      "version": "2.0.2",
+      "resolved": "https://registry.npmjs.org/require-from-string/-/require-from-string-2.0.2.tgz",
+      "integrity": "sha512-Xf0nWe6RseziFMu+Ap9biiUbmplq6S9/p+7w7YXP/JBHhrUDDUhwa+vANyubuqfZWTveU//DYVGsDG7RKL/vEw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=0.10.0"
+      }
+    },
     "node_modules/robust-predicates": {
       "version": "3.0.3",
       "resolved": "https://registry.npmjs.org/robust-predicates/-/robust-predicates-3.0.3.tgz",
       "integrity": "sha512-NS3levdsRIUOmiJ8FZWCP7LG3QpJyrs/TE0Zpf1yvZu8cAJJ6QMW92H1c7kWpdIHo8RvmLxN/o2JXTKHp74lUA==",
       "license": "Unlicense"
     },
     "node_modules/rollup": {
       "version": "4.63.3",
       "resolved": "https://registry.npmjs.org/rollup/-/rollup-4.63.3.tgz",
       "integrity": "sha512-1i2XreiAoMMXuPGD6Msj2xWrMMkHojNRKivInxGQcg7/1KuPuYlfUutLyh4drnOxUTHX9cHI4wFoat8D/NKaBw==",
@@ -5149,20 +5696,33 @@
       "resolved": "https://registry.npmjs.org/rw/-/rw-1.3.3.tgz",
       "integrity": "sha512-PdhdWy89SiZogBLaw42zdeqtRJ//zFd2PgQavcICDUgJT5oW10QCRKbJ6bg4r0/UY2M6BWd5tkxuGFRvCkgfHQ==",
       "license": "BSD-3-Clause"
     },
     "node_modules/safer-buffer": {
       "version": "2.1.2",
       "resolved": "https://registry.npmjs.org/safer-buffer/-/safer-buffer-2.1.2.tgz",
       "integrity": "sha512-YZo3K82SD7Riyi0E1EQPojLz7kpepnSQI9IyPbHHg1XXXevb5dJI7tpyN2ADxGcQbHG7vcyRHk0cbwqcQriUtg==",
       "license": "MIT"
     },
+    "node_modules/saxes": {
+      "version": "6.0.0",
+      "resolved": "https://registry.npmjs.org/saxes/-/saxes-6.0.0.tgz",
+      "integrity": "sha512-xAg7SOnEhrm5zI3puOOKyy1OMcMlIJZYNJY7xLBwSze0UjhPLnWfj2GF2EpT0jmzaJKIWKHLsaSSajf35bcYnA==",
+      "dev": true,
+      "license": "ISC",
+      "dependencies": {
+        "xmlchars": "^2.2.0"
+      },
+      "engines": {
+        "node": ">=v12.22.7"
+      }
+    },
     "node_modules/scheduler": {
       "version": "0.28.0",
       "resolved": "https://registry.npmjs.org/scheduler/-/scheduler-0.28.0.tgz",
       "integrity": "sha512-juorfCmIkIw8tT+p5BXSm6PJjQF/ycEYmKyzURCIt/RaZIhL+PulbQ9Yu2z1HdOJDdqDTlxA1+xKBmHXJsczAw==",
       "license": "MIT"
     },
     "node_modules/semver": {
       "version": "6.3.1",
       "resolved": "https://registry.npmjs.org/semver/-/semver-6.3.1.tgz",
       "integrity": "sha512-BR7VvDCVHO+q2xBEWskxS6DJE1qRnb7DxzUrogb71CWoSficBxYsiAGd+Kl0mmq/MprG9yArRkyrQxTO6XjMzA==",
@@ -5332,20 +5892,66 @@
         "fdir": "^6.5.0",
         "picomatch": "^4.0.4"
       },
       "engines": {
         "node": ">=12.0.0"
       },
       "funding": {
         "url": "https://github.com/sponsors/SuperchupuDev"
       }
     },
+    "node_modules/tldts": {
+      "version": "7.4.13",
+      "resolved": "https://registry.npmjs.org/tldts/-/tldts-7.4.13.tgz",
+      "integrity": "sha512-iHtaIWWIbMDkCeJdTBzZFGgbluE5J+oHlb2g7+oAz1S1gpuVpabRZdQyd471Vl8UUkcz2vXSL8xZH2kyCe8tfA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "tldts-core": "^7.4.13"
+      },
+      "bin": {
+        "tldts": "bin/cli.js"
+      }
+    },
+    "node_modules/tldts-core": {
+      "version": "7.4.13",
+      "resolved": "https://registry.npmjs.org/tldts-core/-/tldts-core-7.4.13.tgz",
+      "integrity": "sha512-mbYsrih5FRtGxs3Usvl/PqwJsNpp+jsmrdFviiK02teHDG0/HebBG/pqCylje3kzgXYzuLoHJF/0mz9W53t8Xg==",
+      "dev": true,
+      "license": "MIT"
+    },
+    "node_modules/tough-cookie": {
+      "version": "6.0.2",
+      "resolved": "https://registry.npmjs.org/tough-cookie/-/tough-cookie-6.0.2.tgz",
+      "integrity": "sha512-exgYmnmL/sJpR3upZfXG5PoatXQii55xAiXGXzY+sROLZ/Y+SLcp9PgJNI9Vz37HpQ74WvDcLT8eqm+kV3FzrA==",
+      "dev": true,
+      "license": "BSD-3-Clause",
+      "dependencies": {
+        "tldts": "^7.0.5"
+      },
+      "engines": {
+        "node": ">=16"
+      }
+    },
+    "node_modules/tr46": {
+      "version": "6.0.0",
+      "resolved": "https://registry.npmjs.org/tr46/-/tr46-6.0.0.tgz",
+      "integrity": "sha512-bLVMLPtstlZ4iMQHpFHTR7GAGj2jxi8Dg0s2h2MafAE4uSWF98FC/3MomU51iQAMf8/qDUbKWf5GxuvvVcXEhw==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "punycode": "^2.3.1"
+      },
+      "engines": {
+        "node": ">=20"
+      }
+    },
     "node_modules/trim-lines": {
       "version": "3.0.1",
       "resolved": "https://registry.npmjs.org/trim-lines/-/trim-lines-3.0.1.tgz",
       "integrity": "sha512-kRj8B+YHZCc9kQYdWfJB2/oUl9rA99qbowYYBtr4ui4mZyAQ2JpvVBd/6U2YloATfqBhBTSMhTpgBHtU0Mf3Rg==",
       "license": "MIT",
       "funding": {
         "type": "github",
         "url": "https://github.com/sponsors/wooorm"
       }
     },
@@ -5381,20 +5987,30 @@
       "dev": true,
       "license": "Apache-2.0",
       "bin": {
         "tsc": "bin/tsc",
         "tsserver": "bin/tsserver"
       },
       "engines": {
         "node": ">=14.17"
       }
     },
+    "node_modules/undici": {
+      "version": "8.10.2",
+      "resolved": "https://registry.npmjs.org/undici/-/undici-8.10.2.tgz",
+      "integrity": "sha512-/y4/bH9YNU5hi9NIrpOuvGXFcxrj3CMrV+/AYpowAYTpHn8gX/XPFjNy766FPoYY0miQhdW977JFWKGNhBdwyQ==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=22.19.0"
+      }
+    },
     "node_modules/unified": {
       "version": "11.0.5",
       "resolved": "https://registry.npmjs.org/unified/-/unified-11.0.5.tgz",
       "integrity": "sha512-xKvGhPWw3k84Qjh8bI3ZeJjqnyadK+GEFtazSfZv/rKeTkTjOJho6mFqh2SM96iIcZokxiOpg78GazTSg8+KHA==",
       "license": "MIT",
       "dependencies": {
         "@types/unist": "^3.0.0",
         "bail": "^2.0.0",
         "devlop": "^1.0.0",
         "extend": "^3.0.0",
@@ -5727,37 +6343,102 @@
     "node_modules/vitest/node_modules/tinyexec": {
       "version": "1.3.0",
       "resolved": "https://registry.npmjs.org/tinyexec/-/tinyexec-1.3.0.tgz",
       "integrity": "sha512-QKAl9m8gWWGHV8jZcPeym6j+XULi6tOf1mT83WYJ4Lk2ytW/uwAWkrP0uFsdoYMdueVJ0qs26wZ+23xeB4ibNQ==",
       "dev": true,
       "license": "MIT",
       "engines": {
         "node": ">=18"
       }
     },
+    "node_modules/w3c-xmlserializer": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/w3c-xmlserializer/-/w3c-xmlserializer-5.0.0.tgz",
+      "integrity": "sha512-o8qghlI8NZHU1lLPrpi2+Uq7abh4GGPpYANlalzWxyWteJOCsr/P+oPBA49TOLu5FTZO4d3F9MnWJfiMo4BkmA==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "xml-name-validator": "^5.0.0"
+      },
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/webidl-conversions": {
+      "version": "8.0.1",
+      "resolved": "https://registry.npmjs.org/webidl-conversions/-/webidl-conversions-8.0.1.tgz",
+      "integrity": "sha512-BMhLD/Sw+GbJC21C/UgyaZX41nPt8bUTg+jWyDeg7e7YN4xOM05YPSIXceACnXVtqyEw/LMClUQMtMZ+PGGpqQ==",
+      "dev": true,
+      "license": "BSD-2-Clause",
+      "engines": {
+        "node": ">=20"
+      }
+    },
+    "node_modules/whatwg-mimetype": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/whatwg-mimetype/-/whatwg-mimetype-5.0.0.tgz",
+      "integrity": "sha512-sXcNcHOC51uPGF0P/D4NVtrkjSU2fNsm9iog4ZvZJsL3rjoDAzXZhkm2MWt1y+PUdggKAYVoMAIYcs78wJ51Cw==",
+      "dev": true,
+      "license": "MIT",
+      "engines": {
+        "node": ">=20"
+      }
+    },
+    "node_modules/whatwg-url": {
+      "version": "17.1.2",
+      "resolved": "https://registry.npmjs.org/whatwg-url/-/whatwg-url-17.1.2.tgz",
+      "integrity": "sha512-TEZA+Zqxin7Jjsm2cjRohCmen5awh+hT6Zi3VZdqZlNRk7zvOI/9WpBFg/DWlA56bWnzwm6DuB8NS0EsxQH9uQ==",
+      "dev": true,
+      "license": "MIT",
+      "dependencies": {
+        "@exodus/bytes": "^1.15.1",
+        "tr46": "^6.0.0",
+        "webidl-conversions": "^8.0.1"
+      },
+      "engines": {
+        "node": "^22.14.0 || >=24.0.0"
+      }
+    },
     "node_modules/why-is-node-running": {
       "version": "2.3.0",
       "resolved": "https://registry.npmjs.org/why-is-node-running/-/why-is-node-running-2.3.0.tgz",
       "integrity": "sha512-hUrmaWBdVDcxvYqnyh09zunKzROWjbZTiNy8dBEjkS7ehEDQibXJ7XvlmtbwuTclUiIyN+CyXQD4Vmko8fNm8w==",
       "dev": true,
       "license": "MIT",
       "dependencies": {
         "siginfo": "^2.0.0",
         "stackback": "0.0.2"
       },
       "bin": {
         "why-is-node-running": "cli.js"
       },
       "engines": {
         "node": ">=8"
       }
     },
+    "node_modules/xml-name-validator": {
+      "version": "5.0.0",
+      "resolved": "https://registry.npmjs.org/xml-name-validator/-/xml-name-validator-5.0.0.tgz",
+      "integrity": "sha512-EvGK8EJ3DhaHfbRlETOWAS5pO9MZITeauHKJyb8wyajUfQUenkIg2MvLDTZ4T/TgIcm3HU0TFBgWWboAZ30UHg==",
+      "dev": true,
+      "license": "Apache-2.0",
+      "engines": {
+        "node": ">=18"
+      }
+    },
+    "node_modules/xmlchars": {
+      "version": "2.2.0",
+      "resolved": "https://registry.npmjs.org/xmlchars/-/xmlchars-2.2.0.tgz",
+      "integrity": "sha512-JZnDKK8B0RCDw84FNdDAIpZK+JuJw+s7Lz8nksI7SIuU3UXJJslUthsi+uWBUYOwPFwW7W7PRLRfUKpxjtjFCw==",
+      "dev": true,
+      "license": "MIT"
+    },
     "node_modules/yallist": {
       "version": "3.1.1",
       "resolved": "https://registry.npmjs.org/yallist/-/yallist-3.1.1.tgz",
       "integrity": "sha512-a4UGQaWPH59mOXUYnAG2ewncQS4i4F43Tv3JoAM+s2VDAmS9NsK8GpDMLrCHPksFT7h3K6TOoUNn2pb7RoXx4g==",
       "dev": true,
       "license": "ISC"
     },
     "node_modules/zwitch": {
       "version": "2.0.4",
       "resolved": "https://registry.npmjs.org/zwitch/-/zwitch-2.0.4.tgz",
diff --git a/skillgrid-ui/package.json b/skillgrid-ui/package.json
index 34a1511..5a58bbc 100644
--- a/skillgrid-ui/package.json
+++ b/skillgrid-ui/package.json
@@ -25,19 +25,22 @@
     "mermaid": "^12.0.0",
     "react": "^19.1.0",
     "react-dom": "^19.1.0",
     "react-markdown": "^10.1.0",
     "rehype-sanitize": "^6.0.0",
     "remark-gfm": "^4.0.1",
     "sigma": "^3.0.3"
   },
   "devDependencies": {
     "@tailwindcss/vite": "^4.1.11",
+    "@testing-library/dom": "^10.4.2",
+    "@testing-library/react": "^16.3.3",
     "@types/react": "^19.1.8",
     "@types/react-dom": "^19.1.6",
     "@vitejs/plugin-react": "^4.6.0",
+    "jsdom": "^30.1.0",
     "tailwindcss": "^4.1.11",
     "typescript": "~5.8.3",
     "vite": "^6.3.5",
     "vitest": "^5.0.1"
   }
 }
diff --git a/skillgrid-ui/src/app.tsx b/skillgrid-ui/src/app.tsx
index acffb39..27583c1 100644
--- a/skillgrid-ui/src/app.tsx
+++ b/skillgrid-ui/src/app.tsx
@@ -84,20 +84,32 @@ const plansRoute = createRoute({
   path: '/plans',
   component: lazyPage(() => import('./features/plans/PlansPage').then((m) => ({ default: m.PlansPage }))),
 })
 
 const activityRoute = createRoute({
   getParentRoute: () => rootRoute,
   path: '/activity',
   component: lazyPage(() => import('./features/activity/ActivityPage').then((m) => ({ default: m.ActivityPage }))),
 })
 
+const handoffRoute = createRoute({
+  getParentRoute: () => rootRoute,
+  path: '/handoff',
+  component: lazyPage(() => import('./features/handoff/HandoffHubPage').then((m) => ({ default: m.HandoffHubPage }))),
+})
+
+const decisionsRoute = createRoute({
+  getParentRoute: () => rootRoute,
+  path: '/decisions',
+  component: lazyPage(() => import('./features/decisions/DecisionsPage').then((m) => ({ default: m.DecisionsPage }))),
+})
+
 const gitRoute = createRoute({
   getParentRoute: () => rootRoute,
   path: '/git',
   component: lazyPage(() => import('./features/git/GitPage').then((m) => ({ default: m.GitPage }))),
 })
 
 const prototypesRoute = createRoute({
   getParentRoute: () => rootRoute,
   path: '/prototypes',
   component: lazyPage(() => import('./features/prototypes/PrototypesPage').then((m) => ({ default: m.PrototypesPage }))),
@@ -119,20 +131,22 @@ const routeTree = rootRoute.addChildren([
   indexRoute,
   trackerRoute,
   graphRoute,
   filesRoute,
   memoriesRoute,
   sessionsRoute,
   searchRoute,
   docsRoute,
   plansRoute,
   activityRoute,
+  handoffRoute,
+  decisionsRoute,
   gitRoute,
   prototypesRoute,
   settingsRoute,
   notFoundRoute,
 ])
 
 const router = createRouter({ routeTree })
 
 export function App() {
   return <RouterProvider router={router} />
diff --git a/skillgrid-ui/src/components/layout/AppLayout.tsx b/skillgrid-ui/src/components/layout/AppLayout.tsx
index ede8e06..a5e5392 100644
--- a/skillgrid-ui/src/components/layout/AppLayout.tsx
+++ b/skillgrid-ui/src/components/layout/AppLayout.tsx
@@ -7,21 +7,23 @@ const MNEMONIC_ITEMS: Array<{ to: string; label: string }> = [
   { to: '/mnemonic/graph', label: 'Graph' },
   { to: '/mnemonic/files', label: 'Files' },
   { to: '/mnemonic/memories', label: 'Memories' },
   { to: '/mnemonic/sessions', label: 'Sessions' },
   { to: '/mnemonic/search', label: 'Search' },
 ]
 
 const STANDALONE_ITEMS: Array<{ to: string; label: string }> = [
   { to: '/docs', label: 'Docs' },
   { to: '/plans', label: 'Plans' },
+  { to: '/handoff', label: 'Handoff' },
   { to: '/activity', label: 'Activity' },
+  { to: '/decisions', label: 'Decisions' },
   { to: '/git', label: 'Git' },
   { to: '/prototypes', label: 'Prototypes' },
   { to: '/settings', label: 'Settings' },
 ]
 
 export function AppLayout() {
   // Phase 7.4 responsive: the sidebar is a slide-over on mobile (hidden by
   // default, opened via the hamburger) and a fixed column on lg+ screens.
   const [navOpen, setNavOpen] = useState(false)
   const hamburgerRef = useRef<HTMLButtonElement>(null)
diff --git a/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx b/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
new file mode 100644
index 0000000..c38d0a6
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
@@ -0,0 +1,78 @@
+import { render, screen, fireEvent } from '@testing-library/react'
+import { describe, it, expect, vi } from 'vitest'
+import { DecisionCard, type Decision } from './DecisionCard'
+
+const pending: Decision = {
+  id: 1,
+  topicKey: 'interview/demo/layout-1',
+  title: 'Layout',
+  createdAt: '2026-09-19T10:00:00Z',
+  updatedAt: '2026-09-19T10:00:00Z',
+  visibility: 'team',
+  content: {
+    question: 'Which layout?',
+    options: [
+      { id: 'a', label: 'Single column' },
+      { id: 'b', label: 'Two column' },
+    ],
+    recommended: 'a',
+    state: 'pending',
+  },
+}
+
+const answered: Decision = {
+  ...pending,
+  id: 2,
+  content: {
+    ...pending.content,
+    state: 'answered',
+    answeredOption: 'b',
+    answerNote: 'picked two column for density',
+  },
+}
+
+describe('DecisionCard', () => {
+  it('renders options with the recommended one highlighted', () => {
+    render(<DecisionCard decision={pending} onAnswer={vi.fn()} />)
+    expect(screen.getByText('Which layout?')).toBeTruthy()
+    expect(screen.getByText('Single column')).toBeTruthy()
+    expect(screen.getByText('Two column')).toBeTruthy()
+    const rec = screen.getByTestId('decision-option-a')
+    expect(rec.className).toContain('recommended')
+  })
+
+  it('renders no iframe for a decision without visual', () => {
+    const { container } = render(<DecisionCard decision={pending} onAnswer={vi.fn()} />)
+    expect(container.querySelector('iframe')).toBeNull()
+  })
+
+  it('renders the sandboxed iframe for a decision with visual', () => {
+    const withVisual: Decision = {
+      ...pending,
+      content: { ...pending.content, visual: 'demo/a.html' },
+    }
+    const { container } = render(<DecisionCard decision={withVisual} onAnswer={vi.fn()} />)
+    const iframe = container.querySelector('iframe')
+    expect(iframe).not.toBeNull()
+    expect(iframe?.getAttribute('src')).toBe('/prototype/demo/a.html')
+    expect(iframe?.getAttribute('sandbox')).toBe('allow-scripts')
+    expect(iframe?.getAttribute('sandbox')).not.toContain('allow-same-origin')
+  })
+
+  it('posts the chosen option and note on submit', () => {
+    const onAnswer = vi.fn()
+    render(<DecisionCard decision={pending} onAnswer={onAnswer} />)
+    fireEvent.change(screen.getByPlaceholderText(/optional note/i), {
+      target: { value: 'prefer a' },
+    })
+    fireEvent.click(screen.getByTestId('decision-submit'))
+    expect(onAnswer).toHaveBeenCalledWith(1, { optionId: 'a', note: 'prefer a' })
+  })
+
+  it('re-submits a different option on an answered decision', () => {
+    const onAnswer = vi.fn()
+    render(<DecisionCard decision={answered} onAnswer={onAnswer} />)
+    fireEvent.click(screen.getByTestId('decision-submit'))
+    expect(onAnswer).toHaveBeenCalledWith(2, { optionId: 'b', note: '' })
+  })
+})
diff --git a/skillgrid-ui/src/features/decisions/DecisionCard.tsx b/skillgrid-ui/src/features/decisions/DecisionCard.tsx
new file mode 100644
index 0000000..5e33944
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/DecisionCard.tsx
@@ -0,0 +1,171 @@
+import { useState } from 'react'
+
+// DecisionCard renders one decision from the inbox: the question, its
+// lettered options (the recommended one highlighted), an optional note, and a
+// Submit that hands {optionId, note} to onAnswer. On an answered decision it
+// shows the stored choice + note; the options stay re-answerable (the backend
+// appends a version) and re-selecting the picked option re-submits it.
+// A decision whose content carries a `visual` gets a sandboxed prototype
+// iframe (src=/prototype/{visual}, sandbox="allow-scripts", no
+// allow-same-origin) rendering the companion's throwaway HTML from
+// .skillgrid/prototype/ — the same sandbox contract as SandboxPreview.
+
+export interface DecisionOption {
+  id: string
+  label: string
+  rationale?: string
+}
+
+export interface DecisionContent {
+  question: string
+  options: DecisionOption[]
+  recommended?: string
+  state: string
+  answeredOption?: string
+  answerNote?: string
+  updatedBy?: string
+  visual?: string
+}
+
+export interface Decision {
+  id: number
+  topicKey: string
+  title: string
+  createdAt: string
+  updatedAt: string
+  visibility: string
+  parseError?: boolean
+  content: DecisionContent
+}
+
+function day(iso: string): string {
+  const d = new Date(iso)
+  return isNaN(d.getTime()) ? iso : d.toISOString().slice(0, 10)
+}
+
+export function DecisionCard({
+  decision,
+  onAnswer,
+}: {
+  decision: Decision
+  onAnswer: (id: number, answer: { optionId: string; note: string }) => void
+}) {
+  const [note, setNote] = useState('')
+  const [selected, setSelected] = useState<string | null>(null)
+  const [busy, setBusy] = useState(false)
+
+  if (decision.parseError) {
+    return (
+      <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-200">
+        Unreadable decision: {decision.topicKey}
+      </div>
+    )
+  }
+
+  const { content } = decision
+  const answered = content.state === 'answered'
+  const pickedId = answered ? content.answeredOption : undefined
+  const effective = selected ?? pickedId ?? content.recommended ?? ''
+  const hasVisual = Boolean(content.visual)
+
+  const submit = () => {
+    if (!effective || busy) return
+    setBusy(true)
+    onAnswer(decision.id, { optionId: effective, note })
+  }
+
+  return (
+    <div className="rounded-md border border-edge bg-card p-4">
+      <div className="mb-1 flex items-baseline justify-between gap-2">
+        <span className="truncate text-xs uppercase tracking-wide text-zinc-500">
+          {decision.topicKey}
+        </span>
+        <span className="shrink-0 text-[11px] text-zinc-500">{day(decision.createdAt)}</span>
+      </div>
+      <h3 className="mb-3 font-semibold text-zinc-100">{content.question}</h3>
+
+      <div className="flex flex-col gap-2" role="radiogroup" aria-label="Options">
+        {content.options.map((o, i) => {
+          const isRec = o.id === content.recommended
+          const isPicked = answered && o.id === content.answeredOption
+          const isEffective = effective === o.id
+          return (
+            <button
+              key={o.id}
+              type="button"
+              role="radio"
+              aria-checked={isEffective}
+              data-testid={`decision-option-${o.id}`}
+              disabled={busy}
+              onClick={() => setSelected(o.id)}
+              className={[
+                'flex items-start gap-2 rounded border p-2 text-left text-sm transition',
+                isRec ? 'recommended border-accent/70 bg-accent/10' : 'border-edge hover:bg-bg',
+                isEffective ? 'ring-1 ring-accent' : '',
+                isPicked ? 'border-emerald-500/60 bg-emerald-500/10' : '',
+                busy ? 'opacity-60' : '',
+              ].join(' ')}
+            >
+              <span className="mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border border-zinc-500 text-[10px] font-semibold text-zinc-300">
+                {String.fromCharCode(65 + i)}
+              </span>
+              <span className="min-w-0 flex-1">
+                <span className="font-medium text-zinc-200">{o.label}</span>
+                {isRec && !answered && (
+                  <span className="ml-2 rounded bg-accent/20 px-1 py-0.5 text-[10px] uppercase tracking-wide text-accent">
+                    recommended
+                  </span>
+                )}
+                {isPicked && (
+                  <span className="ml-2 text-[11px] text-emerald-300">your choice</span>
+                )}
+                {o.rationale && (
+                  <span className="mt-0.5 block text-xs text-zinc-500">{o.rationale}</span>
+                )}
+              </span>
+            </button>
+          )
+        })}
+      </div>
+
+      {answered && content.answerNote && (
+        <div className="mt-2 text-xs text-zinc-500">
+          Note: <span className="text-zinc-300">{content.answerNote}</span>
+          {content.updatedBy && (
+            <span className="ml-2 text-zinc-600">· {content.updatedBy}</span>
+          )}
+        </div>
+      )}
+
+      {hasVisual && (
+        <div className="mt-3 overflow-hidden rounded-md border border-edge bg-white">
+          <iframe
+            title={`Prototype: ${content.visual}`}
+            src={`/prototype/${content.visual}`}
+            sandbox="allow-scripts"
+            className="h-[420px] w-full border-0"
+          />
+        </div>
+      )}
+
+      <div className="mt-3 flex items-center gap-2">
+        <input
+          value={note}
+          onChange={(e) => setNote(e.target.value)}
+          placeholder="Optional note for the agent…"
+          aria-label="Optional note"
+          className="min-w-0 flex-1 rounded-md border border-edge bg-bg px-2 py-1 text-sm text-zinc-200 placeholder:text-zinc-600 focus:border-accent/60 focus:outline-none"
+        />
+        <button
+          type="button"
+          data-testid="decision-submit"
+          disabled={!effective || busy}
+          onClick={submit}
+          className="shrink-0 rounded-md bg-accent/80 px-3 py-1.5 text-xs font-medium text-zinc-950 hover:bg-accent disabled:opacity-40"
+        >
+          {answered ? 'Update answer' : 'Submit'}
+        </button>
+      </div>
+    </div>
+  )
+}
diff --git a/skillgrid-ui/src/features/decisions/DecisionsPage.test.tsx b/skillgrid-ui/src/features/decisions/DecisionsPage.test.tsx
new file mode 100644
index 0000000..197a4d4
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/DecisionsPage.test.tsx
@@ -0,0 +1,84 @@
+import { render, screen } from '@testing-library/react'
+import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
+import { DecisionsPage } from './DecisionsPage'
+import * as api from './api'
+
+// The page polls on a 4s interval. To test the interval without holding the
+// suite open, we spy on the global setInterval the component installs and
+// invoke the captured callback by hand (real timers, so findBy* waits work).
+function renderPolling() {
+  const intervalSpy = vi
+    .spyOn(globalThis, 'setInterval')
+    .mockImplementation(() => 1 as unknown as ReturnType<typeof setInterval>)
+  const utils = render(<DecisionsPage />)
+  const flush = () => {
+    for (const [arg] of intervalSpy.mock.calls as unknown as Array<[(cb: unknown) => void]>) {
+      if (typeof arg === 'function') (arg as () => void)()
+    }
+  }
+  return { ...utils, flush, intervalSpy }
+}
+
+let fetchSpy: Mock
+
+function row(id: number, state: 'pending' | 'answered'): api.DecisionRow {
+  return {
+    id,
+    topicKey: `demo/decision-${id}`,
+    title: `Decision ${id}`,
+    createdAt: '2026-09-19T10:00:00Z',
+    updatedAt: '2026-09-19T10:00:00Z',
+    visibility: 'team',
+    content: {
+      question: `Question ${id}?`,
+      options: [
+        { id: 'a', label: `Option A ${id}` },
+        { id: 'b', label: `Option B ${id}` },
+      ],
+      recommended: 'a',
+      state,
+      ...(state === 'answered' ? { answeredOption: 'b' } : {}),
+    },
+  }
+}
+
+describe('DecisionsPage', () => {
+  beforeEach(() => {
+    fetchSpy = vi.spyOn(api, 'fetchDecisions').mockResolvedValue([])
+    vi.spyOn(api, 'answerDecision').mockResolvedValue(undefined)
+  })
+  afterEach(() => {
+    vi.restoreAllMocks()
+  })
+
+  it('renders the pending decision inbox (question + options + created)', async () => {
+    fetchSpy.mockResolvedValue([row(1, 'pending')])
+    render(<DecisionsPage />)
+    expect(await screen.findByText('Question 1?')).toBeTruthy()
+    expect(screen.getByText('Option A 1')).toBeTruthy()
+    expect(screen.getByText('Option B 1')).toBeTruthy()
+    expect(screen.getByText(/2026-09-19/)).toBeTruthy()
+  })
+
+  it('fetches pending by default', async () => {
+    render(<DecisionsPage />)
+    await screen.findByText('No pending decisions.')
+    expect(fetchSpy).toHaveBeenCalledWith('pending')
+  })
+
+  it('refreshes the inbox on an interval', async () => {
+    fetchSpy.mockResolvedValue([row(1, 'pending')])
+    const { flush } = renderPolling()
+    await screen.findByText('Question 1?')
+    expect(fetchSpy).toHaveBeenCalledTimes(1)
+    flush()
+    expect(fetchSpy).toHaveBeenCalledTimes(2)
+  })
+
+  it('shows a stored answer for an answered decision', async () => {
+    fetchSpy.mockResolvedValue([row(2, 'answered')])
+    render(<DecisionsPage />)
+    await screen.findByText('Question 2?')
+    expect(screen.getByText(/your choice/i)).toBeTruthy()
+  })
+})
diff --git a/skillgrid-ui/src/features/decisions/DecisionsPage.tsx b/skillgrid-ui/src/features/decisions/DecisionsPage.tsx
new file mode 100644
index 0000000..da18d76
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/DecisionsPage.tsx
@@ -0,0 +1,86 @@
+import { useCallback, useEffect, useState } from 'react'
+import { DecisionCard, type Decision } from './DecisionCard'
+import { answerDecision, DecisionApiError, fetchDecisions } from './api'
+
+// DecisionsPage is the decision inbox: the pending interview questions the
+// agent posted, pollable on an interval (no WebSocket/SSE), with an answer
+// round-trip. A pending decision appears here, the user picks an option +
+// optional note and submits (POST …/answer), the card flips to state=answered
+// and shows the stored answer; a second submit re-answers (backend appends a
+// version). The default tab is the pending inbox.
+
+const REFRESH_MS = 4000
+
+type Tab = 'pending' | 'answered' | 'all'
+
+export function DecisionsPage() {
+  const [tab, setTab] = useState<Tab>('pending')
+  const [rows, setRows] = useState<Decision[]>([])
+  const [error, setError] = useState('')
+
+  const load = useCallback(async () => {
+    try {
+      setRows(await fetchDecisions(tab))
+      setError('')
+    } catch (e) {
+      setError(e instanceof DecisionApiError ? e.message : 'failed to load decisions')
+    }
+  }, [tab])
+
+  useEffect(() => {
+    void load()
+    const t = setInterval(() => void load(), REFRESH_MS)
+    return () => clearInterval(t)
+  }, [load])
+
+  const onAnswer = useCallback(
+    async (id: number, answer: { optionId: string; note: string }) => {
+      await answerDecision(id, answer)
+      void load()
+    },
+    [load],
+  )
+
+  return (
+    <div className="flex h-full flex-col">
+      <div className="flex items-center gap-3 border-b border-edge px-4 py-2">
+        <h1 className="text-sm font-semibold text-zinc-100">Decisions</h1>
+        <div className="flex gap-1" role="tablist" aria-label="Decision state">
+          {(['pending', 'answered', 'all'] as const).map((t) => (
+            <button
+              key={t}
+              type="button"
+              role="tab"
+              aria-selected={tab === t}
+              onClick={() => setTab(t)}
+              className={[
+                'rounded px-2.5 py-1 text-xs transition-colors',
+                tab === t ? 'bg-accent/20 text-accent' : 'text-zinc-500 hover:bg-card hover:text-zinc-300',
+              ].join(' ')}
+            >
+              {t}
+            </button>
+          ))}
+        </div>
+      </div>
+
+      {error && (
+        <div className="border-b border-red-500/30 bg-red-500/10 px-4 py-2 text-xs text-red-300">
+          {error}
+        </div>
+      )}
+
+      <div className="min-h-0 flex-1 overflow-y-auto p-4">
+        {rows.length === 0 ? (
+          <div className="text-sm text-zinc-500">No {tab} decisions.</div>
+        ) : (
+          <div className="mx-auto flex max-w-2xl flex-col gap-3">
+            {rows.map((d) => (
+              <DecisionCard key={d.id} decision={d} onAnswer={onAnswer} />
+            ))}
+          </div>
+        )}
+      </div>
+    </div>
+  )
+}
diff --git a/skillgrid-ui/src/features/decisions/api.ts b/skillgrid-ui/src/features/decisions/api.ts
new file mode 100644
index 0000000..e79cd8d
--- /dev/null
+++ b/skillgrid-ui/src/features/decisions/api.ts
@@ -0,0 +1,78 @@
+import { currentProjectName } from '../../lib/projects'
+import type { Decision } from './DecisionCard'
+
+// The decisions bridge read path is open; the answer (write) path is
+// token-gated when the server sets SKILLGRID_HTTP_TOKEN. The token comes from
+// the Settings view (localStorage) — the SPA cannot read the server's env, so
+// the user supplies it once and it is attached to the mutating request, exactly
+// like the tracker/kanban write routes.
+const TOKEN_KEY = 'skillgrid.httpToken'
+
+export class DecisionApiError extends Error {
+  status: number
+  constructor(message: string, status: number) {
+    super(message)
+    this.status = status
+  }
+}
+
+function getToken(): string {
+  try {
+    return window.localStorage.getItem(TOKEN_KEY) ?? ''
+  } catch {
+    return ''
+  }
+}
+
+function authHeaders(): Record<string, string> {
+  const token = getToken()
+  return token ? { Authorization: `Bearer ${token}` } : {}
+}
+
+async function parseError(res: Response): Promise<string> {
+  try {
+    const j = (await res.json()) as { error?: string }
+    if (j.error) return j.error
+  } catch {
+    /* keep statusText */
+  }
+  return res.statusText
+}
+
+// DecisionRow mirrors the backend GET /mnemonic/decisions row (camelCase JSON).
+export type DecisionRow = Decision
+
+let resolvedProject: string | null = null
+
+async function resolveProject(): Promise<string> {
+  if (resolvedProject) return resolvedProject
+  resolvedProject = await currentProjectName()
+  return resolvedProject
+}
+
+export async function fetchDecisions(state: string): Promise<DecisionRow[]> {
+  const project = await resolveProject()
+  const q = new URLSearchParams({ project })
+  if (state) q.set('state', state)
+  const res = await fetch(`/mnemonic/decisions?${q.toString()}`, {
+    headers: { Accept: 'application/json' },
+  })
+  if (!res.ok) throw new DecisionApiError(await parseError(res), res.status)
+  return (await res.json()) as DecisionRow[]
+}
+
+export async function answerDecision(
+  id: number,
+  answer: { optionId: string; note: string },
+): Promise<void> {
+  const project = await resolveProject()
+  const res = await fetch(
+    `/mnemonic/decisions/${id}/answer?project=${encodeURIComponent(project)}`,
+    {
+      method: 'POST',
+      headers: { 'Content-Type': 'application/json', Accept: 'application/json', ...authHeaders() },
+      body: JSON.stringify(answer),
+    },
+  )
+  if (!res.ok) throw new DecisionApiError(await parseError(res), res.status)
+}
diff --git a/skillgrid-ui/src/vite-env.d.ts b/skillgrid-ui/src/vite-env.d.ts
index 11f02fe..d3a8309 100644
--- a/skillgrid-ui/src/vite-env.d.ts
+++ b/skillgrid-ui/src/vite-env.d.ts
@@ -1 +1,2 @@
 /// <reference types="vite/client" />
+/// <reference types="vitest" />
diff --git a/skillgrid-ui/vite.config.ts b/skillgrid-ui/vite.config.ts
index bf09ccd..1d9fce0 100644
--- a/skillgrid-ui/vite.config.ts
+++ b/skillgrid-ui/vite.config.ts
@@ -1,22 +1,32 @@
+/// <reference types="vitest/config" />
 import { defineConfig } from 'vite'
 import react from '@vitejs/plugin-react'
 import tailwindcss from '@tailwindcss/vite'
 
 export default defineConfig({
   base: './',
   plugins: [react(), tailwindcss()],
   server: {
     proxy: {
       '/api': 'http://127.0.0.1:7438',
     },
   },
+  test: {
+    // Component tests (decisions view) render real DOM + act on it; the
+    // existing pure-logic tests keep running fine under jsdom too.
+    environment: 'jsdom',
+    // globals enables @testing-library/react's afterEach auto-cleanup (it
+    // looks the cleanup hook up via a global import), so each render() is
+    // torn down between tests.
+    globals: true,
+  },
   build: {
     outDir: '../skillgrid-cli/internal/mnemonic/http/ui/dist',
     emptyOutDir: true,
     // mermaid (~728 kB) is a single monolithic lib that Rollup can't split
     // further; it's dynamic-imported so it's lazy-loaded only when a diagram
     // renders, but it still trips the 500 kB warning. Bump the limit so the
     // build is quiet — the real win is per-library cacheable chunks, not
     // inlining heavy deps into the app bundle.
     chunkSizeWarningLimit: 800,
     rollupOptions: {
