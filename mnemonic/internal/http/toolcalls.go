package http

import (
	"net/http"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
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
	Content        string `json:"content"`
}

// handleToolCallCreate appends one harness tool call to the session's event
// stream. A session the harness never registered (sessionStart hook skipped or
// raced) is created on first sight so the call is not lost.
func (s *Server) handleToolCallCreate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var b toolCallBody
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if b.SessionID == "" {
		b.SessionID = id
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()
	created, err := h.Memory().EnsureSession(r.Context(), b.SessionID, projectID, b.Directory, "", b.Agent)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if d := context_harness.DecideGate(b.Content, os.Getenv("SKILLGRID_CTX_BYPASS") == "1", context_harness.SandboxThreshold); d.Gate {
		if _, err := context_harness.StoreToolOutput(r.Context(), h.Store().DB, b.SessionID, projectID, b.ToolName, b.Content); err != nil {
			writeError(w, http.StatusInternalServerError, "sandbox store: "+err.Error())
			return
		}
		b.ContentPreview = d.Summary + " — " + d.Pointer
	}
	payload := memory.HookPayload{
		SessionID:      b.SessionID,
		ToolName:       b.ToolName,
		ActionType:     b.Type,
		File:           memory.StripPrivate(b.Path),
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
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]any{"ok": true, "project": projectID})
}
