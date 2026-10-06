package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// sessionCLIFixture opens a fresh project store (the SAME store file the CLI
// will open via --project/--dir) and resolves the CWD project id the SAME way
// the CLI resolves it (project.Resolve on CWD). The CLI is run in its real
// package dir with CWD pinned to the module package and the store pinned via
// --project/--dir. Returns the data dir, the pinned project id, and the store.
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
// package dir with CWD pinned there and the store pinned via --project/--dir.
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
