package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// newPhase7ProtoServer builds a server whose sddRoot points at a temp repo with
// a .stitch/ prototype (index.html) + a sibling file outside .stitch/ that must
// NOT be reachable (path-traversal threat).
func newPhase7ProtoServer(t *testing.T) http.Handler {
	t.Helper()
	repo := t.TempDir()
	stitch := filepath.Join(repo, ".stitch")
	if err := os.MkdirAll(stitch, 0o755); err != nil {
		t.Fatalf("mkdir .stitch: %v", err)
	}
	// A prototype (a self-contained HTML file).
	if err := os.WriteFile(filepath.Join(stitch, "proto-a.html"),
		[]byte("<html><body><h1>Proto A</h1></body></html>"), 0o644); err != nil {
		t.Fatalf("write proto: %v", err)
	}
	// A secret file OUTSIDE .stitch/ that must not be servable.
	if err := os.WriteFile(filepath.Join(repo, "secret.html"),
		[]byte("<html><body>SECRET</body></html>"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	// A subdirectory prototype (to test nesting is still sandboxed).
	sub := filepath.Join(stitch, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "proto-b.html"),
		[]byte("<html><body>Proto B</body></html>"), 0o644); err != nil {
		t.Fatalf("write proto-b: %v", err)
	}

	t.Setenv("SKILLGRID_DOCS_CWD", repo)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	return NewServer(svc).Handler()
}

func TestPhase7_Prototypes(t *testing.T) {
	h := newPhase7ProtoServer(t)

	// --- GET /prototypes → list of prototype files (sandboxed) ---
	rr := doGet(t, h, "/prototypes")
	if rr.Code != http.StatusOK {
		t.Fatalf("/prototypes: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "proto-a.html") {
		t.Errorf("/prototypes missing proto-a.html; body=%s", body)
	}
	if !strings.Contains(body, "proto-b.html") {
		t.Errorf("/prototypes missing nested proto-b.html; body=%s", body)
	}

	// --- GET /prototypes/{id} happy → the prototype HTML ---
	rr = doGet(t, h, "/prototypes/proto-a.html")
	if rr.Code != http.StatusOK {
		t.Fatalf("/prototypes/proto-a.html: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Proto A") {
		t.Errorf("proto-a content = %q, want 'Proto A'", rr.Body.String())
	}

	// --- nested prototype ---
	rr = doGet(t, h, "/prototypes/nested/proto-b.html")
	if rr.Code != http.StatusOK {
		t.Fatalf("nested proto-b: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Proto B") {
		t.Errorf("proto-b content = %q, want 'Proto B'", rr.Body.String())
	}

	// --- path traversal: `..` must NOT escape .stitch/ ---
	for _, id := range []string{
		"../secret.html",
		"..%2Fsecret.html",
		"%2e%2e/secret.html",
		"../secret.html",
	} {
		rr = doGet(t, h, "/prototypes/"+id)
		// Either 400/404 (rejected) or, if served, must NOT contain SECRET.
		if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "SECRET") {
			t.Errorf("traversal %q: served SECRET (escaped .stitch/); status=%d body=%s", id, rr.Code, rr.Body.String())
		}
	}

	// --- absolute path → rejected ---
	rr = doGet(t, h, "/prototypes/%2Fetc%2Fhostname")
	if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "SECRET") {
		t.Errorf("absolute path served SECRET; status=%d", rr.Code)
	}

	// --- unknown id → 404 ---
	rr = doGet(t, h, "/prototypes/nope.html")
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown proto: status = %d, want 404; body=%s", rr.Code, rr.Body.String())
	}
}
