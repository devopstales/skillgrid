// Package observability implements the read-only CTX observability surface
// (plan §17): a per-request bucket writer that folds proxy savings into
// ctx_stats_history, a periodic rollup runner that re-derives buckets
// without a restart, and the Stats / StatsHistory readers that back the
// GET /stats and GET /stats-history endpoints.
//
// Every read degrades to empty/zero on a missing ctx_* table (ADR-0016
// fail-open); every write degrades to a no-op. The package opens no
// database of its own — it works through a *store.Store (the ctx accessor
// over the shared project store, ADR-0033).
package observability

import (
	"fmt"
	"sync"
	"time"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// Granularity is one of the four standard rollup windows.
type Granularity string

const (
	// Hourly is the finest rollup window.
	Hourly Granularity = "hourly"
	// Daily rolls up per calendar day.
	Daily Granularity = "daily"
	// Weekly rolls up per 7-day window (aligned to the Unix epoch, not to
	// ISO week boundaries).
	Weekly Granularity = "weekly"
	// Monthly rolls up per calendar month.
	Monthly Granularity = "monthly"
)

// DefaultGranularities is the standard rollup set (plan §17.2, config
// default). Used when the caller does not supply an explicit list.
var DefaultGranularities = []Granularity{Hourly, Daily, Weekly, Monthly}

// Stats is the live session-metric snapshot returned by GET /stats
// (plan §17.2, §17.5). Mode is the latest session's mode (seam for the T11
// run-mode header; not wired here). Every counter is zero-valued when the
// ctx tables are missing (ADR-0016).
type Stats struct {
	Mode               string `json:"mode"`
	TokensSaved        int64  `json:"tokens_saved"`
	CacheHits          int64  `json:"cache_hits"`
	CacheInvalidations int64  `json:"cache_invalidations"`
	Requests           int64  `json:"requests"`
	Sessions           int    `json:"sessions"`
}

// Bucket is one row of the durable savings history for a single
// granularity window.
type Bucket struct {
	Granularity Granularity `json:"granularity"`
	BucketStart int64       `json:"bucket_start"`
	TokensSaved int64       `json:"tokens_saved"`
	Requests    int64       `json:"requests"`
	Sessions    int64       `json:"sessions"`
	CacheHits   int64       `json:"cache_hits"`
}

// History is the durable savings history returned by GET /stats-history
// (plan §17.2). Buckets is keyed by granularity (the requested rollup set)
// and each list is ordered by BucketStart ascending.
type History struct {
	Buckets map[Granularity][]Bucket `json:"buckets"`
}

// Saving is the per-turn proxy savings row as written by the proxy
// (ctx_proxy_savings, migration 057). It is the bucket writer's input.
// Mirrors ctxstore.ProxySaving without the ctxstore import so the pure Roll
// helper stays store-free and trivially testable.
type Saving struct {
	SavingID     string
	SessionID    string
	Turn         int64
	Upstream     string
	Mode         string
	TokensBefore int64
	TokensAfter  int64
	CreatedAt    int64 // unix seconds
}

// bucketKey is the deterministic identity of a stats bucket: granularity +
// aligned window start. INSERT OR REPLACE on this id makes repeated writes
// (per-request fold + periodic re-derivation) idempotent — they merge, never
// double-count.
func bucketKey(g Granularity, start int64) string {
	return fmt.Sprintf("%s-%d", g, start)
}

// windowSeconds returns the number of seconds in one window of granularity g.
func windowSeconds(g Granularity) int64 {
	switch g {
	case Hourly:
		return 3600
	case Daily:
		return 86400
	case Weekly:
		return 7 * 86400
	case Monthly:
		return 30 * 86400 // approx; aligned via startOfMonth
	default:
		return 0
	}
}

// windowStart returns the aligned window start (unix seconds) containing t.
// Monthly aligns to the first of the UTC month; the others truncate to the
// window size from the Unix epoch.
func windowStart(g Granularity, t time.Time) int64 {
	switch g {
	case Monthly:
		y, m, _ := t.UTC().Date()
		return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).Unix()
	case Hourly, Daily, Weekly:
		return t.Unix() / windowSeconds(g) * windowSeconds(g)
	default:
		return 0
	}
}

// Roll computes the per-granularity bucket rows for a single per-turn saving.
// It is pure: no store, no clock. TokensSaved is the compression delta
// (TokensBefore - TokensAfter, clamped >= 0). Requests counts one turn.
// CacheHits contributes to the bucket only for a cache-mode saving (the
// proxy records a cache hit only in cache mode). Sessions is 1 per saving —
// the periodic rollup (Recompute) re-derives the distinct-session count
// across all savings, so per-saving Session=1 is a safe lower bound that
// Recompute overwrites with the exact count.
func Roll(s *Saving) (buckets []*ctxstore.StatsBucket) {
	if s == nil {
		return nil
	}
	t := time.Unix(s.CreatedAt, 0)
	saved := s.TokensBefore - s.TokensAfter
	if saved < 0 {
		saved = 0
	}
	cacheHit := int64(0)
	if s.Mode == "cache" {
		cacheHit = 1
	}
	now := t.Unix()
	for _, g := range DefaultGranularities {
		start := windowStart(g, t)
		buckets = append(buckets, &ctxstore.StatsBucket{
			BucketID:    bucketKey(g, start),
			Granularity: string(g),
			BucketStart: start,
			TokensSaved: saved,
			Requests:    1,
			Sessions:    1,
			CacheHits:   cacheHit,
			CreatedAt:   now,
		})
	}
	return buckets
}

