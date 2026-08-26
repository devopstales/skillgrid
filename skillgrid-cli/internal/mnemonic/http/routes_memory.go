package http

import (
	"net/http"

	"skillgrid-cli/internal/mnemonic/memory"
	"skillgrid-cli/internal/mnemonic/service"
)

func (s *Server) handleSessionCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Directory string `json:"directory"`
		Project   string `json:"project"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Directory == "" {
		writeError(w, http.StatusBadRequest, "directory is required")
		return
	}

	sessionID, projectID, err := s.svc.SessionStart(r.Context(), body.Directory)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"session_id": sessionID,
		"project":    projectID,
	})
}

func (s *Server) handleSessionEnd(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session id is required")
		return
	}

	var body struct {
		Summary string `json:"summary"`
		Project string `json:"project"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	projectID, err := projectFromRequest(r, body.Project)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.svc.SessionEnd(r.Context(), projectID, sessionID, body.Summary); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": sessionID,
		"status":     "ended",
	})
}

func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r, "")
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

	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessionDTOs(sessions)})
}

func (s *Server) handleObservationCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SessionID string `json:"session_id"`
		Type      string `json:"type"`
		Title     string `json:"title"`
		Content   string `json:"content"`
		Project   string `json:"project"`
		Scope     string `json:"scope"`
		TopicKey  string `json:"topic_key"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	projectID, err := projectFromRequest(r, body.Project)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := s.svc.SaveObservation(r.Context(), projectID, service.SaveObservationInput{
		Title:     body.Title,
		Type:      body.Type,
		Content:   body.Content,
		Scope:     body.Scope,
		TopicKey:  body.TopicKey,
		SessionID: body.SessionID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "q is required")
		return
	}

	projectID, err := projectFromRequest(r, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

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

	writeJSON(w, http.StatusOK, map[string]any{"observations": observationDTOs(hits)})
}

func sessionDTOs(sessions []memory.Session) []map[string]any {
	out := make([]map[string]any, len(sessions))
	for i, sess := range sessions {
		out[i] = map[string]any{
			"id":         sess.ID,
			"project":    sess.Project,
			"directory":  sess.Directory,
			"started_at": sess.StartedAt,
			"ended_at":   sess.EndedAt,
			"summary":    sess.Summary,
			"status":     sess.Status,
		}
	}
	return out
}

func observationDTOs(obs []memory.Observation) []map[string]any {
	out := make([]map[string]any, len(obs))
	for i, o := range obs {
		m := map[string]any{
			"id":              o.ID,
			"session_id":      o.SessionID,
			"type":            o.Type,
			"title":           o.Title,
			"content":         o.Content,
			"project":         o.Project,
			"scope":           o.Scope,
			"normalized_hash": o.NormalizedHash,
			"revision_count":  o.RevisionCount,
			"created_at":      o.CreatedAt,
			"updated_at":      o.UpdatedAt,
		}
		if o.TopicKey != "" {
			m["topic_key"] = o.TopicKey
		}
		out[i] = m
	}
	return out
}
