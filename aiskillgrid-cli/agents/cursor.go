package agents

import "path/filepath"

type Cursor struct{}

func (Cursor) Name() string { return "cursor" }

func (c Cursor) Install(ctx Context) (Result, error) {
	var skillsDir, rulesDir, mcpPath string
	if ctx.Scope == ScopeProject {
		skillsDir = filepath.Join(ctx.ProjectDir, ".cursor", "skills")
		rulesDir = filepath.Join(ctx.ProjectDir, ".cursor", "rules")
		mcpPath = filepath.Join(ctx.ProjectDir, ".cursor", "mcp.json")
	} else {
		skillsDir = filepath.Join(ctx.HomeRoot, ".cursor", "skills")
		rulesDir = filepath.Join(ctx.HomeRoot, ".cursor", "rules")
		mcpPath = filepath.Join(ctx.HomeRoot, ".cursor", "mcp.json")
	}
	written, err := installSkillsRulesAndMCP(ctx, skillsDir, rulesDir, "", mcpPath, "mcpServers")
	return Result{Agent: c.Name(), Written: written}, err
}
