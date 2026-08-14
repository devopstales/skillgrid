package agents

import "path/filepath"

// Copilot is the VS Code (Copilot) client adapter.
// Global MCP: ~/.config/Code/User/mcp.json (Linux/macOS XDG) or %APPDATA%/Code/User/mcp.json
// Project MCP: .vscode/mcp.json
// Skills: .github/skills (project) or ~/.copilot/skills (global)
type Copilot struct{}

func (Copilot) Name() string { return "copilot" }

func (c Copilot) Install(ctx Context) (Result, error) {
	var skillsDir, mcpPath string
	if ctx.Scope == ScopeProject {
		skillsDir = filepath.Join(ctx.ProjectDir, ".github", "skills")
		mcpPath = filepath.Join(ctx.ProjectDir, ".vscode", "mcp.json")
	} else {
		skillsDir = filepath.Join(ctx.HomeRoot, ".copilot", "skills")
		mcpPath = filepath.Join(ctx.ConfigDir, "Code", "User", "mcp.json")
	}
	written, err := installSkillsAndMCP(ctx, skillsDir, mcpPath, "servers")
	return Result{Agent: c.Name(), Written: written}, err
}
