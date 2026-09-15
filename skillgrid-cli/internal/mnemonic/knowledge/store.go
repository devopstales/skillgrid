package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/extract"
)

// Store is the knowledge pass's persistence seam: it upserts knowledge nodes
// (doc/config/sql) and their confidence-labeled edges into the 005 graph. The
// indexer passes a Store backed by the pass DB (a fresh *sql.DB opened after
// the 005 tx commits — the store's single-connection pool deadlocks once the
// committed 005 tx holds the write lock, so the pass runs after, not in, that
// tx). A knowledge node is always resolvable because the 005 symbols it points
// at are already committed; the pass is advisory and does not roll back 005.
type Store struct {
	db *sql.DB
}

// NewStore wraps a *sql.DB (the indexer's pass DB in production — a fresh
// *sql.DB opened after the 005 tx commits — or a scratch store in tests).
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// docNodeID resolves (or lazily creates) the doc node id for a file path.
// It is idempotent: a doc node is one-per-file (015 idx_doc_nodes_file unique).
func (s *Store) docNodeID(path string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM doc_nodes WHERE path = ?`, path).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	// The doc's source file must exist for the cascade; resolve it by path.
	var fileID int64
	if ferr := s.db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&fileID); ferr != nil {
		return 0, ferr
	}
	res, err := s.db.Exec(`INSERT INTO doc_nodes (file_id, title, path) VALUES (?, ?, ?)`,
		fileID, DocTitle(path, nil), path)
	if err != nil {
		return 0, fmt.Errorf("insert doc node %s: %w", path, err)
	}
	return res.LastInsertId()
}

// SaveDoc upserts one doc node and its references edges (target-state: the
// doc's prior references edges are pruned first, then the freshly extracted
// set is written). A link target that does not resolve to an indexed doc node
// is dropped (drop-not-guess for docs, per 03.2 — a markdown link to a
// non-doc is not a doc->doc reference). Never errors on a bad link: it is
// simply skipped.
func (s *Store) SaveDoc(ctx context.Context, path string, doc *DocResult) (int, error) {
	if doc == nil || !doc.IsDoc {
		return 0, nil
	}
	nodeID, err := s.docNodeID(path)
	if err != nil {
		return 0, err
	}
	// Target-state: prune this doc's prior references edges, then upsert.
	if _, err := s.db.Exec(`
		DELETE FROM edges
		WHERE kind = 'references' AND from_id = ?`, nodeID); err != nil {
		return 0, err
	}
	stored := 0
	for _, link := range doc.Links {
		target := resolveDocTarget(path, link.Target)
		toID, ok := s.resolveDocNode(target)
		if !ok {
			continue // unresolvable doc reference → dropped (not fabricated)
		}
		// valid_from records when the doc link was observed (014 step 10
		// temporal edges); on conflict the existing row's observation time
		// is kept (first observation wins).
		if _, err := s.db.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, context, confidence_score, line, valid_from)
			VALUES ('references', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
			  confidence = excluded.confidence,
			  context = excluded.context,
			  confidence_score = excluded.confidence_score`,
			nodeID, fileIDFor(s.db, path), toID, target, target, link.Confidence, "doc_link", extract.ConfidenceScore(link.Confidence), link.Line, time.Now().Unix()); err != nil {
			return stored, fmt.Errorf("upsert doc reference %s: %w", link.Target, err)
		}
		stored++
	}
	// Doc→symbol edges: a symbol name the doc text mentions, when it resolves
	// to EXACTLY ONE indexed symbol (0 or >1 matches → skipped, drop-not-guess).
	// These share the kind='references' prune above (from_id = this doc node),
	// so they are target-state per doc like the doc→doc links.
	for _, ref := range doc.SymbolRefs {
		var matches int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE name = ?`, ref.Name).Scan(&matches); err != nil {
			return stored, err
		}
		if matches != 1 {
			continue
		}
		var symID int64
		if err := s.db.QueryRow(`SELECT id FROM symbols WHERE name = ?`, ref.Name).Scan(&symID); err != nil {
			return stored, err
		}
		if _, err := s.db.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, context, confidence_score, line, valid_from)
			VALUES ('references', ?, ?, ?, ?, ?, 'INFERRED', 'doc_mention', 0.85, ?, ?)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
			  confidence = excluded.confidence,
			  context = excluded.context,
			  confidence_score = excluded.confidence_score`,
			nodeID, fileIDFor(s.db, path), symID, ref.Name, ref.Name, ref.Line, time.Now().Unix()); err != nil {
			return stored, fmt.Errorf("upsert doc symbol reference %s: %w", ref.Name, err)
		}
		stored++
	}
	return stored, nil
}

