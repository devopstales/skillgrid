package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// newProtoServer builds a server whose sddRoot points at a temp repo with a
// .skillgrid/prototype/ tree (the companion's throwaway HTML prototypes,
// distinct from the .stitch/ design prototypes) plus a sibling file outside
// the root that must NOT be reachable (path-traversal threat).
func newProtoServer(t *testing.T) http.Handler {
	t.Helper()
	repo := t.TempDir()
	proto := filepath.Join(repo, ".skillgrid", "prototype", "demo")
	if err := os.MkdirAll(proto, 0o755); err != nil {
		t.Fatalf("mkdir prototype: %v", err)
	}
	if err := os.WriteFile(filepath.Join(proto, "a.html"),
		[]byte("<html><body><h1>variant A</h1></body></html>"), 0o644); err != nil {
		t.Fatalf("write a.html: %v", err)
	}
	// A secret file OUTSIDE .skillgrid/prototype/ that must not be servable.
	if err := os.WriteFile(filepath.Join(repo, "secret.html"),
		[]byte("<html><body>SECRET</body></html>"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	t.Setenv("SKILLGRID_DOCS_CWD", repo)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	return NewServer(svc).Handler()
}

func TestPrototypeServeAndTraversal(t *testing.T) {
	h := newProtoServer(t)

	// --- happy path: serves the seeded file as HTML ---
	rr := doGet(t, h, "/prototype/demo/a.html")
	if rr.Code != http.StatusOK {
		t.Fatalf("/prototype/demo/a.html: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "variant A") {
		t.Errorf("content = %q, want 'variant A'", rr.Body.String())
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}

	// --- traversal / absolute ids → 400 ---
	// Bare `..` ids are rewritten by the mux's clean-path redirect before
	// reaching the handler (307 to a path outside the /prototype subtree);
	// %-encoded forms reach the handler and must be 400. A successful
	// response must never be the secret (no escape past the root either way).
	wants := map[string]int{
		"307": http.StatusTemporaryRedirect,
		"400": http.StatusBadRequest,
		"404": http.StatusNotFound,
	}
	for _, tc := range []struct{ id, want string }{
		{"../a.html", "307"},
		{"..%2F..%2Fa.html", "400"},
		{"%2e%2e/secret.html", "400"},
		{"%2Fetc%2Fpasswd", "400"},
		{"demo/../../secret.html", "307"},
		{"demo/../secret.html", "307"},
	} {
		rr = doGet(t, h, "/prototype/"+tc.id)
		if rr.Code != wants[tc.want] {
			t.Errorf("id %q: status = %d, want %s; body=%s", tc.id, rr.Code, tc.want, rr.Body.String())
		}
		if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "SECRET") {
			t.Errorf("id %q: served SECRET (escaped .skillgrid/prototype/)", tc.id)
		}
	}

	// --- unknown file → 404 ---
	rr = doGet(t, h, "/prototype/demo/nope.html")
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown: status = %d, want 404; body=%s", rr.Code, rr.Body.String())
	}
}
