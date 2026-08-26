package http

import (
	"net/http"

	"skillgrid-cli/internal/mnemonic/search"
)

func (s *Server) handleCodeStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r, "")
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

	limit := queryInt(r, "limit", 20)
	hits, err := s.svc.CodeSearch(r.Context(), projectID, query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"hits": codeHitDTOs(hits)})
}

func codeHitDTOs(hits []search.CodeHit) []map[string]any {
	out := make([]map[string]any, len(hits))
	for i, hit := range hits {
		out[i] = map[string]any{
			"path":       hit.Path,
			"start_line": hit.StartLine,
			"end_line":   hit.EndLine,
			"snippet":    hit.Snippet,
			"score":      hit.Score,
		}
	}
	return out
}
