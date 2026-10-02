package docs

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const sddPath = ".skillgrid/sdd"

// TeamRun is one execution workspace under .skillgrid/sdd that has a ledger.
// ADR-0020: these files are the live team record. This handler only reads them.
type TeamRun struct {
	Name       string       `json:"name"`
	Heading    string       `json:"heading,omitempty"`
	Ledger     string       `json:"ledger,omitempty"`
	WaveLedger string       `json:"wave_ledger,omitempty"`
	Updated    string       `json:"updated,omitempty"`
	InFlight   int          `json:"in_flight"`
	Done       int          `json:"done"`
	Members    []TeamMember `json:"members,omitempty"`
}

// TeamMember is one row from a parallel-ledger table or a progress task line.
type TeamMember struct {
	Agent   string         `json:"agent,omitempty"`
	Task    string         `json:"task,omitempty"`
	Status  string         `json:"status,omitempty"`
	Owns    string         `json:"owns,omitempty"`
	Needs   string         `json:"needs,omitempty"`
	Test    string         `json:"test,omitempty"`
	Session *MemberSession `json:"session,omitempty"`
}

var progressTaskRE = regexp.MustCompile(`(?m)^[ \t]*[-*]?[ \t]*Task[ \t]+(\S+):[ \t]*(\S+)`)

// ticketLineRE matches the as-built ledger shape:
// "- TICKET-01 (TASK-032): complete — verified at …"
var ticketLineRE = regexp.MustCompile(`(?m)^[ \t]*[-*][ \t]*(TICKET-\d+)(?:[ \t]*\((TASK-[\d.]+)\))?[ \t]*:[ \t]*([A-Za-z0-9_-]+)`)

var progressNeedsRE = regexp.MustCompile(`(?m)^[ \t]*[-*]?[ \t]*Task[ \t]+(\S+):[ \t]*Needs:[ \t]*(.+)$`)

// NewTeamRuns returns GET /sdd/runs. A missing .skillgrid/sdd directory is an
// empty list, so the Teams page can render its empty state.
func NewTeamRuns(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runs, err := ListTeamRuns(r.Context(), cwd)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, TeamRunsPayload(runs))
	})
}

// TeamRunsPayload is the GET /sdd/runs response body for runs.
func TeamRunsPayload(runs []TeamRun) map[string]any {
	inFlight, done := 0, 0
	for _, run := range runs {
		inFlight += run.InFlight
		done += run.Done
	}
	return map[string]any{
		"source":    sddPath,
		"count":     len(runs),
		"in_flight": inFlight,
		"done":      done,
		"runs":      runs,
	}
}

// ListTeamRuns reads one level of .skillgrid/sdd and keeps directories that
// contain progress.md or parallel-ledger.md. Newest ledger first.
func ListTeamRuns(ctx context.Context, cwd string) ([]TeamRun, error) {
	root := filepath.Join(cwd, sddPath)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []TeamRun{}, nil
		}
		return nil, err
	}
	var out []TeamRun
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if !e.IsDir() {
			continue
		}
		run, ok := readTeamRun(filepath.Join(root, e.Name()), e.Name())
		if ok {
			markShipped(cwd, e.Name(), &run)
			out = append(out, run)
		}
	}
	sortRuns(out)
	return out, nil
}

// markShipped closes ledger rows once the change has a directory under
// .skillgrid/archive. The ledger file stays as written; the board shows done.
func markShipped(cwd, name string, run *TeamRun) {
	info, err := os.Stat(filepath.Join(cwd, ".skillgrid", "archive", name))
	if err != nil || !info.IsDir() {
		return
	}
	for i := range run.Members {
		run.Members[i].Status = "complete"
	}
	run.InFlight = 0
	run.Done = len(run.Members)
}

func readTeamRun(dir, name string) (TeamRun, bool) {
	run := TeamRun{Name: name}
	var newest time.Time
	if text, mod, ok := readLedger(filepath.Join(dir, "parallel-ledger.md")); ok {
		run.WaveLedger = filepath.ToSlash(filepath.Join(sddPath, name, "parallel-ledger.md"))
		run.Heading = firstHeading(text)
		run.Members = parseWaveMembers(text)
		newest = mod
	}
	if text, mod, ok := readLedger(filepath.Join(dir, "progress.md")); ok {
		run.Ledger = filepath.ToSlash(filepath.Join(sddPath, name, "progress.md"))
		if run.Heading == "" {
			run.Heading = firstHeading(text)
		}
		if len(run.Members) == 0 {
			run.Members = parseProgressMembers(text)
		}
		if mod.After(newest) {
			newest = mod
		}
	}
	run.Members = applyDependencyGate(run.Members)
	if run.Ledger == "" && run.WaveLedger == "" {
		return TeamRun{}, false
	}
	if !newest.IsZero() {
		run.Updated = newest.UTC().Format(time.RFC3339)
	}
	for _, m := range run.Members {
		inFlight, done := tallyStatus(m.Status)
		run.InFlight += inFlight
		run.Done += done
	}
	return run, true
}

