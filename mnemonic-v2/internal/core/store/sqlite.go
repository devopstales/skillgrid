package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/syncid"
	"modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// sqliteStore implements Store on SQLite (one file per project, WAL).
type sqliteStore struct {
	db   *sql.DB
	path string
	// pooled is true when the DB handle came from the refcounted pool
	// (Close releases a pool ref rather than closing the DB).
	pooled bool
	// txnHook is the committed-transaction counter seam (Task 10).
	txnHook func()
}

var _ Store = (*sqliteStore)(nil)

// Open opens or creates the SQLite database for project under dataDir and
// runs its migrations. Refcounted through the handle pool: a second Open for
// the same path returns the shared handle (same *sql.DB). Set
// MNEMONIC_DISABLE_CACHE=1 to bypass the pool.
//
// dsn is reserved for the Postgres dialect; the SQLite dialect derives its DSN
// from dataDir/project (the WAL pragmas + _txlock) and ignores dsn.
func Open(ctx context.Context, dataDir, project string, dialect Dialect, dsn string) (Store, error) {
	if dialect != SQLite {
		return nil, fmt.Errorf("store.Open: dialect %v not implemented yet (SQLite only in Task 1)", dialect)
	}
	project = strings.TrimSpace(project)
	if project == "" {
		return nil, errors.New("store.Open: project is required")
	}
	if strings.Contains(project, "..") {
		return nil, fmt.Errorf("store.Open: invalid project id %q", project)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("store.Open: create data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, project+".sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("store.Open: create db dir: %w", err)
	}
	db, err := poolAcquire(dbPath, func() (*sql.DB, error) {
		return openSQLite(dbPath)
	})
	if err != nil {
		return nil, err
	}
	pooled := !poolDisabled()
	if err := migrateSQLite(db); err != nil {
		if pooled {
			_ = poolRelease(dbPath)
		} else {
			_ = db.Close()
		}
		return nil, fmt.Errorf("store.Open: migrate: %w", err)
	}
	return &sqliteStore{db: db, path: dbPath, pooled: pooled}, nil
}

// openSQLite opens the file with the Engram-proven WAL DSN: pragmas as
// _pragma query parameters (applied to every physical connection),
// _txlock=immediate (BEGIN IMMEDIATE so busy_timeout serializes writers),
// and a single-connection pool (MaxOpenConns=1).
func openSQLite(dbPath string) (*sql.DB, error) {
	q := url.Values{}
	q.Set("_txlock", "immediate")
	for _, p := range []string{
		"busy_timeout(5000)",
		"journal_mode(WAL)",
		"synchronous(NORMAL)",
		"foreign_keys(1)",
	} {
		q.Add("_pragma", p)
	}
	fileURL := (&url.URL{Scheme: "file", Path: dbPath}).String()
	dsn := fileURL + "?" + q.Encode()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// ── Migrations (ported from v1 store.go:462-494 + the index_meta runner) ──

// migrateSQLite runs every unapplied embedded migration, statement by
// statement (trigger-aware splitter), recording each in index_meta.
func migrateSQLite(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS index_meta (key TEXT PRIMARY KEY, schema_version INTEGER NOT NULL DEFAULT 0)`); err != nil {
		return fmt.Errorf("create index_meta: %w", err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
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
		if err := db.QueryRow(`SELECT COUNT(*) > 0 FROM index_meta WHERE key = ?`, "migration:"+name).Scan(&done); err != nil {
			return fmt.Errorf("check %s: %w", name, err)
		}
		if done {
			continue
		}
		b, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		for _, stmt := range splitSQLStatements(string(b)) {
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				if isDuplicateColumnErr(err) {
					// Additive-only discipline: a column that already
					// exists (partial apply) is safe to skip.
					continue
				}
				return fmt.Errorf("exec %s: %w", name, err)
			}
		}
		if _, err := db.Exec(`INSERT INTO index_meta (key, schema_version) VALUES (?, 1)`, "migration:"+name); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
	}
	if _, err := db.Exec(`
		INSERT INTO index_meta (key, schema_version) VALUES ('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET schema_version = excluded.schema_version`,
		len(names),
	); err != nil {
		return fmt.Errorf("record schema_version: %w", err)
	}
	return nil
}

// isDuplicateColumnErr reports "duplicate column name", the only ADD COLUMN
// failure safe to skip on a partially migrated store.
func isDuplicateColumnErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}

// splitSQLStatements splits a migration script into statements. Semicolons
// terminate plain statements, but a CREATE TRIGGER body runs from its BEGIN
// to its matching END. Single-quoted strings, -- line comments, and /* */
// block comments are skipped. Ported from v1 store.go:462-494.
func splitSQLStatements(s string) []string {
	var (
		out    []string
		cur    strings.Builder
		inS    bool
		inLine bool
		inBLK  bool
		inTrig bool
	)
	flush := func() {
		out = append(out, strings.TrimSpace(cur.String()))
		cur.Reset()
	}
	opensTrigger := func() bool {
		return strings.Contains(cur.String(), "CREATE TRIGGER")
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inLine:
			if c == '\n' {
				inLine = false
			}
			cur.WriteByte(c)
		case inBLK:
			if c == '*' && i+1 < len(s) && s[i+1] == '/' {
				cur.WriteString("*/")
				i++
				inBLK = false
			} else {
				cur.WriteByte(c)
			}
		case inS:
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					cur.WriteString("''")
					i++
				} else {
					inS = false
					cur.WriteByte(c)
				}
			} else {
				cur.WriteByte(c)
			}
		case inTrig:
			if (c == 'E' || c == 'e') && matchWord(s, i, "END") {
				j := i + 3
				for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
					j++
				}
				if j < len(s) && s[j] == ';' {
					cur.WriteString(s[i : j+1])
					i = j
				} else {
					cur.WriteString(s[i : i+3])
					i += 2
				}
				flush()
				inTrig = false
			} else {
				cur.WriteByte(c)
			}
		default:
			switch c {
			case '\'':
				inS = true
				cur.WriteByte(c)
			case '-':
				if i+1 < len(s) && s[i+1] == '-' {
					inLine = true
					cur.WriteString("--")
					i++
				} else {
					cur.WriteByte(c)
				}
			case '/':
				if i+1 < len(s) && s[i+1] == '*' {
					inBLK = true
					cur.WriteString("/*")
					i++
				} else {
					cur.WriteByte(c)
				}
			case ';':
				flush()
			default:
				cur.WriteByte(c)
				if (c == 'B' || c == 'b') && matchWord(s, i, "BEGIN") && opensTrigger() {
					inTrig = true
				}
			}
		}
	}
	flush()
	return out
}

