package main

// cli_test.go — the CLI's command-surface tests: the project-selection
// rules (spec 005-transport-cli §3), the strict doctor's exit contract,
// and the thin stubs that delegate to the other changes.

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// testCtx is the fixed CLI test context (the store's SQLite handle is
// process-scoped in the tests; per-call cancellation is a transport
// concern the CLI passes through untouched).
func testCtx() context.Context { return context.Background() }

// stubDetector is the fixed detector the CLI test fixtures use (no git
// shell-out): it wraps the real project.Detector (so the env var and the
// chain still apply) but passes a no-op git binding (no repository in
// the temp dirs). The tests that need the fixture project pass it
// explicitly.
type stubDetector struct {
	det project.Detector
}

func (s stubDetector) Resolve(cwd, explicit string) (string, error) {
	return s.det.Resolve(cwd, explicit, func(string) (string, error) {
		return "", os.ErrNotExist
	})
}

// openTestStore opens a real SQLite store for project "p" in dir (the CLI
// fixtures use the same store production opens — the wiring under test is
// the dispatch, not the storage).
func openTestStore(t *testing.T, dir string) store.Store {
	t.Helper()
	st, err := store.Open(testCtx(), dir, "p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// newTestHandler builds a Handler over a real SQLite store with nil
// intelligence seams (absent seams → the INV-3 deterministic floors) and
// the stub detector (no git in tests).
func newTestHandler(t *testing.T) *handler.Handler {
	t.Helper()
	st := openTestStore(t, t.TempDir())
	return newTestHandlerWithStore(t, st)
}

// newTestHandlerWithStore builds a Handler over a caller-provided store
// (the run helpers inject the fixture store; the parity tests share one
// store across the CLI and MCP paths). The detector is the stub (the env
// var still applies; git is a no-op).
func newTestHandlerWithStore(t *testing.T, st store.Store) *handler.Handler {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	det := project.Detector{EnvVar: c.Project.EnvVar}
	return handler.NewHandler(memory.NewService(st, c, memory.NewEventBus(), nil, nil), c, stubDetector{det: det})
}

// saveReq builds one valid save request for project p.
func saveReq(p, title string) memory.SaveRequest {
	return memory.SaveRequest{
		SessionID: "s", Type: "decision", Title: title, Content: "C", Project: p,
	}
}

// TestCLIProjectSelection [cli-project-selection]: --project beats
// MNEMONIC_PROJECT; --all reads every project; explicit + --all is a
// usage error.
func TestCLIProjectSelection(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "envproj")
	err := runCLIError(t, "mem", "search", "q", "--project", "explicit", "--all")
	if err == nil {
		t.Fatalf("explicit project + --all must be a usage error")
	}
	if !strings.Contains(err.Error(), "--all") {
		t.Fatalf("the usage error must name the --all conflict: %v", err)
	}
}

// TestCLIAllFanoutNotWired [cli-project-selection] fix round 1: --all
// alone (no explicit project) is a usage error until the cross-project
// store lands — it must NOT silently resolve to the current project (one
// project's rows). The message names the deferred fan-out and points at
// --project.
func TestCLIAllFanoutNotWired(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "envproj")
	out, code := captureRun(t, "mem", "search", "q", "--all")
	if code != exitUsage {
		t.Fatalf("bare --all on a read command must be a usage error (exit 2), got %d (output: %s)", code, out)
	}
	if !strings.Contains(out, "--all") || !strings.Contains(out, "not yet wired") {
		t.Fatalf("the usage error must name --all and the deferred fan-out: %s", out)
	}
}

// TestCLIAllFanoutNotWiredGet [cli-project-selection] triangulation: the
// rejection is per read command, not search-specific — --all on `mem get`
// is refused the same way (it too would silently resolve to one project).
func TestCLIAllFanoutNotWiredGet(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "envproj")
	out, code := captureRun(t, "mem", "get", "--id", "1", "--all")
	if code != exitUsage {
		t.Fatalf("--all on mem get must be a usage error (exit 2), got %d (output: %s)", code, out)
	}
	if !strings.Contains(out, "not yet wired") {
		t.Fatalf("the usage error must name the deferred fan-out: %s", out)
	}
}

// TestDoctorStrictExitsNonZeroOnGap [doctor-strict-exits-nonzero-on-gap]:
// no embedder configured → doctor reports degraded → --strict exits 1.
func TestDoctorStrictExitsNonZeroOnGap(t *testing.T) {
	code := runCLIExit(t, "doctor", "--strict")
	if code != 1 {
		t.Fatalf("strict doctor with a capability gap must exit 1, got %d", code)
	}
}

