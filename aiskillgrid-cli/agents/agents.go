package agents

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aiskillgrid/aiskillgrid/home"
	"github.com/aiskillgrid/aiskillgrid/mcpmerge"
)

type Scope = home.Scope

const (
	ScopeGlobal  = home.ScopeGlobal
	ScopeProject = home.ScopeProject
)

type Context struct {
	Scope       Scope
	ProjectDir  string
	HomeRoot    string         // user home for global paths
	ConfigDir   string         // XDG/APPDATA config dir
	PackRoot    string         // synced repo root (~/.aiskillgrid/tools)
	ResolvedMCP map[string]any // optional; when set, used instead of pack file
}

type Result struct {
	Agent   string
	Written []string
}

type Agent interface {
	Name() string
	Install(ctx Context) (Result, error)
}

func All() []Agent {
	// v1 focus only. Planned later: Claude Code, pi, Gemini CLI, Antigravity, Codex.
	return []Agent{
		Kilo{},
		OpenCode{},
		Cursor{},
		Copilot{},
	}
}

func ByNames(names []string) ([]Agent, error) {
	index := map[string]Agent{}
	for _, a := range All() {
		index[a.Name()] = a
	}
	var out []Agent
	for _, n := range names {
		a, ok := index[n]
		if !ok {
			return nil, fmt.Errorf("unknown agent %q (v1: kilo, opencode, cursor, copilot)", n)
		}
		out = append(out, a)
	}
	return out, nil
}

func skillsSrc(packRoot string) string {
	return filepath.Join(packRoot, "packs", "skills")
}

func rulesSrc(packRoot string) string {
	return filepath.Join(packRoot, "packs", "rules")
}

func mcpPack(packRoot string) string {
	return filepath.Join(packRoot, "packs", "mcp", "servers.json")
}

func copySkills(srcDir, dstDir string) ([]string, error) {
	var written []string
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(dstDir, e.Name())
		if err := copyDir(src, dst); err != nil {
			return written, err
		}
		written = append(written, dst)
	}
	return written, nil
}

// copyRules copies flat rule files from packs/rules into dstDir.
// When destExt is non-empty (e.g. ".instructions.md"), .mdc sources are renamed
// to that extension for agents that do not use Cursor-style rules.
func copyRules(srcDir, dstDir, destExt string) ([]string, error) {
	var written []string
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		src := filepath.Join(srcDir, name)
		dstName := name
		if destExt != "" && filepath.Ext(name) == ".mdc" {
			dstName = strings.TrimSuffix(name, filepath.Ext(name)) + destExt
		}
		dst := filepath.Join(dstDir, dstName)
		if err := copyFile(src, dst); err != nil {
			return written, err
		}
		written = append(written, dst)
	}
	return written, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func mergeMCPFile(path, mcpKey, packRoot string, resolved map[string]any) error {
	servers := resolved
	var err error
	if servers == nil {
		servers, err = mcpmerge.LoadPackServers(mcpPack(packRoot))
		if err != nil {
			return err
		}
	}
	root, err := mcpmerge.LoadOrEmpty(path)
	if err != nil {
		return err
	}
	root = mcpmerge.MergeMCPServers(root, mcpKey, servers, home.KeyPrefix)
	return mcpmerge.WriteObject(path, root)
}

func installSkillsRulesAndMCP(ctx Context, skillsDir, rulesDir, rulesExt, mcpPath, mcpKey string) ([]string, error) {
	var written []string
	copied, err := copySkills(skillsSrc(ctx.PackRoot), skillsDir)
	if err != nil {
		return nil, err
	}
	written = append(written, copied...)
	if rulesDir != "" {
		rules, err := copyRules(rulesSrc(ctx.PackRoot), rulesDir, rulesExt)
		if err != nil {
			return written, err
		}
		written = append(written, rules...)
	}
	if err := mergeMCPFile(mcpPath, mcpKey, ctx.PackRoot, ctx.ResolvedMCP); err != nil {
		return written, err
	}
	written = append(written, mcpPath)
	return written, nil
}

// installSkillsAndMCP is kept for callers that only need skills+MCP (no rules dir).
func installSkillsAndMCP(ctx Context, skillsDir, mcpPath, mcpKey string) ([]string, error) {
	return installSkillsRulesAndMCP(ctx, skillsDir, "", "", mcpPath, mcpKey)
}
