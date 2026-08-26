package codeindex

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"skillgrid-cli/internal/mnemonic/store"
)

// Config controls incremental indexing behavior.
type Config struct {
	Include      []string
	Exclude      []string
	ChunkLines   int
	ChunkOverlap int
}

// Stats summarizes one indexing run.
type Stats struct {
	FilesIndexed int
	FilesSkipped int
	FilesDeleted int
	ChunksAdded  int
}

// Indexer incrementally indexes source files into the store.
type Indexer struct {
	store *store.Store
}

// New creates an Indexer backed by st.
func New(st *store.Store) *Indexer {
	return &Indexer{store: st}
}

type existingFile struct {
	ID          int64
	MtimeNs     int64
	Size        int64
	ContentHash string
}

// Run scans root and upserts changed files; removes stale entries.
func (idx *Indexer) Run(ctx context.Context, root string, cfg Config) (Stats, error) {
	var stats Stats
	if idx == nil || idx.store == nil || idx.store.DB == nil {
		return stats, fmt.Errorf("indexer not initialized")
	}

	scanned, err := Scan(root, cfg.Include, cfg.Exclude)
	if err != nil {
		return stats, err
	}

	existing, err := loadExistingFiles(idx.store.DB)
	if err != nil {
		return stats, err
	}

	scannedPaths := make(map[string]struct{}, len(scanned))
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := idx.store.DB.BeginTx(ctx, nil)
	if err != nil {
		return stats, err
	}
	defer tx.Rollback()

	for _, file := range scanned {
		if err := ctx.Err(); err != nil {
			return stats, err
		}

		scannedPaths[file.Path] = struct{}{}

		prev, ok := existing[file.Path]
		if ok && prev.MtimeNs == file.MtimeNs && prev.Size == file.Size && prev.ContentHash == file.Hash {
			stats.FilesSkipped++
			continue
		}

		fileID, err := upsertFile(tx, file, now)
		if err != nil {
			return stats, err
		}

		if ok {
			if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE file_id = ?`, fileID); err != nil {
				return stats, fmt.Errorf("delete chunks for %s: %w", file.Path, err)
			}
		}

		chunks := ChunkLines(file.Contents, cfg.ChunkLines, cfg.ChunkOverlap)
		for _, chunk := range chunks {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO chunks (file_id, start_line, end_line, text, content_hash) VALUES (?, ?, ?, ?, ?)`,
				fileID, chunk.StartLine, chunk.EndLine, chunk.Text, chunk.ContentHash,
			); err != nil {
				return stats, fmt.Errorf("insert chunk for %s: %w", file.Path, err)
			}
			stats.ChunksAdded++
		}

		stats.FilesIndexed++
	}

	for path, prev := range existing {
		if _, ok := scannedPaths[path]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, prev.ID); err != nil {
			return stats, fmt.Errorf("delete file %s: %w", path, err)
		}
		stats.FilesDeleted++
	}

	if err := tx.Commit(); err != nil {
		return stats, err
	}
	return stats, nil
}

func loadExistingFiles(db *sql.DB) (map[string]existingFile, error) {
	rows, err := db.Query(`SELECT id, path, mtime_ns, size, content_hash FROM files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]existingFile)
	for rows.Next() {
		var f existingFile
		var path string
		if err := rows.Scan(&f.ID, &path, &f.MtimeNs, &f.Size, &f.ContentHash); err != nil {
			return nil, err
		}
		out[path] = f
	}
	return out, rows.Err()
}

func upsertFile(tx *sql.Tx, file ScannedFile, indexedAt string) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(path) DO UPDATE SET
		   mtime_ns = excluded.mtime_ns,
		   size = excluded.size,
		   content_hash = excluded.content_hash,
		   indexed_at = excluded.indexed_at`,
		file.Path, file.MtimeNs, file.Size, file.Hash, indexedAt,
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err == nil && id > 0 {
		return id, nil
	}

	var fileID int64
	err = tx.QueryRow(`SELECT id FROM files WHERE path = ?`, file.Path).Scan(&fileID)
	return fileID, err
}

// Status reports aggregate index statistics.
type Status struct {
	FileCount  int
	ChunkCount int
	LastIndexed string
}

// GetStatus returns current index stats from the store.
func GetStatus(st *store.Store) (Status, error) {
	var status Status
	if st == nil || st.DB == nil {
		return status, fmt.Errorf("store not initialized")
	}

	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&status.FileCount); err != nil {
		return status, err
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&status.ChunkCount); err != nil {
		return status, err
	}

	var last sql.NullString
	if err := st.DB.QueryRow(`SELECT MAX(indexed_at) FROM files`).Scan(&last); err != nil {
		return status, err
	}
	if last.Valid {
		status.LastIndexed = last.String
	}
	return status, nil
}
