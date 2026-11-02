package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/syncid"
	_ "github.com/lib/pq" // ADR-0041: the PG driver of record for the PG render.
)

//go:embed migrations_pg/*.sql
var pgMigrationsFS embed.FS

// pgStore implements Store on Postgres (one schema per project, spec
// 001-storage §2 S-3; READ COMMITTED — the PG default, spec 001-storage §5).
// The Go driver surface is database/sql; lib/pq is the only driver import
// (ADR-0041 — a future swap is a one-file change).
type pgStore struct {
	db      *sql.DB
	schema  string
	txnHook func()
	syncOn  bool
}

var _ Store = (*pgStore)(nil)
var _ SyncSeam = (*pgStore)(nil)

// RawExec runs a raw statement against the database (the same test seam
// shape as the SQLite store).
func (s *pgStore) RawExec(query string, args ...any) error {
	_, err := s.db.Exec(query, args...)
	return err
}

// RawExecRowsAffected runs a raw statement and returns the rows it
// affected (the SQLite store's seam shape — the autosync lease takeover
// uses it; the PG dialect is gated out of autosync until the lease leg is
// ported, so this path is parity for the seam, not exercised yet).
func (s *pgStore) RawExecRowsAffected(query string, args ...any) (int64, error) {
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RawQuery runs a raw SELECT and returns every row (the SQLite store's
// RawQuery seam shape, so the memory package's dialect-agnostic fixtures
// work unchanged).
func (s *pgStore) RawQuery(query string, args ...any) ([][]any, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out [][]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		out = append(out, vals)
	}
	return out, rows.Err()
}

// Probe runs a raw query and reports whether any row was returned (the
// SQLite store's Probe seam shape).
func (s *pgStore) Probe(query string, args ...any) (bool, error) {
	var found bool
	err := s.db.QueryRow(query, args...).Scan(&found)
	return found, err
}

// ProbeInt runs a raw query that must return a single integer (the SQLite
// store's ProbeInt seam shape).
func (s *pgStore) ProbeInt(query string, args ...any) (int, error) {
	var n int
	err := s.db.QueryRow(query, args...).Scan(&n)
	return n, err
}

// openPG opens the Postgres database for dsn, resolves the project schema
// (fail fast — no public fallback), and runs the migrations.
func openPG(ctx context.Context, project, dsn string) (*pgStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("store.Open: Postgres dialect requires a DSN")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	// Fail fast: the server must be reachable before any schema work.
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	schema := "schema_" + pgCanonical(project)
	// The schema name may contain '-' (Canonicalize maps '/' → '-'), so it
	// is always double-quoted in DDL — a bare identifier would break.
	quoted := `"` + schema + `"`
	// A dedicated connection for the whole open (CREATE SCHEMA + migrate):
	// the search_path is session-level, so every statement on this
	// connection sees the project schema first.
	conn, err := db.Conn(ctx)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open postgres: conn: %w", err)
	}
	// public is included so extensions installed in public (vector, pg_trgm)
	// are resolvable; the project schema comes first so its objects win.
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`SET search_path = "%s", pg_catalog, public`, schema)); err != nil {
		conn.Close()
		_ = db.Close()
		return nil, fmt.Errorf("open postgres: search_path: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS `+quoted); err != nil {
		conn.Close()
		_ = db.Close()
		return nil, fmt.Errorf("open postgres: create schema %s: %w", schema, err)
	}
	st := &pgStore{db: db, schema: schema}
	// Extensions are installed in public (database-level, shared across
	// projects). IF NOT EXISTS makes this idempotent.
	if _, err := conn.ExecContext(ctx, `SET search_path = public`); err != nil {
		conn.Close()
		_ = db.Close()
		return nil, fmt.Errorf("open postgres: search_path for extensions: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		conn.Close()
		_ = db.Close()
		return nil, fmt.Errorf("open postgres: create extension vector: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS pg_trgm`); err != nil {
		conn.Close()
		_ = db.Close()
		return nil, fmt.Errorf("open postgres: create extension pg_trgm: %w", err)
	}
	// Restore the project-schema search_path for the migration set.
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`SET search_path = "%s", pg_catalog, public`, schema)); err != nil {
		conn.Close()
		_ = db.Close()
		return nil, fmt.Errorf("open postgres: search_path for migrate: %w", err)
	}
	if err := st.migrateConn(conn); err != nil {
		conn.Close()
		_ = db.Close()
		return nil, fmt.Errorf("store.Open: migrate: %w", err)
	}
	conn.Close()
	return st, nil
}

// pgCanonical is project.Canonicalize inlined (the store must not import
// core/project — the dependency direction is the other way; the rule is
// mirrored here and covered by the project package's own tests).
func pgCanonical(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "/", "-")
	return strings.TrimSuffix(strings.TrimLeft(name, "/"), "-")
}

// qualified qualifies a bare table name for the store's schema (the schema
// is double-quoted — it may contain '-' from Canonicalize).
func (s *pgStore) qualified(table string) string { return `"` + s.schema + `"` + "." + table }

// ── Migrations ──────────────────────────────────────────────────────────────

// migrate runs every unapplied embedded PG migration, statement by statement,
// recording each in the schema's index_meta (the SQLite runner's pattern; the
// DOLLAR-quoted trigger/function bodies need a different splitter).
func (s *pgStore) migrate() error {
	// A dedicated connection: the search_path is session-level, so the
	// runner pins one physical connection for the whole migration set.
	conn, err := s.db.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("migrate conn: %w", err)
	}
	defer conn.Close()
	if err := s.migrateConn(conn); err != nil {
		return err
	}
	return nil
}

