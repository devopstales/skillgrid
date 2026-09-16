package install

import (
	"os"
	"path/filepath"
)

// personaTargetDir maps an agent key to the harness's global agent directory.
// Kilo is "agent" (singular); opencode/cursor are "agents" (plural) — that
// asymmetry is per-harness, do not normalize. ok is false for an agent with no
// persona target.
func personaTargetDir(home, agentKey string) (string, bool) {
	switch agentKey {
	case "opencode":
		return filepath.Join(home, ".config", "opencode", "agents"), true
	case "kilo":
		return filepath.Join(home, ".config", "kilo", "agent"), true
	case "cursor":
		return filepath.Join(home, ".cursor", "agents"), true
	default:
		return "", false
	}
}

// installPersonas copies the harness's persona files from the synced repo into
// the harness's global agent directory, flattened.
//
// The source is ALWAYS the synced repo at c.RepoDir/agents/<agentKey>/
// (default: ~/.skillgrid/repos/skillgrid/agents/<agentKey>/), never the user's
// working checkout. This makes persona distribution deterministic: it ships
// exactly what is on the DefaultBranch, regardless of how the install was
// invoked (--sync-repo, --skip-clone, or a fresh clone). c.RepoDir is the
// synced clone for the default path and the sync target for --sync-repo, so in
// both cases it points at the canonical copy.
//
// Missing source is a no-op (verbose log), not an error — personas are optional
// and a partial repo sync must not fail the install.
func installPersonas(c *Config, agentKey string) error {
	home, err := configHome(c)
	if err != nil {
		return err
	}
	dstDir, ok := personaTargetDir(home, agentKey)
	if !ok {
		return nil // agent without a persona target (future-proof)
	}
	src := filepath.Join(c.RepoDir, "agents", agentKey)
	if _, err := os.Stat(src); err != nil {
		VerboseOut(c, "no agents/"+agentKey+"/ in synced repo — skipping persona install")
		return nil
	}
	return copyFlat(src, dstDir, c.DryRun)
}

// copyFlat walks srcDir and writes each top-level regular file to dstDir/<name>,
// creating dstDir as needed. Subdirectories under srcDir are skipped — only the
// per-harness .md files are distributed. This differs from copyAll, which
// preserves the nested tree.
func copyFlat(srcDir, dstDir string, dryRun bool) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(dstDir, name)
		if dryRun {
			Out("      [dry-run] cp", src, dst)
			continue
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
			return err
		}
		Out("      copied", dst)
	}
	return nil
}
