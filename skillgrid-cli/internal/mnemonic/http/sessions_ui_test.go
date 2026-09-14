package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readManual locates docs/user-manual/10-webui.md relative to this package
// (skillgrid-cli/internal/mnemonic/http → 4 levels up is the repo root), so
// the 06.7 assertion works no matter where the test binary runs from.
func readManual(t *testing.T) (string, error) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := filepath.Join(wd, "..", "..", "..", "..", "docs", "user-manual", "10-webui.md")
	b, err := os.ReadFile(root)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func containsAll(body, sub string) bool { return strings.Contains(body, sub) }

// 06.3 [AFK] Sessions menu entry: session list (title, started_at, status),
// recent context, click → summary pane. The entry is live (not a stub) and
// wired to the new P6 read routes.
func TestStep06_Sessions(t *testing.T) {
	h := newHandler(t)
	_, body := getUI(t, h, "/")
	// Sessions entry is live: no stub tag, no aria-disabled.
	if containsAll(body, "Sessions <span class=\"stub-tag\">") {
		t.Error("sessions entry must be live (still shows P6 stub tag)")
	}
	if containsAll(body, `href="/sessions" data-route="sessions" aria-disabled="true"`) {
		t.Error("sessions entry must not be aria-disabled once live")
	}
	mustContain(t, body, `data-route="sessions"`, `href="/sessions"`)
	// The Welcome page is a 6-entry nav grid (roadmap removed once all live);
	// the sessions card links the entry and no P6 "todo" marker remains.
	mustContain(t, body, "all entries live")
	if containsAll(body, `phase-id todo">P6`) {
		t.Error("P6 must no longer be marked todo once live")
	}
	_, js := getUI(t, h, "/app.js")
	// Entry is live: STUBS empty, LIVE includes sessions, render() routes it.
	mustContain(t, js,
		"renderSessions", "/sessions", "/context", "sessOpenSummary",
		`"welcome", "tracker", "docs", "memory", "code", "sessions"`,
	)
	if containsAll(js, "sessions: { phase:") {
		t.Error("sessions must be removed from STUBS once live")
	}
	// The render() dispatcher has a parallel sessions branch (microtask-deferred).
	mustContain(t, js, `route === "sessions"`, "renderSessions(stub)")
	// Session list rows carry the three spec'd fields.
	mustContain(t, js, "sessRowHtml", "sess-item-title", "sess-item-date", "sess-status-badge")
	// Recent context strip reuses GET /context (the Memory entry's consumer).
	mustContain(t, js, "sessLoadContext", `memUrl("/context?limit=10")`)
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".sess-page", ".sess-grid", ".sess-item", ".sess-summary", ".sess-status-badge", ".sess-context")
}

// 06.4 [AFK] Sessions 404 renders an in-pane error, never a blank pane: the
// summary pane's catch renders an error card (the same isolated-error idiom
// as the Memory detail pane and the Code source view).
func TestStep06_Session404(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	// The summary fetch is in a try/catch whose catch writes an in-pane error
	// ("Could not load session summary") into #sess-summary, not a blank pane.
	mustContain(t, js,
		"sessOpenSummary", "Could not load session summary",
		`class="code-error"`, "/summary",
	)
	// The list and context widgets have their own isolated error renders.
	mustContain(t, js, "Sessions unavailable", "Context unavailable")
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".code-error {")
}

// 06.5 [AFK] openapi.yaml final review: every P2–P6 route documented with a
// valid example. The two new P6 routes are present with examples, and the
// key P2–P5 routes survive the review.
func TestStep06_OpenAPI(t *testing.T) {
	h := newHandler(t)
	code, spec := getUI(t, h, "/openapi.yaml")
	if code != http.StatusOK {
		t.Fatalf("expected /openapi.yaml 200, got %d", code)
	}
	// P6 new routes, with a request/response example on each.
	mustContain(t, spec,
		"operationId: sessionList",
		"operationId: sessionSummary",
		"SessionList",
		"SessionSummary",
		"example:",
	)
	// P2–P5 routes still documented (regression check on the final review).
	mustContain(t, spec,
		"/tracker/config", // P2
		"/docs/changes",   // P3
		"/observations",   // P4
		"/code/status",    // P5
		"/code/search",    // P5
	)
	if !containsAll(spec, "openapi: 3.") {
		t.Error("openapi.yaml lost its version header")
	}
}

// 06.6 [AFK] /swagger-ui loads and exercises each new route: the swagger
// shell is wired (initializer references the served spec) and the served
// openapi.yaml — the document Swagger UI reads — contains the new routes.
func TestStep06_SwaggerUI(t *testing.T) {
	h := newHandler(t)
	code, body := getUI(t, h, "/swagger-ui")
	if code != http.StatusOK {
		t.Fatalf("expected /swagger-ui 200, got %d", code)
	}
	mustContain(t, body, `id="view-swagger"`)
	// The initializer served from /swagger-ui points at /openapi.yaml.
	code2, init := getUI(t, h, "/swagger-ui/swagger-initializer.js")
	if code2 != http.StatusOK {
		t.Fatalf("expected /swagger-ui/swagger-initializer.js 200, got %d", code2)
	}
	mustContain(t, init, "/openapi.yaml", "SwaggerUIBundle")
	// The document the UI reads carries the new P6 routes + the old routes.
	code3, spec := getUI(t, h, "/openapi.yaml")
	if code3 != http.StatusOK {
		t.Fatalf("expected /openapi.yaml 200, got %d", code3)
	}
	mustContain(t, spec, "operationId: sessionList", "operationId: sessionSummary", "operationId: sessionCreate")
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js, "loadSwagger", "swagger-initializer.js")
}

// 06.7 [AFK] User manual serve section documents the six menu entries and
// the per-provider tracker-CLI dependency (the tracker bridge shells out to
// the provider's CLI: `backlog` for Backlog.md, the provider CLI otherwise).
func TestStep06_UserManual(t *testing.T) {
	data, err := readManual(t)
	if err != nil {
		t.Fatalf("read user manual: %v", err)
	}
	// All six menu entries are named.
	mustContain(t, data, "Welcome", "Tracker", "Docs", "Memory", "Code", "Sessions")
	// Per-provider tracker-CLI dependency is documented.
	if !containsAll(data, "backlog") && !containsAll(data, "Backlog.md") {
		t.Error("manual must document the Backlog.md tracker-CLI dependency")
	}
	mustContain(t, data, "CLI")
}