// migrateConn runs the migration set on a dedicated connection whose
// search_path already points at the project schema (the openPG / Migrate
// entry points each pin their own connection).
func (s *pgStore) migrateConn(conn *sql.Conn) error {
	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS index_meta (key TEXT PRIMARY KEY, schema_version INTEGER NOT NULL DEFAULT 0)`); err != nil {
		return fmt.Errorf("create index_meta: %w", err)
	}
	entries, err := pgMigrationsFS.ReadDir("migrations_pg")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		var done bool
		if err := conn.QueryRowContext(context.Background(), `SELECT COUNT(*) > 0 FROM index_meta WHERE key = $1`, "migration:"+name).Scan(&done); err != nil {
			return fmt.Errorf("check %s: %w", name, err)
		}
		if done {
			continue
		}
		b, err := pgMigrationsFS.ReadFile("migrations_pg/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		for _, stmt := range splitPGStatements(string(b)) {
			if stmt == "" {
				continue
			}
			if _, err := conn.ExecContext(context.Background(), stmt); err != nil {
				return fmt.Errorf("exec %s: %w", name, err)
			}
		}
		if _, err := conn.ExecContext(context.Background(), `INSERT INTO index_meta (key, schema_version) VALUES ($1, 1)`, "migration:"+name); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
	}
	if _, err := conn.ExecContext(context.Background(), `
		INSERT INTO index_meta (key, schema_version) VALUES ('schema_version', $1)
		ON CONFLICT(key) DO UPDATE SET schema_version = excluded.schema_version`,
		len(names),
	); err != nil {
		return fmt.Errorf("record schema_version: %w", err)
	}
	return nil
}

// splitPGStatements splits a PG migration script into statements. Semicolons
// terminate statements EXCEPT inside dollar-quoted bodies (the plpgsql
// trigger functions, which run from $body$ to the matching $body$).
// Single/double-quoted strings and -- line comments are skipped.
func splitPGStatements(s string) []string {
	var (
		out   []string
		cur   strings.Builder
		inSQ  bool
		inDQ  bool
		inDol bool
		dol   string // the open dollar-quote tag, e.g. "$$" or "$body$"
	)
	flush := func() {
		out = append(out, strings.TrimSpace(cur.String()))
		cur.Reset()
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inDol:
			if c == '$' && strings.HasPrefix(s[i:], dol) {
				cur.WriteString(dol)
				i += len(dol) - 1
				inDol = false
			} else {
				cur.WriteByte(c)
			}
		case inSQ:
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					cur.WriteString("''")
					i++
				} else {
					inSQ = false
					cur.WriteByte(c)
				}
			} else {
				cur.WriteByte(c)
			}
		case inDQ:
			if c == '"' {
				inDQ = false
			}
			cur.WriteByte(c)
		case c == '\'':
			inSQ = true
			cur.WriteByte(c)
		case c == '"':
			inDQ = true
			cur.WriteByte(c)
		case c == '$':
			// dollar-quote opener: $$ or $tag$
			if i+1 < len(s) && s[i+1] == '$' {
				// $$ (empty tag)
				dol = "$$"
				cur.WriteString("$$")
				i += 1
				inDol = true
				continue
			}
			if i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z' {
				// $tag$
				if j := strings.IndexByte(s[i+2:], '$'); j >= 0 && j < 63 {
					dol = s[i : i+2+j+1]
					cur.WriteString(dol)
					i += len(dol) - 1
					inDol = true
					continue
				}
			}
			cur.WriteByte(c)
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			// line comment: copy through the newline
			j := i
			for j < len(s) && s[j] != '\n' {
				j++
			}
			if j < len(s) {
				cur.WriteString(s[i:j])
				i = j - 1
			} else {
				cur.WriteString(s[i:])
				i = j
			}
		case c == ';':
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}

// ── helpers ─────────────────────────────────────────────────────────────────

// scanObservationPG scans an observations row (26 columns, same order as the
// SQLite leg's scanObservationFields) into an Observation.
func scanObservationPG(row interface {
	Scan(dest ...any) error
}, withRank bool) (Observation, error) {
	var (
		o           Observation
		toolName    sql.NullString
		project     sql.NullString
		topicKey    sql.NullString
		normalized  sql.NullString
		lastSeenAt  sql.NullTime
		reviewAfter sql.NullTime
		superseded  sql.NullString
		expiresAt   sql.NullTime
		provJSON    string
		embedding   []byte
		embedModel  sql.NullString
		deletedAt   sql.NullTime
		rank        sql.NullFloat64 // trailing rank column (retrieval queries)
		createdAt   time.Time
		updatedAt   time.Time
	)
	dest := []any{
		&o.ID, &o.SyncID, &o.SessionID, &o.Type, &o.Title, &o.Content,
		&toolName, &project, &o.Scope, &topicKey, &normalized,
		&o.RevisionCount, &o.DuplicateCount, &lastSeenAt, &o.Pinned,
		&reviewAfter, &superseded, &expiresAt, &embedding, &embedModel,
		&o.Importance, &o.RetrievalUsage, &provJSON, &createdAt, &updatedAt,
		&deletedAt,
	}
	if withRank {
		dest = append(dest, &rank)
	}
	if err := row.Scan(dest...); err != nil {
		return o, err
	}
	o.ToolName = stringPtr(toolName)
	o.Project = stringPtr(project)
	o.TopicKey = stringPtr(topicKey)
	o.NormalizedHash = normalized.String
	o.LastSeenAt = timePtr(lastSeenAt)
	o.ReviewAfter = timePtr(reviewAfter)
	o.SupersededBySyncID = stringPtr(superseded)
	o.ExpiresAt = timePtr(expiresAt)
	o.Embedding = embedding
	o.EmbeddingModel = embedModel.String
	o.DeletedAt = timePtr(deletedAt)
	o.CreatedAt = stamp(createdAt)
	o.UpdatedAt = stamp(updatedAt)
	_ = json.Unmarshal([]byte(provJSON), &o.Provenance)
	return o, nil
}

// timePtr renders a nullable TIMESTAMPTZ into the store's timestamp string.
func timePtr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.UTC().Format("2006-01-02 15:04:05")
	return &s
}

// stamp renders a TIMESTAMPTZ (always valid in the schema — NOT NULL) into
// the store's timestamp string.
func stamp(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05") }

// obsSelectColumnsPG is the observation column list for PG retrieval queries
// (same 26 columns, same order, as the SQLite leg's obsSelectColumns).
func (s *pgStore) obsSelectColumnsPG() string {
	return `o.id, COALESCE(o.sync_id,'') AS sync_id, o.session_id, o.type, o.title, o.content,
	o.tool_name, o.project, o.scope, o.topic_key, o.normalized_hash,
	o.revision_count, o.duplicate_count, o.last_seen_at, o.pinned,
	o.review_after, o.superseded_by_sync_id, o.expires_at, o.embedding,
	o.embedding_model, o.importance, o.retrieval_usage, o.provenance,
	o.created_at, o.updated_at, o.deleted_at`
}

// searchRankBoostPG is the composite-rank boost factor (the SQLite leg's
// searchRankBoostSQL rendered for PG: age in days from
// COALESCE(last_seen_at, updated_at, created_at), recency half-life 30 days;
// the boost weights are the spec-pinned literals).
func (s *pgStore) searchRankBoostPG() string {
	return `(1.0
	+ 0.10 * CASE WHEN o.pinned THEN 1.0 ELSE 0.0 END
	+ 0.06 / (1.0 + ABS(EXTRACT(EPOCH FROM (now() - COALESCE(o.last_seen_at, o.updated_at, o.created_at))) / 86400.0) / 30.0)
	+ 0.04 * (CAST(o.revision_count AS DOUBLE PRECISION) + CAST(o.duplicate_count AS DOUBLE PRECISION)) / (CAST(o.revision_count AS DOUBLE PRECISION) + CAST(o.duplicate_count AS DOUBLE PRECISION) + 4.0))`
}

// pgNow renders the store's shared timestamp format (nowStamp, the SQLite
// leg's) as a UTC literal for PG TIMESTAMPTZ columns.
func pgNow() string { return nowStamp() }

// ── observations ────────────────────────────────────────────────────────────

// AddObservation persists a new observation (the same Go orchestration as
// the SQLite leg; the dialect difference is the SQL).
func (s *pgStore) AddObservation(ctx context.Context, p AddObservationParams) (AddObservationResult, error) {
	now := pgNow()
	syncID := syncid.NewSyncID("obs")
	prov := Provenance{SessionID: p.SessionID, CreatedAt: now}
	provJSON, _ := json.Marshal(prov)
	hash := hashNormalized(p.Content)

	var id int64
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AddObservationResult{}, err
	}
	defer tx.Rollback()

	if err := s.ensureSessionTxn(ctx, tx, p.SessionID, p.Project); err != nil {
		return AddObservationResult{}, err
	}
	project := s.resolveProjectTxn(ctx, tx, p.SessionID, p.Project)

	if err := tx.QueryRowContext(ctx, `
		INSERT INTO `+s.qualified("observations")+` (
			sync_id, session_id, type, title, content, tool_name, project, scope,
			topic_key, normalized_hash,
			last_seen_at, importance, retrieval_usage, provenance, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), 0.5, 0, $11, $12, $13)
		RETURNING id`,
		syncID, p.SessionID, p.Type, p.Title, p.Content,
		nullableString(p.ToolName), nullableString(project),
		scopeOrDefault(p.Scope), nullableString(p.TopicKey), hash,
		provJSON, now, now,
	).Scan(&id); err != nil {
		return AddObservationResult{}, err
	}
	// ledger row (replay-safe save idempotency)
	if p.OperationID != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO `+s.qualified("observation_save_operations")+` (operation_id, fingerprint, observation_id)
			VALUES ($1, $2, $3)`,
			p.OperationID, "v1:"+hash, id,
		); err != nil {
			return AddObservationResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return AddObservationResult{}, err
	}
	return AddObservationResult{ID: id, SyncID: syncID}, nil
}

