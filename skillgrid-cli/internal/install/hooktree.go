package install

// Selective mirror: copies only the whitelisted directories from the repo
// checkout into ~/.skillgrid/. Everything else stays in
// ~/.skillgrid/repos/skillgrid/ and is accessed from there.
//
// Whitelisted dirs (remove-then-copy per dir, dry-run aware):
//
//	.agents/    → ~/.skillgrid/.agents/    (agent config reference)
//	docs/       → ~/.skillgrid/docs/       (documentation)
//	git-hooks/  → ~/.skillgrid/git-hooks/  (git hooks; core.hooksPath target)
//	hooks/      → ~/.skillgrid/hooks/      (hook implementations; shims resolve ../hooks/ relatively)
//
// Operational dirs under ~/.skillgrid/ (mnemonic, repos, backup, bin, tmp,
// logs, config.d) are never touched — they are not in the whitelist.

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// mirrorDirs are the repo-relative directories copied to ~/.skillgrid/.
var mirrorDirs = []string{".agents", "docs", "git-hooks", "hooks"}

// mirrorCopySkip reports whether rel (relative to the copied root) must be
// skipped: .git and node_modules at any depth.
func mirrorCopySkip(rel string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == ".git" || part == "node_modules" {
			return true
		}
	}
	return false
}

// mirrorProtectDst are destination names never deleted or overwritten —
// live operational state under ~/.skillgrid/.
var mirrorProtectDst = map[string]bool{
	"mnemonic": true, "repos": true, "backup": true, "bin": true,
	"tmp": true, "logs": true, "config.d": true,
}

// syncMirrorDirs copies each whitelisted dir from RepoDir → RepoHome, and
// removes stale non-whitelisted entries at the ~/.skillgrid/ top level that
// correspond to repo entries (leftover from a previous full-tree mirror).
// Missing source dirs are skipped (verbose log).
// Dry-run prints the plan and writes nothing.
func syncMirrorDirs(c *Config) error {
	whitelist := make(map[string]bool, len(mirrorDirs))
	for _, name := range mirrorDirs {
		whitelist[name] = true
	}

	// Remove stale top-level entries: present in RepoDir, not whitelisted,
	// not protected, and present in RepoHome.
	if entries, err := os.ReadDir(c.RepoDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if whitelist[name] || mirrorProtectDst[name] || name == ".git" || name == "node_modules" {
				continue
			}
			dst := filepath.Join(c.RepoHome, name)
			if _, err := os.Stat(dst); err != nil {
				continue
			}
			if c.DryRun {
				Out("      [dry-run] remove stale", dst)
				continue
			}
			if err := removeIfPresent(dst); err != nil {
				return err
			}
			Out("      removed stale", dst)
		}
	}

	// Copy each whitelisted dir.
	for _, name := range mirrorDirs {
		src := filepath.Join(c.RepoDir, name)
		dst := filepath.Join(c.RepoHome, name)
		if _, err := os.Stat(src); err != nil {
			VerboseOut(c, "mirror skip (source missing):", name)
			continue
		}
		if c.DryRun {
			Out("      [dry-run] mirror", src, "→", dst)
			continue
		}
		if err := removeIfPresent(dst); err != nil {
			return err
		}
		if err := copyAllFiltered(src, dst, mirrorCopySkip); err != nil {
			return err
		}
		Out("      mirrored", dst)
	}
	return nil
}

// copyAllFiltered recursively copies src to dst, skipping entries for which
// skip(rel) is true. Like copyAll, it preserves permission bits and symlinks.
func copyAllFiltered(src, dst string, skip func(rel string) bool) error {
	os.MkdirAll(dst, 0o755)
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel != "." && skip(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			os.MkdirAll(filepath.Dir(target), 0o755)
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			os.Remove(target)
			return os.Symlink(link, target)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		os.MkdirAll(filepath.Dir(target), 0o755)
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// wireGitHooks points git's global core.hooksPath at the staged git-hooks dir.
// It is a no-op when already wired. Git failures are returned; the caller
// decides whether they are fatal (Run treats them as warnings so a missing git
// binary cannot kill an otherwise good install).
func wireGitHooks(c *Config) error {
	target := filepath.Join(c.RepoHome, "git-hooks")
	if _, err := os.Stat(target); err != nil {
		return err
	}
	if _, err := exec.LookPath("git"); err != nil {
		return err
	}
	cur, err := gitConfigGet("core.hooksPath")
	if err == nil && cur == target {
		VerboseOut(c, "core.hooksPath already wired →", target)
		return nil
	}
	if c.DryRun {
		Out("      [dry-run] git config --global core.hooksPath", target)
		return nil
	}
	if err := gitConfigSet("core.hooksPath", target); err != nil {
		return err
	}
	Out("      wired core.hooksPath →", target)
	return nil
}

func gitConfigGet(key string) (string, error) {
	out, err := exec.Command("git", "config", "--global", key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitConfigSet(key, value string) error {
	cmd := exec.Command("git", "config", "--global", key, value)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// copyAsset copies a repo-relative asset to dst.
func copyAsset(repoRoot, relPath, dst string, dryRun bool) error {
	return copyFile(filepath.Join(repoRoot, relPath), dst, dryRun)
}

// copyFile copies a single file to dst, overwriting.
func copyFile(src, dst string, dryRun bool) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if dryRun {
		Out("      [dry-run] cp", src, dst)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return err
	}
	Out("      copied", dst)
	return nil
}
