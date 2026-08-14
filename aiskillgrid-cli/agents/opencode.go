package agents

import "path/filepath"

type OpenCode struct{}

func (OpenCode) Name() string { return "opencode" }

func (o OpenCode) Install(ctx Context) (Result, error) {
	var skillsDir, mcpPath string
	if ctx.Scope == ScopeProject {
		skillsDir = filepath.Join(ctx.ProjectDir, ".opencode", "skills")
		mcpPath = filepath.Join(ctx.ProjectDir, "opencode.json")
	} else {
		skillsDir = filepath.Join(ctx.ConfigDir, "opencode", "skills")
		mcpPath = filepath.Join(ctx.ConfigDir, "opencode", "opencode.json")
	}
	written, err := installSkillsAndMCP(ctx, skillsDir, mcpPath, "mcp")
	return Result{Agent: o.Name(), Written: written}, err
}
