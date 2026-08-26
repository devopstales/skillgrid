package setup

import (
	"os"
	"path/filepath"
)

const protocolRelPath = "plugins/mnemonic/shared/memory-protocol.md"

// ProtocolMarkdown returns the Mnemonic memory protocol markdown.
// Prefers the synced repo copy at ~/.skillgrid/repos/skillgrid when present.
func ProtocolMarkdown() string {
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".skillgrid", "repos", "skillgrid", protocolRelPath)
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}
	return ""
}

// ProtocolMarkdownFromRepo reads protocol text from repoRoot when the file exists.
func ProtocolMarkdownFromRepo(repoRoot string) string {
	if repoRoot != "" {
		path := filepath.Join(repoRoot, protocolRelPath)
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}
	return ProtocolMarkdown()
}
