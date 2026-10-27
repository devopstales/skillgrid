package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
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

// TestMemoryGlobalPrefetchFailsOpen is the G7 gate (TICKET-05 review fix):
// `memory global prefetch` is a fail-open seam — when the global store cannot
// be opened, the consumer (plugin / system-prompt injection) must never see an
// error: exit 0, nothing on stdout, nothing on stderr (empty string).
func TestMemoryGlobalPrefetchFailsOpen(t *testing.T) {
	// A file where the store wants a directory: OpenGlobal fails.
	dataDir := t.TempDir()
	if err := os.WriteFile(dataDir+"/blocker", []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	cmd := exec.Command("go", "run", ".", "memory", "global", "prefetch",
		"--dir", dataDir+"/blocker", "--session-id", "s-failopen")
	cmd.Dir = mustWD(t)
	var out, stderr strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("memory global prefetch must fail open with exit 0 (fail-open contract): %v\nstdout: %s\nstderr: %s",
			err, out.String(), stderr.String())
	}
	if out.String() != "" {
		t.Fatalf("fail-open prefetch must print nothing to stdout, got: %s", out.String())
	}
	if stderr.String() != "" {
		t.Fatalf("fail-open prefetch must print nothing to stderr, got: %s", stderr.String())
	}
}

// --- CLI surface for global memory (change 2026-10-24-global-llm-wiki,
// --- Change A, TICKET-08).

// openMemoryGlobalStore opens the global store pseudo-project store in dataDir
// (the CLI tests assert rows in it directly).
func openMemoryGlobalStore(t *testing.T, dataDir string) *store.Store {
	t.Helper()
	gst, err := store.Open(dataDir, "global")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	t.Cleanup(func() { gst.Close() })
	return gst
}

// runMemoryCLIGlobal runs `go run . memory global <args...>` against the data
// dir and returns the combined output; it fails the test on a non-zero exit.
func runMemoryCLIGlobal(t *testing.T, dataDir, project string, args ...string) string {
	t.Helper()
	full := append([]string{"run", ".", "memory", "global"}, args...)
	full = append(full, "--project", project, "--dir", dataDir)
	cmd := exec.Command("go", full...)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MEMORY_SESSION_ID=s-memcli")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("memory global %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// TestMemoryFactAddScopeGlobalTarget (BDD: "happy path — CLI adds a global
// fact with scope and target", G1): `memory fact add --scope global
// --target memory` writes the fact to global.sqlite with
// category='memory' and prints the new fact id.
func TestMemoryFactAddScopeGlobalTarget(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-scope-global"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()

	out := runMemoryCLI(t, dataDir, project, "fact", "add", "test fact", "--scope", "global", "--target", "memory")
	if !strings.Contains(out, "added fact") {
		t.Fatalf("memory fact add --scope global did not confirm the new fact id: %s", out)
	}

	gst := openMemoryGlobalStore(t, dataDir)
	var content, category string
	if err := gst.DB.QueryRow(
		`SELECT content, category FROM facts WHERE content = 'test fact'`,
	).Scan(&content, &category); err != nil {
		t.Fatalf("fact missing from the global store: %v", err)
	}
	if category != "memory" {
		t.Errorf("global fact category = %q, want 'memory'", category)
	}
	// The fact did NOT land in the project store.
	var projFacts int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts WHERE content = 'test fact'`).Scan(&projFacts); err != nil {
		t.Fatalf("count project facts: %v", err)
	}
	if projFacts != 0 {
		t.Errorf("project store holds %d rows for the global fact, want 0", projFacts)
	}
}

// TestMemoryFactAddTargetUser (triangulation of the target plumbing):
// --target user stamps category='user' in the global store.
func TestMemoryFactAddTargetUser(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-scope-user"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()

	out := runMemoryCLI(t, dataDir, project, "fact", "add", "user profile note", "--scope", "global", "--target", "user")
	if !strings.Contains(out, "added fact") {
		t.Fatalf("memory fact add --scope global --target user did not confirm: %s", out)
	}
	gst := openMemoryGlobalStore(t, dataDir)
	var category string
	if err := gst.DB.QueryRow(
		`SELECT category FROM facts WHERE content = 'user profile note'`,
	).Scan(&category); err != nil {
		t.Fatalf("fact missing from the global store: %v", err)
	}
	if category != "user" {
		t.Errorf("global fact category = %q, want 'user'", category)
	}
}

// TestMemoryFactAddProjectScopeDefault (BDD: "happy path — project fact add
// writes to project store (unchanged)"): without --scope the CLI keeps the
// pre-change behavior — the fact lands in the project store only.
func TestMemoryFactAddProjectScopeDefault(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-scope-default"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()

	out := runMemoryCLI(t, dataDir, project, "fact", "add", "project default note")
	if !strings.Contains(out, "added fact") {
		t.Fatalf("memory fact add (default scope) did not confirm: %s", out)
	}
	var projFacts int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts WHERE content = 'project default note'`).Scan(&projFacts); err != nil {
		t.Fatalf("count project facts: %v", err)
	}
	if projFacts != 1 {
		t.Errorf("project store holds %d rows, want 1 (the default-scope fact)", projFacts)
	}
	var globalFacts int
	if err := openMemoryGlobalStore(t, dataDir).DB.QueryRow(
		`SELECT COUNT(*) FROM facts WHERE content = 'project default note'`).Scan(&globalFacts); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if globalFacts != 0 {
		t.Errorf("global store holds %d rows, want 0 (project-scoped add)", globalFacts)
	}
}

