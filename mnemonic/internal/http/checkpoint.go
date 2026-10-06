package http

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/checkpoint"
	"github.com/devopstales/skillgrid/mnemonic/internal/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

type checkpointClaimResponse struct {
	Due    bool   `json:"due"`
	Reason string `json:"reason"`
	Prompt string `json:"prompt"`
}

// handleCheckpointClaim serves POST /sessions/{id}/checkpoint/claim.
func (s *Server) handleCheckpointClaim(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		ctx := r.Context()
		st, err := h.Memory().CheckpointState(ctx, sessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		cfg := config.Load(h.Root()).Checkpoint
		now := s.now()
		due, reason := checkpoint.Decide(st, cfg, now)
		if !due {
			writeJSON(w, http.StatusOK, checkpointClaimResponse{Due: false, Reason: reason})
			return
		}

		events, err := queryCheckpointEvents(ctx, h.Store().DB, projectID, sessionID, st.LastWriteAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		obs, err := querySessionObservations(ctx, h.Store().DB, projectID, sessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		const maxExistingTitles = 20
		titles := make([]string, 0, len(obs))
		for _, o := range obs {
			if t := o.Title; t != "" {
				titles = append(titles, t)
			}
			if len(titles) >= maxExistingTitles {
				break
			}
		}
		digest := checkpoint.BuildDigest(events, 1500)
		prompt := checkpoint.RenderPrompt(checkpoint.PromptInput{
			SessionID:       sessionID,
			Project:         projectID,
			Digest:          digest,
			ExistingTitles:  titles,
			MaxObservations: cfg.MaxObservations,
		})
		if err := h.Memory().ClaimCheckpoint(ctx, sessionID, now); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, checkpointClaimResponse{Due: true, Prompt: prompt})
	})
}

func queryCheckpointEvents(ctx context.Context, db *sql.DB, projectID, sessionID string, afterWrite time.Time) ([]checkpoint.Event, error) {
	f := toolEventFilter{
		Session:          sessionID,
		Asc:              true,
		IncludeLifecycle: false,
		Limit:            2000,
	}
	if !afterWrite.IsZero() {
		f.SinceExclusive = afterWrite.UTC().Format(time.RFC3339)
	}
	rows, err := queryToolEvents(ctx, db, projectID, f)
	if err != nil {
		return nil, err
	}
	out := make([]checkpoint.Event, 0, len(rows))
	for _, e := range rows {
		out = append(out, checkpoint.Event{
			Sequence: e.Sequence,
			Action:   e.Action,
			Tool:     e.Tool,
			Path:     e.Path,
			Command:  e.Command,
			Result:   e.Result,
			At:       e.TS,
		})
	}
	return out, nil
}
