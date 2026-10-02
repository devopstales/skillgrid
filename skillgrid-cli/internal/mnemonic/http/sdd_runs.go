package http

import (
	"context"
	"net/http"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/docs"
)

// handleTeamRuns serves GET /sdd/runs: the execution ledgers (ADR-0020) with
// each row joined to the Mnemonic session that worked it, so the Teams view can
// show the harness, its live state, and a link to that session.
func (s *Server) handleTeamRuns(w http.ResponseWriter, r *http.Request) {
	root := sddRoot()
	runs, err := docs.ListTeamRuns(r.Context(), root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sessions := s.repoSessions(r.Context(), root)
	now := time.Now().UTC()
	for i := range runs {
		for j := range runs[i].Members {
			ms := docs.MatchSession(runs[i].Members[j], sessions)
			if ms == nil {
				continue
			}
			ms.Live = docs.LiveStatus(ms.Status, ms.LastActive, now)
			runs[i].Members[j].Session = ms
		}
	}
	writeJSON(w, http.StatusOK, docs.TeamRunsPayload(runs))
}

// repoSessions lists the sessions in the project that owns repo, the store
// mem_session_start writes to for agents working in that checkout.
func (s *Server) repoSessions(ctx context.Context, repo string) []docs.MemberSession {
	h, cleanup, err := s.openHandleForDir(repo)
	if err != nil {
		return nil
	}
	defer cleanup()
	projectID := h.ProjectID()
	rows, err := h.Store().DB.QueryContext(ctx, `
		SELECT s.id, COALESCE(s.agent,''), COALESCE(s.status,'active'), COALESCE(s.title,''),
		       COALESCE(s.started_at,''),
		       COALESCE((SELECT MAX(e.timestamp) FROM session_events e WHERE e.session_id = s.id),
		                s.ended_at, s.started_at, '')
		FROM sessions s
		WHERE s.project = ? AND COALESCE(TRIM(s.title),'') != ''`, projectID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []docs.MemberSession
	for rows.Next() {
		ms := docs.MemberSession{Project: projectID}
		if err := rows.Scan(&ms.ID, &ms.Agent, &ms.Status, &ms.Title, &ms.StartedAt, &ms.LastActive); err != nil {
			return out
		}
		out = append(out, ms)
	}
	return out
}
