package handoff

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

// CheckpointInput is a named, intentional checkpoint marker.
type CheckpointInput struct {
	Name     string // e.g. "before-apply-auth-foundation"
	Evidence string // optional short verification note
	PRDPath  string // optional; auto-detect when empty
	SpecDir  string // optional; auto-detect when empty
}

// RecordCheckpoint places (or refreshes, by name) a checkpoint marker: it
// captures branch / HEAD / dirty state and auto-detects the active spec dir
// and handoff file when not supplied. Idempotent on (project, name).
func (h *Hub) RecordCheckpoint(ctx context.Context, in CheckpointInput) (*Checkpoint, error) {
	if h == nil || h.DB == nil {
		return nil, errors.New("handoff hub not initialized")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("checkpoint name is required")
	}

	branch, _ := gitOut(ctx, h.RepoDir, "rev-parse", "--abbrev-ref", "HEAD")
	commit, _ := gitOut(ctx, h.RepoDir, "rev-parse", "HEAD")

	specDir := strings.TrimSpace(in.SpecDir)
	if specDir == "" {
		specDir = detectActiveSpecDir(h.RepoDir)
	}
	handoffFile := detectHandoffFile(h.RepoDir)
	prd := strings.TrimSpace(in.PRDPath)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := h.DB.ExecContext(ctx, `
		INSERT INTO checkpoints
			(project, name, branch, "commit", dirty, prd_path, spec_dir, handoff_file,
			 evidence, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'open', ?)
		ON CONFLICT(project, name) DO UPDATE SET
			branch = excluded.branch,
			"commit" = excluded."commit",
			dirty = excluded.dirty,
			prd_path = COALESCE(checkpoints.prd_path, excluded.prd_path),
			spec_dir = COALESCE(checkpoints.spec_dir, excluded.spec_dir),
			handoff_file = COALESCE(checkpoints.handoff_file, excluded.handoff_file),
			evidence = excluded.evidence,
			status = 'open',
			created_at = excluded.created_at`,
		h.Project, name, nullString(branch), nullString(commit),
		dirtyInt(h.RepoDir), nullString(prd), nullString(specDir), nullString(handoffFile),
		nullString(in.Evidence), now)
	if err != nil {
		return nil, fmt.Errorf("upsert checkpoint: %w", err)
	}
	cp, err := h.getCheckpoint(ctx, name)
	if err != nil {
		return nil, err
	}
	return cp, nil
}

// VerifyReport is the drift report comparing a checkpoint's recorded state
// with the repo's current state.
type VerifyReport struct {
	Checkpoint  Checkpoint
	BranchMatch bool
	CommitMatch bool
	DirtyNow    bool
	// ChangedSince lists files changed between the checkpoint commit and HEAD
	// (empty when the commit is unchanged or git is unavailable).
	ChangedSince []string
	// Recommendation is one of: continue | refresh | inspect-drift | pause.
	Recommendation string
}

// VerifyCheckpoint compares the recorded checkpoint against current git state
// and returns the drift report. It does not revert anything. The checkpoint's
// status is updated: verified (no drift), stale (drift detected). Unknown
// checkpoints fail closed.
func (h *Hub) VerifyCheckpoint(ctx context.Context, name string) (*VerifyReport, error) {
	if h == nil || h.DB == nil {
		return nil, errors.New("handoff hub not initialized")
	}
	cp, err := h.getCheckpoint(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("checkpoint %q: %w", name, err)
	}

	curBranch, _ := gitOut(ctx, h.RepoDir, "rev-parse", "--abbrev-ref", "HEAD")
	curCommit, _ := gitOut(ctx, h.RepoDir, "rev-parse", "HEAD")
	dirtyNow := dirtyInt(h.RepoDir) == 1

	branchMatch := cp.Branch == curBranch
	commitMatch := cp.Commit == curCommit

	rep := &VerifyReport{
		Checkpoint:  *cp,
		BranchMatch: branchMatch,
		CommitMatch: commitMatch,
		DirtyNow:    dirtyNow,
	}
	if commitMatch && strings.TrimSpace(cp.Commit) != "" {
		if out, err := gitOut(ctx, h.RepoDir, "diff", "--name-only", cp.Commit, "HEAD"); err == nil {
			for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
				if line != "" {
					rep.ChangedSince = append(rep.ChangedSince, line)
				}
			}
		}
	}

	switch {
	case commitMatch && branchMatch && !dirtyNow:
		rep.Recommendation = "continue"
	case commitMatch && branchMatch:
		// Same commit but working tree dirty: the marker is still valid,
		// the dirt is in-flight work.
		rep.Recommendation = "inspect-drift"
	default:
		rep.Recommendation = "refresh"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	status := "verified"
	if rep.Recommendation != "continue" {
		status = "stale"
	}
	if _, err := h.DB.ExecContext(ctx, `
		UPDATE checkpoints SET status = ?, verified_at = ?
		WHERE project = ? AND name = ?`,
		status, now, h.Project, name); err != nil {
		return rep, fmt.Errorf("record verification: %w", err)
	}
	return rep, nil
}

