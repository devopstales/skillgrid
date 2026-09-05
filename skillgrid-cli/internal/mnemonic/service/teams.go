package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/files"
)

// Sentinel errors for teams lifecycle.
var (
	ErrNoPendingTasks = errors.New("no pending tasks")
	ErrUnknownTask    = errors.New("unknown task id")
)

const (
	taskStatusPending     = "pending"
	taskStatusInProgress  = "in-progress"
	taskStatusReviewSpec  = "review_spec"
	taskStatusDone        = "done"
	defaultTeamID         = "default"
	defaultTeamName       = "default"
)

// SpawnTaskParams creates a pending team task with a filesystem brief.
type SpawnTaskParams struct {
	Directory string
	TeamID    string
	Title     string
	Brief     string
	Priority  int
	CreatedBy string
}

// TaskView is metadata returned from pull/read without embedding full content.
type TaskView struct {
	ID         string
	TeamID     string
	Title      string
	Status     string
	BriefPath  string
	OutputPath string
	Priority   int
	AssignedTo string
}

// SpawnTask writes brief.md via ContentPlane then inserts a pending task row.
func (s *Service) SpawnTask(ctx context.Context, p SpawnTaskParams) (string, error) {
	_ = ctx
	if strings.TrimSpace(p.Title) == "" {
		return "", fmt.Errorf("title is required")
	}
	if strings.TrimSpace(p.Brief) == "" {
		return "", fmt.Errorf("brief is required")
	}
	h, cleanup, err := s.openTeamsHandle(p.Directory)
	if err != nil {
		return "", err
	}
	defer cleanup()

	teamID := strings.TrimSpace(p.TeamID)
	if teamID == "" {
		teamID = defaultTeamID
	}
	if err := h.ensureTeam(teamID, defaultTeamName); err != nil {
		return "", err
	}

	taskID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = h.content.Write(files.KindTasks, taskID, "brief.md", []byte(p.Brief), func(relPath string) error {
		_, execErr := h.store.DB.Exec(`
			INSERT INTO tasks (id, team_id, title, brief_path, status, created_by, priority, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			taskID, teamID, p.Title, relPath, taskStatusPending, nullIfEmpty(p.CreatedBy), p.Priority, now, now,
		)
		return execErr
	})
	if err != nil {
		return "", err
	}
	return taskID, nil
}

// PullNextTask claims the highest-priority pending task for memberID.
func (s *Service) PullNextTask(ctx context.Context, directory, memberID string) (*TaskView, error) {
	_ = ctx
	if strings.TrimSpace(memberID) == "" {
		return nil, fmt.Errorf("member id is required")
	}
	h, cleanup, err := s.openTeamsHandle(directory)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := h.ensureMember(defaultTeamID, memberID, "agent"); err != nil {
		return nil, err
	}

	tx, err := h.store.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var view TaskView
	err = tx.QueryRow(`
		SELECT id, team_id, title, status, brief_path, COALESCE(output_path, ''), priority, COALESCE(assigned_to, '')
		FROM tasks
		WHERE status = ?
		ORDER BY priority DESC, created_at ASC
		LIMIT 1`, taskStatusPending).Scan(
		&view.ID, &view.TeamID, &view.Title, &view.Status, &view.BriefPath, &view.OutputPath, &view.Priority, &view.AssignedTo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoPendingTasks
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(`
		UPDATE tasks SET status = ?, assigned_to = ?, updated_at = ? WHERE id = ? AND status = ?`,
		taskStatusInProgress, memberID, now, view.ID, taskStatusPending,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	view.Status = taskStatusInProgress
	view.AssignedTo = memberID
	return &view, nil
}

// ReadTask returns task metadata and brief markdown.
func (s *Service) ReadTask(ctx context.Context, directory, taskID string) (*TaskView, string, error) {
	_ = ctx
	h, cleanup, err := s.openTeamsHandle(directory)
	if err != nil {
		return nil, "", err
	}
	defer cleanup()

	view, err := h.loadTask(taskID)
	if err != nil {
		return nil, "", err
	}
	brief, err := h.content.Read(view.BriefPath)
	if err != nil {
		return nil, "", err
	}
	return view, string(brief), nil
}

// SubmitOutput writes output.md and advances status to review_spec.
func (s *Service) SubmitOutput(ctx context.Context, directory, taskID, summary, output string) error {
	_ = ctx
	if strings.TrimSpace(output) == "" {
		return fmt.Errorf("output is required")
	}
	h, cleanup, err := s.openTeamsHandle(directory)
	if err != nil {
		return err
	}
	defer cleanup()

	if _, err := h.loadTask(taskID); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	resultID := uuid.New().String()
	_, err = h.content.Write(files.KindTasks, taskID, "output.md", []byte(output), func(relPath string) error {
		tx, txErr := h.store.DB.Begin()
		if txErr != nil {
			return txErr
		}
		defer func() { _ = tx.Rollback() }()
		if _, txErr = tx.Exec(`
			UPDATE tasks SET output_path = ?, status = ?, updated_at = ? WHERE id = ?`,
			relPath, taskStatusReviewSpec, now, taskID,
		); txErr != nil {
			return txErr
		}
		if _, txErr = tx.Exec(`
			INSERT INTO task_results (id, task_id, output_path, summary, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			resultID, taskID, relPath, nullIfEmpty(summary), now,
		); txErr != nil {
			return txErr
		}
		return tx.Commit()
	})
	return err
}

// SubmitReviewParams records a peer review for a task.
type SubmitReviewParams struct {
	Directory  string
	TaskID     string
	ReviewerID string
	ReviewType string
	Passed     bool
	Comments   string
}

// SubmitReview writes review markdown and inserts a reviews row.
func (s *Service) SubmitReview(ctx context.Context, p SubmitReviewParams) error {
	_ = ctx
	if strings.TrimSpace(p.TaskID) == "" {
		return fmt.Errorf("task id is required")
	}
	if strings.TrimSpace(p.ReviewerID) == "" {
		return fmt.Errorf("reviewer id is required")
	}
	if strings.TrimSpace(p.ReviewType) == "" {
		p.ReviewType = "spec_compliance"
	}
	h, cleanup, err := s.openTeamsHandle(p.Directory)
	if err != nil {
		return err
	}
	defer cleanup()

	if _, err := h.loadTask(p.TaskID); err != nil {
		return err
	}
	if err := h.ensureMember(defaultTeamID, p.ReviewerID, "reviewer"); err != nil {
		return err
	}

	filename := "spec_review.md"
	if p.ReviewType == "code_quality" {
		filename = "code_review.md"
	}
	reviewID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	passed := 0
	if p.Passed {
		passed = 1
	}
	_, err = h.content.Write(files.KindReviews, p.TaskID, filename, []byte(p.Comments), func(relPath string) error {
		_, execErr := h.store.DB.Exec(`
			INSERT INTO reviews (id, task_id, reviewer_id, review_type, passed, comments_path, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			reviewID, p.TaskID, p.ReviewerID, p.ReviewType, passed, relPath, now,
		)
		return execErr
	})
	return err
}

// MarkDone sets task status to done.
func (s *Service) MarkDone(ctx context.Context, directory, taskID string) error {
	_ = ctx
	h, cleanup, err := s.openTeamsHandle(directory)
	if err != nil {
		return err
	}
	defer cleanup()

	if _, err := h.loadTask(taskID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := h.store.DB.Exec(`
		UPDATE tasks SET status = ?, updated_at = ?, completed_at = ? WHERE id = ?`,
		taskStatusDone, now, now, taskID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUnknownTask
	}
	return nil
}

func (s *Service) openTeamsHandle(directory string) (*projectHandle, func(), error) {
	dir := strings.TrimSpace(directory)
	if dir == "" {
		return s.openProjectFromCWD()
	}
	return s.openProjectForDirectory(dir)
}

func (h *projectHandle) ensureTeam(id, name string) error {
	_, err := h.store.DB.Exec(`
		INSERT INTO teams (id, name) VALUES (?, ?)
		ON CONFLICT(id) DO NOTHING`, id, name)
	return err
}

func (h *projectHandle) ensureMember(teamID, memberID, role string) error {
	if err := h.ensureTeam(teamID, defaultTeamName); err != nil {
		return err
	}
	_, err := h.store.DB.Exec(`
		INSERT INTO team_members (id, team_id, role, status) VALUES (?, ?, ?, 'idle')
		ON CONFLICT(id) DO NOTHING`, memberID, teamID, role)
	return err
}

func (h *projectHandle) loadTask(taskID string) (*TaskView, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, ErrUnknownTask
	}
	var view TaskView
	err := h.store.DB.QueryRow(`
		SELECT id, team_id, title, status, brief_path, COALESCE(output_path, ''), priority, COALESCE(assigned_to, '')
		FROM tasks WHERE id = ?`, taskID).Scan(
		&view.ID, &view.TeamID, &view.Title, &view.Status, &view.BriefPath, &view.OutputPath, &view.Priority, &view.AssignedTo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnknownTask
	}
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
