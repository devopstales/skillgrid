package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// getUI fetches an embedded UI asset through the real handler tree.
func getUI(t *testing.T, h http.Handler, target string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	return rr.Code, string(body)
}

func mustContain(t *testing.T, body string, subs ...string) {
	t.Helper()
	for _, s := range subs {
		if !strings.Contains(body, s) {
			t.Errorf("expected UI to contain %q", s)
		}
	}
}

// TestStep01_Shell: GET / serves the menu shell with path routing + project selector.
func TestStep01_Shell(t *testing.T) {
	h := newHandler(t)
	code, body := getUI(t, h, "/")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	mustContain(t, body,
		`id="menu"`, `data-route="welcome"`, `data-route="tracker"`,
		`data-route="docs"`, `data-route="memory"`, `data-route="code"`,
		`data-route="sessions"`, `href="/welcome"`, `href="/tracker"`, `href="/docs"`,
		`href="/memory"`, `href="/code"`, `href="/sessions"`,
		`id="project"`, `id="crumb-current"`, `id="content"`,
		`<script src="/app.js">`, `<link rel="stylesheet" href="/app.css">`,
	)
	if strings.Contains(body, "#/") {
		t.Error("shell must use path routing, found hash-route hrefs")
	}
	// Every menu path serves the same shell (client router picks the entry).
	for _, target := range []string{"/welcome", "/tracker", "/docs", "/memory", "/code"} {
		code, page := getUI(t, h, target)
		if code != http.StatusOK {
			t.Errorf("expected %s 200, got %d", target, code)
			continue
		}
		if !strings.Contains(page, `id="menu"`) {
			t.Errorf("%s must serve the dashboard shell", target)
		}
		if strings.Contains(page, "#/") {
			t.Errorf("%s shell must use path routing", target)
		}
	}
	// P6: /sessions is now the session-list API (server.go). An HTML request
	// without ?project= (a browser reload on the Sessions entry) still gets the
	// shell via the API's shell fallback.
	req := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	page, _ := io.ReadAll(rr.Result().Body)
	if rr.Code != http.StatusOK {
		t.Errorf("expected /sessions (HTML reload) 200, got %d", rr.Code)
	} else if !strings.Contains(string(page), `id="menu"`) {
		t.Errorf("/sessions HTML reload must serve the dashboard shell")
	}
	if code, js := getUI(t, h, "/app.js"); code != http.StatusOK || !strings.Contains(js, "currentRoute") {
		t.Errorf("expected /app.js router, got %d", code)
	}
	if code, css := getUI(t, h, "/app.css"); code != http.StatusOK || !strings.Contains(css, ".sidebar") {
		t.Errorf("expected /app.css shell styles, got %d", code)
	}
}

// TestStep01_Welcome: static Welcome page — a 6-entry navigation grid (all
// dashboard routes) with zero fetches. Refreshed from the P1 "roadmap + 2
// swagger cards" layout once all entries are live.
func TestStep01_Welcome(t *testing.T) {
	h := newHandler(t)
	_, body := getUI(t, h, "/")
	mustContain(t, body,
		`id="view-welcome"`, "all entries live", "Skillgrid Dashboard",
		`href="/memory"`, `href="/tracker"`, `href="/code"`,
		`href="/sessions"`, `href="/docs"`, `href="/swagger-ui"`,
	)
	// The stale P1 copy must be gone (roadmap + "phase 3 online" + placeholder card).
	for _, stale := range []string{"phase 3 online", "Roadmap", "later phases", "More dashboards"} {
		if strings.Contains(body, stale) {
			t.Errorf("welcome must not contain stale P1 copy %q", stale)
		}
	}
	// Welcome content is static HTML: the router must not fetch on /welcome.
	_, js := getUI(t, h, "/app.js")
	renderIdx := strings.Index(js, "function render()")
	if renderIdx < 0 {
		t.Fatal("expected render() in app.js")
	}
	// Between render() start and the first ensureProjects() call there must be
	// an early return for the welcome route (no fetch before it).
	welcomeReturn := strings.Index(js[renderIdx:], `route === "welcome"`)
	callIdx := strings.Index(js[renderIdx:], "ensureProjects()")
	if welcomeReturn < 0 || callIdx < 0 || welcomeReturn > callIdx {
		t.Error("welcome route must return before any data fetch in render()")
	}
}

// TestStep01_Stubs: P1 shipped P3–P6 as labeled disabled stubs; by P6 every
// entry is live, so the shell carries no stub tags or aria-disabled links and
// the stub view node remains (unused) while STUBS is empty.
func TestStep01_Stubs(t *testing.T) {
	h := newHandler(t)
	_, body := getUI(t, h, "/")
	if strings.Contains(body, "stub-tag") || strings.Contains(body, `aria-disabled="true"`) {
		t.Errorf("all entries are live after P6: no stub tags or aria-disabled links may remain: %s", body)
	}
	mustContain(t, body, `id="view-stub"`)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js, "STUBS", "Coming in")
	if strings.Contains(js, "phase:") {
		t.Errorf("STUBS must be empty after P6, found a stub phase entry")
	}
}

// TestStep01_SwaggerLink: swagger-ui + openapi serve from the menu links.
func TestStep01_SwaggerLink(t *testing.T) {
	h := newHandler(t)
	code, body := getUI(t, h, "/swagger-ui")
	if code != http.StatusOK {
		t.Fatalf("expected /swagger-ui 200, got %d", code)
	}
	// Bundle asset URLs must resolve under /swagger-ui/{file} (issue: root
	// /swagger-ui.css 404). No ./-relative or root-relative bundle refs remain.
	for _, banned := range []string{`"./`, `"/swagger-ui.css"`, `"/swagger-ui-bundle.js"`, `"index.css"`} {
		if strings.Contains(body, banned) {
			t.Errorf("swagger page still contains unrewritten ref %q", banned)
		}
	}
	for _, asset := range []string{
		"/swagger-ui/swagger-ui.css", "/swagger-ui/swagger-ui-bundle.js",
		"/swagger-ui/swagger-ui-standalone-preset.js",
		"/swagger-ui/swagger-initializer.js", "/swagger-ui/index.css",
	} {
		if code, _ := getUI(t, h, asset); code != http.StatusOK {
			t.Errorf("expected %s 200, got %d", asset, code)
		}
	}
	code, spec := getUI(t, h, "/openapi.yaml")
	if code != http.StatusOK || spec == "" {
		t.Errorf("expected /openapi.yaml 200 with content, got %d", code)
	}
}

// TestStep01_NoCDN: shell assets are self-contained (no external hosts).
func TestStep01_NoCDN(t *testing.T) {
	h := newHandler(t)
	for _, target := range []string{"/", "/app.js", "/app.css"} {
		_, body := getUI(t, h, target)
		for _, banned := range []string{"https://", "http://", "cdn.", "unpkg", "googleapis", "//fonts"} {
			if strings.Contains(body, banned) {
				t.Errorf("%s contains banned external ref %q", target, banned)
			}
		}
	}
}
