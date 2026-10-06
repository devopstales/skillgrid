package http

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"net/http"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// newPhase6PlansServer builds a server whose docsCwd points at a temp repo with
// a fixture .skillgrid/specs/<slug>/ (briefing + tasks) and
// .skillgrid/sdd/<slug>/ (progress.md ledger), so /plans + /specs can be tested.
func newPhase6PlansServer(t *testing.T) http.Handler {
	t.Helper()
	repo := t.TempDir()

	spec := filepath.Join(repo, ".skillgrid", "specs", "2026-01-01-sample-change")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatalf("mkdir spec: %v", err)
	}
	// briefing.md — the `> **STATUS:**` line is the machine-parseable state.
	_ = os.WriteFile(filepath.Join(spec, "briefing.md"), []byte(
		"# Change: sample-change\n\n"+
			"> **STATUS:** `planning` (2026-01-01)\n\n"+
			"**Goal:** A sample change for the plans endpoint test.\n\n"+
			"**Depends on:** none\n"), 0o644)
	// tasks.md — a phase with a 2-item checklist (1 done) → 50% progress.
	_ = os.WriteFile(filepath.Join(spec, "tasks.md"), []byte(
		"# Tasks: sample-change\n\n"+
			"> **STATUS:** `planning` (2026-01-01)\n\n"+
			"### Tasks\n\n"+
			"- [x] 1.1 first task\n"+
			"- [ ] 1.2 second task\n"), 0o644)
	// findings.md — optional, should be surfaced in the file list.
	_ = os.WriteFile(filepath.Join(spec, "findings.md"), []byte("# Findings\nresearch\n"), 0o644)

	// A second spec, archived/done, with 2/2 tasks → 100%.
	spec2 := filepath.Join(repo, ".skillgrid", "specs", "2025-12-31-done-change")
	if err := os.MkdirAll(spec2, 0o755); err != nil {
		t.Fatalf("mkdir spec2: %v", err)
	}
	_ = os.WriteFile(filepath.Join(spec2, "briefing.md"), []byte("# Change: done-change\n\n> **STATUS:** `done`\n"), 0o644)
	_ = os.WriteFile(filepath.Join(spec2, "tasks.md"), []byte("# Tasks: done-change\n\n- [x] 1.1 a\n- [x] 1.2 b\n"), 0o644)

	// SDD ledger for the sample change: a progress.md with step statuses.
	sdd := filepath.Join(repo, ".skillgrid", "sdd", "2026-01-01-sample-change")
	if err := os.MkdirAll(sdd, 0o755); err != nil {
		t.Fatalf("mkdir sdd: %v", err)
	}
	_ = os.WriteFile(filepath.Join(sdd, "progress.md"), []byte(`# SDD ledger — plan: sample-change

## Steps
- Task 01 (step-one): COMPLETE (aaa1111..bbb2222). Verdict PASS.
- Task 02 (step-two): PENDING.
`), 0o644)
	_ = os.WriteFile(filepath.Join(sdd, "task-01-step-one-brief.md"), []byte("## 01-step-one\n\n### Goal\ndo the thing\n"), 0o644)

	t.Setenv("SKILLGRID_DOCS_CWD", repo)
	// dataDir must exist for service.New; use a fresh temp dir (plans/specs read
	// the filesystem, not the DB).
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	return NewServer(svc).Handler()
}

