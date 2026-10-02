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
