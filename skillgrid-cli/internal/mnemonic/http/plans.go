package http

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// statusRE matches the `> **STATUS:** <value>` blockquote line used as the
// de-facto frontmatter in SDD briefing.md / tasks.md files. Same shape as the
// docs package's parseMeta regex (kept local so the plans bridge has no
// cross-package dependency on the unexported parseMeta).
var statusRE = regexp.MustCompile(`(?im)^>\s*\*{2}STATUS:\*{2}\s*` + "`?([a-z][a-z0-9-]+)`?")

// checkboxRE matches a markdown task checkbox line.
var checkboxRE = regexp.MustCompile(`(?m)^\s*[-*]\s+\[( |x|X)\]`)

// parseStatus extracts the STATUS value from an SDD markdown file.
func parseStatus(md string) string {
	m := statusRE.FindStringSubmatch(md)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// checklistProgress counts `- [ ]`/`- [x]` lines and returns done/total (and a
// 0..1 ratio). Files with no checkboxes yield 0/0.
func checklistProgress(md string) (done, total int) {
	for _, m := range checkboxRE.FindAllStringSubmatch(md, -1) {
		total++
		if m[1] == "x" || m[1] == "X" {
			done++
		}
	}
	return done, total
}

// sddRoot returns the docs sandbox root. It reads SKILLGRID_DOCS_CWD fresh each
// call (rather than the package-level docsCwd var, which is fixed at init) so
// tests can override the root via t.Setenv.
func sddRoot() string {
	if v := strings.TrimSpace(os.Getenv("SKILLGRID_DOCS_CWD")); v != "" {
		return v
	}
	return "."
}

func sddSpecsDir() string { return filepath.Join(sddRoot(), ".skillgrid", "specs") }

func sddLedgerDir() string { return filepath.Join(sddRoot(), ".skillgrid", "sdd") }

// planSummary is one entry in the /plans list.
type planSummary struct {
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
	TasksDone int    `json:"tasksDone"`
	TasksTotal int   `json:"tasksTotal"`
	HasLedger bool   `json:"hasLedger"`
}

// planDetail extends planSummary with the file inventory, spec markdown, and
// the linked SDD ledger steps.
type planDetail struct {
	planSummary
	Files  []planFile  `json:"files"`
	Steps  []planStep  `json:"steps"`
	Briefing string    `json:"briefing"`
	Tasks    string    `json:"tasks"`
}

type planFile struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type planStep struct {
	Raw      string `json:"raw"`
	Status   string `json:"status"`
}

// handlePlans serves GET /plans — aggregates .skillgrid/specs/* into plan cards
// (name, status, checklist progress, whether an SDD ledger exists).
func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	specsDir := sddSpecsDir()
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]any{"plans": []planSummary{}})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	plans := []planSummary{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		dir := filepath.Join(specsDir, name)
		briefing := readMDFile(filepath.Join(dir, "briefing.md"))
		tasks := readMDFile(filepath.Join(dir, "tasks.md"))
		status := parseStatus(briefing)
		if status == "" {
			status = parseStatus(tasks)
		}
		done, total := checklistProgress(tasks)
		progress := 0.0
		if total > 0 {
			progress = float64(done) / float64(total)
		}
		_, lerr := os.Stat(filepath.Join(sddLedgerDir(), name, "progress.md"))
		hasLedger := lerr == nil
		plans = append(plans, planSummary{
			Name: name, Status: status, Progress: progress,
			TasksDone: done, TasksTotal: total, HasLedger: hasLedger,
		})
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].Name > plans[j].Name }) // newest date first
	writeJSON(w, http.StatusOK, map[string]any{"plans": plans})
}

// handlePlanDetail serves GET /plans/{id} — detail for one spec: status,
// progress, file inventory, spec markdown, and the linked SDD ledger steps
// (parsed from .skillgrid/sdd/<id>/progress.md `## Steps` lines). Unknown id →
// 404.
func (s *Server) handlePlanDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "plan id required")
		return
	}
	dir := filepath.Join(sddSpecsDir(), id)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		writeError(w, http.StatusNotFound, "unknown plan: "+id)
		return
	}

	// File inventory (top-level files in the spec dir).
	files := []planFile{}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			size := 0
			if err == nil {
				size = int(info.Size())
			}
			files = append(files, planFile{Name: e.Name(), Size: size})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	briefing := readMDFile(filepath.Join(dir, "briefing.md"))
	tasks := readMDFile(filepath.Join(dir, "tasks.md"))
	status := parseStatus(briefing)
	if status == "" {
		status = parseStatus(tasks)
	}
	done, total := checklistProgress(tasks)
	progress := 0.0
	if total > 0 {
		progress = float64(done) / float64(total)
	}

	steps := parseLedgerSteps(filepath.Join(sddLedgerDir(), id, "progress.md"))

	writeJSON(w, http.StatusOK, planDetail{
		planSummary: planSummary{Name: id, Status: status, Progress: progress, TasksDone: done, TasksTotal: total, HasLedger: len(steps) > 0},
		Files:       files, Steps: steps, Briefing: briefing, Tasks: tasks,
	})
}

// parseLedgerSteps extracts the `## Steps` checklist lines from a progress.md
// SDD ledger and classifies each as COMPLETE / PENDING / other.
func parseLedgerSteps(progressPath string) []planStep {
	data, err := os.ReadFile(progressPath)
	if err != nil {
		return nil
	}
	var steps []planStep
	inSteps := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inSteps = strings.EqualFold(trimmed, "## Steps")
			continue
		}
		if inSteps && (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			status := "PENDING"
			upper := strings.ToUpper(trimmed)
			if strings.Contains(upper, "COMPLETE") || strings.Contains(upper, "DONE") {
				status = "COMPLETE"
			}
			steps = append(steps, planStep{Raw: trimmed, Status: status})
		}
	}
	return steps
}

// handleSpecs serves GET /specs — a flat list of spec markdown files under
// .skillgrid/specs (relative paths), so the UI can build a file tree.
func (s *Server) handleSpecs(w http.ResponseWriter, r *http.Request) {
	root := sddSpecsDir()
	files := []string{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

// handleSpecContent serves GET /specs/{path...} — the markdown content of one
// spec file (sandboxed to .skillgrid/specs). Unknown path → 404.
func (s *Server) handleSpecContent(w http.ResponseWriter, r *http.Request) {
	rel := r.PathValue("path")
	if rel == "" {
		writeError(w, http.StatusBadRequest, "spec path required")
		return
	}
	root := sddSpecsDir()
	full := filepath.Join(root, filepath.FromSlash(rel))
	// Sandbox: the resolved path must stay under root (no .. escape).
	relCheck, err := filepath.Rel(root, full)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		writeError(w, http.StatusNotFound, "unknown spec: "+rel)
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "unknown spec: "+rel)
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": filepath.ToSlash(rel), "content": string(data)})
}

// readMDFile reads a file's contents, returning "" on any error (missing file
// is a normal case for partial spec dirs).
func readMDFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
