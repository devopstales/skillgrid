package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// registerFactsRoutes mounts the Fact Memory routes. The five routes wrap the
// existing facts.Store (Add / SearchWith / Forget / Decay / DecayAll) and are
// write-gated like the other mutating routes. The project resolves the handle;
// the session id (optional) is recorded on the store's session_events trail,
// defaulting to "project" so a fact always lands on a known session (Add
// carries a foreign key to sessions.session_id).
func (s *Server) registerFactsRoutes() {
	s.mux.HandleFunc("POST /facts", s.requireWriteAuth(s.handleFactAdd))
	s.mux.HandleFunc("POST /facts/search", s.requireWriteAuth(s.handleFactSearch))
	s.mux.HandleFunc("POST /facts/{id}/forget", s.requireWriteAuth(s.handleFactForget))
	s.mux.HandleFunc("POST /facts/{id}/decay", s.requireWriteAuth(s.handleFactDecay))
	s.mux.HandleFunc("POST /facts/decay-all", s.requireWriteAuth(s.handleFactDecayAll))
}

// factsSessionID resolves the optional session id from the query, defaulting
// to "project" (the session that must exist for the Add trail row's foreign
// key to hold).
func factsSessionID(r *http.Request) string {
	if sid := strings.TrimSpace(r.URL.Query().Get("session_id")); sid != "" {
		return sid
	}
	return "project"
}

// handleFactAdd — POST /facts {content, session_id?} → 200 {id}.
// Empty (or blank) content is a 400; a missing store is a 500.
func (s *Server) handleFactAdd(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		var body struct {
			Content   string `json:"content"`
			SessionID string `json:"session_id"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
			return
		}
		if strings.TrimSpace(body.Content) == "" {
			writeError(w, http.StatusBadRequest, "content is required")
			return
		}
		sid := body.SessionID
		if sid == "" {
			sid = factsSessionID(r)
		}
		store := facts.New(h.Store().DB, projectID)
		id, err := store.Add(r.Context(), sid, body.Content)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "add fact: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
	})
}

// handleFactSearch — POST /facts/search {query, limit?} → 200 {facts}.
// An empty or unmatched query returns 200 with an empty facts array.
func (s *Server) handleFactSearch(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		var body struct {
			Query          string `json:"query"`
			Limit          int    `json:"limit"`
			IncludeDeleted bool   `json:"include_deleted"`
		}
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
			return
		}
		store := facts.New(h.Store().DB, projectID)
		fac, err := store.SearchWith(r.Context(), factsSessionID(r), body.Query, body.Limit, body.IncludeDeleted)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "search facts: "+err.Error())
			return
		}
		if fac == nil {
			fac = []facts.Fact{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"facts": fac})
	})
}

// handleFactForget — POST /facts/{id}/forget → 204. An unknown id is a 404.
func (s *Server) handleFactForget(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		factID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid fact id: "+err.Error())
			return
		}
		store := facts.New(h.Store().DB, projectID)
		if err := store.Forget(r.Context(), factsSessionID(r), factID); err != nil {
			if isUnknownFactError(err, factID) {
				writeError(w, http.StatusNotFound, "fact not found: "+strconv.FormatInt(factID, 10))
				return
			}
			writeError(w, http.StatusInternalServerError, "forget fact: "+err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// handleFactDecay — POST /facts/{id}/decay → 200 {score}. An unknown id is a 404.
func (s *Server) handleFactDecay(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		factID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid fact id: "+err.Error())
			return
		}
		store := facts.New(h.Store().DB, projectID)
		score, err := store.Decay(r.Context(), factsSessionID(r), factID)
		if err != nil {
			if isUnknownFactError(err, factID) {
				writeError(w, http.StatusNotFound, "fact not found: "+strconv.FormatInt(factID, 10))
				return
			}
			writeError(w, http.StatusInternalServerError, "decay fact: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"score": score})
	})
}

// handleFactDecayAll — POST /facts/decay-all {threshold?} → 200 {decayed, purged}.
// The threshold defaults to facts.DefaultPurgeThreshold when omitted.
func (s *Server) handleFactDecayAll(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		threshold := facts.DefaultPurgeThreshold
		var body struct {
			Threshold *float64 `json:"threshold"`
		}
		if err := decodeJSON(r, &body); err == nil && body.Threshold != nil {
			threshold = *body.Threshold
		}
		store := facts.New(h.Store().DB, projectID)
		decayed, purged, err := store.DecayAll(r.Context(), factsSessionID(r), threshold)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "decay all: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"decayed": decayed, "purged": purged})
	})
}

// isUnknownFactError reports whether the store error is the "unknown fact id"
// signal for the given id (the store formats it as "…: unknown fact id N").
func isUnknownFactError(err error, id int64) bool {
	return err != nil && strings.Contains(err.Error(), "unknown fact id "+strconv.FormatInt(id, 10))
}
