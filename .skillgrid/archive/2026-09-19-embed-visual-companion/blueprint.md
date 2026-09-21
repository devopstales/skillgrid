# Embed Visual Companion (Mnemonic Decision Bridge) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the visual companion a *decision bridge* on top of Mnemonic: the agent posts interview questions (options + a recommended option + rationale) as governed observations; the dashboard renders them as a decision inbox; the user approves/answers; the agent reads the decision back. When a question is clearer *shown* than told, the agent also writes a throwaway interactive prototype to `.skillgrid/prototype/<topic>/<variant>.html`, and the dashboard renders it in the existing sandboxed-iframe preview, linked from the decision card. No WebSocket, no second process, no new store — the companion is a view over `type=decision` observations (durable decisions) plus on-disk `.skillgrid/prototype/` HTML (the visual), both served by `skillgrid serve`.

**Architecture:** Two artifacts, two stores, one dashboard. **(1) Decisions — a Mnemonic convention, not a migration.** The agent uses the **existing** `mem_save`/`mem_update` MCP tools to write an observation with `type: decision`, `topic_key: interview/<slug>/<question>`, `visibility: team`, and a structured JSON `content` body carrying `question`, `options[]` (each with `id`, `label`, `rationale?`), `recommended` (an option id), `state` (`pending` | `answered` | `superseded`), and an optional `visual` (the prototype id, see below). The dashboard gains a **Decisions** view that polls `GET /mnemonic/decisions` (a thin read-only route over the existing store) and renders pending decision cards; the user answers/approves via a new `POST /mnemonic/decisions/{id}/answer` (write-gated, reuses the governance write path to set `state: answered` + record the answer). The agent reads the answer by polling `mem_search` / the existing activity stream for `topic_key` rows whose `content.state == answered`. **(2) Prototypes — on-disk, in the repo.** When a question is clearer shown than told, the agent (via the existing `sketch` skill) writes throwaway standalone interactive HTML to `.skillgrid/prototype/<topic>/<variant>.html`; the decision's `content.visual` carries the prototype id (`<topic>/<variant>.html`). A new thin route `GET /prototype/{id...}` serves that directory, **reusing the existing `stitchFile` path-traversal guard** (sandboxed to `.skillgrid/prototype/`), and the dashboard renders it in the **existing** `SandboxPreview` iframe (`sandbox="allow-scripts"`, no `allow-same-origin` — the same trust rule as the `.stitch/` Prototypes view and grill's `visual.html`). Only a re-authored variant reloads the iframe (grill's version-bump trick). **Polling is the transport** (grill-with-ui + oh-my-opencode-slim both poll; only the raw brainstorming companion uses WS, and we drop it). The long-lived `skillgrid serve` process already *is* the dashboard and already reads/writes this store, so there is nothing new to keep alive.

**Tech Stack:** Go 1.22+ (`skillgrid-cli`); existing `internal/mnemonic/http` Go server + `internal/mnemonic/memory` service (save/search/set-status/share) + SQLite store (`observations` table, `017_layered_memory_governance` for owner/status/visibility/`observation_versions`/`acl_grants`); existing `mem_save`/`mem_update`/`mem_search` MCP tools; React 19 + TypeScript + Vite 6 (`skillgrid-ui`) for the Decisions view. **No new dependency.**

**Spec:** `.skillgrid/specs/2026-09-19-embed-visual-companion/briefing.md`

**Findings:** `.skillgrid/specs/2026-09-19-embed-visual-companion/findings.md` (reference designs: grill-with-ui, oh-my-opencode-slim `/interview`, the raw brainstorming companion — see the **Reference synthesis** note in Global Constraints).

**Supersedes:** the 2026-09-19 blueprint v1 (Node `server.cjs` port: WS + fs.watch + verbatim `frame-template.html`/`helper.js`). That path is abandoned in favor of the Mnemonic decision bridge; the `frame`/`helper` reuse, the WebSocket, and the `?key=` gate are all dropped.

## Global Constraints

- **Mnemonic is the bridge.** All state lives in the existing `observations` table. No new table, no new store, no `state.json`/`events.jsonl`. The companion is a *view over decision-observations*.
- **Convention, not migration.** The decision payload is structured JSON inside `content` + the existing `type`/`topic_key`/`visibility` columns. Do **not** add a `decisions` table or new columns in this change (that is the deferred heavier alternative — see the decision-subsection of the Hypothesis). If the inbox query gets painful later, a `decisions` table is a follow-up change, not this one.
- **`status` is closed to `active | superseded | archived`** (see `memory/governance.go` `IsValidStatus`). Therefore the approval gate is a **convention field `state` inside `content`** (`pending` | `answered` | `superseded`), *not* a new `status` value. Do not try to use `status` for pending/approved — it is reserved for the governance lifecycle.
- **Polling, not push.** The dashboard polls; the agent polls. No WebSocket, no SSE for the decision channel (the existing `/activity/stream` SSE may be reused to *surface* new decisions in the Activity feed, but it is not the decision transport). This matches grill-with-ui and oh-my-opencode-slim, both of which are explicitly "polling, not realtime."
- **Loopback only.** Inherits `skillgrid serve`'s `127.0.0.1` bind. No new host/port.
- **Governance is inherited.** Decision observations get `owner`, `visibility` (default `team` so the dashboard's reader sees them), `acl_grants`, and an **append-only version history** (`observation_versions`) for free — every answer/approval is an `mem_update`, which appends a version. That is the durable, auditable decision trail.
- **Prototypes live in the repo at `.skillgrid/prototype/<topic>/<variant>.html`.** This is the on-disk artifact store for the visual function (chosen over a Mnemonic content blob). It is *separate* from the existing `.stitch/` directory (which holds Stitch-generated design-system prototypes) and from `sketch.dir` (`{specs_root}/{topic}/sketches/` — where the `sketch` skill currently drops its throwaway HTML). The `sketch` skill must be pointed at `.skillgrid/prototype/<topic>/` for the companion flow (a config/skill note, not a code change in this blueprint). `.skillgrid/` is already under the artifacts root and is expected to be gitignored for scratch; confirm it's in `.gitignore` (the prototypes are throwaway by design).
- **Prototype serving reuses the existing `stitchFile` traversal guard.** `GET /prototype/{id...}` must sandbox to `.skillgrid/prototype/` exactly like `GET /prototypes/{id...}` sandboxes to `.stitch/` (reject absolute ids, `..` segments, and any path that escapes the root after `filepath.Clean`). Do not invent a new traversal check.
- **MCP surface is frozen.** `mem_save`/`mem_update`/`mem_search` signatures and return shapes are unchanged. The agent already calls them; this change only defines the *shape of the content* they carry and adds thin HTTP routes for the dashboard (decisions read/answer + prototype serve).
- **Write-gated like the rest.** New write routes go through the existing `requireWriteAuth` (Bearer `SKILLGRID_HTTP_TOKEN`), exactly like `/mnemonic/memories/{id}/status`.
- **Bundle budget.** The Decisions view is its own lazy chunk; `npm run build:check` must pass.
- **Reference synthesis (why this shape).** grill-with-ui: structured interview (questions/recommendations/threads), polling `GET /state`, staged single **Send**, durable `events.jsonl` + design doc, "server IS the monitor." oh-my-opencode-slim `/interview`: questions + *suggested answers marked recommended*, a **dumb dashboard aggregator** on a fixed port that **auto-failovers and rebuilds from on-disk markdown**, polling transport, in-memory runtime + durable file. brainstorming companion: mockup display, raw WebSocket + fs.watch, `?key=` gate — the *weakest* on durability, the only one using WS. **The Mnemonic bridge adopts grill's structure + oh-my-opencode-slim's durability, but the "dumb dashboard that rebuilds from disk" is the Mnemonic store, and the "smart session" is the agent via existing MCP tools.**

## Hypothesis

**Claim:** The companion can be a pure view over `type=decision` Mnemonic observations — agent writes via existing `mem_save`, dashboard reads via one thin route, user answers via one thin write route, agent reads back via existing `mem_search`/activity — with zero new transport, zero new store, and a durable governed decision trail.

**Right condition:** With only `skillgrid serve` running, an agent `mem_save(type=decision, topic_key=interview/<slug>/<q>, visibility=team, content={question,options,recommended,state:pending})` appears in the dashboard's Decisions inbox; the user's answer (via `POST /mnemonic/decisions/{id}/answer`) sets `state=answered` + records the chosen option + appends a version; the agent's next `mem_search(topic_key)` returns the answered decision; the whole round-trip survives a server restart (it's in SQLite).

**Wrong condition:** Any of the above requires a new table/column, a WebSocket, a second process, or a store the agent cannot already read/write via MCP.

**Thinnest MVP:** Task 1 (decision content convention + `GET /mnemonic/decisions` read route) and Task 3 (user answer write route). If the read route cannot filter `type=decision` + parse `content.state`, or the write route cannot reuse the governance path, the approach is invalidated.

**Door check:** Task 1. If `GET /mnemonic/decisions` cannot list `type=decision` observations and parse their `content.state` from the existing store, stop and reconsider (the whole bridge rests on reading decisions back out of the store the agent already writes to).

> **Deferred sub-decision (do NOT do it in this change):** *structured `content` convention* (this plan) vs a *dedicated `decisions` table*. Start with the convention (migration-free); promote to a table only if the inbox query/filtering proves painful in use. Recorded here so the executor does not "helpfully" add a table.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Decision `content` is untrusted JSON** (agent-authored, parsed by the dashboard) | Applicable | `GET /mnemonic/decisions` and the answer route parse `content` with `encoding/json` into a typed struct; malformed `content` on a `type=decision` row is **skipped** (not failed) with a debug log, and the row is still returned with a `parseError` flag so the UI can show "unreadable decision." Never crash the handler on one bad row. | `TestDecisionsSkipsMalformedContent` — a `type=decision` row with broken JSON is skipped in the list and does not 500. |
| **`topic_key` collision / upsert surprise** | Applicable | The interview `topic_key` is `interview/<slug>/<question-id>` (stable per question). Reusing the same `topic_key` with `mem_save` **upserts** the same row and bumps `revision_count` (existing behavior, `memory/service.go`) — that is how the agent updates a question in place (e.g. changes a recommendation). Assert this in the test so an executor doesn't accidentally create duplicate rows. | `TestDecisionUpsertBumpsRevision` — saving the same `topic_key` twice yields one row with `revision_count` incremented and a version-history entry. |
| **`state` drift (agent and user both write)** | Applicable | Single-writer-per-field: the **agent** owns `question/options/recommended` and `state` transitions `pending→superseded`; the **user** owns the `answer` + the `pending→answered` transition (via the answer route). The answer route is the only writer of `state=answered`. A re-answer appends another version (append-only history), never overwrites. | `TestAnswerAppendsVersion` — answering twice yields two `observation_versions` rows; latest content has the second answer. |
| **Visibility gate (dashboard reader must see decisions)** | Applicable | Decision observations are saved with `visibility: team` (not the default `private`), so the dashboard's reader identity sees them. The read route filters `type='decision' AND deleted_at IS NULL` and does not require the reader to be the owner (mirrors how the memories list already works). | `TestDecisionVisibleToReader` — a `visibility=team` decision is returned to a non-owner reader; a `private` one is not. |
| **Write auth on the answer route** | Applicable | `POST /mnemonic/decisions/{id}/answer` is wrapped in `requireWriteAuth` (same as `/mnemonic/memories/{id}/status`). No `SKILLGRID_HTTP_TOKEN` set → open (loopback); token set → Bearer required. | `TestAnswerRequiresAuth` — with a token configured, a no-auth answer POST is 401. |
| **SSE/WS client disconnect leaks** | N/A: the decision channel is polling, not a persistent connection. (The reused `/activity/stream` SSE already has its own leak-free poller; this change does not touch it.) | n/a | n/a |
| **Subprocess / shell commands** | N/A: no subprocess. The agent is woken by *polling* `mem_search`, not by a Monitor stdout. | n/a | n/a |
| **Mnemonic tool contract** | Applicable (but unchanged) | `mem_save`/`mem_update`/`mem_search` signatures are frozen; this change only fixes the *content shape* they carry. Guarded by the existing MCP tests + a new test asserting the decision content schema round-trips through `mem_save`→`mem_search`. | `TestDecisionMCPRoundTrip` — `mem_save` a decision, `mem_search` it back, parse `content`, assert schema fields present. |
| **Prototype path traversal** (the `visual` id is agent-authored and served from disk) | Applicable | `GET /prototype/{id...}` reuses the `stitchFile` guard shape: reject empty/absolute ids, `..` segments, and any path that escapes `.skillgrid/prototype/` after `filepath.Clean`; serve with `X-Content-Type-Options: nosniff`; render in the sandboxed iframe (no `allow-same-origin`) so prototype scripts can't read dashboard cookies. | `TestPrototypeTraversal` — `../`, absolute, and double-encode-escaping ids → 400; a real file → 200; unknown → 404. |

## One-Way-Door Checkpoints

- **None.** No migration, no new table/column, no published-contract break, no public API shape change. The MCP surface is frozen; this *adds* one read route, one write route, and a React view, and fixes a content convention. Reversal = delete the two routes + the view + the convention.

## Change Classification

- **`standard`** → verification floor **L2** (full `go test ./...` + `go build ./...` + `npm run build:check`). New trust boundary (untrusted JSON parsing) + a write route + polling concurrency push it to standard; no data migration or auth-model change, so not `risky`/`high-risk`.

## Must-Haves (Goal-Backward Verification)

**Truths:**
1. An agent `mem_save` of a `type=decision` observation is returned by `GET /mnemonic/decisions` with its `content` parsed (`question`, `options`, `recommended`, `state`). *backstop* (held-out: live agent → dashboard render).
2. `GET /mnemonic/decisions` returns only `type='decision'` rows, `deleted_at IS NULL`, and **skips** rows with malformed `content` (flagged, not 500).
3. A user answer via `POST /mnemonic/decisions/{id}/answer` sets `content.state=answered`, records the chosen `optionId` (and free text if any), and **appends an `observation_versions` row** (durable, auditable).
4. The agent reads the answer back via `mem_search` on the decision's `topic_key` (existing tool, no new read tool). *backstop*.
5. Reusing a decision's `topic_key` with `mem_save` **upserts** the same row (bumps `revision_count` + appends a version) — the agent's in-place question-update mechanism.
6. A `visibility=team` decision is visible to the dashboard's non-owner reader; a `private` one is not.
7. The answer route is write-gated: with `SKILLGRID_HTTP_TOKEN` set, a no-auth POST is 401.
8. The Decisions dashboard view renders pending decision cards (question + lettered options + the recommended option highlighted) and posts the user's choice to the answer route.
9. The whole round-trip (save → answer → read-back) survives a server restart because it is in SQLite. *backstop*.
10. `GET /prototype/{id...}` serves a file from `.skillgrid/prototype/<topic>/<variant>.html` and **rejects** absolute ids, `..` segments, and any id that escapes `.skillgrid/prototype/` (400), returning 404 for unknown files.
11. A decision whose `content.visual` is set renders its prototype in the **existing sandboxed iframe** (`sandbox="allow-scripts"`, no `allow-same-origin`) inside the decision card; a decision without `visual` renders no iframe.

**Artifacts:**
- `skillgrid-cli/internal/mnemonic/http/decisions.go` — `GET /mnemonic/decisions` + `POST /mnemonic/decisions/{id}/answer` + the decision content type (with the `visual` field).
- `skillgrid-cli/internal/mnemonic/http/prototype.go` — `GET /prototype/{id...}` + `prototypeRoot()`/`prototypeFile()` (reuses the `stitchFile` guard shape, root = `.skillgrid/prototype`).
- `skillgrid-cli/internal/mnemonic/http/decisions_test.go` + `skillgrid-cli/internal/mnemonic/http/prototype_test.go` — the RED tests above.
- `skillgrid-ui/src/features/decisions/` — `DecisionsPage.tsx`, `api.ts`, `DecisionCard.tsx` (with the sandboxed-iframe visual, reusing `SandboxPreview`).
- `ui/openapi.yaml` entries for `/mnemonic/decisions`, `/mnemonic/decisions/{id}/answer`, `/prototype/{id...}`.
- Nav entry in `skillgrid-ui/src/components/layout/AppLayout.tsx` + route in `skillgrid-ui/src/app.tsx`.
- A short convention doc: `docs/user-guide/10-decision-companion.md` (the content schema + the poll loop + the `.skillgrid/prototype/` convention), mirroring `09-serve-dashboard.md`.

**Key links:**
- Agent `mem_save(type=decision, topic_key, visibility=team, content{...state:pending, visual?})` → `observations` row.
- `GET /mnemonic/decisions` → `SELECT … WHERE type='decision'` + parse `content` → decision cards.
- Agent writes prototype HTML → `.skillgrid/prototype/<topic>/<variant>.html` → `GET /prototype/{topic/variant.html}` (traversal-guarded) → sandboxed iframe in the decision card.
- User picks option (+ optional visual note) → `POST /mnemonic/decisions/{id}/answer {optionId, note?}` → `content.state=answered` + version append.
- Agent `mem_search(topic_key)` / activity poll → reads `content.state==answered` + the chosen option → proceeds.
- `registerRoutes()` → the new routes (registered alongside the existing mnemonic routes; `/mnemonic/decisions` is **not** in the `apiPrefixes` 404-catch-all list, and `/prototype/{id...}` is an explicit route registered before the SPA fallback — verified against `embed.go`).

## Global Constraints (build/ordering)

- The two Go routes live in the existing `http` package next to `mnemonic_files.go`; they open the project via `s.openHandleFor(projectID)` and use the existing `h.Memory()` service (save/search) and store (`h.Store().DB`) — no new service.
- The answer route's "set state + append version" must go through the **same code path** `mem_update` uses (so the version history is identical to any other update). Prefer calling the existing service method that backs `mem_update`; if none exposes "update content only," add a small `Memory().UpdateContent(id, newContentJSON)` in `memory/governance.go` (it already owns `SetStatus`/`SetVisibility` and appends versions — see `observation_versions`).
- The UI build (`task ui:build` / `npm run build:check`) runs before `go build` so `ui/dist` (now with the decisions chunk) is embedded.
- The agent-side contract (how the skill calls `mem_save` for a decision) is documented in `docs/user-guide/10-decision-companion.md` + injected into the brainstorming/interviewing skills as a convention note; **no skill file is edited in this change** beyond that doc note (skills are a separate repo concern).

---

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

### Task 2: `POST /mnemonic/decisions/{id}/answer` write route (state flip + version append)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/decisions.go` (add `handleDecisionAnswer`)
- Modify: `skillgrid-cli/internal/mnemonic/memory/governance.go` (add `UpdateContent(id, newContentJSON)` if no existing "update content + append version" method is exposed by `mem_update`'s backend)
- Modify: `skillgrid-cli/internal/mnemonic/http/decisions_test.go`

**Interfaces:**
- Consumes: `requireWriteAuth`, `s.openHandleFor(projectID)`, `h.Memory()` (governance service), `decodeJSON`.
- Produces:
  - `func (s *Server) handleDecisionAnswer(w http.ResponseWriter, r *http.Request)`
  - (if added) `func (s *Service) UpdateContent(ctx context.Context, id int64, newContent string) error` in `memory/governance.go` — sets `observations.content`, appends an `observation_versions` row, bumps `revision_count`, sets `updated_at`.

**SATISFIES:** scenarios `decision-answer-records` + `decision-answer-appends-version` + `decision-answer-requires-auth`.

- [ ] **Step 1: Write the failing test**

```go
func TestDecisionAnswerRecords(t *testing.T) {
	s, h := newDecisionTestServer(t)
	id := seedDecision(t, h, "interview/demo/layout-1",
		`{"question":"Which layout?","options":[{"id":"a","label":"A"},{"id":"b","label":"B"}],"recommended":"a","state":"pending"}`, "team")

	body := `{"optionId":"b","note":"two column reads better"}`
	req := httptest.NewRequest("POST", "/mnemonic/decisions/"+itoa(id)+"/answer?project=testproj", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("answer: got %d: %s", w.Code, w.Body.String()) }

	// read back
	req = httptest.NewRequest("GET", "/mnemonic/decisions?project=testproj", nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	var out []decisionRow
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	var got *decisionRow
	for i := range out { if out[i].ID == id { got = &out[i] } }
	if got == nil { t.Fatalf("decision not found after answer") }
	if got.Content.State != "answered" { t.Errorf("state not flipped: %q", got.Content.State) }
	if got.Content.AnsweredOption != "b" || got.Content.AnswerNote != "two column reads better" {
		t.Errorf("answer not recorded: %+v", got.Content)
	}
}

func TestDecisionAnswerAppendsVersion(t *testing.T) {
	s, h := newDecisionTestServer(t)
	id := seedDecision(t, h, "interview/demo/v", `{"question":"q","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team")
	post := func(note string) {
		body := `{"optionId":"a","note":"` + note + `"}`
		req := httptest.NewRequest("POST", "/mnemonic/decisions/"+itoa(id)+"/answer?project=testproj", strings.NewReader(body))
		s.Handler().ServeHTTP(httptest.NewRecorder(), req)
	}
	post("first")
	post("second")
	var n int
	_ = h.Store().DB.QueryRow(`SELECT COUNT(*) FROM observation_versions WHERE observation_id = ?`, id).Scan(&n)
	if n < 2 { t.Errorf("expected >=2 version rows, got %d", n) }
}

func TestDecisionAnswerRequiresAuth(t *testing.T) {
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret")
	s, h := newDecisionTestServer(t) // rebuilt after env set so NewServer picks up the token
	id := seedDecision(t, h, "interview/demo/auth", `{"question":"q","options":[{"id":"a","label":"A"}],"state":"pending"}`, "team")
	req := httptest.NewRequest("POST", "/mnemonic/decisions/"+itoa(id)+"/answer?project=testproj",
		strings.NewReader(`{"optionId":"a"}`))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 401 { t.Errorf("no-auth answer with token set: got %d want 401", w.Code) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestDecisionAnswer' -v`
Expected: FAIL — `handleDecisionAnswer` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// decisions.go — add
type decisionAnswerBody struct {
	OptionID string `json:"optionId"`
	Note     string `json:"note"`
}

func (s *Server) handleDecisionAnswer(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil { writeError(w, http.StatusBadRequest, err.Error()); return }
	id64, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil { writeError(w, http.StatusBadRequest, "id must be an integer"); return }
	var in decisionAnswerBody
	if err := decodeJSON(r, &in); err != nil { writeError(w, http.StatusBadRequest, err.Error()); return }
	if in.OptionID == "" { writeError(w, http.StatusBadRequest, "optionId is required"); return }

	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil { writeError(w, http.StatusNotFound, err.Error()); return }
	defer cleanup()

	// Load the current decision, flip state, record the answer.
	var content string
	if err := h.Store().DB.QueryRowContext(r.Context(),
		`SELECT content FROM observations WHERE id = ? AND project = ? AND type='decision' AND deleted_at IS NULL`,
		id64, projectID).Scan(&content); err != nil {
		writeError(w, http.StatusNotFound, "no such decision")
		return
	}
	c, ok := parseDecisionContent(content)
	if !ok { writeError(w, http.StatusBadRequest, "decision content unreadable"); return }
	// validate the option exists
	valid := false
	for _, o := range c.Options { if o.ID == in.OptionID { valid = true } }
	if !valid { writeError(w, http.StatusUnprocessableEntity, "optionId not in this decision's options"); return }
	c.State = "answered"
	c.AnsweredOption = in.OptionID
	c.AnswerNote = in.Note
	c.UpdatedBy = "user"
	newContent, _ := json.Marshal(c)

	// Persist through the governance path so a version is appended.
	if err := h.Memory().UpdateContent(r.Context(), id64, string(newContent)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "state": c.State})
}
```

In `memory/governance.go`, add (mirroring `SetStatus`'s shape, but for `content` + version append):

```go
// UpdateContent sets observations.content for a row and appends an
// observation_versions history entry (the same path mem_update uses), bumping
// revision_count and updated_at. Used by the decisions answer route so the
// approval trail is append-only and identical to any other update.
func (s *Service) UpdateContent(ctx context.Context, id int64, newContent string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil { return fmt.Errorf("update content: begin: %w", err) }
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`UPDATE observations SET content = ?, updated_at = ?, revision_count = revision_count + 1
		 WHERE id = ? AND project = ? AND deleted_at IS NULL`, newContent, now, id, s.projectID); err != nil {
		return fmt.Errorf("update content: %w", err)
	}
	if n, _ := (func() (int64, error) {
		return tx.QueryRowContext(ctx,
			`SELECT changes() FROM observations WHERE id = ?`, id).Scan(&n)
	})(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO observation_versions (observation_id, content, created_at) VALUES (?,?,?)`,
		id, newContent, now); err != nil {
		return fmt.Errorf("update content: version: %w", err)
	}
	return tx.Commit()
}
```

> Confirm the exact `observation_versions` columns against `migrations/017_layered_memory_governance.sql` before writing the INSERT (the schema test in `governance_schema_test.go` shows the table exists; match its real column names — `observation_id`/`content`/`created_at` are the expected shape, verify and adjust).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestDecisionAnswer' -v -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/decisions.go skillgrid-cli/internal/mnemonic/http/decisions_test.go skillgrid-cli/internal/mnemonic/memory/governance.go
git commit -m "feat(decisions): answer route — state flip + append-only version"
```

---

### Task 3: React `/decisions` view (decision inbox + answer round-trip)

**Files:**
- Create: `skillgrid-ui/src/features/decisions/api.ts`
- Create: `skillgrid-ui/src/features/decisions/DecisionsPage.tsx`
- Create: `skillgrid-ui/src/features/decisions/DecisionCard.tsx`
- Modify: `skillgrid-ui/src/app.tsx` (add `decisionsRoute`)
- Modify: `skillgrid-ui/src/components/layout/AppLayout.tsx` (add nav entry)

**Interfaces:**
- Consumes: `GET /mnemonic/decisions?project=…&state=…`, `POST /mnemonic/decisions/{id}/answer?project=…`; `currentProjectName` from `../../lib/projects`.
- Produces: `DecisionsPage` (default-exported route component); `decisionsApi` with `fetchDecisions(project, state)`, `answerDecision(project, id, {optionId, note})`.

**SATISFIES:** scenarios `decision-view-renders` + `decision-view-roundtrip`.

- [ ] **Step 1: Write the failing test**

```tsx
// skillgrid-ui/src/features/decisions/DecisionCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { DecisionCard } from './DecisionCard'

const d = {
  id: 1, topicKey: 'interview/demo/layout-1', title: 'Layout',
  createdAt: '', updatedAt: '', visibility: 'team', parseError: false,
  content: {
    question: 'Which layout?',
    options: [{ id: 'a', label: 'Single column' }, { id: 'b', label: 'Two column' }],
    recommended: 'a', state: 'pending',
  },
}

describe('DecisionCard', () => {
  it('renders options with the recommended one highlighted', () => {
    render(<DecisionCard decision={d as any} onAnswer={vi.fn()} />)
    expect(screen.getByText('Which layout?')).toBeTruthy()
    const rec = screen.getByTestId('decision-option-a')
    expect(rec.className).toContain('recommended')
  })
  it('posts the chosen option on click', () => {
    const onAnswer = vi.fn()
    render(<DecisionCard decision={d as any} onAnswer={onAnswer} />)
    fireEvent.click(screen.getByTestId('decision-option-b'))
    expect(onAnswer).toHaveBeenCalledWith(1, { optionId: 'b', note: '' })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-ui && npm test -- DecisionCard`
Expected: FAIL — module not found.

- [ ] **Step 3: Write minimal implementation**

```tsx
// DecisionCard.tsx
import { useState } from 'react'

export interface DecisionOption { id: string; label: string; rationale?: string }
export interface Decision {
  id: number; topicKey: string; title: string; createdAt: string; updatedAt: string
  visibility: string; parseError?: boolean
  content: { question: string; options: DecisionOption[]; recommended?: string; state: string; answeredOption?: string; answerNote?: string }
}

export function DecisionCard({ decision, onAnswer }: { decision: Decision; onAnswer: (id: number, a: { optionId: string; note: string }) => void }) {
  const [note, setNote] = useState('')
  const answered = decision.content.state === 'answered'
  if (decision.parseError) {
    return <div className="rounded-md border border-warn bg-warn-soft p-3 text-sm">Unreadable decision: {decision.topicKey}</div>
  }
  return (
    <div className="rounded-md border border-edge bg-panel p-4">
      <div className="mb-1 text-xs uppercase tracking-wide text-ink-3">{decision.topicKey}</div>
      <h3 className="mb-3 font-semibold">{decision.content.question}</h3>
      <div className="flex flex-col gap-2">
        {decision.content.options.map((o) => {
          const isRec = o.id === decision.content.recommended
          const isPicked = answered && o.id === decision.content.answeredOption
          return (
            <button
              key={o.id}
              data-testid={`decision-option-${o.id}`}
              disabled={answered}
              onClick={() => onAnswer(decision.id, { optionId: o.id, note })}
              className={[
                'rounded border p-2 text-left text-sm transition',
                isRec ? 'border-accent bg-accent-soft recommended' : 'border-edge hover:bg-background',
                isPicked ? 'border-ok bg-ok-soft' : '',
              ].join(' ')}
            >
              <span className="font-medium">{o.label}</span>
              {isRec && !answered && <span className="ml-2 text-xs text-accent">recommended</span>}
              {isPicked && <span className="ml-2 text-xs text-ok">your choice</span>}
              {o.rationale && <div className="mt-1 text-xs text-ink-2">{o.rationale}</div>}
            </button>
          )
        })}
      </div>
      {!answered && (
        <input
          className="mt-3 w-full rounded border border-edge bg-background px-2 py-1 text-sm"
          placeholder="Optional note for the agent…"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      )}
      {answered && decision.content.answerNote && (
        <div className="mt-2 text-xs text-ink-2">Note: {decision.content.answerNote}</div>
      )}
    </div>
  )
}
```

```tsx
// DecisionsPage.tsx
import { useCallback, useEffect, useState } from 'react'
import { DecisionCard, type Decision } from './DecisionCard'
import * as api from './api'

export function DecisionsPage() {
  const [tab, setTab] = useState<'pending' | 'answered' | 'all'>('pending')
  const [rows, setRows] = useState<Decision[]>([])
  const [err, setErr] = useState<string | null>(null)

  const load = useCallback(async () => {
    try { setRows(await api.fetchDecisions(tab === 'all' ? '' : tab)) }
    catch (e) { setErr(String(e)) }
  }, [tab])

  useEffect(() => { void load(); const t = setInterval(load, 4000); return () => clearInterval(t) }, [load])

  const onAnswer = useCallback(async (id: number, a: { optionId: string; note: string }) => {
    await api.answerDecision(id, a); void load()
  }, [load])

  return (
    <div className="flex h-full flex-col gap-3 overflow-y-auto p-6">
      <div className="flex gap-1 text-sm">
        {(['pending', 'answered', 'all'] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)}
            className={'rounded px-3 py-1 ' + (tab === t ? 'bg-accent-soft text-accent' : 'text-ink-2 hover:bg-background')}>{t}</button>
        ))}
      </div>
      {err && <div className="text-error">{err}</div>}
      {rows.length === 0 && <div className="text-ink-3">No {tab} decisions.</div>}
      <div className="flex flex-col gap-3">
        {rows.map((d) => <DecisionCard key={d.id} decision={d} onAnswer={onAnswer} />)}
      </div>
    </div>
  )
}
```

```ts
// api.ts
import { currentProjectName } from '../../lib/projects'
import type { Decision } from './DecisionCard'

let project: string | null = null
async function resolveProject(): Promise<string> {
  if (project) return project
  project = await currentProjectName()
  return project
}

export async function fetchDecisions(state: string): Promise<Decision[]> {
  const p = await resolveProject()
  const q = new URLSearchParams({ project: p }); if (state) q.set('state', state)
  const res = await fetch(`/mnemonic/decisions?${q}`)
  if (!res.ok) throw new Error(`decisions ${res.status}`)
  return res.json()
}

export async function answerDecision(id: number, a: { optionId: string; note: string }): Promise<void> {
  const p = await resolveProject()
  const res = await fetch(`/mnemonic/decisions/${id}/answer?project=${encodeURIComponent(p)}`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(a),
  })
  if (!res.ok) throw new Error(`answer ${res.status}`)
}
```

Wire route + nav (mirror `activityRoute` / `STANDALONE_ITEMS`):

```tsx
// app.tsx
const decisionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/decisions',
  component: lazyPage(() => import('./features/decisions/DecisionsPage').then((m) => ({ default: m.DecisionsPage }))),
})
// add decisionsRoute to routeTree.children
// AppLayout.tsx STANDALONE_ITEMS: { to: '/decisions', label: 'Decisions' },
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-ui && npm test -- DecisionCard && npm run build:check`
Expected: PASS (tests + bundle budget).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-ui/src/features/decisions/ skillgrid-ui/src/app.tsx skillgrid-ui/src/components/layout/AppLayout.tsx
git commit -m "feat(ui): decisions view — decision inbox + answer round-trip (polling)"
```

---

### Task 4: openapi + docs convention + MCP round-trip guard

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml` (add `/mnemonic/decisions` + `/mnemonic/decisions/{id}/answer`)
- Create: `docs/user-guide/10-decision-companion.md` (content schema + the poll loop)
- Create: `skillgrid-cli/internal/mnemonic/http/usermanual_decisions_test.go` (doc-existence guard, mirror `usermanual_phase7_test.go`)
- Create: `skillgrid-cli/internal/mnemonic/mcp/decision_roundtrip_test.go` (MCP round-trip guard)

**Interfaces:**
- Consumes: existing `mem_save`/`mem_search` MCP tools (frozen).
- Produces: openapi entries; `TestDecisionsUserManual`; `TestDecisionMCPRoundTrip`.

**SATISFIES:** scenarios `decision-docs-exist` + `decision-mcp-roundtrip`.

- [ ] **Step 1: Write the failing tests**

```go
// usermanual_decisions_test.go
func TestDecisionsUserManual(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docs", "user-guide", "10-decision-companion.md"))
	if err != nil { t.Fatalf("missing decision-companion doc: %v", err) }
	for _, marker := range []string{"type: decision", "state: pending", "recommended", "mem_save", "GET /mnemonic/decisions"} {
		if !strings.Contains(string(data), marker) { t.Errorf("doc missing marker %q", marker) }
	}
}

// decision_roundtrip_test.go (in mcp pkg) — save a decision via mem_save, search it back, parse content.
func TestDecisionMCPRoundTrip(t *testing.T) {
	// (adapt to the package's existing MCP test harness — see single_open_test.go / e2e_memory_ext_test.go)
	// 1. mem_save{type:decision, topic_key:interview/demo/rt, visibility:team, content:`{"question":"Q","options":[{"id":"a","label":"A"}],"recommended":"a","state":"pending"}`}
	// 2. mem_search{query or topic_key} → find the row
	// 3. parse content JSON → assert question/options/recommended/state present
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run TestDecisionsUserManual -v && go test ./internal/mnemonic/mcp/ -run TestDecisionMCPRoundTrip -v`
Expected: FAIL — doc missing, MCP test not written.

- [ ] **Step 3: Write implementation (doc + openapi + MCP test)**

`docs/user-guide/10-decision-companion.md` — the convention:

````markdown
# Decisions — the Mnemonic decision companion

The companion is a **view over `type=decision` Mnemonic observations**. The agent
posts interview questions; the dashboard renders them; the user answers; the agent
reads the answer back. No WebSocket, no second process, no new store — polling is the
transport.

## The decision content schema (JSON in `observations.content`)

```json
{
  "question": "Which layout for the homepage?",
  "options": [
    { "id": "a", "label": "Single column", "rationale": "cleanest reading" },
    { "id": "b", "label": "Two column" }
  ],
  "recommended": "a",
  "state": "pending"
}
```

Fields:
- `question` (required) — the question, in domain language.
- `options[]` (required, ≥1) — `id` (stable, `a`/`b`/…), `label`, optional `rationale`.
- `recommended` (optional) — the id of the option the agent recommends (highlighted in the UI).
- `state` (required) — `pending` | `answered` | `superseded`. **This is a content field, not the governance `status`** (`status` is reserved for `active|superseded|archived`).
- `answeredOption`, `answerNote` — set by the user's answer (the dashboard writes these).
- `updatedBy` — `agent` or `user`.
- `visual` (optional) — the prototype id `<topic>/<variant>.html`, served from `.skillgrid/prototype/` (see below). When present, the decision card renders it in a sandboxed iframe.

## The loop (polling, not realtime)

1. **Agent → user:** `mem_save` an observation: `type: decision`, `topic_key: interview/<slug>/<question-id>`, `visibility: team`, `content: {…state:pending}`. Reusing the same `topic_key` **upserts** the row (in-place question update; bumps the version history).
2. **Dashboard** polls `GET /mnemonic/decisions?state=pending` and renders the cards.
3. **User → agent:** click an option (or add a note) → `POST /mnemonic/decisions/{id}/answer {optionId, note}` → sets `state: answered`, records the choice, **appends a version** (durable, auditable trail).
4. **Agent** polls `mem_search` on the decision's `topic_key` (or the activity feed) and proceeds when `content.state == answered`.

## Prototypes (the visual function)

When a question is clearer *shown* than told, the agent writes throwaway standalone
interactive HTML to `.skillgrid/prototype/<topic>/<variant>.html` (via the existing `sketch`
skill, pointed at this dir for the companion flow) and sets `content.visual` to
`<topic>/<variant>.html`. The dashboard serves it at `GET /prototype/<topic>/<variant>.html`
(sandboxed to `.skillgrid/prototype/`, traversal-guarded) and renders it in a sandboxed
iframe (`sandbox="allow-scripts"`, no `allow-same-origin`) inside the decision card. The
prototype is throwaway by design; the durable record is the decision + its version history.
`sketch.dir` (the skill's default `{specs_root}/{topic}/sketches/`) is for standalone sketch
work; the companion uses `.skillgrid/prototype/` so the dashboard can serve it.

## Durability

Everything is in the Mnemonic SQLite store: it survives a server restart, is
versioned (`observation_versions`), owned, and governed (`visibility`, `acl_grants`).
A resumed session reads the same decisions.
````

Add the two openapi paths (mirror the `/mnemonic/memories/{id}/status` shape) and write the MCP round-trip test.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestDecision' -v && go test ./internal/mnemonic/mcp/ -run TestDecisionMCPRoundTrip -v`
Expected: PASS.

- [ ] **Step 5: Run full suite + build + UI build:check**

Run: `cd skillgrid-cli && go build ./... && go test ./... 2>&1 | tail -20 && cd ../skillgrid-ui && npm run build:check`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml skillgrid-cli/internal/mnemonic/http/usermanual_decisions_test.go skillgrid-cli/internal/mnemonic/mcp/decision_roundtrip_test.go docs/user-guide/10-decision-companion.md
git commit -m "docs(decisions): openapi + user-guide convention + MCP round-trip guard"
```

---

### Task 5: Prototype serve route + sandboxed iframe in the decision card (the visual function)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/prototype.go`
- Create: `skillgrid-cli/internal/mnemonic/http/prototype_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (register `GET /prototype/{id...}`)
- Modify: `skillgrid-ui/src/features/decisions/DecisionCard.tsx` (render `content.visual` in the sandboxed iframe)
- Modify: `skillgrid-ui/src/features/decisions/DecisionCard.test.tsx` (iframe render test)

**Interfaces:**
- Consumes: `sddRoot()` (existing, `plans.go`); the `stitchFile` guard *shape* (existing, `prototypes.go`) — replicated for the `.skillgrid/prototype/` root; `SandboxPreview` (existing, `features/prototypes/SandboxPreview.tsx`) or a minimal inline `<iframe sandbox="allow-scripts">`.
- Produces:
  - `func prototypeRoot() string` — `filepath.Join(sddRoot(), ".skillgrid", "prototype")`
  - `func prototypeFile(id string) (string, bool)` — the `stitchFile` guard shape, root = `prototypeRoot()`.
  - `func (s *Server) handlePrototypeDecision(w http.ResponseWriter, r *http.Request)` — `GET /prototype/{id...}`.

**SATISFIES:** scenarios `prototype-serve-traversal` + `decision-card-renders-visual`.

- [ ] **Step 1: Write the failing test**

```go
// prototype_test.go
package http

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrototypeServeAndTraversal(t *testing.T) {
	s := newTestServerForCwd(t, tempCwd(t)) // sets SKILLGRID_DOCS_CWD to a temp dir (see plans_test.go pattern)
	dir := filepath.Join(sddRoot(), ".skillgrid", "prototype", "demo")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "a.html"), []byte("<h1>variant A</h1>"), 0o644)

	// good id
	req := httptest.NewRequest("GET", "/prototype/demo/a.html", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "variant A") {
		t.Fatalf("serve: got %d %q", w.Code, w.Body.String())
	}

	// traversal / absolute ids → 400
	for _, id := range []string{"../a.html", "..%2F..%2Fa.html", "/etc/passwd", "a/../../x.html"} {
		req = httptest.NewRequest("GET", "/prototype/"+id, nil)
		w = httptest.NewRecorder()
		s.Handler().ServeHTTP(w, req)
		if w.Code != 400 { t.Errorf("id %q: got %d want 400", id, w.Code) }
	}
	// unknown file → 404
	req = httptest.NewRequest("GET", "/prototype/demo/nope.html", nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 404 { t.Errorf("unknown: got %d want 404", w.Code) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run TestPrototypeServe -v`
Expected: FAIL — `handlePrototypeDecision`, `prototypeFile` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// prototype.go
package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// prototypeRoot returns the .skillgrid/prototype/ directory (under sddRoot,
// the repo root). This is the on-disk store for the companion's throwaway
// interactive prototypes (agent-authored HTML), separate from .stitch/.
func prototypeRoot() string { return filepath.Join(sddRoot(), ".skillgrid", "prototype") }

// prototypeFile resolves a prototype id to an absolute path sandboxed to
// .skillgrid/prototype/ — the same path-traversal guard shape as stitchFile
// (prototypes.go). ok=false for empty/absolute ids, '..' segments, or any path
// that escapes the root after filepath.Clean.
func prototypeFile(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}
	if filepath.IsAbs(id) || id[0] == '/' || (len(id) >= 2 && id[1] == ':') {
		return "", false
	}
	root := prototypeRoot()
	full := filepath.Join(root, filepath.FromSlash(id))
	cleanFull := filepath.Clean(full)
	cleanRoot := filepath.Clean(root)
	rel, err := filepath.Rel(cleanRoot, cleanFull)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return cleanFull, true
}

// handlePrototypeDecision serves GET /prototype/{id...} — the HTML of one
// companion prototype, sandboxed to .skillgrid/prototype/. Traversal/absolute
// ids → 400; unknown → 404. Rendered client-side in a sandboxed iframe
// (sandbox="allow-scripts", no allow-same-origin).
func (s *Server) handlePrototypeDecision(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	full, ok := prototypeFile(id)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid prototype id: "+id)
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "unknown prototype: "+id)
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
```

Register in `registerRoutes()` (next to the existing `s.mux.HandleFunc("GET /prototypes/…")` lines):
```go
s.mux.HandleFunc("GET /prototype/{id...}", s.handlePrototypeDecision)
```
> Note: `/prototype/{id...}` is a distinct subtree from the existing `/prototypes` (plural) route — Go 1.22 mux treats them separately. Register it before `registerUIRoutes()` so the SPA fallback doesn't catch it.

In `DecisionCard.tsx`, render the visual when present (reuse the sandboxed-iframe pattern from `SandboxPreview.tsx`):
```tsx
{decision.content.visual && (
  <div className="mt-3 overflow-hidden rounded-md border border-edge">
    <iframe
      src={`/prototype/${decision.content.visual}`}
      title={`Prototype: ${decision.content.visual}`}
      sandbox="allow-scripts"
      className="h-[420px] w-full bg-white"
    />
  </div>
)}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestPrototype' -v -race && cd ../skillgrid-ui && npm test -- DecisionCard`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/prototype.go skillgrid-cli/internal/mnemonic/http/prototype_test.go skillgrid-cli/internal/mnemonic/http/server.go skillgrid-ui/src/features/decisions/
git commit -m "feat(decisions): prototype serve route (.skillgrid/prototype) + sandboxed iframe in the card"
```

---

## Self-Review

1. **Spec coverage:** every capability (agent posts decision, dashboard reads, user answers, agent reads back, durable/versioned, visibility-gated, write-authed, **agent authors + dashboard renders a prototype**, doc'd) has a task. The deferred `decisions`-table sub-decision is explicitly *not* a task. ✔
2. **Must-haves coverage:** truths 1–11 map to Tasks 1–5; `backstop` truths (1, 4, 9) get held-out tests — Task 1 live render (door check), Task 4 `TestDecisionMCPRoundTrip`, and a manual restart check during the door check. Truths 10–11 (prototype serve + sandboxed render) are covered by Task 5's `TestPrototypeServeAndTraversal` + the `DecisionCard` iframe test. Artifacts + key links listed. ✔
3. **One-way-door completeness:** none — no migration/table/contract break; listed as "None". ✔
4. **Placeholder scan:** no TBD/TODO/"similar to Task N". Two spots flag "confirm against existing code" — `observation_versions` column names (Task 2) and the MCP test harness shape (Task 4) — these are explicit verification steps, not holes, because the schema/harness already exist and the executor must match them. ✔
5. **Type consistency:** `decisionContent`/`decisionOption`/`decisionRow`/`decisionAnswerBody` defined once (Task 1) and reused in Tasks 2–3; `handleDecisions`/`handleDecisionAnswer`/`parseDecisionContent`/`UpdateContent` names consistent; UI `Decision`/`DecisionOption` mirror the Go JSON. ✔

## Plan Review

- Verdict: **READY FOR EXECUTION**
- Findings: 0 Critical, 1 Important (Task 2 must verify the exact `observation_versions` column names against `migrations/017_layered_memory_governance.sql` before the INSERT — a wrong column name is a silent runtime failure, so the test in Task 2 is the gate), 3 Minor (deferred: the MCP round-trip test reuses the existing harness in `single_open_test.go`/`e2e_memory_ext_test.go`; the visibility filter in Task 1 must be matched verbatim to `handleMnemonicMemories` so the reader predicate doesn't drift; Task 5's `prototypeFile` guard is a deliberate copy of `stitchFile`'s shape for a different root — keep them in lockstep if one changes).
- Reviewed: 2026-09-19

## Execution Handoff

Blueprint has 5 tasks → invoke `skillgrid:slicing` to break into vertical tracer-bullet tickets with execution waves, producing `tasks.md` alongside this blueprint.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task + two-stage review.
2. **Inline Execution** — `skillgrid:simple-execution`, batch with checkpoints.

**Door check first:** execute Task 1 alone and confirm, on a live `skillgrid serve`, that an agent `mem_save` of a `type=decision` observation appears in `GET /mnemonic/decisions`. If it doesn't parse back out, stop.
