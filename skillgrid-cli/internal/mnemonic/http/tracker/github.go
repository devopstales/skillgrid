package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// githubAdapter shells out to the `gh` CLI (repo inferred from git remote).
// Verified (2026-09-11): `gh issue list --json number,title,state` emits a
// JSON array; unknown issues fail with "Could not resolve" on stderr.
type githubAdapter struct{}

func (a *githubAdapter) Name() string    { return ProviderGitHub }
func (a *githubAdapter) CLIName() string { return "gh" }

func (a *githubAdapter) Config(ctx context.Context) (TrackerConfig, error) {
	return TrackerConfig{Provider: ProviderGitHub, Statuses: []string{"open", "closed"}}, nil
}

type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	UpdatedAt string `json:"updatedAt"`
}

func ghItem(g ghIssue) UnifiedTask {
	labels := make([]string, 0, len(g.Labels))
	for _, l := range g.Labels {
		labels = append(labels, l.Name)
	}
	users := make([]string, 0, len(g.Assignees))
	for _, u := range g.Assignees {
		users = append(users, u.Login)
	}
	return UnifiedTask{
		ID: fmt.Sprintf("%d", g.Number), Title: g.Title, Description: g.Body,
		Status: strings.ToLower(g.State), Labels: labels, Assignees: users,
		Board:     boardFor(ProviderGitHub, g.State),
		UpdatedAt: g.UpdatedAt, Provider: ProviderGitHub,
	}
}

func (a *githubAdapter) List(ctx context.Context) ([]UnifiedTask, error) {
	out, err := runCLI(ctx, "gh", "issue", "list", "--state", "all", "--limit", "100",
		"--json", "number,title,state,labels,assignees,updatedAt")
	if err != nil {
		return nil, err
	}
	var issues []ghIssue
	if err := json.Unmarshal([]byte(out), &issues); err != nil {
		return nil, &errBadOutput{CLI: "gh", Provider: ProviderGitHub, Reason: "issue list is not JSON"}
	}
	tasks := make([]UnifiedTask, 0, len(issues))
	for _, g := range issues {
		tasks = append(tasks, ghItem(g))
	}
	return tasks, nil
}

func (a *githubAdapter) Get(ctx context.Context, id string) (UnifiedTask, error) {
	out, err := runCLI(ctx, "gh", "issue", "view", id,
		"--json", "number,title,body,state,labels,assignees,comments,updatedAt")
	if err != nil {
		if isNotFoundText(failedOutput(err)) {
			return UnifiedTask{}, &errNotFound{ID: id}
		}
		return UnifiedTask{}, err
	}
	var g ghIssue
	if err := json.Unmarshal([]byte(out), &g); err != nil {
		return UnifiedTask{}, &errBadOutput{CLI: "gh", Provider: ProviderGitHub, Reason: "issue view is not JSON"}
	}
	return ghItem(g), nil
}

func (a *githubAdapter) SetStatus(ctx context.Context, id, status string) (UnifiedTask, error) {
	var cmd string
	switch strings.ToLower(status) {
	case "closed", "close", "done":
		cmd = "close"
	case "open", "opened", "reopen", "todo":
		cmd = "reopen"
	default:
		return UnifiedTask{}, &errUnsupported{Reason: fmt.Sprintf("GitHub supports close/reopen only (got %q)", status)}
	}
	if _, err := runCLI(ctx, "gh", "issue", cmd, id); err != nil {
		if isNotFoundText(failedOutput(err)) {
			return UnifiedTask{}, &errNotFound{ID: id}
		}
		return UnifiedTask{}, err
	}
	return a.Get(ctx, id)
}

// Dependencies: GitHub issues have no native dependency edges; the Kanban
// dependency mini-graph degrades to empty (the board still renders).
func (a *githubAdapter) Dependencies(ctx context.Context, id string) (UnifiedTaskDeps, error) {
	if _, err := a.Get(ctx, id); err != nil {
		return UnifiedTaskDeps{}, err
	}
	return UnifiedTaskDeps{TaskID: id, DepsIn: []string{}, DepsOut: []string{}}, nil
}

func isNotFoundText(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "not found") || strings.Contains(l, "could not resolve") ||
		strings.Contains(l, "does not exist") || strings.Contains(l, "404")
}
