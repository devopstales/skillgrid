package sync

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSyncCloneAndPull(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	remote := t.TempDir()
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run(remote, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(remote, "README"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(remote, "add", ".")
	run(remote, "commit", "--no-verify", "-m", "chore: init fixture")

	home := t.TempDir()
	tools := filepath.Join(home, "tools")
	res, err := Sync(tools, remote)
	if err != nil {
		t.Fatal(err)
	}
	if res.Rev == "" {
		t.Fatal("empty rev")
	}
	if _, err := os.Stat(filepath.Join(tools, "README")); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(remote, "README"), []byte("hi2"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(remote, "add", ".")
	run(remote, "commit", "--no-verify", "-m", "chore: update fixture")

	res2, err := Sync(tools, remote)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Rev == res.Rev {
		t.Fatal("expected new rev after pull")
	}
}

func TestSyncRejectsDirty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	remote := t.TempDir()
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = remote
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	_ = os.WriteFile(filepath.Join(remote, "f"), []byte("1"), 0o644)
	for _, args := range [][]string{
		{"add", "."},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "--no-verify", "-m", "chore: init"},
	} {
		c := exec.Command("git", args...)
		c.Dir = remote
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	tools := filepath.Join(t.TempDir(), "tools")
	if _, err := Sync(tools, remote); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tools, "f"), []byte("dirty"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(tools, remote); err == nil {
		t.Fatal("expected dirty error")
	}
}