// ListCheckpoints returns the project's checkpoints, newest-first (limit
// clamped to 1..200).
func (h *Hub) ListCheckpoints(ctx context.Context, limit int) ([]Checkpoint, error) {
	if h == nil || h.DB == nil {
		return nil, errors.New("handoff hub not initialized")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := h.DB.QueryContext(ctx, `
		SELECT id, name, branch, "commit", dirty, prd_path, spec_dir, handoff_file,
		       evidence, status, created_at, COALESCE(verified_at,'')
		FROM checkpoints WHERE project = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, h.Project, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Checkpoint{}
	for rows.Next() {
		var c Checkpoint
		var branch, commit, prd, spec, hf, evidence sql.NullString
		var verifiedAt sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &branch, &commit, &c.Dirty, &prd, &spec,
			&hf, &evidence, &c.Status, &c.CreatedAt, &verifiedAt); err != nil {
			return nil, err
		}
		c.Branch = branch.String
		c.Commit = commit.String
		c.PRDPath = prd.String
		c.SpecDir = spec.String
		c.HandoffFile = hf.String
		c.Evidence = evidence.String
		c.VerifiedAt = verifiedAt.String
		out = append(out, c)
	}
	return out, rows.Err()
}

// ArchiveCheckpoints flips the checkpoints linked to a spec dir to 'archived'
// (called by the ship/finish flow). Returns the number archived.
func (h *Hub) ArchiveCheckpoints(ctx context.Context, specDir string) (int, error) {
	if h == nil || h.DB == nil {
		return 0, errors.New("handoff hub not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := h.DB.ExecContext(ctx, `
		UPDATE checkpoints SET status = 'archived', verified_at = ?
		WHERE project = ? AND status IN ('open', 'verified', 'stale')
		  AND spec_dir = ?`, now, h.Project, specDir)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (h *Hub) getCheckpoint(ctx context.Context, name string) (*Checkpoint, error) {
	row := h.DB.QueryRowContext(ctx, `
		SELECT id, name, branch, "commit", dirty, prd_path, spec_dir, handoff_file,
		       evidence, status, created_at, COALESCE(verified_at,'')
		FROM checkpoints WHERE project = ? AND name = ?`, h.Project, name)
	var c Checkpoint
	var branch, commit, prd, spec, hf, evidence sql.NullString
	var verifiedAt sql.NullString
	if err := row.Scan(&c.ID, &c.Name, &branch, &commit, &c.Dirty, &prd, &spec,
		&hf, &evidence, &c.Status, &c.CreatedAt, &verifiedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("unknown checkpoint %q", name)
		}
		return nil, err
	}
	c.Branch = branch.String
	c.Commit = commit.String
	c.PRDPath = prd.String
	c.SpecDir = spec.String
	c.HandoffFile = hf.String
	c.Evidence = evidence.String
	c.VerifiedAt = verifiedAt.String
	return &c, nil
}

// dirtyInt returns 1 when the repo working tree is dirty (porcelain non-empty),
// 0 otherwise or when git is unavailable.
func dirtyInt(repoDir string) int {
	if strings.TrimSpace(repoDir) == "" {
		return 0
	}
	out, err := gitOut(context.Background(), repoDir, "status", "--porcelain")
	if err != nil || strings.TrimSpace(out) == "" {
		return 0
	}
	return 1
}

// detectActiveSpecDir finds the newest .skillgrid/sdd/<change>/ directory
// (by directory mtime) that is not 'debug' or 'tasks'. Empty when none.
func detectActiveSpecDir(repoDir string) string {
	sdd := filepath.Join(repoDir, ".skillgrid", "sdd")
	entries, err := os.ReadDir(sdd)
	if err != nil {
		return ""
	}
	var best string
	var bestMtime int64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n := e.Name()
		if n == "debug" || n == "tasks" || n == "archive" || n == ".gitkeep" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		m := info.ModTime().Unix()
		if m > bestMtime {
			bestMtime = m
			best = filepath.Join(".skillgrid", "sdd", n)
		}
	}
	return best
}

// detectHandoffFile returns the repo-relative .cleave path when a bundle is
// present (i.e. a handoff exists to point at), "" otherwise.
func detectHandoffFile(repoDir string) string {
	if info, err := os.Stat(filepath.Join(repoDir, ".skillgrid", ".cleave")); err == nil && info.IsDir() {
		return ".skillgrid/.cleave"
	}
	return ""
}
