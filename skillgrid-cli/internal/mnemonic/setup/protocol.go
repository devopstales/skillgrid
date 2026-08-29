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
		for _, rel := range []string{"plugins/opencode/" + protocolRelPath, "plugins/kilo/" + protocolRelPath} {
			path := filepath.Join(home, ".skillgrid", "repos", "skillgrid", rel)
			if data, err := os.ReadFile(path); err == nil {
				return string(data)
			}
		}
	}
	return ""
}

// ProtocolMarkdownFromRepo reads protocol text from repoRoot when the file exists.
func ProtocolMarkdownFromRepo(repoRoot string) string {
	if repoRoot != "" {
		for _, rel := range []string{"plugins/opencode/" + protocolRelPath, "plugins/kilo/" + protocolRelPath} {
			path := filepath.Join(repoRoot, rel)
			if data, err := os.ReadFile(path); err == nil {
				return string(data)
			}
		}
	}
	return ProtocolMarkdown()
}
