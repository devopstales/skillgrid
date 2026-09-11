// Package http exposes the Mnemonic REST API with optional bearer token auth.
package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/docs"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
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

// docsCwd is the directory the docs bridge sandboxes reads against. Overridable
// in tests; defaults to the server process's working directory (the repo
// root, matching the tracker bridge's doc/config reads).
var docsCwd = func() string {
	if v := strings.TrimSpace(os.Getenv("SKILLGRID_DOCS_CWD")); v != "" {
		return v
	}
	return "."
}()

// openHandleFor opens a single Project Handle for a resolved project id
// (config root "."), mirroring the MCP single-open lifecycle: each request
// opens the project once and works through the handle instead of the
// re-opening facade.
func (s *Server) openHandleFor(projectID string) (*service.ProjectHandle, func(), error) {
	return s.svc.Open(projectID)
}

// openHandleForDir opens a single Project Handle for a directory-rooted route
// (project resolved from the directory).
func (s *Server) openHandleForDir(directory string) (*service.ProjectHandle, func(), error) {
	return s.svc.OpenForDirectory(directory)
}

// scopeForSave reproduces the scope normalization service.SaveObservation
// applied before memory.Save ("" → "project", "personal" → "user").
func scopeForSave(scope string) string {
	if scope == "" {
		return "project"
	}
	if scope == "personal" {
		return "user"
	}
	return scope
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)

	s.mux.HandleFunc("POST /sessions", s.requireWriteAuth(s.handleSessionCreate))
	s.mux.HandleFunc("POST /sessions/{id}/end", s.requireWriteAuth(s.handleSessionEnd))
	s.mux.HandleFunc("POST /sessions/{id}/title", s.requireWriteAuth(s.handleSessionSetTitle))
	s.mux.HandleFunc("GET /sessions/{id}", s.handleSessionGet)

	s.mux.HandleFunc("GET /context", s.handleContext)
	s.mux.HandleFunc("GET /context/compaction", s.handleContextCompaction)
	s.mux.HandleFunc("GET /memory/status", s.handleMemoryStatus)
	s.mux.HandleFunc("GET /memory/last-save-at", s.handleMemoryLastSaveAt)
	s.mux.HandleFunc("POST /observations", s.requireWriteAuth(s.handleObservationCreate))
	s.mux.HandleFunc("POST /observations/passive", s.requireWriteAuth(s.handleObservationPassive))
	s.mux.HandleFunc("GET /observations/recent", s.handleObservationsRecent)
	s.mux.HandleFunc("GET /observations", s.handleObservationsList)
	// P4 Memory entry: open read of a single observation (full content, same
	// shape as mem_get_observation).
	s.mux.HandleFunc("GET /observations/{id}", s.handleObservationGet)
	s.mux.HandleFunc("GET /search", s.handleSearch)
	s.mux.HandleFunc("POST /prompts", s.requireWriteAuth(s.handlePromptCreate))

	s.mux.HandleFunc("GET /memory/timeline", s.handleMemoryTimeline)
	s.mux.HandleFunc("PATCH /memory/observations/{id}", s.requireWriteAuth(s.handleObservationUpdate))
	s.mux.HandleFunc("DELETE /memory/observations/{id}", s.requireWriteAuth(s.handleObservationDelete))
	// P4 Memory entry: pin/unpin are write-gated (idempotent); governance
	// mutations (share/status) + the governed-asset view (governance) are
	// write-gated too, except the governance read which is open.
	s.mux.HandleFunc("POST /memory/observations/{id}/pin", s.requireWriteAuth(s.handleObservationPin))
	s.mux.HandleFunc("POST /memory/observations/{id}/unpin", s.requireWriteAuth(s.handleObservationUnpin))
	s.mux.HandleFunc("POST /memory/observations/{id}/share", s.requireWriteAuth(s.handleObservationShare))
	s.mux.HandleFunc("POST /memory/observations/{id}/status", s.requireWriteAuth(s.handleObservationStatus))
	s.mux.HandleFunc("GET /memory/observations/{id}/governance", s.handleObservationGovernance)
	s.mux.HandleFunc("GET /memory/reviews", s.handleMemoryReviews)
	s.mux.HandleFunc("POST /memory/reviews/{id}", s.requireWriteAuth(s.handleMemoryReviewMark))
	s.mux.HandleFunc("POST /memory/relations", s.requireWriteAuth(s.handleRelationCreate))
	s.mux.HandleFunc("DELETE /memory/relations", s.requireWriteAuth(s.handleRelationRemove))
	s.mux.HandleFunc("GET /relations/{id}", s.handleRelationsOf)
	s.mux.HandleFunc("GET /relations", s.handleRelationsBetween)
	s.mux.HandleFunc("GET /memory/project", s.handleMemoryCurrentProject)
	s.mux.HandleFunc("GET /memory/doctor", s.handleMemoryDoctor)

	s.mux.HandleFunc("POST /projects/migrate", s.requireWriteAuth(s.handleProjectsMigrate))
	s.mux.HandleFunc("POST /projects/merge", s.requireWriteAuth(s.handleProjectsMerge))

	s.mux.HandleFunc("GET /code/status", s.handleCodeStatus)
	s.mux.HandleFunc("POST /code/index", s.requireWriteAuth(s.handleCodeIndex))
	s.mux.HandleFunc("GET /code/files", s.handleCodeFiles)
	s.mux.HandleFunc("GET /code/search", s.handleCodeSearch)
	s.mux.HandleFunc("GET /code/read", s.handleCodeRead)

	s.mux.HandleFunc("GET /web/lookup", s.handleWebLookup)
	s.mux.HandleFunc("POST /web/cache", s.requireWriteAuth(s.handleWebCacheSave))
	s.mux.HandleFunc("GET /web/search", s.handleWebSearch)
	s.mux.HandleFunc("GET /web/entry/{id}", s.handleWebEntry)
	s.mux.HandleFunc("GET /web/status", s.handleWebStatus)

	s.mux.HandleFunc("GET /projects", s.handleProjects)

	s.registerTeamsRoutes()
	s.registerTrackerRoutes()
	s.registerDocsRoutes()
	s.registerUIRoutes()
}

