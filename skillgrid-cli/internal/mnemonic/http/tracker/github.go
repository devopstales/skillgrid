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

func ghItem(g ghIssue) Item {
	labels := make([]string, 0, len(g.Labels))
	for _, l := range g.Labels {
		labels = append(labels, l.Name)
	}
	users := make([]string, 0, len(g.Assignees))
	for _, u := range g.Assignees {
		users = append(users, u.Login)
	}
	return Item{
		ID: fmt.Sprintf("%d", g.Number), Title: g.Title, Description: g.Body,
		Status: strings.ToLower(g.State), Labels: labels, Assignees: users,
		Board:     boardFor(ProviderGitHub, g.State),
		UpdatedAt: g.UpdatedAt, Provider: ProviderGitHub,
	}
}

func (a *githubAdapter) List(ctx context.Context) ([]Item, error) {
	out, err := runCLI(ctx, "gh", "issue", "list", "--state", "all", "--limit", "100",
		"--json", "number,title,state,labels,assignees,updatedAt")
	if err != nil {
		return nil, err
	}
	var issues []ghIssue
	if err := json.Unmarshal([]byte(out), &issues); err != nil {
		return nil, &errBadOutput{CLI: "gh", Provider: ProviderGitHub, Reason: "issue list is not JSON"}
	}
	items := make([]Item, 0, len(issues))
	for _, g := range issues {
		items = append(items, ghItem(g))
	}
	return items, nil
}

func (a *githubAdapter) Get(ctx context.Context, id string) (Item, error) {
	out, err := runCLI(ctx, "gh", "issue", "view", id,
		"--json", "number,title,body,state,labels,assignees,comments,updatedAt")
	if err != nil {
		if isNotFoundText(failedOutput(err)) {
			return Item{}, &errNotFound{ID: id}
		}
		return Item{}, err
	}
	var g ghIssue
	if err := json.Unmarshal([]byte(out), &g); err != nil {
		return Item{}, &errBadOutput{CLI: "gh", Provider: ProviderGitHub, Reason: "issue view is not JSON"}
	}
	return ghItem(g), nil
}

func (a *githubAdapter) SetStatus(ctx context.Context, id, status string) (Item, error) {
	var cmd string
	switch strings.ToLower(status) {
	case "closed", "close":
		cmd = "close"
	case "open", "opened", "reopen":
		cmd = "reopen"
	default:
		return Item{}, &errUnsupported{Reason: fmt.Sprintf("GitHub supports close/reopen only (got %q)", status)}
	}
	if _, err := runCLI(ctx, "gh", "issue", cmd, id); err != nil {
		if isNotFoundText(failedOutput(err)) {
			return Item{}, &errNotFound{ID: id}
		}
		return Item{}, err
	}
	return a.Get(ctx, id)
}

func isNotFoundText(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "not found") || strings.Contains(l, "could not resolve") ||
		strings.Contains(l, "does not exist") || strings.Contains(l, "404")
}
