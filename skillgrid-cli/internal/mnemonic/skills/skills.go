// Package skills is the Agent Skill registry (change 2026-09-04-hermes-memory,
// TICKET-03): reusable scripts kept as FS files under .skillgrid/files/skills/
// with SQL metadata in the 011 skills table and a lexical FTS5 index
// (skills_fts) over name/description. It is the explicit registry leg of the
// dual skill path — the intent-match leg stays on memory_type="skill"
// observations (memory.MatchSkills), which this package does not touch.
// TICKET-03 ships Write/List/Search; the sandboxed use_skill executor and the
// hybrid search mode arrive in TICKET-04/05.
package skills

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Store is the Agent Skill registry handle over one project store. db is the
// 011-migrated project store; root is the workspace directory that owns the
// .skillgrid/ scratch tree (the ContentPlane convention).
type Store struct {
	db   *sql.DB
	root string
}

// New wraps the project store's database handle and workspace root.
func New(db *sql.DB, root string) *Store {
	return &Store{db: db, root: root}
}

// Skill is one registry row (the read model for List/Search).
type Skill struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Language    string `json:"language"`
	Description string `json:"description,omitempty"`
	CodePath    string `json:"code_path"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ErrNameTaken is the sentinel for a write_skill name collision with
// overwrite=false (the tool surfaces it verbatim).
var ErrNameTaken = errors.New("skill name already exists; pass overwrite=true to replace it")

// knownLanguages is the extension map for skill file names. It mirrors the
// code-index language set (memory.extLanguage); a language outside it is
// rejected before any FS or SQL side effect (path escape / unknown language
// reject without exec — step 03 boundary, enforced at the registry here).
var knownLanguages = map[string]string{
	"go":         "go",
	"typescript": "ts",
	"javascript": "js",
	"python":     "py",
	"rust":       "rs",
	"java":       "java",
	"ruby":       "rb",
	"csharp":     "cs",
	"c":          "c",
	"cpp":        "cpp",
	"swift":      "swift",
	"kotlin":     "kt",
	"php":        "php",
	"sh":         "sh",
	"bash":       "sh",
	"zsh":        "sh",
	"shell":      "sh",
	"markdown":   "md",
	"md":         "md",
	"json":       "json",
	"yaml":       "yaml",
	"yml":        "yaml",
	"toml":       "toml",
	"sql":        "sql",
	"html":       "html",
	"css":        "css",
	"r":          "r",
	"lua":        "lua",
	"perl":       "pl",
}

// defaultSearchLimit caps a Search call when the caller passes limit <= 0.
const defaultSearchLimit = 20

// validName reports whether name is a single, safe file basename: no
// separators, no path traversal, not the dot names, ASCII alnum/dash/underscore
// (the registry names are identifiers, not arbitrary strings).
func validName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, "/\\\x00") || strings.Contains(name, "..") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

// skillDir returns the absolute skills directory under the workspace root
// (.skillgrid/files/skills — NOT .agents/skills, a different concept).
func (s *Store) skillDir() string {
	return filepath.Join(s.root, ".skillgrid", "files", "skills")
}

// Write creates (or, with overwrite, replaces) an Agent Skill: the FS file
// under .skillgrid/files/skills/{name}.{ext}, the SQL metadata row, and the
// FTS index (kept in sync by the 011 triggers). With overwrite=false a name
// collision with a LIVE skill returns ErrNameTaken and touches nothing — not
// even the FS file. With overwrite=true the live row is updated in place
// (the name is UNIQUE, so one row per name) and the FS file is rewritten.
// A name that is soft-deleted is treated as free: its row is reused (resurrected).
func (s *Store) Write(ctx context.Context, name, language, description, code string, overwrite bool) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("skills store not initialized")
	}
	name = strings.TrimSpace(name)
	if !validName(name) {
		return 0, fmt.Errorf("invalid skill name %q: use alnum, dash, underscore or dot — no path separators", name)
	}
	ext, ok := knownLanguages[strings.ToLower(strings.TrimSpace(language))]
	if !ok {
		return 0, fmt.Errorf("unknown language %q: supported languages: %s", language, supportedLanguages())
	}

	var existingID int64
	var existingDeletedAt sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, deleted_at FROM skills WHERE name = ?`, name).
		Scan(&existingID, &existingDeletedAt)
	switch {
	case err == nil:
		if !existingDeletedAt.Valid && !overwrite {
			return 0, fmt.Errorf("%w: %q", ErrNameTaken, name)
		}
	case errors.Is(err, sql.ErrNoRows):
		// New skill.
	default:
		return 0, fmt.Errorf("lookup skill: %w", err)
	}

	absPath := filepath.Join(s.skillDir(), name+"."+ext)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin skill write: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	var id int64
	switch {
	case existingID == 0:
		res, err := tx.ExecContext(ctx, `
			INSERT INTO skills (name, language, description, code_path, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			name, strings.ToLower(strings.TrimSpace(language)), description, absPath, now, now)
		if err != nil {
			return 0, fmt.Errorf("insert skill: %w", err)
		}
		id, err = res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("skill id: %w", err)
		}
	default:
		// Replace the live row or resurrect the soft-deleted one in place.
		if _, err := tx.ExecContext(ctx, `
			UPDATE skills SET language = ?, description = ?, code_path = ?,
			    deleted_at = NULL, updated_at = ?
			WHERE id = ?`,
			strings.ToLower(strings.TrimSpace(language)), description, absPath, now, existingID); err != nil {
			return 0, fmt.Errorf("update skill: %w", err)
		}
		id = existingID
	}

	// FS last: if the file write fails, the transaction rolls back so SQL and
	// disk never diverge (the ContentPlane FS-first convention inverted — the
	// row is the identity, the file is the payload).
	if err := os.MkdirAll(s.skillDir(), 0o755); err != nil {
		return 0, fmt.Errorf("create skills dir: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(code), 0o644); err != nil {
		return 0, fmt.Errorf("write skill file: %w", err)
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(absPath)
		return 0, fmt.Errorf("commit skill write: %w", err)
	}
	return id, nil
}

// List returns every live (non-soft-deleted) skill with its metadata, newest
// first. Soft-deleted rows are omitted by default (the audit read goes through
// the SQL directly).
func (s *Store) List(ctx context.Context) ([]Skill, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("skills store not initialized")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, language, COALESCE(description, ''), code_path, created_at, updated_at
		FROM skills
		WHERE deleted_at IS NULL
		ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()
	var out []Skill
	for rows.Next() {
		var sk Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Language, &sk.Description,
			&sk.CodePath, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		out = append(out, sk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skills: %w", err)
	}
	return out, nil
}

// Search is the default lexical skill search (TICKET-03): FTS5 over
// skills_fts (name + description), bm25-ranked, soft-deleted skills excluded.
// It is the explicit-registry leg; the intent-match leg (memory_type="skill"
// observations) is untouched.
func (s *Store) Search(ctx context.Context, query string, limit int) ([]Skill, error) {
	return s.SearchWith(ctx, query, limit, false)
}

// SearchWith is Search with an explicit soft-delete filter: includeDeleted
// lifts the deleted_at IS NULL clause (the audit/escape-hatch read).
func (s *Store) SearchWith(ctx context.Context, query string, limit int, includeDeleted bool) ([]Skill, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("skills store not initialized")
	}
	ftsQuery, err := buildSkillFTSQuery(query)
	if err != nil {
		return nil, err
	}
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	deletedClause := ""
	if !includeDeleted {
		deletedClause = " AND sk.deleted_at IS NULL"
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT sk.id, sk.name, sk.language, COALESCE(sk.description, ''),
		       sk.code_path, sk.created_at, sk.updated_at
		FROM skills sk
		INNER JOIN skills_fts ON skills_fts.rowid = sk.id
		WHERE skills_fts MATCH ?`+deletedClause+`
		ORDER BY bm25(skills_fts)
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search skills: %w", err)
	}
	defer rows.Close()
	var out []Skill
	for rows.Next() {
		var sk Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Language, &sk.Description,
			&sk.CodePath, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		out = append(out, sk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skill search: %w", err)
	}
	return out, nil
}

// buildSkillFTSQuery converts a plain-text query into a safe FTS5 MATCH
// expression (the same any-term recall convention as the facts search): terms
// are quoted (FTS special characters can't break the syntax) and OR-joined.
// Returns "" for a blank query.
func buildSkillFTSQuery(query string) (string, error) {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return "", nil
	}
	escaped := make([]string, len(terms))
	for i, term := range terms {
		escaped[i] = `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
	}
	return strings.Join(escaped, " OR "), nil
}

// supportedLanguages renders the knownLanguages set as a sorted, deduped
// "a, b, c" string for the unknown-language error message.
func supportedLanguages() string {
	exts := map[string]struct{}{}
	for lang := range knownLanguages {
		exts[lang] = struct{}{}
	}
	out := make([]string, 0, len(exts))
	for lang := range exts {
		out = append(out, lang)
	}
	// Stable order: simple insertion sort (the set is tiny).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return strings.Join(out, ", ")
}