func TestPhase6_Plans(t *testing.T) {
	h := newPhase6PlansServer(t)

	// --- /plans → both specs, with status + progress ---
	rr := doGet(t, h, "/plans")
	if rr.Code != http.StatusOK {
		t.Fatalf("/plans: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var pm map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &pm); err != nil {
		t.Fatalf("/plans unmarshal: %v", err)
	}
	plans, _ := pm["plans"].([]any)
	if len(plans) != 2 {
		t.Fatalf("/plans: got %d, want 2; body=%s", len(plans), rr.Body.String())
	}
	byName := map[string]map[string]any{}
	for _, p := range plans {
		m := p.(map[string]any)
		byName[m["name"].(string)] = m
	}
	sample, ok := byName["2026-01-01-sample-change"]
	if !ok {
		t.Fatalf("/plans: missing sample-change; got %+v", byName)
	}
	if sample["status"] != "planning" {
		t.Errorf("sample.status = %v, want 'planning'", sample["status"])
	}
	// progress: 1 of 2 tasks done → 0.5
	if pg, _ := sample["progress"].(float64); pg < 0.49 || pg > 0.51 {
		t.Errorf("sample.progress = %v, want ~0.5", sample["progress"])
	}
	done, ok := byName["2025-12-31-done-change"]
	if !ok {
		t.Fatalf("/plans: missing done-change")
	}
	if done["status"] != "done" {
		t.Errorf("done.status = %v, want 'done'", done["status"])
	}
	if pg, _ := done["progress"].(float64); pg < 0.99 {
		t.Errorf("done.progress = %v, want 1.0", pg)
	}

	// --- /plans/{id} → detail (steps, files, progress) ---
	rr = doGet(t, h, "/plans/2026-01-01-sample-change")
	if rr.Code != http.StatusOK {
		t.Fatalf("/plans/{id}: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var dm map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &dm); err != nil {
		t.Fatalf("/plans/{id} unmarshal: %v", err)
	}
	if dm["name"] != "2026-01-01-sample-change" {
		t.Errorf("detail.name = %v", dm["name"])
	}
	// files should include briefing.md + tasks.md + findings.md
	files, _ := dm["files"].([]any)
	fileNames := map[string]bool{}
	for _, f := range files {
		fm := f.(map[string]any)
		fileNames[fm["name"].(string)] = true
	}
	for _, want := range []string{"briefing.md", "tasks.md", "findings.md"} {
		if !fileNames[want] {
			t.Errorf("detail.files missing %s; got %+v", want, fileNames)
		}
	}
	// linked SDD ledger steps (from progress.md)
	steps, _ := dm["steps"].([]any)
	if len(steps) != 2 {
		t.Errorf("detail.steps len = %d, want 2 (progress.md ledger)", len(steps))
	}

	// --- unknown plan → 404 ---
	rr = doGet(t, h, "/plans/nope")
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown plan: status = %d, want 404", rr.Code)
	}
}

func TestPlanDetail_ReadsArchivedChange(t *testing.T) {
	repo := t.TempDir()
	dir := filepath.Join(repo, ".skillgrid", "archive", "2026-09-04-hermes-memory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	body := "# Change: 004-hermes-memory\n\n> **STATUS:** `revised`\n\nHermes Fact Memory\n"
	if err := os.WriteFile(filepath.Join(dir, "briefing.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write briefing: %v", err)
	}
	t.Setenv("SKILLGRID_DOCS_CWD", repo)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	h := NewServer(service.New(dataDir)).Handler()

	rr := doGet(t, h, "/plans/2026-09-04-hermes-memory")
	if rr.Code != http.StatusOK {
		t.Fatalf("archived plan: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var detail map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	briefing, _ := detail["briefing"].(string)
	if !strings.Contains(briefing, "Hermes Fact Memory") {
		t.Fatalf("briefing = %q", briefing)
	}
}

func TestPhase6_Specs(t *testing.T) {
	h := newPhase6PlansServer(t)

	// --- /specs → list of spec files ---
	rr := doGet(t, h, "/specs")
	if rr.Code != http.StatusOK {
		t.Fatalf("/specs: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var sm map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &sm); err != nil {
		t.Fatalf("/specs unmarshal: %v", err)
	}
	files, _ := sm["files"].([]any)
	if len(files) < 4 { // at least the 4 md files across both spec dirs
		t.Errorf("/specs: got %d files, want >= 4; body=%s", len(files), rr.Body.String())
	}

	// --- /specs/{path} → markdown content ---
	rr = doGet(t, h, "/specs/2026-01-01-sample-change/briefing.md")
	if rr.Code != http.StatusOK {
		t.Fatalf("/specs/{path}: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var cm map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &cm); err != nil {
		t.Fatalf("/specs/{path} unmarshal: %v", err)
	}
	content, _ := cm["content"].(string)
	if content == "" || len(content) < 10 {
		t.Errorf("/specs/{path}.content too short: %q", content)
	}

	// --- unknown spec path → 404 ---
	rr = doGet(t, h, "/specs/nope/nope.md")
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown spec: status = %d, want 404", rr.Code)
	}
}
