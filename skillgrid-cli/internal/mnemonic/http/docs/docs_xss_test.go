package docs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedXSSRepo writes a doc whose body contains a raw <script> tag and a
// language-mermaid fence, plus a mermaid default config fixture.
func seedXSSRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("docs/xss.md", "# XSS\n\n<script>alert('pwned')</script>\n\n```mermaid\ngraph TD; A-->B;\n```\n")
	return root
}

// 3.2 [RED] Threat: Mermaid/markdown XSS — untrusted markdown + language-mermaid
// must be served inert (the SPA renders/sanitizes; the server never executes and
// the rendered fallback escapes raw HTML), and the mermaid securityLevel default
// is 'strict'.
func TestPhase3_MermaidSanitize(t *testing.T) {
	cwd := seedXSSRepo(t)
	mux := http.NewServeMux()
	mux.Handle("GET /docs/content", NewContent(cwd))
	mux.Handle("GET /docs/render", NewRender(cwd))

	// (a) The content endpoint must return the <script> as inert text (it is the
	// raw body the SPA sanitizes) — it must NOT be pre-rendered into live HTML.
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/content?path=docs/xss.md", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("content: expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "alert('pwned')") {
		t.Errorf("content missing the raw script body for SPA sanitization: %s", rr.Body.String())
	}

	// (b) The SSR render fallback must ESCAPE the <script> tag (no live element).
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/render?path=docs/xss.md", nil))
	body := rr.Body.String()
	if strings.Contains(body, "<script>") {
		t.Errorf("render fallback emitted a live <script> tag (XSS): %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Errorf("render fallback did not escape the script tag: %s", body)
	}

	// (c) The mermaid securityLevel default exposed to the SPA is 'strict'.
	if lvl := MermaidSecurityLevel(); lvl != "strict" {
		t.Errorf("MermaidSecurityLevel() = %q, want \"strict\"", lvl)
	}
}
