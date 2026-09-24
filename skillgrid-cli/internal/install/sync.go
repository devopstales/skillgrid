package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// SyncRepo copies the repository at srcPath into c.RepoDir, mirrors the
// whitelisted dirs (.agents, docs, git-hooks) to ~/.skillgrid/, and wires
// git hooks.
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
		for _, name := range mirrorDirs {
			if _, err := os.Stat(filepath.Join(abs, name)); err == nil {
				fmt.Fprintln(os.Stderr, "      [dry-run] mirror", filepath.Join(abs, name), "→", filepath.Join(c.RepoHome, name))
			}
		}
		if !c.SkipAgentsCopy {
			if _, err := os.Stat(filepath.Join(abs, ".agents")); err == nil {
				fmt.Fprintln(os.Stderr, "      [dry-run] copy", filepath.Join(abs, ".agents"), "→", c.AgentsDir)
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

	if err := syncMirrorDirs(c); err != nil {
		return err
	}

	if !c.SkipAgentsCopy {
		srcAgents := filepath.Join(abs, ".agents")
		if info, err := os.Stat(srcAgents); err == nil && info.IsDir() {
			if err := removeIfPresent(c.AgentsDir); err != nil {
				return err
			}
			if err := copyAll(srcAgents, c.AgentsDir); err != nil {
				return fmt.Errorf("copy .agents: %w", err)
			}
			fmt.Fprintln(os.Stderr, "      copied .agents/ → ~/.agents/")
		}
	} else {
		fmt.Fprintln(os.Stderr, "      skipped ~/.agents copy (--skip-agents)")
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
