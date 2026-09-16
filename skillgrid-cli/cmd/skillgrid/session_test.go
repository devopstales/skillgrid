package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// sessionCLIFixture opens a fresh project store (the SAME store file the MCP
// handlers will open via the injected service) and resolves the CWD project id
// the SAME way the CLI resolves it (project.Resolve on CWD). The CLI is run in
// its real package dir with CWD pinned to the module package and the store
// pinned via --project/--dir, so the store + row are shared with the in-process
// MCP handlers. Returns the data dir, the pinned project id, and the store.
func sessionCLIFixture(t *testing.T) (dataDir, proj string, st *store.Store) {
	t.Helper()
	dataDir = t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	pid, err := project.Resolve(wd)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	proj = pid
	st, err = store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return
}

// runSessionCLI runs the real CLI (`go run . session <args>`) in its real
// package dir with CWD pinned there and the store pinned via --project/--dir,
// so the store + row it writes are shared with the in-process MCP handlers.
func runSessionCLI(t *testing.T, dataDir, proj, cwd string, args ...string) (string, int) {
	t.Helper()
	cmdArgs := append([]string{"run", ".", "session"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"SKILLGRID_MNEMONIC_DATA_DIR="+dataDir,
		"MNEMONIC_PROJECT="+proj,
	)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		code = -1
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
	}
	return string(out), code
}

// newMCPReq builds a minimal CallToolRequest for an MCP handler call.
func newMCPReq(name string, args map[string]any) mcplib.CallToolRequest {
	req := mcplib.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	return req
}

// mcpText extracts the text content of a CallToolResult.
func mcpText(t *testing.T, res *mcplib.CallToolResult) string {
	t.Helper()
	var out string
	for _, c := range res.Content {
		if tc, ok := c.(mcplib.TextContent); ok {
			out += tc.Text
		}
	}
	return out
}

// injectMCPSvc pins the project (CWD resolution) and injects a service rooted
// at dataDir, so the in-process MCP handlers open the SAME store the CLI uses.
func injectMCPSvc(t *testing.T, dataDir, proj, cwd string) {
	t.Helper()
	t.Setenv("MNEMONIC_PROJECT", proj)
	mcp.SetService(service.New(dataDir))
	t.Cleanup(func() { mcp.SetService(nil) })
	oldWD, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
}

