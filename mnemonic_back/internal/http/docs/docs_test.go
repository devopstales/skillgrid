package docs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedRepo lays out a minimal docs/skillgit tree plus a secret file that a
// traversal must never reach.
func seedRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	docs := filepath.Join(root, "docs", "skillgrid")
	for _, dir := range []string{
		filepath.Join(docs, "changes", "009-web-admin-dashboard"),
		filepath.Join(docs, "changes", "004-hermes-memory"),
		filepath.Join(docs, "archive", "001-hybrid-teams-architecture"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(docs, "changes", "009-web-admin-dashboard", "change.md"),
		"# Change: 009-web-admin-dashboard\n\n**Ticket:** TASK-001\n")
	write(filepath.Join(docs, "changes", "009-web-admin-dashboard", "tasks.md"),
		"# Tasks: 009-web-admin-dashboard\n\n- [ ] 01.1 do the thing\n")
	write(filepath.Join(docs, "changes", "004-hermes-memory", "change.md"),
		"# Change: 004-hermes-memory\n\n**Ticket:** `task-001` (plan ticket)\n")
	write(filepath.Join(docs, "archive", "001-hybrid-teams-architecture", "change.md"),
		"# Change: 001-hybrid-teams-architecture\n")
	write(filepath.Join(root, "secret.md"), "SECRET=traversal-found-me\n")
	return root
}

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	cwd := seedRepo(t)
	list := NewList(cwd)
	detail := NewDetail(cwd)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		// Minimal router (mux would 307-clean dot-segment paths before the
		// guard runs; production mounts the same handlers on literal
		// patterns, where the guard is defense-in-depth).
		switch {
		case r.URL.Path == "/docs/changes":
			list.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/docs/changes/"):
			name := strings.TrimPrefix(r.URL.Path, "/docs/changes/")
			r.SetPathValue("name", name)
			detail.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

func doGet(t *testing.T, h http.Handler, target string) (int, string) {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
	return rr.Code, rr.Body.String()
}

// 03.1 [RED] Threat: path traversal — `..`, absolute paths, and unknown
// names are blocked; a happy name returns change.md + tasks.md + ticket.
func TestStep03_Traversal(t *testing.T) {
	h := newHandler(t)

	if code, body := doGet(t, h, "/docs/changes/../secret"); code != http.StatusBadRequest {
		t.Errorf("GET /docs/changes/../secret: expected 400, got %d (%s)", code, body)
	}
	if code, body := doGet(t, h, "/docs/changes/nope"); code != http.StatusNotFound {
		t.Errorf("GET /docs/changes/nope: expected 404, got %d (%s)", code, body)
	}

	code, body := doGet(t, h, "/docs/changes/009-web-admin-dashboard")
	if code != http.StatusOK {
		t.Fatalf("happy name: expected 200, got %d (%s)", code, body)
	}
	for _, want := range []string{
		"change_md", "tasks_md", "do the thing", "TASK-001",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("happy name missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, "SECRET=traversal-found-me") {
		t.Error("a file outside the docs roots leaked into the response")
	}
}

// 03.2 [AFK] GET /docs/changes returns the change list (name, status,
// ticket id parsed from change.md Ticket:).
func TestStep03_List(t *testing.T) {
	h := newHandler(t)
	code, body := doGet(t, h, "/docs/changes")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", code, body)
	}
	for _, want := range []string{
		"009-web-admin-dashboard", "004-hermes-memory", "001-hybrid-teams-architecture",
		"TASK-001",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("list missing %q in %s", want, body)
		}
	}
}
