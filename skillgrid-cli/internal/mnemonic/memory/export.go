package memory

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// COGX-inspired portable JSON export (change 014, step 11). The bundle is the
// unit of portability: a self-describing JSON document of observations, graph
// edges, and symbol embeddings, using only JSON primitives (number, string,
// base64 string, RFC3339 string) so it imports cleanly into any other
// per-project store on any machine.

// ExportBundle is the full export of one project store.
type ExportBundle struct {
	// Format is the COGX-inspired schema version tag.
	Format       string            `json:"format"`
	Project      string            `json:"project"`
	ExportedAt   string            `json:"exported_at"`
	Observations []ExportRecord    `json:"observations"`
	GraphEdges   []EdgeRecord      `json:"graph_edges"`
	Embeddings   []EmbeddingRecord `json:"embeddings"`
}

// ExportRecord is one observation in the portable format. The embeddings map
// is keyed by symbol id (the observation's graph_ref bridge plus, for an
// observation that has no graph binding, none) — values are the base64-encoded
// little-endian float32 vectors from the embeddings table.
type ExportRecord struct {
	ID         int64             `json:"id"`
	Type       string            `json:"type"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
	Embeddings map[string]string `json:"embeddings,omitempty"`
	GraphRef   *int64            `json:"graph_ref,omitempty"`
}

// EdgeRecord is one codeindex graph edge with its temporal window
// (0 = active / NULL).
type EdgeRecord struct {
	ID          int64    `json:"id"`
	Kind        string   `json:"kind"`
	FromID      int64    `json:"from_id"`
	FileID      *int64   `json:"file_id,omitempty"`
	ToID        *int64   `json:"to_id,omitempty"`
	ToName      string   `json:"to_name,omitempty"`
	TargetPath  *string  `json:"target_path,omitempty"`
	Confidence  string   `json:"confidence"`
	Line        int      `json:"line"`
	ValidFrom   int64    `json:"valid_from"`
	ValidTo     *int64   `json:"valid_to,omitempty"`
}

// EmbeddingRecord is one symbol embedding from the bridge table; Vector is
// base64-encoded little-endian float32 bytes.
type EmbeddingRecord struct {
	SymbolID  int64  `json:"symbol_id"`
	Model     string `json:"model"`
	Dim       int    `json:"dim"`
	Vector    string `json:"vector"`
	UpdatedAt string `json:"updated_at"`
}

// ExportProject reads every observation (with its graph_ref), every codeindex
// graph edge, and every symbol embedding from this service's store and returns
// them as a self-describing bundle. It is read-only: the source store is never
// mutated. Embedding vectors are base64-encoded (the embeddings BLOB is
// little-endian float32 per the step 07 convention).
func (s *Service) ExportProject(ctx context.Context) (*ExportBundle, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	db := s.store.DB
	bundle := &ExportBundle{
		Format:     "cogx-v1",
		Project:    s.projectID,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Observations: every live row for this project, plus the base64 embedding
	// for its graph_ref symbol (the triple-store cross-link, step 07).
	rows, err := db.QueryContext(ctx, `
		SELECT o.id, o.type, o.content, o.title, o.session_id, o.project, o.scope,
		       o.topic_key, o.source, o.normalized_hash, o.revision_count, o.prompt_id,
		       o.created_at, o.updated_at, o.pinned, o.duplicate_count, o.last_seen_at,
		       o.expires_at, o.tool_name, o.owner, o.visibility, o.status, o.retrieval_usage,
		       o.graph_ref
		FROM observations o
		WHERE o.deleted_at IS NULL AND o.project = ?
		ORDER BY o.id`, s.projectID)
	if err != nil {
		return nil, fmt.Errorf("export observations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			rec           ExportRecord
			title         string
			sessID        string
			projectCol    string
			scope         string
			topicKey      sql.NullString
			source, hash  sql.NullString
			promptID      sql.NullInt64
			revision      int
			pinned, dups  int
			lastSeen, expires, toolName, owner, visibility, status sql.NullString
			createdAt, updatedAt sql.NullString
			usage         int
			graphRef      sql.NullInt64
		)
		if err := rows.Scan(
			&rec.ID, &rec.Type, &rec.Content, &title, &sessID, &projectCol, &scope,
			&topicKey, &source, &hash, &revision, &promptID,
			&createdAt, &updatedAt, &pinned, &dups, &lastSeen, &expires, &toolName,
			&owner, &visibility, &status, &usage, &graphRef,
		); err != nil {
			return nil, fmt.Errorf("scan export observation: %w", err)
		}
		rec.Metadata = map[string]string{
			"title":           title,
			"session_id":      sessID,
			"project":         projectCol,
			"scope":           scope,
			"created_at":      createdAt.String,
			"updated_at":      updatedAt.String,
			"visibility":      visibility.String,
			"status":          status.String,
			"revision_count":  fmt.Sprintf("%d", revision),
			"pinned":          fmt.Sprintf("%t", pinned == 1),
			"retrieval_usage": fmt.Sprintf("%d", usage),
			"topic_key":       nullStringOrEmpty(topicKey),
			"source":          nullStringOrEmpty(source),
			"normalized_hash": nullStringOrEmpty(hash),
			"owner":           nullStringOrEmpty(owner),
			"tool_name":       nullStringOrEmpty(toolName),
			"last_seen_at":    nullStringOrEmpty(lastSeen),
			"expires_at":      nullStringOrEmpty(expires),
		}
		if graphRef.Valid {
			rec.GraphRef = &graphRef.Int64
			// The observation's embedding is the bridge table row for its
			// graph_ref symbol (step 07). Best-effort: a missing bridge row
			// simply leaves the embeddings map empty.
			var blob []byte
			if err := db.QueryRowContext(ctx,
				`SELECT vector FROM embeddings WHERE symbol_id = ?`, graphRef.Int64).Scan(&blob); err == nil && len(blob) > 0 {
				rec.Embeddings = map[string]string{
					"vector": base64.StdEncoding.EncodeToString(blob),
				}
			}
		}
		bundle.Observations = append(bundle.Observations, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export observations: %w", err)
	}

	// Graph edges: the full history set (temporal window included, step 10).
	edgeRows, err := db.QueryContext(ctx, `
		SELECT e.id, e.kind, e.from_id, e.file_id, e.to_id, e.to_name, e.target_path,
		       e.confidence, e.line, e.valid_from, e.valid_to
		FROM edges e ORDER BY e.id`)
	if err != nil {
		return nil, fmt.Errorf("export edges: %w", err)
	}
	defer edgeRows.Close()
	for edgeRows.Next() {
		var (
			rec        EdgeRecord
			fileID     sql.NullInt64
			toID       sql.NullInt64
			toName     sql.NullString
			targetPath sql.NullString
			confidence string
			line       int
			validFrom  int64
			validTo    sql.NullInt64
		)
		if err := edgeRows.Scan(&rec.ID, &rec.Kind, &rec.FromID, &fileID, &toID, &toName,
			&targetPath, &confidence, &line, &validFrom, &validTo); err != nil {
			return nil, fmt.Errorf("scan export edge: %w", err)
		}
		rec.Confidence = confidence
		rec.Line = line
		rec.ValidFrom = validFrom
		if fileID.Valid {
			v := fileID.Int64
			rec.FileID = &v
		}
		if toID.Valid {
			v := toID.Int64
			rec.ToID = &v
		}
		if toName.Valid {
			rec.ToName = toName.String
		}
		if targetPath.Valid {
			v := targetPath.String
			rec.TargetPath = &v
		}
		if validTo.Valid {
			v := validTo.Int64
			rec.ValidTo = &v
		}
		bundle.GraphEdges = append(bundle.GraphEdges, rec)
	}
	if err := edgeRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export edges: %w", err)
	}

	// Embeddings: every symbol bridge row, vector base64-encoded.
	embRows, err := db.QueryContext(ctx, `
		SELECT symbol_id, model, dim, vector, updated_at
		FROM embeddings ORDER BY symbol_id`)
	if err != nil {
		return nil, fmt.Errorf("export embeddings: %w", err)
	}
	defer embRows.Close()
	for embRows.Next() {
		var (
			rec  EmbeddingRecord
			blob []byte
		)
		if err := embRows.Scan(&rec.SymbolID, &rec.Model, &rec.Dim, &blob, &rec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan export embedding: %w", err)
		}
		rec.Vector = base64.StdEncoding.EncodeToString(blob)
		bundle.Embeddings = append(bundle.Embeddings, rec)
	}
	if err := embRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export embeddings: %w", err)
	}
	return bundle, nil
}

// WriteExport streams the project export as JSON to w. It is the CLI
// boundary for large stores: a json.Encoder writes each top-level value
// incrementally to the writer, so the whole bundle is never materialized as a
// single []byte (the encoder flushes per-Encode). The bundle itself is still
// assembled in memory (one record per store row) — the streaming boundary is
// the encoding step.
func (s *Service) WriteExport(ctx context.Context, w io.Writer) error {
	bundle, err := s.ExportProject(ctx)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(bundle)
}

// ImportProject reads a COGX-inspired export (from ExportProject/WriteExport)
// and inserts its records into this service's store. It is designed to run
// against a FRESH/EMPTY store on another machine.
//
// Upsert/skip strategy (documented choice):
//   - Observations: INSERT with the source row's id (idempotent re-import —
//     a re-run is a no-op when the id is already present, thanks to
//     INSERT OR IGNORE). The session id is ensured (INSERT OR IGNORE) because
//     observations reference sessions(id).
//   - Edges: INSERT OR IGNORE on the composite unique constraint — re-import
//     never duplicates an edge, and the temporal window is preserved.
//   - Embeddings: INSERT OR REPLACE keyed on symbol_id — the latest vector
//     wins, mirroring the codeindex upsert semantics.
//   - Symbols: recreated with their source ids (needed so graph_ref and
//     embeddings resolve after import); files are recreated too (edges FK).
//     All are idempotent (INSERT OR IGNORE on the unique keys).
func (s *Service) ImportProject(ctx context.Context, r io.Reader) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	db := s.store.DB
	var bundle ExportBundle
	if err := json.NewDecoder(r).Decode(&bundle); err != nil {
		return fmt.Errorf("decode export: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import tx: %w", err)
	}
	defer tx.Rollback()

	// 1) Files (edges FK target) — idempotent by path.
	seenFiles := map[int64]bool{}
	for i := range bundle.GraphEdges {
		e := &bundle.GraphEdges[i]
		if e.FileID == nil || seenFiles[*e.FileID] {
			continue
		}
		seenFiles[*e.FileID] = true
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO files (id, path, mtime_ns, size, content_hash, indexed_at)
			VALUES (?, ?, 0, 0, 'imported', ?)`,
			*e.FileID, fmt.Sprintf("imported/%d", *e.FileID),
			time.Now().UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("import file %d: %w", *e.FileID, err)
		}
	}

	// 2) Symbols referenced by edges or embeddings, with source ids.
	neededSymbols := map[int64]bool{}
	for i := range bundle.GraphEdges {
		e := &bundle.GraphEdges[i]
		neededSymbols[e.FromID] = true
		if e.ToID != nil {
			neededSymbols[*e.ToID] = true
		}
	}
	for i := range bundle.Embeddings {
		neededSymbols[bundle.Embeddings[i].SymbolID] = true
	}
	if len(neededSymbols) > 0 {
		firstFileID := int64(0)
		for fid := range seenFiles {
			firstFileID = fid
			break
		}
		for symID := range neededSymbols {
			if _, err := tx.ExecContext(ctx, `
				INSERT OR IGNORE INTO symbols (id, file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
				VALUES (?, ?, ?, ?, 'imported', 'go', '', 1, 2, 'imported', ?)`,
				symID, firstFileID, fmt.Sprintf("imported-symbol-%d", symID),
				fmt.Sprintf("imported-symbol-%d", symID), fmt.Sprintf("uid-imported-%d", symID)); err != nil {
				return fmt.Errorf("import symbol %d: %w", symID, err)
			}
		}
	}

	// 3) Sessions referenced by observations — idempotent by id.
	for i := range bundle.Observations {
		o := &bundle.Observations[i]
		sessID, ok := o.Metadata["session_id"]
		if !ok || sessID == "" {
			sessID = "imported-session"
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
			VALUES (?, ?, '/tmp', ?, 'active')`,
			sessID, s.projectID, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return fmt.Errorf("import session %s: %w", sessID, err)
		}
	}

	// 4) Observations with source ids (INSERT OR IGNORE = skip on re-import).
	for i := range bundle.Observations {
		o := &bundle.Observations[i]
		m := o.Metadata
		if m == nil {
			m = map[string]string{}
		}
		sessID := m["session_id"]
		if sessID == "" {
			sessID = "imported-session"
		}
		title := m["title"]
		if title == "" {
			title = fmt.Sprintf("imported-%d", o.ID)
		}
		var graphRef any
		if o.GraphRef != nil {
			graphRef = *o.GraphRef
		}
		_, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO observations (
				id, session_id, type, title, content, project, scope, topic_key,
				normalized_hash, revision_count, created_at, updated_at, source,
				tool_name, owner, visibility, status, retrieval_usage, expires_at, graph_ref
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			o.ID, sessID, o.Type, title, o.Content, s.projectID, m["scope"], nullString(m["topic_key"]),
			m["normalized_hash"], atoi64orZero(m["revision_count"]), m["created_at"], m["updated_at"],
			m["source"], nullString(m["tool_name"]), nullString(m["owner"]),
			m["visibility"], m["status"], atoi64orZero(m["retrieval_usage"]),
			nullString(m["expires_at"]), graphRef)
		if err != nil {
			return fmt.Errorf("import observation %d: %w", o.ID, err)
		}
	}

	// 5) Edges with temporal window (INSERT OR IGNORE on the unique constraint).
	for i := range bundle.GraphEdges {
		e := &bundle.GraphEdges[i]
		var fileID, toID, targetPath any
		if e.FileID != nil {
			fileID = *e.FileID
		}
		if e.ToID != nil {
			toID = *e.ToID
		}
		if e.TargetPath != nil {
			targetPath = *e.TargetPath
		}
		var validTo any
		if e.ValidTo != nil {
			validTo = *e.ValidTo
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO edges (
				id, kind, from_id, file_id, to_id, to_name, target_path, confidence, line,
				valid_from, valid_to
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.Kind, e.FromID, fileID, toID, e.ToName, targetPath, e.Confidence, e.Line,
			e.ValidFrom, validTo); err != nil {
			return fmt.Errorf("import edge %d: %w", e.ID, err)
		}
	}

	// 6) Embeddings (INSERT OR REPLACE = latest vector wins).
	for i := range bundle.Embeddings {
		em := &bundle.Embeddings[i]
		blob, err := base64.StdEncoding.DecodeString(em.Vector)
		if err != nil {
			return fmt.Errorf("decode embedding %d: %w", em.SymbolID, err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO embeddings (symbol_id, model, dim, vector, updated_at)
			VALUES (?, ?, ?, ?, ?)`,
			em.SymbolID, em.Model, em.Dim, blob, em.UpdatedAt); err != nil {
			return fmt.Errorf("import embedding %d: %w", em.SymbolID, err)
		}
	}

	return tx.Commit()
}

// nullStringOrEmpty renders a NullString as "" when NULL.
func nullStringOrEmpty(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

// atoi64orZero parses an int from a metadata string; 0 when absent/invalid.
func atoi64orZero(s string) int {
	if s == "" {
		return 0
	}
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
