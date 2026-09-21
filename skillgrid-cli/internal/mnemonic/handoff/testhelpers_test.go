package handoff

import (
	"os"
	"os/exec"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// storeOpen opens a fresh per-test store (one SQLite file in a temp dir).
func storeOpen(t *testing.T) (*store.Store, error) {
	t.Helper()
	dir := t.TempDir()
	return store.Open(dir, "hubtest")
}

// runGit runs a git command in dir, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"SKILLGRID_PROTECTED_BRANCHES=protected-main")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
