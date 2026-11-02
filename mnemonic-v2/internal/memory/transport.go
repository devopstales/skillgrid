package memory

// Sync transport (spec 01-memory/006 §2-3, §7 — Task 16). The
// MutationTransport seam is the boundary between the sync engine (this
// package's Push/Pull cycle) and the remote: the PG transport (prod)
// speaks to the team's self-hosted Postgres, one schema per project; the
// in-memory fake (tests) records batches instead. The chunk codec
// (chunkcodec.go) is the content-addressed export/import format.
//
// Project isolation (spec 006 §2) is the core constraint: every query is
// qualified schema_<sanitized>.<table> — no search_path reliance, no
// cross-project JOIN. A connection that cannot resolve the project schema
// fails at EnsureSchema; there is no public fallback.
//
// The push/pull cycle (Push/Pull below) is what the autosync manager
// (Task 17) drives per cycle and what the CLI sync push/pull (Task 14's
// subcommands, now real) runs once: push reads the local outbox
// (ListPendingMutations) and acks what the transport accepted; pull reads
// the peer since the local cursor and applies each mutation through
// SyncService.ApplyPulled (INV-4 — the same through-store path).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"

	_ "github.com/lib/pq" // ADR-0041: the PG driver of record for the PG transport.
)

// MaxBatch is the hard per-push batch cap (spec 006 §3: ≤100 mutations per
// transaction; the config's PushLimit may lower it, never raise it).
const MaxBatch = 100

// DefaultMaxPushBytes is the per-push batch payload ceiling (8 MiB — the
// config's MaxPushBytes default).
const DefaultMaxPushBytes = 8 * 1024 * 1024

// ErrBatchTooLarge is the batch-cap refusal (a push batch over MaxBatch is
// refused before it reaches the transport — the autosync pages it).
var ErrBatchTooLarge = errors.New("sync: push batch exceeds the cap")

// ErrPaused marks a push refused because the project is paused
// (project_controls.paused=1 — the write was skipped, the cursor is
// untouched; the CLI reports "sync-paused").
var ErrPaused = errors.New("sync-paused")

// ErrPushTooLarge marks a push refused because the batch's total payload
// exceeds the config's MaxPushBytes (the brief's ≤8MiB half of the
// "≤100/batch, ≤8MiB" contract).
var ErrPushTooLarge = errors.New("sync: push batch exceeds the byte limit")

// MutationTransport is the seam between the sync engine and the remote
// (spec 006 §5). Adapters: PGTransport (prod — the team's PG) and the
// in-memory fake (tests). The batch cap (≤MaxBatch) and the payload
// ceiling (≤MaxPushBytes) are enforced at the transport boundary — a
// batch over the cap is refused before it reaches the remote.
type MutationTransport interface {
	// PushMutations writes a batch (≤MaxBatch) in ONE PG transaction and
	// returns the acked seq (the highest local seq in the batch). A paused
	// project (project_controls.paused=1) writes nothing and returns the
	// current cursor + ErrPaused.
	PushMutations(ctx context.Context, project string, batch []store.Mutation) (int64, error)
	// PullMutations returns the project schema's mutations with seq >
	// sinceSeq, in seq order (the pull cursor advances past them).
	PullMutations(ctx context.Context, project string, sinceSeq int64) ([]store.Mutation, error)
	// EnsureSchema resolves + creates the project's schema and the remote
	// table set (idempotent). A connection that cannot resolve the schema
	// fails here — no public fallback (spec 006 §2 fail-fast).
	EnsureSchema(ctx context.Context, project string) error
	Close() error
}

// TransportConfig is the transport wiring: the DSN (the PG server) plus the
// config-derived limits. The DSN is the config's sync-peer DSN
// (datasource.dsn / MNEMONIC_PG_DSN — the team's PG).
type TransportConfig struct {
	DSN          string
	PushLimit    int
	MaxPushBytes int
}

// newPGTransport is the transport-construction seam: production builds the
// PG transport; the CLI tests swap in a fake (the seam is the testable
// boundary — the CLI wiring is under test without a PG server).
var newPGTransport = func(cfg TransportConfig) (MutationTransport, error) {
	return NewPGTransport(cfg.DSN, cfg.PushLimit, cfg.MaxPushBytes)
}

// Push runs one push for the project's outbox batch: EnsureSchema first
// (fail-fast — an unresolvable project never writes, spec 006 §2), the
// batch cap is enforced at the transport boundary (over MaxBatch is
// refused before it reaches the remote), the transport writes the batch in
// one transaction, and the local outbox is acked at the returned seq (the
// outbox drains — the acked rows leave the pending set; the append-only
// journal is otherwise untouched). An empty batch is a valid no-op (the
// autosync's idle cycle). It returns the acked seq.
func Push(ctx context.Context, st store.Store, tr MutationTransport, project string, batch []store.Mutation) (int64, error) {
	if err := EnsureSchema(ctx, tr, project); err != nil {
		return 0, err
	}
	if len(batch) > MaxBatch {
		return 0, fmt.Errorf("%w: %d > %d", ErrBatchTooLarge, len(batch), MaxBatch)
	}
	ack, err := tr.PushMutations(ctx, project, batch)
	if err != nil {
		return 0, err
	}
	if ack > 0 {
		if err := ackMutations(ctx, st, ack); err != nil {
			return 0, err
		}
	}
	return ack, nil
}

