package store

// postgres_code.go — the pgStore code-index render (TICKET-08c, M4). It
// mirrors sqlite.go's code-index render with the dialect-specific FTS/prefix
// rendering: the tsvector MATCH (the 'simple' config, the trigram analog)
// replaces FTS5 MATCH, and the ILIKE fallback (the memory change's
// searchLIKEPG pattern) replaces SQLite's LIKE for the < 3-rune case. The
// ON CONFLICT upserts carry over (PG supports them). The vector methods
// write the pgvector BLOB ($1::vector) — no ANN tables on PG (the PG floor
// is brute-force, TICKET-08a's degraded: brute_force).

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/syncid"
)

// pgSymbolColumns is the read projection the PG walker scans (the dialect
// analog of sqlite.go's symbolColumns — PG uses COALESCE, not ifnull, and the
// timestamps scan as *time.Time, not TEXT, so the FileRow scans are
// dialect-specific below).
const pgSymbolColumns = `id, COALESCE(sync_id,''), name, COALESCE(qualified_name,''),
	COALESCE(kind,''), COALESCE(language,''), COALESCE(signature,''),
	start_line, end_line, uid, file_id`

// pgEdgeColumns is the edge read projection (the dialect analog of
// sqlite.go's edgeColumns).
const pgEdgeColumns = `id, kind, from_id, to_id, COALESCE(to_name,''),
	file_id, COALESCE(confidence,'EXTRACTED'), line, COALESCE(context,'')`

// ── code-index write surface ────────────────────────────────────────────────

// UpsertFile upserts a files row by path (the dialect render of the SQLite
// UpsertFile — ON CONFLICT carries over; sync_id + indexed_at are store-owned,
// INV-6 + provenance). The $n placeholders are the PG driver's.
func (s *pgStore) UpsertFile(ctx context.Context, f FileRow) (int64, error) {
	return s.upsertFilePG(ctx, s.db, f, syncid.NewSyncID("file"))
}

func (s *pgStore) upsertFilePG(ctx context.Context, q executor, f FileRow, syncID string) (int64, error) {
	now := f.IndexedAt
	if now == "" {
		now = pgNow()
	}
	var id int64
	err := q.(interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}).QueryRowContext(ctx, `
		INSERT INTO `+s.qualified("files")+`
			(sync_id, path, mtime_ns, size, content_hash, language, ast_hash, indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(path) DO UPDATE SET
			mtime_ns = excluded.mtime_ns,
			size = excluded.size,
			content_hash = excluded.content_hash,
			language = excluded.language,
			ast_hash = excluded.ast_hash,
			indexed_at = excluded.indexed_at
		RETURNING id`,
		syncID, f.Path, f.MtimeNS, f.Size, notNullOrEmpty(f.ContentHash),
		f.Language, notNullOrEmpty(f.ASTHash), now,
	).Scan(&id)
	return id, err
}

// UpsertSymbols upserts the file's symbols ON CONFLICT(uid) (CI-3 local
// upsert key; sync_id minted on first insert, INV-6, preserved on conflict).
func (s *pgStore) UpsertSymbols(ctx context.Context, fileID int64, syms []Symbol) (int64, error) {
	return s.upsertSymbolsPG(ctx, s.db, fileID, syms)
}