// registerDocsRoutes mounts the read-only SDD docs bridge on the two routes
// it owns (GET /docs/changes, GET /docs/changes/{name}). The GET /docs shell
// page is registered separately by registerUIRoutes (exact pattern, no
// conflict with these literal routes).
func (s *Server) registerDocsRoutes() {
	s.mux.Handle("GET /docs/changes", docs.NewList(docsCwd))
	s.mux.Handle("GET /docs/changes/{name}", docs.NewDetail(docsCwd))
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
	title := r.URL.Query().Get("title")
	id := r.URL.Query().Get("id")
	type body struct {
		ID        string `json:"id,omitempty"`
		Directory string `json:"directory,omitempty"`
		Title     string `json:"title,omitempty"`
	}
	var b body
	if r.Body != nil {
		_ = decodeJSON(r, &b)
	}
	if b.Directory != "" {
		dir = b.Directory
	}
	if b.Title != "" {
		title = b.Title
	}
	if b.ID != "" {
		id = b.ID
	}

	if id != "" {
		// Caller supplied an authoritative ID — register under it, idempotent.
		sessionID, projectID, existed, err := s.svc.SessionStartByClientID(r.Context(), id, dir, title)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := map[string]any{
			"session_id": sessionID,
			"project_id": projectID,
			"created":    !existed,
		}
		if title != "" {
			out["title"] = title
		}
		status := http.StatusCreated
		if existed {
			status = http.StatusOK
		}
		writeJSON(w, status, out)
		return
	}

	sessionID, projectID, err := s.svc.SessionStart(r.Context(), dir, title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]any{
		"session_id": sessionID,
		"project_id": projectID,
	}
	if title != "" {
		out["title"] = title
	}
	writeJSON(w, http.StatusCreated, out)
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
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	if err := h.Memory().SessionEnd(r.Context(), id, summary); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": id, "status": "ended"})
}