// AddPulledObservation inserts a pulled observation with the caller's
// (source's) sync_id — the sync apply seam (the SQLite leg's AddPulledObservation
// pattern; INV-4: no special remote path).
func (s *pgStore) AddPulledObservation(ctx context.Context, p AddObservationParams, syncID string) (AddObservationResult, error) {
	now := pgNow()
	prov := Provenance{SessionID: p.SessionID, CreatedAt: now}
	provJSON, _ := json.Marshal(prov)
	hash := hashNormalized(p.Content)

	var id int64
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AddObservationResult{}, err
	}
	defer tx.Rollback()

	if err := s.ensureSessionTxn(ctx, tx, p.SessionID, p.Project); err != nil {
		return AddObservationResult{}, err
	}
	project := s.resolveProjectTxn(ctx, tx, p.SessionID, p.Project)

	if err := tx.QueryRowContext(ctx, `
		INSERT INTO `+s.qualified("observations")+` (
			sync_id, session_id, type, title, content, tool_name, project, scope,
			topic_key, normalized_hash,
			last_seen_at, importance, retrieval_usage, provenance, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), 0.5, 0, $11, $12, $13)
		RETURNING id`,
		syncID, p.SessionID, p.Type, p.Title, p.Content,
		nullableString(p.ToolName), nullableString(project),
		scopeOrDefault(p.Scope), nullableString(p.TopicKey), hash,
		provJSON, now, now,
	).Scan(&id); err != nil {
		return AddObservationResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return AddObservationResult{}, err
	}
	return AddObservationResult{ID: id, SyncID: syncID}, nil
}

// GetObservation fetches an observation by local id.
func (s *pgStore) GetObservation(ctx context.Context, id int64, includeHistory bool) (Observation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+s.obsSelectColumnsPG()+`
		FROM `+s.qualified("observations")+` o WHERE o.id = $1 AND o.deleted_at IS NULL`, id)
	return scanObservationPG(row, false)
}

// UpdateObservationForProject updates the mutable fields of an observation,
// scoped to expectedProject.
func (s *pgStore) UpdateObservationForProject(ctx context.Context, id int64, expectedProject string, p UpdateParams) error {
	var existing Observation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
		SELECT `+s.obsSelectColumnsPG()+`
		FROM `+s.qualified("observations")+` o WHERE o.id = $1 AND o.deleted_at IS NULL`, id)
	if existing, err = scanObservationPG(row, false); err != nil {
		return err
	}
	// project guard
	proj := deref(existing.Project)
	if proj != "" && strings.TrimSpace(expectedProject) != "" && proj != strings.TrimSpace(expectedProject) {
		return fmt.Errorf("store: observation %d belongs to project %q, not %q", id, proj, expectedProject)
	}

	typ, title, content := existing.Type, existing.Title, existing.Content
	topicKey := deref(existing.TopicKey)
	if p.Type != nil {
		typ = *p.Type
	}
	if p.Title != nil {
		title = *p.Title
	}
	if p.Content != nil {
		content = *p.Content
	}
	if p.TopicKey != nil {
		topicKey = *p.TopicKey
	}
	pinned := existing.Pinned
	if p.Pinned != nil {
		pinned = *p.Pinned
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE `+s.qualified("observations")+`
		SET type = $1, title = $2, content = $3, topic_key = $4, pinned = $5,
		    normalized_hash = $6, updated_at = now()
		WHERE id = $7`,
		typ, title, content, nullableString(topicKey), pinned, hashNormalized(content), id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteObservationForProject soft-deletes (or hard-deletes when hard=true)
// an observation, scoped to expectedProject.
func (s *pgStore) DeleteObservationForProject(ctx context.Context, id int64, expectedProject string, hard bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// verify project guard
	var proj string
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(project,'') FROM `+s.qualified("observations")+` WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&proj)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %d", ErrObservationNotFound, id)
	}
	if err != nil {
		return err
	}
	if proj != "" && strings.TrimSpace(expectedProject) != "" && proj != strings.TrimSpace(expectedProject) {
		return fmt.Errorf("store: observation %d belongs to project %q, not %q", id, proj, expectedProject)
	}

	if hard {
		// the tsv trigger keeps the index in step; the ledger FK is SET NULL.
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+s.qualified("observations")+` WHERE id = $1`, id); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE `+s.qualified("observations")+`
			SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ConsolidateObservations executes a confirmed consolidation (the SQLite leg's
// ConsolidateObservations semantics: sources soft-deleted with
// superseded_by_sync_id → the survivor's sync_id, journal in the same txn).
func (s *pgStore) ConsolidateObservations(ctx context.Context, p ConsolidateParams) (ConsolidateResult, error) {
	res := ConsolidateResult{SurvivorID: p.SurvivorID, DeletedIDs: p.SourceIDs}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ConsolidateResult{}, err
	}
	defer tx.Rollback()
	if err := s.consolidateTxnPG(ctx, tx, p); err != nil {
		return ConsolidateResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ConsolidateResult{}, err
	}
	if s.txnHook != nil {
		s.txnHook()
	}
	return res, nil
}