// ackMutations marks the local outbox acked up to and including seq (the
// push's local half: the journal rows leave the pending set — the
// append-only journal is otherwise untouched).
func ackMutations(ctx context.Context, st store.Store, upToSeq int64) error {
	_ = ctx
	return applyPulledSeam(st).RawExec(
		`UPDATE sync_mutations SET acked_at = datetime('now') WHERE seq <= ? AND acked_at IS NULL`,
		upToSeq)
}

// EnsureSchema resolves + creates the project's remote schema (idempotent)
// — the fail-fast boundary (spec 006 §2: a connection that cannot resolve
// the schema fails here, no public fallback). The push/pull cycles call it
// first, so an unresolvable project never writes or reads.
func EnsureSchema(ctx context.Context, tr MutationTransport, project string) error {
	return tr.EnsureSchema(ctx, project)
}

// Pull reads the peer's mutations since sinceSeq (the transport is scoped
// to the project's schema — the isolation boundary). The caller applies
// them through SyncService.ApplyPulled (PullAndApply does that); the
// cursor advances per applied mutation.
func Pull(ctx context.Context, tr MutationTransport, project string, sinceSeq int64) ([]store.Mutation, error) {
	return tr.PullMutations(ctx, project, sinceSeq)
}

// PullAndApply is Pull + the through-store apply (the CLI pull / the
// autosync's pull cycle): EnsureSchema first (fail-fast), fetch since the
// cursor, apply each via SyncService.ApplyPulled (the cursor advances per
// mutation), replay the deferred queue, and return what was applied.
func PullAndApply(ctx context.Context, st store.Store, tr MutationTransport, project string, sinceSeq int64) (int, error) {
	if err := EnsureSchema(ctx, tr, project); err != nil {
		return 0, err
	}
	muts, err := Pull(ctx, tr, project, sinceSeq)
	if err != nil {
		return 0, err
	}
	svc := &SyncService{st: st}
	for _, m := range muts {
		if err := svc.ApplyPulled(ctx, m); err != nil {
			return 0, err
		}
	}
	svc.ReplayDeferred(ctx)
	return len(muts), nil
}

// ── PG transport (the team's self-hosted Postgres, ADR-0041) ──────────────

// pgTransport is the production MutationTransport: database/sql + lib/pq
// against the team's PG, one schema per project (spec 006 §2). Every query
// is qualified schema_<sanitized>.<table> — the schema is the isolation
// boundary, so no search_path reliance.
type pgTransport struct {
	db       *sql.DB
	pushLim  int
	maxPushB int
}

var _ MutationTransport = (*pgTransport)(nil)

// NewPGTransport builds the PG transport for dsn. pushLimit/maxPushB are
// the config's limits (defaults: MaxBatch / 8 MiB).
func NewPGTransport(dsn string, pushLimit, maxPushB int) (MutationTransport, error) {
	if dsn == "" {
		return nil, errors.New("sync: PG transport requires a DSN (datasource.dsn / MNEMONIC_PG_DSN)")
	}
	if pushLimit <= 0 {
		pushLimit = MaxBatch
	}
	if maxPushB <= 0 {
		maxPushB = DefaultMaxPushBytes
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sync: open postgres: %w", err)
	}
	return &pgTransport{db: db, pushLim: pushLimit, maxPushB: maxPushB}, nil
}

// pgSchemaName is the project's remote schema (project.Canonicalize — the
// same rule the PG store renders; it may contain '-', so it is always
// double-quoted).
func pgSchemaName(projectName string) string {
	return `"` + "schema_" + project.Canonicalize(projectName) + `"`
}

// qualified qualifies a bare table for the project schema.
func (p *pgTransport) qualified(project, table string) string {
	return pgSchemaName(project) + "." + table
}

