package install

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// LSP server installation + harness configuration (--lsp, opt-in).
//
// The catalog ships the language servers the mnemonic --lsp indexing tier and
// the agent harnesses (opencode, skill-pi) consume. Bin must stay in sync
// with mnemonic/internal/pdg defaultServers — TestLSPTargetsBinsMatchPDGServers
// guards the drift.

// LSPTool describes one LSP server the installer can bring up. It follows the
// SecTool contract: skip if Bin is on PATH, warn+skip when Manager is missing,
// best-effort (one failure does not abort the run).
type LSPTool struct {
	Name    string
	Manager string // "go", "npm", or "cargo"
	// InstallArgs are the exact arguments passed to the manager, after its name.
	InstallArgs []string
	// Bin is the resulting PATH binary; when present and found, the install is skipped.
	Bin string
	// Hint is the manual fallback shown when the manager is missing from PATH.
	Hint string
}

// LSPTargets returns the LSP servers installed with --lsp, in install order.
//
// The typescript entry installs both the runtime and the server in one npm
// command (typescript is a peer of typescript-language-server). Its Bin is
// tsserver (the runtime) so a machine with either half already skips: the
// all-or-nothing pair never half-installs.
func LSPTargets() []LSPTool {
	return []LSPTool{
		{
			Name:        "gopls",
			Manager:     "go",
			InstallArgs: []string{"install", "golang.org/x/tools/gopls@latest"},
			Bin:         "gopls",
			Hint:        "install Go, then: go install golang.org/x/tools/gopls@latest",
		},
		{
			Name:        "pyright",
			Manager:     "npm",
			InstallArgs: []string{"install", "-g", "pyright"},
			Bin:         "pyright-langserver",
			Hint:        "npm install -g pyright",
		},
		{
			Name:        "cclsp",
			Manager:     "npm",
			InstallArgs: []string{"install", "-g", "cclsp"},
			Bin:         "cclsp",
			Hint:        "npm install -g cclsp (bundles typescript-language-server; add `npm i -g typescript` for the peer)",
		},
		{
			Name:        "typescript-language-server",
			Manager:     "npm",
			InstallArgs: []string{"install", "-g", "typescript", "typescript-language-server"},
			Bin:         "typescript-language-server",
			Hint:        "npm install -g typescript typescript-language-server",
		},
		{
			Name:        "rust-analyzer",
			Manager:     "cargo",
			InstallArgs: []string{"install", "rust-analyzer"},
			Bin:         "rust-analyzer",
			Hint:        "install the Rust toolchain (https://rustup.rs), then: cargo install rust-analyzer",
		},
		{
			Name:        "jj-language-server",
			Manager:     "npm",
			InstallArgs: []string{"install", "-g", "jj-language-server"},
			Bin:         "jj-language-server",
			Hint:        "npm install -g jj-language-server",
		},
	}
}

// lspLookPath is exec.LookPath, replaced in tests.
var lspLookPath = exec.LookPath

// installLSPTools brings up the LSP servers (LSPTargets) with their own
// managers. Non-fatal per server: a missing manager warns and skips its tool
// with the manual fallback (same contract as installSecurityTools); a failing
// install of one server does not abort the rest.
func installLSPTools(c *Config) error {
	for _, t := range LSPTargets() {
		if t.Bin != "" {
			if _, err := lspLookPath(t.Bin); err == nil {
				Out("      skip", t.Name, "(already installed)")
				continue
			}
		}
		if _, err := lspLookPath(t.Manager); err != nil {
			Out("      warn", t.Name, "skipped —", t.Manager, "not on PATH; "+t.Hint)
			continue
		}
		if c.DryRun {
			Out(append([]any{"      [dry-run]", t.Manager}, toAny(t.InstallArgs)...)...)
			continue
		}
		Out(append([]any{"      ", t.Manager}, toAny(t.InstallArgs)...)...)
		if err := run(c, "", t.Manager, t.InstallArgs...); err != nil {
			return fmt.Errorf("%s %s: %w", t.Manager, strings.Join(t.InstallArgs, " "), err)
		}
	}
	return nil
}

// --- harness configuration ---

// LSPEntry is one language-server registration for a harness config.
type LSPEntry struct {
	ID     string
	Name   string
	Server string // command binary on PATH
	Args   []string
	// Extensions are the file extensions the server handles (dot-prefixed).
	Extensions []string
	// Languages / RootMarkers / FileGlobs are skill-pi lsp.conf routing fields.
	Languages   []string
	FileGlobs   []string
	RootMarkers []string
}

