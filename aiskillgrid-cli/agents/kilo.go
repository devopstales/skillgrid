package agents

import "path/filepath"

// Kilo covers both Kilo CLI and Kilo Code in v1 (shared skills + kilo.jsonc paths).
type Kilo struct{}

func (Kilo) Name() string { return "kilo" }

func (k Kilo) Install(ctx Context) (Result, error) {
	var skillsDir, mcpPath string
	if ctx.Scope == ScopeProject {
		skillsDir = filepath.Join(ctx.ProjectDir, ".kilo", "skills")
		mcpPath = filepath.Join(ctx.ProjectDir, ".kilo", "kilo.jsonc")
	} else {
		skillsDir = filepath.Join(ctx.HomeRoot, ".kilo", "skills")
		mcpPath = filepath.Join(ctx.ConfigDir, "kilo", "kilo.jsonc")
	}
	written, err := installSkillsAndMCP(ctx, skillsDir, mcpPath, "mcp")
	return Result{Agent: k.Name(), Written: written}, err
}
