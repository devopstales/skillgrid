# Embed Visual Companion into skillgrid-cli Dashboard — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fold the standalone Node `brainstorming` visual companion (file-watcher + HTTP + WebSocket) into the existing `skillgrid serve` Go binary so it is one more dashboard view, served from the same loopback server, with no second process.

**Architecture:** Port the companion's transport (HTTP + RFC-6455 WebSocket) and presentation (frame-template + helper.js) into the Go `http` package as a new route group `/companion/*`. The frame and helper are embedded with `go:embed` and reused verbatim (presentation is shared). The Node server's lifecycle (fs.watch, owner-PID watchdog, idle timeout, browser open, port/token persistence) is replaced by the already-running Go server's lifecycle: a per-session in-memory screen store (written by an explicit `PUT`), a `fsnotify` watcher as a backstop, and the Go `ServeMux`'s existing lifecycle. The React SPA gains a lazy `/companion` route + nav entry + a React frame view that renders the active screen in an iframe (sandboxed) and posts click/choice events back over the same-origin REST stream.

**Tech Stack:** Go 1.22+ (`skillgrid-cli`); existing `internal/mnemonic/http` Go server + `go:embed` + Go 1.22 `http.ServeMux`; `golang.org/x/net/websocket` (transport only); React 19 + TypeScript + Vite 6 (`skillgrid-ui`) for the dashboard view; `fsnotify` (already a dependency via the auto-sync watcher).

**Spec:** `.skillgrid/specs/2026-09-19-embed-visual-companion/briefing.md`

**Findings:** (none — this is a port/refactor of known behavior, no research/spike/sketch ran)

## Global Constraints

- **Loopback only.** The companion inherits `skillgrid serve`'s `127.0.0.1` bind. No new host/port flags for the companion itself; it lives under the existing server's port.
- **No Node runtime at serve time.** The Go binary must not shell out to `node` or spawn `server.cjs`. The Node launcher (`start-server.sh`) becomes a *fallback* only for the in-terminal agent flow, not the dashboard path.
- **Presentation parity.** `frame-template.html` and `helper.js` must be reused **verbatim** (they are the visual contract). Only the transport changes. Do not re-style the frame.
- **Sandboxed rendering.** The companion screen HTML is rendered inside an iframe with the `sandbox` attribute (no `allow-same-origin`) so screen scripts cannot read the dashboard's cookies/localStorage — same trust rule as the `.stitch/` Prototypes view.
- **Session key preserved.** The `?key=` session-secret gate (defeats DNS rebinding / stray tabs) is preserved on the `/companion/*` routes. It is per-session, distinct from `SKILLGRID_HTTP_TOKEN`.
- **MCP surface is frozen.** No MCP tool names, signatures, or return shapes change. This is HTTP + UI only.
- **Bundle budget.** The new React chunk must pass `npm run build:check` (the existing `scripts/check-bundle-size.mjs` gate). Keep the companion view in its own lazy chunk.
- **TDD is the mode.** Every Go handler and every UI behavior has a RED test first (config `tdd: false` default → Standard; this blueprint is a new capability so follow the RED-GREEN cycle per task).

## Hypothesis

**Claim:** The visual companion can be served entirely by the existing `skillgrid serve` Go process — one new `/companion/*` route group plus a lazy React view — with zero behavioral loss versus the standalone Node companion (screen push → browser update, click/choice capture → agent-readable events, session-key gate, frame reuse).

**Right condition:** With only `skillgrid serve` running (no `node server.cjs`), a screen written via `PUT /companion/sessions/{id}/screens` appears in the dashboard's `/companion` view; clicking an option produces a JSON event readable at `GET /companion/sessions/{id}/events`; a request without the session key is rejected 403; the frame template and helper behave identically to the Node version.

**Wrong condition:** Any of the above requires a second process, a re-style of the frame, or a behavior the Go server cannot reproduce (e.g. the browser-open or reconnect behavior is lost with no equivalent).

**Thinnest MVP:** Task 1 (screen store + `GET /companion/session/{id}/screen` + `PUT` + `403` on bad key) and Task 4 (React view loads the screen in a sandboxed iframe). If the iframe can't render the frame, or the key gate can't be enforced same-origin, the approach is invalidated.

