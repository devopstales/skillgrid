package mcp

import "testing"

// sessionHandoffFixture pins the project to a stable bucket, points the CWD at
// a temp dir, and injects a service rooted at a temp data dir. It chdirs into
// dir so handlers that open the CWD project resolve to that store. Returns
// the CWD dir first.
//
// Kept after the session events layer consolidation (TICKET-06 removed the
// session_handoff/session_resume dispatch tests): TestMemSave in
// server_test.go still seeds a real session through this fixture.
func sessionHandoffFixture(t *testing.T) (dir, dataDir string) {
	t.Helper()
	dataDir = t.TempDir()
	dir = t.TempDir()
	pinProjectCwd(t, dataDir, dir, "handoffmcp-probe")
	return dir, dataDir
}
