package handoff

import "strings"

// parseContext extracts the [skillgrid-context] block from a commit message
// body. The block format (owned by work-unit-commits + checkpoint-state.sh):
//
//	[skillgrid-context]
//	Task: ...
//	Decisions: ...
//	Remaining: ...
//	Tried: ...
//	[/skillgrid-context]
//
// Field values are single-line; a field present but empty keeps its empty
// string. Returns a zero Context when the block is absent. The parse is
// intentionally independent of the bash script so the Go engine is the single
// source of truth (the script keeps its own copy for standalone use).
func parseContext(body string) Context {
	var c Context
	lines := strings.Split(body, "\n")
	inBlock := false
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		switch {
		case !inBlock:
			if strings.EqualFold(trimmed, "[skillgrid-context]") {
				inBlock = true
			}
		case strings.EqualFold(trimmed, "[/skillgrid-context]"):
			return c
		case len(trimmed) >= 6 && strings.EqualFold(trimmed[:4], "task") && trimmed[4] == ':':
			c.Task = strings.TrimSpace(trimmed[5:])
		case len(trimmed) >= 7 && strings.EqualFold(trimmed[:5], "tried") && trimmed[5] == ':':
			c.Tried = strings.TrimSpace(trimmed[6:])
		case len(trimmed) >= 11 && strings.EqualFold(trimmed[:9], "decisions") && trimmed[9] == ':':
			c.Decisions = strings.TrimSpace(trimmed[10:])
		case len(trimmed) >= 11 && strings.EqualFold(trimmed[:9], "remaining") && trimmed[9] == ':':
			c.Remaining = strings.TrimSpace(trimmed[10:])
		}
	}
	return c
}
