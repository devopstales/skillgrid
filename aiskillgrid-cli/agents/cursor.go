package agents

import "path/filepath"

type Cursor struct{}

func (Cursor) Name() string { return "cursor" }

func (c Cursor) Install(ctx Context) (Result, error) {
	var skillsDir, mcpPath string
	if ctx.Scope == ScopeProject {
		skillsDir = filepath.Join(ctx.ProjectDir, ".cursor", "skills")
		mcpPath = filepath.Join(ctx.ProjectDir, ".cursor", "mcp.json")
	} else {
		skillsDir = filepath.Join(ctx.HomeRoot, ".cursor", "skills")
		mcpPath = filepath.Join(ctx.HomeRoot, ".cursor", "mcp.json")
	}
	written, err := installSkillsAndMCP(ctx, skillsDir, mcpPath, "mcpServers")
	return Result{Agent: c.Name(), Written: written}, err
}
