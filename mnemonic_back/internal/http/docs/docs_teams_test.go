package docs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestListTeamRuns_ParsesLedgersAndSkipsEmptyDirs(t *testing.T) {
	root := t.TempDir()
	sdd := filepath.Join(root, ".skillgrid", "sdd")

	wave := filepath.Join(sdd, "2026-10-02-memory")
	if err := os.MkdirAll(wave, 0o755); err != nil {
		t.Fatal(err)
	}
	ledger := `# Parallel ledger — memory wave

| agent | task | status | Owns | changed files | test result |
|---|---|---|---|---|---|
| decay-config | TICKET-03 | dispatched | config/load.go | | |
| query-cache | TICKET-04 | done | store/query_cache.go | query_cache.go | pass |
`
	if err := os.WriteFile(filepath.Join(wave, "parallel-ledger.md"), []byte(ledger), 0o644); err != nil {
		t.Fatal(err)
	}

	seq := filepath.Join(sdd, "2026-10-01-serial")
	if err := os.MkdirAll(seq, 0o755); err != nil {
		t.Fatal(err)
	}
	progress := `# SDD ledger — plan: tasks.md

- Task 1: complete
- Task 1: Work unit evidence is not a status
- Task 2: Owns: internal/foo.go
- Task 2: in-progress
`
	if err := os.WriteFile(filepath.Join(seq, "progress.md"), []byte(progress), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(sdd, "blueprint"), 0o755); err != nil {
		t.Fatal(err)
	}

	runs, err := ListTeamRuns(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("runs = %d, want 2 (%+v)", len(runs), runs)
	}

	var waveRun, seqRun TeamRun
	for _, r := range runs {
		switch r.Name {
		case "2026-10-02-memory":
			waveRun = r
		case "2026-10-01-serial":
			seqRun = r
		default:
			t.Errorf("unexpected run %q", r.Name)
		}
	}
	if waveRun.InFlight != 1 || waveRun.Done != 1 {
		t.Fatalf("wave counts in_flight=%d done=%d", waveRun.InFlight, waveRun.Done)
	}
	if len(waveRun.Members) != 2 || waveRun.Members[0].Agent != "decay-config" || waveRun.Members[0].Status != "dispatched" {
		t.Fatalf("wave members = %+v", waveRun.Members)
	}
	if waveRun.Heading != "Parallel ledger — memory wave" {
		t.Errorf("heading = %q", waveRun.Heading)
	}
	if seqRun.Done != 1 || seqRun.InFlight != 1 {
		t.Fatalf("serial counts in_flight=%d done=%d members=%+v", seqRun.InFlight, seqRun.Done, seqRun.Members)
	}
	if len(seqRun.Members) != 2 {
		t.Fatalf("Owns line should not be a member, got %+v", seqRun.Members)
	}
}