// TestSessionMirrorsMCP is 04.1 [RED] — the CLI session handoff/resume/status
// mirrors the MCP Session Relay on the SAME project store: the same cleave
// bundle + session_handoffs row, the same resume prompt, the same status.
//
// The handoff is produced through the REAL CLI (subprocess); the resume and
// status are produced through the MCP handlers (in-process) and the CLI
// respectively. All run against the same store (same --project/--dir), so the
// row + bundle + resume outcome must agree.
func TestSessionMirrorsMCP(t *testing.T) {
	dataDir, proj, st := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	progress := "CLI parity: implemented the session CLI"
	knowledge := "fail closed: write files before the row"
	nextPrompt := "Resume: run the verify phase for change 006"
	const handoffID = "cli-parity-handoff"

	// (1) CLI session handoff writes the cleave bundle + row.
	hoOut, code := runSessionCLI(t, dataDir, proj, cwd,
		"handoff", "--progress", progress,
		"--knowledge", knowledge,
		"--next-prompt", nextPrompt,
		"--handoff-id", handoffID,
		"--project", proj, "--dir", dataDir)
	if code != 0 {
		t.Fatalf("session handoff exited non-zero: %d\n%s", code, hoOut)
	}
	var ho struct {
		HandoffID string   `json:"handoff_id"`
		Paths     []string `json:"paths"`
	}
	if err := json.Unmarshal([]byte(hoOut), &ho); err != nil {
		t.Fatalf("unmarshal handoff: %v (out %s)", err, hoOut)
	}
	if ho.HandoffID != handoffID {
		t.Fatalf("handoff_id = %q, want %s", ho.HandoffID, handoffID)
	}
	if len(ho.Paths) != 3 {
		t.Fatalf("handoff must return 3 cleave paths, got %v", ho.Paths)
	}
	// The three .cleave/ files exist on disk (under the resolved root).
	cleaveDir := filepath.Join(ho.Paths[0], "..")
	for _, name := range []string{"PROGRESS.md", "KNOWLEDGE.md", "NEXT_PROMPT.md"} {
		if _, err := os.Stat(filepath.Join(cleaveDir, name)); err != nil {
			t.Fatalf("expected cleave file %s on disk: %v", name, err)
		}
	}
	// Clean up the bundle so it does not leak into other tests that assert on
	// the same CWD-relative .cleave/ directory.
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(cleaveDir, "..")) })

	// (2) The row landed in the SAME store the MCP handlers will use.
	var rows int
	if err := st.DB.QueryRow(
		"SELECT COUNT(*) FROM session_handoffs WHERE project=? AND handoff_id=?",
		proj, handoffID).Scan(&rows); err != nil {
		t.Fatalf("count handoff row: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 handoff row in the shared store, got %d", rows)
	}

	// (3) MCP session_resume on the SAME store reads back the CLI handoff.
	injectMCPSvc(t, dataDir, proj, cwd)
	resRes, err := mcpCallResume(t, handoffID)
	if err != nil {
		t.Fatalf("mcp session_resume: %v", err)
	}
	var resOut struct {
		Prompt    string `json:"prompt"`
		HandoffID string `json:"handoff_id"`
	}
	if err := json.Unmarshal([]byte(mcpText(t, resRes)), &resOut); err != nil {
		t.Fatalf("unmarshal mcp resume: %v (text %s)", err, mcpText(t, resRes))
	}
	if resOut.Prompt != nextPrompt {
		t.Errorf("mcp resume prompt = %q, want the CLI-written NEXT_PROMPT %q", resOut.Prompt, nextPrompt)
	}
	if resOut.HandoffID != handoffID {
		t.Errorf("mcp resume handoff_id = %q, want %s", resOut.HandoffID, handoffID)
	}

	// (4) CLI session resume on the SAME store reads back the same prompt.
	cliResOut, code := runSessionCLI(t, dataDir, proj, cwd,
		"resume", handoffID, "--project", proj, "--dir", dataDir)
	if code != 0 {
		t.Fatalf("session resume exited non-zero: %d\n%s", code, cliResOut)
	}
	var cliRes struct {
		Prompt    string `json:"prompt"`
		HandoffID string `json:"handoff_id"`
	}
	if err := json.Unmarshal([]byte(cliResOut), &cliRes); err != nil {
		t.Fatalf("unmarshal cli resume: %v (out %s)", err, cliResOut)
	}
	if cliRes.Prompt != resOut.Prompt {
		t.Errorf("cli resume prompt = %q, want the MCP resume prompt %q", cliRes.Prompt, resOut.Prompt)
	}
	if cliRes.HandoffID != resOut.HandoffID {
		t.Errorf("cli resume handoff_id = %q, want the MCP resume id %q", cliRes.HandoffID, resOut.HandoffID)
	}

	// (5) MCP session_status + CLI session status agree on the handoff count.
	stRes, err := mcpCallStatus(t, map[string]any{})
	if err != nil {
		t.Fatalf("mcp session_status: %v", err)
	}
	var stOut struct {
		HandoffCount int `json:"handoff_count"`
	}
	if err := json.Unmarshal([]byte(mcpText(t, stRes)), &stOut); err != nil {
		t.Fatalf("unmarshal mcp status: %v (text %s)", err, mcpText(t, stRes))
	}
	if stOut.HandoffCount != 1 {
		t.Errorf("mcp status handoff_count = %d, want 1", stOut.HandoffCount)
	}
	cliStOut, code := runSessionCLI(t, dataDir, proj, cwd,
		"status", "--project", proj, "--dir", dataDir)
	if code != 0 {
		t.Fatalf("session status exited non-zero: %d\n%s", code, cliStOut)
	}
	var cliSt struct {
		HandoffCount int `json:"handoff_count"`
	}
	if err := json.Unmarshal([]byte(cliStOut), &cliSt); err != nil {
		t.Fatalf("unmarshal cli status: %v (out %s)", err, cliStOut)
	}
	if cliSt.HandoffCount != stOut.HandoffCount {
		t.Errorf("cli status handoff_count = %d, want the MCP count %d", cliSt.HandoffCount, stOut.HandoffCount)
	}
}

// mcpCallResume calls the session_resume MCP handler in-process.
func mcpCallResume(t *testing.T, handoffID string) (*mcplib.CallToolResult, error) {
	t.Helper()
	res, err := mcp.HandleSessionResumeForTest(context.Background(), newMCPReq("session_resume", map[string]any{
		"handoff_id": handoffID,
	}))
	if err != nil {
		return nil, err
	}
	if res.IsError {
		return nil, &mcpErr{mcpText(t, res)}
	}
	return res, nil
}

