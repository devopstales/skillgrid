package http

import (
	"net/http"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/policy"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

type policyEvalBody struct {
	SessionID string `json:"session_id"`
	Agent     string `json:"agent"`
	Directory string `json:"directory"`
	Action    string `json:"action"`
	Tool      string `json:"tool"`
	Path      string `json:"path"`
	Command   string `json:"command"`
}

// policyDir is where the repo policy is looked up: the session's directory
// when the harness sent one, otherwise the project root.
func policyDir(directory string, h *service.ProjectHandle) string {
	if strings.TrimSpace(directory) != "" {
		return directory
	}
	return h.Root()
}

// handlePolicyEvaluate serves POST /policy/evaluate (ADR-0021). It answers
// {effect, message, rule}. A broken policy file answers allow with an error
// field (fail-open). block/warn/guide decisions are recorded on the session.
func (s *Server) handlePolicyEvaluate(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var b policyEvalBody
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()

	pol, perr := policy.Load(policyDir(b.Directory, h))
	if perr != nil {
		writeJSON(w, http.StatusOK, map[string]any{"effect": policy.Allow, "message": "", "rule": "", "error": perr.Error()})
		return
	}
	action := b.Action
	if action == "" || (action != "file_read" && action != "file_write" && action != "command_exec" && action != "tool_use") {
		action = memory.ActionForTool(b.Tool, b.Action)
	}
	in := policy.Input{
		Action: action, Path: b.Path, Command: b.Command, Tool: b.Tool,
		Agent: b.Agent, Project: projectID, Directory: b.Directory,
	}
	if b.SessionID != "" && pol.Enabled {
		in.Counters = h.Memory().SessionCounters(r.Context(), b.SessionID)
	}
	d := pol.Evaluate(in)

	recorded := false
	if res := policy.Result(d.Effect); res != "" && b.SessionID != "" {
		if _, err := h.Memory().EnsureSession(r.Context(), b.SessionID, projectID, b.Directory, "", b.Agent); err == nil {
			recorded = h.Memory().RecordPolicyDecision(r.Context(), memory.PolicyDecision{
				SessionID: b.SessionID, Action: action, Tool: b.Tool, Path: b.Path, Command: b.Command,
				Result: res, Rule: d.Rule, Message: d.Message,
			}) == nil
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"effect": d.Effect, "message": d.Message, "rule": d.Rule, "action": action, "recorded": recorded,
	})
}

// handlePolicyGet serves GET /policy — the merged, read-only rule set for the
// Security page.
func (s *Server) handlePolicyGet(w http.ResponseWriter, r *http.Request) {
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		pol, err := policy.Load(policyDir(r.URL.Query().Get("directory"), h))
		resp := map[string]any{"project": projectID, "enabled": false, "rules": []policy.Rule{}, "files": []string{}}
		if err != nil {
			resp["error"] = err.Error()
			writeJSON(w, http.StatusOK, resp)
			return
		}
		resp["enabled"] = pol.Enabled
		if pol.Rules != nil {
			resp["rules"] = pol.Rules
		}
		if pol.Files != nil {
			resp["files"] = pol.Files
		}
		resp["repoFile"] = policy.RepoFile(h.Root())
		writeJSON(w, http.StatusOK, resp)
	})
}
