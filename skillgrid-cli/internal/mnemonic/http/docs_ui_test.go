package http

import (
	"net/http"
	"strings"
	"testing"
)

// 03.3 [AFK] Docs menu entry: change list → click → change.md + tasks.md
// viewer ship in P3 (Docs is a live entry, not a stub).
func TestStep03_DocsUI(t *testing.T) {
	h := newHandler(t)
	_, body := getUI(t, h, "/")
	if strings.Contains(body, "Docs <span class=\"stub-tag\">") {
		t.Error("docs entry must be live (still shows P3 stub tag)")
	}
	mustContain(t, body, `data-route="docs"`, `href="/docs"`)
	if strings.Contains(body, `href="/docs" data-route="docs" aria-disabled="true"`) {
		t.Error("docs entry must not be aria-disabled once live")
	}
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"renderDocs", "docQuery", "docOpen",
		"/docs/changes", "/docs/changes/",
		"doc-list", "doc-row", "doc-view", "doc-md",
		"change.md", "tasks.md",
		"Back to changes",
		"mdToHtml", // rendered markdown view, not plain text
	)
	// The docs viewer must render markdown (not dump raw text into a <pre>).
	if !strings.Contains(js, `mdToHtml(d.change_md)`) || !strings.Contains(js, `mdToHtml(d.tasks_md)`) {
		t.Error("docs viewer must render change.md/tasks.md as markdown via mdToHtml")
	}
	if strings.Contains(js, "docs: { phase:") {
		t.Error("docs must be removed from STUBS once live")
	}
	// The stub renderer for live routes must not claim docs is unbuilt.
	if i := strings.Index(js, "Coming in"); i >= 0 && strings.Contains(js[i:i+120], "Docs") {
		t.Error("docs stub copy must be gone")
	}
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".doc-list", ".doc-row", ".doc-view", ".doc-md")
}

// 03.4 [AFK] Two-way tracker links: docs → tracker item via Ticket:; tracker
// detail → docs via the referenced change path.
func TestStep03_Links(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	// Docs → Tracker: the Ticket: id (backticked in change.md) is stripped to
	// its bare id and deep-links to the tracker entry.
	mustContain(t, js, "docStripTicket", "Open tracker item", `/tracker?`)
	// Tracker detail → Docs: change-path doc refs link back into the docs
	// entry with ?change=, and the click handler navigates + closes.
	mustContain(t, js, "trkDocLink", `changes\/([a-z0-9][a-z0-9-]*)`, "data-docchange")
}

// 03.5 [AFK] openapi.yaml documents the docs routes with examples.
func TestStep03_OpenAPI(t *testing.T) {
	h := newHandler(t)
	code, body := getUI(t, h, "/openapi.yaml")
	if code != http.StatusOK {
		t.Fatalf("expected /openapi.yaml 200, got %d", code)
	}
	mustContain(t, body,
		"/docs/changes", "/docs/changes/{name}",
		"DocChange", "DocChangeDetail",
	)
}