// lspServerEntries returns the per-language entries written into the harness
// configs. One entry per language the installer ships a server for.
func lspServerEntries() []LSPEntry {
	return []LSPEntry{
		{
			ID: "go", Name: "Go", Server: "gopls", Args: []string{"--stdio"},
			Extensions:  []string{".go"},
			Languages:   []string{"go"},
			RootMarkers: []string{"go.mod"},
		},
		{
			ID: "python", Name: "Python", Server: "pyright-langserver", Args: []string{"--stdio"},
			Extensions:  []string{".py", ".pyi"},
			Languages:   []string{"python"},
			RootMarkers: []string{"pyproject.toml", "setup.py"},
		},
		{
			ID: "typescript", Name: "TypeScript", Server: "typescript-language-server", Args: []string{"--stdio"},
			Extensions:  []string{".ts", ".tsx", ".js", ".jsx"},
			Languages:   []string{"typescript", "javascript"},
			RootMarkers: []string{"package.json", "tsconfig.json"},
		},
		{
			ID: "rust", Name: "Rust", Server: "rust-analyzer", Args: []string{},
			Extensions:  []string{".rs"},
			Languages:   []string{"rust"},
			RootMarkers: []string{"Cargo.toml"},
		},
		{
			ID: "java", Name: "Java", Server: "jj-language-server", Args: []string{"--stdio"},
			Extensions:  []string{".java"},
			Languages:   []string{"java"},
			RootMarkers: []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle"},
		},
	}
}

// configureLSP writes the LSP registrations into the selected agent's harness
// config (opencode: the "lsp" object in opencode.jsonc; skill-pi:
// conf.d/lsp.conf + the cclsp MCP package). It runs after installLSPTools so
// the binaries it references exist. It is best-effort: a binary the install
// step could not bring up (e.g. no cargo) still gets configured, with a warn
// telling the user to re-run after installing the manager.
func configureLSP(c *Config) error {
	for _, key := range c.Agents {
		switch key {
		case "opencode":
			if err := configureLSPOpenCode(c); err != nil {
				return fmt.Errorf("opencode lsp config: %w", err)
			}
		case "skill-pi":
			if err := configureLSPPi(c); err != nil {
				return fmt.Errorf("skill-pi lsp config: %w", err)
			}
		}
	}
	return nil
}

// configureLSPOpenCode upserts the "lsp" object into ~/.config/opencode/
// opencode.jsonc. Existing server entries and every other config key are
// preserved (raw merge of the "lsp" object); an entry whose "disabled" is
// already true is left untouched (explicit user opt-out wins).
func configureLSPOpenCode(c *Config) error {
	home, err := configHome(c)
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	if err := ensureConfigFile(cfgPath, c.DryRun); err != nil {
		return err
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	merged, changed := mergeOpenCodeLSP(data, c.DryRun, cfgPath)
	if !changed {
		return nil
	}
	if c.DryRun {
		return nil
	}
	return os.WriteFile(cfgPath, merged, 0o644)
}

// mergeOpenCodeLSP returns the config bytes with the "lsp" object merged in
// (existing entries preserved, catalog entries added/replaced, user-disabled
// entries skipped) and whether anything changed. The whole config is
// decoded/re-encoded via json (the harness accepts JSON; JSONC comments are
// not expected in a machine-managed config).
func mergeOpenCodeLSP(data []byte, dryRun bool, cfgPath string) ([]byte, bool) {
	changed := false
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		// A malformed config is not ours to heal: fail the step.
		return nil, false
	}
	var lsp map[string]json.RawMessage
	if raw, ok := root["lsp"]; ok {
		if err := json.Unmarshal(raw, &lsp); err != nil || lsp == nil {
			// Present but not an object (e.g. true): replace with the merged object.
			lsp = map[string]json.RawMessage{}
		}
	} else {
		lsp = map[string]json.RawMessage{}
	}
	for _, e := range lspServerEntries() {
		if raw, ok := lsp[e.ID]; ok {
			var entry struct {
				Disabled bool `json:"disabled"`
			}
			if err := json.Unmarshal(raw, &entry); err == nil && entry.Disabled {
				Out("      skip", e.ID, "(disabled in", cfgPath+")")
				continue // user opt-out wins
			}
		}
		b, merr := json.Marshal(map[string]any{
			"command":    append([]string{e.Server}, e.Args...),
			"extensions": e.Extensions,
		})
		if merr != nil {
			panic(merr)
		}
		lsp[e.ID] = b
		changed = true
	}
	merged, merr := json.Marshal(lsp)
	if merr != nil {
		panic(merr)
	}
	root["lsp"] = merged
	out, uerr := json.Marshal(root)
	if uerr != nil {
		panic(uerr)
	}
	return out, changed
}

