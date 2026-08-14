package githooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	managedMarker = "aiskillgrid-managed: commit-msg"
	prevSuffix    = ".aiskillgrid-prev"
)

// InstallCommitMsgHook installs packs/git-hooks/commit-msg into the project's
// .git/hooks/commit-msg. If a non-Skillgrid commit-msg already exists, it is
// preserved as commit-msg.aiskillgrid-prev and chained after the stripper.
//
// No-op (nil error) when projectDir is not a git work tree.
func InstallCommitMsgHook(projectDir, packRoot string) (written string, err error) {
	gitDir, err := resolveGitDir(projectDir)
	if err != nil {
		return "", nil // not a git repo — skip
	}
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return "", err
	}

	src := filepath.Join(packRoot, "packs", "git-hooks", "commit-msg")
	body, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("read hook pack: %w", err)
	}

	dst := filepath.Join(hooksDir, "commit-msg")
	prev := dst + prevSuffix

	if existing, err := os.ReadFile(dst); err == nil {
		if !strings.Contains(string(existing), managedMarker) {
			// Preserve user/project hook once (do not overwrite a prior backup).
			if _, err := os.Stat(prev); os.IsNotExist(err) {
				if err := os.WriteFile(prev, existing, 0o755); err != nil {
					return "", fmt.Errorf("backup existing commit-msg: %w", err)
				}
				_ = os.Chmod(prev, 0o755)
			}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := os.WriteFile(dst, body, 0o755); err != nil {
		return "", err
	}
	if err := os.Chmod(dst, 0o755); err != nil {
		return "", err
	}
	return dst, nil
}

func resolveGitDir(projectDir string) (string, error) {
	// Prefer .git directory; skip bare or unusual layouts for v1.
	gitPath := filepath.Join(projectDir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return gitPath, nil
	}
	// Worktree: .git is a file pointing at gitdir
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(data))
	const prefix = "gitdir: "
	if !strings.HasPrefix(line, prefix) {
		return "", fmt.Errorf("unsupported .git file")
	}
	gitDir := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(projectDir, gitDir)
	}
	return gitDir, nil
}
