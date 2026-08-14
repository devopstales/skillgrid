package agents

import "path/filepath"

// Copilot is the VS Code (Copilot) client adapter.
// Global MCP: ~/.config/Code/User/mcp.json (Linux/macOS XDG) or %APPDATA%/Code/User/mcp.json
// Project MCP: .vscode/mcp.json
// Skills: .github/skills (project) or ~/.copilot/skills (global)
// Rules: .github/instructions/*.instructions.md (project) or ~/.copilot/instructions (global)
type Copilot struct{}

func (Copilot) Name() string { return "copilot" }

func (c Copilot) Install(ctx Context) (Result, error) {
	var skillsDir, rulesDir, mcpPath string
	if ctx.Scope == ScopeProject {
		skillsDir = filepath.Join(ctx.ProjectDir, ".github", "skills")
		rulesDir = filepath.Join(ctx.ProjectDir, ".github", "instructions")
		mcpPath = filepath.Join(ctx.ProjectDir, ".vscode", "mcp.json")
	} else {
		skillsDir = filepath.Join(ctx.HomeRoot, ".copilot", "skills")
		rulesDir = filepath.Join(ctx.HomeRoot, ".copilot", "instructions")
		mcpPath = filepath.Join(ctx.ConfigDir, "Code", "User", "mcp.json")
	}
	written, err := installSkillsRulesAndMCP(ctx, skillsDir, rulesDir, ".instructions.md", mcpPath, "servers")
	return Result{Agent: c.Name(), Written: written}, err
}
