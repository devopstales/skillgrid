package handoff

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Record captures HEAD as a change snapshot (idempotent upsert on
// (project, commit)). A repo with no commits, a missing git binary, or a
// commit without a [skillgrid-context] block all still produce a row — the
// context/changed-files fields simply stay NULL. Returns the snapshot.
func (h *Hub) Record(ctx context.Context) (*Snapshot, error) {
	if h == nil || h.DB == nil {
		return nil, errors.New("handoff hub not initialized")
	}
	g := gitState{ctx: ctx, dir: h.RepoDir}

	branch, _ := gitOut(ctx, h.RepoDir, "rev-parse", "--abbrev-ref", "HEAD")
	commit, err := gitOut(ctx, h.RepoDir, "rev-parse", "HEAD")
	if err != nil {
		// Unborn HEAD / no git: nothing to snapshot.
		return nil, fmt.Errorf("resolve HEAD: %w", err)
	}
	short, _ := gitOut(ctx, h.RepoDir, "rev-parse", "--short", "HEAD")
	subject, _ := gitOut(ctx, h.RepoDir, "log", "-1", "--pretty=%s")
	author, _ := gitOut(ctx, h.RepoDir, "log", "-1", "--pretty=%an")
	committedAt, _ := gitOut(ctx, h.RepoDir, "log", "-1", "--pretty=%cI")
	body, _ := gitOut(ctx, h.RepoDir, "log", "-1", "--pretty=%B")

	if committedAt == "" {
		committedAt = time.Now().UTC().Format(time.RFC3339)
	}

	var contextJSON sql.NullString
	if c := parseContext(body); c != (Context{}) {
		if raw, err := json.Marshal(c); err == nil && string(raw) != "{}" {
			contextJSON = sql.NullString{String: string(raw), Valid: true}
		}
	}

	var changedFiles sql.NullString
	if files, err := g.changedFiles(commit); err == nil && len(files) > 0 {
		if raw, err := json.Marshal(files); err == nil {
			changedFiles = sql.NullString{String: string(raw), Valid: true}
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = h.DB.ExecContext(ctx, `
		INSERT INTO change_snapshots
			(project, branch, "commit", commit_short, subject, context_json,
			 changed_files, author, committed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project, "commit") DO UPDATE SET
			branch = excluded.branch,
			commit_short = excluded.commit_short,
			subject = excluded.subject,
			context_json = COALESCE(change_snapshots.context_json, excluded.context_json),
			changed_files = COALESCE(change_snapshots.changed_files, excluded.changed_files),
			author = excluded.author,
			committed_at = excluded.committed_at`,
		h.Project, nullString(branch), commit, nullString(short), nullString(subject),
		contextJSON, changedFiles, nullString(author), committedAt, now)
	if err != nil {
		return nil, fmt.Errorf("upsert snapshot: %w", err)
	}
	return h.getSnapshot(ctx, commit)
}

// Backfill seeds change_snapshots by walking `git log` (newest first, up to
// limit commits). Idempotent: existing rows are left untouched. Returns the
// number of commits inspected. limit <= 0 means the default (100).
func (h *Hub) Backfill(ctx context.Context, limit int) (int, error) {
	if h == nil || h.DB == nil {
		return 0, errors.New("handoff hub not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	g := gitState{ctx: ctx, dir: h.RepoDir}
	// One pass: hash, subject, author, committer-date (ISO), body — body is
	// the last field of each record, so a record is "hash\tsubject\tauthor\tdate\tbody".
	out, err := gitOut(ctx, h.RepoDir, "log",
		"-n", fmt.Sprint(limit),
		"--pretty=format:%H%x09%s%x09%an%x09%cI%x09%B%x00")
	if err != nil {
		return 0, fmt.Errorf("walk git log: %w", err)
	}
	if out == "" {
		return 0, nil
	}
	n := 0
	for _, record := range stringsSplitNull(out) {
		if record == "" {
			continue
		}
		parts := strings.SplitN(record, "\t", 5)
		if len(parts) < 5 {
			continue
		}
		commit, subject, author, committedAt, body := parts[0], parts[1], parts[2], parts[3], parts[4]
		body = strings.TrimSpace(strings.TrimSuffix(body, "\n"))

		var contextJSON sql.NullString
		if c := parseContext(body); c != (Context{}) {
			if raw, err := json.Marshal(c); err == nil && string(raw) != "{}" {
				contextJSON = sql.NullString{String: string(raw), Valid: true}
			}
		}
		var changedFiles sql.NullString
		if files, err := g.changedFiles(commit); err == nil && len(files) > 0 {
			if raw, err := json.Marshal(files); err == nil {
				changedFiles = sql.NullString{String: string(raw), Valid: true}
			}
		}
		// created_at is a plain insertion timestamp. Ordering is by id DESC
		// (insertion order), which equals git log's newest-first walk — this
		// is deterministic and independent of committer timestamps (which can
		// share a second or even go backwards across branches).
		created := time.Now().UTC().Format(time.RFC3339)
		if _, err := h.DB.ExecContext(ctx, `
			INSERT INTO change_snapshots
				(project, branch, "commit", commit_short, subject, context_json,
				 changed_files, author, committed_at, created_at)
			VALUES (?, 'unknown', ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(project, "commit") DO NOTHING`,
			h.Project, commit, nullString(shortHash(commit)), nullString(subject),
			contextJSON, changedFiles, nullString(author),
			committedAt, created); err != nil {
			return n, fmt.Errorf("backfill upsert %s: %w", commit, err)
		}
		n++
	}
	return n, nil
}

// ListSnapshots returns the newest-first change log (limit clamped to 1..500).
func (h *Hub) ListSnapshots(ctx context.Context, limit int) ([]Snapshot, error) {
	if h == nil || h.DB == nil {
		return nil, errors.New("handoff hub not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, branch, "commit", COALESCE(commit_short,''), COALESCE(subject,''),
		       context_json, changed_files, author, committed_at
		FROM change_snapshots
		WHERE project = ?
		ORDER BY id ASC
		LIMIT ?`, h.Project, limit)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	out := []Snapshot{}
	for rows.Next() {
		var s Snapshot
		var contextJSON, changedFiles, author sql.NullString
		var branch sql.NullString
		if err := rows.Scan(&s.ID, &branch, &s.Commit, &s.CommitShort, &s.Subject,
			&contextJSON, &changedFiles, &author, &s.CommittedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		s.Branch = branch.String
		s.ContextJSON = contextJSON.String
		s.ChangedFiles = changedFiles.String
		s.Author = author.String
		out = append(out, s)
	}
	return out, rows.Err()
}

// CountSnapshots returns the number of stored snapshots for the project.
func (h *Hub) CountSnapshots(ctx context.Context) (int, error) {
	var n int
	err := h.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM change_snapshots WHERE project = ?`, h.Project).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (h *Hub) getSnapshot(ctx context.Context, commit string) (*Snapshot, error) {
	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, branch, "commit", COALESCE(commit_short,''), COALESCE(subject,''),
		       context_json, changed_files, author, committed_at
		FROM change_snapshots WHERE project = ? AND "commit" = ?`, h.Project, commit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, errors.New("snapshot not found")
	}
	var s Snapshot
	var branch, contextJSON, changedFiles, author sql.NullString
	if err := rows.Scan(&s.ID, &branch, &s.Commit, &s.CommitShort, &s.Subject,
		&contextJSON, &changedFiles, &author, &s.CommittedAt); err != nil {
		return nil, err
	}
	s.Branch = branch.String
	s.ContextJSON = contextJSON.String
	s.ChangedFiles = changedFiles.String
	s.Author = author.String
	return &s, nil
}

type gitState struct {
	ctx context.Context
	dir string
}

// changedFiles lists the files a commit touched via diff-tree
// (--root handles the first commit of a repo). Best-effort: any error yields
// an empty list, never a failure.
func (g gitState) changedFiles(commit string) ([]ChangedFile, error) {
	out, err := gitOut(g.ctx, g.dir, "diff-tree", "-r", "--name-status", "--root",
		"--no-commit-id", commit)
	if err != nil || out == "" {
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	var files []ChangedFile
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// diff-tree --name-status: "M\tpath" (tab-separated); renames are
		// "R100\told\tnew" — keep the new path.
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		path := parts[len(parts)-1]
		files = append(files, ChangedFile{Path: path, Status: status})
	}
	return files, nil
}

// stringsSplitNull splits on NUL bytes (strings has no SplitNUL).
func stringsSplitNull(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func nullString(s string) sql.NullString {
	if strings.TrimSpace(s) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func shortHash(commit string) string {
	if len(commit) >= 7 {
		return commit[:7]
	}
	return commit
}
