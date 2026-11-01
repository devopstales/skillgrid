package http

import (
	"net/http"
)

// apiPrefixes are the top-level API route groups. Unknown tails under these
// prefixes return 404 JSON (never a shell page), so the client can distinguish
// a missing API resource from a client-router path. Exact routes registered
// elsewhere on the mux take precedence over these wildcards (Go 1.22 mux
// specificity), so this list only catches the gaps.
var apiPrefixes = []string{
	"sessions", "memory", "observations", "code", "web", "projects",
	"prompts", "relations", "context", "health", "tracker", "docs",
	// NOTE (review B2): the Phase 6/7 read bridges (activity/plans/git/
	// specs) are deliberately NOT in this list — they are
	// SPA-routed (a browser GET serves the shell so the client router can
	// handle the sub-path), and adding them here registers a conflicting
	// `GET /{prefix}/{rest...}` wildcard against their explicit routes.
}

// registerUIRoutes mounts the API-prefix 404 catch-alls and the final GET
// fallback. The embedded SPA was removed; a browser GET on an unmatched path
// gets a minimal "no UI" page instead of the SPA shell.
func (s *Server) registerUIRoutes() {
	for _, p := range apiPrefixes {
		p := p
		s.mux.HandleFunc("GET /"+p+"/{rest...}", func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusNotFound, "not found")
		})
	}
	// The mux clean-path rule redirects a bare "/tracker" (or "/docs") to the
	// trailing-slash form before the wildcard can serve it; the explicit
	// routes win over the redirect.
	s.mux.HandleFunc("GET /tracker", s.handleShellPage)
	s.mux.HandleFunc("GET /docs", s.handleShellPage)

	// Final fallback: ANY unmatched GET path (including /) serves the
	// minimal "no UI" page. Exact routes and the API-prefix 404 catch-alls
	// registered above take precedence by specificity.
	s.mux.HandleFunc("GET /{rest...}", s.handleShellPage)
}

// handleShellPage serves a minimal "no UI" page for browser navigation. The
// embedded SPA was removed; the server is a pure JSON API plus the docs/
// prototype HTML routes.
func (s *Server) handleShellPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Skillgrid</title></head>
<body><p>The embedded UI has been removed. This server is a JSON API. See <a href="/docs">/docs</a> for the docs bridge.</p></body></html>
`))
}