// mcpCallStatus calls the session_status MCP handler in-process.
func mcpCallStatus(t *testing.T, args map[string]any) (*mcplib.CallToolResult, error) {
	t.Helper()
	res, err := mcp.HandleSessionStatusForTest(context.Background(), newMCPReq("session_status", args))
	if err != nil {
		return nil, err
	}
	if res.IsError {
		return nil, &mcpErr{mcpText(t, res)}
	}
	return res, nil
}

type mcpErr struct{ msg string }

func (e *mcpErr) Error() string { return "mcp tool error: " + e.msg }

// TestSessionResumeMissingID is 04.2 [AFK] — a session resume with a missing
// handoff id fails closed: non-zero exit, a clear stderr message, and no
// partial cleave bundle created.
func TestSessionResumeMissingID(t *testing.T) {
	dataDir, proj, _ := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	out, code := runSessionCLI(t, dataDir, proj, cwd,
		"resume", "--project", proj, "--dir", dataDir)
	if code == 0 {
		t.Fatalf("resume with missing id should exit non-zero, got output: %s", out)
	}
	if !strings.Contains(out, "handoff_id") {
		t.Fatalf("resume missing id should name the required handoff_id in stderr, got: %s", out)
	}
}

// TestSessionResumeUnknownID is 04.2 [AFK] — a session resume with an unknown
// handoff id fails closed: non-zero exit, a clear stderr message (unknown
// handoff id), and no partial cleave bundle.
func TestSessionResumeUnknownID(t *testing.T) {
	dataDir, proj, _ := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	out, code := runSessionCLI(t, dataDir, proj, cwd,
		"resume", "no-such-handoff", "--project", proj, "--dir", dataDir)
	if code == 0 {
		t.Fatalf("resume with an unknown id should exit non-zero, got output: %s", out)
	}
	if !strings.Contains(out, "unknown handoff") {
		t.Fatalf("resume unknown id should carry a clear unknown-handoff message, got: %s", out)
	}
}

// TestSessionHandoffMissingFlag is 04.2 [AFK] — a session handoff with a bad /
// missing required flag fails closed: non-zero exit, a clear stderr message.
// The relay's files-before-row invariant (WriteBundle removes any half-bundle)
// guarantees no partial cleave bundle is left behind.
func TestSessionHandoffMissingFlag(t *testing.T) {
	dataDir, proj, _ := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Missing the required --next-prompt.
	out, code := runSessionCLI(t, dataDir, proj, cwd,
		"handoff", "--progress", "did work", "--project", proj, "--dir", dataDir)
	if code == 0 {
		t.Fatalf("handoff with a missing required flag should exit non-zero, got: %s", out)
	}
	if !strings.Contains(out, "next-prompt") {
		t.Fatalf("handoff missing flag should name --next-prompt in stderr, got: %s", out)
	}

	// Unknown flag is also rejected (bad flag), non-zero.
	out, code = runSessionCLI(t, dataDir, proj, cwd,
		"handoff", "--progress", "did work", "--next-prompt", "resume",
		"--bogus-flag", "--project", proj, "--dir", dataDir)
	if code == 0 {
		t.Fatalf("handoff with an unknown flag should exit non-zero, got: %s", out)
	}
}

// TestSessionNoStore is 04.3 [AFK] — with no usable project store (the data
// directory is a file, so the store cannot be opened), session handoff fails
// closed: non-zero exit, a clear stderr error, and no partial cleave bundle.
func TestSessionNoStore(t *testing.T) {
	blockDir := t.TempDir()
	blockFile := filepath.Join(blockDir, "block")
	if err := os.WriteFile(blockFile, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write block file: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	proj, err := project.Resolve(cwd)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}

	out, code := runSessionCLI(t, blockFile, proj, cwd,
		"handoff", "--progress", "did work", "--next-prompt", "resume",
		"--project", proj, "--dir", blockFile)
	if code == 0 {
		t.Fatalf("session handoff with no usable store should exit non-zero, got: %s", out)
	}
	if !strings.Contains(out, "error") {
		t.Fatalf("session handoff with no store should carry a clear error on stderr, got: %s", out)
	}
	// No partial cleave bundle was created (the store never opened, so the
	// relay never ran).
	if entries, err := os.ReadDir(filepath.Join(cwd, ".skillgrid", ".cleave")); err == nil && len(entries) > 0 {
		t.Fatalf("handoff with no usable store must not create a partial .cleave/ bundle: %v", entries)
	}
}