// WriteSavings folds a batch of per-turn savings into ctx_stats_history.
// For each saving it computes the per-granularity buckets (Roll) and merges
// them into the existing bucket rows (if any) before PutStatsBucket. This is
// the per-request write path: cheap, idempotent on BucketID. A nil store or a
// missing table is a no-op (ADR-0016). Returns the number of bucket rows
// written.
func WriteSavings(s *ctxstore.Store, savings []*Saving) int {
	if s == nil || len(savings) == 0 {
		return 0
	}
	written := 0
	// Accumulate per bucket-id so multiple savings landing in the same window
	// merge into a single upsert rather than N upserts.
	merged := make(map[string]*ctxstore.StatsBucket)
	for _, sv := range savings {
		for _, b := range Roll(sv) {
			if existing, ok := merged[b.BucketID]; ok {
				existing.TokensSaved += b.TokensSaved
				existing.Requests += b.Requests
				existing.CacheHits += b.CacheHits
				// Sessions: keep the higher of the two (lower bound; Recompute
				// corrects it). Distinct-session tracking across a batch is not
				// attempted here.
				if b.Sessions > existing.Sessions {
					existing.Sessions = b.Sessions
				}
			} else {
				// Start from the persisted row so a re-fold does not reset
				// counters accumulated by earlier writes in the same window.
				if prev, err := s.GetStatsBucket(b.BucketID); err == nil && prev != nil {
					c := *prev
					c.TokensSaved += b.TokensSaved
					c.Requests += b.Requests
					c.CacheHits += b.CacheHits
					if b.Sessions > c.Sessions {
						c.Sessions = b.Sessions
					}
					merged[b.BucketID] = &c
				} else {
					merged[b.BucketID] = b
				}
			}
		}
	}
	for _, b := range merged {
		if err := s.PutStatsBucket(b); err == nil {
			written++
		}
	}
	return written
}

// Recompute re-derives every ctx_stats_history bucket from the full
// ctx_proxy_savings + ctx_proxy_sessions tables and rewrites them. This is
// the periodic rollup: it keeps the history current without a proxy restart
// and corrects the Sessions count to the exact distinct-session total per
// window. It is idempotent — PutStatsBucket is INSERT OR REPLACE on the
// deterministic BucketID, so re-running yields the same rows. A nil store or
// a missing table is a no-op (ADR-0016). Returns the number of bucket rows
// written.
//
// Because the ctx accessor only offers ListProxySavings(sessionID), this first
// lists all sessions, then fetches each session's savings. That is O(sessions)
// queries — fine for a background tick over a local loopback proxy.
func Recompute(s *ctxstore.Store) int {
	if s == nil {
		return 0
	}
	sessions, err := s.ListProxySessions()
	if err != nil {
		return 0
	}
	// Aggregate per (granularity, windowStart).
	type agg struct {
		tokens   int64
		requests int64
		hits     int64
		sessions map[string]struct{}
	}
	perWindow := make(map[string]*agg)

	for _, sess := range sessions {
		savings, err := s.ListProxySavings(sess.SessionID)
		if err != nil {
			continue
		}
		for _, sv := range savings {
			if sv == nil {
				continue
			}
			t := time.Unix(sv.CreatedAt, 0)
			saved := sv.TokensBefore - sv.TokensAfter
			if saved < 0 {
				saved = 0
			}
			hit := int64(0)
			if sv.Mode == "cache" {
				hit = 1
			}
			for _, g := range DefaultGranularities {
				start := windowStart(g, t)
				key := bucketKey(g, start)
				a, ok := perWindow[key]
				if !ok {
					a = &agg{sessions: make(map[string]struct{})}
					perWindow[key] = a
				}
				a.tokens += saved
				a.requests++
				a.hits += hit
				a.sessions[sv.SessionID] = struct{}{}
			}
		}
	}

	written := 0
	now := time.Now().Unix()
	for key, a := range perWindow {
		var g Granularity
		var start int64
		// Recover granularity + start from the deterministic key
		// "<granularity>-<start>".
		dash := -1
		for i := len(key) - 1; i >= 0; i-- {
			if key[i] == '-' {
				dash = i
				break
			}
		}
		if dash < 0 {
			continue
		}
		g = Granularity(key[:dash])
		fmt.Sscanf(key[dash+1:], "%d", &start)
		b := &ctxstore.StatsBucket{
			BucketID:    key,
			Granularity: string(g),
			BucketStart: start,
			TokensSaved: a.tokens,
			Requests:    a.requests,
			Sessions:    int64(len(a.sessions)),
			CacheHits:   a.hits,
			CreatedAt:   now,
		}
		if err := s.PutStatsBucket(b); err == nil {
			written++
		}
	}
	return written
}

