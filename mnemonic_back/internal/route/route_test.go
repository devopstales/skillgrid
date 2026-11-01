package route

import (
	"path/filepath"
	"strings"
	"testing"
)

// dummyIndex resolves a handler: a name in `explicit` is a same-file/explicit
// match (EXTRACTED); a name in `globalOnly` is a name-only best-effort guess
// (AMBIGUOUS); anything else is unresolvable (dropped). This mirrors the
// txRouteStore policy (same-file first, then a unique global guess).
type dummyIndex struct {
	explicit   map[string]int64
	globalOnly map[string]int64
	// globalCandidates reports the number of global symbols matched by name
	// for a name that does NOT resolve (logged on drops). Absent → 0.
	globalCandidates map[string]int
}

func (d dummyIndex) ResolveHandler(name string) Resolution {
	if id, ok := d.explicit[name]; ok {
		return Resolution{ID: id, UID: "uid-" + name, Confidence: ConfidenceExtracted}
	}
	if id, ok := d.globalOnly[name]; ok {
		return Resolution{ID: id, UID: "uid-" + name, Confidence: ConfidenceAmbiguous, Candidates: 1}
	}
	return Resolution{Candidates: d.globalCandidates[name]}
}

// known builds an index where every name is an explicit (same-file) match —
// the EXTRACTED case.
func known(names ...string) dummyIndex {
	m := map[string]int64{}
	for i, n := range names {
		m[n] = int64(100 + i)
	}
	return dummyIndex{explicit: m}
}

// ambiguousIndex builds an index where the given names are name-only best-effort
// guesses (AMBIGUOUS) and everything else is unresolvable.
func ambiguousIndex(names ...string) dummyIndex {
	m := map[string]int64{}
	for i, n := range names {
		m[n] = int64(200 + i)
	}
	return dummyIndex{globalOnly: m}
}

// nothingIndex resolves nothing (every reference is unresolvable → dropped).
func nothingIndex() dummyIndex {
	return dummyIndex{}
}

func fileSyms(names ...string) []FileSymbol {
	var out []FileSymbol
	for i, n := range names {
		out = append(out, FileSymbol{Name: n, UID: "f-" + n, ID: int64(i + 1)})
	}
	return out
}

// TestRouteNodesForWebFrameworks covers @step-01 (Scenario: Web-framework
// routing files produce route nodes): each fixture routing file yields route
// nodes, each linked by a references edge to its handler, every edge
// EXTRACTED (explicit syntax), and a query for a handler's URL pattern
// surfaces the route.
func TestRouteNodesForWebFrameworks(t *testing.T) {
	cases := []struct {
		file      string
		framework string // the per-route framework expected on the node
		pattern   string
		handler   string
	}{
		{"urls.py", "django", "/home/", "HomeView"},
		{"app.py", "fastapi", "/items/", "list_items"},
		{"server.js", "express", "/users", "listUsers"},
		{"main.go", "gin", "/health", "healthHandler"},
		{"routes.rb", "rails", "/products", "products#index"},
		{"UserController.java", "spring", "/users", "listUsers"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			src := webFrameworkFixture(tc.file, tc.framework)
			idx := known(tc.handler)
			b := Build(filepath.Base(tc.file), []byte(src), fileSyms(tc.handler, "other"), idx)
			if len(b.Nodes) == 0 {
				t.Fatalf("%s: expected route nodes, got none (src: %s)", tc.file, src)
			}
			found := false
			for _, n := range b.Nodes {
				if n.PathPattern != tc.pattern {
					continue
				}
				found = true
				if n.HandlerSymbol == 0 {
					t.Errorf("%s route %q: handler %q not resolved (references edge dropped)", tc.framework, tc.pattern, tc.handler)
				}
				if n.HandlerName != tc.handler {
					t.Errorf("%s route %q: handler = %q, want %q", tc.framework, tc.pattern, n.HandlerName, tc.handler)
				}
				if !n.HandlerExplicit {
					t.Errorf("%s route %q: explicit route should be EXTRACTED (explicit handler)", tc.framework, tc.pattern)
				}
			}
			if !found {
				t.Errorf("%s: no route node with pattern %q (src: %s)", tc.framework, tc.pattern, src)
			}
		})
	}
}

// webFrameworkFixture returns a minimal routing file for the framework.
func webFrameworkFixture(file, framework string) string {
	switch framework {
	case "django":
		return "from . import views\n\nurlpatterns = [\n    path('home/', views.HomeView.as_view(), name='home_view'),\n]\n"
	case "fastapi":
		return "@app.get('/items/')\ndef list_items():\n    return []\n"
	case "express":
		return "function listUsers() {}\n\napp.get('/users', listUsers);\n"
	case "gin":
		return "func healthHandler(c *gin.Context) {}\n\nr.GET(\"/health\", healthHandler)\n"
	case "rails":
		return "get '/products', to: 'products#index'\n"
	case "spring":
		return "@GetMapping(\"/users\")\npublic List users listUsers() {\n    return null;\n}\n"
	}
	return ""
}

// TestRouteNodesForDjangoExplicit covers @step-01 (explicit Django as_view
// handler is EXTRACTED and links the URL pattern to the handler symbol).
func TestRouteNodesForDjangoExplicit(t *testing.T) {
	src := webFrameworkFixture("urls.py", "django")
	b := Build("urls.py", []byte(src), fileSyms("HomeView", "other"), known("HomeView"))
	if len(b.Nodes) != 1 {
		t.Fatalf("expected 1 route node, got %d", len(b.Nodes))
	}
	n := b.Nodes[0]
	if n.PathPattern != "/home/" {
		t.Errorf("pattern = %q, want /home/", n.PathPattern)
	}
	if n.HandlerName != "HomeView" {
		t.Errorf("handler = %q, want HomeView", n.HandlerName)
	}
	if n.Framework != "django" {
		t.Errorf("framework = %q, want django", n.Framework)
	}
	if !strings.Contains(n.Name, "/home/") {
		t.Errorf("route node name %q should carry the URL pattern", n.Name)
	}
	if !n.HandlerExplicit {
		t.Errorf("Django as_view handler must be EXTRACTED (explicit)")
	}
}
