// Typed rows and CRUD for the six ctx_* tables (plan §9.3, TICKET-02). The
// accessors are thin over the shared handle: every SELECT degrades to
// empty/zero and every write degrades to a no-op when the table is missing
// on this database (ADR-0016 fail-open).
package store

import (
	"database/sql"
)

// ProxySession is one row of ctx_proxy_sessions (056).
type ProxySession struct {
	SessionID          string
	Agent              string
	Mode               string
	StartedAt          int64
	EndedAt            int64
	Requests           int64
	TokensBefore       int64
	TokensAfter        int64
	CacheHits          int64
	CacheInvalidations int64
}

// ProxySaving is one row of ctx_proxy_savings (057).
type ProxySaving struct {
	SavingID     string
	SessionID    string
	Turn         int64
	Upstream     string
	Mode         string
	TokensBefore int64
	TokensAfter  int64
	SavingsRatio float64
	CreatedAt    int64
}

// ModeChange is one row of ctx_proxy_mode_history (058).
type ModeChange struct {
	ChangeID  string
	SessionID string
	OldMode   string
	NewMode   string
	Reason    string
	ChangedAt int64
}

// SessionSummary is one row of ctx_session_summaries (059). ShapingApplied
// is a pointer: nil means the column is NULL.
type SessionSummary struct {
	SessionID          string
	Agent              string
	DurationMS         int64
	Turns              int64
	TokensSaved        int64
	JudgmentKeeps      int64
	JudgmentTruncs     int64
	PrefilterPins      int64
	PrefilterDeads     int64
	CacheInvalidations int64
	ShapingApplied     *bool
	EffortRouted       int64
	TrafficMemories    int64
	SummaryJSON        string
	CreatedAt          int64
}

// StatsBucket is one row of ctx_stats_history (060).
type StatsBucket struct {
	BucketID    string
	Granularity string
	BucketStart int64
	TokensSaved int64
	Requests    int64
	Sessions    int64
	CacheHits   int64
	CreatedAt   int64
}

// RollupRow is one aggregated row of Store.Rollup: the summed counters for a
// single granularity across all buckets of ctx_stats_history.
type RollupRow struct {
	Granularity string
	TokensSaved int64
	Requests    int64
	Sessions    int64
	CacheHits   int64
}

// TrafficMemory is one row of ctx_traffic_memories (061).
type TrafficMemory struct {
	MemoryID     string
	SessionID    string
	Category     string
	MemoryText   string
	EvidenceJSON string
	Confidence   float64
	CreatedAt    int64
}

// ---- ctx_proxy_sessions ----