// resolveDocNode resolves a doc target path to its doc node id (creating the
// node if the source file is indexed but the node is not yet written). It
// returns ok=false when the target is not an indexed doc (no file row).
func (s *Store) resolveDocNode(target string) (int64, bool) {
	var id int64
	if err := s.db.QueryRow(`SELECT id FROM doc_nodes WHERE path = ?`, target).Scan(&id); err == nil {
		return id, true
	}
	if err := s.db.QueryRow(`SELECT id FROM files WHERE path = ?`, target).Scan(new(int64)); err != nil {
		return 0, false // the target file is not indexed → not a doc node
	}
	// The file is indexed but the doc node is not yet written (e.g. the target
	// doc is processed later in the run). Create it now so the edge resolves.
	var fileID int64
	if err := s.db.QueryRow(`SELECT id FROM files WHERE path = ?`, target).Scan(&fileID); err != nil {
		return 0, false
	}
	res, err := s.db.Exec(`INSERT INTO doc_nodes (file_id, title, path) VALUES (?, ?, ?)`,
		fileID, DocTitle(target, nil), target)
	if err != nil {
		return 0, false
	}
	id, _ = res.LastInsertId()
	return id, true
}

// configNodeID resolves (or lazily creates) the config node id for a file
// path. It is idempotent: a config node is one-per-file (015
// idx_config_nodes_file unique).
func (s *Store) configNodeID(path string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM config_nodes WHERE path = ?`, path).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	var fileID int64
	if ferr := s.db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&fileID); ferr != nil {
		return 0, ferr
	}
	res, err := s.db.Exec(`INSERT INTO config_nodes (file_id, title, path) VALUES (?, ?, ?)`,
		fileID, filepath.Base(path), path)
	if err != nil {
		return 0, fmt.Errorf("insert config node %s: %w", path, err)
	}
	return res.LastInsertId()
}

// SaveConfig upserts one config node and its configures edges (target-state:
// the config's prior configures edges are pruned first, then the freshly
// extracted set is written). Each reference is resolved against the indexed
// symbols:
//
//	EXTRACTED — the value names a symbol by explicit syntax and resolves to a
//	            live symbol (to_id set).
//	INFERRED  — the value names a symbol by convention and resolves to a live
//	            symbol (to_id set).
//	AMBIGUOUS — the value resolves to NO known symbol. It is KEPT (to_id null,
//	            to_name the literal ref), not dropped, per 03.5: an
//	            unresolvable config reference is surfaced as low-confidence,
//	            never silently discarded.
func (s *Store) SaveConfig(ctx context.Context, path string, cfg *ConfigResult) (int, error) {
	if cfg == nil || !cfg.IsConfig {
		return 0, nil
	}
	nodeID, err := s.configNodeID(path)
	if err != nil {
		return 0, err
	}
	if _, err := s.db.Exec(`
		DELETE FROM edges
		WHERE kind = 'configures' AND from_id = ?`, nodeID); err != nil {
		return 0, err
	}
	stored := 0
	for _, ref := range cfg.Refs {
		var toID sql.NullInt64
		var toUID string
		var matches int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE name = ?`, ref.Value).Scan(&matches)
		if matches == 1 {
			_ = s.db.QueryRow(`SELECT id, uid FROM symbols WHERE name = ?`, ref.Value).Scan(&toID.Int64, &toUID)
			toID.Valid = true
		}
		conf := ref.Confidence
		if !toID.Valid {
			// Unresolvable OR ambiguous config ref → AMBIGUOUS (kept, not
			// dropped, 03.5). matches > 1 is a cross-package name collision:
			// picking the lowest id would silently bind the ref to the wrong
			// symbol as EXTRACTED, so the ref is surfaced AMBIGUOUS (to_id null)
			// rather than guessing.
			conf = ConfidenceAmbiguous
		}
		// valid_from records when the config reference was observed (014
		// step 10 temporal edges); on conflict the existing row's
		// observation time is kept (first observation wins).
		if _, err := s.db.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, context, confidence_score, line, valid_from)
			VALUES ('configures', ?, ?, ?, ?, ?, ?, 'config', ?, ?, ?)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
			  confidence = excluded.confidence,
			  context = excluded.context,
			  confidence_score = excluded.confidence_score`,
			nodeID, fileIDFor(s.db, path), toID, ref.Value, "", conf, extract.ConfidenceScore(conf), ref.Line, time.Now().Unix()); err != nil {
			return stored, fmt.Errorf("upsert config reference %s: %w", ref.Value, err)
		}
		stored++
	}
	return stored, nil
}

// SaveSchema upserts one file's SQL schema (tables + columns) and its
// read/write edges (target-state: the file's prior sql nodes and reads/writes
// edges are pruned first, then the freshly extracted set is written).
//
// DDL: each CREATE TABLE becomes a table node + a column node per column.
// DML: each SELECT/INSERT/UPDATE/DELETE becomes a reads/writes edge from the
// enclosing code symbol (the file's first symbol, or the file itself when no
// symbol is indexed) to the table node. Every reads/writes edge is
// confidence-labeled EXTRACTED (the SQL statement explicitly names the table).
func (s *Store) SaveSchema(ctx context.Context, path string, res *SQLResult) (int, error) {
	if res == nil || !res.IsSQL {
		return 0, nil
	}
	fileID := fileIDFor(s.db, path)
	// Target-state: prune this file's prior sql_schema_nodes and reads/writes
	// edges (the re-extracted set is the full target).
	if fileID != 0 {
		if _, err := s.db.Exec(`DELETE FROM sql_schema_nodes WHERE file_id = ?`, fileID); err != nil {
			return 0, err
		}
		if _, err := s.db.Exec(`
			DELETE FROM edges
			WHERE kind IN ('reads','writes') AND file_id = ?`, fileID); err != nil {
			return 0, err
		}
	}
	stored := 0
	// DDL: table + column nodes.
	for _, t := range res.Tables {
		tn, err := s.upsertSQLNode(fileID, t.Name, "", KindTable, path)
		if err != nil {
			return stored, err
		}
		stored++
		for _, col := range t.Columns {
			if _, err := s.upsertSQLNode(fileID, t.Name, col, KindColumn, path); err != nil {
				return stored, err
			}
			stored++
		}
		_ = tn
	}
	// DML: reads/writes edges from the file's code symbol to the table node.
	if len(res.Access) > 0 {
		fromID := s.firstSymbolID(fileID)
		if fromID == 0 {
			return stored, nil // no code symbol to attribute the access to
		}
		for _, a := range res.Access {
			tn, err := s.tableNodeID(a.Table)
			if err != nil {
				// The accessed table is not in the schema (defined elsewhere or
				// not indexed) — the access is still recorded as a name-only
				// edge (to_name the table, to_id null), EXTRACTED.
				// valid_from records when the access was observed (014 step
				// 10 temporal edges); on conflict the existing row's
				// observation time is kept (first observation wins).
				if _, err := s.db.Exec(`
					INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, context, confidence_score, line, valid_from)
					VALUES (?, ?, ?, NULL, ?, ?, ?, 'sql', ?, ?, ?)
					ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
					  confidence = excluded.confidence,
					  context = excluded.context,
					  confidence_score = excluded.confidence_score`,
					a.Op, fromID, fileID, a.Table, "", a.Confidence, extract.ConfidenceScore(a.Confidence), a.Line, time.Now().Unix()); err != nil {
					return stored, err
				}
				stored++
				continue
			}
			// valid_from records when the access was observed (014 step 10
			// temporal edges); on conflict the existing row's observation
			// time is kept (first observation wins).
			if _, err := s.db.Exec(`
				INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, context, confidence_score, line, valid_from)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'sql', ?, ?, ?)
				ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
				  confidence = excluded.confidence,
				  context = excluded.context,
				  confidence_score = excluded.confidence_score`,
				a.Op, fromID, fileID, tn, a.Table, "", a.Confidence, extract.ConfidenceScore(a.Confidence), a.Line, time.Now().Unix()); err != nil {
				return stored, err
			}
			stored++
		}
	}
	return stored, nil
}

// upsertSQLNode inserts a sql_schema_node (table or column) and returns its id.
func (s *Store) upsertSQLNode(fileID int64, table, column, kind, path string) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO sql_schema_nodes (file_id, table_name, column_name, kind, path) VALUES (?, ?, ?, ?, ?)`,
		fileID, table, column, kind, path)
	if err != nil {
		return 0, fmt.Errorf("insert sql node %s.%s: %w", table, column, err)
	}
	return res.LastInsertId()
}

