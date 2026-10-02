package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

type toolCallBody struct {
	SessionID      string `json:"session_id"`
	Agent          string `json:"agent"`
	Directory      string `json:"directory"`
	Type           string `json:"type"`
	ToolName       string `json:"tool_name"`
	Path           string `json:"path"`
	Command        string `json:"command"`
	ResultStatus   string `json:"result_status"`
	ContentHash    string `json:"content_hash"`
	ContentPreview string `json:"content_preview"`
}

// handleToolCallCreate appends one harness tool call to the session's event
// stream. A session the harness never registered (sessionStart hook skipped or
// raced) is created on first sight so the call is not lost.
func (s *Server) handleToolCallCreate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var b toolCallBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if b.SessionID == "" {
		b.SessionID = id
	}
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
	if _, err := h.Memory().EnsureSession(r.Context(), b.SessionID, projectID, b.Directory, "", b.Agent); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := memory.HookPayload{
		SessionID:      b.SessionID,
		ToolName:       b.ToolName,
		ActionType:     b.Type,
		File:           b.Path,
		Command:        memory.StripPrivate(b.Command),
		ResultStatus:   b.ResultStatus,
		ContentHash:    b.ContentHash,
		ContentPreview: memory.StripPrivate(b.ContentPreview),
	}
	if _, err := h.Memory().RunHook(r.Context(), memory.HookPostToolUse, payload); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
