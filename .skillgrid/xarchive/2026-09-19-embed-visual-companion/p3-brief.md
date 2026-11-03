### TICKET-05 — RED: prototype serve route + iframe acceptance tests (from tasks.md)
### TICKET-05 — RED: prototype serve route + iframe acceptance tests
- **Tracker ID:** TASK-011

- **Scope:** Write the failing acceptance tests (no implementation yet) for the prototype function: `GET /prototype/{id...}` serves `.skillgrid/prototype/<topic>/<variant>.html` with `Content-Type: text/html`, returns `404` for a missing file, returns `400`/`404` for a path-traversal escape (e.g. `/prototype/../../.stitch/foo.html`), and `DecisionCard` renders the sandboxed iframe only when `content.visual` is set (attr `sandbox="allow-scripts"` and **not** `allow-same-origin`).
- **Acceptance:** `go test -count=1 ./internal/mnemonic/http/... -run 'TestPrototype'` and the UI iframe test fail (route/component not yet wired) with the expected assertions; tests compile.
- **SATISFIES:** Scenario "visual companion renders a throwaway prototype" (RED state) + Scenario "path traversal is rejected" (RED state).
- **Files:** `internal/mnemonic/http/prototype_test.go` (new), `skillgrid-ui/src/features/decisions/DecisionCard.test.tsx` (add iframe cases).
- **Size:** ~150 (S)
- **Blocks:** TICKET-06
- **Blocked by:** TICKET-04 (iframe lives in DecisionCard)
- **Fails-when:** the prototype go test and the UI iframe test exit non-zero (expected RED) but packages compile.

### TICKET-06 — prototype serve route + sandboxed iframe (from tasks.md)
### TICKET-06 — prototype serve route + sandboxed iframe in the card
- **Tracker ID:** TASK-012

- **Scope:** Implement `GET /prototype/{id...}` in `internal/mnemonic/http/prototype.go` — root = `.skillgrid/prototype/` (via `sddRoot()`), reuse the `stitchFile` traversal-guard shape (reject absolute ids, `..`, and anything escaping the root after `filepath.Clean`), serve `text/html`. Register the route in `server.go` (distinct subtree from the plural `/prototypes`). Wire `DecisionCard` to render the existing `SandboxPreview` iframe (src=`/prototype/{content.visual}`) only when `visual` is set.
- **Acceptance:** TICKET-05 tests now pass; with a seeded `.skillgrid/prototype/<topic>/a.html`, `GET /prototype/<topic>/a.html` returns the HTML, traversal `GET /prototype/../../etc/passwd` is rejected, and the dashboard decision card shows the prototype in a sandboxed iframe.
- **SATISFIES:** Scenario "visual companion renders a throwaway prototype" (GREEN) + Scenario "path traversal is rejected" (GREEN).
- **Files:** `internal/mnemonic/http/prototype.go` (new), `internal/mnemonic/http/prototype_test.go` (fill in), `internal/mnemonic/http/server.go` (register route), `skillgrid-ui/src/features/decisions/DecisionCard.tsx` (iframe).
- **Size:** ~300 (M)
- **Blocks:** none
- **Blocked by:** TICKET-04, TICKET-05
- **Fails-when:** `go test -count=1 ./internal/mnemonic/http/...` or the UI iframe test exits non-zero.

### Blueprint Task 5 (full)
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