func (s *pgStore) upsertSymbolsPG(ctx context.Context, q executor, fileID int64, syms []Symbol) (int64, error) {
	var count int64
	for _, sym := range syms {
		syncID := syncid.NewSyncID("sym")
		if _, err := q.ExecContext(ctx, `
			INSERT INTO `+s.qualified("symbols")+` (
				sync_id, file_id, name, qualified_name, kind, language,
				signature, start_line, end_line, visibility, is_exported, return_type,
				param_types, content_hash, uid
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			ON CONFLICT(uid) DO UPDATE SET
				file_id = excluded.file_id,
				qualified_name = excluded.qualified_name,
				language = excluded.language,
				signature = excluded.signature,
				start_line = excluded.start_line,
				end_line = excluded.end_line,
				visibility = excluded.visibility,
				is_exported = excluded.is_exported,
				return_type = excluded.return_type,
				param_types = excluded.param_types,
				content_hash = excluded.content_hash`,
			syncID, fileID, sym.Name, nullableOrEmpty(sym.QualifiedName), sym.Kind,
			nullableOrEmpty(sym.Language), nullableOrEmpty(sym.Signature),
			sym.StartLine, sym.EndLine, nullableOrEmpty(sym.Visibility),
			sym.IsExported, nullableOrEmpty(sym.ReturnType),
			nullableOrEmpty(sym.ParamTypes), sym.ContentHash, sym.UID,
		); err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

// ReplaceEdgesForFile is the target-state edge write (spec 002 §9): DELETE
// the file's rows, then upsert its fresh edge set. Empty set = clear.
func (s *pgStore) ReplaceEdgesForFile(ctx context.Context, fileID int64, edges []Edge) error {
	return s.replaceEdgesForFilePG(ctx, s.db, fileID, edges)
}

func (s *pgStore) replaceEdgesForFilePG(ctx context.Context, q executor, fileID int64, edges []Edge) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM `+s.qualified("edges")+` WHERE file_id = $1`, fileID); err != nil {
		return err
	}
	for _, e := range edges {
		fromID := e.FromID
		if fromID == 0 {
			fromID = fileID
		}
		fid := e.FileID
		if fid == 0 {
			fid = fileID
		}
		if _, err := q.ExecContext(ctx, `
			INSERT INTO `+s.qualified("edges")+` (kind, from_id, to_id, to_name, file_id, confidence, line, context)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, line) DO NOTHING`,
			e.Kind, fromID, e.ToID, nullableOrEmpty(e.ToName), fid,
			edgeConfidence(e), nullableInt(e.Line), e.Context,
		); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceUnresolvedForFile is the target-state unresolved ledger write
// (spec 001 §5): delete the file's rows, then re-insert the fresh ledger.
// Empty set = clear.
func (s *pgStore) ReplaceUnresolvedForFile(ctx context.Context, fileID int64, refs []UnresolvedRef) error {
	return s.replaceUnresolvedForFilePG(ctx, s.db, fileID, refs)
}

func (s *pgStore) replaceUnresolvedForFilePG(ctx context.Context, q executor, fileID int64, refs []UnresolvedRef) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM `+s.qualified("unresolved_edges")+` WHERE file_id = $1`, fileID); err != nil {
		return err
	}
	now := pgNow()
	for _, r := range refs {
		if _, err := q.ExecContext(ctx, `
			INSERT INTO `+s.qualified("unresolved_edges")+` (file_id, reference, kind, line, candidates, status, first_seen, last_seen)
			VALUES ($1, $2, $3, $4, $5, 'open', $6, $7)`,
			fileID, r.Reference, r.Kind, nullableInt(r.Line),
			nullableOrEmpty(r.Candidates), now, now,
		); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceChunksForFile is the target-state chunk write (spec 001 §3): DELETE
// the file's chunks (the tsvector trigger desyncs the index), then insert
// the fresh chunk set (the trigger re-indexes it). Empty set = clear.
func (s *pgStore) ReplaceChunksForFile(ctx context.Context, fileID int64, chunks []Chunk) error {
	return s.replaceChunksForFilePG(ctx, s.db, fileID, chunks)
}

func (s *pgStore) replaceChunksForFilePG(ctx context.Context, q executor, fileID int64, chunks []Chunk) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM `+s.qualified("chunks")+` WHERE file_id = $1`, fileID); err != nil {
		return err
	}
	for _, c := range chunks {
		if _, err := q.ExecContext(ctx, `
			INSERT INTO `+s.qualified("chunks")+` (file_id, kind, start_line, end_line, text, content_hash)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			fileID, chunkKind(c), c.StartLine, c.EndLine, c.Text, c.ContentHash,
		); err != nil {
			return err
		}
	}
	return nil
}