// configureLSPPi upserts the lsp.conf servers into the user-scope skill-pi
// config (~/.skill-pi/conf.d/lsp.conf, ADR 0008) and appends the cclsp MCP
// package to the agent settings. Existing servers are matched by "id": a
// skillgrid id is replaced, foreign ids are preserved.
func configureLSPPi(c *Config) error {
	home, err := configHome(c)
	if err != nil {
		return err
	}
	confPath, err := piLSPConfPath(home)
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".skill-pi", "agent", "settings.json")
	return configureLSPPiFiles(c, confPath, settingsPath)
}

// piLSPConfPath resolves the user-scope lsp.conf (dirname(agent-dir)/conf.d/
// lsp.conf, mirroring skill-pi's own resolution; SKILL_PI_AGENT_DIR honored).
func piLSPConfPath(home string) (string, error) {
	agentDir := filepath.Join(home, ".skill-pi", "agent")
	if v := os.Getenv("SKILL_PI_AGENT_DIR"); v != "" {
		agentDir = v
	}
	return filepath.Join(filepath.Dir(agentDir), "conf.d", "lsp.conf"), nil
}

// configureLSPPiFiles is the testable core: confPath/settingsPath injected.
func configureLSPPiFiles(c *Config, confPath, settingsPath string) error {
	var raw []byte
	if data, err := os.ReadFile(confPath); err == nil {
		raw = data
	} else {
		raw = []byte("{}")
	}
	updated, err := sjson.SetRawBytes(raw, "servers", []byte(piLSPServers(raw)))
	if err != nil {
		return fmt.Errorf("set servers: %w", err)
	}
	if c.DryRun {
		Out("      [dry-run] write", confPath)
	} else {
		if err := writeJSONFile(confPath, updated); err != nil {
			return err
		}
		Out("      wrote", confPath)
	}
	if err := upsertPiCCLSPPackage(c, settingsPath); err != nil {
		return err
	}
	return nil
}

// piLSPServers returns the merged "servers" array as raw JSON: existing
// foreign servers are kept verbatim (raw slice), skillgrid ids are replaced
// with the current catalog entries.
func piLSPServers(raw []byte) string {
	skillgridIDs := map[string]bool{}
	for _, e := range lspServerEntries() {
		skillgridIDs[e.ID] = true
	}
	parts := []string{}
	for _, s := range gjson.GetBytes(raw, "servers").Array() {
		id := s.Get("id").String()
		// Keep foreign servers verbatim; drop skillgrid ids (replaced below).
		// Anonymous entries (no id) are kept to avoid data loss.
		if id != "" && skillgridIDs[id] {
			continue
		}
		parts = append(parts, s.Raw)
	}
	for _, e := range lspServerEntries() {
		command := map[string]any{"command": e.Server}
		if len(e.Args) > 0 {
			command["args"] = e.Args
		}
		b, err := json.Marshal(map[string]any{
			"id":          e.ID,
			"name":        e.Name,
			"languages":   e.Languages,
			"rootMarkers": e.RootMarkers,
			"command":     command,
		})
		if err != nil {
			// The map is fixed-shape; a marshal failure is a programming error.
			panic(err)
		}
		parts = append(parts, string(b))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// writeJSONFile writes data to path, creating parent dirs as needed.
func writeJSONFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// upsertPiCCLSPPackage appends "npm:cclsp" to the agent settings "packages"
// array (unique). A missing/empty settings file is created with the key.
func upsertPiCCLSPPackage(c *Config, settingsPath string) error {
	const pkg = "npm:cclsp"
	var raw []byte
	if data, err := os.ReadFile(settingsPath); err == nil {
		raw = data
	} else {
		raw = []byte("{}")
	}
	arr := gjson.GetBytes(raw, "packages").Array()
	exists := false
	for _, v := range arr {
		if v.String() == pkg {
			exists = true
			break
		}
	}
	if exists {
		Out("      skip", pkg, "(already in", settingsPath+")")
		return nil
	}
	parts := []string{}
	for _, v := range arr {
		parts = append(parts, v.Raw)
	}
	updated, err := sjson.SetRawBytes(raw, "packages", []byte("["+strings.Join(append(parts, `"`+pkg+`"`), ",")+"]"))
	if err != nil {
		return fmt.Errorf("append %s: %w", pkg, err)
	}
	if c.DryRun {
		Out("      [dry-run] append", pkg, "to", settingsPath)
		return nil
	}
	if err := writeJSONFile(settingsPath, updated); err != nil {
		return err
	}
	Out("      appended", pkg, "to", settingsPath)
	return nil
}
