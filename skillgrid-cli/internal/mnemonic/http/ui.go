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
	root, err := fs.Sub(uiFS, "ui")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("GET /", http.FileServerFS(root))
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
