package http

import (
	"net/http"
	"strings"
	"testing"
)

// 04.1 [AFK] Memory menu entry is live (search, results, detail, .mem-* styles).
func TestStep04_MemorySearch(t *testing.T) {
	h := newHandler(t)
	_, body := getUI(t, h, "/")
	// Memory entry is live: no stub tag, no aria-disabled.
	if strings.Contains(body, "Memory <span class=\"stub-tag\">") {
		t.Error("memory entry must be live (still shows P4 stub tag)")
	}
	if strings.Contains(body, `href="/memory" data-route="memory" aria-disabled="true"`) {
		t.Error("memory entry must not be aria-disabled once live")
	}
	mustContain(t, body, `data-route="memory"`, `href="/memory"`)
	_, js := getUI(t, h, "/app.js")
	// Entry + search input + results render + markdown reuse + LIVE route.
	mustContain(t, js,
		"renderMemory", "mem-q", "memResultsHtml", "mdToHtml",
		`"welcome", "tracker", "docs", "memory"`,
		"/search?", "mem-item",
	)
	if strings.Contains(js, "memory: { phase:") {
		t.Error("memory must be removed from STUBS once live")
	}
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".mem-page", ".mem-grid", ".mem-item", ".mem-results", ".mem-detail")
}

// 04.2 [AFK] Pin/unpin, delete (confirm), and re-fetch after the action.
func TestStep04_MemoryActions(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		`/pin`, `/unpin`, `confirm(`, "memWrite",
		"memRenderMain", // re-fetch / re-render after the action
	)
}

// 04.3 [AFK] Relations drill-down with a confidence badge and click navigation.
func TestStep04_Relations(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"memLoadRelations", "memRelRow", "/relations/",
		"mem-conf", "confidence", "memOpenDetail",
	)
}

// 04.4 [AFK] Empty state shows suggested prompts from /context as clickable chips.
func TestStep04_SuggestedPrompts(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"memSuggestedPromptsHtml", "/context", "mem-prompt-chip", "Suggested prompts",
	)
}

// 04.5 [AFK] Show-numbers / raw-JSON toggle helper + the pre class.
func TestStep04_ShowNumbers(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js, "showNumbers", "mem-raw", "JSON.stringify", "show numbers")
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".mem-raw")
}

// 04.9 [AFK] Web-cache sub-section preserved inside Memory.
func TestStep04_WebCache(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"memRenderWeb", "/web/search", "/web/status", "Web cache",
	)
}

// 04.16/04.17 [AFK] Share + status writes attach the bearer token (the tracker
// write pattern) and call /share + /status.
func TestStep04_GovernanceWriteGated(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		`"Authorization"`, "Bearer ", "sgmn-token",
		"/share", "/status", "memWriteHeaders",
	)
}

// 04.11 [AFK] In-place edit (PATCH with content) + version-history render.
func TestStep04_EditAppendsVersion(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"PATCH", `{ content:`, "mem-edit-text", "mem-save",
		"versions", "Version history", "/governance",
	)
}

// 04.16 [AFK] Share control: visibility selector + explicit confirm + 400 reason.
func TestStep04_Share(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"mem-visibility", `confirm(`, "Share", "mem-share-error",
		"private", "team", "restricted", "agent",
	)
}

// 04.13 [AFK] Forward-compat placeholder helper used by the two pending widgets.
func TestStep04_ForwardCompatPlaceholder(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"memPlaceholder",
		"Layer drill-down", "Agent loadout",
	)
}

// 04.14 [AFK] Asset-library columns: owner / version / status / usage / visibility.
func TestStep04_AssetLibrary(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"memRow", "revision_count", "retrieval_usage", "owner", "visibility", "status",
	)
}

// 04.15 [AFK] Layer drill-down placeholder (L0→L3) + the flat pre-013 view.
func TestStep04_LayerDrilldown(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js, "Layer drill-down", "L0", "L3", "flat pre-013 view")
}

// 04.16 [AFK] Restricted visibility exposes an ACL/grants editor.
func TestStep04_ShareControl(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js, "mem-grantee", "grants", "restricted", "Grants")
}

// 04.17 [AFK] Status selector (active/superseded/archived) + set handler.
func TestStep04_ReviewStatus(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"mem-status", "active", "superseded", "archived", "mem-status-btn",
	)
}

// 04.18 [AFK] Agent-loadout placeholder (013 bindings not present yet).
func TestStep04_Loadout(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js, "Agent loadout", "visibility=agent bindings")
}

// 04.12 [AFK] openapi.yaml documents the new memory routes + schemas.
func TestStep04_OpenAPI(t *testing.T) {
	h := newHandler(t)
	code, body := getUI(t, h, "/openapi.yaml")
	if code != http.StatusOK {
		t.Fatalf("expected /openapi.yaml 200, got %d", code)
	}
	mustContain(t, body,
		"/observations/{id}",
		"/memory/observations/{id}/pin",
		"/memory/observations/{id}/unpin",
		"/memory/observations/{id}/share",
		"/memory/observations/{id}/status",
		"/memory/observations/{id}/governance",
		"/relations/{id}",
		"Governance", "Relation",
	)
}