// TestMemoryGlobalPendingList (BDD: "happy path — CLI lists and approves
// pending facts", G2): staged pending rows are listed with their content and
// target, in both stores (the queue spans global + project).
func TestMemoryGlobalPendingList(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-pending-list"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()
	gst := openMemoryGlobalStore(t, dataDir)

	now := "2026-10-24T00:00:00Z"
	if _, err := gst.DB.Exec(`
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES ('pending global note', 'memory', 'global', ?, 'pending')`, now); err != nil {
		t.Fatalf("seed global pending: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES ('pending project note', 'user', 'project', ?, 'pending')`, now); err != nil {
		t.Fatalf("seed project pending: %v", err)
	}

	out := runMemoryCLI(t, dataDir, project, "global", "pending")
	for _, want := range []string{"pending global note", "pending project note", "memory", "user"} {
		if !strings.Contains(out, want) {
			t.Fatalf("memory global pending missing %q: %s", want, out)
		}
	}
}

// TestMemoryGlobalApproveAll (BDD: "happy path — CLI lists and approves
// pending facts", G3): `approve all` applies every staged row into the facts
// table of its own store and empties the queue.
func TestMemoryGlobalApproveAll(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-pending-approve"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()
	gst := openMemoryGlobalStore(t, dataDir)

	now := "2026-10-24T00:00:00Z"
	if _, err := gst.DB.Exec(`
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES ('approved global note', 'memory', 'global', ?, 'pending')`, now); err != nil {
		t.Fatalf("seed global pending: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES ('approved project note', 'user', 'project', ?, 'pending')`, now); err != nil {
		t.Fatalf("seed project pending: %v", err)
	}

	out := runMemoryCLI(t, dataDir, project, "global", "approve", "all")
	if !strings.Contains(out, "2") {
		t.Fatalf("memory global approve all did not report 2 approved: %s", out)
	}

	var globalFacts, projectFacts, globalPending, projectPending int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM facts WHERE content = 'approved global note'`).Scan(&globalFacts); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts WHERE content = 'approved project note'`).Scan(&projectFacts); err != nil {
		t.Fatalf("count project facts: %v", err)
	}
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM pending_facts`).Scan(&globalPending); err != nil {
		t.Fatalf("count global pending: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pending_facts`).Scan(&projectPending); err != nil {
		t.Fatalf("count project pending: %v", err)
	}
	if globalFacts != 1 || projectFacts != 1 {
		t.Errorf("approved facts = global %d project %d, want 1/1", globalFacts, projectFacts)
	}
	if globalPending != 0 || projectPending != 0 {
		t.Errorf("pending rows after approve all = global %d project %d, want 0/0", globalPending, projectPending)
	}
}

