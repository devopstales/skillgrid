package http

import (
	"embed"
	"net/http"
	"path"
	"strings"
)

// uiDistFS embeds the Vite build output (built by `task ui:build` into
// ui/dist/). The directory must exist at `go build` time — build:all orders
// ui:build first; a plain `go build` without a prior ui:build fails by design.
//
//go:embed all:ui/dist
var uiDistFS embed.FS

//go:embed ui/openapi.yaml
var uiOpenAPISpec []byte

//go:embed all:ui/swagger
var uiSwaggerFS embed.FS

// apiPrefixes are the top-level API route groups. Unknown tails under these
// prefixes return 404 JSON (never the SPA shell), so the client can distinguish
// a missing API resource from a client-router path. Exact routes registered
// elsewhere on the mux take precedence over these wildcards (Go 1.22 mux
// specificity), so this list only catches the gaps.
var apiPrefixes = []string{
	"sessions", "memory", "observations", "code", "web", "projects",
	"prompts", "relations", "context", "health", "tracker", "docs",
}

// registerUIRoutes mounts the embedded SPA + openapi + swagger assets.
// Order matters: exact routes and API-prefix 404 catch-alls are registered
// before the final SPA fallback.
func (s *Server) registerUIRoutes() {
	s.mux.HandleFunc("GET /openapi.yaml", s.handleOpenAPI)
	// /swagger and /swagger/ both serve the entry page; {file...} covers the
	// flat bundle files (Go 1.22 mux: the two exact patterns coexist with the
	// wildcard because a trailing-slash exact pattern is not a prefix of the
	// wildcard's matches... it IS — so register the wildcard under a distinct
	// subtree by serving the index via a redirect-free exact route only).
	s.mux.HandleFunc("GET /swagger/{file}", s.handleSwaggerAsset)
	s.mux.HandleFunc("GET /swagger", s.handleSwaggerIndex)
	s.mux.HandleFunc("GET /swagger/", s.handleSwaggerIndex)
	s.mux.HandleFunc("GET /assets/{rest...}", s.handleUIAsset)

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

	// Final fallback: ANY unmatched GET path (including /) serves the SPA
	// shell so the client router can take over (/tracker, /plans/whatever,
	// /git). Exact routes and the API-prefix 404 catch-alls registered above
	// take precedence by specificity.
	s.mux.HandleFunc("GET /{rest...}", s.handleShellPage)
}

// handleShellPage serves the embedded SPA index.html.
func (s *Server) handleShellPage(w http.ResponseWriter, r *http.Request) {
	data, err := uiDistFS.ReadFile("ui/dist/index.html")
	if err != nil {
		http.Error(w, "dashboard shell missing from binary (run task ui:build)", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// handleUIAsset serves content-hashed build assets from the embedded dist.
// Multi-segment rest (e.g. a future nested asset dir) is allowed; traversal
// is impossible because the path is resolved inside the embedded FS.
func (s *Server) handleUIAsset(w http.ResponseWriter, r *http.Request) {
	rest := r.PathValue("rest")
	if rest == "" || rest == "." || rest == ".." || strings.HasPrefix(rest, "../") {
		http.NotFound(w, r)
		return
	}
	data, err := uiDistFS.ReadFile("ui/dist/assets/" + rest)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if ct := mimeByExt(rest); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	_, _ = w.Write(data)
}

func mimeByExt(name string) string {
	switch path.Ext(name) {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".svg":
		return "image/svg+xml"
	case ".woff2":
		return "font/woff2"
	case ".map":
		return "application/json"
	default:
		return ""
	}
}

// handleOpenAPI serves the embedded OpenAPI spec.
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(uiOpenAPISpec)
}

// handleSwaggerIndex serves the swagger-ui entry page.
func (s *Server) handleSwaggerIndex(w http.ResponseWriter, r *http.Request) {
	s.serveSwaggerFile(w, r, "index.html")
}

// handleSwaggerAsset serves a single flat file from the embedded swagger
// bundle; subdirectories are rejected (the bundle is flat).
func (s *Server) handleSwaggerAsset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("file")
	if name == "" || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}
	s.serveSwaggerFile(w, r, name)
}

func (s *Server) serveSwaggerFile(w http.ResponseWriter, r *http.Request, name string) {
	data, err := uiSwaggerFS.ReadFile("ui/swagger/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ct := mimeByExt(name)
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	_, _ = w.Write(data)
}
