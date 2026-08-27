// Package http exposes the Mnemonic REST API with optional bearer token auth.
package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/webcache"
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

func (s *Server) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("directory")
	if dir == "" {
		dir = "."
	}
	sessionID, projectID, err := s.svc.SessionStart(r.Context(), dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"session_id": sessionID,
		"project_id": projectID,
	})
}

func (s *Server) handleSessionEnd(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/sessions/")
	id = strings.TrimSuffix(id, "/end")
	projectID := r.URL.Query().Get("project")
	summary := ""
	if r.Body != nil {
		var body struct {
			Summary string `json:"summary"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		summary = body.Summary
	}
	if err := s.svc.SessionEnd(r.Context(), projectID, id, summary); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": id, "status": "ended"})
}

func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := queryInt(r, "limit", 5)
	sessions, err := s.svc.RecentContext(r.Context(), projectID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *Server) handleObservationCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in service.SaveObservationInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := s.svc.SaveObservation(r.Context(), projectID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	query := r.URL.Query().Get("query")
	matchMode := r.URL.Query().Get("match_mode")
	if matchMode == "" {
		matchMode = "any"
	}
	limit := queryInt(r, "limit", 20)
	hits, err := s.svc.SearchObservations(r.Context(), projectID, query, matchMode, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observations": hits})
}

func (s *Server) handleCodeStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status, stale, err := s.svc.CodeStatus(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"file_count":   status.FileCount,
		"chunk_count":  status.ChunkCount,
		"last_indexed": status.LastIndexed,
		"stale":        stale,
	})
}

func (s *Server) handleCodeSearch(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	query := r.URL.Query().Get("query")
	limit := queryInt(r, "limit", 20)
	hits, err := s.svc.CodeSearch(r.Context(), projectID, query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}

func (s *Server) handleWebLookup(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()
	result, err := s.svc.WebLookup(r.Context(), projectID, webcache.LookupInput{
		Source:      q.Get("source"),
		URL:         q.Get("url"),
		Query:       q.Get("query"),
		LibraryID:   q.Get("library_id"),
		VersionTag:  q.Get("version_tag"),
		RepoName:    q.Get("repo_name"),
		Question:    q.Get("question"),
		SortParams:  q.Get("sort_params"),
		Title:       q.Get("title"),
		ContentHash: q.Get("content_hash"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleWebCacheSave(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in webcache.SaveWebInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := s.svc.WebSave(r.Context(), projectID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleWebSearch(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	query := r.URL.Query().Get("query")
	source := r.URL.Query().Get("source")
	freshOnly := r.URL.Query().Get("fresh_only") != "false"
	limit := queryInt(r, "limit", 20)
	hits, err := s.svc.WebSearch(r.Context(), projectID, query, source, freshOnly, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": hits})
}

func (s *Server) handleWebStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	st, err := s.svc.WebCacheStatus(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func projectFromRequest(r *http.Request) (string, error) {
	if p := strings.TrimSpace(r.URL.Query().Get("project")); p != "" {
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
