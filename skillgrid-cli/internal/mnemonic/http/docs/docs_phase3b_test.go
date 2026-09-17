package docs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 3b.1 [AFK] The backlogdocs root exposes .backlog/decisions (ADR) and
// .backlog/docs (doc-NNN/PRD), and root=scoping does not leak across roots.
func TestPhase3b_BacklogDocsTree(t *testing.T) {
	h := newMDHandler(t)
	do := func(target string) (int, string) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
		return rr.Code, rr.Body.String()
	}

	// root=backlogdocs shows the ADR + the doc
	code, body := do("/docs/tree?root=backlogdocs")
	if code != http.StatusOK {
		t.Fatalf("backlogdocs tree: expected 200, got %d (%s)", code, body)
	}
	for _, want := range []string{
		"decision-1", "accepted", "doc-001",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("backlogdocs tree missing %q in %s", want, body)
		}
	}
	// it must not leak the tasks root or the sdd root
	if strings.Contains(body, "TASK-001") {
		t.Errorf("backlogdocs tree leaked the tasks root: %s", body)
	}
	if strings.Contains(body, "briefing.md") {
		t.Errorf("backlogdocs tree leaked the sdd root: %s", body)
	}

	// root=all still includes both new artifacts alongside the old roots
	_, body = do("/docs/tree")
	for _, want := range []string{"decision-1", "doc-001", "TASK-001", "briefing.md"} {
		if !strings.Contains(body, want) {
			t.Errorf("all tree missing %q in %s", want, body)
		}
	}
}

// 3b.2 [AFK] An ADR served via /docs/content carries docType=adr +
// decisionStatus (lower-cased) from its frontmatter; the body keeps its
// Context/Decision/Consequences sections for schema-aware rendering.
func TestPhase3b_ADRContent(t *testing.T) {
	h := newMDHandler(t)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
		"/docs/content?path=.backlog/decisions/decision-1%20-%20Use-Tailwind-v4.md", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("adr content: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var out MDContent
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.DocType != DocTypeADR {
		t.Errorf("adr docType: expected %q, got %q", DocTypeADR, out.DocType)
	}
	if out.DecisionStatus != "accepted" {
		t.Errorf("adr decisionStatus: expected \"accepted\", got %q", out.DecisionStatus)
	}
	for _, want := range []string{"## Context", "## Decision", "## Consequences"} {
		if !strings.Contains(out.Body, want) {
			t.Errorf("adr body missing %q", want)
		}
	}
}

// 3b.3 [AFK] A PRD under .backlog/docs is classified docType=prd; a plain
// doc-NNN is docType=doc; a Backlog task stays docType=task.
func TestPhase3b_DocTypeClassification(t *testing.T) {
	h := newMDHandler(t)
	get := func(path string) MDContent {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/content?path="+path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("content %s: expected 200, got %d (%s)", path, rr.Code, rr.Body.String())
		}
		var out MDContent
		if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return out
	}

	// A PRD: detected by id "prd-..." (frontmatter id, not path)
	prdPath := ".backlog/docs/prd-001%20-%20Dashboard%20PRD.md"
	// the fixture uses doc-001 (a guide), so add a PRD classification check via
	// the real seeded doc-001 (doc) and a synthetic PRD via id prefix.
	doc := get(".backlog/docs/doc-001%20-%20Guide.md")
	if doc.DocType != DocTypeDoc {
		t.Errorf("doc-001 docType: expected %q, got %q", DocTypeDoc, doc.DocType)
	}

	// the seeded ADR is already covered above; a Backlog task must be "task"
	task := get(".backlog/tasks/TASK-001-something.md")
	if task.DocType != DocTypeTask {
		t.Errorf("task docType: expected %q, got %q", DocTypeTask, task.DocType)
	}
	_ = prdPath
}

// 3b.4 [AFK] PRD classification: a doc whose frontmatter id begins with "prd"
// (or whose title mentions PRD) is docType=prd.
func TestPhase3b_PRDContent(t *testing.T) {
	cwd := t.TempDir()
	w := func(rel, content string) {
		p := filepath.Join(cwd, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w(".backlog/docs/prd-001 - Dashboard PRD.md",
		"---\nid: prd-001\ntitle: Dashboard PRD\ntype: prd\n---\n\n# Dashboard PRD\n\nProblem.\n")

	h := http.NewServeMux()
	h.Handle("GET /docs/content", NewContent(cwd))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
		"/docs/content?path=.backlog/docs/prd-001%20-%20Dashboard%20PRD.md", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("prd content: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var out MDContent
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.DocType != DocTypePRD {
		t.Errorf("prd docType: expected %q, got %q", DocTypePRD, out.DocType)
	}
}
