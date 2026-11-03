package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness"
	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness/clm"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

type clmRevisionRequest struct {
	SessionID string   `json:"session_id"`
	Document  string   `json:"document"`
	Messages  []string `json:"messages"`
}

// handleCLMRevision serves POST /sessions/{id}/clm/revision. It is the audit
// and revision-persistence seam for the Context Harness (CLM, ADR-0027): the
// Node checkpoint hook POSTs the parsed mirror edit and the Go side validates
// and stores it. Every path is fail-open — a malformed body, CLM off (the
// default), a validation miss, or a persist error all respond 200 so a CLM
// failure never blocks the agent.
func (s *Server) handleCLMRevision(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		var req clmRevisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		if req.SessionID == "" {
			req.SessionID = sessionID
		}
		cfg := h.Memory().CLMConfig()
		// Fail-open: CLM off (the default) or any error -> 200 ok, nothing persisted.
		if !cfg.Enabled {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		rev, err := buildCLMRevision(r.Context(), h, req, projectID, cfg)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		if err := clmPersist(r.Context(), h, rev); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "revision": rev.Revision})
	})
}

// buildCLMRevision validates the edit and fills a clm.Revision ready to persist.
// It is the thin capture-path counterpart of the Node request path: the
// authoritative overflow math lives in the clm package; here we just record the
// edit, a token estimate, and the base calibration factor (this path has no
// provider token count yet). The revision number is the session's highest
// existing revision plus one.
func buildCLMRevision(ctx context.Context, h *service.ProjectHandle, req clmRevisionRequest, projectID string, cfg config.CLM) (*clm.Revision, error) {
	edit := clm.Edit{Nonce: req.Document, Messages: req.Messages}
	if err := clm.ValidateRevision(&edit, req.Document); err != nil {
		return nil, err
	}
	messages := strings.Join(req.Messages, "\n")
	var next int
	if err := h.Store().DB.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(revision),0)+1 FROM context_revisions WHERE session_id = ?`,
		req.SessionID).Scan(&next); err != nil {
		return nil, err
	}
	return &clm.Revision{
		SessionID:         req.SessionID,
		ProjectID:         projectID,
		Revision:          next,
		AnchorCount:       len(req.Messages),
		AnchorDigest:      req.Document,
		Messages:          messages,
		SizeEstimate:      context_harness.EstimateTokens(messages),
		CalibrationFactor: clm.BaseFactor(),
	}, nil
}

func clmPersist(ctx context.Context, h *service.ProjectHandle, rev *clm.Revision) error {
	return clm.PersistRevision(ctx, h.Store().DB, rev)
}
