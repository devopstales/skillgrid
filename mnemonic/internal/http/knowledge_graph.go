package http

import (
	"net/http"

	"github.com/devopstales/skillgrid/mnemonic/internal/knowledge"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// handleKnowledgeGraph is GET /knowledge/graph?project= — the open read of
// the read-only knowledge concept graph. The graph is derived PER CALL from
// the project's current knowledge bundle (h.Root()) and is never stored:
// there is no write path by construction (no requireWriteAuth, no store).
func (s *Server) handleKnowledgeGraph(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, _ string) {
		g, err := knowledge.DeriveConceptGraph(h.Root())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "derive concept graph: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, g)
	})
}
