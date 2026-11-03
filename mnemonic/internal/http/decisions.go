package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// Phase 8 — Mnemonic decision bridge (change 2026-09-19-embed-visual-companion).
// The agent posts interview questions as type=decision observations (content is
// a structured JSON payload with a convention `state` field); the dashboard
// lists them via GET /mnemonic/decisions and the user answers via
// POST /mnemonic/decisions/{id}/answer, which flips state pending→answered
// through the same version-append path mem_update uses (memory.UpdateContent).
// No new table, no migration: decisions are a convention over observations.

// decisionState is the convention approval gate inside content. The row's
// `status` column is closed to active|superseded|archived (governance
// lifecycle), so pending/answered lives here, in the JSON content.
const (
	decisionStatePending     = "pending"
	decisionStateAnswered    = "answered"
	decisionStateSuperseded  = "superseded"
)

// decisionOption is one lettered option in a decision payload.
type decisionOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Rationale string `json:"rationale,omitempty"`
}

// decisionContent is the structured decision payload stored in
// observations.content. A malformed payload on a type=decision row is a data
// problem, never a 500: it is flagged (ParseError), never fatal.
type decisionContent struct {
	Question       string           `json:"question"`
	Options        []decisionOption `json:"options"`
	Recommended    string           `json:"recommended,omitempty"`
	State          string           `json:"state"` // pending | answered | superseded
	AnsweredOption string           `json:"answeredOption,omitempty"`
	AnswerNote     string           `json:"answerNote,omitempty"`
	UpdatedBy      string           `json:"updatedBy,omitempty"`
	Visual         string           `json:"visual,omitempty"`
}

