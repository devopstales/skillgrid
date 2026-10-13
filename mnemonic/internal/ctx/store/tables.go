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

// TrafficMemory is one row of ctx_traffic_memories (061, status columns added
// by 068). Status is one of "candidate" (the learner proposed it; the default
// until a human applies it), "applied" (ctx learn apply promoted it into the
// compression context) or "revoked" (ctx learn revoke demoted it so the
// learner stops re-suggesting it). AppliedAt is a pointer: nil means NULL
// (never applied).
type TrafficMemory struct {
	MemoryID     string
	SessionID    string
	Category     string
	MemoryText   string
	EvidenceJSON string
	Confidence   float64
	CreatedAt    int64
	Status       string
	AppliedAt    *int64
}

// hasTrafficMemoryStatus probes whether the live ctx_traffic_memories table
// carries the 068 status column. Writers use it to decide whether they may
// name the column in an UPDATE/INSERT, and readers use it to drop the
// applied_at column from their SELECT; on a pre-068 schema every traffic
// accessor degrades fail-open rather than erroring (ADR-0016).
func (s *Store) hasTrafficMemoryStatus() bool {
	var n int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('ctx_traffic_memories') WHERE name = 'status'`,
	).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// trafficMemorySelectCols returns the shared SELECT column list for every
// ctx_traffic_memories reader. On a live 068 schema it names status (COALESCEd
// to its 'candidate' default) and applied_at; on a pre-068 schema (no status
// column) it drops applied_at and SELECTs the COALESCE literal, so a row on an
// old store reads as candidate/NULL instead of erroring on the unknown column
// (ADR-0016). The two branches return exactly nine columns, matching
// scanTrafficMemory.
func (s *Store) trafficMemorySelectCols() string {
	if s.hasTrafficMemoryStatus() {
		return `
		memory_id, session_id, category, memory_text,
		evidence_json, confidence, created_at,
		COALESCE(status, 'candidate') AS status,
		applied_at`
	}
	// Pre-068: no status/applied_at columns. SELECT the candidate literal and
	// SQL NULL so the nine-column scan succeeds (applied_at scans to a nil
	// *int64).
	return `
		memory_id, session_id, category, memory_text,
		evidence_json, confidence, created_at,
		'candidate' AS status,
		NULL AS applied_at`
}

// scanTrafficMemory scans one ctx_traffic_memories row.
func (s *Store) scanTrafficMemory(row interface {
	Scan(dest ...any) error
}) (*TrafficMemory, error) {
	var m TrafficMemory
	err := row.Scan(&m.MemoryID, &m.SessionID, &m.Category, &m.MemoryText,
		&m.EvidenceJSON, &m.Confidence, &m.CreatedAt, &m.Status, &m.AppliedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
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
	// A pre-068 store has no status/applied_at columns (ADR-0016): writing them
	// would hard-fail, so insert the base columns only and let the readers
	// COALESCE the status to its 'candidate' default.
	if !s.hasTrafficMemoryStatus() {
		return s.execFailOpen("ctx_traffic_memories", `
			INSERT OR REPLACE INTO ctx_traffic_memories
				(memory_id, session_id, category, memory_text,
				 evidence_json, confidence, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.MemoryID, m.SessionID, m.Category, m.MemoryText,
			m.EvidenceJSON, m.Confidence, m.CreatedAt)
	}
	return s.execFailOpen("ctx_traffic_memories", `
		INSERT OR REPLACE INTO ctx_traffic_memories
			(memory_id, session_id, category, memory_text,
			 evidence_json, confidence, created_at, status, applied_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.MemoryID, m.SessionID, m.Category, m.MemoryText,
		m.EvidenceJSON, m.Confidence, m.CreatedAt, m.Status, m.AppliedAt)
}

func (s *Store) GetTrafficMemory(memoryID string) (*TrafficMemory, error) {
	row := s.db.QueryRow(`
		SELECT `+s.trafficMemorySelectCols()+`
		FROM ctx_traffic_memories WHERE memory_id = ?`, memoryID)
	m, err := s.scanTrafficMemory(row)
	if err == sql.ErrNoRows || failOpenOpen(err, "ctx_traffic_memories") {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) ListTrafficMemories(sessionID string) ([]*TrafficMemory, error) {
	rows, err := s.queryFailOpen("ctx_traffic_memories", `
		SELECT `+s.trafficMemorySelectCols()+`
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
		m, err := s.scanTrafficMemory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
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
		SELECT `+s.trafficMemorySelectCols()+`
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
		m, err := s.scanTrafficMemory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
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

// SetTrafficMemoryStatus transitions one ctx_traffic_memories row to status
// ("candidate" | "applied" | "revoked"). Apply (status="applied") stamps
// appliedAt (unix millis); any demotion away from applied (revoked or
// candidate) clears applied_at to NULL — per the T13 brief, applied_at is set
// only while a memory is active, NULL otherwise.
//
// Returns (true, nil) when the row exists and was updated. Returns
// (false, nil) — NOT an error — when the row does not exist (the gate maps
// this to its not-found error) or the table is missing (ADR-0016 fail-open).
// Any other error (corrupt row, constraint) propagates.
func (s *Store) SetTrafficMemoryStatus(memoryID, status string, appliedAt *int64) (bool, error) {
	if status == "applied" {
		// Activating: stamp the provided applied_at.
		return s.execTrafficStatus(memoryID, status, appliedAt, "set")
	}
	// Demoting (revoked/candidate): clear applied_at so it is NULL again.
	return s.execTrafficStatus(memoryID, status, nil, "clear")
}

// execTrafficStatus runs the status UPDATE, choosing the applied_at handling
// by appliedAtMode: "set" writes the provided timestamp (apply), "clear"
// resets applied_at to NULL (revoke/demote). A missing table or a pre-068
// schema (no status column) degrades to (false, nil) — the write is lost, not
// fatal (ADR-0016).
func (s *Store) execTrafficStatus(memoryID, status string, appliedAt *int64, appliedAtMode string) (bool, error) {
	if !s.hasTrafficMemoryStatus() {
		// Pre-068 schema (or missing table): the status cannot be persisted.
		// Degrade to a no-op rather than an unknown-column error.
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM ctx_traffic_memories WHERE memory_id = ?`, memoryID).Scan(&exists); err != nil {
			if failOpenOpen(err, "ctx_traffic_memories") {
				return false, nil
			}
			return false, err
		}
		return exists > 0, nil
	}
	query := `UPDATE ctx_traffic_memories SET status = ?`
	if appliedAtMode == "set" {
		query += `, applied_at = ?`
	} else {
		query += `, applied_at = NULL`
	}
	query += ` WHERE memory_id = ?`
	args := []any{status}
	if appliedAtMode == "set" {
		args = append(args, appliedAt)
	}
	args = append(args, memoryID)
	res, err := s.db.Exec(query, args...)
	if err != nil {
		if failOpenOpen(err, "ctx_traffic_memories") {
			return false, nil
		}
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