func (s *Server) handleSessionSetTitle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	projectID := r.URL.Query().Get("project")
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	if err := h.Memory().SessionSetTitle(r.Context(), id, body.Title); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session_id": id, "title": body.Title})
}

func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := queryInt(r, "limit", 5)
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	sessions, err := h.Memory().RecentContext(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

// handleContextCompaction returns a compact, session-scoped context block for
// the compaction prompt: the current session's title/summary plus the newest
// few observations. The plugin injects this so nothing is lost in summarisation.
func (s *Server) handleContextCompaction(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sessionID := r.URL.Query().Get("session_id")
	limit := queryInt(r, "limit", 5)
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	ctxOut, err := h.Memory().CompactionContext(r.Context(), sessionID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"context": ctxOut})
}

// handleSessionGet returns a single session including started_at, which the
// plugin uses for save-nudge age checks.
func (s *Server) handleSessionGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	// Look up across projects: resolve the project from the query, else list.
	projectID := r.URL.Query().Get("project")
	st := map[string]any{"id": id}
	if projectID != "" {
		h, cleanup, err := s.openHandleFor(projectID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		defer cleanup()
		started, err := h.Memory().SessionStartedAt(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if !started.IsZero() {
			st["started_at"] = started.UTC().Format(time.RFC3339)
		}
	}
	writeJSON(w, http.StatusOK, st)
}

// handleMemoryLastSaveAt returns the newest observation timestamp so the
// plugin can compute how long since the last save (for the debounced nudge).
func (s *Server) handleMemoryLastSaveAt(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	ts, err := h.Memory().LastObservationAt(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := map[string]any{"last_save_at": ""}
	if !ts.IsZero() {
		out["last_save_at"] = ts.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, out)
}

// handlePromptCreate persists a captured user prompt.
func (s *Server) handlePromptCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in service.PromptInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(projectID) == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	id, err := h.Memory().SavePrompt(r.Context(), in)
	if err != nil {
		if errors.Is(err, memory.ErrPromptTooSmall) {
			writeJSON(w, http.StatusAccepted, map[string]any{"captured": false, "reason": "too-small"})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "captured": true})
}

// handleObservationPassive extracts learnings from free text and persists them.
func (s *Server) handleObservationPassive(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var in service.PassiveInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(projectID) == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	res, err := h.Memory().CapturePassive(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleProjectsMigrate rolls data from oldProject into newProject.
func (s *Server) handleProjectsMigrate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OldProject string `json:"old_project"`
		NewProject string `json:"new_project"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	moved, err := s.svc.MigrateProjects(r.Context(), body.OldProject, body.NewProject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"old_project": body.OldProject,
		"new_project": body.NewProject,
		"rows_moved":  moved,
	})
}

// handleProjectsMerge is the mem_merge_projects surface: same copy as
// MigrateProjects, plus an alias record so future writes to the legacy name
// land in the canonical store.
func (s *Server) handleProjectsMerge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source    string `json:"source"`
		Canonical string `json:"canonical"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(body.Source) == "" || strings.TrimSpace(body.Canonical) == "" {
		writeError(w, http.StatusBadRequest, "source and canonical are required")
		return
	}
	if body.Source == body.Canonical {
		writeJSON(w, http.StatusOK, map[string]any{"merged": false, "reason": "source and canonical identical"})
		return
	}
	moved, canonical, err := s.svc.MergeProjects(r.Context(), body.Source, body.Canonical)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"merged":         true,
		"source":         body.Source,
		"canonical":      canonical,
		"rows_moved":     moved,
		"alias_recorded": true,
	})
}

// ───────────────────────────────── Memory extension routes ───────────────────