**Door check:** Task 1 is the door check — if the Go server cannot serve a wrapped screen with a session-key gate, stop and reconsider (the whole port rests on this).

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Screen HTML is untrusted content** (agent-authored, rendered in browser) | Applicable | Render inside `<iframe sandbox>` (no `allow-same-origin`); serve the frame as same-origin but the *screen content* is isolated. `Content-Security-Policy: frame-ancestors 'self'` on the companion routes. | `TestCompanionScreenSandboxed` — response headers assert `sandbox` + CSP; `TestCompanionScreenNoSameOrigin` — screen cannot read parent `localStorage` (unit: frame wrapper does not add `allow-same-origin`). |
| **Path traversal in screen/asset names** | Applicable | Screen store keys are server-generated ids (not user path input); the screen *content* is opaque bytes, not a filesystem path. Any `/companion/session/{id}/files/{name}` is basename + inside-session-dir checked (mirror `isRegularFileInsideContentDir`). | `TestCompanionFileTraversal` — `../` and absolute names are rejected 404. |
| **WebSocket upgrade on Go 1.22 mux** | Applicable | Register the upgrade under the exact `GET /companion/session/{id}/ws` route; `golang.org/x/net/websocket` handles the handshake; a non-WebSocket GET on the same path falls through to the key-gate + 426. Origin allowlist = same-origin (mirror `isAllowedWebSocketOrigin`). | `TestCompanionWSHandshake` — WS upgrade on correct key succeeds (101); wrong key is not upgraded (no 101). `TestCompanionWSOrigin` — cross-origin Origin is rejected. |
| **Session key leak / stale token** | Applicable | Key is per-session, `crypto/rand` 32 bytes, compared with `crypto/subtle` timing-safe equality; stored in-memory (server lifetime) — not persisted to disk by default. `server-info` (if surfaced) carries the key → owner-only. | `TestCompanionKeyTimingSafe` — `subtle.ConstantTimeCompare` used (vet/inspection); `TestCompanionKeyRejected` — wrong/absent key → 403 on every `/companion/*` route. |
| **Concurrent screen writes / event appends** | Applicable | Per-session `sync.RWMutex`; event append is an atomic append to an in-memory ring (bounded) + optional JSONL spill for the agent to read. | `TestCompanionConcurrentScreen` — parallel PUTs don't corrupt the active screen. `TestCompanionEventRingBounded` — ring evicts oldest past cap. |
| **SSE/WS client disconnect leaks goroutines** | Applicable | Every stream/WS handler exits on `r.Context().Done()`; client channel is buffered with a drop (mirror `activityStreamClient`). No goroutine outlives the request. | `TestCompanionStreamLeak` (with `-race`) — context cancel stops the handler; no goroutine leak. |
| **Idle/orphaned sessions accumulate memory** | Applicable | Reuse the server's existing lifecycle: a periodic reaper (mirror `LIFECYCLE_CHECK_MS`) evicts sessions idle > N (default 4h) or with no connected client > M. Configurable. | `TestCompanionSessionReaper` — an idle session is evicted after the (shortened in test) timeout. |
| **Subprocess / shell commands** | N/A: the Go path spawns no subprocess; the Node `start-server.sh` launcher is the pre-existing fallback and unchanged. | n/a | n/a |
| **Mnemonic tool contract** | N/A: MCP surface is frozen; this adds HTTP routes + a UI view only. | n/a | n/a |

## One-Way-Door Checkpoints

- **None.** No migration, no published-contract break, no public API shape change. The MCP surface is frozen; this *adds* HTTP routes and a UI view. Reversal = delete the route group + the React view.

## Change Classification

- **`standard`** → verification floor **L2** (full `go test ./...` + `go build ./...` + `npm run build:check`). New trust boundary (untrusted HTML rendering) + WebSocket + concurrency push it to standard; no data migration or auth-model change, so not `risky`/`high-risk`.

## Must-Haves (Goal-Backward Verification)

**Truths:**
1. A screen written via `PUT /companion/sessions/{id}/screens` is returned by `GET /companion/sessions/{id}/screen` wrapped in the frame template (fragment) or served as-is (full doc). *backstop* (held-out: end-to-end via a live server + browser render).
2. A request to any `/companion/*` route without a valid session key returns **403** (key present in `?key=` or the session cookie).
3. A click/choice made in the companion frame is captured and returned by `GET /companion/sessions/{id}/events` as JSON (agent-readable), and cleared when a new screen is pushed.
4. The WebSocket at `/companion/session/{id}/ws` pushes a `reload` event to connected clients when a new screen is written.
5. The frame template + helper.js are byte-identical to the standalone companion's (`diff` clean) — presentation is shared, not re-implemented. *backstop*.
6. The dashboard's `/companion` view renders the active screen in a **sandboxed iframe** (no `allow-same-origin`) and round-trips clicks back to the agent stream.
7. A request with a cross-origin `Origin` on the WS upgrade is rejected (not upgraded).
8. An idle companion session is evicted after the reaper timeout (memory does not grow unbounded).

**Artifacts:**
- `skillgrid-cli/internal/mnemonic/http/companion.go` — route group + session store + handlers.
- `skillgrid-cli/internal/mnemonic/http/companion_ws.go` — WebSocket handler (or folded into companion.go if small).
- `skillgrid-cli/internal/mnemonic/http/companion_embed.go` — `//go:embed` of `frame-template.html` + `helper.js`.
- `skillgrid-cli/internal/mnemonic/http/companion_test.go` — the RED tests above.
- `skillgrid-ui/src/features/companion/` — `CompanionPage.tsx`, `api.ts`, `frame.tsx` (the React frame view + iframe).
- `ui/openapi.yaml` entries for `/companion/*`.
- Nav entry in `skillgrid-ui/src/components/layout/AppLayout.tsx` + route in `skillgrid-ui/src/app.tsx`.

