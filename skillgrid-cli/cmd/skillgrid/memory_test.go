package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedMemoryCLIStore opens a project store and inserts a sessions row so the
// facts trail (session_events FK on session_id) has a valid session to attach
// to. It returns the store so the caller can seed facts directly.
func seedMemoryCLIStore(t *testing.T, dataDir, project string) *store.Store {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s-memcli', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return st
}

// runMemoryCLI runs `go run . memory <args...>` with the given data dir and
// returns the combined output; it fails the test on a non-zero exit.
func runMemoryCLI(t *testing.T, dataDir, project string, args ...string) string {
	t.Helper()
	full := append([]string{"run", ".", "memory"}, args...)
	full = append(full, "--project", project, "--dir", dataDir)
	cmd := exec.Command("go", full...)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MEMORY_SESSION_ID=s-memcli")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("memory %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// runMemoryCLIExpectError runs the memory CLI without failing the test on a
// non-zero exit (negative-arg assertions).
func runMemoryCLIExpectError(t *testing.T, dataDir, project string, args ...string) (string, error) {
	t.Helper()
	full := append([]string{"run", ".", "memory"}, args...)
	full = append(full, "--project", project, "--dir", dataDir)
	cmd := exec.Command("go", full...)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MEMORY_SESSION_ID=s-memcli")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestMemoryFactAddAndList is 04 [failure] — `memory fact add` matches the
// fact_add MCP (returns the new fact id) and `memory fact list` shows it.
// Scenario: CLI memory and skill match MCP or fail cleanly.
func TestMemoryFactAddAndList(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-fact-add"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()

	out := runMemoryCLI(t, dataDir, project, "fact", "add", "the deployment target is eu-west-1")
	if !strings.Contains(out, "added fact 1") {
		t.Fatalf("memory fact add did not confirm the new fact id: %s", out)
	}
	// --json emits the machine-readable fact_add shape.
	out = runMemoryCLI(t, dataDir, project, "fact", "add", "--content", "second fact", "--json")
	if !strings.Contains(out, `"event": "fact_add"`) || !strings.Contains(out, `"fact_id": 2`) {
		t.Fatalf("memory fact add --json missing the fact_add shape: %s", out)
	}

	// list shows both live facts, newest first, no trail event required.
	out = runMemoryCLI(t, dataDir, project, "fact", "list")
	if !strings.Contains(out, "eu-west-1") || !strings.Contains(out, "second fact") {
		t.Fatalf("memory fact list missing the added facts: %s", out)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "CONTENT") {
		t.Fatalf("memory fact list should print the table header: %s", out)
	}
	out = runMemoryCLI(t, dataDir, project, "fact", "list", "--json")
	if !strings.Contains(out, `"count": 2`) {
		t.Fatalf("memory fact list --json missing the count: %s", out)
	}
}

// TestMemoryFactSearch is 04 [failure] — `memory fact search` matches the
// fact_search MCP: it finds the seeded fact, honors --mode hybrid (degrading
// to the lexical FTS path), and rejects an invalid --mode before any store is
// opened (exit 2).
func TestMemoryFactSearch(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-fact-search"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()
	st.DB.Exec(`INSERT INTO facts (content, created_at, updated_at) VALUES ('k8s ingress uses nginx upstream', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')`)

	// Lexical FTS search finds the seeded fact.
	out := runMemoryCLI(t, dataDir, project, "fact", "search", "nginx")
	if !strings.Contains(out, "k8s ingress") {
		t.Fatalf("memory fact search did not find the seeded fact: %s", out)
	}
	// --mode hybrid is accepted (degrades to the lexical path, no embedder).
	out = runMemoryCLI(t, dataDir, project, "fact", "search", "ingress", "--mode", "hybrid")
	if !strings.Contains(out, "k8s ingress") {
		t.Fatalf("memory fact search --mode hybrid did not find the seeded fact: %s", out)
	}
	// --json emits the facts array shape.
	out = runMemoryCLI(t, dataDir, project, "fact", "search", "nginx", "--json")
	if !strings.Contains(out, `"facts":`) || !strings.Contains(out, `"count": 1`) {
		t.Fatalf("memory fact search --json missing the facts shape: %s", out)
	}
	// An invalid --mode fails before any store access (exit 2, usage error).
	out, err := runMemoryCLIExpectError(t, dataDir, project, "fact", "search", "nginx", "--mode", "trigram")
	if err == nil {
		t.Fatalf("memory fact search --mode trigram should fail:\n%s", out)
	}
	if !strings.Contains(out, "--mode must be fts or hybrid") {
		t.Fatalf("memory fact search invalid mode missing the clear error: %s", out)
	}
}

// TestMemoryFactSearchHybridDegraded is the --mode hybrid leg with no embedder
// active: it must route through hybrid.SearchMemory (RRF-fused output), print
// the fused table (ID/SCORE/SOURCE columns), and note the FTS-only degradation
// on stderr. The --json form carries the legs bookkeeping instead of a note.
func TestMemoryFactSearchHybridDegraded(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-fact-hybrid"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()
	st.DB.Exec(`INSERT INTO facts (content, created_at, updated_at) VALUES ('k8s ingress uses nginx upstream', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')`)

	// Table form: fused RRF columns + the no-embedder note on stderr.
	out := runMemoryCLI(t, dataDir, project, "fact", "search", "ingress", "--mode", "hybrid")
	if !strings.Contains(out, "k8s ingress") {
		t.Fatalf("hybrid search did not find the seeded fact: %s", out)
	}
	if !strings.Contains(out, "hybrid mode: no embedder available, using FTS-only") {
		t.Fatalf("hybrid search without an embedder must note the degradation: %s", out)
	}
	if !strings.Contains(out, "ID\tSCORE") && !strings.Contains(out, "SCORE") {
		t.Fatalf("hybrid search table should carry the fused score column: %s", out)
	}

	// JSON form: fused facts + legs bookkeeping, no stderr note in the object.
	out = runMemoryCLI(t, dataDir, project, "fact", "search", "ingress", "--mode", "hybrid", "--json")
	if !strings.Contains(out, `"legs":`) || !strings.Contains(out, `"fts"`) {
		t.Fatalf("hybrid search --json missing the legs bookkeeping: %s", out)
	}
	if !strings.Contains(out, `"facts":`) || !strings.Contains(out, "k8s ingress") {
		t.Fatalf("hybrid search --json missing the fused facts: %s", out)
	}
}

// TestMemoryFactForget is 04 [failure] — `memory fact forget` matches the
// fact_forget MCP: it soft-deletes the fact, the fact disappears from the
// default list, and a non-numeric fact id fails before any store is opened.
func TestMemoryFactForget(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-fact-forget"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()
	st.DB.Exec(`INSERT INTO facts (content, created_at, updated_at) VALUES ('forget me please', '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z')`)

	out := runMemoryCLI(t, dataDir, project, "fact", "forget", "1")
	if !strings.Contains(out, "forgotten fact 1") {
		t.Fatalf("memory fact forget did not confirm the soft-delete: %s", out)
	}
	// The default list now hides the soft-deleted fact.
	out = runMemoryCLI(t, dataDir, project, "fact", "list")
	if strings.Contains(out, "forget me please") {
		t.Fatalf("memory fact list should hide the forgotten fact: %s", out)
	}
	// --include-deleted brings it back.
	out = runMemoryCLI(t, dataDir, project, "fact", "list", "--include-deleted")
	if !strings.Contains(out, "forget me please") {
		t.Fatalf("memory fact list --include-deleted missing the forgotten fact: %s", out)
	}
	// A non-numeric fact id is a usage error (exit 2) before the store opens.
	out, err := runMemoryCLIExpectError(t, dataDir, project, "fact", "forget", "nope")
	if err == nil {
		t.Fatalf("memory fact forget with a non-numeric id should fail:\n%s", out)
	}
	if !strings.Contains(out, "requires a numeric --fact-id") {
		t.Fatalf("memory fact forget invalid id missing the clear error: %s", out)
	}
}

// TestMemoryFactDecay is 04 [failure] — `memory fact decay` matches the
// fact_decay MCP: it applies the 014 AKL importance decay and reports the new
// score.
func TestMemoryFactDecay(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-fact-decay"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()
	st.DB.Exec(`INSERT INTO facts (content, created_at, updated_at) VALUES ('decay target fact', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)

	out := runMemoryCLI(t, dataDir, project, "fact", "decay", "1")
	if !strings.Contains(out, "decayed fact 1") || !strings.Contains(out, "score") {
		t.Fatalf("memory fact decay did not report the new score: %s", out)
	}
	out = runMemoryCLI(t, dataDir, project, "fact", "decay", "1", "--json")
	if !strings.Contains(out, `"event": "fact_decay"`) || !strings.Contains(out, `"new_score":`) {
		t.Fatalf("memory fact decay --json missing the fact_decay shape: %s", out)
	}
}

// TestMemoryFactInvalidAction is 04 [failure] — an unknown memory fact
// command fails with a clear error before any store is opened, so Fact
// Memory is never corrupted by a usage mistake.
func TestMemoryFactInvalidAction(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-fact-invalid"

	// Unknown fact subcommand → usage error, no store touched.
	out, err := runMemoryCLIExpectError(t, dataDir, project, "fact", "warp", "1")
	if err == nil {
		t.Fatalf("memory fact warp should fail:\n%s", out)
	}
	if !strings.Contains(out, "unknown memory fact command") {
		t.Fatalf("memory fact invalid command missing the clear error: %s", out)
	}
	// Unknown group (not "fact") → usage error too.
	out, err = runMemoryCLIExpectError(t, dataDir, project, "note", "add", "x")
	if err == nil {
		t.Fatalf("memory note add should fail:\n%s", out)
	}
	if !strings.Contains(out, "unknown memory group") {
		t.Fatalf("memory unknown group missing the clear error: %s", out)
	}
	// The store was never opened: no project db file was created.
	if _, err := os.Stat(dataDir + "/" + project + ".sqlite"); err == nil {
		t.Fatalf("invalid memory actions must not open the store (project db exists)")
	}
	// Bare `memory` with no group → usage on stderr, exit 2.
	out, err = runMemoryCLIExpectError(t, dataDir, project)
	if err == nil || !strings.Contains(out, "usage: skillgrid memory fact") {
		t.Fatalf("bare memory should print usage: %s (err=%v)", out, err)
	}
}

// TestMemoryFactAddRequiresSession is 04 [failure] — `memory fact add` with
// no session id (flag or env) is a usage error before the store opens: the
// trail event FK would otherwise reject the insert.
func TestMemoryFactAddRequiresSession(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-fact-no-session"
	cmdArgs := []string{"run", ".", "memory", "fact", "add", "no session fact",
		"--project", project, "--dir", dataDir}
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MEMORY_SESSION_ID=")
	raw, err := cmd.CombinedOutput()
	out := string(raw)
	if err == nil {
		t.Fatalf("memory fact add without a session should fail:\n%s", out)
	}
	if !strings.Contains(out, "requires a --session-id") {
		t.Fatalf("memory fact add missing the session error: %s", out)
	}
	// The store was never opened.
	if _, err := os.Stat(dataDir + "/" + project + ".sqlite"); err == nil {
		t.Fatalf("a missing session must not open the store (project db exists)")
	}
}