func TestListTeamRuns_ParsesTicketLines(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".skillgrid", "sdd", "2026-10-02-webui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	progress := `# SDD ledger — plan: tasks.md

- TICKET-01 (TASK-032): complete — verified at d5e01985
- TICKET-04 (TASK-036): blocked — halted at precondition
`
	if err := os.WriteFile(filepath.Join(dir, "progress.md"), []byte(progress), 0o644); err != nil {
		t.Fatal(err)
	}

	runs, err := ListTeamRuns(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || len(runs[0].Members) != 2 {
		t.Fatalf("runs = %+v", runs)
	}
	if runs[0].Members[0].Task != "TICKET-01 (TASK-032)" || runs[0].Members[0].Status != "complete" {
		t.Fatalf("first = %+v", runs[0].Members[0])
	}
	if runs[0].Members[1].Task != "TICKET-04 (TASK-036)" || runs[0].Members[1].Status != "blocked" {
		t.Fatalf("second = %+v", runs[0].Members[1])
	}
	if runs[0].Done != 1 || runs[0].InFlight != 1 {
		t.Fatalf("counts in_flight=%d done=%d", runs[0].InFlight, runs[0].Done)
	}
}

func TestListTeamRuns_TicketLineWithoutTaskID(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".skillgrid", "sdd", "bare-ticket")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	progress := "- TICKET-9: complete\n- TICKET-8: verified at HEAD\n"
	if err := os.WriteFile(filepath.Join(dir, "progress.md"), []byte(progress), 0o644); err != nil {
		t.Fatal(err)
	}
	runs, err := ListTeamRuns(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || len(runs[0].Members) != 1 {
		t.Fatalf("members = %+v", runs)
	}
	if runs[0].Members[0].Task != "TICKET-9" || runs[0].Members[0].Status != "complete" || runs[0].Done != 1 {
		t.Fatalf("member = %+v done=%d", runs[0].Members[0], runs[0].Done)
	}
}

func TestListTeamRuns_NeedsBlocksUntilDependencyCompletes(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".skillgrid", "sdd", "coupled")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	progress := `# Coupled

- Task 1: in-progress
- Task 2: Needs: 1
- Task 2: dispatched
`
	if err := os.WriteFile(filepath.Join(dir, "progress.md"), []byte(progress), 0o644); err != nil {
		t.Fatal(err)
	}
	runs, err := ListTeamRuns(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || len(runs[0].Members) != 2 {
		t.Fatalf("runs = %+v", runs)
	}
	var task2 TeamMember
	for _, m := range runs[0].Members {
		if m.Task == "2" {
			task2 = m
		}
	}
	if task2.Status != "blocked" {
		t.Fatalf("task 2 = %+v, want blocked while task 1 is in progress", task2)
	}
}

func TestListTeamRuns_NeedsAllowsDispatchWhenDependencyIsDone(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".skillgrid", "sdd", "coupled-done")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	progress := "- Task 1: complete\n- Task 2: Needs: 1\n- Task 2: dispatched\n"
	if err := os.WriteFile(filepath.Join(dir, "progress.md"), []byte(progress), 0o644); err != nil {
		t.Fatal(err)
	}
	runs, err := ListTeamRuns(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	var task2 TeamMember
	for _, m := range runs[0].Members {
		if m.Task == "2" {
			task2 = m
		}
	}
	if task2.Status != "dispatched" || task2.Needs != "1" {
		t.Fatalf("task 2 = %+v, want dispatched with needs 1", task2)
	}
}

func TestListTeamRuns_ArchiveMarksMembersDone(t *testing.T) {
	root := t.TempDir()
	name := "2026-09-24-mnemonic-memory-improvements"
	dir := filepath.Join(root, ".skillgrid", "sdd", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ledger := `| agent | task | status | Owns |
|---|---|---|---|
| decay-config | TICKET-03 | dispatched | config/load.go |
`
	if err := os.WriteFile(filepath.Join(dir, "parallel-ledger.md"), []byte(ledger), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".skillgrid", "archive", name), 0o755); err != nil {
		t.Fatal(err)
	}
	runs, err := ListTeamRuns(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || len(runs[0].Members) != 1 {
		t.Fatalf("runs = %+v", runs)
	}
	if runs[0].Members[0].Status != "complete" || runs[0].InFlight != 0 || runs[0].Done != 1 {
		t.Fatalf("shipped run = %+v", runs[0])
	}
}

func TestListTeamRuns_OtherArchiveDoesNotCloseRun(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".skillgrid", "sdd", "still-open")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ledger := "| agent | task | status |\n|---|---|---|\n| scout | TICKET-01 | dispatched |\n"
	if err := os.WriteFile(filepath.Join(dir, "parallel-ledger.md"), []byte(ledger), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".skillgrid", "archive", "some-other-change"), 0o755); err != nil {
		t.Fatal(err)
	}
	runs, err := ListTeamRuns(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	if runs[0].Members[0].Status != "dispatched" || runs[0].InFlight != 1 || runs[0].Done != 0 {
		t.Fatalf("open run = %+v", runs[0])
	}
}

func TestTeamRunsHandler_MissingRootIsEmpty(t *testing.T) {
	h := NewTeamRuns(t.TempDir())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/sdd/runs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Source string    `json:"source"`
		Count  int       `json:"count"`
		Runs   []TeamRun `json:"runs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Source != ".skillgrid/sdd" || resp.Count != 0 || len(resp.Runs) != 0 {
		t.Fatalf("resp = %+v", resp)
	}
}