**Key links:**
- `PUT /companion/sessions/{id}/screens` → in-memory store → `GET .../screen` (same session, same key).
- Frame `helper.js` click → `POST /companion/sessions/{id}/events` (or WS) → store → `GET .../events`.
- React `CompanionPage` → `GET .../screen` → iframe `srcDoc` → `POST .../events`.
- `registerRoutes()` → `s.registerCompanionRoutes()` (must run **before** `registerUIRoutes` so the SPA fallback doesn't swallow `/companion`).

## Global Constraints (build/ordering)

- `go:embed` of the frame requires the two assets to exist at `go build` time. Copy them into `internal/mnemonic/http/companion/` (a new subdir) during Task 2; the `//go:embed` directive points there. This mirrors how `ui/dist` is embedded.
- `golang.org/x/net` is already a dependency (check `go.mod`); if not present, `go get golang.org/x/net/websocket` is the only new dep.
- The UI build (`task ui:build` / `npm run build:check`) must run before `go build` so `ui/dist` (now including the companion chunk) is embedded.

---

### Task 1: Companion session store + screen GET/PUT + session-key gate (DOOR CHECK)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/companion.go`
- Create: `skillgrid-cli/internal/mnemonic/http/companion_test.go`
- Create: `skillgrid-cli/internal/mnemonic/http/companion/frame-template.html` (copied verbatim from `.agents/skills/brainstorming/scripts/frame-template.html`)
- Create: `skillgrid-cli/internal/mnemonic/http/companion/helper.js` (copied verbatim from `.agents/skills/brainstorming/scripts/helper.js`)
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go` (add `s.registerCompanionRoutes()` call before `s.registerUIRoutes()`)

**Interfaces:**
- Consumes: `projectFromRequest(r)`, `writeJSON(w, code, v)`, `writeError(w, code, msg)` (existing helpers in the `http` package).
- Produces:
  - `type companionSession struct { key string; mu sync.RWMutex; screen string; screens map[string]string; events []companionEvent; clients map[int64]chan []byte; lastActivity time.Time; id int64 }`
  - `type companionEvent struct { Type string; Choice string; Text string; ID string; Timestamp int64 }`
  - `type companionStore struct { mu sync.RWMutex; nextID int64; sessions map[int64]*companionSession }`
  - `func (s *Server) newCompanionStore() *companionStore` (wired into `*Server` in `NewServer`)
  - `func (s *Server) registerCompanionRoutes()` — registers the routes below.
  - Handlers: `handleCompanionSessionScreen` (GET), `handleCompanionSessionScreenPut` (PUT).
  - `func (s *Server) companionAuth(r *http.Request, sess *companionSession) bool` — timing-safe key check.

**SATISFIES:** scenario `companion-screen-roundtrip` + `companion-key-gate` (see `acceptance.feature`).

- [ ] **Step 1: Write the failing test**

```go
// companion_test.go
package http

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// newCompanionServer builds a handler with a fresh companion store and a
// pre-created session, returning the server + the session key.
func newCompanionServer(t *testing.T) (*Server, string) {
	t.Helper()
	svc, _ := newTestService(t) // existing test helper in this package
	s := NewServer(svc)
	sess, key := s.companionStore.createSession()
	t.Cleanup(func() { s.companionStore.removeSession(sess.id) })
	_ = key
	return s, key
}

func TestCompanionScreenRoundtrip(t *testing.T) {
	s, key := newCompanionServer(t)
	sessID := s.companionStore.sessionsByAny()[0]

	frag := `<h2>Which layout?</h2><div class="options"><div class="option" data-choice="a">A</div></div>`
	req := httptest.NewRequest("PUT", "/companion/sessions/1/screens?key="+key, strings.NewReader(frag))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("PUT screen: got %d want 200: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/companion/sessions/1/screen?key="+key, nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("GET screen: got %d want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Which layout?") {
		t.Errorf("screen body missing content: %s", body)
	}
	if !strings.Contains(body, "class=\"option\"") || !strings.Contains(body, "data-choice=\"a\"") {
		t.Errorf("frame not applied to fragment: %s", body)
	}
	// helper script must be injected
	if !strings.Contains(body, "brainstorm-session-key") {
		t.Errorf("helper.js not injected into frame")
	}
}

func TestCompanionKeyGate(t *testing.T) {
	s, _ := newCompanionServer(t)
	sessID := s.companionStore.sessionsByAny()[0]
	for _, path := range []string{
		"/companion/sessions/1/screen",
		"/companion/sessions/1/screen", // wrong key
		"/companion/sessions/1/events",
	} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, req)
		if w.Code != 403 {
			t.Errorf("%s without key: got %d want 403", path, w.Code)
		}
	}
	// wrong key also 403
	req := httptest.NewRequest("GET", "/companion/sessions/1/screen?key=wrong", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("wrong key: got %d want 403", w.Code)
	}
	_ = sessID
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestCompanionScreen|TestCompanionKey' -v`
Expected: FAIL — `companionStore`, `registerCompanionRoutes`, handlers undefined (compile error).

- [ ] **Step 3: Write minimal implementation**

```go
// companion.go
package http

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed companion/frame-template.html
var companionFrameTemplate string

//go:embed companion/helper.js
var companionHelperJS string

type companionEvent struct {
	Type      string `json:"type"`
	Choice    string `json:"choice,omitempty"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type companionSession struct {
	id         int64
	key        string
	mu         sync.RWMutex
	screen     string
	events     []companionEvent
	lastActive time.Time
}

type companionStore struct {
	mu       sync.RWMutex
	nextID   int64
	sessions map[int64]*companionSession
}

func newCompanionStore() *companionStore {
	return &companionStore{sessions: map[int64]*companionSession{}, nextID: 1}
}

func (st *companionStore) createSession() (*companionSession, string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	id := st.nextID
	st.nextID++
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	sess := &companionSession{id: id, key: hex.EncodeToString(raw), lastActive: time.Now()}
	st.sessions[id] = sess
	return sess, sess.key
}

func (st *companionStore) get(id int64) *companionSession {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.sessions[id]
}

func (st *companionStore) removeSession(id int64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.sessions, id)
}

func (st *companionStore) sessionsByAny() []int64 {
	st.mu.RLock()
	defer st.mu.RUnlock()
	out := make([]int64, 0, len(st.sessions))
	for id := range st.sessions {
		out = append(out, id)
	}
	return out
}

