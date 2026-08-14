package sync

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Result struct {
	Path string
	Rev  string
}

func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func ensureGitAvailable() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is required on PATH for sync")
	}
	return nil
}

// Sync clones repoURL into toolsDir or fast-forward pulls if already a clone.
func Sync(toolsDir, repoURL string) (Result, error) {
	if err := ensureGitAvailable(); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Dir(toolsDir), 0o755); err != nil {
		return Result{}, err
	}

	if isGitRepo(toolsDir) {
		status, err := runGit(toolsDir, "status", "--porcelain")
		if err != nil {
			return Result{}, err
		}
		if status != "" {
			return Result{}, fmt.Errorf("tools checkout has local changes; commit or discard them before sync:\n%s", status)
		}
		if _, err := runGit(toolsDir, "pull", "--ff-only"); err != nil {
			return Result{}, err
		}
	} else {
		if entries, err := os.ReadDir(toolsDir); err == nil && len(entries) > 0 {
			return Result{}, fmt.Errorf("tools directory %s exists and is not a git repo; remove it or empty it before sync", toolsDir)
		}
		_ = os.RemoveAll(toolsDir)
		parent := filepath.Dir(toolsDir)
		if _, err := runGit(parent, "clone", repoURL, toolsDir); err != nil {
			return Result{}, err
		}
	}

	rev, err := runGit(toolsDir, "rev-parse", "HEAD")
	if err != nil {
		return Result{}, err
	}
	return Result{Path: toolsDir, Rev: rev}, nil
}

func RevParse(toolsDir string) (string, error) {
	if !isGitRepo(toolsDir) {
		return "", fmt.Errorf("no synced repo at %s", toolsDir)
	}
	return runGit(toolsDir, "rev-parse", "--short", "HEAD")
}
