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

// 3b.2 [AFK] An ADR served via /docs/content carries docType=adr +
// decisionStatus (lower-cased) from its frontmatter; the body keeps its
// Context/Decision/Consequences sections for schema-aware rendering.
func TestPhase3b_ADRContent(t *testing.T) {
	h := newMDHandler(t)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
		"/docs/content?path=.skillgrid/adr/0007-sdd-docs.md", nil))
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
	for _, want := range []string{"## Context and Problem Statement", "## Decision Outcome"} {
		if !strings.Contains(out.Body, want) {
			t.Errorf("adr body missing %q", want)
		}
	}
}

// 3b.2b [AFK] The skillgrid root surfaces first-class ADR + PRD + spec artifacts
// (specs/, adr/, prd/) and hides everything else (archive/, config.yaml, ...);
// ADRs/PRDs classify as adr/prd with their decisionStatus.
func TestPhase3b_SkillgridRootDocs(t *testing.T) {
	h := newMDHandler(t)
	do := func(target string) (int, string) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
		return rr.Code, rr.Body.String()
	}

	// root=skillgrid includes the numeric ADR + PRD + spec file
	code, body := do("/docs/tree?root=skillgrid")
	if code != http.StatusOK {
		t.Fatalf("skillgrid tree: expected 200, got %d (%s)", code, body)
	}
	for _, want := range []string{"0007-sdd-docs", "0001-dashboard", "briefing.md", "specs", "adr", "prd"} {
		if !strings.Contains(body, want) {
			t.Errorf("skillgrid tree missing %q in %s", want, body)
		}
	}
	// archive/ and the hidden .skillgrid subdirs must NOT appear
	for _, gone := range []string{"archive", "glossary", "config.yaml"} {
		if strings.Contains(body, gone) {
			t.Errorf("skillgrid tree should hide %q in %s", gone, body)
		}
	}

	// the ADR classifies as adr with decisionStatus from frontmatter
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
		"/docs/content?path=.skillgrid/adr/0007-sdd-docs.md", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("skillgrid adr content: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var adr MDContent
	if err := json.Unmarshal(rr.Body.Bytes(), &adr); err != nil {
		t.Fatalf("decode adr: %v", err)
	}
	if adr.DocType != DocTypeADR {
		t.Errorf("skillgrid adr docType: expected %q, got %q", DocTypeADR, adr.DocType)
	}
	if adr.DecisionStatus != "accepted" {
		t.Errorf("skillgrid adr decisionStatus: expected \"accepted\", got %q", adr.DecisionStatus)
	}

	// the PRD classifies as prd
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
		"/docs/content?path=.skillgrid/prd/0001-dashboard.md", nil))
	var prd MDContent
	if err := json.Unmarshal(rr.Body.Bytes(), &prd); err != nil {
		t.Fatalf("decode prd: %v", err)
	}
	if prd.DocType != DocTypePRD {
		t.Errorf("skillgrid prd docType: expected %q, got %q", DocTypePRD, prd.DocType)
	}
}

// 3b.3 [AFK] A Backlog task classifies docType=task; a Backlog.md-style doc-NNN
// classifies docType=doc; an ADR in .skillgrid/adr classifies docType=adr.
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

	// a Backlog task must be "task"
	task := get(".backlog/tasks/TASK-001-something.md")
	if task.DocType != DocTypeTask {
		t.Errorf("task docType: expected %q, got %q", DocTypeTask, task.DocType)
	}

	// the skillgrid ADR must be "adr"
	adr := get(".skillgrid/adr/0007-sdd-docs.md")
	if adr.DocType != DocTypeADR {
		t.Errorf("skillgrid adr docType: expected %q, got %q", DocTypeADR, adr.DocType)
	}

	// a plain doc-NNN must be "doc"
	doc := get("docs/guide.md")
	if doc.DocType != DocTypeNote && doc.DocType != DocTypeDoc {
		t.Errorf("docs/guide.md docType: expected doc/note, got %q", doc.DocType)
	}
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
	w(".skillgrid/prd/0002-dashboard.md",
		"---\nid: 0002\ntitle: Dashboard PRD\ntype: prd\n---\n\n# Dashboard PRD\n\nProblem.\n")

	h := http.NewServeMux()
	h.Handle("GET /docs/content", NewContent(cwd))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
		"/docs/content?path=.skillgrid/prd/0002-dashboard.md", nil))
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
