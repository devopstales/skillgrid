package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SyncRepo copies the repository at srcPath into c.RepoDir and overwrites
// ~/.agents from srcPath/.agents (if present).
//
// Use this when the user already has a local clone of the skillgrid repo
// and wants to install it without a network git clone.
func (c *Config) SyncRepo(srcPath string) error {
	abs, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("source path not found: %s", srcPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", srcPath)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "  syncing repo %s → %s\n", abs, c.RepoDir)

	if err := ensureHomeStruct(c); err != nil {
		return err
	}

	if c.DryRun {
		fmt.Fprintln(os.Stderr, "      [dry-run] copy", abs, "→", c.RepoDir)
		if _, err := os.Stat(filepath.Join(abs, ".agents")); err == nil {
			fmt.Fprintln(os.Stderr, "      [dry-run] copy", filepath.Join(abs, ".agents"), "→", c.AgentsDir)
		}
		if ops, err := planMirror(abs); err == nil {
			for _, op := range ops {
				fmt.Fprintln(os.Stderr, "      [dry-run] mirror", op.action, filepath.Join(abs, op.name), "→", filepath.Join(c.RepoHome, op.name))
			}
		}
		fmt.Fprintln(os.Stderr, "      [dry-run] git config --global core.hooksPath", filepath.Join(c.RepoHome, "git-hooks"))
		fmt.Fprintln(os.Stderr, "      no changes were written (dry run)")
		return nil
	}

	if err := removeIfPresent(c.RepoDir); err != nil {
		return err
	}
	if err := copyAll(abs, c.RepoDir); err != nil {
		return fmt.Errorf("copy repo: %w", err)
	}
	fmt.Fprintln(os.Stderr, "      copied repo")

	srcAgents := filepath.Join(abs, ".agents")
	if info, err := os.Stat(srcAgents); err == nil && info.IsDir() && !c.SkipAgentsCopy {
		if err := removeIfPresent(c.AgentsDir); err != nil {
			return err
		}
		if err := copyAll(srcAgents, c.AgentsDir); err != nil {
			return fmt.Errorf("copy .agents: %w", err)
		}
		fmt.Fprintln(os.Stderr, "      copied .agents/ → ~/.agents/")
	} else if c.SkipAgentsCopy {
		fmt.Fprintln(os.Stderr, "      skipped ~/.agents copy (--skip-agents)")
	}

	// Mirror the checkout next to itself so consumers read from
	// ~/.skillgrid/{hooks,git-hooks,plugins,...}.
	if err := syncFullTree(c); err != nil {
		return err
	}
	if err := wireGitHooks(c); err != nil {
		fmt.Fprintln(os.Stderr, "  warning: git hooks wiring:", err)
	}

	fmt.Fprintln(os.Stderr, "\n  done")
	return nil
}

func removeIfPresent(path string) error {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	fmt.Fprintln(os.Stderr, "      removing", path)
	return os.RemoveAll(path)
}

var _ = strings.TrimSpace
