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
