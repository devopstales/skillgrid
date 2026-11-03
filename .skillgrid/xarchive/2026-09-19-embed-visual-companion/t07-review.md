# Review package: 1086a2b..8148b5d

## Commits
8148b5d docs(http): openapi + user guide for decision companion (TICKET-07)
49b3a25 test(http): RED round-trip guard for decision MCP read-back (TICKET-07)

## Files changed
 docs/user-guide/10-decision-companion.md           | 180 ++++++++++++++++++
 .../internal/mnemonic/http/decisions_test.go       | 154 ++++++++++++++++
 .../internal/mnemonic/http/openapi_phase8_test.go  |  90 +++++++++
 .../internal/mnemonic/http/ui/openapi.yaml         | 202 ++++++++++++++++++++-
 4 files changed, 624 insertions(+), 2 deletions(-)

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
diff --git a/skillgrid-cli/internal/mnemonic/http/decisions_test.go b/skillgrid-cli/internal/mnemonic/http/decisions_test.go
index 26b1214..1dace79 100644
--- a/skillgrid-cli/internal/mnemonic/http/decisions_test.go
+++ b/skillgrid-cli/internal/mnemonic/http/decisions_test.go
@@ -403,10 +403,164 @@ func TestDecisionUpsertBumpsRevision(t *testing.T) {
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
