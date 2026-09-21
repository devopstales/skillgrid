package http

import (
	"encoding/json"
	"net/http"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/handoff"
)

// handleHandoffSnapshots serves GET /handoff/snapshots?limit=N — the
// change-snapshot log (change_snapshots), newest-first. This is the Handoff
// Hub's change log backed by the git-derived snapshot index (change
// 015-handoff-hub). It is a read-only view; the durable record is the commit
// (and its [skillgrid-context] block). The route was renamed from
// /activity/snapshots in sessions-activity-unification.
func (s *Server) handleHandoffSnapshots(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	hub := &handoff.Hub{DB: h.Store().DB, Project: h.ProjectID(), RepoDir: h.Root()}

	limit := queryInt(r, "limit", 100)
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}
	snapshots, err := hub.ListSnapshots(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]any, 0, len(snapshots))
	for _, sn := range snapshots {
		out = append(out, snapshotToJSON(sn))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project":   projectID,
		"snapshots": out,
		"limit":     limit,
	})
}

// handleHandoffStatus serves GET /handoff/status — the one-call "where are we"
// hub view: latest snapshot, open checkpoints, and recent handoff refs.
func (s *Server) handleHandoffStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	hub := &handoff.Hub{DB: h.Store().DB, Project: h.ProjectID(), RepoDir: h.Root()}

	snapshots, _ := hub.ListSnapshots(r.Context(), 1)
	checkpoints, _ := hub.ListCheckpoints(r.Context(), 50)
	refs, _ := hub.ListHandoffRefs(r.Context(), 50)
	var latest any
	if len(snapshots) > 0 {
		latest = snapshotToJSON(snapshots[0])
	}
	cps := make([]any, 0, len(checkpoints))
	for _, c := range checkpoints {
		cps = append(cps, map[string]any{
			"id":          c.ID,
			"name":        c.Name,
			"branch":      c.Branch,
			"commit":      c.Commit,
			"dirty":       c.Dirty,
			"prdPath":     c.PRDPath,
			"specDir":     c.SpecDir,
			"handoffFile": c.HandoffFile,
			"evidence":    c.Evidence,
			"status":      c.Status,
			"createdAt":   c.CreatedAt,
			"verifiedAt":  c.VerifiedAt,
		})
	}
	refOut := make([]any, 0, len(refs))
	for _, rf := range refs {
		refOut = append(refOut, map[string]any{
			"handoffId":   rf.HandoffID,
			"handoffType": rf.HandoffType,
			"fromCommit":  rf.FromCommit,
			"toCommit":    rf.ToCommit,
			"specDir":     rf.SpecDir,
			"teamId":      rf.TeamID,
			"taskId":      rf.TaskID,
			"createdAt":   rf.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project":         projectID,
		"latest_snapshot": latest,
		"checkpoints":     cps,
		"handoff_refs":    refOut,
	})
}

// snapshotToJSON maps a handoff.Snapshot to the camelCase JSON shape the admin
// UI expects (the raw struct has Go-cased fields).
func snapshotToJSON(sn handoff.Snapshot) map[string]any {
	out := map[string]any{
		"id":            sn.ID,
		"branch":        sn.Branch,
		"commit":        sn.Commit,
		"commitShort":   sn.CommitShort,
		"subject":       sn.Subject,
		"author":        sn.Author,
		"committedAt":   sn.CommittedAt,
		"changedFiles":  sn.ChangedFiles,
	}
	if sn.ContextJSON != "" {
		var c struct {
			Task      string `json:"task"`
			Decisions string `json:"decisions"`
			Remaining string `json:"remaining"`
			Tried     string `json:"tried"`
		}
		if json.Unmarshal([]byte(sn.ContextJSON), &c) == nil {
			out["context"] = c
		}
	}
	return out
}