func (s *pgStore) consolidateTxnPG(ctx context.Context, tx *sql.Tx, p ConsolidateParams) error {
	// the survivor's journal payload (read inside the txn, same shape as the
	// save gate's enqueueUpsert payload).
	var sess, scope, typ, title, content string
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(session_id,''), COALESCE(scope,''), COALESCE(type,''), COALESCE(title,''), COALESCE(content,'')
		 FROM `+s.qualified("observations")+` WHERE id = $1`, p.SurvivorID).
		Scan(&sess, &scope, &typ, &title, &content); err != nil {
		return fmt.Errorf("store: consolidate survivor read: %w", err)
	}
	payload := `{"sync_id":"` + jsonEscape(p.SurvivorSyncID) +
		`","session_id":"` + jsonEscape(sess) +
		`","type":"` + jsonEscape(typ) +
		`","title":"` + jsonEscape(title) +
		`","content":"` + jsonEscape(content) +
		`","project":"` + jsonEscape(p.TargetProject) +
		`","scope":"` + jsonEscape(scope) + `"}`
	for _, id := range p.SourceIDs {
		if _, err := tx.ExecContext(ctx, `
			UPDATE `+s.qualified("observations")+`
			 SET deleted_at = now(), superseded_by_sync_id = $1, updated_at = now()
			 WHERE id = $2 AND deleted_at IS NULL`, p.SurvivorSyncID, id); err != nil {
			return fmt.Errorf("store: consolidate supersede source: %w", err)
		}
		if s.syncOn && s.enrolledInTx(ctx, tx, p.TargetProject) {
			var srcSyncID string
			if err := tx.QueryRowContext(ctx, `SELECT sync_id FROM `+s.qualified("observations")+` WHERE id = $1`, id).Scan(&srcSyncID); err != nil {
				return fmt.Errorf("store: consolidate source sync id: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO `+s.qualified("sync_mutations")+` (target_key, entity, entity_key, op, payload, source, project)
				VALUES ('local', 'observation', $1, 'delete', '{}', 'local', $2)`,
				srcSyncID, p.TargetProject); err != nil {
				return fmt.Errorf("store: consolidate source journal: %w", err)
			}
		}
	}
	if s.syncOn && s.enrolledInTx(ctx, tx, p.TargetProject) {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO `+s.qualified("sync_mutations")+` (target_key, entity, entity_key, op, payload, source, project)
			VALUES ('local', 'observation', $1, 'upsert', $2, 'local', $3)`,
			p.SurvivorSyncID, payload, p.TargetProject); err != nil {
			return fmt.Errorf("store: consolidate survivor journal: %w", err)
		}
	}
	return nil
}

// ensureSessionTxn creates the session row if missing (auto-register).
func (s *pgStore) ensureSessionTxn(ctx context.Context, tx *sql.Tx, sessionID, project string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO `+s.qualified("sessions")+` (id, project, directory, started_at)
		VALUES ($1, $2, '', now())
		ON CONFLICT(id) DO NOTHING`,
		sessionID, nullableString(project),
	)
	return err
}

// resolveProjectTxn: explicit project wins; otherwise inherit the session's
// project.
func (s *pgStore) resolveProjectTxn(ctx context.Context, tx *sql.Tx, sessionID, project string) string {
	if strings.TrimSpace(project) != "" {
		return strings.TrimSpace(project)
	}
	var p string
	_ = tx.QueryRowContext(ctx, `SELECT COALESCE(project, '') FROM `+s.qualified("sessions")+` WHERE id = $1`, sessionID).Scan(&p)
	return p
}

// enrolledInTx is the in-transaction enrollment check.
func (s *pgStore) enrolledInTx(ctx context.Context, tx *sql.Tx, project string) bool {
	var n int
	_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+s.qualified("sync_enrolled_projects")+` WHERE project = $1`, project).Scan(&n)
	return n > 0
}

// syncEnabled / SetSyncEnabled (the SyncSeam shape, the SQLite leg's pattern).
func (s *pgStore) syncEnabled() bool      { return s.syncOn }
func (s *pgStore) SetSyncEnabled(on bool) { s.syncOn = on }

// ── relations ───────────────────────────────────────────────────────────────

// AddRelation inserts a memory_relations row and returns the stored row
// (the SQLite leg's AddRelation semantics: sync_id minted here, "rel" prefix,
// INV-6; judgment_status defaults to 'pending').
func (s *pgStore) AddRelation(ctx context.Context, p AddRelationParams) (Relation, error) {
	var rel Relation
	syncID := syncid.NewSyncID("rel")
	var createdAt time.Time
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO `+s.qualified("memory_relations")+` (sync_id, source_id, target_id, relation, reason, evidence, session_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, judgment_status, COALESCE(reason,''), COALESCE(evidence,''), COALESCE(session_id,''), created_at`,
		syncID, p.SourceID, p.TargetID, p.Relation, nullableString(p.Reason), nullableString(p.Evidence), nullableString(p.SessionID)).
		Scan(&rel.ID, &rel.JudgmentStatus, &rel.Reason, &rel.Evidence, &rel.SessionID, &createdAt)
	if err != nil {
		return rel, err
	}
	rel.SyncID = syncID
	rel.SourceID = p.SourceID
	rel.TargetID = p.TargetID
	rel.Relation = p.Relation
	rel.CreatedAt = stamp(createdAt)
	return rel, nil
}

// AddPulledRelation inserts a pulled relation with the caller's (source's)
// sync_id (the SQLite leg's AddPulledRelation pattern).
func (s *pgStore) AddPulledRelation(ctx context.Context, p AddRelationParams, syncID string) (Relation, error) {
	var rel Relation
	var createdAt time.Time
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO `+s.qualified("memory_relations")+` (sync_id, source_id, target_id, relation, reason, evidence, session_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, judgment_status, COALESCE(reason,''), COALESCE(evidence,''), COALESCE(session_id,''), created_at`,
		syncID, p.SourceID, p.TargetID, p.Relation, nullableString(p.Reason), nullableString(p.Evidence), nullableString(p.SessionID)).
		Scan(&rel.ID, &rel.JudgmentStatus, &rel.Reason, &rel.Evidence, &rel.SessionID, &createdAt)
	if err != nil {
		return rel, err
	}
	rel.SyncID = syncID
	rel.SourceID = p.SourceID
	rel.TargetID = p.TargetID
	rel.Relation = p.Relation
	rel.CreatedAt = stamp(createdAt)
	return rel, nil
}

// JudgeRelation records a verdict on a pending candidate (guarded on
// judgment_status = 'pending' — an already-judged row is a no-op).
func (s *pgStore) JudgeRelation(ctx context.Context, id int64, p JudgeParams) error {
	conf := any(nil)
	if p.Confidence != nil {
		conf = *p.Confidence
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE `+s.qualified("memory_relations")+`
		 SET relation = $1, reason = $2, evidence = $3, confidence = $4,
		     judgment_status = 'judged',
		     marked_by_actor = $5, marked_by_kind = $6, marked_by_model = $7,
		     updated_at = now()
		 WHERE id = $8 AND judgment_status = 'pending'`,
		p.Relation, nullableString(p.Reason), nullableString(p.Evidence), conf,
		p.MarkedByActor, p.MarkedByKind, nullableString(p.MarkedByModel), id)
	return err
}

// JudgeMany records a batch of verdicts in ONE transaction (the
// mem_judge_many seam — 1–32 judgments in one queued write).
func (s *pgStore) JudgeMany(ctx context.Context, rs []JudgeRequest, base JudgeParams) ([]Relation, error) {
	var out []Relation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, r := range rs {
		rel, err := s.getRelationTxn(ctx, tx, r.JudgmentID)
		if err != nil {
			return nil, fmt.Errorf("store: judge_many relation %d: %w", r.JudgmentID, err)
		}
		if rel.JudgmentStatus == "judged" {
			out = append(out, rel)
			continue
		}
		conf := r.Confidence
		p := base
		p.Relation = r.Relation
		p.Reason = r.Reason
		p.Evidence = r.Evidence
		p.Confidence = &conf
		if err := s.judgeRelationTxn(ctx, tx, r.JudgmentID, p); err != nil {
			return nil, err
		}
		rel, err = s.getRelationTxn(ctx, tx, r.JudgmentID)
		if err != nil {
			return nil, err
		}
		out = append(out, rel)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if s.txnHook != nil {
		s.txnHook()
	}
	return out, nil
}

func (s *pgStore) judgeRelationTxn(ctx context.Context, tx *sql.Tx, id int64, p JudgeParams) error {
	conf := any(nil)
	if p.Confidence != nil {
		conf = *p.Confidence
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE `+s.qualified("memory_relations")+`
		 SET relation = $1, reason = $2, evidence = $3, confidence = $4,
		     judgment_status = 'judged',
		     marked_by_actor = $5, marked_by_kind = $6, marked_by_model = $7,
		     updated_at = now()
		 WHERE id = $8 AND judgment_status = 'pending'`,
		p.Relation, nullableString(p.Reason), nullableString(p.Evidence), conf,
		p.MarkedByActor, p.MarkedByKind, nullableString(p.MarkedByModel), id)
	return err
}

// GetRelation reads one memory_relations row by id (sql.ErrNoRows when absent).
func (s *pgStore) GetRelation(ctx context.Context, id int64) (Relation, error) {
	return s.getRelationTxn(ctx, s.db, id)
}

func (s *pgStore) getRelationTxn(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id int64) (Relation, error) {
	var rel Relation
	var confidence sql.NullFloat64
	var actor, kind, model sql.NullString
	var createdAt time.Time
	err := q.QueryRowContext(ctx, `
		SELECT id, COALESCE(sync_id,''), source_id, target_id, relation,
		       COALESCE(reason,''), COALESCE(evidence,''), COALESCE(session_id,''),
		       judgment_status, confidence,
		       COALESCE(marked_by_actor,''), COALESCE(marked_by_kind,''), COALESCE(marked_by_model,''),
		       created_at
		 FROM `+s.qualified("memory_relations")+` WHERE id = $1`, id).
		Scan(&rel.ID, &rel.SyncID, &rel.SourceID, &rel.TargetID, &rel.Relation,
			&rel.Reason, &rel.Evidence, &rel.SessionID, &rel.JudgmentStatus, &confidence,
			&actor, &kind, &model, &createdAt)
	if err != nil {
		return rel, err
	}
	if confidence.Valid {
		rel.Confidence = confidence.Float64
	}
	rel.MarkedByActor = actor.String
	rel.MarkedByKind = kind.String
	rel.MarkedByModel = model.String
	rel.CreatedAt = stamp(createdAt)
	return rel, nil
}

// ── retrieval ───────────────────────────────────────────────────────────────

// Search runs the tsvector FTS path (weighted ts_rank_cd), with the < 3-rune
// pg_trgm/LIKE fallback. The composite rank is the same formula as the
// SQLite leg (relevance × the pinned/recency/stability boosts), with the
// relevance base clamped to 1.0 (the SQLite leg's searchRankRelevanceBase
// rationale: the boosts stay load-bearing in the all-identical case).
func (s *pgStore) Search(ctx context.Context, q SearchQuery) (SearchResult, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if hasShortFTSTerm(q.Query) {
		return s.searchLIKEPG(ctx, q, limit)
	}
	// The relevance base is ts_rank_cd (weighted, cover-density) clamped to
	// >= 1.0 so the pinned/recency/stability boosts stay load-bearing in the
	// all-identical case (the SQLite leg's searchRankRelevanceBase rationale).
	rankExpr := `GREATEST(ts_rank_cd(o.tsv, plainto_tsquery('simple', $1), 32), 1.0) * ` + s.searchRankBoostPG()
	qStr := `SELECT ` + s.obsSelectColumnsPG() + `, ` + rankExpr + ` AS rank
		FROM ` + s.qualified("observations") + ` o
		WHERE o.tsv @@ plainto_tsquery('simple', $1) AND o.deleted_at IS NULL`
	args := []any{q.Query}
	argN := 2
	if q.Project != "" {
		qStr += ` AND LOWER(o.project) = $` + itoa(argN)
		args = append(args, strings.ToLower(q.Project))
		argN++
	}
	if q.Type != "" {
		qStr += ` AND o.type = $` + itoa(argN)
		args = append(args, q.Type)
		argN++
	}
	if q.Scope != "" {
		qStr += ` AND o.scope = $` + itoa(argN)
		args = append(args, q.Scope)
		argN++
	}
	qStr += ` ORDER BY rank DESC LIMIT $` + itoa(argN)
	args = append(args, limit)
	return s.queryObservationsPG(ctx, qStr, args...)
}

// searchLIKEPG is the bounded substring fallback for short terms (spec
// 001-storage §3: pg_trgm / LIKE): the same 5-column LIKE match set as the
// SQLite leg, ranked by updated_at (the SQLite leg's fallback rank — parity).
func (s *pgStore) searchLIKEPG(ctx context.Context, q SearchQuery, limit int) (SearchResult, error) {
	termGroups := make([]string, 0)
	args := []any{}
	argN := 1
	for range strings.Fields(q.Query) {
		termGroups = append(termGroups,
			`(o.title ILIKE $`+itoa(argN)+` OR o.content ILIKE $`+itoa(argN)+` OR o.type ILIKE $`+itoa(argN)+` OR o.project ILIKE $`+itoa(argN)+` OR o.topic_key ILIKE $`+itoa(argN)+`)`)
		args = append(args, "%"+q.Query+"%")
		argN++
	}
	qStr := `SELECT ` + s.obsSelectColumnsPG() + `, 0.0 AS rank
		FROM ` + s.qualified("observations") + ` o
		WHERE o.deleted_at IS NULL`
	for _, g := range termGroups {
		qStr += ` AND ` + g
	}
	if q.Project != "" {
		qStr += ` AND LOWER(o.project) = $` + itoa(argN)
		args = append(args, strings.ToLower(q.Project))
		argN++
	}
	if q.Type != "" {
		qStr += ` AND o.type = $` + itoa(argN)
		args = append(args, q.Type)
		argN++
	}
	if q.Scope != "" {
		qStr += ` AND o.scope = $` + itoa(argN)
		args = append(args, q.Scope)
		argN++
	}
	qStr += ` ORDER BY o.updated_at DESC, o.id DESC LIMIT $` + itoa(argN)
	args = append(args, limit)
	return s.queryObservationsPG(ctx, qStr, args...)
}

// itoa is an int → string helper for the $n placeholder builder.
func itoa(n int) string { return strconv.Itoa(n) }

// queryObservationsPG runs a SELECT returning the 26 observation columns plus
// a trailing rank column.
func (s *pgStore) queryObservationsPG(ctx context.Context, q string, args ...any) (SearchResult, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return SearchResult{}, err
	}
	defer rows.Close()
	var res SearchResult
	for rows.Next() {
		obs, err := scanObservationPG(rows, true)
		if err != nil {
			return SearchResult{}, err
		}
		res.Observations = append(res.Observations, obs)
	}
	return res, rows.Err()
}

// VectorSearch is the pgvector leg (spec 001-storage §4): ORDER BY the
// cosine-distance operator over the vector column, the same degraded contract
// as the SQLite leg (empty + Degraded when there are no comparable vectors).
// The vector(N) dimension is pinned in the migration; a query of a different
// length is not comparable (the same rule as the SQLite leg's BLOB leg).
func (s *pgStore) VectorSearch(ctx context.Context, q VecQuery) (VecResult, error) {
	if len(q.QueryVector) == 0 {
		return VecResult{Degraded: true}, nil
	}
	topK := q.TopK
	if topK <= 0 {
		topK = 10
	}
	// pgvector accepts the vector literal as a text array-of-floats cast.
	lit := make([]string, len(q.QueryVector))
	for i, f := range q.QueryVector {
		lit[i] = fmt.Sprintf("%g", f)
	}
	vecLit := "[" + strings.Join(lit, ",") + "]"
	args := []any{vecLit}
	qStr := `SELECT ` + s.obsSelectColumnsPG() + `
		FROM ` + s.qualified("observations") + ` o
		WHERE o.deleted_at IS NULL AND o.embedding IS NOT NULL
		  AND vector_dims(o.embedding) = ` + itoa(len(q.QueryVector))
	if q.Project != "" {
		qStr += ` AND LOWER(o.project) = $2`
		args = append(args, strings.ToLower(q.Project))
	}
	qStr += ` ORDER BY o.embedding <-> $1::vector LIMIT $` + itoa(len(args)+1)
	args = append(args, topK)
	rows, err := s.db.QueryContext(ctx, qStr, args...)
	if err != nil {
		// degrade on any query failure (INV-3) — never a silent error-as-success.
		return VecResult{Degraded: true}, err
	}
	defer rows.Close()
	var res VecResult
	for rows.Next() {
		obs, err := scanObservationPG(rows, false)
		if err != nil {
			return VecResult{Degraded: true}, err
		}
		res.Observations = append(res.Observations, obs)
	}
	if err := rows.Err(); err != nil {
		return VecResult{Degraded: true}, err
	}
	if len(res.Observations) == 0 {
		return VecResult{Degraded: true}, nil
	}
	return res, nil
}

// Context is the orientation layer (spec 01-memory/005 §2): pinned first
// (capped at 20), then recent, trimmed to the byte budget — the same
// semantics as the SQLite leg.
func (s *pgStore) Context(ctx context.Context, q ContextQuery) (ContextResult, error) {
	budget := q.MaxBytes
	if budget <= 0 {
		budget = 16 * 1024
	}
	if budget > 64*1024 {
		budget = 64 * 1024
	}
	qStr := `SELECT ` + s.obsSelectColumnsPG() + `
		FROM ` + s.qualified("observations") + ` o
		WHERE o.deleted_at IS NULL`
	args := []any{}
	argN := 1
	if q.Project != "" {
		qStr += ` AND LOWER(o.project) = $` + itoa(argN)
		args = append(args, strings.ToLower(q.Project))
		argN++
	}
	qStr += ` ORDER BY CASE WHEN o.pinned THEN 0 ELSE 1 END, o.updated_at DESC, o.id DESC
		LIMIT $` + itoa(argN)
	args = append(args, 40)
	rows, err := s.db.QueryContext(ctx, qStr, args...)
	if err != nil {
		return ContextResult{}, err
	}
	defer rows.Close()
	var all []Observation
	for rows.Next() {
		obs, err := scanObservationPG(rows, false)
		if err != nil {
			return ContextResult{}, err
		}
		all = append(all, obs)
	}
	if err := rows.Err(); err != nil {
		return ContextResult{}, err
	}
	res := ContextResult{}
	pinnedSeen := 0
	for _, obs := range all {
		if obs.Pinned {
			if pinnedSeen >= 20 {
				continue
			}
			pinnedSeen++
		}
		cost := len(obs.Content) + len(obs.Title)
		if res.TotalBytes+cost > budget && res.TotalBytes > 0 {
			break
		}
		res.Observations = append(res.Observations, obs)
		res.TotalBytes += cost
	}
	return res, nil
}

// ── sync ────────────────────────────────────────────────────────────────────

// EnqueueMutation journals a mutation and returns its seq.
func (s *pgStore) EnqueueMutation(ctx context.Context, m Mutation) (int64, error) {
	var seq int64
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO `+s.qualified("sync_mutations")+` (target_key, entity, entity_key, op, payload, source, project)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING seq`,
		m.TargetKey, m.Entity, m.EntityKey, m.Op, m.Payload, m.Source, nullableString(m.Project),
	).Scan(&seq); err != nil {
		return 0, err
	}
	return seq, tx.Commit()
}

// ListPendingMutations returns un-acked mutations after the cursor.
func (s *pgStore) ListPendingMutations(ctx context.Context, cursor int64, limit int) ([]Mutation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT seq, target_key, entity, entity_key, op, payload, COALESCE(source,'local'), COALESCE(project,''), occurred_at
		FROM `+s.qualified("sync_mutations")+`
		WHERE seq > $1 AND acked_at IS NULL
		ORDER BY seq ASC LIMIT $2`, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Mutation
	for rows.Next() {
		var m Mutation
		var occurredAt time.Time
		if err := rows.Scan(&m.Seq, &m.TargetKey, &m.Entity, &m.EntityKey, &m.Op, &m.Payload, &m.Source, &m.Project, &occurredAt); err != nil {
			return nil, err
		}
		m.OccurredAt = stamp(occurredAt)
		out = append(out, m)
	}
	return out, rows.Err()
}

// ApplyPulledMutation applies one pulled mutation through the store's local
// write path (the SQLite leg's hook pattern — memory.init registers the
// dispatch; the store never imports memory).
func (s *pgStore) ApplyPulledMutation(ctx context.Context, m Mutation) error {
	if applyPulledMutationHook != nil {
		return applyPulledMutationHook(ctx, s, m)
	}
	return ErrNotImplemented{Method: "ApplyPulledMutation (register the hook — memory.init)"}
}

// SetEnrolledForTest enrolls a project in sync_enrolled_projects (test-only
// seam, the SQLite leg's shape).
func (s *pgStore) SetEnrolledForTest(project string) error {
	_, err := s.db.Exec(`
		INSERT INTO `+s.qualified("sync_enrolled_projects")+` (project) VALUES ($1)
		ON CONFLICT(project) DO NOTHING`, project)
	return err
}

// Tombstone records a delete tombstone.
func (s *pgStore) Tombstone(ctx context.Context, entity, key string) error {
	_, err := s.db.Exec(`
		INSERT INTO `+s.qualified("sync_delete_tombstones")+` (entity, entity_key, deleted_at)
		VALUES ($1, $2, now())
		ON CONFLICT(entity, entity_key) DO UPDATE SET deleted_at = now()`,
		entity, key)
	return err
}

// ── additive methods ────────────────────────────────────────────────────────

// MarkReviewed resets review_after = now + type-months (the SQLite leg's
// MarkReviewed semantics).
func (s *pgStore) MarkReviewed(ctx context.Context, id int64) error {
	var typ string
	if err := s.db.QueryRowContext(ctx, `SELECT type FROM `+s.qualified("observations")+` WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&typ); err != nil {
		return err
	}
	months := decayMonths(typ)
	var ra any
	if months > 0 {
		ra = time.Now().UTC().AddDate(0, months, 0).Format("2006-01-02 15:04:05")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE `+s.qualified("observations")+` SET review_after = $1, updated_at = now() WHERE id = $2`, ra, id)
	return err
}

// BumpRetrievalUsage increments retrieval_usage per id.
func (s *pgStore) BumpRetrievalUsage(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	for i, id := range ids {
		placeholders[i] = "$" + itoa(i+1)
		args = append(args, id)
	}
	args = append(args, len(args)+1)
	_, err := s.db.ExecContext(ctx, `UPDATE `+s.qualified("observations")+`
		SET retrieval_usage = retrieval_usage + 1, updated_at = now()
		WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	return err
}

// SearchPrompts searches the prompts tsvector index and returns each hit with
// its full identity (PromptRow).
func (s *pgStore) SearchPrompts(ctx context.Context, q PromptSearchQuery) (PromptSearchResult, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	qStr := `SELECT p.id, COALESCE(p.sync_id,''), p.session_id, COALESCE(p.project,''), p.content, p.created_at
		FROM ` + s.qualified("user_prompts") + ` p
		WHERE p.tsv @@ plainto_tsquery('simple', $1)`
	args := []any{q.Query}
	argN := 2
	if q.Project != "" {
		qStr += ` AND LOWER(p.project) = $` + itoa(argN)
		args = append(args, strings.ToLower(q.Project))
		argN++
	}
	qStr += ` ORDER BY ts_rank(p.tsv, plainto_tsquery('simple', $1)) DESC LIMIT $` + itoa(argN)
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, qStr, args...)
	if err != nil {
		return PromptSearchResult{Degraded: true}, err
	}
	defer rows.Close()
	var res PromptSearchResult
	for rows.Next() {
		var row PromptRow
		var createdAt time.Time
		if err := rows.Scan(&row.ID, &row.SyncID, &row.SessionID, &row.Project, &row.Content, &createdAt); err != nil {
			return PromptSearchResult{Degraded: true}, err
		}
		row.CreatedAt = stamp(createdAt)
		res.Prompts = append(res.Prompts, row)
	}
	return res, rows.Err()
}

// TouchForTest sets last_seen_at and revision_count on an observation
// (test-only hook for the ranking boosts).
func (s *pgStore) TouchForTest(ctx context.Context, id int64, lastSeenAt string, revisionCount int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE `+s.qualified("observations")+`
		SET last_seen_at = $1, revision_count = $2 WHERE id = $3`, lastSeenAt, revisionCount, id)
	return err
}

// InsertPrompt is a test-only seam (the SQLite leg's InsertPrompt pattern):
// inserts a user_prompts row directly so SearchPrompts can be exercised
// end-to-end. Returns the prompt's local id.
func (s *pgStore) InsertPrompt(ctx context.Context, sessionID, content, project string) (int64, error) {
	syncID := syncid.NewSyncID("prm")
	var id int64
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := s.ensureSessionTxn(ctx, tx, sessionID, project); err != nil {
		return 0, err
	}
	resolved := s.resolveProjectTxn(ctx, tx, sessionID, project)
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO `+s.qualified("user_prompts")+` (sync_id, session_id, content, project)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		syncID, sessionID, content, nullableString(resolved),
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// SetTxnHook sets the committed-transaction counter (the Task 10 seam).
func (s *pgStore) SetTxnHook(hook func()) {
	s.txnHook = hook
}

// ── lifecycle ───────────────────────────────────────────────────────────────

// Migrate re-runs the migration set (idempotent, additive-only).
func (s *pgStore) Migrate(ctx context.Context) error {
	return s.migrate()
}

// Close closes the database handle (the PG store is not pooled — each Open
// takes a fresh connection pool; the schema is the isolation boundary).
func (s *pgStore) Close() error {
	return s.db.Close()
}

// Ping checks the database is alive.
func (s *pgStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// BeginSaveTxn is the save-transaction gate seam (the SQLite leg's shape).
// The PG gate shares the SaveTXN interface but renders its SQL for PG.
func (s *pgStore) BeginSaveTxn(ctx context.Context) (SaveTXN, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &pgSaveTXN{st: s, tx: tx}, nil
}

// pgSaveTXN is the PG render of the save gate (same interface, PG SQL).
type pgSaveTXN struct {
	st *pgStore
	tx *sql.Tx
}

func (t *pgSaveTXN) CheckSession(ctx context.Context, sessionID, project string) (string, error) {
	resolved := strings.TrimSpace(project)
	if strings.TrimSpace(sessionID) == "" {
		return resolved, nil
	}
	if resolved == "" {
		var p string
		if err := t.tx.QueryRowContext(ctx, `SELECT COALESCE(project, '') FROM `+t.st.qualified("sessions")+` WHERE id = $1`, sessionID).Scan(&p); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("store: session lookup: %w", err)
		}
		resolved = p
	}
	var owned, endedAt sql.NullString
	err := t.tx.QueryRowContext(ctx, `SELECT COALESCE(ownership_mode,'shared'), COALESCE(ended_at::text,'') FROM `+t.st.qualified("sessions")+` WHERE id = $1`, sessionID).Scan(&owned, &endedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// Auto-register (create-on-first-use, the SQLite leg's behavior).
		if _, err := t.tx.ExecContext(ctx, `
			INSERT INTO `+t.st.qualified("sessions")+` (id, project, directory, ownership_mode, started_at, created_at)
			VALUES ($1, $2, '', 'shared', now(), now())
			ON CONFLICT(id) DO NOTHING`,
			sessionID, nullableString(resolved)); err != nil {
			return "", fmt.Errorf("store: auto-register session: %w", err)
		}
		return resolved, nil
	}
	if err != nil {
		return "", fmt.Errorf("store: session lookup: %w", err)
	}
	if endedAt.String != "" {
		return "", ErrSessionEnded
	}
	if owned.String == "project_owned" && resolved != "" && strings.TrimSpace(project) != "" {
		var rowProj string
		_ = t.tx.QueryRowContext(ctx, `SELECT COALESCE(project,'') FROM `+t.st.qualified("sessions")+` WHERE id = $1`, sessionID).Scan(&rowProj)
		if rowProj != resolved {
			return "", ErrSessionProjectMismatch
		}
	}
	return resolved, nil
}

func (t *pgSaveTXN) LookupHash(ctx context.Context, project, scope, typ, title, normalizedHash string) (int64, bool, error) {
	var id int64
	err := t.tx.QueryRowContext(ctx, `
		SELECT id FROM `+t.st.qualified("observations")+`
		 WHERE normalized_hash = $1 AND COALESCE(project,'') = $2 AND scope = $3 AND type = $4 AND title = $5
		   AND deleted_at IS NULL`,
		normalizedHash, project, scope, typ, title).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (t *pgSaveTXN) LookupTopicKey(ctx context.Context, project, scope, topicKey string) (int64, bool, error) {
	var id int64
	err := t.tx.QueryRowContext(ctx, `
		SELECT id FROM `+t.st.qualified("observations")+`
		 WHERE COALESCE(project,'') = $1 AND scope = $2 AND topic_key = $3 AND deleted_at IS NULL`,
		project, scope, topicKey).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (t *pgSaveTXN) LookupSyncID(ctx context.Context, id int64) (string, error) {
	var out string
	err := t.tx.QueryRowContext(ctx, `SELECT sync_id FROM `+t.st.qualified("observations")+` WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&out)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrObservationNotFound
	}
	return out, err
}

