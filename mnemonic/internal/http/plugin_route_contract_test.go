package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// dollarBraceRe matches any ${...} template placeholder so a source check can
// normalize it to the canonical {id} path segment.
var dollarBraceRe = regexp.MustCompile(`\$\{[^}]*\}`)

// TestNativePluginRouteContract guards the native TS plugins against a stale
// API surface: every endpoint a plugin fetches must resolve to a registered
// route on the Go mux. A plugin path typo would otherwise 404 silently in the
// live harness (WINDOWS W020).
//
// Two checks:
//  1. Source contract — each pinned endpoint appears in the plugin source
//     (template `${id}` is accepted in place of the path segment `{id}`).
//  2. Registration contract — each endpoint is a registered route, detected by
//     method-presence probing: a registered path returns 405 (method-not-allowed)
//     for the wrong method, while an unregistered path falls through to a 404.
func TestNativePluginRouteContract(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	s := NewServer(svc)
	h := s.Handler()

	root := findRepoRootFromPluginDir(t)

	// Expected per-plugin endpoints, pinned to the real route map. `{id}` is
	// the registered Go path segment; `${id}` is the JS template equivalent.
	expected := map[string][]string{
		"skillgrid-squad.ts": {
			"/teams/tasks",
			"/teams/tasks/pull",
			"/teams/tasks/{id}/output",
			"/teams/tasks/{id}/reviews",
			"/teams/tasks/{id}/done",
		},
		"skillgrid-events.ts": {
			"/policy/evaluate",
		},
		"mnemonic-memory.ts": {
			"/observations",
			"/memory/observations/{id}",
			"/search",
			"/facts/search",
			"/sessions",
			"/sessions/{id}/end",
			"/sessions/{id}/summary",
		},
		"skillgrid-compaction.ts": {
			"/context/compaction",
			"/sessions",
			"/sessions/{id}/end",
		},
		"mnemonic-codeindex.ts": {
			"/code/status",
			"/code/index",
			"/code/files",
		},
	}

	// (1) Source contract.
	for name, paths := range expected {
		raw, err := os.ReadFile(filepath.Join(root, "plugins", "opencode", name))
		if err != nil {
			t.Fatalf("%s: cannot read plugin source: %v", name, err)
		}
		src := string(raw)
		for _, p := range paths {
			// Normalize the source so any ${...} template placeholder (e.g.
			// ${args.task_id}) becomes {id}, then match the pinned endpoint.
			// This tolerates the plugin's variable names.
			canonical := dollarBraceRe.ReplaceAllString(src, "{id}")
			if strings.Contains(canonical, p) {
				continue
			}
			// Some plugins build paths by string concatenation, e.g.
			// "/sessions/" + sessionID + "/end", so the full path is not
			// contiguous in source. As a fallback, require the literal
			// segments around the id to all be present.
			segs := strings.Split(p, "/")
			var all bool
			all = true
			for _, seg := range segs {
				if seg == "" || seg == "{id}" {
					continue
				}
				if !strings.Contains(canonical, "/"+seg) && !strings.Contains(canonical, seg+"\"") && !strings.Contains(canonical, seg+"'") {
					all = false
					break
				}
			}
			if !all {
				t.Errorf("%s: expected endpoint %q not referenced in plugin source (stale contract?)", name, p)
			}
		}
	}

	// (2) Registration contract: every unique endpoint must be a registered
	// route. Probe with a method the route does not allow (e.g. DELETE on a
	// POST-only path): a registered path yields 405, an unregistered path 404.
	seen := map[string]bool{}
	var endpoints []string
	for _, paths := range expected {
		for _, p := range paths {
			// Probe the concrete path: substitute the {id} segment with a token
			// that cannot be a real resource id.
			concrete := strings.ReplaceAll(p, "{id}", "zzzprobe")
			if !seen[concrete] {
				seen[concrete] = true
				endpoints = append(endpoints, concrete)
			}
		}
	}
	for _, concrete := range endpoints {
		if status := probeMethod(h, concrete); status == http.StatusNotFound {
			t.Errorf("%s: endpoint is not a registered route (404 on method-presence probe) — route renamed or plugin path stale", concrete)
		}
	}
}

// probeMethod issues a DELETE to path and returns the status. A registered
// route that does not accept DELETE answers 405; an unregistered path 404s.
func probeMethod(h http.Handler, path string) int {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code
}

// findRepoRootFromPluginDir locates the repo root from this test's location so
// the test reads the committed plugin sources rather than a temp copy.
func findRepoRootFromPluginDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "plugins", "opencode", "skillgrid-squad.ts")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate repo root containing plugins/opencode/ from %s", wd)
	return ""
}