// matchWord reports whether s[i:] starts the identifier w as a whole word.
func matchWord(s string, i int, w string) bool {
	if i+len(w) > len(s) {
		return false
	}
	for k := 0; k < len(w); k++ {
		a, b := s[i+k], w[k]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

// ── Write retry (ported from Engram store.go:10180-10207) ──────────────────

const (
	sqlitePrimaryBusy   = 5
	sqlitePrimaryLocked = 6
)

var sqliteWriteRetryBackoffs = []time.Duration{
	10 * time.Millisecond,
	25 * time.Millisecond,
	50 * time.Millisecond,
	100 * time.Millisecond,
	200 * time.Millisecond,
}

// withWriteRetry retries fn on SQLITE_BUSY/SQLITE_LOCKED with the Engram
// backoffs.
func (s *sqliteStore) withWriteRetry(fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= len(sqliteWriteRetryBackoffs); attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if !isRetryableSQLiteLockError(err) || attempt == len(sqliteWriteRetryBackoffs) {
				return err
			}
			time.Sleep(sqliteWriteRetryBackoffs[attempt])
			continue
		}
		return nil
	}
	return lastErr
}

func isRetryableSQLiteLockError(err error) bool {
	if err == nil {
		return false
	}
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		primary := sqliteErr.Code() & 0xff
		return primary == sqlitePrimaryBusy || primary == sqlitePrimaryLocked
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database is busy") ||
		strings.Contains(msg, "sqlite_busy") ||
		strings.Contains(msg, "sqlite_locked")
}

// ── helpers ─────────────────────────────────────────────────────────────────

// nowStamp is the timestamp format used for created_at/updated_at/etc.
func nowStamp() string { return time.Now().UTC().Format("2006-01-02 15:04:05") }

