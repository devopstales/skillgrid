package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// gitlabAdapter shells out to the `glab` CLI (repo inferred from git remote).
// Verified (2026-09-11): `glab issue list --help` documents `-O json` output
// and `glab issue view --help` documents `-F json`; `glab issue reopen`
// exists. Unknown issues fail non-zero (404 text).
type gitlabAdapter struct{}

func (a *gitlabAdapter) Name() string    { return ProviderGitLab }
func (a *gitlabAdapter) CLIName() string { return "glab" }

func (a *gitlabAdapter) Config(ctx context.Context) (TrackerConfig, error) {
	return TrackerConfig{Provider: ProviderGitLab, Statuses: []string{"opened", "closed"}}, nil
}

type glabIssue struct {
	IID         int      `json:"iid"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	State       string   `json:"state"`
	Labels      []string `json:"labels"`
	UpdatedAt   string   `json:"updated_at"`
	Assignees   []struct {
		Username string `json:"username"`
	} `json:"assignees"`
}

func glabItem(g glabIssue) UnifiedTask {
	users := make([]string, 0, len(g.Assignees))
	for _, u := range g.Assignees {
		users = append(users, u.Username)
	}
	return UnifiedTask{
		ID: fmt.Sprintf("%d", g.IID), Title: g.Title, Description: g.Description,
		Status: strings.ToLower(g.State), Labels: g.Labels, Assignees: users,
		Board:     boardFor(ProviderGitLab, g.State),
		UpdatedAt: g.UpdatedAt, Provider: ProviderGitLab,
	}
}

func (a *gitlabAdapter) List(ctx context.Context) ([]UnifiedTask, error) {
	out, err := runCLI(ctx, "glab", "issue", "list", "--all", "-O", "json", "-P", "100")
	if err != nil {
		return nil, err
	}
	var issues []glabIssue
	if err := json.Unmarshal([]byte(out), &issues); err != nil {
		return nil, &errBadOutput{CLI: "glab", Provider: ProviderGitLab, Reason: "issue list is not JSON"}
	}
	tasks := make([]UnifiedTask, 0, len(issues))
	for _, g := range issues {
		tasks = append(tasks, glabItem(g))
	}
	return tasks, nil
}

func (a *gitlabAdapter) Get(ctx context.Context, id string) (UnifiedTask, error) {
	out, err := runCLI(ctx, "glab", "issue", "view", id, "-F", "json")
	if err != nil {
		if isNotFoundText(failedOutput(err)) {
			return UnifiedTask{}, &errNotFound{ID: id}
		}
		return UnifiedTask{}, err
	}
	var g glabIssue
	if err := json.Unmarshal([]byte(out), &g); err != nil {
		return UnifiedTask{}, &errBadOutput{CLI: "glab", Provider: ProviderGitLab, Reason: "issue view is not JSON"}
	}
	return glabItem(g), nil
}

func (a *gitlabAdapter) SetStatus(ctx context.Context, id, status string) (UnifiedTask, error) {
	var cmd string
	switch strings.ToLower(status) {
	case "closed", "close", "done":
		cmd = "close"
	case "opened", "open", "reopen", "todo":
		cmd = "reopen"
	default:
		return UnifiedTask{}, &errUnsupported{Reason: fmt.Sprintf("GitLab supports close/reopen only (got %q)", status)}
	}
	if _, err := runCLI(ctx, "glab", "issue", cmd, id); err != nil {
		if isNotFoundText(failedOutput(err)) {
			return UnifiedTask{}, &errNotFound{ID: id}
		}
		return UnifiedTask{}, err
	}
	return a.Get(ctx, id)
}

// Dependencies: GitLab issues have no native dependency edges here; the
// Kanban dependency mini-graph degrades to empty.
func (a *gitlabAdapter) Dependencies(ctx context.Context, id string) (UnifiedTaskDeps, error) {
	if _, err := a.Get(ctx, id); err != nil {
		return UnifiedTaskDeps{}, err
	}
	return UnifiedTaskDeps{TaskID: id, DepsIn: []string{}, DepsOut: []string{}}, nil
}