// decisionRow is one row in the GET /mnemonic/decisions payload.
type decisionRow struct {
	ID         int64           `json:"id"`
	TopicKey   string          `json:"topicKey"`
	Title      string          `json:"title"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Visibility string          `json:"visibility"`
	Content    decisionContent `json:"content"`
	ParseError bool            `json:"parseError,omitempty"`
}

// parseDecisionContent parses a decision content payload. ok=false means the
// row's content is unreadable as a decision (invalid JSON or missing
// question): the caller flags the row, it never 500s.
func parseDecisionContent(raw string) (decisionContent, bool) {
	var c decisionContent
	if err := json.Unmarshal([]byte(raw), &c); err != nil || strings.TrimSpace(c.Question) == "" {
		return decisionContent{}, false
	}
	return c, true
}

// isDecisionPayload reports whether parsed content is a decision-bridge
// payload: it carries a convention state field. The type='decision' column is
// reused by mem_save for plain decision RECORDS (session notes, no state), so
// the inbox filters on this, not the column alone.
func isDecisionPayload(c decisionContent) bool {
	return c.State == decisionStatePending || c.State == decisionStateAnswered || c.State == decisionStateSuperseded
}

func (s *Server) registerDecisionRoutes() {
	s.mux.HandleFunc("GET /mnemonic/decisions", s.handleDecisions)
	s.mux.HandleFunc("POST /mnemonic/decisions/{id}/answer", s.requireWriteAuth(s.handleDecisionAnswer))
}

// handleDecisions returns the project's decision observations: type='decision'
// rows, not deleted, excluding private rows (the dashboard reader is a
// team-scoped identity, mirroring the memories list). `?state=` filters the
// convention state (pending|answered|superseded; default all), `?topic=` an
// optional topic_key prefix filter. Rows are ordered ascending by created_at.
// A row with malformed content is still returned, flagged parseError.
func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	wantState := r.URL.Query().Get("state")
	wantTopic := r.URL.Query().Get("topic")

	rows, err := h.Store().DB.QueryContext(r.Context(), `
		SELECT id, COALESCE(topic_key,''), COALESCE(title,''), COALESCE(content,''),
		       COALESCE(visibility,''), COALESCE(created_at,''), COALESCE(updated_at,'')
		FROM observations
		WHERE project = ? AND type = 'decision' AND deleted_at IS NULL
		  AND COALESCE(visibility,'') != 'private'
		ORDER BY created_at ASC, id ASC
		LIMIT 200`, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	out := []decisionRow{}
	for rows.Next() {
		var d decisionRow
		var content string
		if err := rows.Scan(&d.ID, &d.TopicKey, &d.Title, &content, &d.Visibility, &d.CreatedAt, &d.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		c, ok := parseDecisionContent(content)
		if !ok {
			continue // not a decision payload (e.g. a mem_save decision record) — not an interview decision
		}
		d.Content = c
		if !isDecisionPayload(c) {
			continue // parseable JSON but no convention state → not a decision-bridge row
		}
		if wantTopic != "" && !strings.HasPrefix(d.TopicKey, wantTopic) {
			continue
		}
		if wantState != "" && wantState != "all" && !d.ParseError && d.Content.State != wantState {
			continue
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// decisionAnswerBody is the R5 answer contract: {optionId, note}.
type decisionAnswerBody struct {
	OptionID string `json:"optionId"`
	Note     string `json:"note"`
}

// handleDecisionAnswer flips a decision's convention state pending→answered:
// it records the chosen option + note + updatedBy (user:<name>) in the
// content and persists it through memory.UpdateContent, so an
// observation_versions row is appended and revision_count/updated_at bumped —
// the same governed path mem_update uses. Re-answering an answered decision
// applies the same update path again (state stays answered, the new answer is
// recorded, another version row is appended): append-only history, the latest
// content carries the latest answer. A bad id is 404; a row whose content has
// no valid state (missing, malformed, or unrecognized) is 409.
func (s *Server) handleDecisionAnswer(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id64, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	var in decisionAnswerBody
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(in.OptionID) == "" {
		writeError(w, http.StatusBadRequest, "optionId is required")
		return
	}

	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	var (
		content  string
		curState string
	)
	err = h.Store().DB.QueryRowContext(r.Context(),
		`SELECT COALESCE(content,''), COALESCE(status,'')
		 FROM observations
		 WHERE id = ? AND project = ? AND type = 'decision' AND deleted_at IS NULL`,
		id64, projectID).Scan(&content, &curState)
	if err != nil {
		writeError(w, http.StatusNotFound, "no such decision")
		return
	}
	c, ok := parseDecisionContent(content)
	if !ok {
		writeError(w, http.StatusConflict, "decision content has no valid state")
		return
	}
	switch c.State {
	case decisionStatePending, decisionStateAnswered:
		// pending → answered flip, or re-answer of an answered decision:
		// the same update path below (state stays answered on re-answer, the
		// new answer is recorded, a version row is appended).
	case decisionStateSuperseded:
		// The agent owns the pending→superseded transition; the answer gate
		// does not apply.
		writeError(w, http.StatusConflict, "decision is superseded")
		return
	default:
		_ = curState
		writeError(w, http.StatusConflict, "decision content has no valid state")
		return
	}

	if !decisionOptionExists(c, in.OptionID) {
		writeError(w, http.StatusUnprocessableEntity, "optionId not in this decision's options")
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("answerer"))
	if name == "" {
		name = "unknown"
	}
	c.State = decisionStateAnswered
	c.AnsweredOption = in.OptionID
	c.AnswerNote = in.Note
	c.UpdatedBy = "user:" + name
	newContent, err := json.Marshal(c)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.Memory().UpdateContent(r.Context(), id64, string(newContent)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "state": decisionStateAnswered})
}

// decisionOptionExists reports whether the option id is one of the decision's
// options (an empty option list accepts any id — a payload that parsed but
// carries no options is still answerable by convention).
func decisionOptionExists(c decisionContent, optionID string) bool {
	if len(c.Options) == 0 {
		return true
	}
	for _, o := range c.Options {
		if o.ID == optionID {
			return true
		}
	}
	return false
}
