package tracker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// backlogAdapter is the FILE-BASED Backlog.md provider (Phase 2). It reads
// `.backlog/tasks/*.md` YAML frontmatter directly — no CLI dependency — so the
// board works even when the `backlog` CLI is absent or segfaults. Status
// writes shell out to `backlog task edit <id> -s <status>` (the one file
// mutation, per the Global Constraints).
//
// Frontmatter keys (verified against real task files 2026-09-16):
//   id, title, status, priority, type, assignee ([]), labels ([]),
//   dependencies ([]), milestone, parent, due_date / dueDate,
//   created_date / createdAt, updated_date / updatedAt.
type backlogAdapter struct {
	// tasksDir overrides .backlog/tasks (tests inject a temp dir).
	tasksDir string
}

func (a *backlogAdapter) Name() string    { return ProviderBacklogMD }
func (a *backlogAdapter) CLIName() string { return "" } // file-based, no CLI for reads

func (a *backlogAdapter) dir() string {
	if a.tasksDir != "" {
		return a.tasksDir
	}
	return ".backlog/tasks"
}

func (a *backlogAdapter) milestonesDir() string {
	return filepath.Join(filepath.Dir(a.dir()), "milestones")
}

// backlogStatuses is the canonical Backlog.md status vocabulary (the statuses
// the dashboard offers for drag-drop and the `task edit` write path). This is
// the Backlog.md status set, not a guess — it matches the frontmatter `status`
// values the task tooling emits.
var backlogStatuses = []string{
	"Draft", "needs-triage", "needs-info", "ready-for-agent", "ready-for-human",
	"in-progress", "done", "blocked", "wontfix",
}

func (a *backlogAdapter) Config(ctx context.Context) (TrackerConfig, error) {
	return TrackerConfig{
		Provider:   ProviderBacklogMD,
		Statuses:   append([]string{}, backlogStatuses...),
		Types:      []string{"feature", "bug", "enhancement", "refactor", "docs", "chore"},
		Priorities: []string{"high", "medium", "low"},
		Version:    "file-based",
		Schema:     "1",
	}, nil
}

// parseFrontmatter reads the leading `---`-fenced YAML block of a task file.
// Returns the raw block + the markdown body. No external YAML dep: the values
// are flat scalars and simple inline/block lists, parsed line-by-line.
func parseFrontmatter(raw string) (fm map[string]string, fmLists map[string][]string, body string, ok bool) {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, nil, raw, false
	}
	fm = map[string]string{}
	fmLists = map[string][]string{}
	var currentList string
	end := -1
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "---" {
			end = i
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Block list item (indented "- value")
		if strings.HasPrefix(trimmed, "- ") {
			if currentList != "" {
				fmLists[currentList] = append(fmLists[currentList], unquote(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))))
			}
			continue
		}
		// key: value or key:
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if val == "" {
			currentList = key // expect block list items
			continue
		}
		currentList = ""
		if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
			// Inline list: [a, b, c] or []
			inner := strings.TrimSuffix(strings.TrimPrefix(val, "["), "]")
			var vals []string
			for _, p := range strings.Split(inner, ",") {
				if v := strings.TrimSpace(unquote(strings.TrimSpace(p))); v != "" {
					vals = append(vals, v)
				}
			}
			fmLists[key] = vals
		} else {
			fm[key] = unquote(val)
		}
	}
	if end < 0 {
		return nil, nil, raw, false
	}
	body = strings.Join(lines[end+1:], "\n")
	return fm, fmLists, body, true
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// firstKey returns the first present value among the given keys (for the
// created/updated date aliases).
func firstKey(fm map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(fm[k]); v != "" {
			return v
		}
	}
	return ""
}

var (
	acBeginRE = regexp.MustCompile(`(?i)<!--\s*AC:BEGIN\s*-->`)
	acEndRE   = regexp.MustCompile(`(?i)<!--\s*AC:END\s*-->`)
	acItemRE  = regexp.MustCompile(`^\s*[-*]\s*\[[ xX]\]`)
	acDoneRE  = regexp.MustCompile(`^\s*[-*]\s*\[[xX]\]`)
)

// countAC parses the AC:BEGIN/AC:END block in a task body and returns the
// number of checked and total checkbox items. Returns (0, 0) when the block
// is absent.
func countAC(body string) (completed, total int) {
	begin := acBeginRE.FindStringIndex(body)
	if begin == nil {
		return 0, 0
	}
	rest := body[begin[1]:]
	end := acEndRE.FindStringIndex(rest)
	if end == nil {
		return 0, 0
	}
	block := rest[:end[0]]
	for _, line := range strings.Split(block, "\n") {
		if acItemRE.MatchString(line) {
			total++
			if acDoneRE.MatchString(line) {
				completed++
			}
		}
	}
	return completed, total
}