// StartRollup launches a background goroutine that re-derives the
// ctx_stats_history buckets (Recompute) every tick until stop is closed. The
// stats rollup always runs; the per-session ctx_session_summaries rollup
// (SummarizeSessions) rides the same loop only when sessionSummary is true, so
// it is gated on the caller's config (cfg.Observability.SessionSummary) at this
// call site. It returns a stop function that closes stop and waits for the
// goroutine to exit (no leak). The first pass runs immediately so a fresh
// process backfills the history (and, when enabled, the session summaries)
// before the first tick. A non-positive tick falls back to a 5-minute default.
//
// The session-summary rollup is fail-open (ADR-0016) so a failed or disabled
// pass never disturbs the stats rollup it runs alongside.
func StartRollup(s *ctxstore.Store, tick time.Duration, sessionSummary bool, snapshotPath string, stop chan struct{}) (stopFn func()) {
	if s == nil {
		return func() {}
	}
	if tick <= 0 {
		tick = 5 * time.Minute
	}
	done := make(chan struct{})
	var once sync.Once
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		// Immediate first pass: stats history, per-session summaries when the
		// operator enabled the session-summary rollup, and the durable JSON
		// snapshot when a snapshot path is configured.
		pass(s, sessionSummary, snapshotPath)
		t := time.NewTicker(tick)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				pass(s, sessionSummary, snapshotPath)
			}
		}
	}()
	return func() {
		once.Do(func() { close(stop) })
		wg.Wait()
	}
}

// pass runs one rollup pass: recompute the stats-history buckets, then the
// optional per-session summaries and the optional durable JSON snapshot.
// Every step is fail-open (ADR-0016) — a fault in any step yields no new rows
// / no new file but never blocks the tick.
func pass(s *ctxstore.Store, sessionSummary bool, snapshotPath string) {
	_ = Recompute(s)
	if sessionSummary {
		// The rollup is fail-open (ADR-0016) — SummarizeSessions only returns
		// an error on a fault the operator should not be blocked on by a
		// stats tick — so the error is intentionally dropped.
		_, _ = SummarizeSessions(s, time.Now().Unix())
	}
	if snapshotPath != "" {
		// The snapshot mirrors ctx_stats_history to disk so it survives a
		// re-index (@step-13). Fail-open: a write fault yields no file.
		_ = WriteHistorySnapshot(s, snapshotPath, nil)
	}
}

// Snapshot returns the live session-metric snapshot (plan §17.2/§17.5) from
// ctx_proxy_sessions. Every counter is zero-valued on a missing table
// (ADR-0016). Mode is the latest session's mode (by EndedAt, then StartedAt),
// the seam for the T11 run-mode header.
func Snapshot(s *ctxstore.Store) (*Stats, error) {
	out := &Stats{}
	if s == nil {
		return out, nil
	}
	sessions, err := s.ListProxySessions()
	if err != nil {
		return out, nil
	}
	out.Sessions = len(sessions)
	var latest *ctxstore.ProxySession
	for _, sess := range sessions {
		if sess == nil {
			continue
		}
		out.Requests += sess.Requests
		out.CacheHits += sess.CacheHits
		out.CacheInvalidations += sess.CacheInvalidations
		saved := sess.TokensBefore - sess.TokensAfter
		if saved > 0 {
			out.TokensSaved += saved
		}
		if latest == nil || newer(sess, latest) {
			latest = sess
		}
	}
	if latest != nil {
		out.Mode = latest.Mode
	}
	return out, nil
}

// newer reports whether a is strictly newer than b by EndedAt then StartedAt.
func newer(a, b *ctxstore.ProxySession) bool {
	if a.EndedAt != b.EndedAt {
		return a.EndedAt > b.EndedAt
	}
	return a.StartedAt > b.StartedAt
}

// StatsHistory returns the durable savings history (plan §17.2) for the
// requested granularities, read from ctx_stats_history. A missing table yields
// an empty per-granularity list (ADR-0016). The returned History.Buckets map
// contains an entry for every requested granularity (possibly empty) so the
// JSON shape is stable.
func StatsHistory(s *ctxstore.Store, granularities []Granularity) (*History, error) {
	out := &History{Buckets: make(map[Granularity][]Bucket)}
	if len(granularities) == 0 {
		granularities = DefaultGranularities
	}
	for _, g := range granularities {
		buckets := []Bucket{}
		if s != nil {
			rows, err := s.ListStatsBuckets(string(g))
			if err == nil {
				for _, r := range rows {
					if r == nil {
						continue
					}
					buckets = append(buckets, Bucket{
						Granularity: g,
						BucketStart: r.BucketStart,
						TokensSaved: r.TokensSaved,
						Requests:    r.Requests,
						Sessions:    r.Sessions,
						CacheHits:   r.CacheHits,
					})
				}
			}
		}
		out.Buckets[g] = buckets
	}
	return out, nil
}
