package http

import (
	"net/http"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

type usageBody struct {
	Model     string `json:"model"`
	Input     int64  `json:"input"`
	Output    int64  `json:"output"`
	Cache     int64  `json:"cache"`
	Total     bool   `json:"total"`
	Agent     string `json:"agent"`
	Directory string `json:"directory"`
	// ReportedCost is the harness's own cost figure, used only when the
	// model has no entry in the price table.
	ReportedCost *float64 `json:"reported_cost"`
}

// handleSessionUsage serves POST /sessions/{id}/usage {model, input, output,
// cache, total?}. Cost = tokens × the price table (config.d/pricing.yaml); an
// unpriced model leaves cost unknown unless the harness reported its own.
func (s *Server) handleSessionUsage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var b usageBody
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if b.Input < 0 || b.Output < 0 || b.Cache < 0 {
		writeError(w, http.StatusBadRequest, "token counts must be non-negative")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	if _, err := h.Memory().EnsureSession(r.Context(), id, projectID, b.Directory, "", b.Agent); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	model := b.Model
	if model == "" {
		if cur, err := h.Memory().GetSessionUsage(r.Context(), id); err == nil {
			model = cur.Model
		}
	}
	cost := config.LoadPricing(h.Root()).Cost(model, b.Input, b.Output, b.Cache)
	if cost == nil && b.ReportedCost != nil && *b.ReportedCost >= 0 {
		cost = b.ReportedCost
	}
	if b.Input == 0 && b.Output == 0 && b.Cache == 0 && !b.Total {
		cost = nil
	}
	u, err := h.Memory().RecordSessionUsage(r.Context(), id, memory.UsageReport{
		Model: b.Model, Input: b.Input, Output: b.Output, Cache: b.Cache, Cost: cost, Total: b.Total,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, u)
}
