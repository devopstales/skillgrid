// Package observability: session-summary rollup (T11). This file derives one
// wide ctx_session_summaries row per distinct session from ctx_proxy_sessions
// (the authoritative per-session source accumulated by the proxy's
// recordTelemetry) and writes them through the shared ctx accessor. The
// /api/session-summaries reader (T10) is unchanged; this is the writer.
package observability

import (
	"encoding/json"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// sessionSummaryJSON is the deterministic shape of the summary_json column:
// the mode and the raw counters straight from the proxy session row, plus the
// clamped savings delta. It is a small, stable, machine-readable digest of the
// session — the reader parses it fail-open and never assumes a shape beyond
// this.
type sessionSummaryJSON struct {
	Mode         string `json:"mode"`
	Requests     int64  `json:"requests"`
	TokensBefore int64  `json:"tokens_before"`
	TokensAfter  int64  `json:"tokens_after"`
	TokensSaved  int64  `json:"tokens_saved"`
}

// SummarizeSessions derives one ctx_session_summaries row per distinct session
// from ctx_proxy_sessions and writes them. Returns the number of rows written.
//
// Fail-open (ADR-0016): a nil store or a missing ctx_proxy_sessions /
// ctx_session_summaries table returns (0, nil) — never an error. The rollup is
// idempotent: PutSessionSummary is INSERT OR REPLACE on the session_id PK, so
// re-rolling the same session updates the row in place and never appends a
// duplicate. The judgment/prefilter/shaping/traffic columns are left at their
// zero/NULL values here — T12/T14 populate them later (do not fabricate).
func SummarizeSessions(st *ctxstore.Store, now int64) (int, error) {
	if st == nil {
		return 0, nil
	}
	sessions, err := st.ListProxySessions()
	if err != nil {
		// Missing table (or other fault): fail open to (0, nil).
		return 0, nil
	}
	written := 0
	for _, sess := range sessions {
		if sess == nil || sess.SessionID == "" {
			continue
		}
		saved := sess.TokensBefore - sess.TokensAfter
		if saved < 0 {
			saved = 0
		}
		duration := sess.EndedAt - sess.StartedAt
		if duration < 0 {
			duration = 0
		}
		payload := sessionSummaryJSON{
			Mode:         sess.Mode,
			Requests:     sess.Requests,
			TokensBefore: sess.TokensBefore,
			TokensAfter:  sess.TokensAfter,
			TokensSaved:  saved,
		}
		blob, jerr := json.Marshal(payload)
		if jerr != nil {
			// Deterministic marshalling of a fixed struct cannot fail; if it
			// ever does, fall back to an empty object rather than erroring.
			blob = []byte("{}")
		}
		row := &ctxstore.SessionSummary{
			SessionID:          sess.SessionID,
			Agent:              sess.Agent,
			DurationMS:         duration,
			Turns:              sess.Requests,
			TokensSaved:        saved,
			JudgmentKeeps:      0,
			JudgmentTruncs:     0,
			PrefilterPins:      0,
			PrefilterDeads:     0,
			CacheInvalidations: sess.CacheInvalidations,
			ShapingApplied:     nil,
			EffortRouted:       0,
			TrafficMemories:    0,
			SummaryJSON:        string(blob),
			CreatedAt:          now,
		}
		if perr := st.PutSessionSummary(row); perr != nil {
			// Fail-open: a write fault (e.g. missing table) drops this row
			// rather than erroring the rollup.
			continue
		}
		written++
	}
	return written, nil
}
