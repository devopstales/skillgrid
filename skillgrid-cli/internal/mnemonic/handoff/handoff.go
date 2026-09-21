// Package handoff is the Handoff Hub (change 015-handoff-hub): it unifies the
// three existing checkpoint sources into one queryable surface.
//
//   - change_snapshots — a git-derived, append-only log of EVERY commit
//     (Record/Backfill), reconstructed from `git log` on demand.
//   - checkpoints      — named, intentional markers (RecordCheckpoint) with
//     drift verification (VerifyCheckpoint) and per-change archiving.
//   - handoff_refs     — the join between a handoff artifact (session cleave
//     bundle, team task, checkpoint marker) and the commit range + spec dir it
//     covers (RecordHandoffRef).
//
// The Hub holds the store DB plus the repo root. Git access goes through small
// exec helpers (same pattern as http/git.go); every git call degrades to a
// NULL/zero field rather than failing the hub — the hub must never be blocked
// by a missing git binary or an unborn HEAD.
package handoff

import (
	"context"
	"database/sql"
	"os/exec"
	"strings"
)

// DB is the SQL surface the hub needs. *sql.DB satisfies it.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Hub is one project's handoff hub.
type Hub struct {
	DB      DB
	Project string
	RepoDir string
}

// Snapshot is one row of change_snapshots.
type Snapshot struct {
	ID          int64
	Branch      string
	Commit      string
	CommitShort string
	Subject     string
	ContextJSON string
	ChangedFiles string
	Author      string
	CommittedAt string
}

// Context is the parsed [skillgrid-context] block of a commit message.
type Context struct {
	Task      string `json:"task,omitempty"`
	Decisions string `json:"decisions,omitempty"`
	Remaining string `json:"remaining,omitempty"`
	Tried     string `json:"tried,omitempty"`
}

// ChangedFile is one entry of the changed_files JSON array.
type ChangedFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// Checkpoint is one row of checkpoints.
type Checkpoint struct {
	ID          int64
	Name        string
	Branch      string
	Commit      string
	Dirty       bool
	PRDPath     string
	SpecDir     string
	HandoffFile string
	Evidence    string
	Status      string
	CreatedAt   string
	VerifiedAt  string
}

// HandoffRef is one row of handoff_refs.
type HandoffRef struct {
	HandoffID  string
	HandoffType string
	FromCommit string
	ToCommit   string
	SpecDir    string
	TeamID     string
	TaskID     string
	CreatedAt  string
}

func gitOut(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
