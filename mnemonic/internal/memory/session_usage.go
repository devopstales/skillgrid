package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// SessionUsage is a session's running token and cost totals. CostUSD is nil
// until usage for a priced model is reported.
type SessionUsage struct {
	SessionID    string   `json:"session_id"`
	Model        string   `json:"model"`
	InputTokens  int64    `json:"input_tokens"`
	OutputTokens int64    `json:"output_tokens"`
	CacheTokens  int64    `json:"cache_tokens"`
	CostUSD      *float64 `json:"cost_usd"`
}

// UsageReport is one harness usage report. Total=true replaces the session's
// counters (harnesses that re-read their own transcript on every idle); the
// default adds a delta. Cost is the report's cost (nil when unpriced): it is
// added to, or replaces, the running cost only when non-nil.
type UsageReport struct {
	Model  string
	Input  int64
	Output int64
	Cache  int64
	Cost   *float64
	Total  bool
}

// RecordSessionUsage applies a usage report to an existing session row.
func (s *Service) RecordSessionUsage(ctx context.Context, sessionID string, u UsageReport) (SessionUsage, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return SessionUsage{}, errors.New("memory service not initialized")
	}
	if u.Input < 0 || u.Output < 0 || u.Cache < 0 {
		return SessionUsage{}, errors.New("token counts must be non-negative")
	}
	sessionID = strings.TrimSpace(sessionID)
	model := strings.TrimSpace(u.Model)

	var cost sql.NullFloat64
	if u.Cost != nil {
		cost = sql.NullFloat64{Float64: *u.Cost, Valid: true}
	}
	var q string
	if u.Total {
		q = `UPDATE sessions SET
			input_tokens = ?, output_tokens = ?, cache_tokens = ?,
			model = COALESCE(NULLIF(?, ''), model),
			cost_usd = COALESCE(?, cost_usd)
			WHERE id = ?`
	} else {
		q = `UPDATE sessions SET
			input_tokens = input_tokens + ?, output_tokens = output_tokens + ?, cache_tokens = cache_tokens + ?,
			model = COALESCE(NULLIF(?, ''), model),
			cost_usd = CASE WHEN ? IS NULL THEN cost_usd ELSE COALESCE(cost_usd, 0) + ? END
			WHERE id = ?`
	}
	args := []any{u.Input, u.Output, u.Cache, model, cost}
	if !u.Total {
		args = append(args, cost)
	}
	args = append(args, sessionID)
	res, err := s.store.DB.ExecContext(ctx, q, args...)
	if err != nil {
		return SessionUsage{}, fmt.Errorf("record usage: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return SessionUsage{}, fmt.Errorf("session %s not found", sessionID)
	}
	return s.GetSessionUsage(ctx, sessionID)
}

// GetSessionUsage reads a session's token and cost totals.
func (s *Service) GetSessionUsage(ctx context.Context, sessionID string) (SessionUsage, error) {
	out := SessionUsage{SessionID: sessionID}
	var cost sql.NullFloat64
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT COALESCE(model,''), input_tokens, output_tokens, cache_tokens, cost_usd
		FROM sessions WHERE id = ?`, sessionID).
		Scan(&out.Model, &out.InputTokens, &out.OutputTokens, &out.CacheTokens, &cost)
	if err != nil {
		return SessionUsage{}, err
	}
	if cost.Valid {
		v := cost.Float64
		out.CostUSD = &v
	}
	return out, nil
}