// TestDoctorNonStrictExitsZeroOnGap [doctor-strict-exits-nonzero-on-gap]
// triangulation: the same gap without --strict exits 0 (the report is
// human-readable; strict mode is the CI switch).
func TestDoctorNonStrictExitsZeroOnGap(t *testing.T) {
	code := runCLIExit(t, "doctor")
	if code != 0 {
		t.Fatalf("non-strict doctor reports the gap but exits 0, got %d", code)
	}
}

// TestDoctorJSONCarriesChecks: the --json doctor output carries the
// per-check rows (the machine-readable half of the contract).
func TestDoctorJSONCarriesChecks(t *testing.T) {
	out := runCLI(t, "doctor", "--json")
	var env map[string]any
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("doctor --json output is not a JSON object: %v (%s)", err, out)
	}
	checks, ok := env["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatalf("doctor --json must carry the check rows: %s", out)
	}
}

// TestServePrintsPointer: the deprecated serve alias prints a pointer to
// mcp/CLI and exits 0 (the spec: no-op alias until v2.1).
func TestServePrintsPointer(t *testing.T) {
	out := runCLI(t, "serve")
	if !strings.Contains(out, "mcp") {
		t.Fatalf("serve must point at the mcp command: %s", out)
	}
}

// TestStubSubcommands: setup is a delegation stub — it prints a pointer (the
// dispatch is complete; the delegation is a one-line change later when the
// owning change lands). index is now a REAL command (TICKET-07 — the
// code-index CLI group); it requires a project (the detector's resolution)
// and exits 0 for a read (status) when a project is given. migrate (the
// v1→v2 seed, TICKET-19) is removed from the CLI surface — it is now an
// unknown command (the cutover is the operator's call: delete the v1 store
// and generate a fresh v2 db, or run the seed programmatically).
func TestStubSubcommands(t *testing.T) {
	// setup is the remaining delegation stub.
	if out := runCLI(t, "setup", "opencode"); out == "" {
		t.Fatalf("setup must print a delegation pointer")
	}
	// index is now a real command: the status read (a project is required —
	// the detector's resolution). A bare `index` with no project selector in
	// a non-git cwd is a handler error (exit 1), not a stub pointer.
	installRunEnv(t)
	if _, code := captureRun(t, "index", "status", "--project", "p"); code != exitOK {
		t.Fatalf("index status --project p must exit 0 (the code_status read); got %d", code)
	}
	// migrate is removed: it is now an unknown command (usage error).
	if _, code := captureRun(t, "migrate"); code != exitUsage {
		t.Fatalf("migrate (removed) must exit %d as an unknown command; got %d", exitUsage, code)
	}
}

// TestTrailRecent: trail recent prints the newest retrieval trails
// (the CLI exposes the INV-5 audit trail; the store records it).
func TestTrailRecent(t *testing.T) {
	installRunEnv(t)
	// Seed a retrieval (the search records the trail).
	_, code := captureRun(t, "mem", "search", "seedquery", "--project", "p")
	if code != exitOK {
		t.Fatalf("seed search exited %d", code)
	}
	out, code := captureRun(t, "trail", "recent", "--project", "p")
	if code != exitOK {
		t.Fatalf("trail recent exited %d (output: %s)", code, out)
	}
	if !strings.Contains(out, "seedquery") {
		t.Fatalf("trail recent must list the seeded retrieval: %s", out)
	}
}

// TestTrailShow: trail show prints one trail by id.
func TestTrailShow(t *testing.T) {
	installRunEnv(t)
	_, code := captureRun(t, "mem", "search", "showquery", "--project", "p")
	if code != exitOK {
		t.Fatalf("seed search exited %d", code)
	}
	out, code := captureRun(t, "trail", "show", "1", "--project", "p")
	if code != exitOK {
		t.Fatalf("trail show exited %d (output: %s)", code, out)
	}
	if !strings.Contains(out, "showquery") {
		t.Fatalf("trail show must print the trail row: %s", out)
	}
}

// TestSyncStatus: sync status prints the per-target lifecycle table
// (an empty table when nothing has synced — the table header still
// renders).
func TestSyncStatus(t *testing.T) {
	out := runCLI(t, "sync", "status")
	if !strings.Contains(out, "target") {
		t.Fatalf("sync status must print the target table: %s", out)
	}
}

// TestSyncEnroll: sync enroll toggles a project's enrollment (the
// enrollment gate the save's ENQUEUE step reads).
func TestSyncEnroll(t *testing.T) {
	out := runCLI(t, "sync", "enroll", "p")
	if !strings.Contains(strings.ToLower(out), "enroll") {
		t.Fatalf("sync enroll must confirm the enrollment: %s", out)
	}
}