// DeleteFile removes the files row; the FK cascades take its chunks,
// symbols, edges, and unresolved rows (spec 001 §6 / 002 §9, M10).
func (s *pgStore) DeleteFile(ctx context.Context, path string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM `+s.qualified("files")+` WHERE path = $1`, path)
	return err
}

// PruneOrphanSymbols deletes symbols whose uid is not in the declared target
// set (spec 002 §9 — the file changed, so its old symbols are gone; the FK
// cascade takes their edges).
func (s *pgStore) PruneOrphanSymbols(ctx context.Context, targetUIDs []string) error {
	if len(targetUIDs) == 0 {
		_, err := s.db.ExecContext(ctx, `DELETE FROM `+s.qualified("symbols"))
		return err
	}
	ph := make([]string, len(targetUIDs))
	args := make([]any, len(targetUIDs))
	for i, u := range targetUIDs {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = u
	}
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM `+s.qualified("symbols")+` WHERE uid NOT IN (`+strings.Join(ph, ",")+`)`, args...)
	return err
}

// UpdateIndexFreshness upserts the one freshness record (spec 001 §4).
// capabilities / extractor_tiers are JSONB on PG — a nil value becomes NULL
// (extractor_tiers) or the empty-object default (capabilities is NOT NULL).
func (s *pgStore) UpdateIndexFreshness(ctx context.Context, f FreshnessRecord) error {
	updated := f.UpdatedAt
	if updated == "" {
		updated = pgNow()
	}
	last := nullableOrEmpty(derefString(f.LastIndexedAt))
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+s.qualified("index_freshness")+` (
			project, extractor_stamp, schema_fingerprint, last_indexed_at,
			files_indexed, files_deleted, files_oversized,
			symbols_count, edges_count, chunks_count, unresolved_count,
			coverage_fair, capabilities, extractor_tiers, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT(project) DO UPDATE SET
			extractor_stamp = excluded.extractor_stamp,
			schema_fingerprint = excluded.schema_fingerprint,
			last_indexed_at = excluded.last_indexed_at,
			files_indexed = excluded.files_indexed,
			files_deleted = excluded.files_deleted,
			files_oversized = excluded.files_oversized,
			symbols_count = excluded.symbols_count,
			edges_count = excluded.edges_count,
			chunks_count = excluded.chunks_count,
			unresolved_count = excluded.unresolved_count,
			coverage_fair = excluded.coverage_fair,
			capabilities = excluded.capabilities,
			extractor_tiers = excluded.extractor_tiers,
			updated_at = excluded.updated_at`,
		f.Project, f.ExtractorStamp, f.SchemaFingerprint, last,
		f.FilesIndexed, f.FilesDeleted, f.FilesOversized,
		f.SymbolsCount, f.EdgesCount, f.ChunksCount, f.UnresolvedCount,
		f.CoverageFair, jsonbOrEmpty(f.Capabilities), jsonbOrNull(f.ExtractorTiers), updated,
	)
	return err
}

// jsonbOrEmpty renders a JSONB value for the NOT NULL capabilities column
// (an empty/nil value becomes the empty object, the column's default).
func jsonbOrEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return "{}"
	}
	return s
}

