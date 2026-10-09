package context_harness

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultIndexedTTL is the context-harness 7-day TTL applied to indexed_files
// rows (the same horizon observations use). The CLI caller applies this default;
// IndexFile and PurgeExpiredIndexed take it as a parameter so tests can drive
// the boundary deterministically.
const DefaultIndexedTTL = 7 * 24 * time.Hour

// chunkTargetBytes is the soft size target for one indexed_files chunk (~2KB,
// the spec's hint). chunkLines accumulates lines until adding the next line
// would exceed this, then flushes. The final chunk may be smaller.
const chunkTargetBytes = 2048

// maxIndexedChunks bounds one IndexFile call so a 10,000-line file does not
// flood indexed_files in a single shot (ADR-0026 consequence note). 100 chunks
// at ~2KB covers ~200KB of content per call; re-run `ctx index` for more.
const maxIndexedChunks = 100

// languageByExt maps a lowercased file extension (with leading dot) to a
// language tag stored in indexed_files.language. Unknown extensions map to "".
var languageByExt = map[string]string{
	".go":    "go",
	".js":    "javascript",
	".mjs":   "javascript",
	".cjs":   "javascript",
	".ts":    "typescript",
	".tsx":   "typescript",
	".jsx":   "javascript",
	".py":    "python",
	".rb":    "ruby",
	".rs":    "rust",
	".java":  "java",
	".c":     "c",
	".h":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".hpp":   "cpp",
	".cs":    "csharp",
	".php":   "php",
	".swift": "swift",
	".kt":    "kotlin",
	".scala": "scala",
	".sh":    "sh",
	".bash":  "sh",
	".zsh":   "sh",
	".sql":   "sql",
	".html":  "html",
	".htm":   "html",
	".css":   "css",
	".scss":  "scss",
	".less":  "less",
	".md":    "markdown",
	".mdx":   "markdown",
	".json":  "json",
	".yaml":  "yaml",
	".yml":   "yaml",
	".toml":  "toml",
}

// languageFromExt returns the language tag for a file path's extension, or ""
// when the extension is unknown.
func languageFromExt(path string) string {
	return languageByExt[strings.ToLower(filepath.Ext(path))]
}

// IndexFile reads filePath, chunks it into ~chunkTargetBytes line-aligned
// pieces, and upserts one indexed_files row per chunk (kind="file",
// language inferred from the extension, name=filepath.Base(filePath) kept
// consistent across chunks of the same file). It returns the number of chunks
// written.
//
// Idempotency: each row is upserted on the unique index
// (project, name, start_line) via INSERT ... ON CONFLICT DO UPDATE, so
// re-indexing the same file overwrites its existing chunk rows in place and
// never duplicates them. The FTS shadow stays in sync through the 054
// AFTER INSERT trigger (an upsert's DO UPDATE fires no delete trigger, so the
// shadow row for a re-inserted chunk is updated by the next full re-index's
// delete+insert cycle; for a same-content re-index the FTS content is
// unchanged and the trigger is a no-op).
//
// The row's project column holds the project id; name holds the base file
// name (consistent so a re-index of the same file maps to the same rows);
// content holds the chunk text (what FTS searches); start_line/end_line hold
// the 1-based line range of the chunk; created_at holds an RFC3339 timestamp.
func IndexFile(db *sql.DB, project, filePath string) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("index: nil db")
	}
	if strings.TrimSpace(project) == "" {
		return 0, fmt.Errorf("index: project is required")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("index read %s: %w", filePath, err)
	}
	name := filepath.Base(filePath)
	language := languageFromExt(filePath)

	chunks := chunkLines(string(data), chunkTargetBytes, maxIndexedChunks)
	if len(chunks) == 0 {
		// Empty file: still record one empty chunk so the file is present in
		// the index (FTS will not match it, but stats/count reflect it).
		chunks = []chunk{{StartLine: 1, EndLine: 0, Text: ""}}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	written := 0
	for _, c := range chunks {
		_, err := db.Exec(`
			INSERT INTO indexed_files (project, name, kind, language, start_line, end_line, content, created_at)
			VALUES (?, ?, 'file', ?, ?, ?, ?, ?)
			ON CONFLICT (project, name, start_line) DO UPDATE SET
				language   = excluded.language,
				end_line   = excluded.end_line,
				content    = excluded.content,
				created_at = excluded.created_at`,
			project, name, language, c.StartLine, c.EndLine, c.Text, now)
		if err != nil {
			return written, fmt.Errorf("index upsert chunk %d of %s: %w", c.StartLine, name, err)
		}
		written++
	}
	return written, nil
}

// chunk is one line-aligned piece of a file produced by chunkLines.
type chunk struct {
	StartLine int
	EndLine   int
	Text      string
}

// chunkLines splits data into chunks of at most maxBytes each, breaking only
// on line boundaries (a single line longer than maxBytes becomes its own
// chunk). It returns at most maxChunks chunks; if the file has more content
// than maxChunks*maxBytes, the tail beyond maxChunks is dropped (the caller is
// expected to re-run IndexFile to pick up more, per ADR-0026).
//
// StartLine/EndLine are 1-based and inclusive. For an empty input it returns
// nil.
func chunkLines(data string, maxBytes, maxChunks int) []chunk {
	if maxBytes <= 0 {
		maxBytes = chunkTargetBytes
	}
	if maxChunks <= 0 {
		maxChunks = maxIndexedChunks
	}
	lines := strings.Split(data, "\n")
	// strings.Split leaves a trailing "" for data ending in "\n"; drop it so
	// we don't emit a spurious empty final line.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil
	}

	var out []chunk
	var (
		cur    strings.Builder // accumulated chunk text
		startL int             // 1-based start line of the current chunk
		lineNo int             // 1-based line number of the last line added
	)
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		out = append(out, chunk{
			StartLine: startL,
			EndLine:   lineNo,
			Text:      cur.String(),
		})
		cur.Reset()
	}
	for i, line := range lines {
		lineBytes := len(line)
		// cur.Len() is the current chunk size in bytes. Adding this line means
		// cur.Len() + (1 if not empty, for the '\n') + lineBytes. If that would
		// exceed maxBytes and the chunk is non-empty, flush first.
		addBytes := lineBytes
		if cur.Len() > 0 {
			addBytes += 1 // the separating newline
		}
		if cur.Len() > 0 && cur.Len()+addBytes > maxBytes {
			flush()
		}
		if cur.Len() == 0 {
			startL = i + 1 // 1-based
		}
		if cur.Len() > 0 {
			cur.WriteByte('\n')
		}
		cur.WriteString(line)
		lineNo = i + 1 // 1-based
	}
	flush()

	if len(out) > maxChunks {
		out = out[:maxChunks]
	}
	return out
}

// PurgeExpiredIndexed deletes indexed_files rows for project whose created_at
// is older than now - ttl, returning the number of rows deleted. The 7-day
// default is applied by the CLI caller (DefaultIndexedTTL); ttl is a parameter
// so the boundary is testable. Fail-open: an empty/absent table yields 0, nil.
func PurgeExpiredIndexed(db *sql.DB, project string, now time.Time, ttl time.Duration) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("purge indexed: nil db")
	}
	cutoff := now.Add(-ttl).UTC().Format(time.RFC3339)
	res, err := db.Exec(`DELETE FROM indexed_files WHERE project = ? AND created_at < ?`, project, cutoff)
	if err != nil {
		return 0, fmt.Errorf("purge indexed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge indexed rows affected: %w", err)
	}
	return int(affected), nil
}