// TestMemoryGlobalRejectAll (BDD: "failure — rejected pending fact is
// discarded", CLI layer): `reject all` discards every staged row without
// writing any fact row.
func TestMemoryGlobalRejectAll(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-pending-reject"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()
	gst := openMemoryGlobalStore(t, dataDir)

	now := "2026-10-24T00:00:00Z"
	if _, err := gst.DB.Exec(`
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES ('rejected global note', 'memory', 'global', ?, 'pending')`, now); err != nil {
		t.Fatalf("seed global pending: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES ('rejected project note', 'user', 'project', ?, 'pending')`, now); err != nil {
		t.Fatalf("seed project pending: %v", err)
	}

	out := runMemoryCLI(t, dataDir, project, "global", "reject", "all")
	if !strings.Contains(out, "2") {
		t.Fatalf("memory global reject all did not report 2 rejected: %s", out)
	}
	var globalFacts, projectFacts, globalPending, projectPending int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&globalFacts); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&projectFacts); err != nil {
		t.Fatalf("count project facts: %v", err)
	}
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM pending_facts`).Scan(&globalPending); err != nil {
		t.Fatalf("count global pending: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pending_facts`).Scan(&projectPending); err != nil {
		t.Fatalf("count project pending: %v", err)
	}
	if globalFacts != 0 || projectFacts != 0 {
		t.Errorf("rejected rows wrote facts = global %d project %d, want 0/0", globalFacts, projectFacts)
	}
	if globalPending != 0 || projectPending != 0 {
		t.Errorf("pending rows after reject all = global %d project %d, want 0/0", globalPending, projectPending)
	}
}

// TestMemoryGlobalPrefetchSnapshot (BDD: "edge — CLI prefetch prints the
// frozen snapshot", G4): the snapshot block carries both labeled section
// headers and the § separator between entries.
func TestMemoryGlobalPrefetchSnapshot(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-prefetch-snap"
	st := seedMemoryCLIStore(t, dataDir, project)
	defer st.Close()
	gst := openMemoryGlobalStore(t, dataDir)

	if _, err := gst.DB.Exec(`
		INSERT INTO facts (content, category, created_at, updated_at)
		VALUES ('first global memory entry', 'memory', '2026-10-24T00:00:00Z', '2026-10-24T00:00:00Z')`); err != nil {
		t.Fatalf("seed global fact: %v", err)
	}
	if _, err := gst.DB.Exec(`
		INSERT INTO facts (content, category, created_at, updated_at)
		VALUES ('second global memory entry', 'memory', '2026-10-24T00:00:01Z', '2026-10-24T00:00:01Z')`); err != nil {
		t.Fatalf("seed global fact: %v", err)
	}
	if _, err := gst.DB.Exec(`
		INSERT INTO facts (content, category, created_at, updated_at)
		VALUES ('one user profile entry', 'user', '2026-10-24T00:00:02Z', '2026-10-24T00:00:02Z')`); err != nil {
		t.Fatalf("seed user fact: %v", err)
	}

	out := runMemoryCLI(t, dataDir, project, "global", "prefetch")
	for _, want := range []string{"GLOBAL MEMORY", "USER PROFILE", "first global memory entry", "one user profile entry", "§"} {
		if !strings.Contains(out, want) {
			t.Fatalf("memory global prefetch output missing %q: %s", want, out)
		}
	}
}