func (t *pgSaveTXN) InsertObservation(ctx context.Context, p GateInsert) (int64, string, error) {
	now := pgNow()
	prov := Provenance{SessionID: p.SessionID, Writer: p.Writer, CreatedAt: now}
	provJSON, _ := json.Marshal(prov)
	var id int64
	err := t.tx.QueryRowContext(ctx, `
		INSERT INTO `+t.st.qualified("observations")+` (
			sync_id, session_id, type, title, content, tool_name, project, scope,
			topic_key, normalized_hash, revision_count, duplicate_count, last_seen_at,
			importance, retrieval_usage, provenance, created_at, updated_at, review_after
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, 0, now(), $11, 0, $12, $13, $14, $15)
		RETURNING id`,
		p.SyncID, p.SessionID, p.Type, p.Title, p.Content,
		nullableString(p.ToolName), nullableString(p.Project), scopeOrDefault(p.Scope),
		nullableString(p.TopicKey), p.NormalizedHash,
		p.Importance, provJSON, now, now, nullableString(deref(p.ReviewAfter)),
	).Scan(&id)
	if err != nil {
		return 0, "", err
	}
	// replay-safe ledger row (same fingerprint rule as the SQLite gate).
	if p.OperationID != "" {
		fp := "v1:" + storeOpFingerprint(p.Type, p.Title, p.Project, p.Scope, p.TopicKey, p.Content)
		if _, err := t.tx.ExecContext(ctx, `
			INSERT INTO `+t.st.qualified("observation_save_operations")+` (operation_id, fingerprint, observation_id)
			VALUES ($1, $2, $3)`,
			p.OperationID, fp, id); err != nil {
			return 0, "", err
		}
	}
	// the captured prompt (same transaction; INV-6 inside the gate).
	if p.PromptContent != "" {
		if _, err := t.tx.ExecContext(ctx, `
			INSERT INTO `+t.st.qualified("user_prompts")+` (sync_id, session_id, content, project, created_at)
			VALUES ($1, $2, $3, $4, now())`,
			syncid.NewSyncID("prm"), p.SessionID, p.PromptContent, nullableString(p.Project)); err != nil {
			return 0, "", err
		}
	}
	// step 7: the sync mutation is enrollment-gated.
	if t.st.syncOn && t.enrolled(p.Project) {
		if err := t.enqueueUpsert(p.Project, p.SyncID); err != nil {
			return 0, "", err
		}
	}
	return id, p.SyncID, nil
}