// hashNormalized: lowercase + whitespace-collapse + SHA-256 hex (Engram
// store.go:13665).
func hashNormalized(content string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(content), " "))
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}

// nullableString turns "" into a SQL NULL.
func nullableString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// ensureSessionTx creates the session row if missing (auto-register, shared
// ownership).
func (s *sqliteStore) ensureSessionTx(tx *sql.Tx, sessionID, project string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	_, err := tx.Exec(`
		INSERT INTO sessions (id, project, directory, started_at)
		VALUES (?, ?, '', datetime('now'))
		ON CONFLICT(id) DO NOTHING`,
		sessionID, nullableString(project),
	)
	return err
}

// resolveProjectTx: explicit project wins; otherwise inherit the session's
// project.
func (s *sqliteStore) resolveProjectTx(tx *sql.Tx, sessionID, project string) string {
	if strings.TrimSpace(project) != "" {
		return strings.TrimSpace(project)
	}
	var p string
	_ = tx.QueryRow(`SELECT ifnull(project, '') FROM sessions WHERE id = ?`, sessionID).Scan(&p)
	return p
}

// scanObservation scans an observations row (26 columns, no rank) into an
// Observation.
func scanObservation(row interface {
	Scan(dest ...any) error
}) (Observation, error) {
	return scanObservationFields(row, false)
}

