## Why

OpenSpec changes and tasks currently have no terminal-native interface for viewing, navigating, and managing them. Operators working in a terminal (Kilo, OpenCode TUI, SSH sessions) need a fast keyboard-driven interface for their OpenSpec backlog — similar to how `dossier`, `Backlog.md`, and `beads_viewer` provide TUI views over structured work items.

## What Changes

- Add `skillgrid-tui` command providing a terminal UI for OpenSpec changes and tasks
- Implement change list view with status, progress, and last-modified columns
- Implement task list view with epic grouping and checkbox toggling
- Support keyboard navigation (vim-style), filtering, and search
- Integrate with existing `openspec list --json` and `openspec status --change --json` for data

## Capabilities

### New Capabilities

- `openspec-ticketing-tui`: Terminal UI for browsing and managing OpenSpec changes and tasks

### Modified Capabilities

None — this is a new standalone tool.

## Impact

- **Affected code**: New `cmd/tui.go` in `skillgrid-cli`, new `internal/tui/` package
- **Affected systems**: Terminal UI using Bubble Tea (Go TUI framework)
- **Dependencies**: `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`
- **Users**: All skillgrid users who prefer terminal-based backlog management