func (t *pgSaveTXN) ReviseObservation(ctx context.Context, id int64, p GateInsert) (int, error) {
	var curTitle, curContent string
	var curRev int
	err := t.tx.QueryRowContext(ctx, `SELECT title, content, revision_count FROM `+t.st.qualified("observations")+` WHERE id = $1 AND deleted_at IS NULL`, id).
		Scan(&curTitle, &curContent, &curRev)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w: %d", ErrObservationNotFound, id)
	}
	if err != nil {
		return 0, err
	}
	newRev := curRev + 1
	// snapshot the pre-state first (spec 002-schema §2.3).
	if _, err := t.tx.ExecContext(ctx, `
		INSERT INTO `+t.st.qualified("observation_versions")+` (observation_id, version, title, content, created_at)
		VALUES ($1, $2, $3, $4, now())`,
		id, newRev, nullableString(curTitle), curContent); err != nil {
		return 0, err
	}
	if _, err := t.tx.ExecContext(ctx, `
		UPDATE `+t.st.qualified("observations")+`
		SET title = $1, content = $2, normalized_hash = $3, revision_count = $4,
		    review_after = $5, updated_at = now()
		WHERE id = $6`,
		p.Title, p.Content, p.NormalizedHash, newRev,
		nullableString(deref(p.ReviewAfter)), id); err != nil {
		return 0, err
	}
	if p.PromptContent != "" {
		if _, err := t.tx.ExecContext(ctx, `
			INSERT INTO `+t.st.qualified("user_prompts")+` (sync_id, session_id, content, project, created_at)
			VALUES ($1, $2, $3, $4, now())`,
			syncid.NewSyncID("prm"), p.SessionID, p.PromptContent, nullableString(p.Project)); err != nil {
			return 0, err
		}
	}
	if t.st.syncOn && t.enrolled(p.Project) {
		if err := t.enqueueUpsert(p.Project, p.SyncID); err != nil {
			return 0, err
		}
	}
	return newRev, nil
}

