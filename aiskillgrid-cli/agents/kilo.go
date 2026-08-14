package agents

import "path/filepath"

// Kilo covers both Kilo CLI and Kilo Code in v1 (shared skills + kilo.jsonc paths).
type Kilo struct{}

func (Kilo) Name() string { return "kilo" }

func (k Kilo) Install(ctx Context) (Result, error) {
	var skillsDir, rulesDir, mcpPath string
	if ctx.Scope == ScopeProject {
		skillsDir = filepath.Join(ctx.ProjectDir, ".kilo", "skills")
		rulesDir = filepath.Join(ctx.ProjectDir, ".kilo", "rules")
		mcpPath = filepath.Join(ctx.ProjectDir, ".kilo", "kilo.jsonc")
	} else {
		skillsDir = filepath.Join(ctx.HomeRoot, ".kilo", "skills")
		rulesDir = filepath.Join(ctx.HomeRoot, ".kilo", "rules")
		mcpPath = filepath.Join(ctx.ConfigDir, "kilo", "kilo.jsonc")
	}
	written, err := installSkillsRulesAndMCP(ctx, skillsDir, rulesDir, "", mcpPath, "mcp")
	return Result{Agent: k.Name(), Written: written}, err
}
