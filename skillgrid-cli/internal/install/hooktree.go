package install

// Full-tree mirror ($REPO/* recursively → ~/.skillgrid/).
//
// The repo checkout is the source of truth. `skillgrid install` mirrors the
// whole repo tree into ~/.skillgrid/ (remove-then-copy per top-level entry,
// dry-run aware) and points git's global core.hooksPath at the mirrored
// git-hooks. Consumers then read from the mirror:
//
//	plugins from ~/.skillgrid/plugins,
//	git hooks from ~/.skillgrid/git-hooks (shims resolve ../hooks/ relatively,
//	so they work identically from the checkout and the mirror),
//	implementations via copy from ~/.skillgrid/hooks.
//
// Safety exclusions (documented, not silent):
//
//	mirrorSkipAny   — never copied, at any depth: .git, node_modules
//	mirrorSkipTop   — never copied (repo worktree state, not content):
//	                  .skillgrid (the repo's own project state would nest as
//	                  ~/.skillgrid/.skillgrid), dist, out (build outputs)
//	mirrorProtectDst — never created, deleted, or overwritten at the
//	                  destination (live operational state): mnemonic (SQLite
//	                  data), repos (checkouts), backup, bin, tmp, logs,
//	                  config.d (user config)

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// mirrorSkipAny are entry names never copied, at any depth.
var mirrorSkipAny = map[string]bool{".git": true, "node_modules": true}

// mirrorSkipTop are top-level entries never copied (worktree state, not content).
// .git and node_modules are also in mirrorSkipAny so nested copies skip them.
var mirrorSkipTop = map[string]bool{
	".git": true, "node_modules": true,
	".skillgrid": true, "dist": true, "out": true,
}

// mirrorProtectDst are destination names never created, deleted, or overwritten.
var mirrorProtectDst = map[string]bool{
	"mnemonic": true, "repos": true, "backup": true, "bin": true,
	"tmp": true, "logs": true, "config.d": true,
}

// mirrorOp is one planned top-level mirror action.
type mirrorOp struct {
	name   string // top-level entry name
	action string // "copy" | "skip-source" | "protect-dst"
}

// planMirror classifies every top-level entry of srcDir without writing.
func planMirror(srcDir string) ([]mirrorOp, error) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, err
	}
	var ops []mirrorOp
	for _, e := range entries {
		name := e.Name()
		switch {
		case mirrorSkipTop[name]:
			ops = append(ops, mirrorOp{name, "skip-source"})
		case mirrorProtectDst[name]:
			ops = append(ops, mirrorOp{name, "protect-dst"})
		default:
			ops = append(ops, mirrorOp{name, "copy"})
		}
	}
	return ops, nil
}

// syncFullTree mirrors RepoDir/* → RepoHome/* per planMirror. Protected
// destinations are left untouched; destinations for "copy" ops are replaced so
// a re-run converges (stale files from a previous mirror disappear).
// Dry-run prints the plan and writes nothing.
func syncFullTree(c *Config) error {
	ops, err := planMirror(c.RepoDir)
	if err != nil {
		return err
	}
	for _, op := range ops {
		src := filepath.Join(c.RepoDir, op.name)
		dst := filepath.Join(c.RepoHome, op.name)
		switch op.action {
		case "skip-source":
			VerboseOut(c, "mirror skip (worktree state):", op.name)
		case "protect-dst":
			VerboseOut(c, "mirror protect (operational):", op.name)
		default:
			if c.DryRun {
				Out("      [dry-run] mirror", src, "→", dst)
				continue
			}
			if err := removeIfPresent(dst); err != nil {
				return err
			}
			info, err := os.Stat(src)
			if err != nil {
				return err
			}
			if info.IsDir() {
				if err := copyAllFiltered(src, dst, mirrorCopySkip); err != nil {
					return err
				}
			} else if err := copyFile(src, dst, false); err != nil {
				return err
			}
			Out("      mirrored", dst)
		}
	}
	return nil
}

// mirrorCopySkip reports whether rel (relative to the copied root) must be
// skipped: .git and node_modules at any depth.
func mirrorCopySkip(rel string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if mirrorSkipAny[part] {
			return true
		}
	}
	return false
}

// copyAllFiltered recursively copies src to dst, skipping entries for which
// skip(rel) is true. Like copyAll, it preserves permission bits.
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

// resolveAssetPath prefers the staged copy under ~/.skillgrid for assets that
// live under plugins/ in the repo, falling back to the checkout. Non-plugin
// relatives always resolve under repoRoot.
func resolveAssetPath(home, repoRoot, rel string) string {
	if home != "" && strings.HasPrefix(rel, "plugins/") {
		staged := filepath.Join(home, ".skillgrid", "plugins", strings.TrimPrefix(rel, "plugins/"))
		if _, err := os.Stat(staged); err == nil {
			return staged
		}
	}
	return filepath.Join(repoRoot, rel)
}

// copyAsset copies a repo-relative asset to dst, preferring the staged copy
// under ~/.skillgrid/plugins when present (see resolveAssetPath).
func copyAsset(home, repoRoot, relPath, dst string, dryRun bool) error {
	return copyFile(resolveAssetPath(home, repoRoot, relPath), dst, dryRun)
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
