package tracker

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// jiraAdapter shells out to the `jira` CLI.
// The project key comes from SKILLGRID_JIRA_PROJECT or the onboarded tracker
// doc ("project key PROJ") — never guessed; without it every call 501s.
// `jira` is absent from this dev machine, so the table parsers below are
// header-driven and degrade to 502 with reason on unparsable output.
type jiraAdapter struct {
	// projectKey resolves the Jira project key; nil uses the default chain.
	projectKey func() (string, error)
}

func (a *jiraAdapter) Name() string    { return ProviderJira }
func (a *jiraAdapter) CLIName() string { return "jira" }

var jiraKeyRe = regexp.MustCompile(`(?i)project key\s+([A-Z][A-Z0-9]+)`)

func (a *jiraAdapter) key() (string, error) {
	if a.projectKey != nil {
		return a.projectKey()
	}
	if v := strings.TrimSpace(os.Getenv("SKILLGRID_JIRA_PROJECT")); v != "" {
		return v, nil
	}
	if cwd, err := os.Getwd(); err == nil {
		if doc, err := os.ReadFile(cwd + "/docs/skillgrid/agents/issue-tracker.md"); err == nil {
			if m := jiraKeyRe.FindStringSubmatch(string(doc)); m != nil {
				return m[1], nil
			}
		}
	}
	return "", &errUnresolvable{Reason: "jira project key not configured (set SKILLGRID_JIRA_PROJECT or record it in docs/skillgrid/agents/issue-tracker.md)"}
}

func (a *jiraAdapter) Config(ctx context.Context) (TrackerConfig, error) {
	key, err := a.key()
	if err != nil {
		return TrackerConfig{}, err
	}
	items, err := a.List(ctx)
	if err != nil {
		return TrackerConfig{}, err
	}
	seen := map[string]bool{}
	var statuses []string
	for _, it := range items {
		if !seen[it.Status] {
			seen[it.Status] = true
			statuses = append(statuses, it.Status)
		}
	}
	return TrackerConfig{Provider: ProviderJira, Statuses: statuses, ProjectKey: key}, nil
}

var jiraColSplit = regexp.MustCompile(`\s{2,}`)

// parseJiraTable maps header names to column indexes, then reads rows.
// Returns items with id/title/status/priority/updated when those headers exist.
func parseJiraTable(out string) ([]Item, error) {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("no rows")
	}
	header := jiraColSplit.Split(strings.TrimSpace(lines[0]), -1)
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	ki, okK := idx["key"]
	si, okS := idx["summary"]
	if !okK || !okS {
		return nil, fmt.Errorf("no KEY/SUMMARY columns")
	}
	ti, _ := idx["status"]
	pi, _ := idx["priority"]
	ui, _ := idx["updated"]
	var items []Item
	for _, ln := range lines[1:] {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		cols := jiraColSplit.Split(strings.TrimSpace(ln), -1)
		get := func(i int) string {
			if i < len(cols) {
				return strings.TrimSpace(cols[i])
			}
			return ""
		}
		items = append(items, Item{
			ID: get(ki), Title: get(si), Status: get(ti),
			Board:    boardFor(ProviderJira, get(ti)),
			Priority: get(pi), UpdatedAt: get(ui), Provider: ProviderJira,
		})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no rows")
	}
	return items, nil
}

func (a *jiraAdapter) List(ctx context.Context) ([]Item, error) {
	key, err := a.key()
	if err != nil {
		return nil, err
	}
	out, err := runCLI(ctx, "jira", "issue", "list", "-q",
		fmt.Sprintf("project = %s ORDER BY updated DESC", key), "--plain")
	if err != nil {
		return nil, err
	}
	items, perr := parseJiraTable(out)
	if perr != nil {
		return nil, &errBadOutput{CLI: "jira", Provider: ProviderJira, Reason: "issue list unparsable: " + perr.Error()}
	}
	return items, nil
}

// parseJiraView reads `Key: Value` lines (summary/status/assignee/priority/updated).
func parseJiraView(id, out string) (Item, error) {
	it := Item{ID: id, Provider: ProviderJira}
	kv := regexp.MustCompile(`(?im)^\s*(summary|status|assignee|priority|updated)\s*:\s*(.+?)\s*$`)
	found := false
	for _, m := range kv.FindAllStringSubmatch(out, -1) {
		found = true
		switch strings.ToLower(m[1]) {
		case "summary":
			it.Title = strings.TrimSpace(m[2])
		case "status":
			it.Status = strings.TrimSpace(m[2])
		case "assignee":
			it.Assignees = []string{strings.TrimSpace(m[2])}
		case "priority":
			it.Priority = strings.TrimSpace(m[2])
		case "updated":
			it.UpdatedAt = strings.TrimSpace(m[2])
		}
	}
	if !found || it.Title == "" || it.Status == "" {
		return Item{}, fmt.Errorf("no summary/status fields")
	}
	it.Board = boardFor(ProviderJira, it.Status)
	return it, nil
}

func (a *jiraAdapter) Get(ctx context.Context, id string) (Item, error) {
	if _, err := a.key(); err != nil {
		return Item{}, err
	}
	out, err := runCLI(ctx, "jira", "issue", "view", id, "--comments", "10")
	if err != nil {
		if isNotFoundText(failedOutput(err)) {
			return Item{}, &errNotFound{ID: id}
		}
		return Item{}, err
	}
	it, perr := parseJiraView(id, out)
	if perr != nil {
		return Item{}, &errBadOutput{CLI: "jira", Provider: ProviderJira, Reason: "issue view unparsable: " + perr.Error()}
	}
	return it, nil
}

func (a *jiraAdapter) SetStatus(ctx context.Context, id, status string) (Item, error) {
	if _, err := a.key(); err != nil {
		return Item{}, err
	}
	if _, err := runCLI(ctx, "jira", "issue", "move", id, status); err != nil {
		if isNotFoundText(failedOutput(err)) {
			return Item{}, &errNotFound{ID: id}
		}
		lower := strings.ToLower(failedOutput(err))
		if strings.Contains(lower, "invalid transition") || strings.Contains(lower, "not a valid") {
			return Item{}, &errBadStatus{Status: status, Valid: nil}
		}
		return Item{}, err
	}
	return a.Get(ctx, id)
}