func (s *Server) handleMemoryTimeline(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "id query param is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	window := parseWindow(r.URL.Query().Get("window"), 1*time.Hour)
	limit := queryInt(r, "limit", 5)
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	tl, err := h.Memory().Timeline(r.Context(), id, window, limit)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"anchor_id": id, "before": tl.Before, "after": tl.After})
}

func (s *Server) handleObservationUpdate(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	var in memory.UpdateInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	if err := h.Memory().Update(r.Context(), id, in); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "updated": true})
}

func (s *Server) handleObservationDelete(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	hard := r.URL.Query().Get("hard") == "true"
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	if err := h.Memory().Delete(r.Context(), id, hard); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true, "hard": hard})
}

// obsIDFromPath parses the {id} path segment; 400 on a non-integer id.
func obsIDFromPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return 0, false
	}
	return id, true
}

// handleObservationGet is the open read of a single observation (P4 Memory
// detail pane). Returns the full, untruncated observation in the same shape as
// mem_get_observation; 404 for an unknown id.
func (s *Server) handleObservationGet(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, ok := obsIDFromPath(w, r)
	if !ok {
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	obs, err := h.Memory().Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, obs)
}

// handleObservationPin / handleObservationUnpin toggle the local "sticky"
// marker. Both are idempotent (pin an already-pinned / unpin a non-pinned
// observation returns 200, service-level behavior preserved).
func (s *Server) handleObservationPin(w http.ResponseWriter, r *http.Request) {
	s.handleObservationPinUnpin(w, r, true)
}

func (s *Server) handleObservationUnpin(w http.ResponseWriter, r *http.Request) {
	s.handleObservationPinUnpin(w, r, false)
}

func (s *Server) handleObservationPinUnpin(w http.ResponseWriter, r *http.Request, pin bool) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, ok := obsIDFromPath(w, r)
	if !ok {
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	var (
		e error
		m *memory.Service = h.Memory()
	)
	if pin {
		e = m.Pin(r.Context(), id)
	} else {
		e = m.Unpin(r.Context(), id)
	}
	if e != nil {
		writeError(w, http.StatusNotFound, e.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "pinned": pin})
}

// shareBody is the body for POST /memory/observations/{id}/share.
type shareBody struct {
	Visibility string   `json:"visibility"`
	Grants     []string `json:"grants,omitempty"`
}

// handleObservationShare widens an observation's visibility (013 governance).
// Idempotent; an unknown visibility target → 400, leaving visibility unchanged.
func (s *Server) handleObservationShare(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, ok := obsIDFromPath(w, r)
	if !ok {
		return
	}
	var in shareBody
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	if err := h.Memory().Share(r.Context(), id, memory.ShareInput{Visibility: in.Visibility, Grants: in.Grants}); err != nil {
		if strings.Contains(err.Error(), "invalid share target") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "visibility": in.Visibility})
}

// statusBody is the body for POST /memory/observations/{id}/status.
type statusBody struct {
	Status string `json:"status"`
}

// handleObservationStatus sets an observation's lifecycle status explicitly
// (active|superseded|archived). Unknown status → 400.
func (s *Server) handleObservationStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, ok := obsIDFromPath(w, r)
	if !ok {
		return
	}
	var in statusBody
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	if err := h.Memory().SetStatus(r.Context(), id, in.Status); err != nil {
		if strings.Contains(err.Error(), "invalid status") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": in.Status})
}

// handleObservationGovernance is the open read of the governed-asset view
// (owner, append-only version history, status, retrieval usage, visibility,
// ACL grants). This is the data behind the P4 governance widgets.
func (s *Server) handleObservationGovernance(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, ok := obsIDFromPath(w, r)
	if !ok {
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	g, err := h.Memory().Governance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleMemoryReviews(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := queryInt(r, "limit", 20)
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	due, err := h.Memory().ListReviews(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"due": due, "count": len(due)})
}

func (s *Server) handleMemoryReviewMark(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	ta, err := h.Memory().MarkReviewed(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "marked_reviewed": true, "review_after": ta})
}

