package setup

import (
	"os"
	"path/filepath"
)

const protocolRelPath = "memory-protocol.md"

// ProtocolMarkdown returns the Mnemonic memory protocol markdown.
// Prefers the synced repo copy at ~/.skillgrid/repos/skillgrid when present.
func ProtocolMarkdown() string {
	if home, err := os.UserHomeDir(); err == nil {
		for _, rel := range protocolRelPaths() {
			path := filepath.Join(home, ".skillgrid", "repos", "skillgrid", rel)
			if data, err := os.ReadFile(path); err == nil {
				return string(data)
			}
		}
	}
	return ""
}

// protocolRelPaths lists where the protocol markdown may live, repo-relative.
// The shared rule under .agents/skills is the current home (AGENTS.md
// "Memory"); the plugins/ copies are the older layout.
func protocolRelPaths() []string {
	return []string{
		"plugins/_shared/" + protocolRelPath,
		"plugins/opencode/" + protocolRelPath,
		"plugins/kilo/" + protocolRelPath,
		".agents/skills/_shared/rules/mnemonic-memory.md",
	}
}

// ProtocolMarkdownFromRepo reads protocol text from repoRoot when the file exists.
// The repo copy wins over a staged install under ~/.skillgrid/plugins.
func ProtocolMarkdownFromRepo(repoRoot string) string {
	if repoRoot != "" {
		for _, rel := range protocolRelPaths() {
			path := filepath.Join(repoRoot, rel)
			if data, err := os.ReadFile(path); err == nil {
				return string(data)
			}
		}
	}
	for _, rel := range protocolRelPaths() {
		if staged := stagedPluginPath(rel); staged != "" {
			if data, err := os.ReadFile(staged); err == nil {
				return string(data)
			}
		}
	}
	return ProtocolMarkdown()
}
