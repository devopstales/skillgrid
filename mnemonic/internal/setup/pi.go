package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SetupPi registers Mnemonic for Pi (skill-pi, ADR-0043): the native TS
// extension folder plugins/pi/mnemonic/ is copied as a directory into the
// agent dir (<agent-dir>/extensions/mnemonic/, default
// ~/.skill-pi/agent/extensions/). The skill-pi launcher explicitly
// enumerates <agent-dir>/extensions subdirectories under --no-extensions
// isolation and resolves each one's entry point with pi's own convention
// (package.json "pi.extensions" manifest, else index.ts, else index.js), so
// no settings.json edit is needed.
//
// Environment for the running Pi process (documented, not written):
//   SKILLGRID_MNEMONIC_HTTP_URL — base URL of `mnemonic serve`
//   SKILLGRID_HTTP_TOKEN        — bearer token when the server sets one
//   SKILLGRID_AGENT             — agent label (the extension defaults to "pi")
func SetupPi(home, repoRoot string, dryRun bool) error {
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}
	agentDir, err := piAgentDir(home)
	if err != nil {
		return err
	}
	srcDir := filepath.Join(repoRoot, "plugins", "pi", "mnemonic")
	if staged := stagedPluginPath("pi/mnemonic"); staged != "" {
		srcDir = staged
	}
	if _, err := os.Stat(filepath.Join(srcDir, "index.ts")); err != nil {
		return fmt.Errorf("pi extension not found at %s (sync repo or run from checkout)", srcDir)
	}
	dstDir := filepath.Join(agentDir, "extensions", "mnemonic")
	for _, name := range piExtensionFiles {
		src := filepath.Join(srcDir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read pi extension file %s: %w", name, err)
		}
		dst := filepath.Join(dstDir, name)
		if dryRun {
			logInfo("[dry-run] cp " + src + " " + dst)
			continue
		}
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}
	logInfo("installed pi extension " + dstDir)
	return nil
}

// piExtensionFiles are the extension folder's source files, in copy order.
// package.json carries the "pi.extensions" manifest the launcher's
// resolveDirEntryPoints reads; index.ts is the entry point.
var piExtensionFiles = []string{
	"package.json",
	"index.ts",
	"api.js",
	"mnemonic-memory.ts",
	"mnemonic-facts.ts",
	"mnemonic-codeindex.ts",
	"ctx-sandbox.ts",
	"skillgrid-events.ts",
	"skillgrid-compaction.ts",
}

// piExtensionEntry reports whether a copied extension dir resolves to an
// entry point under the same convention skill-pi's launcher uses
// (manifest "pi.extensions" paths that exist, else index.ts, else index.js).
func piExtensionEntry(dir string) string {
	if manifestPath := filepath.Join(dir, "package.json"); fileExists(manifestPath) {
		if data, err := os.ReadFile(manifestPath); err == nil {
			var manifest struct {
				Pi struct {
					Extensions []string `json:"extensions"`
				} `json:"pi"`
			}
			if json.Unmarshal(data, &manifest) == nil {
				for _, p := range manifest.Pi.Extensions {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					if fileExists(filepath.Join(dir, p)) {
						return p
					}
				}
			}
		}
	}
	for _, name := range []string{"index.ts", "index.js"} {
		if fileExists(filepath.Join(dir, name)) {
			return name
		}
	}
	return ""
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// piAgentDir returns the skill-pi agent dir, honoring SKILL_PI_AGENT_DIR
// (skill-pi's lib/config.js resolution) and defaulting to ~/.skill-pi/agent.
func piAgentDir(home string) (string, error) {
	if v := os.Getenv("SKILL_PI_AGENT_DIR"); v != "" {
		return v, nil
	}
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(home, ".skill-pi", "agent"), nil
}
