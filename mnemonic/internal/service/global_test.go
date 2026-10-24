// Global store bootstrap (change 2026-10-24-global-llm-wiki, TICKET-01).
//
// RED notes (the failures that prove each test exercises new code):
//   - TestOpenGlobalCreatesUnmangledStore / TestOpenGlobalCategoryColumn:
//     compile error — Service.OpenGlobal and the GlobalProjectID constant do
//     not exist yet.
//   - TestListProjectsExcludesGlobal / TestRunTTLExpirySkipsGlobal: the
//     __global__ pseudo-project is currently treated like any other project
//     id, so ListProjects includes it and RunTTLExpiry opens + sweeps it
//     (an expired global observation gets retired and counted).
package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// globalStoreFile is the exact store file name the pseudo-project must
// produce (door check G5).
const globalStoreFile = GlobalProjectID + ".sqlite"

// TestOpenGlobalCreatesUnmangledStore is the door check for the whole change:
// OpenGlobal() creates the global store at dataDir/__global__.sqlite through
// the same store.Open path as project stores, and the file name is exactly
// __global__.sqlite — not what project.NormalizeID would produce for the id
// ("__global__" collapses to "-global-", a leading-dash file name).
func TestOpenGlobalCreatesUnmangledStore(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)

	h, cleanup, err := svc.OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal: %v", err)
	}
	defer cleanup()
	if h.ProjectID() != GlobalProjectID {
		t.Fatalf("ProjectID = %q, want %q (the pseudo-project id must not be mangled by normalization)", h.ProjectID(), GlobalProjectID)
	}

	name := filepath.Base(h.Store().Path())
	if name != globalStoreFile {
		t.Fatalf("store file = %q, want exactly %q", name, globalStoreFile)
	}
	if _, err := os.Stat(filepath.Join(dataDir, globalStoreFile)); err != nil {
		t.Fatalf("%s missing from data dir: %v", globalStoreFile, err)
	}
	// A mangled open (via NormalizeID) would have created a leading-dash
	// file; assert the data dir holds no other store file.
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("read data dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sqlite") && e.Name() != globalStoreFile {
			t.Fatalf("unexpected store file %q in data dir (id mangled by normalization?)", e.Name())
		}
	}
}

