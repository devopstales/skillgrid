package loop

import (
	"os"
	"path/filepath"
	"strings"
)

// ActiveProjectFile is the last project a prime or an explicit tool open
// pinned. MCP tools read it when the process cwd is an ambiguous parent of
// several git repos and the call did not pass project=.
func ActiveProjectFile() string {
	if d := strings.TrimSpace(os.Getenv("SKILLGRID_MNEMONIC_DATA_DIR")); d != "" {
		return filepath.Join(d, "active-project")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "skillgrid-mnemonic-active-project")
	}
	return filepath.Join(home, ".skillgrid", "mnemonic", "active-project")
}

// WriteActiveProject records id as the project later tool calls should open
// when cwd resolution is ambiguous.
func WriteActiveProject(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	path := ActiveProjectFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(id+"\n"), 0o644)
}

// ReadActiveProject returns the pinned project id, or empty when unset.
func ReadActiveProject() string {
	b, err := os.ReadFile(ActiveProjectFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
