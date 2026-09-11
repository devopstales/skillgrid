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
	// routes (/tracker vs /tracker/config, etc.).
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

func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	page, err := uiFS.ReadFile("ui/swagger/index.html")
	if err != nil {
		http.Error(w, "swagger ui missing from binary", http.StatusInternalServerError)
		return
	}
	html := string(page)
	// The dist bundle references favicons that don't ship with us.
	html = strings.ReplaceAll(html, `href="./favicon-32x32.png"`, `href="/favicon.png"`)
	html = strings.ReplaceAll(html, `href="./favicon-16x16.png"`, `href="/favicon.png"`)
	// The dist bundle uses ./-relative asset URLs, but the page is served at
	// /swagger-ui (no trailing slash), so ./x resolves to /x at the root and
	// 404s. Rewrite to the /swagger-ui/{file} asset routes below.
	html = strings.ReplaceAll(html, `"./`, `"/swagger-ui/`)
	html = strings.ReplaceAll(html, `"index.css"`, `"/swagger-ui/index.css"`)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
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
