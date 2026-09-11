package http

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed ui
var uiFS embed.FS

// registerUIRoutes mounts the data viewer at / and Swagger UI at /docs.
func (s *Server) registerUIRoutes() {
	s.mux.HandleFunc("GET /openapi.yaml", s.handleOpenAPI)
	s.mux.HandleFunc("GET /swagger-ui", s.handleSwaggerUI)
	s.mux.HandleFunc("GET /swagger-ui/{file...}", s.handleSwaggerAsset)
	// Path-routed menu entries serve the same shell (client router renders the
	// entry from location.pathname). Exact patterns coexist with the API
	// routes (/tracker vs /tracker/config, etc.). /swagger-ui is a separate
	// route (handleSwaggerUI → shell) so Swagger mounts inside #content with
	// the sidebar visible.
	for _, page := range []string{"/welcome", "/tracker", "/docs", "/memory", "/code", "/sessions"} {
		s.mux.HandleFunc("GET "+page, s.handleShellPage)
		s.mux.HandleFunc("GET "+page+"/", s.handleShellPage)
	}
	root, err := fs.Sub(uiFS, "ui")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("GET /", http.FileServerFS(root))
}

// handleShellPage serves the dashboard shell for path-routed menu entries.
func (s *Server) handleShellPage(w http.ResponseWriter, r *http.Request) {
	page, err := uiFS.ReadFile("ui/index.html")
	if err != nil {
		http.Error(w, "dashboard shell missing from binary", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	spec, err := uiFS.ReadFile("ui/openapi.yaml")
	if err != nil {
		http.Error(w, "openapi spec missing from binary", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(spec)
}

// handleSwaggerUI serves the dashboard shell at /swagger-ui so the sidebar and
// topbar stay visible; the client router mounts Swagger UI inside #content on
// the "swagger" route (loadSwagger in app.js injects the bundle scripts). The
// bundle assets still resolve via the /swagger-ui/{file} routes below.
func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	s.handleShellPage(w, r)
}

func (s *Server) handleSwaggerAsset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("file")
	if name == "" || strings.Contains(name, "/") || strings.ContainsRune(name, 0) {
		http.NotFound(w, r)
		return
	}
	data, err := uiFS.ReadFile("ui/swagger/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch {
	case strings.HasSuffix(name, ".js"):
		if name == "swagger-initializer.js" {
			data = []byte(strings.ReplaceAll(string(data),
				`"https://petstore.swagger.io/v2/swagger.json"`, `"/openapi.yaml"`))
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(name, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(name, ".png"):
		w.Header().Set("Content-Type", "image/png")
	}
	w.Write(data)
}