// TestSessionHandoffWatchdogPastThreshold wires the step-05 watchdog to the CLI:
// with SKILLGRID_HANDOFF_WATCHDOG enabled and --usage at/past the threshold,
// `session handoff --watchdog` runs the SAME Handoff path (a handoff row + 3
// cleave files) and reports handed_off:true.
func TestSessionHandoffWatchdogPastThreshold(t *testing.T) {
	dataDir, proj, st := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SKILLGRID_HANDOFF_WATCHDOG", "1")
	t.Setenv("SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD", "0.8")

	// The watchdog is an AUTO-trigger: it generates its own handoff id (the
	// operator's --handoff-id is not used on the watchdog path), so assert the
	// returned id + row by that id, not a fixed one.
	out, code := runSessionCLI(t, dataDir, proj, cwd,
		"handoff", "--watchdog", "--usage", "0.95",
		"--progress", "watchdog: context full", "--next-prompt", "resume after overflow",
		"--project", proj, "--dir", dataDir)
	if code != 0 {
		t.Fatalf("watchdog handoff (past threshold) exited non-zero: %d\n%s", code, out)
	}
	var wd struct {
		HandedOff bool     `json:"handed_off"`
		HandoffID string   `json:"handoff_id"`
		Paths     []string `json:"paths"`
	}
	if err := json.Unmarshal([]byte(out), &wd); err != nil {
		t.Fatalf("unmarshal watchdog handoff: %v (out %s)", err, out)
	}
	if !wd.HandedOff {
		t.Fatalf("watchdog past threshold must hand off, got: %s", out)
	}
	if wd.HandoffID == "" {
		t.Fatalf("watchdog handoff must return a generated handoff_id, got: %s", out)
	}
	if len(wd.Paths) != 3 {
		t.Fatalf("watchdog handoff must return 3 cleave paths, got %v", wd.Paths)
	}
	// The SAME Handoff path ran: a row (by the generated id) + 3 cleave files exist.
	var rows int
	if err := st.DB.QueryRow(
		"SELECT COUNT(*) FROM session_handoffs WHERE project=? AND handoff_id=?",
		proj, wd.HandoffID).Scan(&rows); err != nil {
		t.Fatalf("count handoff row: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 handoff row from the watchdog, got %d", rows)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(wd.Paths[0], "..", "..")) })
}

// TestSessionHandoffWatchdogNoOp covers the off-by-default and below-threshold
// cases: env unset (default) OR enabled-but-below-threshold -> no handoff row,
// no cleave files, and the CLI reports handed_off:false (never auto-hands-off).
func TestSessionHandoffWatchdogNoOp(t *testing.T) {
	dataDir, proj, st := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cleanupCleave := func() { _ = os.RemoveAll(filepath.Join(cwd, ".skillgrid", ".cleave")) }

	// (a) Off by default: env unset -> no-op even at usage 0.999.
	{
		out, code := runSessionCLI(t, dataDir, proj, cwd,
			"handoff", "--watchdog", "--usage", "0.999",
			"--progress", "p", "--next-prompt", "n", "--handoff-id", "wd-default",
			"--project", proj, "--dir", dataDir)
		if code != 0 {
			t.Fatalf("watchdog (env unset) should no-op with exit 0, got %d\n%s", code, out)
		}
		var wd struct {
			HandedOff bool `json:"handed_off"`
		}
		if err := json.Unmarshal([]byte(out), &wd); err != nil {
			t.Fatalf("unmarshal: %v (out %s)", err, out)
		}
		if wd.HandedOff {
			t.Fatalf("watchdog must be a no-op when SKILLGRID_HANDOFF_WATCHDOG is unset, got: %s", out)
		}
		var rows int
		if err := st.DB.QueryRow("SELECT COUNT(*) FROM session_handoffs WHERE project=? AND handoff_id=?", proj, "wd-default").Scan(&rows); err != nil {
			t.Fatalf("count: %v", err)
		}
		if rows != 0 {
			t.Fatalf("watchdog no-op must not write a handoff row, got %d", rows)
		}
	}

	// (b) Enabled but below threshold -> no-op.
	t.Setenv("SKILLGRID_HANDOFF_WATCHDOG", "1")
	t.Setenv("SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD", "0.8")
	{
		out, code := runSessionCLI(t, dataDir, proj, cwd,
			"handoff", "--watchdog", "--usage", "0.5",
			"--progress", "p", "--next-prompt", "n", "--handoff-id", "wd-below",
			"--project", proj, "--dir", dataDir)
		if code != 0 {
			t.Fatalf("watchdog (below threshold) should no-op with exit 0, got %d\n%s", code, out)
		}
		var wd struct {
			HandedOff bool `json:"handed_off"`
		}
		if err := json.Unmarshal([]byte(out), &wd); err != nil {
			t.Fatalf("unmarshal: %v (out %s)", err, out)
		}
		if wd.HandedOff {
			t.Fatalf("watchdog below threshold must not hand off, got: %s", out)
		}
		var rows int
		if err := st.DB.QueryRow("SELECT COUNT(*) FROM session_handoffs WHERE project=? AND handoff_id=?", proj, "wd-below").Scan(&rows); err != nil {
			t.Fatalf("count: %v", err)
		}
		if rows != 0 {
			t.Fatalf("watchdog below threshold must not write a handoff row, got %d", rows)
		}
	}
	t.Cleanup(cleanupCleave)
}

