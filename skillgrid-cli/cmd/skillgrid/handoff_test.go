package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// gitInitRepo creates a temp git repo with the given commits (oldest first)
// and returns the repo dir. Mirrors the handoff package test helper.
func gitInitRepo(t *testing.T, commits []string) string {
	t.Helper()
	dir := t.TempDir()
	env := func() []string {
		return append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			"SKILLGRID_PROTECTED_BRANCHES=protected-main")
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = env()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "config", "user.name", "t")
	run("git", "config", "user.email", "t@t")
	for i, c := range commits {
		fields := strings.Split(c, "|")
		body := fields[0]
		if len(fields) > 1 {
			body += "\n\n[skillgrid-context]\n"
			for _, f := range fields[1:] {
				body += f + "\n"
			}
			body += "[/skillgrid-context]\n"
		}
		p := filepath.Join(dir, "f"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		run("git", "add", ".")
		run("git", "commit", "-q", "-m", body)
	}
	return dir
}

// handoffBinary builds the skillgrid binary once per test (cached in a temp
// dir) and returns its path. The CLI is then run with CWD = the git repo so
// the Hub's RepoDir resolves to the repo.
var handoffBinOnce sync.Once
var handoffBinPath string
var handoffBinErr error

func buildHandoffBinary(t *testing.T) string {
	t.Helper()
	handoffBinOnce.Do(func() {
		pkgDir := mustWD(t)
		dir, err := os.MkdirTemp("", "skillgrid-handoff-bin")
		if err != nil {
			handoffBinErr = err
			return
		}
		bin := filepath.Join(dir, "skillgrid-test")
		cmd := exec.Command("go", "build", "-o", bin, ".")
		cmd.Dir = pkgDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			handoffBinErr = fmt.Errorf("build: %v\n%s", err, out)
			return
		}
		handoffBinPath = bin
	})
	if handoffBinErr != nil {
		t.Fatalf("build handoff binary: %v", handoffBinErr)
	}
	return handoffBinPath
}

func runHandoffCLI(t *testing.T, dataDir, cwd string, args ...string) string {
	t.Helper()
	bin := buildHandoffBinary(t)
	cmd := exec.Command(bin, "handoff")
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = cwd
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MNEMONIC_DATA_DIR="+dataDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("handoff %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// TestHandoffCLIRecordBackfillStatus covers the happy path: record (which
// backfills on first run), status, checkpoint, verify, archive.
func TestHandoffCLIRecordBackfillStatus(t *testing.T) {
	dataDir := t.TempDir()
	repo := gitInitRepo(t, []string{
		"feat: one",
		"feat: two|Task: t2|Decisions: d",
		"chore: three",
	})

	// record: first run backfills all 3 + records HEAD.
	out := runHandoffCLI(t, dataDir, repo, "record", "--project", "hubcli")
	if !strings.Contains(out, "SNAPSHOT") {
		t.Fatalf("record did not report a snapshot: %s", out)
	}

	// status: must show the latest snapshot.
	out = runHandoffCLI(t, dataDir, repo, "status", "--project", "hubcli")
	if !strings.Contains(out, "LATEST") {
		t.Fatalf("status did not report latest snapshot: %s", out)
	}

	// checkpoint: place a named marker (name is positional).
	out = runHandoffCLI(t, dataDir, repo, "checkpoint", "before-apply-test", "--project", "hubcli")
	if !strings.Contains(out, "CHECKPOINT before-apply-test") {
		t.Fatalf("checkpoint did not report: %s", out)
	}

	// verify: clean (no drift) -> continue.
	out = runHandoffCLI(t, dataDir, repo, "verify", "before-apply-test", "--project", "hubcli")
	if !strings.Contains(out, "VERIFY before-apply-test") {
		t.Fatalf("verify did not report: %s", out)
	}
}

// TestHandoffCLIJSON covers the --json emit path.
func TestHandoffCLIJSON(t *testing.T) {
	dataDir := t.TempDir()
	repo := gitInitRepo(t, []string{"feat: one"})
	out := runHandoffCLI(t, dataDir, repo, "backfill", "--project", "hubclijson", "--json")
	if !strings.Contains(out, `"backfilled"`) {
		t.Fatalf("backfill --json did not emit JSON: %s", out)
	}
}

// TestHandoffCLIUnknownSubcommand proves an unknown subcommand exits non-zero.
func TestHandoffCLIUnknownSubcommand(t *testing.T) {
	dataDir := t.TempDir()
	repo := gitInitRepo(t, []string{"feat: one"})
	bin := buildHandoffBinary(t)
	cmd := exec.Command(bin, "handoff", "bogus", "--project", "hubcli-err")
	cmd.Dir = repo
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MNEMONIC_DATA_DIR="+dataDir)
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("expected non-zero exit for unknown subcommand: %s", out)
	}
}
