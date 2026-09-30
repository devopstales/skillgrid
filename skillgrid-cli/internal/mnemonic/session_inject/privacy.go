package session_inject

import (
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// Injectable reports whether a session event is safe to include in the
// injected context, returning !e.IsSensitive.
func Injectable(e memory.Event) bool {
	return !e.IsSensitive
}

// ObservationInjectable reports whether an observation is safe to include.
// Observations whose title or content mentions "private" are excluded.
func ObservationInjectable(o memory.Observation) bool {
	lower := strings.ToLower(o.Title + " " + o.Content)
	return !strings.Contains(lower, "private")
}

// redactFullPaths replaces absolute local path prefixes with a placeholder so
// full local paths do not leak into injected context. Project-relative paths
// are left unchanged.
func redactFullPaths(s string) string {
	for _, prefix := range []string{"/Users/", "/home/", `C:\`} {
		s = strings.ReplaceAll(s, prefix, "<path>/")
	}
	return s
}