// TestSessionHandoffWatchdogInvalidConfig fails closed: an invalid threshold
// (non-numeric) -> non-zero exit + a clear config error + NO handoff row. A bad
// --usage fraction (out of [0,1]) also fails closed.
func TestSessionHandoffWatchdogInvalidConfig(t *testing.T) {
	dataDir, proj, st := sessionCLIFixture(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// (a) Invalid (non-numeric) threshold -> fail closed, no handoff.
	t.Setenv("SKILLGRID_HANDOFF_WATCHDOG", "1")
	t.Setenv("SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD", "not-a-number")
	out, code := runSessionCLI(t, dataDir, proj, cwd,
		"handoff", "--watchdog", "--usage", "0.95",
		"--progress", "p", "--next-prompt", "n", "--handoff-id", "wd-invalid",
		"--project", proj, "--dir", dataDir)
	if code == 0 {
		t.Fatalf("watchdog with an invalid threshold should fail closed (non-zero), got: %s", out)
	}
	if !strings.Contains(strings.ToUpper(out), "THRESHOLD") {
		t.Fatalf("invalid-threshold watchdog should name the config error, got: %s", out)
	}
	var rows int
	if err := st.DB.QueryRow("SELECT COUNT(*) FROM session_handoffs WHERE project=? AND handoff_id=?", proj, "wd-invalid").Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 0 {
		t.Fatalf("invalid-threshold watchdog must not hand off (no row), got %d", rows)
	}

	// (b) Bad --usage fraction (out of [0,1]) -> fail closed at arg validation.
	out, code = runSessionCLI(t, dataDir, proj, cwd,
		"handoff", "--watchdog", "--usage", "1.5",
		"--progress", "p", "--next-prompt", "n", "--handoff-id", "wd-badusage",
		"--project", proj, "--dir", dataDir)
	if code == 0 {
		t.Fatalf("watchdog with a bad --usage should fail closed (non-zero), got: %s", out)
	}
	if !strings.Contains(out, "--usage") {
		t.Fatalf("bad --usage should name the flag in stderr, got: %s", out)
	}
}

// TestSessionNoStoreResume is 04.3 [AFK] — resume/status also fail closed with
// no usable store: non-zero exit + clear error.
func TestSessionNoStoreResume(t *testing.T) {
	blockDir := t.TempDir()
	blockFile := filepath.Join(blockDir, "block")
	if err := os.WriteFile(blockFile, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write block file: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	proj, err := project.Resolve(cwd)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}

	out, code := runSessionCLI(t, blockFile, proj, cwd,
		"resume", "some-id", "--project", proj, "--dir", blockFile)
	if code == 0 {
		t.Fatalf("session resume with no usable store should exit non-zero, got: %s", out)
	}
	if !strings.Contains(out, "error") {
		t.Fatalf("session resume with no store should carry a clear error, got: %s", out)
	}

	out, code = runSessionCLI(t, blockFile, proj, cwd,
		"status", "--project", proj, "--dir", blockFile)
	if code == 0 {
		t.Fatalf("session status with no usable store should exit non-zero, got: %s", out)
	}
	if !strings.Contains(out, "error") {
		t.Fatalf("session status with no store should carry a clear error, got: %s", out)
	}
}
