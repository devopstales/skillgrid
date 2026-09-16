package main

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedMemCLIPeriod creates a project store with a session, a saved observation,
// and a distilled L2/L3 layer set (so `mem layers` has something to inspect).
// It returns the project id, the observation id, and the session id.
func seedMemCLIPeriod(t *testing.T, dataDir, project string) (int64, string) {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	mem := memory.New(st, project)
	ctx := context.Background()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status, title, summary)
		VALUES ('sess-memcli', ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', 'mem cli session', ?)`,
		project, "## Goal\nmem cli parity\n\n## Key Learnings:\n- The CLI must expose parity with the memory tools for layers and governance"); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	id, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "sess-memcli",
		Type:      "decision",
		Title:     "mem cli parity probe",
		Content:   "CLI parity body for mem governance/share/layers",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	// Distill the session so observation_layers + personas exist (mem layers).
	if _, err := layer.Distill(ctx, mem, "sess-memcli", layer.DistillOptions{}); err != nil {
		t.Fatalf("distill: %v", err)
	}
	return id, "sess-memcli"
}

func runMemCLI(t *testing.T, dataDir string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"run", ".", "mem"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MNEMONIC_DATA_DIR="+dataDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mem %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// TestMemCLIParity is 03.8 [AFK] — the CLI exposes parity with the memory
// tools: `mem layers`, `mem governance`, and `mem share` each route through the
// same seams as their MCP counterparts.
// Scenario: cli-parity-for-layer-governance-share.
func TestMemCLIParity(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-parity"
	obsID, sessID := seedMemCLIPeriod(t, dataDir, project)

	// mem layers <session_id>: returns the L0→L1→L2→L3 chain.
	out := runMemCLI(t, dataDir, "layers", sessID, "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "l0_session") || !strings.Contains(out, sessID) {
		t.Fatalf("mem layers missing the L0 session chain: %s", out)
	}
	// The distilled L2/L3 layers must be present (provenance-linked).
	if !strings.Contains(out, "L2") && !strings.Contains(out, "L3") {
		t.Fatalf("mem layers missing distilled L2/L3 layers: %s", out)
	}

	// mem governance <id>: returns the governed-asset view.
	out = runMemCLI(t, dataDir, "governance", strconv.FormatInt(obsID, 10), "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "visibility") || !strings.Contains(out, "owner") {
		t.Fatalf("mem governance missing asset fields: %s", out)
	}

	// mem share <id> --target-visibility team: widens visibility to team.
	out = runMemCLI(t, dataDir, "share", strconv.FormatInt(obsID, 10),
		"--target-visibility", "team", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "team") {
		t.Fatalf("mem share did not report the team visibility: %s", out)
	}

	// Verify the share landed (governance now shows team).
	out = runMemCLI(t, dataDir, "governance", strconv.FormatInt(obsID, 10), "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"team"`) {
		t.Fatalf("after mem share, governance must show visibility team: %s", out)
	}
}

// runMemCLIExpectError runs the mem CLI and returns (output, non-nil error);
// it does NOT t.Fatal on a non-zero exit (for negative-arg assertions).
func runMemCLIExpectError(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	cmdArgs := append([]string{"run", ".", "mem"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MNEMONIC_DATA_DIR="+dataDir)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestMemSearchModeFlag is 02.3 [AFK] — the CLI `mem search --mode` accepts
// trigram/prefix/phrase/all, routes trigram/prefix through the FTS match-mode
// read path (BudgetedRetrievalAsRootFTS), and rejects invalid modes with a
// clear error (exit 2, no store access).
func TestMemSearchModeFlag(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-mode-flag"
	seedMemCLIPeriod(t, dataDir, project)

	// --mode trigram: the FTS fact leg runs in trigram mode (no error,
	// budgeted result shape).
	out := runMemCLI(t, dataDir, "search", "mem", "--mode", "trigram", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"observations"`) {
		t.Fatalf("mem search --mode trigram should return the budgeted search shape, got: %s", out)
	}

	// --mode prefix: same read path, prefix FTS expansion.
	out = runMemCLI(t, dataDir, "search", "mem", "--mode", "prefix", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"observations"`) {
		t.Fatalf("mem search --mode prefix should return the budgeted search shape, got: %s", out)
	}

	// --mode phrase: the default (phrase OR) alias.
	out = runMemCLI(t, dataDir, "search", "mem", "--mode", "phrase", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"observations"`) {
		t.Fatalf("mem search --mode phrase should return the budgeted search shape, got: %s", out)
	}

	// --mode all: AND-joined phrases.
	out = runMemCLI(t, dataDir, "search", "mem", "--mode", "all", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"observations"`) {
		t.Fatalf("mem search --mode all should return the budgeted search shape, got: %s", out)
	}

	// Invalid mode: rejected clearly with a non-zero exit, before any store
	// access (the flag parsing layer validates, so the error is ours).
	out, err := runMemCLIExpectError(t, dataDir, "search", "mem", "--mode", "bogus", "--project", project, "--dir", dataDir)
	if err == nil {
		t.Fatalf("mem search --mode bogus should fail, got: %s", out)
	}
	if !strings.Contains(out, "invalid match mode") {
		t.Fatalf("invalid --mode should be rejected clearly, got: %s", out)
	}
}

// TestMemCLIBudgetedContext is the finding-03.3 proof: the CLI `mem context`
// honors its --char budget flag (previously ignored). A session with a long
// summary is char-truncated with an explicit "N chars omitted" marker when a
// small --char cap is passed.
func TestMemCLIBudgetedContext(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-budgeted-ctx"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	longSummary := "## Goal\nmem context budget probe\n\n## Key Learnings:\n- " + strings.Repeat("learn", 300)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status, title, summary)
		VALUES ('sess-ctx', ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', 'ctx budget session', ?)`,
		project, longSummary); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	st.Close()

	// A small --char cap must truncate the long summary (explicit marker).
	out := runMemCLI(t, dataDir, "context", "--project", project, "--char", "40", "--dir", dataDir)
	if !strings.Contains(out, "chars omitted") {
		t.Fatalf("mem context with --char 40 must char-truncate the summary (explicit 'chars omitted'), got: %s", out)
	}
	if !strings.Contains(out, `"truncated": true`) {
		t.Fatalf("mem context with a truncating --char must report truncated:true, got: %s", out)
	}
}

// TestMemCLIBadArgs proves the CLI rejects bad mem args clearly (parity with
// the MCP bad-args behavior).
func TestMemCLIBadArgs(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-badargs"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()

	// mem layers with no target → clear error (exit 2).
	cmd := exec.Command("go", "run", ".", "mem", "layers", "--project", project, "--dir", dataDir)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MNEMONIC_DATA_DIR="+dataDir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("mem layers with no target should fail, got: %s", out)
	}
	if !strings.Contains(string(out), "requires") && !strings.Contains(string(out), "session_id") {
		t.Fatalf("bad mem layers args should be rejected clearly, got: %s", out)
	}

	// mem share with an invalid visibility → clear error.
	id := "1"
	cmd = exec.Command("go", "run", ".", "mem", "share", id, "--target-visibility", "bogus", "--project", project, "--dir", dataDir)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MNEMONIC_DATA_DIR="+dataDir)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("mem share with bad visibility should fail, got: %s", out)
	}
}

// TestMemGraphRiskCLI is 23.4 [AFK]: `mem graph --risk` lists high-risk hub
// files sorted by risk_score, filtered by a configurable --threshold.
func TestMemGraphRiskCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-risk"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	mem := memory.New(st, project)
	ctx := context.Background()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	// 9 files: hub.go imported by 5 (hub_score 5/9 ≈ 0.556 > 0.5),
	// leaf.go + leaf2.go + leaf3.go non-hub.
	seedRiskCLIFile(t, st, "hub.go")
	for i := 0; i < 5; i++ {
		seedRiskCLIImporter(t, st, "rk"+string(rune('a'+i))+".go")
	}
	seedRiskCLIFile(t, st, "leaf.go")
	seedRiskCLIFile(t, st, "leaf2.go")
	seedRiskCLIFile(t, st, "leaf3.go")
	if _, err := mem.IdentifyHubFiles(ctx); err != nil {
		t.Fatalf("IdentifyHubFiles: %v", err)
	}
	// Hub observation (risk ≈ 0.556, above default threshold 0.5).
	if _, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "s1", Type: "decision", Title: "hub obs",
		Content: "about hub.go", Source: "hub.go",
	}); err != nil {
		t.Fatalf("save hub obs: %v", err)
	}
	// Leaf observation (risk 0, below threshold).
	if _, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "s1", Type: "decision", Title: "leaf obs",
		Content: "about leaf.go", Source: "leaf.go",
	}); err != nil {
		t.Fatalf("save leaf obs: %v", err)
	}
	if err := mem.RecomputeRiskScores(ctx); err != nil {
		t.Fatalf("RecomputeRiskScores: %v", err)
	}

	// Default threshold (0.5): only the hub observation is shown.
	out := runMemCLI(t, dataDir, "graph", "--risk", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "hub obs") {
		t.Fatalf("mem graph --risk missing hub observation: %s", out)
	}
	if strings.Contains(out, "leaf obs") {
		t.Fatalf("mem graph --risk should not show the leaf observation (risk 0 < 0.5): %s", out)
	}
	if !strings.Contains(out, `"threshold"`) {
		t.Fatalf("mem graph --risk missing threshold field: %s", out)
	}

	// --threshold 0: both observations are shown.
	out = runMemCLI(t, dataDir, "graph", "--risk", "--threshold", "0", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, "hub obs") || !strings.Contains(out, "leaf obs") {
		t.Fatalf("mem graph --risk --threshold 0 should show both: %s", out)
	}

	// --threshold 1: nothing is shown (all risk scores < 1).
	out = runMemCLI(t, dataDir, "graph", "--risk", "--threshold", "1", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"count": 0`) {
		t.Fatalf("mem graph --risk --threshold 1 should show zero entries: %s", out)
	}
}

// seedRiskCLIFile plants a file + package symbol in the CLI test store.
func seedRiskCLIFile(t *testing.T, st *store.Store, path string) int64 {
	t.Helper()
	var fileID int64
	if err := st.DB.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?, 1, 100, ?, '2026-01-01T00:00:00Z')
		RETURNING id`, path, "h-"+path).Scan(&fileID); err != nil {
		t.Fatalf("insert file %s: %v", path, err)
	}
	var symID int64
	uid := "uid-cli-" + path
	if err := st.DB.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, ?, 'package', 1, 1, ?, ?)
		RETURNING id`, fileID, path, "ch-cli-"+path, uid).Scan(&symID); err != nil {
		t.Fatalf("insert symbol %s: %v", path, err)
	}
	return symID
}

// seedRiskCLIImporter plants an importing file + symbol + imports edge.
// The hub symbol id is looked up as the symbol of the file named "hub.go".
func seedRiskCLIImporter(t *testing.T, st *store.Store, path string) {
	t.Helper()
	var hubSymID int64
	if err := st.DB.QueryRow(`
		SELECT s.id FROM symbols s
		JOIN files f ON f.id = s.file_id
		WHERE f.path = 'hub.go' LIMIT 1`).Scan(&hubSymID); err != nil {
		t.Fatalf("lookup hub symbol: %v", err)
	}
	var fileID int64
	if err := st.DB.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?, 1, 100, ?, '2026-01-01T00:00:00Z')
		RETURNING id`, path, "h-"+path).Scan(&fileID); err != nil {
		t.Fatalf("insert importer %s: %v", path, err)
	}
	var symID int64
	uid := "uid-cliimp-" + path
	if err := st.DB.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, ?, 'package', 1, 1, ?, ?)
		RETURNING id`, fileID, path, "ch-cliimp-"+path, uid).Scan(&symID); err != nil {
		t.Fatalf("insert importer symbol %s: %v", path, err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from)
		VALUES ('imports', ?, ?, ?, ?, ?, 'EXTRACTED', 1, 0)`,
		symID, fileID, hubSymID, path, path); err != nil {
		t.Fatalf("insert imports edge %s: %v", path, err)
	}
}