// jsonbOrNull renders a JSONB value for the nullable extractor_tiers column
// (an empty/nil value becomes NULL).
func jsonbOrNull(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// ── code-index graph reads (the dialect-agnostic walker's surface) ──────────

// EdgesByFrom returns the edges leaving fromID (the walker's forward view).
func (s *pgStore) EdgesByFrom(ctx context.Context, fromID int64, kinds []string) ([]EdgeRow, error) {
	qStr := `SELECT ` + pgEdgeColumns + ` FROM ` + s.qualified("edges") + ` WHERE from_id = $1`
	args := []any{fromID}
	ks, ka := pgKindsWhere(kinds, 2)
	qStr += ks
	args = append(args, ka...)
	return s.queryEdgeRowsPG(ctx, qStr, args...)
}

// EdgesByTo returns the edges entering toID (the walker's reverse view).
func (s *pgStore) EdgesByTo(ctx context.Context, toID int64, kinds []string) ([]EdgeRow, error) {
	qStr := `SELECT ` + pgEdgeColumns + ` FROM ` + s.qualified("edges") + ` WHERE to_id = $1`
	args := []any{toID}
	ks, ka := pgKindsWhere(kinds, 2)
	qStr += ks
	args = append(args, ka...)
	return s.queryEdgeRowsPG(ctx, qStr, args...)
}

// EdgesByNameOnlyToName returns the name-only (unresolved) edges whose
// to_name matches (CI-INV-3 — the query-time name-only resolution surface).
func (s *pgStore) EdgesByNameOnlyToName(ctx context.Context, toName string, kinds []string) ([]EdgeRow, error) {
	qStr := `SELECT ` + pgEdgeColumns + ` FROM ` + s.qualified("edges") + ` WHERE to_id IS NULL AND to_name = $1`
	args := []any{toName}
	ks, ka := pgKindsWhere(kinds, 2)
	qStr += ks
	args = append(args, ka...)
	return s.queryEdgeRowsPG(ctx, qStr, args...)
}

// pgKindsWhere builds the optional edge-kind filter with $n placeholders
// (startN is the next free placeholder index).
func pgKindsWhere(kinds []string, startN int) (string, []any) {
	if len(kinds) == 0 {
		return "", nil
	}
	ph := make([]string, len(kinds))
	args := make([]any, len(kinds))
	for i, k := range kinds {
		ph[i] = fmt.Sprintf("$%d", startN+i)
		args[i] = k
	}
	return " AND kind IN (" + strings.Join(ph, ",") + ")", args
}

// queryEdgeRowsPG runs an pgEdgeColumns SELECT and scans the rows.
func (s *pgStore) queryEdgeRowsPG(ctx context.Context, q string, args ...any) ([]EdgeRow, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EdgeRow
	for rows.Next() {
		e, err := scanEdgeRowPG(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// scanEdgeRowPG scans the pgEdgeColumns projection (the PG dialect's
// NULL-able columns come back as sql.Null* values).
func scanEdgeRowPG(row interface {
	Scan(dest ...any) error
}) (EdgeRow, error) {
	var (
		r               EdgeRow
		to              sql.NullInt64
		name, conf, ctx sql.NullString
		line            sql.NullInt64
	)
	if err := row.Scan(&r.ID, &r.Kind, &r.FromID, &to, &name, &r.FileID,
		&conf, &line, &ctx); err != nil {
		return r, err
	}
	if to.Valid {
		r.ToID = &to.Int64
	}
	r.ToName = name.String
	r.Confidence = conf.String
	if line.Valid {
		r.Line = int(line.Int64)
	}
	r.Context = ctx.String
	return r, nil
}

// SymbolByName returns every symbol named name (the name floor; the
// query-time name-only resolution + code_resolve surface).
func (s *pgStore) SymbolByName(ctx context.Context, name string) ([]SymbolRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+pgSymbolColumns+` FROM `+s.qualified("symbols")+` WHERE name = $1 ORDER BY id`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SymbolRow
	for rows.Next() {
		r, err := scanSymbolRowPG(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SymbolByUID returns one symbol by its local upsert key (ErrNoRows when
// absent).
func (s *pgStore) SymbolByUID(ctx context.Context, uid string) (SymbolRow, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+pgSymbolColumns+` FROM `+s.qualified("symbols")+` WHERE uid = $1`, uid)
	return scanSymbolRowPG(row)
}

// SymbolByID returns one symbol by its local ID (ErrNoRows when absent).
func (s *pgStore) SymbolByID(ctx context.Context, id int64) (SymbolRow, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+pgSymbolColumns+` FROM `+s.qualified("symbols")+` WHERE id = $1`, id)
	return scanSymbolRowPG(row)
}

// scanSymbolRowPG scans the pgSymbolColumns projection.
func scanSymbolRowPG(row interface {
	Scan(dest ...any) error
}) (SymbolRow, error) {
	var (
		r                   SymbolRow
		qn, kind, lang, sig sql.NullString
	)
	if err := row.Scan(&r.ID, &r.SyncID, &r.Name, &qn, &kind, &lang, &sig,
		&r.StartLine, &r.EndLine, &r.UID, &r.FileID); err != nil {
		return r, err
	}
	r.QualifiedName = qn.String
	r.Kind = kind.String
	r.Language = lang.String
	r.Signature = sig.String
	return r, nil
}

// SymbolsByFile returns all symbols in a file, ordered by start_line.
func (s *pgStore) SymbolsByFile(ctx context.Context, fileID int64) ([]SymbolRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+pgSymbolColumns+` FROM `+s.qualified("symbols")+` WHERE file_id = $1 ORDER BY start_line`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SymbolRow
	for rows.Next() {
		r, err := scanSymbolRowPG(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// FileByPath returns one files row by its stable key (the watch fingerprint
// re-check + the door-check surface). The TIMESTAMPTZ indexed_at scans as a
// time.Time, rendered to the store's fixed-width format (the lag derivation
// compares these as strings — lexicographic == chronological).
func (s *pgStore) FileByPath(ctx context.Context, path string) (FileRow, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(sync_id,''), path, mtime_ns, size, content_hash,
		       COALESCE(language,''), COALESCE(ast_hash,''), indexed_at
		FROM `+s.qualified("files")+` WHERE path = $1`, path)
	return scanFileRowPG(row)
}

// FileByID returns one files row by its local ID (ErrNoRows when absent).
func (s *pgStore) FileByID(ctx context.Context, id int64) (FileRow, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(sync_id,''), path, mtime_ns, size, content_hash,
		       COALESCE(language,''), COALESCE(ast_hash,''), indexed_at
		FROM `+s.qualified("files")+` WHERE id = $1`, id)
	return scanFileRowPG(row)
}

// scanFileRowPG scans a files row (the TIMESTAMPTZ indexed_at → the store's
// fixed-width text format).
func scanFileRowPG(row interface {
	Scan(dest ...any) error
}) (FileRow, error) {
	var (
		f          FileRow
		astHash    sql.NullString
		indexedAt  sql.NullTime
	)
	if err := row.Scan(&f.ID, &f.SyncID, &f.Path, &f.MtimeNS, &f.Size, &f.ContentHash,
		&f.Language, &astHash, &indexedAt); err != nil {
		return f, err
	}
	f.ASTHash = astHash.String
	if indexedAt.Valid {
		f.IndexedAt = indexedAt.Time.UTC().Format("2006-01-02 15:04:05")
	}
	return f, nil
}

// FilesByLanguage returns all files with the given language, ordered by path.
func (s *pgStore) FilesByLanguage(ctx context.Context, lang string) ([]FileRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(sync_id,''), path, mtime_ns, size, content_hash,
		       COALESCE(language,''), COALESCE(ast_hash,''), indexed_at
		FROM `+s.qualified("files")+` WHERE language = $1 ORDER BY path`, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileRow
	for rows.Next() {
		f, err := scanFileRowPG(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ── code-index vectors (the pgvector BLOB) ──────────────────────────────────

// pgVectorLiteral renders a vector as the pgvector text literal the driver
// casts via $n::vector (the VectorSearch pattern, the text array-of-floats).
func pgVectorLiteral(v []float32) string {
	lit := make([]string, len(v))
	for i, f := range v {
		lit[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(lit, ",") + "]"
}

// EmbedSymbol stores a symbol's vector in the embeddings pgvector BLOB table
// (spec 006 §4.1 — the BLOB is the source of truth; the ANN render is
// TICKET-08a). The $1::vector cast is the pgvector write.
func (s *pgStore) EmbedSymbol(ctx context.Context, symbolID int64, model string, vec []float32) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+s.qualified("embeddings")+` (symbol_id, model, dim, vector, language, updated_at)
		VALUES ($1, $2, $3, $4::vector, $5, $6)
		ON CONFLICT(symbol_id) DO UPDATE SET
			model = excluded.model,
			dim = excluded.dim,
			vector = excluded.vector,
			updated_at = excluded.updated_at`,
		symbolID, model, len(vec), pgVectorLiteral(vec), nil, pgNow(),
	)
	return err
}

// EmbedChunk stores a chunk's vector in the chunk_embeddings pgvector BLOB
// table.
func (s *pgStore) EmbedChunk(ctx context.Context, chunkID int64, model string, vec []float32) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+s.qualified("chunk_embeddings")+` (chunk_id, model, dim, vector, language, updated_at)
		VALUES ($1, $2, $3, $4::vector, $5, $6)
		ON CONFLICT(chunk_id) DO UPDATE SET
			model = excluded.model,
			dim = excluded.dim,
			vector = excluded.vector,
			updated_at = excluded.updated_at`,
		chunkID, model, len(vec), pgVectorLiteral(vec), nil, pgNow(),
	)
	return err
}

// ClearVectors empties both pgvector BLOB tables (the model-swap guard's
// DELETE, spec 006 §4.2 — a model change makes every stored vector
// non-comparable).
func (s *pgStore) ClearVectors(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM `+s.qualified("embeddings")); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM `+s.qualified("chunk_embeddings"))
	return err
}

// ── code-index trail (the shared retrieval_trails table, migration 003) ────

// CountRetrievals counts the retrieval_trails rows for a project.
func (s *pgStore) CountRetrievals(ctx context.Context, project string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM `+s.qualified("retrieval_trails")+` WHERE project = $1`, project).Scan(&count)
	return count, err
}

// RecordRetrieval appends one row to the shared retrieval_trails table
// (the memory change owns the table — migration 003; T15/INV-5).
func (s *pgStore) RecordRetrieval(ctx context.Context, t RetrievalTrail) error {
	ts := t.Timestamp
	if ts == "" {
		ts = pgNow()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+s.qualified("retrieval_trails")+` (route, query, project, filters, ids, degraded, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t.Route, t.Query, nullableOrEmpty(t.Project),
		nullableOrEmpty(t.Filters), nullableOrEmpty(t.IDs),
		t.Degraded, ts,
	)
	return err
}

// ── code-index seams ────────────────────────────────────────────────────────

// ExecDDL executes DDL statements in order (T21 — the table-adding seam the
// KB's KBStore uses; each statement is split internally so a multi-statement
// blob is carried). The PG dialect's splitter is dollar-quote-aware (the
// trigger bodies). The statements run OUTSIDE an explicit transaction (the
// SQLite leg's rationale — DDL durability per statement).
func (s *pgStore) ExecDDL(ctx context.Context, statements []string) error {
	for _, stmt := range statements {
		for _, sub := range splitPGStatements(stmt) {
			if sub == "" {
				continue
			}
			if _, err := s.db.ExecContext(ctx, sub); err != nil {
				return err
			}
		}
	}
	return nil
}

// Begin opens the per-file transaction (K1 — the idempotency claim rests on
// it: one file's write all commits or all rolls back). lib/pq's *sql.Tx is
// the tx surface (the same pattern as the memory change's PG tx surface).
func (s *pgStore) Begin(ctx context.Context) (Txn, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &pgCodeTxn{st: s, tx: tx}, nil
}

// pgCodeTxn carries the per-file code-index writes over one open transaction
// (K1). Commit settles it; any error path must Rollback.
type pgCodeTxn struct {
	st *pgStore
	tx *sql.Tx
}

// UpsertFile is the Txn twin of UpsertFile (inside the per-file tx).
func (t *pgCodeTxn) UpsertFile(ctx context.Context, f FileRow) (int64, error) {
	return t.st.upsertFilePG(ctx, t.tx, f, syncid.NewSyncID("file"))
}

// UpsertSymbols is the Txn twin of UpsertSymbols.
func (t *pgCodeTxn) UpsertSymbols(ctx context.Context, fileID int64, syms []Symbol) (int64, error) {
	return t.st.upsertSymbolsPG(ctx, t.tx, fileID, syms)
}

// ReplaceEdgesForFile is the Txn twin of ReplaceEdgesForFile.
func (t *pgCodeTxn) ReplaceEdgesForFile(ctx context.Context, fileID int64, edges []Edge) error {
	return t.st.replaceEdgesForFilePG(ctx, t.tx, fileID, edges)
}

// ReplaceUnresolvedForFile is the Txn twin of ReplaceUnresolvedForFile.
func (t *pgCodeTxn) ReplaceUnresolvedForFile(ctx context.Context, fileID int64, refs []UnresolvedRef) error {
	return t.st.replaceUnresolvedForFilePG(ctx, t.tx, fileID, refs)
}

// ReplaceChunksForFile is the Txn twin of ReplaceChunksForFile.
func (t *pgCodeTxn) ReplaceChunksForFile(ctx context.Context, fileID int64, chunks []Chunk) error {
	return t.st.replaceChunksForFilePG(ctx, t.tx, fileID, chunks)
}

// Commit settles the per-file tx (and fires the committed-transaction hook,
// the memory change's discipline).
func (t *pgCodeTxn) Commit(ctx context.Context) error {
	err := t.tx.Commit()
	if err == nil && t.st.txnHook != nil {
		t.st.txnHook()
	}
	return err
}

// Rollback undoes the per-file tx.
func (t *pgCodeTxn) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