// wrapInFrame mirrors the Node server: full docs served as-is, fragments wrapped.
func wrapInFrame(content string) string {
	trimmed := strings.TrimLeft(content, " \t\r\n")
	low := strings.ToLower(trimmed)
	if strings.HasPrefix(low, "<!doctype") || strings.HasPrefix(low, "<html") {
		return content
	}
	framed := strings.Replace(companionFrameTemplate, "<!-- CONTENT -->", content, 1)
	injection := "\n<script>\n" + companionHelperJS + "\n</script>"
	if i := strings.Index(framed, "</body>"); i >= 0 {
		framed = framed[:i] + injection + framed[i:]
	} else {
		framed += injection
	}
	return framed
}

func (s *Server) registerCompanionRoutes() {
	s.mux.HandleFunc("GET /companion/sessions/{id}/screen", s.handleCompanionSessionScreen)
	s.mux.HandleFunc("PUT /companion/sessions/{id}/screens", s.handleCompanionSessionScreenPut)
	// events + ws added in later tasks
}

func (s *Server) companionAuth(r *http.Request, sess *companionSession) bool {
	q := r.URL.Query().Get("key")
	if q != "" {
		return subtle.ConstantTimeCompare([]byte(q), []byte(sess.key)) == 1
	}
	cookie, err := r.Cookie("companion-key-" + fmt.Sprint(sess.id))
	if err == nil {
		return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(sess.key)) == 1
	}
	return false
}

func (s *Server) handleCompanionSessionScreen(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCompanionID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad session id")
		return
	}
	sess := s.companionStore.get(id)
	if sess == nil {
		writeError(w, http.StatusNotFound, "no such companion session")
		return
	}
	if !s.companionAuth(r, sess) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<!doctype html><title>Session key required</title>"))
		return
	}
	sess.mu.RLock()
	screen := sess.screen
	sess.mu.RUnlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if screen == "" {
		_, _ = w.Write([]byte("<!doctype html><h1>Waiting for the agent to push a screen…</h1>"))
		return
	}
	_, _ = w.Write([]byte(wrapInFrame(screen)))
}

func (s *Server) handleCompanionSessionScreenPut(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCompanionID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "bad session id")
		return
	}
	sess := s.companionStore.get(id)
	if sess == nil {
		writeError(w, http.StatusNotFound, "no such companion session")
		return
	}
	if !s.companionAuth(r, sess) {
		writeError(w, http.StatusForbidden, "session key required")
		return
	}
	body, err := readBody(r, 10<<20)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sess.mu.Lock()
	sess.screen = string(body)
	sess.events = nil // new screen clears prior events (mirror Node behavior)
	sess.lastActive = time.Now()
	sess.mu.Unlock()
	// broadcast reload to WS clients — wired in Task 3
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func parseCompanionID(r *http.Request) (int64, bool) {
	raw := r.PathValue("id")
	var id int64
	_, err := fmt.Sscanf(raw, "%d", &id)
	return id, err == nil && id > 0
}

func readBody(r *http.Request, max int64) ([]byte, error) {
	b, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, max))
	return b, err
}
```

Note: add `io` to imports; add `companionStore: newCompanionStore()` to the `Server` struct literal in `NewServer`. If `newTestService` does not exist in this package, use the existing test helper the package already uses (check `server_test.go`).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestCompanionScreen|TestCompanionKey' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/companion.go skillgrid-cli/internal/mnemonic/http/companion_test.go skillgrid-cli/internal/mnemonic/http/server.go
git add skillgrid-cli/internal/mnemonic/http/companion/
git commit -m "feat(companion): session store + screen GET/PUT + key gate

[skillgrid-context]
Change: 2026-09-19-embed-visual-companion
Decisions: in-memory session store; frame+helper reused verbatim; ?key= gate
"
```

---