func (s *Store) PutProxySession(p *ProxySession) error {
	return s.execFailOpen("ctx_proxy_sessions", `
		INSERT OR REPLACE INTO ctx_proxy_sessions
			(session_id, agent, mode, started_at, ended_at, requests,
			 tokens_before, tokens_after, cache_hits, cache_invalidations)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.SessionID, p.Agent, p.Mode, p.StartedAt, p.EndedAt, p.Requests,
		p.TokensBefore, p.TokensAfter, p.CacheHits, p.CacheInvalidations)
}

func (s *Store) GetProxySession(sessionID string) (*ProxySession, error) {
	var p ProxySession
	err := s.db.QueryRow(`
		SELECT session_id, agent, mode, started_at, ended_at, requests,
		       tokens_before, tokens_after, cache_hits, cache_invalidations
		FROM ctx_proxy_sessions WHERE session_id = ?`, sessionID).
		Scan(&p.SessionID, &p.Agent, &p.Mode, &p.StartedAt, &p.EndedAt, &p.Requests,
			&p.TokensBefore, &p.TokensAfter, &p.CacheHits, &p.CacheInvalidations)
	if err == sql.ErrNoRows || failOpenOpen(err, "ctx_proxy_sessions") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ListProxySessions() ([]*ProxySession, error) {
	rows, err := s.queryFailOpen("ctx_proxy_sessions", `
		SELECT session_id, agent, mode, started_at, ended_at, requests,
		       tokens_before, tokens_after, cache_hits, cache_invalidations
		FROM ctx_proxy_sessions ORDER BY started_at, session_id`)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var out []*ProxySession
	for rows.Next() {
		var p ProxySession
		if err := rows.Scan(&p.SessionID, &p.Agent, &p.Mode, &p.StartedAt, &p.EndedAt,
			&p.Requests, &p.TokensBefore, &p.TokensAfter, &p.CacheHits, &p.CacheInvalidations); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ---- ctx_proxy_savings ----

func (s *Store) PutProxySaving(p *ProxySaving) error {
	return s.execFailOpen("ctx_proxy_savings", `
		INSERT OR REPLACE INTO ctx_proxy_savings
			(saving_id, session_id, turn, upstream, mode,
			 tokens_before, tokens_after, savings_ratio, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.SavingID, p.SessionID, p.Turn, p.Upstream, p.Mode,
		p.TokensBefore, p.TokensAfter, p.SavingsRatio, p.CreatedAt)
}

func (s *Store) GetProxySaving(savingID string) (*ProxySaving, error) {
	var p ProxySaving
	err := s.db.QueryRow(`
		SELECT saving_id, session_id, turn, upstream, mode,
		       tokens_before, tokens_after, savings_ratio, created_at
		FROM ctx_proxy_savings WHERE saving_id = ?`, savingID).
		Scan(&p.SavingID, &p.SessionID, &p.Turn, &p.Upstream, &p.Mode,
			&p.TokensBefore, &p.TokensAfter, &p.SavingsRatio, &p.CreatedAt)
	if err == sql.ErrNoRows || failOpenOpen(err, "ctx_proxy_savings") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ListProxySavings(sessionID string) ([]*ProxySaving, error) {
	rows, err := s.queryFailOpen("ctx_proxy_savings", `
		SELECT saving_id, session_id, turn, upstream, mode,
		       tokens_before, tokens_after, savings_ratio, created_at
		FROM ctx_proxy_savings WHERE session_id = ? ORDER BY turn, saving_id`, sessionID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var out []*ProxySaving
	for rows.Next() {
		var p ProxySaving
		if err := rows.Scan(&p.SavingID, &p.SessionID, &p.Turn, &p.Upstream, &p.Mode,
			&p.TokensBefore, &p.TokensAfter, &p.SavingsRatio, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ---- ctx_proxy_mode_history ----

func (s *Store) PutModeChange(c *ModeChange) error {
	return s.execFailOpen("ctx_proxy_mode_history", `
		INSERT OR REPLACE INTO ctx_proxy_mode_history
			(change_id, session_id, old_mode, new_mode, reason, changed_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		c.ChangeID, c.SessionID, c.OldMode, c.NewMode, c.Reason, c.ChangedAt)
}

func (s *Store) GetModeChange(changeID string) (*ModeChange, error) {
	var c ModeChange
	err := s.db.QueryRow(`
		SELECT change_id, session_id, old_mode, new_mode, reason, changed_at
		FROM ctx_proxy_mode_history WHERE change_id = ?`, changeID).
		Scan(&c.ChangeID, &c.SessionID, &c.OldMode, &c.NewMode, &c.Reason, &c.ChangedAt)
	if err == sql.ErrNoRows || failOpenOpen(err, "ctx_proxy_mode_history") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ---- ctx_session_summaries ----

func (s *Store) PutSessionSummary(p *SessionSummary) error {
	return s.execFailOpen("ctx_session_summaries", `
		INSERT OR REPLACE INTO ctx_session_summaries
			(session_id, agent, duration_ms, turns, tokens_saved,
			 judgment_keeps, judgment_truncs, prefilter_pins, prefilter_deads,
			 cache_invalidations, shaping_applied, effort_routed,
			 traffic_memories, summary_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.SessionID, p.Agent, p.DurationMS, p.Turns, p.TokensSaved,
		p.JudgmentKeeps, p.JudgmentTruncs, p.PrefilterPins, p.PrefilterDeads,
		p.CacheInvalidations, p.ShapingApplied, p.EffortRouted,
		p.TrafficMemories, p.SummaryJSON, p.CreatedAt)
}

func (s *Store) GetSessionSummary(sessionID string) (*SessionSummary, error) {
	var p SessionSummary
	err := s.db.QueryRow(`
		SELECT session_id, agent, duration_ms, turns, tokens_saved,
		       judgment_keeps, judgment_truncs, prefilter_pins, prefilter_deads,
		       cache_invalidations, shaping_applied, effort_routed,
		       traffic_memories, summary_json, created_at
		FROM ctx_session_summaries WHERE session_id = ?`, sessionID).
		Scan(&p.SessionID, &p.Agent, &p.DurationMS, &p.Turns, &p.TokensSaved,
			&p.JudgmentKeeps, &p.JudgmentTruncs, &p.PrefilterPins, &p.PrefilterDeads,
			&p.CacheInvalidations, &p.ShapingApplied, &p.EffortRouted,
			&p.TrafficMemories, &p.SummaryJSON, &p.CreatedAt)
	if err == sql.ErrNoRows || failOpenOpen(err, "ctx_session_summaries") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ListSessionSummaries() ([]*SessionSummary, error) {
	rows, err := s.queryFailOpen("ctx_session_summaries", `
		SELECT session_id, agent, duration_ms, turns, tokens_saved,
		       judgment_keeps, judgment_truncs, prefilter_pins, prefilter_deads,
		       cache_invalidations, shaping_applied, effort_routed,
		       traffic_memories, summary_json, created_at
		FROM ctx_session_summaries ORDER BY created_at, session_id`)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var out []*SessionSummary
	for rows.Next() {
		var p SessionSummary
		if err := rows.Scan(&p.SessionID, &p.Agent, &p.DurationMS, &p.Turns,
			&p.TokensSaved, &p.JudgmentKeeps, &p.JudgmentTruncs, &p.PrefilterPins,
			&p.PrefilterDeads, &p.CacheInvalidations, &p.ShapingApplied,
			&p.EffortRouted, &p.TrafficMemories, &p.SummaryJSON, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ---- ctx_stats_history ----

func (s *Store) PutStatsBucket(b *StatsBucket) error {
	return s.execFailOpen("ctx_stats_history", `
		INSERT OR REPLACE INTO ctx_stats_history
			(bucket_id, granularity, bucket_start, tokens_saved,
			 requests, sessions, cache_hits, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.BucketID, b.Granularity, b.BucketStart, b.TokensSaved,
		b.Requests, b.Sessions, b.CacheHits, b.CreatedAt)
}

func (s *Store) GetStatsBucket(bucketID string) (*StatsBucket, error) {
	var b StatsBucket
	err := s.db.QueryRow(`
		SELECT bucket_id, granularity, bucket_start, tokens_saved,
		       requests, sessions, cache_hits, created_at
		FROM ctx_stats_history WHERE bucket_id = ?`, bucketID).
		Scan(&b.BucketID, &b.Granularity, &b.BucketStart, &b.TokensSaved,
			&b.Requests, &b.Sessions, &b.CacheHits, &b.CreatedAt)
	if err == sql.ErrNoRows || failOpenOpen(err, "ctx_stats_history") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) ListStatsBuckets(granularity string) ([]*StatsBucket, error) {
	rows, err := s.queryFailOpen("ctx_stats_history", `
		SELECT bucket_id, granularity, bucket_start, tokens_saved,
		       requests, sessions, cache_hits, created_at
		FROM ctx_stats_history WHERE granularity = ?
		ORDER BY bucket_start, bucket_id`, granularity)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var out []*StatsBucket
	for rows.Next() {
		var b StatsBucket
		if err := rows.Scan(&b.BucketID, &b.Granularity, &b.BucketStart, &b.TokensSaved,
			&b.Requests, &b.Sessions, &b.CacheHits, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

// Rollup aggregates ctx_stats_history into tokens_saved / requests /
// sessions / cache_hits per granularity (summing across all buckets of that
// granularity), ordered by granularity. On a missing table it returns an
// empty slice (ADR-0016).
func (s *Store) Rollup() ([]RollupRow, error) {
	rows, err := s.queryFailOpen("ctx_stats_history", `
		SELECT granularity,
		       COALESCE(SUM(tokens_saved), 0),
		       COALESCE(SUM(requests), 0),
		       COALESCE(SUM(sessions), 0),
		       COALESCE(SUM(cache_hits), 0)
		FROM ctx_stats_history
		GROUP BY granularity
		ORDER BY granularity`)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var out []RollupRow
	for rows.Next() {
		var r RollupRow
		if err := rows.Scan(&r.Granularity, &r.TokensSaved,
			&r.Requests, &r.Sessions, &r.CacheHits); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---- ctx_traffic_memories ----

func (s *Store) PutTrafficMemory(m *TrafficMemory) error {
	return s.execFailOpen("ctx_traffic_memories", `
		INSERT OR REPLACE INTO ctx_traffic_memories
			(memory_id, session_id, category, memory_text,
			 evidence_json, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.MemoryID, m.SessionID, m.Category, m.MemoryText,
		m.EvidenceJSON, m.Confidence, m.CreatedAt)
}

func (s *Store) GetTrafficMemory(memoryID string) (*TrafficMemory, error) {
	var m TrafficMemory
	err := s.db.QueryRow(`
		SELECT memory_id, session_id, category, memory_text,
		       evidence_json, confidence, created_at
		FROM ctx_traffic_memories WHERE memory_id = ?`, memoryID).
		Scan(&m.MemoryID, &m.SessionID, &m.Category, &m.MemoryText,
			&m.EvidenceJSON, &m.Confidence, &m.CreatedAt)
	if err == sql.ErrNoRows || failOpenOpen(err, "ctx_traffic_memories") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) ListTrafficMemories(sessionID string) ([]*TrafficMemory, error) {
	rows, err := s.queryFailOpen("ctx_traffic_memories", `
		SELECT memory_id, session_id, category, memory_text,
		       evidence_json, confidence, created_at
		FROM ctx_traffic_memories WHERE session_id = ?
		ORDER BY created_at, memory_id`, sessionID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var out []*TrafficMemory
	for rows.Next() {
		var m TrafficMemory
		if err := rows.Scan(&m.MemoryID, &m.SessionID, &m.Category, &m.MemoryText,
			&m.EvidenceJSON, &m.Confidence, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// ListAllTrafficMemories returns every ctx_traffic_memories row across all
// sessions, ordered by created_at then memory_id. Added for the traffic
// learner's global max_memories eviction (T12); additive — it does not change
// the table DDL. Fail-open to an empty slice when the table is missing
// (ADR-0016).
func (s *Store) ListAllTrafficMemories() ([]*TrafficMemory, error) {
	rows, err := s.queryFailOpen("ctx_traffic_memories", `
		SELECT memory_id, session_id, category, memory_text,
		       evidence_json, confidence, created_at
		FROM ctx_traffic_memories ORDER BY created_at, memory_id`)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()
	var out []*TrafficMemory
	for rows.Next() {
		var m TrafficMemory
		if err := rows.Scan(&m.MemoryID, &m.SessionID, &m.Category, &m.MemoryText,
			&m.EvidenceJSON, &m.Confidence, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// DeleteTrafficMemory removes one ctx_traffic_memories row by memory_id.
// Added for the traffic learner's eviction (T12); additive — it does not
// change the table DDL. Fail-open: a missing table degrades to a no-op
// (ADR-0016).
func (s *Store) DeleteTrafficMemory(memoryID string) error {
	return s.execFailOpen("ctx_traffic_memories", `
		DELETE FROM ctx_traffic_memories WHERE memory_id = ?`, memoryID)
}