func backlogTaskFrom(fm map[string]string, fmLists map[string][]string, body string) UnifiedTask {
	status := strings.TrimSpace(fm["status"])
	acCompleted, acTotal := countAC(body)
	return UnifiedTask{
		ID:           strings.TrimSpace(fm["id"]),
		Title:        strings.TrimSpace(fm["title"]),
		Status:       status,
		Description:  strings.TrimSpace(body),
		Type:         strings.TrimSpace(fm["type"]),
		Priority:     strings.ToLower(strings.TrimSpace(fm["priority"])),
		Assignees:    fmLists["assignee"],
		Labels:       fmLists["labels"],
		DocRefs:      fmLists["references"],
		Board:        boardFor(ProviderBacklogMD, status),
		Dependencies: fmLists["dependencies"],
		ACCompleted:  acCompleted,
		ACTotal:      acTotal,
		Milestone:    strings.TrimSpace(fm["milestone"]),
		Parent:       strings.TrimSpace(fm["parent"]),
		DueDate:      strings.TrimSpace(firstKey(fm, "due_date", "dueDate")),
		CreatedAt:    firstKey(fm, "created_date", "createdAt"),
		UpdatedAt:    firstKey(fm, "updated_date", "updatedAt"),
		Provider:     ProviderBacklogMD,
	}
}

// loadTasks reads every .md in the tasks dir and parses frontmatter.
func (a *backlogAdapter) loadTasks() ([]UnifiedTask, error) {
	dir := a.dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []UnifiedTask{}, nil
		}
		return nil, &errBadOutput{CLI: "", Provider: ProviderBacklogMD, Reason: fmt.Sprintf("read %s: %v", dir, err)}
	}
	var tasks []UnifiedTask
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		fm, fmLists, body, ok := parseFrontmatter(string(raw))
		if !ok {
			continue
		}
		t := backlogTaskFrom(fm, fmLists, body)
		if t.ID == "" {
			continue
		}
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, nil
}

func (a *backlogAdapter) List(ctx context.Context) ([]UnifiedTask, error) {
	return a.loadTasks()
}

func (a *backlogAdapter) Get(ctx context.Context, id string) (UnifiedTask, error) {
	tasks, err := a.loadTasks()
	if err != nil {
		return UnifiedTask{}, err
	}
	for _, t := range tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return UnifiedTask{}, &errNotFound{ID: id}
}

func (a *backlogAdapter) SetStatus(ctx context.Context, id, status string) (UnifiedTask, error) {
	// Validate against the canonical Backlog.md status set (never invented).
	valid := false
	for _, s := range backlogStatuses {
		if strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(status)) {
			valid = true
			break
		}
	}
	if !valid {
		return UnifiedTask{}, &errBadStatus{Status: status, Valid: backlogStatuses}
	}
	// The task must exist (file-based check before shelling out).
	if _, err := a.Get(ctx, id); err != nil {
		return UnifiedTask{}, err
	}
	// Status write via `backlog task edit` (the one file mutation). If the CLI
	// is missing, degrade to 503 (never a direct file write bypassing the tool).
	if _, err := runCLI(ctx, "backlog", "task", "edit", id, "-s", status, "--plain"); err != nil {
		if isNotFoundText(failedOutput(err)) {
			return UnifiedTask{}, &errNotFound{ID: id}
		}
		return UnifiedTask{}, err
	}
	return a.Get(ctx, id)
}

// Milestones reads .backlog/milestones/*.md frontmatter and returns the
// milestone list sorted by id. Returns an empty slice when the directory
// does not exist.
func (a *backlogAdapter) Milestones(ctx context.Context) ([]Milestone, error) {
	dir := a.milestonesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Milestone{}, nil
		}
		return nil, &errBadOutput{CLI: "", Provider: ProviderBacklogMD, Reason: fmt.Sprintf("read %s: %v", dir, err)}
	}
	var ms []Milestone
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		fm, _, _, ok := parseFrontmatter(string(raw))
		if !ok {
			continue
		}
		id := strings.TrimSpace(fm["id"])
		if id == "" {
			continue
		}
		ms = append(ms, Milestone{
			ID:          id,
			Title:       strings.TrimSpace(fm["title"]),
			Description: strings.TrimSpace(fm["description"]),
		})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].ID < ms[j].ID })
	return ms, nil
}

// Dependencies: for Backlog.md, compute edges from the frontmatter `dependencies`
// list across all tasks (DepsOut = this task's deps; DepsIn = tasks that list
// this task as a dependency).
func (a *backlogAdapter) Dependencies(ctx context.Context, id string) (UnifiedTaskDeps, error) {
	tasks, err := a.loadTasks()
	if err != nil {
		return UnifiedTaskDeps{}, err
	}
	found := false
	deps := UnifiedTaskDeps{TaskID: id, DepsIn: []string{}, DepsOut: []string{}}
	for _, t := range tasks {
		if t.ID == id {
			found = true
			deps.DepsOut = t.Dependencies
		}
		for _, d := range t.Dependencies {
			if d == id {
				deps.DepsIn = append(deps.DepsIn, t.ID)
			}
		}
	}
	if !found {
		return UnifiedTaskDeps{}, &errNotFound{ID: id}
	}
	return deps, nil
}
