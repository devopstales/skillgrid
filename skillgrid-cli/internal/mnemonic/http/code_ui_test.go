package http

import (
	"strings"
	"testing"
)

// 05.1 [AFK] Code menu entry is live: freshness banner (last-indexed + stale
// flag) with a Re-index action.
func TestStep05_CodeFreshness(t *testing.T) {
	h := newHandler(t)
	_, body := getUI(t, h, "/")
	// Code entry is live: no stub tag, no aria-disabled.
	if strings.Contains(body, "Code <span class=\"stub-tag\">") {
		t.Error("code entry must be live (still shows P5 stub tag)")
	}
	if strings.Contains(body, `href="/code" data-route="code" aria-disabled="true"`) {
		t.Error("code entry must not be aria-disabled once live")
	}
	mustContain(t, body, `data-route="code"`, `href="/code"`)
	_, js := getUI(t, h, "/app.js")
	// Entry is live: removed from STUBS, added to LIVE, render() routes it.
	mustContain(t, js,
		"renderCode", "/code/status", "Last indexed:", "Re-index",
		"code-reindex", "code-stale", "STALE",
		`"welcome", "tracker", "docs", "memory", "code"`,
	)
	if strings.Contains(js, "code: { phase:") {
		t.Error("code must be removed from STUBS once live")
	}
	// The render() dispatcher has a parallel code branch (microtask-deferred).
	mustContain(t, js, `route === "code"`, "renderCode(stub)")
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".code-banner", ".code-stale-badge", ".code-btn")
}

// 05.2 [AFK] Status card, BM25 search, and source view. A search failure
// renders an isolated error in the results area, not a blank page.
func TestStep05_CodeSearch(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	// Search + source markers: routes, result rows, source view, read route.
	mustContain(t, js,
		"/code/search", "/code/read", "code-q",
		"codeResultsHtml", "codeHitRow", "codeOpenSource",
		"code-pre", "code-source-card",
		"start_line", "end_line",
	)
	// Status card stat tiles.
	mustContain(t, js, "file_count", "chunk_count", "code-stat-tile")
	// Isolated search-failure path: a catch renders "Search failed" into the
	// results box (not a blank page).
	mustContain(t, js, "Search failed", "code-results")
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".code-page", ".code-grid", ".code-hit", ".code-results", ".code-source", ".code-stat-tiles")
}

// 05.3 [AFK] Forward-compat graph placeholder: collapsed <details> labeled
// "Code graph (coming in 010)"; expanding shows the /code/files list fallback.
func TestStep05_GraphPlaceholder(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"codeLoadGraph", "codeLoadFiles", "/code/files",
		"Code graph (coming in 010)",
		`<details class="code-graph-placeholder">`,
		"code-file-list", "file-list fallback",
	)
	// It reuses the shared forward-compat "not available yet" idiom.
	mustContain(t, js, "mem-placeholder-avail")
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".code-graph-placeholder", ".code-file-list", ".code-file-item")
}

// 05.4 [AFK] Show-numbers table twin on Code widgets + per-widget error
// isolation (each code widget has its own try/catch error render).
func TestStep05_CodeWidgets(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	// Show-numbers twin wired for the code results widget.
	mustContain(t, js,
		"showNumbers", "code-raw", "JSON.stringify", "show numbers",
		"code-raw-toggle", "code.showNumbers",
	)
	// Per-widget isolation: each async code widget has a try/catch that renders
	// an error inside its own container (banner/status, search, source, files).
	mustContain(t, js,
		"codeLoadStatus", "codeLoadSearch", "codeOpenSource", "codeLoadFiles",
		"Status unavailable", "Search failed",
		"Could not read", "File list unavailable",
		`class="code-error"`,
	)
	// Re-index is write-gated via the token-attaching fetch; a 401 surfaces as
	// an inline banner message, not a crash.
	mustContain(t, js, "codeReindex", "memWrite", "/code/index", "Re-index failed")
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".code-error {", ".code-raw")
}