func readLedger(path string) (string, time.Time, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", time.Time{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", time.Time{}, false
	}
	return string(data), info.ModTime(), true
}

func firstHeading(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func parseWaveMembers(text string) []TeamMember {
	var header []string
	var rows []TeamMember
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "|") {
			continue
		}
		cells := splitRow(line)
		if len(cells) == 0 || isSeparator(cells) {
			continue
		}
		if header == nil {
			if strings.EqualFold(cells[0], "agent") {
				header = cells
			}
			continue
		}
		rows = append(rows, memberFromRow(header, cells))
	}
	return rows
}

func memberFromRow(header, cells []string) TeamMember {
	var m TeamMember
	for i, name := range header {
		if i >= len(cells) {
			break
		}
		val := cells[i]
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "agent":
			m.Agent = val
		case "task":
			m.Task = val
		case "status":
			m.Status = val
		case "owns":
			m.Owns = val
		case "needs":
			m.Needs = val
		case "test result", "test":
			m.Test = val
		}
	}
	return m
}

func parseProgressMembers(text string) []TeamMember {
	needs := map[string]string{}
	for _, m := range progressNeedsRE.FindAllStringSubmatch(text, -1) {
		id := strings.TrimRight(strings.TrimSpace(m[1]), ":")
		needs[id] = strings.TrimSpace(m[2])
	}
	var out []TeamMember
	for _, m := range progressTaskRE.FindAllStringSubmatch(text, -1) {
		status := strings.TrimRight(m[2], ":")
		if isCoordToken(status) || !knownStatus(status) {
			continue
		}
		task := strings.TrimRight(m[1], ":")
		out = append(out, TeamMember{Task: task, Status: status, Needs: needs[task]})
	}
	for _, m := range ticketLineRE.FindAllStringSubmatch(text, -1) {
		status := m[3]
		if !knownStatus(status) {
			continue
		}
		task := m[1]
		if m[2] != "" {
			task = m[1] + " (" + m[2] + ")"
		}
		out = append(out, TeamMember{Task: task, Status: status})
	}
	return out
}

// applyDependencyGate marks a row blocked while any task named in Needs is
// not complete. A finished row keeps its status.
func applyDependencyGate(members []TeamMember) []TeamMember {
	done := map[string]bool{}
	for _, m := range members {
		if !isDoneStatus(m.Status) {
			continue
		}
		for _, id := range taskIDs(m.Task) {
			done[id] = true
		}
	}
	for i, m := range members {
		if m.Needs == "" || isDoneStatus(m.Status) {
			continue
		}
		for _, id := range splitNeedIDs(m.Needs) {
			if !done[id] {
				members[i].Status = "blocked"
				break
			}
		}
	}
	return members
}

func isDoneStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "done", "complete", "completed":
		return true
	default:
		return false
	}
}

func taskIDs(task string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(task, func(r rune) bool {
		return r == ' ' || r == '(' || r == ')' || r == ','
	}) {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func splitNeedIDs(needs string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(needs, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	}) {
		part = strings.ToLower(strings.Trim(strings.TrimSpace(part), "."))
		if part == "" || part == "and" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func isCoordToken(s string) bool {
	switch strings.ToLower(s) {
	case "owns", "needs":
		return true
	default:
		return false
	}
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if strings.Trim(c, "-:") != "" {
			return false
		}
	}
	return true
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func knownStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "done", "complete", "completed", "dispatched", "in-progress", "in_progress", "review", "review_spec", "failed", "blocked":
		return true
	default:
		return false
	}
}

func tallyStatus(status string) (inFlight, done int) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "done", "complete", "completed":
		return 0, 1
	case "dispatched", "in-progress", "in_progress", "review", "review_spec", "blocked", "failed":
		return 1, 0
	default:
		return 0, 0
	}
}

func sortRuns(runs []TeamRun) {
	// Newest updated first. Equal timestamps keep the name order stable.
	for i := 1; i < len(runs); i++ {
		j := i
		for j > 0 && runLess(runs[j], runs[j-1]) {
			runs[j], runs[j-1] = runs[j-1], runs[j]
			j--
		}
	}
}

func runLess(a, b TeamRun) bool {
	if a.Updated != b.Updated {
		return a.Updated > b.Updated
	}
	return a.Name < b.Name
}
