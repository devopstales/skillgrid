package memory

import (
	"fmt"
	"time"
)

// trails.go — the session trail: the SaveEvent consumer that appends one
// entry per committed save to the session's trail (v1's session_events
// pattern, ported — the append-only, per-session, 0-based sequence-ordered
// log).
//
// The bus is the memory→wiki boundary; the trail is the memory-internal
// audit side of the same event. NewService wires subscribeTrail as the
// first subscriber, so every committed save (the step-10 Publish in
// pipeline.go) appends exactly one row to session_trails. A consumer that
// fails is logged by the bus and never fails the save (spec 01-memory/001
// §5 — the same failure isolation as the wiki consumers). The session row
// is written by the save's own transaction (the gate's CheckSession
// auto-register); this consumer re-checks and registers it only if the
// store's write path diverged (defensive FK guard, never the primary path).

// trailSeam is the store's write surface the trail append goes through.
// The SQLite store's RawExec is the single implementation (the same seam
// pattern the pipeline uses for its Probe/ProbeInt test reads; a store
// without it skips the append rather than failing the save — the append
// is a side effect of the bus, not a pipeline step, so an unwired dialect
// degrades the audit, not the save).
type trailSeam interface {
	RawExec(query string, args ...any) error
}

// subscribeTrail registers the trail-append consumer on the bus (the
// Service constructor wires it first, before any external consumer — the
// trail is the memory's own audit, always on).
func (s *Service) subscribeTrail() {
	if s == nil || s.bus == nil {
		return
	}
	s.bus.Subscribe(func(ev SaveEvent) { s.appendTrail(ev) })
}

// appendTrail appends one trail entry for a committed save. A missing
// session row is registered here before the append so the trail's FK
// resolves; a registration failure does not fail the save (the session
// row is written by the pipeline's own save transaction — the gate's
// CheckSession auto-register; a missing row here means the store's write
// path diverged, and the audit entry is less important than the committed
// fact).
func (s *Service) appendTrail(ev SaveEvent) {
	if s.st == nil || ev.SessionID == "" {
		return
	}
	seam, ok := s.st.(trailSeam)
	if !ok {
		return // no write seam: the append is skipped, not fatal
	}
	if err := s.ensureSession(seam, ev.SessionID, ev.Project); err != nil {
		return
	}
	// The append is a side effect of the bus: a failure here is recovered
	// by the bus's panic handler (an error return is not — so the error
	// panics) and never fails the save.
	if err := s.appendTrailRow(seam, ev); err != nil {
		panic(fmt.Sprintf("memory: trail append (session_id=%s): %v", ev.SessionID, err))
	}
}

// ensureSession registers the session row if missing (shared ownership,
// the resolved project — the same auto-register the pipeline's step-2
// CheckSession does inside the save transaction; here it runs post-commit
// as the trail's FK guard, so the two never conflict: the save's own row
// wins via ON CONFLICT DO NOTHING).
func (s *Service) ensureSession(seam trailSeam, sessionID, project string) error {
	return seam.RawExec(`
		INSERT INTO sessions (id, project, directory, ownership_mode, started_at, created_at)
		VALUES (?, ?, '', 'shared', datetime('now'), datetime('now'))
		ON CONFLICT(id) DO NOTHING`,
		sessionID, nullableProject(project))
}

// appendTrailRow inserts one trail entry at the next per-session sequence
// (the v1 COALESCE(MAX(sequence),-1)+1 guard — the sequence is the
// per-session audit order, monotonically increasing).
func (s *Service) appendTrailRow(seam trailSeam, ev SaveEvent) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	return seam.RawExec(`
		INSERT INTO session_trails (session_id, project, sequence, action_type, sync_id, timestamp)
		VALUES (?, ?, (SELECT COALESCE(MAX(sequence), -1) + 1 FROM session_trails WHERE session_id = ?), 'save', ?, ?)`,
		ev.SessionID, nullableProject(ev.Project), ev.SessionID, ev.SyncID, now)
}

// nullableProject maps "" to NULL (the sessions.project /
// session_trails.project columns are nullable; an empty resolved project
// stores NULL, matching the gate's nullableString convention).
func nullableProject(p string) any {
	if p == "" {
		return nil
	}
	return p
}
