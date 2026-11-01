package memory

import "strings"

// shouldPromote reports whether a finished tool call is worth an observation.
// The event row is always stored. An observation is written only for a
// failing command, a file written more than once in the session, or text
// that states a decision. A fast successful read is not promoted.
func shouldPromote(isErr bool, action string, writesToPath int, preview, command string) bool {
	if isErr {
		return true
	}
	if action == "file_write" && writesToPath >= 2 {
		return true
	}
	blob := strings.ToLower(preview + "\n" + command)
	if strings.Contains(blob, "we decided") || strings.Contains(blob, "decision:") {
		return true
	}
	return false
}

func promoteType(isErr bool, action string, preview, command string) string {
	blob := strings.ToLower(preview + "\n" + command)
	if strings.Contains(blob, "we decided") || strings.Contains(blob, "decision:") {
		return "decision"
	}
	if isErr {
		return "bugfix"
	}
	if action == "file_write" {
		return "pattern"
	}
	return "learning"
}

func promoteTitle(isErr bool, action, path, command string) string {
	switch {
	case isErr && command != "":
		return "Failed command"
	case isErr && path != "":
		return "Failed tool on " + path
	case isErr:
		return "Failed tool call"
	case action == "file_write" && path != "":
		return "Repeated edit " + path
	default:
		return "Noted decision"
	}
}
