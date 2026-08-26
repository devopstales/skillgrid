package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"skillgrid-cli/internal/mnemonic/service"
)

const serverVersion = "0.1.0"

// Server wraps the HTTP handler and optional bearer token for write routes.
type Server struct {
	svc   *service.Service
	token string
	mux   *http.ServeMux
}

// NewServer builds the v1 HTTP API handler tree.
func NewServer(svc *service.Service) *Server {
	token := os.Getenv("SKILLGRID_HTTP_TOKEN")
	s := &Server{svc: svc, token: token, mux: http.NewServeMux()}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)

	s.mux.HandleFunc("POST /sessions", s.requireWriteAuth(s.handleSessionCreate))
	s.mux.HandleFunc("POST /sessions/{id}/end", s.requireWriteAuth(s.handleSessionEnd))

	s.mux.HandleFunc("GET /context", s.handleContext)
	s.mux.HandleFunc("POST /observations", s.requireWriteAuth(s.handleObservationCreate))
	s.mux.HandleFunc("GET /search", s.handleSearch)

	s.mux.HandleFunc("GET /code/status", s.handleCodeStatus)
	s.mux.HandleFunc("GET /code/search", s.handleCodeSearch)

	s.mux.HandleFunc("GET /web/lookup", s.handleWebLookup)
	s.mux.HandleFunc("POST /web/cache", s.requireWriteAuth(s.handleWebCacheSave))
	s.mux.HandleFunc("GET /web/search", s.handleWebSearch)
	s.mux.HandleFunc("GET /web/status", s.handleWebStatus)
}

// Handler returns the root http.Handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// StartHTTP listens on addr until the server stops.
func StartHTTP(addr string, svc *service.Service) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: NewServer(svc).Handler(),
	}
	return srv.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "skillgrid-mnemonic",
		"version": serverVersion,
	})
}

func (s *Server) requireWriteAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + s.token
			if auth != expected {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
		}
		next(w, r)
	}
}

func projectFromRequest(r *http.Request, bodyProject string) (string, error) {
	if p := strings.TrimSpace(r.URL.Query().Get("project")); p != "" {
		return p, nil
	}
	if p := strings.TrimSpace(bodyProject); p != "" {
		return p, nil
	}
	return "", fmt.Errorf("project is required")
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}
