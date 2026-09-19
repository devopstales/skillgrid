package http

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// gitRun runs a git command in dir, returning combined output (test helper).
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// newPhase6GitServer builds a server whose sddRoot points at a temp git repo
// with a few commits (two files, one modified across commits, one author), so
// the /git endpoints have data to return.
func newPhase6GitServer(t *testing.T) http.Handler {
	t.Helper()
	repo := t.TempDir()
	gitRun(t, repo, "init", "-q")
	gitRun(t, repo, "config", "user.email", "tester@example.com")
	gitRun(t, repo, "config", "user.name", "Tester")

	// Disable the inherited protected-branch hook so the fixture commits.
	gitRun(t, repo, "config", "core.hooksPath", filepath.Join(repo, ".nullhooks"))
	_ = os.MkdirAll(filepath.Join(repo, ".nullhooks"), 0o755)

	// Commit 1: two files.
	_ = os.WriteFile(filepath.Join(repo, "a.txt"), []byte("alpha\nbeta\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "b.txt"), []byte("gamma\n"), 0o644)
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-q", "--no-verify", "-m", "chore: initial files")

	// Commit 2: modify a.txt (adds a line → +/- stats, file-history, blame).
	_ = os.WriteFile(filepath.Join(repo, "a.txt"), []byte("alpha\nbeta\ndelta\n"), 0o644)
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-q", "--no-verify", "-m", "feat(a): add delta line")

	// Commit 3: modify b.txt.
	_ = os.WriteFile(filepath.Join(repo, "b.txt"), []byte("gamma\nepsilon\n"), 0o644)
	gitRun(t, repo, "add", "-A")
	gitRun(t, repo, "commit", "-q", "--no-verify", "-m", "feat(b): add epsilon line")

	t.Setenv("SKILLGRID_DOCS_CWD", repo)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	return NewServer(svc).Handler()
}

func TestPhase6_Git(t *testing.T) {
	h := newPhase6GitServer(t)

	// --- /git/commits?limit=50 → 3 commits, newest first ---
	rr := doGet(t, h, "/git/commits?limit=50")
	if rr.Code != http.StatusOK {
		t.Fatalf("/git/commits: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var cm map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &cm); err != nil {
		t.Fatalf("/git/commits unmarshal: %v", err)
	}
	commits, _ := cm["commits"].([]any)
	if len(commits) != 3 {
		t.Fatalf("/git/commits: got %d, want 3; body=%s", len(commits), rr.Body.String())
	}
	first := commits[0].(map[string]any)
	// newest = "feat(b): add epsilon line"
	if first["message"] != "feat(b): add epsilon line" {
		t.Errorf("commits[0].message = %v, want 'feat(b): add epsilon line' (newest first)", first["message"])
	}
	if first["sha"] == "" || first["sha"] == nil {
		t.Errorf("commits[0].sha empty")
	}
	if first["author"] == "" || first["author"] == nil {
		t.Errorf("commits[0].author empty")
	}
	// the feat(b) commit added 1 line to b.txt → additions >= 1
	if add, _ := first["additions"].(float64); add < 1 {
		t.Errorf("commits[0].additions = %v, want >= 1", first["additions"])
	}

	// --- limit honored ---
	rr = doGet(t, h, "/git/commits?limit=2")
	var lm map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &lm)
	if got := len(lm["commits"].([]any)); got != 2 {
		t.Errorf("/git/commits limit=2: got %d, want 2", got)
	}

	// --- /git/commits/{sha} + /git/diff/{sha} ---
	sha := first["sha"].(string)
	rr = doGet(t, h, "/git/commits/"+sha)
	if rr.Code != http.StatusOK {
		t.Fatalf("/git/commits/{sha}: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	rr = doGet(t, h, "/git/diff/"+sha)
	if rr.Code != http.StatusOK {
		t.Fatalf("/git/diff/{sha}: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var dm map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &dm)
	if d, _ := dm["diff"].(string); len(d) < 5 {
		t.Errorf("/git/diff/{sha}.diff too short: %q", d)
	}

	// --- /git/file-history?path=a.txt → 2 commits (initial + delta) ---
	rr = doGet(t, h, "/git/file-history?path=a.txt")
	if rr.Code != http.StatusOK {
		t.Fatalf("/git/file-history: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var fm map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &fm)
	fh, _ := fm["history"].([]any)
	if len(fh) != 2 {
		t.Errorf("/git/file-history a.txt: got %d, want 2; body=%s", len(fh), rr.Body.String())
	}

	// --- /git/blame?path=a.txt → lines blamed to the two commits ---
	rr = doGet(t, h, "/git/blame?path=a.txt")
	if rr.Code != http.StatusOK {
		t.Fatalf("/git/blame: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var bm map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &bm)
	blame, _ := bm["lines"].([]any)
	if len(blame) != 3 { // a.txt has 3 lines
		t.Errorf("/git/blame a.txt: got %d lines, want 3; body=%s", len(blame), rr.Body.String())
	}

	// --- unknown sha → 404; unknown file → 404 ---
	rr = doGet(t, h, "/git/diff/deadbeef")
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown sha: status = %d, want 404", rr.Code)
	}
	rr = doGet(t, h, "/git/file-history?path=nope.txt")
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown file: status = %d, want 404", rr.Code)
	}
}

// TestPhase6_GitNotRepo verifies a non-git sddRoot → 503.
func TestPhase6_GitNotRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo
	t.Setenv("SKILLGRID_DOCS_CWD", dir)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	rr := doGet(t, h, "/git/commits")
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("not-a-repo: status = %d, want 503; body=%s", rr.Code, rr.Body.String())
	}
}
