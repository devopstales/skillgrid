package http

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// errSandboxNoRows aliases database/sql.ErrNoRows for the readability of the
// ctx_get no-hit branch.
var errSandboxNoRows = sql.ErrNoRows

// ctx sandbox routes (ADR-0044): the HTTP surface for the context-harness
// sandbox, so native extensions (the Pi tool_result overflow handler, the
// ctx_* Pi tools) store and recall oversized tool output through the Go
// server instead of a local sidecar. The storage authority stays the
// per-project store; these routes are thin transports.
//
//   - POST /ctx/capture  store one oversized output, return its sandbox id
//     (the receipt pointer) and the gate decision. Always stores — the
//     caller already decided the output is oversized (the client mirrors
//     SandboxThreshold); the server gate is reported, not enforced.
//   - GET /ctx/search    two-leg RRF fusion (sandbox + indexed code)
//   - GET /ctx/stats     row counts + total bytes (loopback-only)
//   - DELETE /ctx/sessions/{id}  purge the session's sandbox rows

// ctxCaptureBody is the POST /ctx/capture request. Content is the full tool
// output; StripPrivate applies on store, mirroring the tool-calls route.
type ctxCaptureBody struct {
	SessionID string `json:"session_id"`
	Agent     string `json:"agent"`
	ToolName  string `json:"tool_name"`
	Content   string `json:"content"`
}

func (s *Server) handleCtxCapture(w http.ResponseWriter, r *http.Request) {
	var b ctxCaptureBody
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if b.SessionID == "" || b.ToolName == "" || b.Content == "" {
		writeError(w, http.StatusBadRequest, "session_id, tool_name, and content are required")
		return
	}
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		gate := context_harness.DecideGate(
			b.Content, os.Getenv("SKILLGRID_CTX_BYPASS") == "1", context_harness.SandboxThreshold)
		id, err := context_harness.StoreToolOutput(
			r.Context(), h.Store().DB, b.SessionID, projectID, b.ToolName, memory.StripPrivate(b.Content))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "sandbox store: "+err.Error())
			return
		}
		resp := map[string]any{
			"sandbox_id": id,
			"gated":      gate.Gate,
			"reason":     gate.Reason,
			"pointer":    gate.Pointer,
		}
		if gate.Gate {
			resp["summary"] = gate.Summary
		}
		writeJSON(w, http.StatusOK, resp)
	})
}

// handleCtxSearch runs the two-leg RRF fusion over the project store.
// Fail-open: a store-level error (absent FTS tables on a fresh store)
// returns an empty result set, not a 500 (ADR-0016).
func (s *Server) handleCtxSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	limit := queryInt(r, "limit", 20)
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, _ string) {
		results, err := context_harness.CtxSearch(h.Store().DB, query, limit)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"results": []any{}, "error": err.Error()})
			return
		}
		if results == nil {
			results = make([]context_harness.CtxResult, 0)
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": results, "count": len(results)})
	})
}

// handleCtxGet fetches one full sandboxed tool output by id. The FTS rows
// carry no rowid of the parent table in a stable form across stores, so the
// direct primary-key read is the retrieval path receipts point at.
func (s *Server) handleCtxGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, _ string) {
		var toolName string
		var output string
		var sizeBytes int
		err := h.Store().DB.QueryRowContext(r.Context(),
			`SELECT tool_name, output, size_bytes FROM tool_outputs WHERE id = ?`, id,
		).Scan(&toolName, &output, &sizeBytes)
		if err != nil {
			if errors.Is(err, errSandboxNoRows) || strings.Contains(err.Error(), "no rows") {
				writeError(w, http.StatusNotFound, "sandbox row not found")
				return
			}
			// Fail-open (ADR-0016): absent table on a fresh store reports
			// not-found rather than a 500.
			writeError(w, http.StatusNotFound, "sandbox store unavailable: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"sandbox_id":  id,
			"tool_name":   toolName,
			"size_bytes":  sizeBytes,
			"output":      output,
		})
	})
}

// handleCtxStats reports sandbox row counts and total stored bytes, plus the
// indexed-chunk row count. Loopback-only like the other ctx observability
// routes: a non-loopback origin gets 403 before the store opens.
func (s *Server) handleCtxStats(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "ctx sandbox routes are loopback only"})
		return
	}
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, _ string) {
		db := h.Store().DB
		var sandboxRows, indexedChunks int
		var sandboxBytes int64
		httpRowCount(r.Context(), db, "tool_outputs", &sandboxRows)
		httpRowCount(r.Context(), db, "indexed_files", &indexedChunks)
		_ = db.QueryRowContext(r.Context(),
			`SELECT COALESCE(SUM(size_bytes), 0) FROM tool_outputs`).Scan(&sandboxBytes)
		writeJSON(w, http.StatusOK, map[string]any{
			"sandbox_rows":   sandboxRows,
			"sandbox_bytes":  sandboxBytes,
			"indexed_chunks": indexedChunks,
		})
	})
}

// handleCtxSessionPurge deletes the session's sandbox rows (fail-open: an
// absent table purges zero rows, matching context_harness.PurgeSession).
func (s *Server) handleCtxSessionPurge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
		n, err := context_harness.PurgeSession(h.Store().DB, projectID, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "purged": n})
	})
}
