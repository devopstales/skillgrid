// Package http exposes the Mnemonic REST API with optional bearer token auth.
package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
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
	s.mux.HandleFunc("GET /search", s.handleSearch)
	s.mux.HandleFunc("POST /prompts", s.requireWriteAuth(s.handlePromptCreate))

	s.mux.HandleFunc("GET /memory/timeline", s.handleMemoryTimeline)
	s.mux.HandleFunc("PATCH /memory/observations/{id}", s.requireWriteAuth(s.handleObservationUpdate))
	s.mux.HandleFunc("DELETE /memory/observations/{id}", s.requireWriteAuth(s.handleObservationDelete))
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
	s.registerUIRoutes()
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
	if err := s.svc.SessionEnd(r.Context(), projectID, id, summary); err != nil {
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
	if err := s.svc.SessionSetTitle(r.Context(), projectID, id, body.Title); err != nil {
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
	sessions, err := s.svc.RecentContext(r.Context(), projectID, limit)
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
	ctxOut, err := s.svc.ContextForCompaction(r.Context(), projectID, sessionID, limit)
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
		started, err := s.svc.SessionStartedAt(r.Context(), projectID, id)
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
	ts, err := s.svc.LastObservationAt(r.Context(), projectID)
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
	id, err := s.svc.SavePrompt(r.Context(), projectID, in)
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
	res, err := s.svc.CapturePassive(r.Context(), projectID, in)
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
	tl, err := s.svc.ObservationTimeline(r.Context(), projectID, id, window, limit)
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
	if err := s.svc.UpdateObservation(r.Context(), projectID, id, in); err != nil {
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
	if err := s.svc.DeleteObservation(r.Context(), projectID, id, hard); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true, "hard": hard})
}

func (s *Server) handleMemoryReviews(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := queryInt(r, "limit", 20)
	due, err := s.svc.ListReviews(r.Context(), projectID, limit)
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
	ta, err := s.svc.MarkReviewReviewed(r.Context(), projectID, id)
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
	out, err := s.svc.RecordRelation(r.Context(), projectID, body.SrcID, body.DstID, rel, body.Reason, body.Confidence)
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
	removed, err := s.svc.RemoveRelation(r.Context(), projectID, srcID, dstID, rel)
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
	rels, err := s.svc.RelationsOf(r.Context(), projectID, id)
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
	rels, err := s.svc.RelationsBetween(r.Context(), projectID, srcID, dstID)
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
	out, err := s.svc.MemoryDoctor(r.Context(), projectID)
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
	st, err := s.svc.MemoryStatus(r.Context(), projectID)
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
	id, err := s.svc.SaveObservation(r.Context(), projectID, in)
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
	sessions, err := s.svc.RecentContext(r.Context(), projectID, limit)
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
	stats, err := s.svc.RunCodeIndex(r.Context(), dir)
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
	result, err := s.svc.ReadIndexedCode(r.Context(), projectID, path, startLine, endLine)
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

func (s *Server) handleCodeFiles(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	paths, err := s.svc.CodeFiles(r.Context(), projectID)
	if err != nil {
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
	obs, err := s.svc.ObservationsRecent(r.Context(), projectID, limit)
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
	entry, err := s.svc.WebGet(r.Context(), projectID, id)
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