// TestMemoryFactInvalidScope (BDD: "failure — CLI with invalid scope returns
// usage error", G5): an unrecognized --scope is a usage error (exit 2) that
// lists the valid values, before any store is opened.
func TestMemoryFactInvalidScope(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-scope-invalid"

	out, err := runMemoryCLIExpectError(t, dataDir, project, "fact", "add", "scope fact", "--scope", "invalid")
	if err == nil {
		t.Fatalf("memory fact add --scope invalid should fail:\n%s", out)
	}
	for _, want := range []string{"project", "global"} {
		if !strings.Contains(out, want) {
			t.Fatalf("invalid scope error must list the valid values, missing %q: %s", want, out)
		}
	}
	// The store was never opened.
	if _, err := os.Stat(dataDir + "/" + project + ".sqlite"); err == nil {
		t.Fatalf("an invalid scope must not open the store (project db exists)")
	}
}

// TestMemoryFactScopeAllOnAddIsUsageError (triangulation of G5): --scope all
// on a write op is a usage error (exit 2, before the store opens) — Route
// rejects writes to scope=all (fused search is read-only).
func TestMemoryFactScopeAllOnAddIsUsageError(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-scope-all-add"

	out, err := runMemoryCLIExpectError(t, dataDir, project, "fact", "add", "scope all fact", "--scope", "all")
	if err == nil {
		t.Fatalf("memory fact add --scope all should fail:\n%s", out)
	}
	if !strings.Contains(out, "read-only") {
		t.Fatalf("scope-all add error should say the scope is read-only: %s", out)
	}
	// The store was never opened.
	if _, err := os.Stat(dataDir + "/" + project + ".sqlite"); err == nil {
		t.Fatalf("a scope-all add must not open the store (project db exists)")
	}
}

// TestMemoryFactScopeAllOnListIsUsageError (T08 review fix): --scope all
// is the fused SEARCH path; there is no fused list, so list --scope all is
// a usage error (exit 2, before the store opens) instead of silently
// reading only the global store.
func TestMemoryFactScopeAllOnListIsUsageError(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-scope-all-list"

	out, err := runMemoryCLIExpectError(t, dataDir, project, "fact", "list", "--scope", "all")
	if err == nil {
		t.Fatalf("memory fact list --scope all should fail:\n%s", out)
	}
	if !strings.Contains(out, "search") {
		t.Fatalf("scope-all list error should point at the fused search path: %s", out)
	}
	// The store was never opened.
	if _, err := os.Stat(dataDir + "/" + project + ".sqlite"); err == nil {
		t.Fatalf("a scope-all list must not open the store (project db exists)")
	}
}

// TestMemoryFactInvalidTarget (triangulation of G5): an unrecognized
// --target is the same usage-error shape (exit 2, valid values listed).
func TestMemoryFactInvalidTarget(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-target-invalid"

	out, err := runMemoryCLIExpectError(t, dataDir, project, "fact", "add", "target fact", "--scope", "global", "--target", "bogus")
	if err == nil {
		t.Fatalf("memory fact add --target bogus should fail:\n%s", out)
	}
	for _, want := range []string{"memory", "user"} {
		if !strings.Contains(out, want) {
			t.Fatalf("invalid target error must list the valid values, missing %q: %s", want, out)
		}
	}
}

// TestMemoryGlobalPrefetchFailsOpenJSON is the triangulation of the fail-open
// gate with a different input: the same dead-store failure with --json must
// also be silent and exit 0 — the JSON consumer gets the empty string too.
func TestMemoryGlobalPrefetchFailsOpenJSON(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(dataDir+"/blocker", []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	cmd := exec.Command("go", "run", ".", "memory", "global", "prefetch",
		"--dir", dataDir+"/blocker", "--session-id", "s-failopen-json", "--json")
	cmd.Dir = mustWD(t)
	var out, stderr strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("memory global prefetch --json must fail open with exit 0: %v\nstdout: %s\nstderr: %s",
			err, out.String(), stderr.String())
	}
	if out.String() != "" || stderr.String() != "" {
		t.Fatalf("fail-open prefetch --json must be silent, got stdout=%q stderr=%q",
			out.String(), stderr.String())
	}
}
