package tracker

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// backlogAdapter shells out to the `backlog` CLI.
// Verified CLI facts (2026-09-11):
//   - `task list --json` / `task view <id> --json` emit {schemaVersion:1, kind, ...}
//   - `config list` is TEXT ONLY (--json/--plain are rejected)
//   - `task view <missing> --json` exits 0 with plain "Task X not found" text
//   - status change is `task edit <id> -s <status>`
type backlogAdapter struct{}

func (a *backlogAdapter) Name() string    { return ProviderBacklogMD }
func (a *backlogAdapter) CLIName() string { return "backlog" }

var backlogListRe = regexp.MustCompile(`(?m)^\s*(statuses|types|priorities|labels):\s*\[(.*?)\]`)

func (a *backlogAdapter) Config(ctx context.Context) (TrackerConfig, error) {
	out, err := runCLI(ctx, "backlog", "config", "list")
	if err != nil {
		return TrackerConfig{}, err
	}
	cfg := TrackerConfig{Provider: ProviderBacklogMD}
	for _, m := range backlogListRe.FindAllStringSubmatch(out, -1) {
		vals := splitCSV(m[2])
		switch m[1] {
		case "statuses":
			cfg.Statuses = vals
		case "types":
			cfg.Types = vals
		case "priorities":
			cfg.Priorities = vals
		}
	}
	if len(cfg.Statuses) == 0 {
		return TrackerConfig{}, &errBadOutput{CLI: "backlog", Provider: ProviderBacklogMD, Reason: "no statuses parsed from config list"}
	}
	return cfg, nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

type backlogTask struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
	Type          string   `json:"type"`
	Priority      string   `json:"priority"`
	Assignees     []string `json:"assignees"`
	Labels        []string `json:"labels"`
	ACCompleted   int      `json:"acceptanceCriteriaCompleted"`
	ACCount       int      `json:"acceptanceCriteriaCount"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
	IsReady       *bool    `json:"isReady"`
	References    []string `json:"references"`
	Documentation []string `json:"documentation"`
}

func backlogItem(t backlogTask) Item {
	detail := ""
	if len(t.References) > 0 {
		detail = strings.Join(t.References, ", ")
	}
	docRefs := append(append([]string{}, t.References...), t.Documentation...)
	return Item{
		ID: t.ID, Title: t.Title, Description: t.Description,
		Status: t.Status, StatusDetail: detail,
		Type: t.Type, Priority: t.Priority, Assignees: t.Assignees,
		Labels: t.Labels, DocRefs: docRefs, Board: boardFor(ProviderBacklogMD, t.Status),
		ACCompleted: t.ACCompleted, ACTotal: t.ACCount,
		IsReady: t.IsReady, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
		Provider: ProviderBacklogMD,
	}
}

func (a *backlogAdapter) List(ctx context.Context) ([]Item, error) {
	out, err := runCLI(ctx, "backlog", "task", "list", "--json")
	if err != nil {
		return nil, err
	}
	var doc struct {
		SchemaVersion int           `json:"schemaVersion"`
		Kind          string        `json:"kind"`
		Tasks         []backlogTask `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		return nil, &errBadOutput{CLI: "backlog", Provider: ProviderBacklogMD, Reason: "task list is not JSON"}
	}
	if doc.SchemaVersion != 1 {
		return nil, &errBadOutput{CLI: "backlog", Provider: ProviderBacklogMD, Reason: fmt.Sprintf("unknown schemaVersion %d", doc.SchemaVersion)}
	}
	items := make([]Item, 0, len(doc.Tasks))
	for _, t := range doc.Tasks {
		items = append(items, backlogItem(t))
	}
	return items, nil
}

func (a *backlogAdapter) Get(ctx context.Context, id string) (Item, error) {
	out, err := runCLI(ctx, "backlog", "task", "view", id, "--json")
	if err != nil {
		// Unknown ids fail either as exit-0 plain text or non-zero with the
		// miss reported on either stream (verified live 2026-09-11).
		if isNotFoundText(failedOutput(err)) {
			return Item{}, &errNotFound{ID: id}
		}
		return Item{}, err
	}
	var doc struct {
		SchemaVersion int         `json:"schemaVersion"`
		Kind          string      `json:"kind"`
		Task          backlogTask `json:"task"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		// `task view <missing> --json` exits 0 with plain "not found" text.
		if strings.Contains(strings.ToLower(out), "not found") {
			return Item{}, &errNotFound{ID: id}
		}
		return Item{}, &errBadOutput{CLI: "backlog", Provider: ProviderBacklogMD, Reason: "task view is not JSON"}
	}
	if doc.Task.ID == "" {
		return Item{}, &errNotFound{ID: id}
	}
	return backlogItem(doc.Task), nil
}

func (a *backlogAdapter) SetStatus(ctx context.Context, id, status string) (Item, error) {
	cfg, err := a.Config(ctx)
	if err != nil {
		return Item{}, err
	}
	valid := false
	for _, s := range cfg.Statuses {
		if s == status {
			valid = true
			break
		}
	}
	if !valid {
		return Item{}, &errBadStatus{Status: status, Valid: cfg.Statuses}
	}
	if _, err := runCLI(ctx, "backlog", "task", "edit", id, "-s", status, "--plain"); err != nil {
		if isNotFoundText(failedOutput(err)) {
			return Item{}, &errNotFound{ID: id}
		}
		return Item{}, err
	}
	return a.Get(ctx, id)
}