// EnsureSchema resolves + creates the project schema and the remote table
// set (idempotent). A connection that cannot create the schema fails — no
// public fallback (spec 006 §2 fail-fast).
func (p *pgTransport) EnsureSchema(ctx context.Context, project string) error {
	schema := pgSchemaName(project)
	if _, err := p.db.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS `+schema); err != nil {
		return fmt.Errorf("sync: ensure schema %s: %w", schema, err)
	}
	// The remote table set (spec 006 §3 — the replicated mutation domain;
	// the entity tables are Task 18's migrations_pg render, a different
	// artifact with the same table names): the mutations log + the
	// sync_state cursor + the per-project pause flag.
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS ` + p.qualified(project, "mutations") + ` (
			seq BIGINT NOT NULL,
			target_key TEXT NOT NULL,
			entity TEXT NOT NULL,
			entity_key TEXT NOT NULL,
			op TEXT NOT NULL,
			payload TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT 'local',
			project TEXT,
			occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (seq)
		)`,
		`CREATE TABLE IF NOT EXISTS ` + p.qualified(project, "sync_state") + ` (
			target_key TEXT PRIMARY KEY,
			lifecycle TEXT NOT NULL DEFAULT 'idle',
			last_enqueued_seq BIGINT NOT NULL DEFAULT 0,
			last_acked_seq BIGINT NOT NULL DEFAULT 0,
			last_pulled_seq BIGINT NOT NULL DEFAULT 0,
			backoff_until TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS ` + p.qualified(project, "project_controls") + ` (
			project TEXT PRIMARY KEY,
			paused BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	} {
		if _, err := p.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sync: remote table: %w", err)
		}
	}
	return nil
}

// PushMutations writes the batch in ONE transaction: insert the mutations,
// then advance the remote's sync_state.last_acked_seq to the batch's
// highest local seq. A paused project (project_controls.paused=1) writes
// nothing and returns ErrPaused (the current cursor). The batch cap
// (p.pushLim, the config's lowered limit — never over MaxBatch) and the
// payload ceiling (p.maxPushB, the config's MaxPushBytes) are enforced
// here at the seam (the brief's "≤100/batch, ≤8MiB" contract).
func (p *pgTransport) PushMutations(ctx context.Context, project string, batch []store.Mutation) (int64, error) {
	if len(batch) > p.pushLim {
		return 0, fmt.Errorf("%w: %d > %d", ErrBatchTooLarge, len(batch), p.pushLim)
	}
	if size := batchBytes(batch); size > p.maxPushB {
		return 0, fmt.Errorf("%w: %d > %d", ErrPushTooLarge, size, p.maxPushB)
	}
	if len(batch) == 0 {
		return 0, nil
	}
	// paused check (read the flag; a fresh schema without the row is not
	// paused — EnsureSchema creates the table, so the query resolves; an
	// empty table means not paused).
	var paused int
	if err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM `+p.qualified(project, "project_controls")+` WHERE paused = TRUE`).
		Scan(&paused); err != nil {
		return 0, fmt.Errorf("sync: paused check: %w", err)
	}
	if paused > 0 {
		var cursor int64
		if cerr := p.db.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(last_acked_seq), 0) FROM `+p.qualified(project, "sync_state")).
			Scan(&cursor); cerr != nil {
			return 0, fmt.Errorf("sync: paused cursor: %w", cerr)
		}
		return cursor, ErrPaused
	}
	var maxSeq int64
	for _, m := range batch {
		if m.Seq > maxSeq {
			maxSeq = m.Seq
		}
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, m := range batch {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO `+p.qualified(project, "mutations")+`
				(seq, target_key, entity, entity_key, op, payload, source, project, occurred_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
				ON CONFLICT (seq) DO NOTHING`,
			m.Seq, m.TargetKey, m.Entity, m.EntityKey, m.Op, m.Payload, m.Source, nullableProj(m.Project)); err != nil {
			return 0, fmt.Errorf("sync: push mutation seq %d: %w", m.Seq, err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO `+p.qualified(project, "sync_state")+`
			(target_key, lifecycle, last_acked_seq) VALUES ('local', 'idle', $1)
			ON CONFLICT (target_key) DO UPDATE SET
			  last_acked_seq = GREATEST(last_acked_seq, excluded.last_acked_seq),
			  updated_at = now()`, maxSeq); err != nil {
		return 0, fmt.Errorf("sync: ack cursor: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return maxSeq, nil
}

// PullMutations reads the project schema's mutations with seq > sinceSeq,
// in seq order (the pull cursor advances past them; the local apply is the
// caller's job via SyncService.ApplyPulled).
func (p *pgTransport) PullMutations(ctx context.Context, project string, sinceSeq int64) ([]store.Mutation, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT seq, target_key, entity, entity_key, op, payload, COALESCE(source,'local'), COALESCE(project,''), occurred_at
		 FROM `+p.qualified(project, "mutations")+`
		 WHERE seq > $1 ORDER BY seq ASC`, sinceSeq)
	if err != nil {
		return nil, fmt.Errorf("sync: pull: %w", err)
	}
	defer rows.Close()
	var out []store.Mutation
	for rows.Next() {
		var m store.Mutation
		var occ any
		if err := rows.Scan(&m.Seq, &m.TargetKey, &m.Entity, &m.EntityKey, &m.Op, &m.Payload, &m.Source, &m.Project, &occ); err != nil {
			return nil, fmt.Errorf("sync: pull scan: %w", err)
		}
		// lib/pq decodes TIMESTAMPTZ to time.Time; the fake/SQLite path
		// carries a string. Handle both (the store's timestamp format).
		switch v := occ.(type) {
		case time.Time:
			m.OccurredAt = v.UTC().Format("2006-01-02 15:04:05")
		case string:
			m.OccurredAt = v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Close closes the PG handle.
func (p *pgTransport) Close() error { return p.db.Close() }

// batchBytes is the total payload size of a batch (the size gate).
func batchBytes(batch []store.Mutation) int {
	n := 0
	for _, m := range batch {
		n += len(m.Payload)
	}
	return n
}

// nullableProj renders the project column (nil when empty — the column is
// nullable on the remote).
func nullableProj(project string) any {
	if project == "" {
		return nil
	}
	return project
}