// tableNodeID resolves a table name to its sql_schema_node id (kind=table).
func (s *Store) tableNodeID(table string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM sql_schema_nodes WHERE table_name = ? AND kind = 'table' ORDER BY id LIMIT 1`, table).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, err
	}
	return id, err
}

// firstSymbolID returns the file's first (lowest id) 005 symbol id, or 0 when
// the file has no indexed symbol (the access edge is then not fabricated).
func (s *Store) firstSymbolID(fileID int64) int64 {
	if fileID == 0 {
		return 0
	}
	var id int64
	err := s.db.QueryRow(`SELECT id FROM symbols WHERE file_id = ? ORDER BY start_line, id LIMIT 1`, fileID).Scan(&id)
	if err != nil {
		return 0
	}
	return id
}

// resolveDocTarget normalizes a markdown link target relative to the source
// doc's directory into an index-relative path (slash-separated).
func resolveDocTarget(from, target string) string {
	base := filepath.Dir(from)
	joined := filepath.Join(base, target)
	return filepath.ToSlash(joined)
}

// fileIDFor resolves a file path to its files.id (0 when unknown — the edges
// table allows a null file_id for knowledge edges).
func fileIDFor(db *sql.DB, path string) int64 {
	var id int64
	_ = db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&id)
	return id
}