// TestOpenGlobalCategoryColumn verifies the global store carries the facts
// schema from migration 011 plus the 072 category column, whose default is
// 'memory' for rows that do not set it (the "existing rows" semantics:
// migration 072 is one-way, so old rows read back as 'memory').
func TestOpenGlobalCategoryColumn(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)

	h, cleanup, err := svc.OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal: %v", err)
	}
	defer cleanup()

	var (
		def     string
		notNull int
	)
	if err := h.Store().DB.QueryRow(`
		SELECT COALESCE(dflt_value, ''), "notnull"
		FROM pragma_table_info('facts') WHERE name = 'category'`).Scan(&def, &notNull); err != nil {
		t.Fatalf("facts.category column missing: %v", err)
	}
	if def != "'memory'" {
		t.Fatalf("category default = %q, want the schema default 'memory' (as quoted in the pragma)", def)
	}
	if notNull == 0 {
		t.Fatal("facts.category must be NOT NULL")
	}

	// A row that omits the column takes the default — the same semantics the
	// migration gives existing rows.
	if _, err := h.Store().DB.Exec(`INSERT INTO facts (content, created_at, updated_at) VALUES ('legacy row', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert fact: %v", err)
	}
	var category string
	if err := h.Store().DB.QueryRow(`SELECT category FROM facts WHERE content = 'legacy row'`).Scan(&category); err != nil {
		t.Fatalf("read category: %v", err)
	}
	if category != "memory" {
		t.Fatalf("row category = %q, want 'memory'", category)
	}
}

// TestListProjectsExcludesGlobal is the listing edge: a data dir holding both
// a real project store and the global store must list only the real project —
// the global store is cross-project infrastructure, not a selectable project.
func TestListProjectsExcludesGlobal(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)

	seedProjectObs(t, svc, "proj-a", "global listing probe", "**What** l **Why** t **Where** t **Learned** —")
	if _, cleanup, err := svc.OpenGlobal(); err != nil {
		t.Fatalf("OpenGlobal: %v", err)
	} else {
		cleanup()
	}

	projects, err := svc.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if !contains(projects, "proj-a") {
		t.Fatalf("ListProjects missing proj-a: got %v", projects)
	}
	if contains(projects, GlobalProjectID) {
		t.Fatalf("ListProjects includes the global pseudo-project: got %v", projects)
	}
}

// TestRunTTLExpirySkipsGlobal is the TTL edge: the sweep must leave the global
// store completely alone (an expired global observation survives the sweep
// and no __global__ count is reported), while per-project stores still retire
// their expired rows normally.
func TestRunTTLExpirySkipsGlobal(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	ctx := context.Background()

	gh, cleanup, err := svc.OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal: %v", err)
	}
	defer cleanup()

	// An expired observation in the GLOBAL store: TTLRetire would soft-delete
	// it (counting it) if the sweep opened the global project.
	gsid, err := gh.memory.SessionStart(ctx, t.TempDir(), "global ttl seed")
	if err != nil {
		t.Fatalf("global session: %v", err)
	}
	gid, err := gh.memory.Save(ctx, memory.SaveInput{
		Title: "global ttl expired", Type: "decision", Content: "this row is past its ttl but global",
		SessionID: gsid, Scope: "project",
	})
	if err != nil {
		t.Fatalf("global save: %v", err)
	}
	if err := gh.memory.SetExpiresAt(ctx, gid, pastTTLForTest()); err != nil {
		t.Fatalf("global set expires: %v", err)
	}

	// An expired observation in the PROJECT store: must be retired normally.
	ph, pcleanup, err := svc.openProject("proj-a", ".")
	if err != nil {
		t.Fatalf("open proj-a: %v", err)
	}
	defer pcleanup()
	psid, err := ph.memory.SessionStart(ctx, t.TempDir(), "proj ttl seed")
	if err != nil {
		t.Fatalf("proj session: %v", err)
	}
	prow, err := ph.memory.Save(ctx, memory.SaveInput{
		Title: "proj ttl expired", Type: "decision", Content: "this row is past its ttl",
		SessionID: psid, Scope: "project",
	})
	if err != nil {
		t.Fatalf("proj save: %v", err)
	}
	if err := ph.memory.SetExpiresAt(ctx, prow, pastTTLForTest()); err != nil {
		t.Fatalf("proj set expires: %v", err)
	}

	counts, err := svc.RunTTLExpiry(ctx)
	if err != nil {
		t.Fatalf("RunTTLExpiry: %v", err)
	}
	if counts[GlobalProjectID] != 0 {
		t.Fatalf("RunTTLExpiry reported a count for the global store: got %v", counts)
	}
	if counts["proj-a"] != 1 {
		t.Fatalf("per-project sweep must retire 1 (the expired proj-a row): got %v", counts)
	}

	var deletedAt sql.NullString
	if err := gh.Store().DB.QueryRow(`SELECT deleted_at FROM observations_core WHERE id = ?`, gid).Scan(&deletedAt); err != nil {
		t.Fatalf("read global row: %v", err)
	}
	if deletedAt.Valid && deletedAt.String != "" {
		t.Fatalf("global store observation was TTL-expired: deleted_at=%q", deletedAt.String)
	}
}

// pastTTLForTest is an RFC3339 timestamp one hour in the past (the same
// convention the cmd/mnemonic mem-expire seed tests use).
func pastTTLForTest() string {
	return time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
}

// TestRunTTLExpirySkipsGlobalWithObservationSeeded uses a different seed
// convention (a raw session + observation row, no service Save) with the same
// inputs as TestRunTTLExpirySkipsGlobal: an expired global observation plus a
// live project observation. It triangulates the skip — the sweep must still
// leave the global row alone while reporting nothing for the global id, and
// the live project row must survive (retired count 0).
func TestRunTTLExpirySkipsGlobalWithObservationSeeded(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	ctx := context.Background()

	plant := func(project, sessionID string, offset time.Duration) int64 {
		t.Helper()
		st, err := store.Open(dataDir, project)
		if err != nil {
			t.Fatalf("open %s: %v", project, err)
		}
		defer st.Close()
		if _, err := st.DB.Exec(`
			INSERT INTO sessions (id, project, directory, started_at, status, title, summary)
			VALUES (?, ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', 'ttl seed', '## Goal\nttl seed')`,
			sessionID, project); err != nil {
			t.Fatalf("insert session %s: %v", project, err)
		}
		mem := memory.New(st, project)
		id, err := mem.Save(ctx, memory.SaveInput{
			Title: "ttl seed", Type: "decision", Content: "triangulation seed row",
			SessionID: sessionID,
		})
		if err != nil {
			t.Fatalf("save %s: %v", project, err)
		}
		if err := mem.SetExpiresAt(ctx, id, time.Now().UTC().Add(offset).Format(time.RFC3339)); err != nil {
			t.Fatalf("set expires %s: %v", project, err)
		}
		return id
	}

	// Global: an EXPIRED row that must survive the sweep.
	// Project: a LIVE row that must survive the sweep (retires 0).
	gid := plant(GlobalProjectID, "sess-global", -1*time.Hour)
	plant("proj-b", "sess-proj", 1*time.Hour)

	counts, err := svc.RunTTLExpiry(ctx)
	if err != nil {
		t.Fatalf("RunTTLExpiry: %v", err)
	}
	if _, present := counts[GlobalProjectID]; present {
		t.Fatalf("RunTTLExpiry reported the global pseudo-project: got %v", counts)
	}
	if counts["proj-b"] != 0 {
		t.Fatalf("live project row must not be retired: got %v", counts)
	}

	st, err := store.Open(dataDir, GlobalProjectID)
	if err != nil {
		t.Fatalf("reopen global store: %v", err)
	}
	defer st.Close()
	var deletedAt sql.NullString
	if err := st.DB.QueryRow(`SELECT deleted_at FROM observations_core WHERE id = ?`, gid).Scan(&deletedAt); err != nil {
		t.Fatalf("read global row: %v", err)
	}
	if deletedAt.Valid && deletedAt.String != "" {
		t.Fatalf("global store observation was TTL-expired: deleted_at=%q", deletedAt.String)
	}
}
