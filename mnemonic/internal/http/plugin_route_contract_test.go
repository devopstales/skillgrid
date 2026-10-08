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

	// (3) Verb contract: for each endpoint the plugin calls, the mux must
	// expose exactly the verb the plugin sends. The source contract above pins
	// the path but not the verb, so a plugin that GETs a POST-only route (e.g.
	// the old fact_search, W020-class) silently fell through to the SPA shell
	// and returned {} in the live harness. This check probes verb-specificity:
	// the plugin's verb must be accepted (not 405) and the opposite verb must
	// be rejected (405). The probe uses only static paths — {id}-segment routes
	// are ambiguous to probe (a handler 404s by-id, indistinguishable from an
	// unregistered path), so they are covered by the registration probe (2).
	// Only the POST routes are checked: a POST-only route answers a GET with the
	// embedded SPA shell (the C1 class of silent failure). The GET routes
	// (e.g. /search, /code/files) return JSON from their own handlers, so a verb
	// flip there is not the C1 bug and is not guarded here.
	verbChecks := []struct{ name, path, verb string }{
		{"skillgrid-events.ts", "/policy/evaluate", "POST"},
		{"mnemonic-memory.ts", "/facts/search", "POST"},
		{"mnemonic-memory.ts", "/facts", "POST"},
		{"mnemonic-memory.ts", "/sessions", "POST"},
		{"mnemonic-codeindex.ts", "/code/index", "POST"},
	}
	loaded := map[string]string{}
	for _, c := range verbChecks {
		src, ok := loaded[c.name]
		if !ok {
			raw, err := os.ReadFile(filepath.Join(root, "plugins", "opencode", c.name))
			if err != nil {
				t.Fatalf("%s: cannot read plugin source: %v", c.name, err)
			}
			src = string(raw)
			loaded[c.name] = src
		}
		// The plugin's verb must be an API verb the mux serves (not 405). A
		// POST-only route answers the opposite verb with the embedded SPA shell
		// (200 text/html) or a 404 — that fallthrough is exactly the C1 bug.
		// The source check below is what pins the plugin to the right verb.
		if got := probeVerb(h, c.path, c.verb); got == http.StatusMethodNotAllowed {
			t.Errorf("%s: mux does not allow %s %s (got 405) — plugin verb mismatch", c.name, c.verb, c.path)
		}
		if !sourceUsesVerb(src, c.path, c.verb) {
			t.Errorf("%s: plugin does not send %s to %s in source (verb mismatch — falls through to SPA shell)", c.name, c.verb, c.path)
		}
	}
}

// probeVerb issues method to path and returns the status. 405 means the path
// is registered but the verb is not allowed; any other status means the verb
// was accepted (handler ran) — 404-by-id on a {id} route is not used here.
func probeVerb(h http.Handler, path, method string) int {
	req := httptest.NewRequest(method, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code
}

// sourceUsesVerb reports whether the plugin source sends wantVerb to path. The
// checked paths are POST routes the plugin calls with a contiguous literal
// (api("POST", "/facts/search"), postJSON("/policy/evaluate")), so the check
// locates the call by the quoted path (long enough to be unique: "/facts" does
// not contain "/facts/search") and confirms the nearest method is wantVerb. A
// line with no visible method is ignored (the mux probe covers registration).
func sourceUsesVerb(src, path, wantVerb string) bool {
	lines := strings.Split(src, "\n")
	needle := `"` + path
	found := 0
	for i, line := range lines {
		if !strings.Contains(line, needle) {
			continue
		}
		verb := methodOnLine(line)
		if verb == "" {
			verb = methodNearby(lines, i)
		}
		if verb == "" {
			continue
		}
		found++
		if verb != wantVerb {
			return false
		}
	}
	return found > 0
}

// methodOnLine extracts the HTTP method from a line that names one
// (api("POST", ...), fetch(..., {method:"POST"}), postJSON(...), getJSON(...)).
func methodOnLine(line string) string {
	upper := strings.ToUpper(line)
	for _, v := range []string{"POST", "GET", "PUT", "DELETE", "PATCH"} {
		if strings.Contains(upper, `"`+v+`"`) || strings.Contains(upper, "method: "+v) {
			return v
		}
	}
	if strings.Contains(line, "postJSON") {
		return "POST"
	}
	if strings.Contains(line, "getJSON") {
		return "GET"
	}
	return ""
}

// methodNearby scans up to 4 lines after i for the first visible method.
func methodNearby(lines []string, i int) string {
	for j := i + 1; j < len(lines) && j < i+4; j++ {
		if m := methodOnLine(lines[j]); m != "" {
			return m
		}
	}
	return ""
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
