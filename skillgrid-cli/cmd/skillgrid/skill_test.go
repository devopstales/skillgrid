package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedSkillCLIStore opens a project store under the given data dir (the skills
// registry lives in SQL + the workspace FS, so only the db needs seeding).
func seedSkillCLIStore2(t *testing.T, dataDir, project string) *store.Store {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return st
}

// runSkillCLI runs `go run . skill <args...>` with the given data dir and
// returns the combined output; it fails the test on a non-zero exit.
func runSkillCLI(t *testing.T, dataDir, project string, args ...string) string {
	t.Helper()
	full := append([]string{"run", ".", "skill"}, args...)
	full = append(full, "--project", project, "--dir", dataDir)
	cmd := exec.Command("go", full...)
	cmd.Dir = mustWD(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("skill %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// runSkillCLIExpectError runs the skill CLI without failing the test on a
// non-zero exit (negative-arg assertions).
func runSkillCLIExpectError(t *testing.T, dataDir, project string, args ...string) (string, error) {
	t.Helper()
	full := append([]string{"run", ".", "skill"}, args...)
	full = append(full, "--project", project, "--dir", dataDir)
	cmd := exec.Command("go", full...)
	cmd.Dir = mustWD(t)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestSkillWriteListSearch is 04 [failure] — `skill write` matches the
// write_skill MCP (returns the skill id + code path and writes the FS file),
// `skill list` matches list_skills, and `skill search` matches search_skills.
// Scenario: CLI memory and skill match MCP or fail cleanly.
func TestSkillWriteListSearch(t *testing.T) {
	dataDir := t.TempDir()
	project := "skillcli-write"
	st := seedSkillCLIStore2(t, dataDir, project)
	defer st.Close()
	root := t.TempDir()

	// write: creates the skill (SQL row + FS file under root/.skillgrid/files/skills).
	out := runSkillCLI(t, dataDir, project, "write",
		"--name", "hello", "--language", "sh",
		"--code", "echo hello from skill", "--description", "prints hello",
		"--root", root)
	if !strings.Contains(out, "wrote skill \"hello\"") || !strings.Contains(out, "id 1") {
		t.Fatalf("skill write did not confirm the new skill: %s", out)
	}
	codePath := root + "/.skillgrid/files/skills/hello.sh"
	if _, err := os.Stat(codePath); err != nil {
		t.Fatalf("skill write did not write the FS file: %v", err)
	}
	// --json emits the machine-readable skill_write shape.
	out = runSkillCLI(t, dataDir, project, "write",
		"--name", "second", "--language", "python", "--code", "print(1)",
		"--root", root, "--json")
	if !strings.Contains(out, `"event": "skill_write"`) || !strings.Contains(out, `"skill_id": 2`) {
		t.Fatalf("skill write --json missing the skill_write shape: %s", out)
	}

	// list: shows both live skills (blank-query FTS falls back to the registry).
	out = runSkillCLI(t, dataDir, project, "list", "--root", root)
	if !strings.Contains(out, "hello") || !strings.Contains(out, "second") {
		t.Fatalf("skill list missing the written skills: %s", out)
	}
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "LANGUAGE") {
		t.Fatalf("skill list should print the table header: %s", out)
	}
	// search: lexical FTS over the registry finds by description/name.
	out = runSkillCLI(t, dataDir, project, "search", "prints", "--root", root)
	if !strings.Contains(out, "hello") {
		t.Fatalf("skill search did not find the matching skill: %s", out)
	}
	out = runSkillCLI(t, dataDir, project, "search", "--query", "prints hello", "--root", root, "--json")
	if !strings.Contains(out, `"skills":`) || !strings.Contains(out, `"count": 1`) {
		t.Fatalf("skill search --json missing the skills shape: %s", out)
	}
}

// TestSkillSearchHybridDegraded is the --mode hybrid leg with no embedder
// active: it must route through hybrid.SearchMemory over the skills scope,
// print the fused table (ID/SCORE/SOURCE columns), and note the FTS-only
// degradation on stderr. The --json form carries the legs bookkeeping.
func TestSkillSearchHybridDegraded(t *testing.T) {
	dataDir := t.TempDir()
	project := "skillcli-hybrid"
	st := seedSkillCLIStore2(t, dataDir, project)
	defer st.Close()
	root := t.TempDir()

	runSkillCLI(t, dataDir, project, "write",
		"--name", "greet", "--language", "sh",
		"--code", "echo greets $1", "--description", "prints a greeting",
		"--root", root)

	// Table form: fused RRF columns + the no-embedder note on stderr.
	out := runSkillCLI(t, dataDir, project, "search", "greeting", "--mode", "hybrid", "--root", root)
	if !strings.Contains(out, "greet") {
		t.Fatalf("hybrid skill search did not find the matching skill: %s", out)
	}
	if !strings.Contains(out, "hybrid mode: no embedder available, using FTS-only") {
		t.Fatalf("hybrid skill search without an embedder must note the degradation: %s", out)
	}
	if !strings.Contains(out, "SCORE") {
		t.Fatalf("hybrid skill search table should carry the fused score column: %s", out)
	}

	// JSON form: fused skills + legs bookkeeping.
	out = runSkillCLI(t, dataDir, project, "search", "greeting", "--mode", "hybrid", "--root", root, "--json")
	if !strings.Contains(out, `"legs":`) || !strings.Contains(out, `"fts"`) {
		t.Fatalf("hybrid skill search --json missing the legs bookkeeping: %s", out)
	}
	if !strings.Contains(out, `"skills":`) || !strings.Contains(out, "greet") {
		t.Fatalf("hybrid skill search --json missing the fused skills: %s", out)
	}
}

// TestSkillExecute is 04 [failure] — `skill execute` matches the use_skill
// MCP: it runs the skill in the sandbox and reports stdout/stderr/exit code.
// Scenario: CLI memory and skill match MCP or fail cleanly.
func TestSkillExecute(t *testing.T) {
	dataDir := t.TempDir()
	project := "skillcli-exec"
	st := seedSkillCLIStore2(t, dataDir, project)
	defer st.Close()
	root := t.TempDir()

	runSkillCLI(t, dataDir, project, "write",
		"--name", "greet", "--language", "sh",
		"--code", "echo greets $1", "--root", root)

	// execute: sandbox run reports stdout + exit 0.
	out := runSkillCLI(t, dataDir, project, "execute", "greet", "--input", "world", "--root", root)
	if !strings.Contains(out, "greets world") || !strings.Contains(out, "exit 0") {
		t.Fatalf("skill execute did not report the sandbox output: %s", out)
	}
	// --json emits the machine-readable result (stdout/exit_code/usage_id).
	out = runSkillCLI(t, dataDir, project, "execute", "greet", "--input", "world", "--root", root, "--json")
	if !strings.Contains(out, `"exit_code": 0`) || !strings.Contains(out, `"usage_id":`) {
		t.Fatalf("skill execute --json missing the result shape: %s", out)
	}
	// A failing script surfaces exit 1 + the failure status.
	runSkillCLI(t, dataDir, project, "write",
		"--name", "boom", "--language", "sh", "--code", "echo oops >&2; exit 3", "--root", root)
	out = runSkillCLI(t, dataDir, project, "execute", "boom", "--root", root)
	if !strings.Contains(out, "exit 3") || !strings.Contains(out, "failure") {
		t.Fatalf("skill execute should report the non-zero exit: %s", out)
	}
}

// TestSkillInvalidAction is 04 [failure] — an unknown skill command fails with
// a clear error before any store is opened, so the registry is never corrupted
// by a usage mistake.
func TestSkillInvalidAction(t *testing.T) {
	dataDir := t.TempDir()
	project := "skillcli-invalid"

	// Unknown skill subcommand → usage error, no store touched.
	out, err := runSkillCLIExpectError(t, dataDir, project, "warp")
	if err == nil {
		t.Fatalf("skill warp should fail:\n%s", out)
	}
	if !strings.Contains(out, "unknown skill command") {
		t.Fatalf("skill invalid command missing the clear error: %s", out)
	}
	// The store was never opened: no project db file was created.
	if _, err := os.Stat(dataDir + "/" + project + ".sqlite"); err == nil {
		t.Fatalf("invalid skill actions must not open the store (project db exists)")
	}
	// `skill write` missing required args → usage error.
	out, err = runSkillCLIExpectError(t, dataDir, project, "write", "--name", "x")
	if err == nil || !strings.Contains(out, "skill write requires") {
		t.Fatalf("skill write without language/code should fail: %s (err=%v)", out, err)
	}
	// `skill execute` without a name → usage error.
	out, err = runSkillCLIExpectError(t, dataDir, project, "execute")
	if err == nil || !strings.Contains(out, "skill execute requires") {
		t.Fatalf("skill execute without a name should fail: %s (err=%v)", out, err)
	}
	// `skill execute` with --session-id but no sessions row → clean failure
	// (FK violation), reported as an error, not a panic.
	out, err = runSkillCLIExpectError(t, dataDir, project, "execute", "greet",
		"--session-id", "no-such-session", "--root", t.TempDir())
	if err == nil {
		t.Fatalf("skill execute with an unknown session id should fail:\n%s", out)
	}
	if !strings.Contains(out, "error:") {
		t.Fatalf("skill execute with an unknown session id should print a clear error: %s", out)
	}
	// Bare `skill` with no command → usage on stderr.
	out, err = runSkillCLIExpectError(t, dataDir, project)
	if err == nil || !strings.Contains(out, "usage: skillgrid skill") {
		t.Fatalf("bare skill should print usage: %s (err=%v)", out, err)
	}
}