// RelationBody is the shared body for mem_judge (verdict) / mem_compare
// (relation). Accepts either key, since the two MCP tools name the same
// field differently.
type relationBody struct {
	SrcID      int64    `json:"src_id"`
	DstID      int64    `json:"dst_id"`
	Verdict    string   `json:"verdict"`
	Relation   string   `json:"relation"`
	Reason     string   `json:"reason"`
	Confidence *float64 `json:"confidence"`
}

func (b *relationBody) effectiveRelation() string {
	if r := strings.TrimSpace(b.Verdict); r != "" {
		return r
	}
	return strings.TrimSpace(b.Relation)
}

func (s *Server) handleRelationCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var body relationBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rel := body.effectiveRelation()
	if rel == "" {
		writeError(w, http.StatusBadRequest, "verdict or relation is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	defer cleanup()
	out, err := h.Memory().RecordRelation(r.Context(), body.SrcID, body.DstID, rel, body.Reason, body.Confidence)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) handleRelationRemove(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()
	srcID, err := strconv.ParseInt(q.Get("src_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "src_id is required")
		return
	}
	dstID, err := strconv.ParseInt(q.Get("dst_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "dst_id is required")
		return
	}
	rel := q.Get("relation")
	if rel == "" {
		writeError(w, http.StatusBadRequest, "relation is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	removed, err := h.Memory().RemoveRelation(r.Context(), srcID, dstID, rel)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"removed": removed, "src_id": srcID, "dst_id": dstID})
}

func (s *Server) handleRelationsOf(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	rels, err := h.Memory().RelationsOf(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "relations": rels, "count": len(rels)})
}

// handleRelationsBetween lists live links between two observations (the
// mem_compare list mode over HTTP).
func (s *Server) handleRelationsBetween(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()
	srcID, err := strconv.ParseInt(q.Get("src_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "src_id is required")
		return
	}
	dstID, err := strconv.ParseInt(q.Get("dst_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "dst_id is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	rels, err := h.Memory().RelationsBetween(r.Context(), srcID, dstID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"src_id": srcID, "dst_id": dstID, "relations": rels, "count": len(rels)})
}

func (s *Server) handleMemoryCurrentProject(w http.ResponseWriter, r *http.Request) {
	cwd, err := os.Getwd()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	info, err := s.svc.CurrentProject(cwd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleMemoryDoctor(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	out, err := httpMemoryDoctor(r.Context(), h, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func parseWindow(v string, def time.Duration) time.Duration {
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func (s *Server) handleMemoryStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	st, err := h.Memory().Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
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
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	id, err := h.Memory().Save(r.Context(), memory.SaveInput{
		Title:         in.Title,
		Type:          in.Type,
		Content:       in.Content,
		Scope:         scopeForSave(in.Scope),
		TopicKey:      in.TopicKey,
		SessionID:     in.SessionID,
		CapturePrompt: in.CapturePrompt,
		ProjectName:   in.ProjectName,
		ToolName:      in.ToolName,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleObservationsRecent(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := queryInt(r, "limit", 5)
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	sessions, err := h.Memory().RecentContext(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *Server) handleCodeIndex(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	h, cleanup, err := s.openHandleForDir(dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	// Same indexing behavior as service.RunCodeIndex: the config for the index
	// run comes from the directory itself (matches the facade), and the embedder
	// is derived from that same single loaded config (no second load).
	cfg := config.Load(dir)
	idxCfg := codeindex.Config{
		Include:      cfg.Include,
		Exclude:      cfg.Exclude,
		ChunkLines:   cfg.ChunkLines,
		ChunkOverlap: cfg.ChunkOverlap,
		MaxFileSize:  cfg.MaxFileSize,
	}
	idx := codeindex.New(h.Store())
	if emb := httpResolveEmbedder(cfg); emb != nil {
		idx = idx.WithEmbedder(emb)
	}
	stats, err := idx.Run(r.Context(), dir, idxCfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"files_indexed": stats.FilesIndexed,
		"files_skipped": stats.FilesSkipped,
		"files_deleted": stats.FilesDeleted,
		"chunks_added":  stats.ChunksAdded,
	})
}

func (s *Server) handleCodeRead(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	q := r.URL.Query()
	path := q.Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	startLine, endLine := 0, 0
	if v := q.Get("start_line"); v != "" {
		startLine, _ = strconv.Atoi(v)
	}
	if v := q.Get("end_line"); v != "" {
		endLine, _ = strconv.Atoi(v)
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	result, err := readIndexedCode(h.Store().DB, path, startLine, endLine)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
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
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	// SearchObservations delegates to the scoped variant with scope "" (any).
	hits, err := h.Memory().SearchWithScope(r.Context(), query, matchMode, "", limit)
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
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	status, err := codeindex.GetStatus(h.Store())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	stale := status.FileCount == 0 || status.LastIndexed == ""
	writeJSON(w, http.StatusOK, map[string]any{
		"file_count":   status.FileCount,
		"chunk_count":  status.ChunkCount,
		"last_indexed": status.LastIndexed,
		"stale":        stale,
	})
}

func (s *Server) handleCodeFiles(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	rows, err := h.Store().DB.QueryContext(r.Context(), `SELECT path FROM files ORDER BY path`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list files: "+err.Error())
		return
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": paths})
}

func (s *Server) handleCodeSearch(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	query := r.URL.Query().Get("query")
	limit := queryInt(r, "limit", 20)
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	hits, err := search.CodeSearch(h.Store().DB, query, limit)
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
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	result, err := h.Web().Lookup(r.Context(), webcache.LookupInput{
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
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	id, err := h.Web().Save(r.Context(), in)
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
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	hits, err := h.Web().Search(r.Context(), query, source, freshOnly, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": hits})
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	ids, err := s.svc.ListProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": ids})
}

func (s *Server) handleObservationsList(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := queryInt(r, "limit", 50)
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	obs, err := h.Memory().Recent(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"observations": obs})
}

func (s *Server) handleWebEntry(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	entry, err := h.Web().Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) handleWebStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	st, err := h.Web().CacheStatus(r.Context())
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
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err == nil && len(body) > 0 {
			_ = r.Body.Close()
			var v any
			if err := json.Unmarshal(body, &v); err == nil {
				if m, ok := v.(map[string]any); ok {
					if p, ok := m["project"].(string); ok && strings.TrimSpace(p) != "" {
						r.Body = io.NopCloser(bytes.NewReader(body))
						return strings.TrimSpace(p), nil
					}
				}
			}
		} else {
			_ = r.Body.Close()
		}
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

// httpMemoryDoctor reproduces service.MemoryDoctor's read-only diagnostics
// (schema version, WAL state, table + FTS row counts and drift, by-type,
// disk size) directly on the handle's store, so the doctor route opens the
// project store exactly once.
func httpMemoryDoctor(ctx context.Context, h *service.ProjectHandle, projectID string) (service.MemoryDoctor, error) {
	out := service.MemoryDoctor{}
	db := h.Store().DB
	if err := db.QueryRowContext(ctx, `SELECT schema_version FROM index_meta WHERE key='schema_version'`).Scan(&out.SchemaVersion); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return out, fmt.Errorf("schema_version: %w", err)
		}
	}
	if err := db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&out.WALMode); err != nil {
		return out, fmt.Errorf("journal_mode: %w", err)
	}
	byType := map[string]int{}
	httpRowCount(ctx, db, "observations", &out.Observations)
	httpRowCount(ctx, db, "files", &out.Files)
	httpRowCount(ctx, db, "chunks", &out.Chunks)
	httpRowCount(ctx, db, "web_cache", &out.WebCache)
	httpRowCount(ctx, db, "prompts", &out.Prompts)

	ftsObs, ftsChunks, ftsWeb := int64(0), int64(0), int64(0)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM observations_fts`).Scan(&ftsObs)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks_fts`).Scan(&ftsChunks)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM web_cache_fts`).Scan(&ftsWeb)
	out.ObservationsFTS = int(ftsObs)
	out.ChunksFTS = int(ftsChunks)
	out.WebCacheFTS = int(ftsWeb)

	rows, err := db.QueryContext(ctx, `
		SELECT type, COUNT(*) FROM observations
		WHERE project = ? AND deleted_at IS NULL GROUP BY type`, projectID)
	if err == nil {
		for rows.Next() {
			var t string
			var c int
			if rows.Scan(&t, &c) == nil {
				byType[t] = c
			}
		}
		rows.Close()
	}
	out.ByType = byType

	if info, err := os.Stat(h.Store().Path()); err == nil {
		out.DiskSizeBytes = info.Size()
	}
	out.FTSDrift = int(ftsObs) - out.Observations
	out.FTSIntegrityOK = out.FTSDrift >= 0
	return out, nil
}

func httpRowCount(ctx context.Context, db *sql.DB, table string, out *int) {
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(out)
}

// httpResolveEmbedder builds the process embedder from an already-loaded
// config, mirroring service.resolveEmbedder (external/onnx/off). A nil return
// means the embedder is off — indexing degrades to the FTS floor.
func httpResolveEmbedder(cfg config.Indexing) embedder.Embedder {
	switch cfg.Embedder.Provider {
	case "external":
		return embedder.NewExternal(embedder.ExternalConfig{
			BaseURL:   cfg.Embedder.BaseURL,
			Model:     cfg.Embedder.Model,
			APIKey:    cfg.Embedder.APIKey,
			Dimension: cfg.Embedder.Dimension,
			Indexing:  httpToAsym(cfg.Embedder.Indexing),
			Query:     httpToAsym(cfg.Embedder.Query),
		})
	case "off", "":
		return nil
	default: // "onnx" is the default
		return embedder.NewOnnx(embedder.OnnxConfig{
			Model:     cfg.Embedder.Model,
			Dimension: cfg.Embedder.Dimension,
			Indexing:  httpToAsym(cfg.Embedder.Indexing),
			Query:     httpToAsym(cfg.Embedder.Query),
		})
	}
}

func httpToAsym(p config.EmbedderParams) embedder.AsymParams {
	return embedder.AsymParams{
		Instructions: p.Instructions,
		InputType:    p.InputType,
		MaxTokens:    p.MaxTokens,
	}
}

// readIndexedCode returns indexed source for path and optional line range
// (byte-compatible copy of the service-layer helper, run on the handle's DB).
func readIndexedCode(db *sql.DB, path string, startLine, endLine int) (map[string]any, error) {
	var fileID int64
	err := db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&fileID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not indexed: %s", path)
	}
	if err != nil {
		return nil, err
	}
	var rows *sql.Rows
	if startLine > 0 {
		if endLine <= 0 {
			endLine = startLine
		}
		rows, err = db.Query(`
			SELECT start_line, end_line, text FROM chunks
			WHERE file_id = ? AND start_line <= ? AND end_line >= ?
			ORDER BY start_line`,
			fileID, endLine, startLine,
		)
	} else {
		rows, err = db.Query(`
			SELECT start_line, end_line, text FROM chunks
			WHERE file_id = ?
			ORDER BY start_line`, fileID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var parts []string
	firstLine := 0
	lastLine := 0
	for rows.Next() {
		var chunkStart, chunkEnd int
		var text string
		if err := rows.Scan(&chunkStart, &chunkEnd, &text); err != nil {
			return nil, err
		}
		if firstLine == 0 || chunkStart < firstLine {
			firstLine = chunkStart
		}
		if chunkEnd > lastLine {
			lastLine = chunkEnd
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no indexed chunks for %s", path)
	}
	return map[string]any{
		"path":       path,
		"start_line": firstLine,
		"end_line":   lastLine,
		"text":       strings.Join(parts, "\n"),
	}, nil
}
