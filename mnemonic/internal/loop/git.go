package loop

import (
	"os/exec"
	"strings"
)

// GitBranch is the current branch name, or empty outside a repo.
func GitBranch(dir string) string {
	return gitOne(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

// ChangedFiles is the uncommitted plus untracked-not-ignored name list,
// then the files touched since HEAD~1 when the worktree is clean.
func ChangedFiles(dir string) []string {
	out := gitOne(dir, "diff", "--name-only", "HEAD")
	files := splitLines(out)
	if len(files) == 0 {
		files = splitLines(gitOne(dir, "diff", "--name-only", "HEAD~1", "HEAD"))
	}
	return files
}

// DiffStat is `git diff --stat` for the same range ChangedFiles uses.
func DiffStat(dir string) string {
	stat := strings.TrimSpace(gitOne(dir, "diff", "--stat", "HEAD"))
	if stat == "" {
		stat = strings.TrimSpace(gitOne(dir, "diff", "--stat", "HEAD~1", "HEAD"))
	}
	return stat
}

func gitOne(dir string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func splitLines(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
