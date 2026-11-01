package mcp

import "testing"

// TestRemovedToolsAbsent is the old-surfaces-removed registry oracle (session
// events layer, TICKET-06): the 9 hub/relay MCP tools are gone from the
// server surface while session_changes (the events-layer read) stays.
func TestRemovedToolsAbsent(t *testing.T) {
	tools := NewServer().ListTools()
	removed := []string{
		"handoff_snapshot", "handoff_status", "handoff_checkpoint", "handoff_verify", "handoff_rollup",
		"session_handoff", "session_resume",
		"session_status", "knowledge_compact",
	}
	for _, name := range removed {
		if _, ok := tools[name]; ok {
			t.Errorf("expected removed tool %q to be absent, but it is still registered", name)
		}
	}
	if _, ok := tools["session_changes"]; !ok {
		t.Errorf("expected tool %q to remain registered", "session_changes")
	}
}