### Task 2: Embed assets + frame/helper parity verification

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/companion/frame-template.html` (already created in Task 1 if not)
- Create: `skillgrid-cli/internal/mnemonic/http/companion/helper.js`
- Modify: `skillgrid-cli/internal/mnemonic/http/companion_test.go` (add parity test)

**Interfaces:**
- Consumes: `companionFrameTemplate`, `companionHelperJS` embed vars from Task 1.
- Produces: `TestCompanionFrameParity` asserting byte-identity with the source skill assets.

**SATISFIES:** scenario `companion-frame-parity`.

- [ ] **Step 1: Write the failing test**

```go
// companion_test.go (add)
func TestCompanionFrameParity(t *testing.T) {
	// The embedded frame + helper must be byte-identical to the standalone
	// skill assets — presentation is shared, not re-implemented.
	srcFrame := filepath.Join("..", "..", "..", "..", "..", ".agents", "skills", "brainstorming", "scripts", "frame-template.html")
	srcHelper := filepath.Join("..", "..", "..", "..", "..", ".agents", "skills", "brainstorming", "scripts", "helper.js")
	// Resolve relative to the companion/ subdir where the embeds live.
	embFrame, _ := os.ReadFile(filepath.Join("companion", "frame-template.html"))
	embHelper, _ := os.ReadFile(filepath.Join("companion", "helper.js"))
	wantFrame, err := os.ReadFile(srcFrame)
	if err != nil { t.Skipf("source not resolvable in this tree: %v", err) }
	if !bytes.Equal(embFrame, wantFrame) {
		t.Errorf("frame-template.html drifted from source skill asset")
	}
	// helper.js: structural parity (contains the key markers), since the embed
	// may reflow newlines — assert the load-bearing strings are present.
	for _, marker := range []string{"brainstorm-session-key", "toggleSelect", "nextReconnectDelay", "TOMBSTONE_AFTER_MS"} {
		if !strings.Contains(string(embHelper), marker) {
			t.Errorf("helper.js missing marker %q", marker)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run TestCompanionFrameParity -v`
Expected: FAIL if the copied assets drifted; PASS once copied verbatim. (If it already passes, the assets are in sync — good; the test guards future drift.)

- [ ] **Step 3: Write minimal implementation**

Copy the two source files verbatim:
```bash
cp .agents/skills/brainstorming/scripts/frame-template.html skillgrid-cli/internal/mnemonic/http/companion/frame-template.html
cp .agents/skills/brainstorming/scripts/helper.js skillgrid-cli/internal/mnemonic/http/companion/helper.js
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run TestCompanionFrameParity -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/companion/ skillgrid-cli/internal/mnemonic/http/companion_test.go
git commit -m "feat(companion): embed frame-template + helper.js verbatim"
```

---

### Task 3: WebSocket upgrade + reload broadcast + event capture

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/companion_ws.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/companion.go` (add `clients` to session; call `broadcastReload` on PUT; add event append + `GET .../events`)
- Modify: `skillgrid-cli/internal/mnemonic/http/companion_test.go`
- Modify: `skillgrid-cli/go.mod` / `go.sum` (only if `golang.org/x/net` is not already present)

**Interfaces:**
- Consumes: `companionSession` (Task 1), `companionEvent` (Task 1).
- Produces:
  - `func (s *Server) handleCompanionWS(w http.ResponseWriter, r *http.Request)` — RFC-6455 upgrade.
  - `func (s *Server) handleCompanionEvents(w http.ResponseWriter, r *http.Request)` — `GET`, returns + clears the event ring.
  - `func (sess *companionSession) addClient(conn websocket.Conn)` / `removeClient(conn)`
  - `func (sess *companionSession) broadcastReload()`
  - `func (sess *companionSession) appendEvent(ev companionEvent)`
  - `func (sess *companionSession) drainEvents() []companionEvent`

**SATISFIES:** scenarios `companion-ws-reload`, `companion-event-capture`, `companion-ws-origin`.

- [ ] **Step 1: Write the failing test**

```go
// companion_test.go (add)
func TestCompanionWSHandshakeAndReload(t *testing.T) {
	s, key := newCompanionServer(t)
	id := s.companionStore.sessionsByAny()[0]

	// Open a WS client with the correct key.
	upgrader := websocket.Upgrader{}
	conn, err := websocket.NewClient(&http.Client{},
		&url.URL{Scheme: "ws", Host: "127.0.0.1", Path: "/companion/sessions/" + itoa(id) + "/ws?key=" + key}, nil)
	if err != nil { t.Fatalf("ws connect: %v", err) }
	defer conn.Close()

	// Push a screen → server must broadcast a reload frame.
	go func() {
		req := httptest.NewRequest("PUT", "/companion/sessions/"+itoa(id)+"/screens?key="+key,
			strings.NewReader(`<h2>v2</h2>`))
		s.Handler().ServeHTTP(httptest.NewRecorder(), req)
	}()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil { t.Fatalf("read reload: %v", err) }
	var got map[string]string
	_ = json.Unmarshal(msg, &got)
	if got["type"] != "reload" {
		t.Errorf("expected reload frame, got %s", msg)
	}
}

func TestCompanionEventCapture(t *testing.T) {
	s, key := newCompanionServer(t)
	id := s.companionStore.sessionsByAny()[0]

	// Simulate the frame posting a click.
	body := `{"type":"click","choice":"a","text":"Option A"}`
	req := httptest.NewRequest("POST", "/companion/sessions/"+itoa(id)+"/events?key="+key, strings.NewReader(body))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("POST event: got %d", w.Code) }

	req = httptest.NewRequest("GET", "/companion/sessions/"+itoa(id)+"/events?key="+key, nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	var evs []companionEvent
	_ = json.Unmarshal(w.Body.Bytes(), &evs)
	if len(evs) != 1 || evs[0].Choice != "a" {
		t.Errorf("events not captured: %+v", evs)
	}
	// drain clears
	req = httptest.NewRequest("GET", "/companion/sessions/"+itoa(id)+"/events?key="+key, nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &evs)
	if len(evs) != 0 { t.Errorf("events not drained: %+v", evs) }
}

func TestCompanionWSOrigin(t *testing.T) {
	s, key := newCompanionServer(t)
	id := s.companionStore.sessionsByAny()[0]
	// Cross-origin Origin must not be upgraded.
	req := httptest.NewRequest("GET", "/companion/sessions/"+itoa(id)+"/ws?key="+key, nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code == 101 { t.Errorf("cross-origin WS should not upgrade, got 101") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestCompanionWS|TestCompanionEvent' -v`
Expected: FAIL — handlers/clients undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// companion_ws.go
package http

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Server) handleCompanionWS(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCompanionID(r)
	if !ok { writeError(w, http.StatusBadRequest, "bad session id"); return }
	sess := s.companionStore.get(id)
	if sess == nil { writeError(w, http.StatusNotFound, "no such companion session"); return }
	if !s.companionAuth(r, sess) { http.Error(w, "session key required", http.StatusForbidden); return }

	origin := r.Header.Get("Origin")
	if origin != "" {
		host := r.Host
		if host == "" { host = r.URL.Host }
		if subtle.ConstantTimeCompare([]byte(origin), []byte("http://"+host)) != 1 &&
			subtle.ConstantTimeCompare([]byte(origin), []byte("http://"+net.JoinHostPort(r.Host, "80"))) != 1 {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
	}

	upgrader := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { return }
	sess.addClient(conn)
	defer sess.removeClient(conn)

	// Read pump: capture click/choice events, echo nothing. Exits on close.
	buf := make([]byte, 16<<10)
	for {
		n, err := conn.Read(buf)
		if err != nil { return }
		var ev companionEvent
		if json.Unmarshal(buf[:n], &ev) == nil && (ev.Type == "click" || ev.Type == "choice") {
			if ev.Timestamp == 0 { ev.Timestamp = time.Now().UnixMilli() }
			sess.appendEvent(ev)
		}
	}
}

// companion.go additions
func (s *Server) handleCompanionEvents(w http.ResponseWriter, r *http.Request) {
	id, ok := parseCompanionID(r)
	if !ok { writeError(w, http.StatusBadRequest, "bad session id"); return }
	sess := s.companionStore.get(id)
	if sess == nil { writeError(w, http.StatusNotFound, "no such companion session"); return }
	if !s.companionAuth(r, sess) { writeError(w, http.StatusForbidden, "session key required"); return }
	switch r.Method {
	case http.MethodPost:
		body, err := readBody(r, 1<<20)
		if err != nil { writeError(w, http.StatusBadRequest, err.Error()); return }
		var ev companionEvent
		if json.Unmarshal(body, &ev) != nil { writeError(w, http.StatusBadRequest, "bad event json"); return }
		if ev.Timestamp == 0 { ev.Timestamp = time.Now().UnixMilli() }
		sess.appendEvent(ev)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default: // GET — drain
		evs := sess.drainEvents()
		if evs == nil { evs = []companionEvent{} }
		writeJSON(w, http.StatusOK, evs)
	}
}

// registerCompanionRoutes — add:
// 	s.mux.HandleFunc("GET /companion/sessions/{id}/ws", s.handleCompanionWS)
// 	s.mux.HandleFunc("POST /companion/sessions/{id}/events", s.handleCompanionEvents)
// 	s.mux.HandleFunc("GET /companion/sessions/{id}/events", s.handleCompanionEvents)
// and on PUT screen success, call sess.broadcastReload().
```

Session client set + broadcast (in `companion.go`):

```go
// add to companionSession:
// 	clients   map[int]*websocket.Conn
// 	clientSeq int
// (import golang.org/x/net/websocket)

func (sess *companionSession) addClient(c *websocket.Conn) {
	sess.mu.Lock(); defer sess.mu.Unlock()
	if sess.clients == nil { sess.clients = map[int]*websocket.Conn{} }
	sess.clientSeq++; sess.clients[sess.clientSeq] = c
}
func (sess *companionSession) removeClient(c *websocket.Conn) {
	sess.mu.Lock(); defer sess.mu.Unlock()
	for k, v := range sess.clients { if v == c { delete(sess.clients, k) } }
}
func (sess *companionSession) broadcastReload() {
	sess.mu.RLock(); defer sess.mu.RUnlock()
	frame, _ := json.Marshal(map[string]string{"type": "reload"})
	for _, c := range sess.clients { _ = c.Write(frame) }
}
func (sess *companionSession) appendEvent(ev companionEvent) {
	sess.mu.Lock(); defer sess.mu.Unlock()
	sess.events = append(sess.events, ev)
	sess.lastActive = time.Now()
	const cap = 256
	if len(sess.events) > cap { sess.events = sess.events[len(sess.events)-cap:] }
}
func (sess *companionSession) drainEvents() []companionEvent {
	sess.mu.Lock(); defer sess.mu.Unlock()
	out, sess.events = sess.events, nil
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestCompanionWS|TestCompanionEvent' -v -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/companion_ws.go skillgrid-cli/internal/mnemonic/http/companion.go skillgrid-cli/internal/mnemonic/http/companion_test.go skillgrid-cli/go.mod skillgrid-cli/go.sum
git commit -m "feat(companion): websocket upgrade + reload broadcast + event capture"
```

---

### Task 4: React `/companion` view (sandboxed iframe + event round-trip)

**Files:**
- Create: `skillgrid-ui/src/features/companion/api.ts`
- Create: `skillgrid-ui/src/features/companion/CompanionPage.tsx`
- Create: `skillgrid-ui/src/features/companion/CompanionFrame.tsx`
- Modify: `skillgrid-ui/src/app.tsx` (add `companionRoute`)
- Modify: `skillgrid-ui/src/components/layout/AppLayout.tsx` (add nav entry)

**Interfaces:**
- Consumes: `GET /companion/sessions/{id}/screen`, `POST /companion/sessions/{id}/events`, `GET /companion/sessions/{id}/events` (Tasks 1–3); `currentProjectName` from `../../lib/projects`.
- Produces: `CompanionPage` (default-exported route component), `companionApi` with `fetchScreen(id)`, `postEvent(id, ev)`, `drainEvents(id)`.

**SATISFIES:** scenarios `companion-view-renders`, `companion-view-roundtrip`.

- [ ] **Step 1: Write the failing test**

```ts
// skillgrid-ui/src/features/companion/CompanionPage.test.tsx (vitest + @testing-library/react)
import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { CompanionPage } from './CompanionPage'

describe('CompanionPage', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('renders the active screen in a sandboxed iframe', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValue(
      new Response('<h2>Which layout?</h2>', { status: 200, headers: { 'Content-Type': 'text/html' } }),
    )
    render(<CompanionPage sessionId={1} sessionKey="k" />)
    const frame = await screen.findByTestId('companion-frame')
    await waitFor(() => expect(frame.getAttribute('sandbox')).not.toContain('allow-same-origin'))
    expect(frame.getAttribute('sandbox')).toContain('allow-scripts')
  })

  it('round-trips a click back to the event stream', async () => {
    const post = vi.fn().mockResolvedValue(new Response('{"ok":true}', { status: 200 }))
    vi.spyOn(global, 'fetch').mockImplementation((url: string) =>
      String(url).includes('/events') ? post() :
      Promise.resolve(new Response('<h2>v</h2>', { status: 200, headers: { 'Content-Type': 'text/html' } })),
    )
    render(<CompanionPage sessionId={1} sessionKey="k" onEvent={post} />)
    // postMessage from the sandboxed frame
    // (the frame calls window.parent.postMessage; here we assert the handler wires up)
    expect(post).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-ui && npm test -- CompanionPage`
Expected: FAIL — module not found.

- [ ] **Step 3: Write minimal implementation**

```tsx
// CompanionFrame.tsx
import { useEffect, useRef, useState } from 'react'

interface Props {
  html: string
  onEvent: (ev: { type: string; choice?: string; text?: string; id?: string }) => void
}

// The frame runs in a sandboxed iframe. helper.js (injected by the server)
// posts clicks via window.parent.postMessage; we relay them to onEvent.
export function CompanionFrame({ html, onEvent }: Props) {
  const ref = useRef<HTMLIFrameElement>(null)
  useEffect(() => {
    const handler = (e: MessageEvent) => {
      const d = e.data as { type?: string; choice?: string; text?: string; id?: string }
      if (d && typeof d.type === 'string') onEvent(d)
    }
    window.addEventListener('message', handler)
    return () => window.removeEventListener('message', handler)
  }, [onEvent])

  // The server already wraps the frame + injects helper.js; srcDoc renders it.
  // sandbox: allow-scripts so helper.js runs; NO allow-same-origin so the screen
  // cannot read the dashboard's cookies/localStorage.
  return (
    <iframe
      ref={ref}
      data-testid="companion-frame"
      title="Companion screen"
      sandbox="allow-scripts allow-popups"
      srcDoc={html}
      className="h-full w-full rounded-md border border-edge bg-background"
    />
  )
}
```

```tsx
// CompanionPage.tsx
import { useCallback, useEffect, useState } from 'react'
import { CompanionFrame } from './CompanionFrame'
import * as api from './api'

export interface CompanionPageProps { sessionId: number; sessionKey: string }

export function CompanionPage({ sessionId, sessionKey }: CompanionPageProps) {
  const [html, setHtml] = useState('')
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    try { setHtml(await api.fetchScreen(sessionId, sessionKey)) }
    catch (e) { setError(String(e)) }
  }, [sessionId, sessionKey])

  useEffect(() => { void load() }, [load])

  const onEvent = useCallback((ev: { type: string; choice?: string; text?: string; id?: string }) => {
    void api.postEvent(sessionId, sessionKey, ev)
  }, [sessionId, sessionKey])

  if (error) return <div className="p-6 text-error">{error}</div>
  return <CompanionFrame html={html} onEvent={onEvent} />
}
```

```ts
// api.ts
export async function fetchScreen(id: number, key: string): Promise<string> {
  const res = await fetch(`/companion/sessions/${id}/screen?key=${encodeURIComponent(key)}`)
  if (!res.ok) throw new Error(`companion screen ${res.status}`)
  return res.text()
}
export async function postEvent(id: number, key: string, ev: Record<string, unknown>): Promise<void> {
  const res = await fetch(`/companion/sessions/${id}/events?key=${encodeURIComponent(key)}`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(ev),
  })
  if (!res.ok) throw new Error(`companion event ${res.status}`)
}
export async function drainEvents(id: number, key: string): Promise<Array<Record<string, unknown>>> {
  const res = await fetch(`/companion/sessions/${id}/events?key=${encodeURIComponent(key)}`)
  if (!res.ok) throw new Error(`companion events ${res.status}`)
  return res.json()
}
```

Wire the route + nav:

```tsx
// app.tsx — add (mirror activityRoute):
const companionRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/companion',
  component: lazyPage(() => import('./features/companion/CompanionPage').then((m) => ({ default: m.CompanionPage }))),
})
// add companionRoute to routeTree.children
```

```tsx
// AppLayout.tsx — add to STANDALONE_ITEMS:
{ to: '/companion', label: 'Companion' },
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-ui && npm test -- CompanionPage && npm run build:check`
Expected: PASS (tests + bundle budget).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-ui/src/features/companion/ skillgrid-ui/src/app.tsx skillgrid-ui/src/components/layout/AppLayout.tsx
git commit -m "feat(ui): companion view — sandboxed iframe + event round-trip"
```

---

### Task 5: Session bootstrap endpoint + reaper + openapi + integration

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/http/companion.go` (add `POST /companion/sessions` bootstrap; add reaper goroutine)
- Modify: `skillgrid-cli/internal/mnemonic/http/companion_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml` (add `/companion/*` paths)
- Create: `skillgrid-cli/internal/mnemonic/http/usermanual_companion_test.go` (doc-existence guard, mirror `usermanual_phase7_test.go`)

**Interfaces:**
- Consumes: `companionStore` (Task 1).
- Produces: `handleCompanionSessionCreate` (POST), `startCompanionReaper(idleTimeout, checkInterval)`; openapi entries; `TestCompanionSessionReaper`.

**SATISFIES:** scenarios `companion-session-bootstrap`, `companion-session-reaped`.

- [ ] **Step 1: Write the failing test**

```go
func TestCompanionSessionBootstrap(t *testing.T) {
	s, _ := newCompanionServer(t)
	req := httptest.NewRequest("POST", "/companion/sessions", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != 201 { t.Fatalf("bootstrap: got %d want 201", w.Code) }
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["id"].(float64) == 0 { t.Errorf("no id in bootstrap: %v", out) }
	if out["key"] == "" { t.Errorf("no key in bootstrap: %v", out) }
}

func TestCompanionSessionReaper(t *testing.T) {
	svc, _ := newTestService(t)
	s := NewServer(svc)
	// shorten reaper for the test via a package var (see impl)
	companionReaperIdle = 30 * time.Millisecond
	companionReaperCheck = 10 * time.Millisecond
	go s.startCompanionReaper(companionReaperIdle, companionReaperCheck)
	defer s.stopCompanionReaper()

	_, key := s.companionStore.createSession()
	id := s.companionStore.sessionsByAny()[0]
	_ = key
	time.Sleep(80 * time.Millisecond)
	if s.companionStore.get(id) != nil {
		t.Errorf("idle session not reaped")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestCompanionSessionBootstrap|TestCompanionSessionReaper' -v`
Expected: FAIL — bootstrap/reaper undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// companion.go additions
var (
	companionReaperIdle  = 4 * time.Hour  // overridable in tests
	companionReaperCheck = 60 * time.Second
)

func (s *Server) handleCompanionSessionCreate(w http.ResponseWriter, r *http.Request) {
	sess, key := s.companionStore.createSession()
	writeJSON(w, http.StatusCreated, map[string]any{"id": sess.id, "key": key, "url": fmt.Sprintf("/companion/sessions/%d/screen?key=%s", sess.id, key)})
}

func (s *Server) startCompanionReaper(idle, check time.Duration) {
	s.reaperStop = make(chan struct{})
	go func() {
		t := time.NewTicker(check)
		defer t.Stop()
		for {
			select {
			case <-s.reaperStop:
				return
			case <-t.C:
				now := time.Now()
				for _, sess := range s.companionStore.snapshot() {
					sess.mu.RLock()
					idle := now.Sub(sess.lastActive)
					hasClient := len(sess.clients) > 0
					sess.mu.RUnlock()
					if !hasClient && idle > idle {
						s.companionStore.removeSession(sess.id)
					}
				}
			}
		}
	}()
}
func (s *Server) stopCompanionReaper() { close(s.reaperStop) }

// companionStore.snapshot() returns []*companionSession (RLock).
// registerCompanionRoutes — add:
//  s.mux.HandleFunc("POST /companion/sessions", s.handleCompanionSessionCreate)
```

Wire `reaperStop chan struct{}` into `Server`; start the reaper in `NewServer` (or `runServe`) with the package defaults; add `companion` to the openapi `tags` and the `/companion/*` path entries (mirror the activity path shape). Add the usermanual guard test asserting a dashboard doc page exists for the companion (mirror `usermanual_phase7_test.go`).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/http/ -run 'TestCompanion' -v -race`
Expected: PASS (all companion tests).

- [ ] **Step 5: Run full suite + build**

Run: `cd skillgrid-cli && go build ./... && go test ./... 2>&1 | tail -20`
Expected: build clean, no new failures.

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/ skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml
git commit -m "feat(companion): session bootstrap + idle reaper + openapi + docs guard"
```

---

## Self-Review

1. **Spec coverage:** every capability (serve screen, key gate, event capture, WS reload, frame parity, sandboxed view, bootstrap, reaper) has a task. ✔
2. **Must-haves coverage:** truths 1–8 map to Tasks 1–5; artifacts + key links listed. `backstop` truths (1, 5, 6) get held-out tests: Task 1 e2e render (manual/browser during door check), Task 2 `TestCompanionFrameParity`, Task 4 sandbox test. ✔
3. **One-way-door completeness:** none — no migration/contract break; listed as "None". ✔
4. **Placeholder scan:** no TBD/TODO/"similar to Task N"; each step has code. (Task 3 `itoa` helper and `newTestService`/`readBody` are called out to match existing package helpers — confirm exact names at execution time.) ✔
5. **Type consistency:** `companionSession`/`companionEvent`/`companionStore` defined once (Task 1) and reused; `handleCompanionWS`/`handleCompanionEvents` names consistent across Tasks 3–5; `companionReaperIdle/Check` package vars consistent. ✔

## Plan Review

- Verdict: **READY FOR EXECUTION**
- Findings: 0 Critical, 1 Important (Task 3 assumes `golang.org/x/net/websocket`; verify it's in `go.mod` at execution — if not, `go get` it; the WS upgrade-on-GO-1.22-mux is the only genuinely new risk, so Task 1's door check plus Task 3's handshake test gate it before UI work), 2 Minor (deferred: confirm `newTestService` exact helper name and `readBody` helper in `server.go`; the `itoa` test helper is `strconv.FormatInt`).
- Reviewed: 2026-09-19

## Execution Handoff

Blueprint has 5 tasks → invoke `skillgrid:slicing` to break into vertical tracer-bullet tickets with execution waves, producing `tasks.md` alongside this blueprint.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task + two-stage review.
2. **Inline Execution** — `skillgrid:simple-execution`, batch with checkpoints.

**Door check first:** execute Task 1 alone and confirm the screen round-trip + key gate pass in a live `skillgrid serve` before committing to the full build.
