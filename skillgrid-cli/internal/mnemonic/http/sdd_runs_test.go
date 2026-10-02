package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestTeamRuns_JoinsLedgerRowToSession(t *testing.T) {
	repo := t.TempDir()
	wave := filepath.Join(repo, ".skillgrid", "sdd", "2026-09-24-mem")
	if err := os.MkdirAll(wave, 0o755); err != nil {
		t.Fatal(err)
	}
	ledger := "# Parallel ledger\n\n| agent | task | status | Owns |\n|---|---|---|---|\n" +
		"| decay-config | TASK-030.05 TICKET-03 | dispatched | config/load.go |\n"
	if err := os.WriteFile(filepath.Join(wave, "parallel-ledger.md"), []byte(ledger), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SKILLGRID_DOCS_CWD", repo)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)

	projID, err := project.Resolve(repo)
	if err != nil && projID == "" {
		t.Fatalf("resolve: %v", err)
	}
	st, err := store.Open(dataDir, projID)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, title, started_at, status, agent)
		VALUES ('sess-decay', ?, ?, 'TICKET-03 mnemonic.decay config', '2026-10-02T09:14:21Z', 'active', 'cursor')`,
		projID, repo); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	st.Close()

	h := NewServer(service.New(dataDir)).Handler()
	rr := doGet(t, h, "/sdd/runs")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Runs []struct {
			Members []struct {
				Agent   string `json:"agent"`
				Session *struct {
					ID      string `json:"id"`
					Project string `json:"project"`
					Agent   string `json:"agent"`
					Live    string `json:"live"`
				} `json:"session"`
			} `json:"members"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Runs) != 1 || len(body.Runs[0].Members) != 1 {
		t.Fatalf("unexpected runs: %s", rr.Body.String())
	}
	s := body.Runs[0].Members[0].Session
	if s == nil {
		t.Fatalf("member has no session: %s", rr.Body.String())
	}
	if s.ID != "sess-decay" || s.Agent != "cursor" || s.Project != projID || s.Live != "idle" {
		t.Fatalf("session = %+v", s)
	}
}