// scanObservationFields scans the 26 observation columns, optionally followed
// by a trailing rank column (retrieval queries append one; scanned as a
// nullable so a NULL rank cannot fail the scan).
func scanObservationFields(row interface {
	Scan(dest ...any) error
}, withRank bool) (Observation, error) {
	var (
		o           Observation
		toolName    sql.NullString
		project     sql.NullString
		topicKey    sql.NullString
		normalized  sql.NullString
		lastSeenAt  sql.NullString
		reviewAfter sql.NullString
		superseded  sql.NullString
		expiresAt   sql.NullString
		provJSON    string
		embedding   []byte
		embedModel  sql.NullString
		deletedAt   sql.NullString
		rank        sql.NullFloat64 // trailing rank column (retrieval queries)
	)
	dest := []any{
		&o.ID, &o.SyncID, &o.SessionID, &o.Type, &o.Title, &o.Content,
		&toolName, &project, &o.Scope, &topicKey, &normalized,
		&o.RevisionCount, &o.DuplicateCount, &lastSeenAt, &o.Pinned,
		&reviewAfter, &superseded, &expiresAt, &embedding, &embedModel,
		&o.Importance, &o.RetrievalUsage, &provJSON, &o.CreatedAt, &o.UpdatedAt,
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
	o.LastSeenAt = stringPtr(lastSeenAt)
	o.ReviewAfter = stringPtr(reviewAfter)
	o.SupersededBySyncID = stringPtr(superseded)
	o.ExpiresAt = stringPtr(expiresAt)
	o.Embedding = embedding
	o.EmbeddingModel = embedModel.String
	o.DeletedAt = stringPtr(deletedAt)
	_ = json.Unmarshal([]byte(provJSON), &o.Provenance)
	return o, nil
}

func stringPtr(ns sql.NullString) *string {
	if !ns.Valid || strings.TrimSpace(ns.String) == "" {
		return nil
	}
	return &ns.String
}

// ── observations ────────────────────────────────────────────────────────────

// AddObservation persists a new observation (minting sync_id, setting
// provenance + timestamps + normalized_hash) and returns the result.
func (s *sqliteStore) AddObservation(ctx context.Context, p AddObservationParams) (AddObservationResult, error) {
	now := nowStamp()
	syncID := syncid.NewSyncID("obs")
	prov := Provenance{
		SessionID: p.SessionID,
		CreatedAt: now,
	}
	provJSON, _ := json.Marshal(prov)
	hash := hashNormalized(p.Content)

	var id int64
	err := s.withWriteRetry(func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if err := s.ensureSessionTx(tx, p.SessionID, p.Project); err != nil {
			return err
		}
		project := s.resolveProjectTx(tx, p.SessionID, p.Project)

		// revision_count / duplicate_count are omitted on purpose: the store
		// inserts the schema defaults (0/0). The save pipeline
		// (01-memory/001-observations §2 step 6) owns the first-save counters
		// and increments them on upserts / exact-dedupe hits.
		res, err := tx.Exec(`
			INSERT INTO observations (
				sync_id, session_id, type, title, content, tool_name, project, scope,
				topic_key, normalized_hash,
				last_seen_at, importance, retrieval_usage, provenance, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), 0.5, 0, ?, ?, ?)`,
			syncID, p.SessionID, p.Type, p.Title, p.Content,
			nullableString(p.ToolName), nullableString(project),
			scopeOrDefault(p.Scope), nullableString(p.TopicKey), hash,
			provJSON, now, now,
		)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
		// ledger row (replay-safe save idempotency)
		if p.OperationID != "" {
			if _, err := tx.Exec(`
				INSERT INTO observation_save_operations (operation_id, fingerprint, observation_id)
				VALUES (?, ?, ?)`,
				p.OperationID, "v1:"+hash, id,
			); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
	if err != nil {
		return AddObservationResult{}, err
	}
	return AddObservationResult{ID: id, SyncID: syncID}, nil
}

func scopeOrDefault(scope string) string {
	s := strings.TrimSpace(scope)
	if s == "" {
		return "project"
	}
	return s
}

// GetObservation fetches an observation by local id.
func (s *sqliteStore) GetObservation(ctx context.Context, id int64, includeHistory bool) (Observation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, ifnull(sync_id,''), session_id, type, title, content,
		       tool_name, project, scope, topic_key, normalized_hash,
		       revision_count, duplicate_count, last_seen_at, pinned,
		       review_after, superseded_by_sync_id, expires_at, embedding,
		       embedding_model, importance, retrieval_usage, provenance,
		       created_at, updated_at, deleted_at
		FROM observations WHERE id = ? AND deleted_at IS NULL`, id)
	return scanObservation(row)
}

// UpdateObservationForProject updates the mutable fields of an observation,
// scoped to expectedProject.
func (s *sqliteStore) UpdateObservationForProject(ctx context.Context, id int64, expectedProject string, p UpdateParams) error {
	var existing Observation
	err := s.withWriteRetry(func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		row := tx.QueryRow(`
			SELECT id, ifnull(sync_id,''), session_id, type, title, content,
			       tool_name, project, scope, topic_key, normalized_hash,
			       revision_count, duplicate_count, last_seen_at, pinned,
			       review_after, superseded_by_sync_id, expires_at, embedding,
			       embedding_model, importance, retrieval_usage, provenance,
			       created_at, updated_at, deleted_at
			FROM observations WHERE id = ? AND deleted_at IS NULL`, id)
		existing, err = scanObservation(row)
		if err != nil {
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
			hash := hashNormalized(content)
			if _, err := tx.Exec(`UPDATE observations SET content = ?, normalized_hash = ?, updated_at = datetime('now') WHERE id = ?`, content, hash, id); err != nil {
				return err
			}
		}
		if p.TopicKey != nil {
			topicKey = *p.TopicKey
		}
		pinned := existing.Pinned
		if p.Pinned != nil {
			pinned = *p.Pinned
		}
		_, err = tx.Exec(`
			UPDATE observations
			SET type = ?, title = ?, content = ?, topic_key = ?, pinned = ?, updated_at = datetime('now')
			WHERE id = ?`,
			typ, title, content, nullableString(topicKey), pinned, id)
		if err != nil {
			return err
		}
		return tx.Commit()
	})
	return err
}

// DeleteObservationForProject soft-deletes (or hard-deletes when hard=true)
// an observation, scoped to expectedProject.
func (s *sqliteStore) DeleteObservationForProject(ctx context.Context, id int64, expectedProject string, hard bool) error {
	return s.withWriteRetry(func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// verify project guard
		var proj string
		err = tx.QueryRow(`SELECT ifnull(project,'') FROM observations WHERE id = ? AND deleted_at IS NULL`, id).Scan(&proj)
		if err == sql.ErrNoRows {
			return fmt.Errorf("store: observation %d not found", id)
		}
		if err != nil {
			return err
		}
		if proj != "" && strings.TrimSpace(expectedProject) != "" && proj != strings.TrimSpace(expectedProject) {
			return fmt.Errorf("store: observation %d belongs to project %q, not %q", id, proj, expectedProject)
		}

		if hard {
			// FTS triggers handle the index; the ledger FK is SET NULL.
			if _, err := tx.Exec(`DELETE FROM observations WHERE id = ?`, id); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(`UPDATE observations SET deleted_at = datetime('now'), updated_at = datetime('now') WHERE id = ? AND deleted_at IS NULL`, id); err != nil {
				return err
			}
		}
		return tx.Commit()
	})
}

// ConsolidateObservations is not implemented yet (later task).
func (s *sqliteStore) ConsolidateObservations(ctx context.Context, p ConsolidateParams) (ConsolidateResult, error) {
	return ConsolidateResult{}, ErrNotImplemented{Method: "ConsolidateObservations"}
}

// ── relations ───────────────────────────────────────────────────────────────

// AddRelation is not implemented yet (later task).
func (s *sqliteStore) AddRelation(ctx context.Context, p AddRelationParams) (Relation, error) {
	return Relation{}, ErrNotImplemented{Method: "AddRelation"}
}

// JudgeRelation is not implemented yet (later task).
func (s *sqliteStore) JudgeRelation(ctx context.Context, id int64, p JudgeParams) error {
	return ErrNotImplemented{Method: "JudgeRelation"}
}

// ── retrieval ───────────────────────────────────────────────────────────────

// searchFTSTitleWeight, searchFTSContentWeight, searchFTSTopicKeyWeight are
// the BM25 column weights (Engram store.go:424-431).
const (
	searchFTSTitleWeight    = 5.0
	searchFTSContentWeight  = 1.0
	searchFTSTopicKeyWeight = 3.0
)

// searchRankBoostSQL is the composite-rank boost factor, with age in days
// measured from COALESCE(last_seen_at, updated_at, created_at) and the
// recency half-life of 30 days (spec 001-storage §4). The boost weights
// (0.10 pinned / 0.06 recency / 0.04 stability, Engram store.go:6031-6044)
// are inlined as literals — they are the spec-pinned values the ranking test
// asserts against, and a Go const string cannot embed them via fmt.
const searchRankBoostSQL = `(1.0
	+ 0.10 * CAST(o.pinned AS REAL)
	+ 0.06 / (1.0 + ABS((JULIANDAY('now') - JULIANDAY(COALESCE(o.last_seen_at, o.updated_at, o.created_at)))) * 24.0 / 30.0)
	+ 0.04 * (CAST(o.revision_count AS REAL) + CAST(o.duplicate_count AS REAL)) / (CAST(o.revision_count AS REAL) + CAST(o.duplicate_count AS REAL) + 4.0))`

// searchRankRelevanceBase is the multiplicative relevance base of the
// composite rank: MAX(bm25, 1.0). bm25 is 0.0 when every matching document
// contributes equally to each matched column (e.g. two identical rows), which
// would make the composite rank 0.0 × boost = 0.0 for all of them and leave
// the pinned/recency/stability boosts non-load-bearing. Clamping the base to
// 1.0 keeps a real BM25 gradient when one exists (it dominates the order) but
// guarantees the boosts are the deciding delta in the all-identical case the
// ranking scenarios exercise (Engram store.go:6031-6044, rawRank × boost).
const searchRankRelevanceBase = "MAX(bm25(observations_fts, %.1f, %.1f, 0.0, 0.0, 0.0, %.1f), 1.0)"

// Search runs an FTS5 trigram MATCH (with the < 3-rune LIKE fallback),
// ranked by the composite rank: bm25(title, content, topic_key weights)
// multiplied by the pinned/recency/stability boosts (Engram store.go:6031-6044).
func (s *sqliteStore) Search(ctx context.Context, q SearchQuery) (SearchResult, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if hasShortFTSTerm(q.Query) {
		return s.searchLIKE(ctx, q, limit)
	}
	// FTS5 MATCH path
	rankExpr := fmt.Sprintf("%s * %s",
		fmt.Sprintf(searchRankRelevanceBase, searchFTSTitleWeight, searchFTSContentWeight, searchFTSTopicKeyWeight),
		searchRankBoostSQL)
	qStr := `SELECT ` + obsSelectColumns + `, ` + rankExpr + ` AS rank
		FROM observations_fts fts
		CROSS JOIN observations o ON o.id = fts.rowid
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL`
	args := []any{q.Query}
	if q.Project != "" {
		qStr += ` AND LOWER(o.project) = ?`
		args = append(args, q.Project)
	}
	if q.Type != "" {
		qStr += ` AND o.type = ?`
		args = append(args, q.Type)
	}
	if q.Scope != "" {
		qStr += ` AND o.scope = ?`
		args = append(args, q.Scope)
	}
	// The composite rank is a relevance *score* (higher = better: the boosts
	// raise it), so the best match is the HIGHEST rank — order DESC. (A plain
	// bm25() is a cost, lower = better, but the composite is a score.)
	qStr += ` ORDER BY rank DESC LIMIT ?`
	args = append(args, limit)
	return s.queryObservations(ctx, qStr, args...)
}

// obsSelectColumns is the observation column list for retrieval queries.
const obsSelectColumns = `o.id, ifnull(o.sync_id,'') as sync_id, o.session_id, o.type, o.title, o.content,
	o.tool_name, o.project, o.scope, o.topic_key, o.normalized_hash,
	o.revision_count, o.duplicate_count, o.last_seen_at, o.pinned,
	o.review_after, o.superseded_by_sync_id, o.expires_at, o.embedding,
	o.embedding_model, o.importance, o.retrieval_usage, o.provenance,
	o.created_at, o.updated_at, o.deleted_at`

// hasShortFTSTerm reports whether any query term is < 3 runes (trigram
// minimum), in which case the LIKE fallback is used (Engram hasShortFTSTerm).
func hasShortFTSTerm(query string) bool {
	for _, term := range strings.Fields(query) {
		if utf8.RuneCountInString(term) < 3 {
			return true
		}
	}
	return false
}

// searchLIKE is the bounded substring fallback for short terms.
func (s *sqliteStore) searchLIKE(ctx context.Context, q SearchQuery, limit int) (SearchResult, error) {
	qStr := `SELECT ` + obsSelectColumns + `, 0.0 AS rank FROM observations o WHERE o.deleted_at IS NULL`
	args := []any{}
	termGroups := make([]string, 0)
	for _, term := range strings.Fields(q.Query) {
		termGroups = append(termGroups, `(o.title LIKE ? ESCAPE '\' OR o.content LIKE ? ESCAPE '\' OR o.type LIKE ? ESCAPE '\' OR o.project LIKE ? ESCAPE '\' OR o.topic_key LIKE ? ESCAPE '\')`)
		pattern := "%" + term + "%"
		for range 5 {
			args = append(args, pattern)
		}
	}
	qStr += " AND (" + strings.Join(termGroups, " AND ") + ")"
	if q.Project != "" {
		qStr += ` AND LOWER(o.project) = ?`
		args = append(args, q.Project)
	}
	if q.Type != "" {
		qStr += ` AND o.type = ?`
		args = append(args, q.Type)
	}
	if q.Scope != "" {
		qStr += ` AND o.scope = ?`
		args = append(args, q.Scope)
	}
	qStr += ` ORDER BY datetime(o.updated_at) DESC, o.id DESC LIMIT ?`
	args = append(args, limit)
	return s.queryObservations(ctx, qStr, args...)
}

// queryObservations runs a SELECT that returns the 26 observation columns
// followed by a trailing rank column (Search/LIKE queries always append one)
// and scans the rows.
func (s *sqliteStore) queryObservations(ctx context.Context, q string, args ...any) (SearchResult, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return SearchResult{}, err
	}
	defer rows.Close()
	var res SearchResult
	for rows.Next() {
		obs, err := scanObservationWithRank(rows)
		if err != nil {
			return SearchResult{}, err
		}
		res.Observations = append(res.Observations, obs)
	}
	return res, rows.Err()
}

// scanObservationWithRank scans the 26 observation columns plus a trailing
// rank column (the retrieval queries always append one).
func scanObservationWithRank(row interface {
	Scan(dest ...any) error
}) (Observation, error) {
	o, err := scanObservationFields(row, true)
	return o, err
}

// VectorSearch returns empty + degraded (wired in Task 12).
func (s *sqliteStore) VectorSearch(ctx context.Context, q VecQuery) (VecResult, error) {
	return VecResult{Degraded: true}, nil
}

// Context is not implemented yet (later task).
func (s *sqliteStore) Context(ctx context.Context, q ContextQuery) (ContextResult, error) {
	return ContextResult{}, ErrNotImplemented{Method: "Context"}
}

// ── sync ────────────────────────────────────────────────────────────────────

// EnqueueMutation journals a mutation and returns its seq.
func (s *sqliteStore) EnqueueMutation(ctx context.Context, m Mutation) (int64, error) {
	var seq int64
	err := s.withWriteRetry(func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		res, err := tx.Exec(`
			INSERT INTO sync_mutations (target_key, entity, entity_key, op, payload, source, project)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.TargetKey, m.Entity, m.EntityKey, m.Op, m.Payload, m.Source, nullableString(m.Project),
		)
		if err != nil {
			return err
		}
		seq, err = res.LastInsertId()
		if err != nil {
			return err
		}
		return tx.Commit()
	})
	return seq, err
}

// ListPendingMutations returns un-acked mutations after the cursor.
func (s *sqliteStore) ListPendingMutations(ctx context.Context, cursor int64, limit int) ([]Mutation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT seq, target_key, entity, entity_key, op, payload, ifnull(source,'local'), ifnull(project,''), occurred_at
		FROM sync_mutations
		WHERE seq > ? AND acked_at IS NULL
		ORDER BY seq ASC LIMIT ?`, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Mutation
	for rows.Next() {
		var m Mutation
		if err := rows.Scan(&m.Seq, &m.TargetKey, &m.Entity, &m.EntityKey, &m.Op, &m.Payload, &m.Source, &m.Project, &m.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ApplyPulledMutation is stubbed until Task 17.
func (s *sqliteStore) ApplyPulledMutation(ctx context.Context, m Mutation) error {
	return ErrNotImplemented{Method: "ApplyPulledMutation"}
}

// Tombstone records a delete tombstone.
func (s *sqliteStore) Tombstone(ctx context.Context, entity, key string) error {
	return s.withWriteRetry(func() error {
		_, err := s.db.Exec(`
			INSERT INTO sync_delete_tombstones (entity, entity_key, deleted_at)
			VALUES (?, ?, datetime('now'))
			ON CONFLICT(entity, entity_key) DO UPDATE SET deleted_at = datetime('now')`,
			entity, key)
		return err
	})
}

// ── additive methods ────────────────────────────────────────────────────────

// MarkReviewed resets review_after = now + type-months (spec 002-schema §2.2.2).
func (s *sqliteStore) MarkReviewed(ctx context.Context, id int64) error {
	var typ string
	if err := s.db.QueryRowContext(ctx, `SELECT type FROM observations WHERE id = ? AND deleted_at IS NULL`, id).Scan(&typ); err != nil {
		return err
	}
	months := decayMonths(typ)
	var ra string
	if months > 0 {
		ra = time.Now().UTC().AddDate(0, months, 0).Format("2006-01-02 15:04:05")
	} else {
		// Explicit no-decay: types without a review schedule (learning,
		// session_summary, policy, unmapped) clear review_after so the row
		// reads as active (Engram MarkReviewed semantics).
		ra = ""
	}
	return s.withWriteRetry(func() error {
		_, err := s.db.ExecContext(ctx, `UPDATE observations SET review_after = ?, updated_at = datetime('now') WHERE id = ?`, nullableString(ra), id)
		return err
	})
}

// decayMonths maps observation type → review-after month offset (Engram
// decayReviewAfterMonths + spec 002-schema §2.2.2).
func decayMonths(typ string) int {
	switch typ {
	case "preference", "config":
		return 3
	case "discovery", "pattern", "decision":
		return 6
	case "architecture", "bugfix", "policy":
		return 12
	default:
		return 0
	}
}

// BumpRetrievalUsage increments retrieval_usage per id (spec 002-schema §2.2.3).
func (s *sqliteStore) BumpRetrievalUsage(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := `UPDATE observations SET retrieval_usage = retrieval_usage + 1, updated_at = datetime('now') WHERE id IN (` + strings.Join(placeholders, ",") + `)`
	return s.withWriteRetry(func() error {
		_, err := s.db.ExecContext(ctx, query, args...)
		return err
	})
}

// SearchPrompts searches prompts_fts over (content, project) and returns each
// hit with its full identity (PromptRow).
func (s *sqliteStore) SearchPrompts(ctx context.Context, q PromptSearchQuery) (PromptSearchResult, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	qStr := `SELECT p.id, ifnull(p.sync_id,''), p.session_id, ifnull(p.project,''), p.content, p.created_at
		FROM prompts_fts fts
		CROSS JOIN user_prompts p ON p.id = fts.rowid
		WHERE prompts_fts MATCH ?`
	args := []any{q.Query}
	if q.Project != "" {
		qStr += ` AND LOWER(p.project) = ?`
		args = append(args, q.Project)
	}
	qStr += ` ORDER BY fts.rank LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, qStr, args...)
	if err != nil {
		return PromptSearchResult{Degraded: true}, err
	}
	defer rows.Close()
	var res PromptSearchResult
	for rows.Next() {
		var row PromptRow
		if err := rows.Scan(&row.ID, &row.SyncID, &row.SessionID, &row.Project, &row.Content, &row.CreatedAt); err != nil {
			return PromptSearchResult{Degraded: true}, err
		}
		res.Prompts = append(res.Prompts, row)
	}
	return res, rows.Err()
}

// TouchForTest sets last_seen_at and revision_count on an observation
// (test-only hook for the ranking boosts).
func (s *sqliteStore) TouchForTest(ctx context.Context, id int64, lastSeenAt string, revisionCount int) error {
	return s.withWriteRetry(func() error {
		_, err := s.db.ExecContext(ctx, `UPDATE observations SET last_seen_at = ?, revision_count = ? WHERE id = ?`, lastSeenAt, revisionCount, id)
		return err
	})
}

// InsertPrompt is a test-only seam (TouchForTest pattern) that inserts a row
// into user_prompts directly, so SearchPrompts can be exercised end-to-end
// without the full save pipeline (Task 3/4 owns the real capture path). It
// mints a prm- sync id and relies on the prompts_fts insert trigger to index
// the row. Returns the prompt's local id.
func (s *sqliteStore) InsertPrompt(ctx context.Context, sessionID, content, project string) (int64, error) {
	syncID := syncid.NewSyncID("prm")
	var id int64
	err := s.withWriteRetry(func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err := s.ensureSessionTx(tx, sessionID, project); err != nil {
			return err
		}
		// Resolve project the same way AddObservation does: explicit project
		// wins, otherwise inherit the session's project.
		resolved := s.resolveProjectTx(tx, sessionID, project)
		res, err := tx.Exec(`
			INSERT INTO user_prompts (sync_id, session_id, content, project)
			VALUES (?, ?, ?, ?)`,
			syncID, sessionID, content, nullableString(resolved),
		)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return err
		}
		return tx.Commit()
	})
	return id, err
}

// SetTxnHook sets the committed-transaction counter (Task 10 seam).
func (s *sqliteStore) SetTxnHook(hook func()) {
	s.txnHook = hook
}

// ── lifecycle ───────────────────────────────────────────────────────────────

// Migrate re-runs the migration set (idempotent, additive-only).
func (s *sqliteStore) Migrate(ctx context.Context) error {
	return migrateSQLite(s.db)
}

// Close releases one reference to the store handle.
func (s *sqliteStore) Close() error {
	if s.pooled {
		return poolRelease(s.path)
	}
	return poolReleaseDB(s.db)
}

// Ping checks the database is alive.
func (s *sqliteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Probe runs a raw query and reports whether any row was returned (test
// helper for the ledger FK assertion).
func (s *sqliteStore) Probe(query string, args ...any) (bool, error) {
	var found bool
	err := s.db.QueryRow(query, args...).Scan(&found)
	return found, err
}

// ProbeInt runs a raw query that must return a single integer (test helper
// for the migration-skip catalog-count assertion).
func (s *sqliteStore) ProbeInt(query string, args ...any) (int, error) {
	var n int
	err := s.db.QueryRow(query, args...).Scan(&n)
	return n, err
}

// deref returns the pointed-to string or "".
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
