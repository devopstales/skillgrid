package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedTTLProjects plants two project stores in dataDir: one with an expired
// observation, one with a live (future) observation, so `mem expire` has
// something to retire in exactly the first project. Stores are opened/closed
// around the writes so the memory.Service (which aliases the store DB) is never
// used after Close.
func seedTTLProjects(t *testing.T, dataDir string) {
	t.Helper()
	ctx := context.Background()

	plant := func(project, sessionID, title, content string, offset time.Duration) {
		t.Helper()
		st, err := store.Open(dataDir, project)
		if err != nil {
			t.Fatalf("open %s: %v", project, err)
		}
		// observations.session_id REFERENCES sessions(id): create the session row first.
		if _, err := st.DB.Exec(`
			INSERT INTO sessions (id, project, directory, started_at, status, title, summary)
			VALUES (?, ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', 'ttl seed', '## Goal\nttl seed')`,
			sessionID, project); err != nil {
			t.Fatalf("insert session %s: %v", project, err)
		}
		mem := memory.New(st, project)
		id, err := mem.Save(ctx, memory.SaveInput{
			SessionID: sessionID, Type: "decision", Title: title, Content: content,
		})
		if err != nil {
			t.Fatalf("save %s: %v", project, err)
		}
		if err := mem.SetExpiresAt(ctx, id, time.Now().UTC().Add(offset).Format(time.RFC3339)); err != nil {
			t.Fatalf("set expires %s: %v", project, err)
		}
		st.Close()
	}

	plant("memcli-expire-exp", "sess-exp", "expire probe expired", "this row is past its ttl", -1*time.Hour)
	plant("memcli-expire-live", "sess-live", "expire probe live", "this row is not expired", 1*time.Hour)
}

// TestMemExpireCLI is 04.3 [AFK] — `skillgrid mem expire` routes through
// RunTTLExpiry on every project, prints a per-project retirement count, and
// exits 0 even when the data dir is empty (no stores to sweep).
func TestMemExpireCLI(t *testing.T) {
	dataDir := t.TempDir()
	seedTTLProjects(t, dataDir)

	// mem expire sweeps all projects and reports per-project counts.
	out := runMemCLI(t, dataDir, "expire", "--dir", dataDir)
	if !strings.Contains(out, "memcli-expire-exp") || !strings.Contains(out, "memcli-expire-live") {
		t.Fatalf("mem expire must report every project: %s", out)
	}
	// The expired project retires 1; the live project retires 0 (pretty-printed
	// JSON: `"project": N`).
	if !strings.Contains(out, `"memcli-expire-exp": 1`) || !strings.Contains(out, `"memcli-expire-live": 0`) {
		t.Fatalf("mem expire must report per-project counts (1 and 0): %s", out)
	}

	// Idempotent second run: nothing left to retire, still exit 0.
	out = runMemCLI(t, dataDir, "expire", "--dir", dataDir)
	if !strings.Contains(out, "memcli-expire-exp") {
		t.Fatalf("second mem expire must still list projects: %s", out)
	}

	// Missing/empty store: a fresh empty data dir still exits 0 (graceful).
	emptyDir := t.TempDir()
	cmd := exec.Command("go", "run", ".", "mem", "expire", "--dir", emptyDir)
	cmd.Dir = mustWD(t)
	cmd.Env = append(cmd.Environ(), "SKILLGRID_MNEMONIC_DATA_DIR="+emptyDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("mem expire on empty data dir should exit 0, got err=%v out=%s", err, out)
	}
}

// TestRunTTLExpiryDirect exercises the service seam behind the CLI: it retires
// expired rows across all projects and skips missing/corrupt stores without
// surfacing an error (best-effort).
func TestRunTTLExpiryDirect(t *testing.T) {
	dataDir := t.TempDir()
	seedTTLProjects(t, dataDir)
	svc := service.New(dataDir)
	counts, err := svc.RunTTLExpiry(context.Background())
	if err != nil {
		t.Fatalf("RunTTLExpiry: %v", err)
	}
	if counts["memcli-expire-exp"] != 1 {
		t.Fatalf("expected 1 retired in expired project, got %d", counts["memcli-expire-exp"])
	}
	if counts["memcli-expire-live"] != 0 {
		t.Fatalf("expected 0 retired in live project, got %d", counts["memcli-expire-live"])
	}
}
