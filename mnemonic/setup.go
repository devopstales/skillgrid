package mnemonic

import (
	"github.com/devopstales/skillgrid/mnemonic/internal/setup"
)

type MCPServerConfig = setup.MCPServerConfig

func FindRepoRoot(start string) string {
	return setup.FindRepoRoot(start)
}

func LoadMCPConfig(repoRoot string) ([]MCPServerConfig, error) {
	return setup.LoadMCPConfig(repoRoot)
}

func RunSetup(agent, repoRoot string, mcpEntries []MCPServerConfig, home string, dryRun bool) error {
	return setup.RunSetup(agent, repoRoot, mcpEntries, home, dryRun)
}

// KanbanExec is the `hermes` CLI invoked during SetupHermes's kanban step;
// exported so the skillgrid-cli installer tests (which cannot reach the
// internal setup package) can stub it and stay hermetic on machines with a
// real hermes binary on PATH.
var KanbanExec = setup.KanbanExec
