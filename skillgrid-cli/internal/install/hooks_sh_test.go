package install

// Shell tests for the guard hooks under the repo's hooks/ directory. The JS
// hooks are the live implementation (checkpoint-state.js dispatches to them and
// `skillgrid install` mirrors hooks/ to ~/.skillgrid/hooks/), so their behavior
// is exercised end-to-end here: build a throwaway git repo and run the real
// hook scripts against it. Skips cleanly when node or go is unavailable.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoHooksDir walks up from this package to the skillgrid repo's hooks/ dir.
func repoHooksDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	for i := 0; i < 8 && dir != ""; i++ {
		if fi, err := os.Stat(filepath.Join(dir, "hooks", "checkpoint-state.js")); err == nil && fi.Mode().IsRegular() {
			return filepath.Join(dir, "hooks")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate repo hooks/ dir above %s", dir)
	return ""
}

func requireTools(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("skipping: %s not on PATH", tool)
		}
	}
}

// newHookRepo builds a temp git repo with an internal/ module and returns its
// root plus the hook script path.
func newHookRepo(t *testing.T, hooksDir string) (string, string) {
	t.Helper()
	root := t.TempDir()
	gm := filepath.Join(root, "go.mod")
	if err := os.WriteFile(gm, []byte("module testmod\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	clean := []string{
		"package main",
		"func main() {}",
		"",
	}
	okPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(okPath, []byte(strings.Join(clean, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@skillgrid.local"},
		{"config", "user.name", "Skillgrid Test"},
		{"add", "-A"},
		{"commit", "-q", "-m", "chore: init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return root, filepath.Join(hooksDir, "precommit-go-vet.js")
}

// runHook runs the go-vet hook from the repo root and returns (exitCode, stderr).
func runHook(t *testing.T, root, hook string) (int, string) {
	t.Helper()
	cmd := exec.Command("node", hook)
	cmd.Dir = root
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("running hook: %v", err)
		}
	}
	return code, errb.String()
}

// gitIn runs a git command confined to dir.
func gitIn(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd
}

func TestGoVetHookSkipsWhenNoGoStaged(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	root, hook := newHookRepo(t, hooksDir)
	// Nothing new staged: the hook must pass.
	code, _ := runHook(t, root, hook)
	if code != 0 {
		t.Fatalf("expected exit 0 with nothing staged, got %d", code)
	}
}

func TestGoVetHookPassesCleanGoFile(t *testing.T) {
	requireTools(t, "node", "git")
	hooksDir := repoHooksDir(t)
	root, hook := newHookRepo(t, hooksDir)
	clean := []string{
		"package internal",
		"",
		"func Clean() int { return 1 }",
		"",
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "x.go"), []byte(strings.Join(clean, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := gitIn(root, "add", "internal/x.go").CombinedOutput(); err != nil {
		t.Fatalf("git add internal/x.go: %v: %s", err, out)
	}
	code, stderr := runHook(t, root, hook)
	if code != 0 {
		t.Fatalf("expected exit 0 for a clean staged Go file, got %d: %s", code, stderr)
	}
}

func TestGoVetHookFailsOnVetError(t *testing.T) {
	requireTools(t, "node", "git", "go")
	hooksDir := repoHooksDir(t)
	root, hook := newHookRepo(t, hooksDir)
	bad := []string{
		"package main",
		"",
		"import \"fmt\"",
		"",
		"func probe() { fmt.Printf(\"%d\", 1, 2) }",
		"",
	}
	if err := os.WriteFile(filepath.Join(root, "bad.go"), []byte(strings.Join(bad, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := gitIn(root, "add", "bad.go").CombinedOutput(); err != nil {
		t.Fatalf("git add bad.go: %v: %s", err, out)
	}
	code, stderr := runHook(t, root, hook)
	if code != 1 {
		t.Fatalf("expected exit 1 for a staged vet error, got %d: %s", code, stderr)
	}
	if !strings.Contains(stderr, "go vet failed") {
		t.Errorf("expected 'go vet failed' in stderr, got: %s", stderr)
	}
}