func (t *pgSaveTXN) MarkDuplicate(ctx context.Context, id int64) error {
	_, err := t.tx.ExecContext(ctx, `
		UPDATE `+t.st.qualified("observations")+`
		SET duplicate_count = duplicate_count + 1, last_seen_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (t *pgSaveTXN) SupersedeTarget(ctx context.Context, targetID int64, newSyncID, project string) error {
	if _, err := t.tx.ExecContext(ctx, `
		UPDATE `+t.st.qualified("observations")+`
		SET deleted_at = now(), superseded_by_sync_id = $1, updated_at = now()
		WHERE id = $2 AND deleted_at IS NULL`, newSyncID, targetID); err != nil {
		return fmt.Errorf("store: supersede target: %w", err)
	}
	if t.st.syncOn && t.enrolled(project) {
		var targetSyncID string
		err := t.tx.QueryRowContext(ctx, `SELECT sync_id FROM `+t.st.qualified("observations")+` WHERE id = $1`, targetID).Scan(&targetSyncID)
		if err != nil {
			return fmt.Errorf("store: supersede target sync id: %w", err)
		}
		if _, err := t.tx.ExecContext(ctx, `
			INSERT INTO `+t.st.qualified("sync_mutations")+` (target_key, entity, entity_key, op, payload, source, project)
			VALUES ('local', 'observation', $1, 'delete', '{}', 'local', $2)`,
			targetSyncID, project); err != nil {
			return fmt.Errorf("store: supersede target journal: %w", err)
		}
	}
	return nil
}

func (t *pgSaveTXN) EnrollProject(ctx context.Context, project string) error {
	_, err := t.tx.ExecContext(ctx, `
		INSERT INTO `+t.st.qualified("sync_enrolled_projects")+` (project) VALUES ($1)
		ON CONFLICT(project) DO NOTHING`, project)
	return err
}

func (t *pgSaveTXN) Commit(ctx context.Context) error {
	err := t.tx.Commit()
	if err == nil && t.st.txnHook != nil {
		t.st.txnHook()
	}
	return err
}

func (t *pgSaveTXN) Rollback() error { return t.tx.Rollback() }

func (t *pgSaveTXN) enrolled(project string) bool {
	var n int
	_ = t.tx.QueryRow(`SELECT COUNT(*) FROM `+t.st.qualified("sync_enrolled_projects")+` WHERE project = $1`, project).Scan(&n)
	return n > 0
}

func (t *pgSaveTXN) enqueueUpsert(project, syncID string) error {
	var sessionID, scope, typ, title, content string
	if err := t.tx.QueryRow(
		`SELECT COALESCE(session_id,''), COALESCE(scope,''), COALESCE(type,''), COALESCE(title,''), COALESCE(content,'')
		 FROM `+t.st.qualified("observations")+` WHERE sync_id = $1`, syncID).
		Scan(&sessionID, &scope, &typ, &title, &content); err != nil {
		return fmt.Errorf("store: journal payload read: %w", err)
	}
	payload := `{"sync_id":"` + jsonEscape(syncID) +
		`","session_id":"` + jsonEscape(sessionID) +
		`","type":"` + jsonEscape(typ) +
		`","title":"` + jsonEscape(title) +
		`","content":"` + jsonEscape(content) +
		`","project":"` + jsonEscape(project) +
		`","scope":"` + jsonEscape(scope) + `"}`
	_, err := t.tx.Exec(`
		INSERT INTO `+t.st.qualified("sync_mutations")+` (target_key, entity, entity_key, op, payload, source, project)
		VALUES ('local', 'observation', $1, 'upsert', $2, 'local', $3)`,
		syncID, payload, project)
	return err
}
