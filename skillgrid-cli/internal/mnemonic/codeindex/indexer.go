// Package codeindex incrementally indexes source files into a project store.
package codeindex

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

const maxFileSize = 512 * 1024

// Config controls incremental indexing behavior.
type Config struct {
	Include      []string
	Exclude      []string
	ChunkLines   int
	ChunkOverlap int
}

// Stats summarizes one indexing run.
type Stats struct {
	FilesIndexed int `json:"files_indexed"`
	FilesSkipped int `json:"files_skipped"`
	FilesDeleted int `json:"files_deleted"`
	ChunksAdded  int `json:"chunks_added"`
}

// ScannedFile is a candidate file discovered under the index root.
type ScannedFile struct {
	Path     string
	MtimeNs  int64
	Size     int64
	Hash     string
	Contents []byte
}

// Scan walks root and returns files matching include/exclude globs.
func Scan(root string, include, exclude []string) ([]ScannedFile, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var files []ScannedFile
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel != "." && shouldSkipDir(rel, exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxFileSize {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if matchesAny(exclude, rel) {
			return nil
		}
		if len(include) > 0 && !matchesAny(include, rel) {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		sum := sha256.Sum256(contents)
		files = append(files, ScannedFile{
			Path:     rel,
			MtimeNs:  info.ModTime().UnixNano(),
			Size:     info.Size(),
			Hash:     hex.EncodeToString(sum[:]),
			Contents: contents,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func shouldSkipDir(rel string, exclude []string) bool {
	rel = filepath.ToSlash(rel)
	for _, pattern := range exclude {
		pattern = filepath.ToSlash(strings.TrimSuffix(pattern, "/**"))
		pattern = strings.TrimPrefix(pattern, "**/")
		if pattern == "" {
			continue
		}
		if rel == pattern || strings.HasPrefix(rel, pattern+"/") {
			return true
		}
		if matchesGlob(pattern, rel) || matchesGlob(pattern+"/**", rel) {
			return true
		}
	}
	return false
}

func matchesAny(patterns []string, path string) bool {
	for _, p := range patterns {
		if matchesGlob(p, path) {
			return true
		}
	}
	return false
}

func matchesGlob(pattern, path string) bool {
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)
	if !strings.Contains(pattern, "**") {
		matched, _ := filepath.Match(pattern, path)
		return matched
	}
	re, err := globToRegexp(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(path)
}

func globToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					i += 2
					b.WriteString("(.*/)?")
				} else {
					i++
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(pattern[i])
		default:
			b.WriteByte(pattern[i])
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// Chunk represents a slice of a file for FTS indexing.
type Chunk struct {
	StartLine   int
	EndLine     int
	Text        string
	ContentHash string
}

// ChunkLines splits content into overlapping windows of ~chunkLines.
func ChunkLines(content []byte, chunkLines, chunkOverlap int) []Chunk {
	if chunkLines <= 0 {
		chunkLines = 80
	}
	text := string(content)
	allLines := strings.Split(text, "\n")
	if len(allLines) == 0 {
		return nil
	}
	step := chunkLines - chunkOverlap
	if step <= 0 {
		step = chunkLines
	}
	var chunks []Chunk
	for start := 0; start < len(allLines); start += step {
		end := start + chunkLines
		if end > len(allLines) {
			end = len(allLines)
		}
		chunkText := strings.Join(allLines[start:end], "\n")
		if strings.TrimSpace(chunkText) == "" {
			if end >= len(allLines) {
				break
			}
			continue
		}
		sum := sha256.Sum256([]byte(chunkText))
		chunks = append(chunks, Chunk{
			StartLine:   start + 1,
			EndLine:     end,
			Text:        chunkText,
			ContentHash: hex.EncodeToString(sum[:]),
		})
		if end >= len(allLines) {
			break
		}
	}
	return chunks
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
	FileCount   int    `json:"file_count"`
	ChunkCount  int    `json:"chunk_count"`
	LastIndexed string `json:"last_indexed,omitempty"`
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
