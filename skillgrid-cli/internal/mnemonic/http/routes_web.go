package http

import (
	"net/http"
	"strconv"

	"skillgrid-cli/internal/mnemonic/webcache"
)

func (s *Server) handleWebLookup(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r, "")
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

	writeJSON(w, http.StatusOK, lookupDTO(result))
}

func (s *Server) handleWebCacheSave(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source     string         `json:"source"`
		Content    string         `json:"content"`
		URL        string         `json:"url"`
		Title      string         `json:"title"`
		Query      string         `json:"query"`
		LibraryID  string         `json:"library_id"`
		VersionTag string         `json:"version_tag"`
		RepoName   string         `json:"repo_name"`
		Question   string         `json:"question"`
		SortParams string         `json:"sort_params"`
		SessionID  string         `json:"session_id"`
		Metadata   map[string]any `json:"metadata"`
		Project    string         `json:"project"`
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

	id, err := s.svc.WebSave(r.Context(), projectID, webcache.SaveWebInput{
		Source:     body.Source,
		Content:    body.Content,
		URL:        body.URL,
		Title:      body.Title,
		Query:      body.Query,
		LibraryID:  body.LibraryID,
		VersionTag: body.VersionTag,
		RepoName:   body.RepoName,
		Question:   body.Question,
		SortParams: body.SortParams,
		SessionID:  body.SessionID,
		Metadata:   body.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func (s *Server) handleWebSearch(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()
	freshOnly := true
	if v := q.Get("fresh_only"); v != "" {
		freshOnly, _ = strconv.ParseBool(v)
	}

	limit := queryInt(r, "limit", 20)
	hits, err := s.svc.WebSearch(r.Context(), projectID, query, q.Get("source"), freshOnly, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"entries": webHitDTOs(hits)})
}

func (s *Server) handleWebStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	st, err := s.svc.WebCacheStatus(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_entries":   st.TotalEntries,
		"expired_entries": st.ExpiredEntries,
		"by_source":       st.BySource,
		"oldest_fetch":    st.OldestFetch,
		"newest_fetch":    st.NewestFetch,
	})
}

func lookupDTO(r webcache.LookupResult) map[string]any {
	m := map[string]any{
		"status": r.Status,
		"fresh":  r.Fresh,
	}
	if r.ID > 0 {
		m["id"] = r.ID
	}
	if r.FetchedAt != "" {
		m["fetched_at"] = r.FetchedAt
	}
	if r.ExpiresAt != "" {
		m["expires_at"] = r.ExpiresAt
	}
	return m
}

func webHitDTOs(hits []webcache.WebHit) []map[string]any {
	out := make([]map[string]any, len(hits))
	for i, hit := range hits {
		out[i] = map[string]any{
			"id":         hit.ID,
			"source":     hit.Source,
			"title":      hit.Title,
			"query":      hit.Query,
			"url":        hit.URL,
			"library_id": hit.LibraryID,
			"fetched_at": hit.FetchedAt,
			"expires_at": hit.ExpiresAt,
		}
	}
	return out
}
