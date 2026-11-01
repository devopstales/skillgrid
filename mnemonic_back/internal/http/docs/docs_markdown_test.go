package docs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedMDRepo lays out the declared markdown doc roots (mapped to the real
// repo layout) plus a secret file that a traversal must never reach.
func seedMDRepo(t *testing.T) string {
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
	// sdd root -> .skillgrid/specs/<change>/
	write(".skillgrid/specs/009-web-admin-dashboard/briefing.md", "# Briefing\n\n## Goal\n\nShip it.\n")
	write(".skillgrid/specs/009-web-admin-dashboard/tasks.md", "# Tasks\n\n- [x] 01.1 done\n")
	// skillgrid first-class docs -> .skillgrid/adr/ (numeric ADR) + .skillgrid/prd/ (numeric PRD)
	write(".skillgrid/adr/0007-sdd-docs.md",
		"# SDD docs viewer\n\n---\nstatus: \"accepted\"\nsupersedes: none\ndate: 2026-09-17\n---\n\n## Context and Problem Statement\n\nWhy.\n\n## Decision Outcome\n\nExistence-gated roots.\n\n### Consequences\n\nGood.\n")
	write(".skillgrid/prd/0001-dashboard.md",
		"---\nid: 0001\ntitle: Dashboard — PRD\ntype: prd\n---\n\n# Dashboard — PRD\n\n## 1. Problem\n\nProblem.\n")
	// backlog root -> .backlog/tasks/
	write(".backlog/tasks/TASK-001-something.md", "---\nid: TASK-001\nstatus: in-progress\npriority: high\n---\n\n# Task one\n\nBody.\n")
	// a hidden .skillgrid subdir (must NOT appear in the skillgrid root tree)
	write(".skillgrid/archive/old-change/briefing.md", "# Old Briefing\n\nArchived.\n")
	// docs root -> docs/
	write("docs/guide.md", "# Guide\n\nSome guide text.\n")
	// root *.md
	write("README.md", "# README\n\nRepo readme.\n")
	// the secret — outside every declared root (a subdir that is not a root)
	write(".private/secret.md", "SECRET=traversal-found-me\n")
	return root
}

// newMDHandler wires the markdown routes exactly as the server registers them.
func newMDHandler(t *testing.T) http.Handler {
	t.Helper()
	cwd := seedMDRepo(t)
	mux := http.NewServeMux()
	mux.Handle("GET /docs/tree", NewTree(cwd))
	mux.Handle("GET /docs/content", NewContent(cwd))
	mux.Handle("GET /docs/search", NewSearch(cwd))
	mux.Handle("GET /docs/render", NewRender(cwd))
	return mux
}

// 3.1 [RED] Threat: path traversal — `..`, absolute paths, and unknown paths
// are blocked; a happy path under a declared root returns the file plus its
// frontmatter and relatedPlans.
func TestPhase3_Traversal(t *testing.T) {
	h := newMDHandler(t)
	do := func(target string) (int, string) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
		return rr.Code, rr.Body.String()
	}

	// traversal / unknown -> 400 (raw .. or absolute) or 404 (clean but absent)
	for _, tc := range []struct{ target string; want int }{
		{"/docs/content?path=../secret.md", 400},
		{"/docs/content?path=..%2Fsecret.md", 400},
		{"/docs/content?path=/etc/passwd", 400},
		{"/docs/content?path=docs%2F..%2Fsecret.md", 400},
		{"/docs/content?path=docs/missing.md", 404},
		{"/docs/content?path=notadoc.txt", 404},
		{"/docs/content?path=", 400},
	} {
		code, body := do(tc.target)
		if code != tc.want {
			t.Errorf("GET %s: expected %d, got %d (%s)", tc.target, tc.want, code, body)
		}
	}

	// happy path under a declared root -> 200 with content + frontmatter
	code, body := do("/docs/content?path=.backlog/tasks/TASK-001-something.md")
	if code != http.StatusOK {
		t.Fatalf("happy backlog path: expected 200, got %d (%s)", code, body)
	}
	for _, want := range []string{"TASK-001", "in-progress", "high", "Body"} {
		if !strings.Contains(body, want) {
			t.Errorf("happy backlog path missing %q in %s", want, body)
		}
	}

	// the secret must never leak on any content path
	for _, target := range []string{
		"/docs/content?path=.private/secret.md",
		"/docs/content?path=docs/../../.private/secret.md",
		"/docs/content?path=..%2F.private%2Fsecret.md",
	} {
		_, body := do(target)
		if strings.Contains(body, "SECRET=traversal-found-me") {
			t.Errorf("secret leaked via %s", target)
		}
	}
}

// Specs→archive twin: kanban doc_refs often keep `.skillgrid/specs/<change>/…`
// after the change ships to archive/. Content must resolve the twin; the tree
// must still hide archive/.
func TestContentFallsBackToArchiveTwin(t *testing.T) {
	h := newMDHandler(t)
	do := func(target string) (int, string) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
		return rr.Code, rr.Body.String()
	}

	code, body := do("/docs/content?path=.skillgrid/specs/old-change/briefing.md")
	if code != http.StatusOK {
		t.Fatalf("archive twin: got %d %s", code, body)
	}
	if !strings.Contains(body, `"path":".skillgrid/archive/old-change/briefing.md"`) {
		t.Fatalf("expected served path to be archive twin, got %s", body)
	}
	if !strings.Contains(body, "Archived.") {
		t.Fatalf("expected archive body, got %s", body)
	}

	// Tree must still hide archive/ (content fallback is not a tree listing).
	code, body = do("/docs/tree?root=skillgrid")
	if code != http.StatusOK {
		t.Fatalf("tree: got %d %s", code, body)
	}
	if strings.Contains(body, "archive") || strings.Contains(body, "old-change") {
		t.Fatalf("archive must stay hidden from tree, got %s", body)
	}
}
