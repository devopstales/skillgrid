package http

import (
	"net/http"
	"strings"
	"testing"
)

// 02.10 [AFK] Tracker menu entry markup + renderer ship in P2 (demo-faithful:
// provider switcher, search + priority filter, 4-column board, rich cards,
// slide-over detail).
func TestStep02_TrackerUI(t *testing.T) {
	h := newHandler(t)
	_, body := getUI(t, h, "/")
	// Tracker entry is live: no stub tag, no aria-disabled.
	if strings.Contains(body, "Tracker <span class=\"stub-tag\">") {
		t.Error("tracker entry must be live (still shows P2 stub tag)")
	}
	mustContain(t, body, `data-route="tracker"`, `href="/tracker"`)
	_, js := getUI(t, h, "/app.js")
	mustContain(t, js,
		"renderTracker", "TRK_PROVIDERS", "TRK_COLUMNS",
		"trk-switcher", "trk-q", "trk-prio", "trk-board", "trk-card",
		"trk-prio-badge", "trk-dialog", "trk-panel", "trk-desc",
		"To Do", "In Progress", "Blocked", "Done",
		"/tracker/config?provider=", "?provider=",
		"Move to",
	)
	if strings.Contains(js, "tracker: { phase:") {
		t.Error("tracker must be removed from STUBS once live")
	}
	_, css := getUI(t, h, "/app.css")
	mustContain(t, css, ".trk-switcher", ".trk-board", ".trk-card", ".trk-panel", ".trk-prio-badge", ".trk-dialog")
}

// 02.11 [AFK] Tracker degraded states + per-widget error isolation ship in P2.
func TestStep02_TrackerDegraded(t *testing.T) {
	h := newHandler(t)
	_, js := getUI(t, h, "/app.js")
	// Not-connected + load-failed empty states; Esc/backdrop close the dialog.
	mustContain(t, js,
		"is not connected", "Failed to load tasks",
		"Escape", "data-close",
	)
}

// 02.12 [AFK] openapi.yaml documents the tracker routes with examples.
func TestStep02_OpenAPI(t *testing.T) {
	h := newHandler(t)
	code, body := getUI(t, h, "/openapi.yaml")
	if code != http.StatusOK {
		t.Fatalf("expected /openapi.yaml 200, got %d", code)
	}
	mustContain(t, body,
		"/tracker/config", "/tracker/tasks", "/tracker/tasks/{id}",
		"/backlog/config", "/backlog/tasks", "/backlog/tasks/{id}",
		"TrackerItem", "TrackerConfig", "trackerSetStatus",
	)
}
