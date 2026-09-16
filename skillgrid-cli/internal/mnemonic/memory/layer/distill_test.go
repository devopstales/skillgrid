package layer

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

type sqlNullString = sql.NullString

type fixture struct {
	st       *store.Store
	svc      *memory.Service
	projectID string
	sessionID string
}

func newDistillFixture(t *testing.T, project, sessionID, summary string) *fixture {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	svc := memory.New(st, project)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, '/tmp', ?, 'active')`, sessionID, project, now); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if summary != "" {
		if _, err := st.DB.Exec(`
			UPDATE sessions SET summary = ? WHERE id = ? AND project = ?`,
			summary, sessionID, project); err != nil {
			t.Fatalf("set summary: %v", err)
		}
	}
	return &fixture{st: st, svc: svc, projectID: project, sessionID: sessionID}
}

// countRows returns the number of rows in table for project.
func countRows(t *testing.T, st *store.Store, project, table, where string) int {
	t.Helper()
	q := `SELECT COUNT(*) FROM ` + table + ` WHERE project = ?`
	args := []any{project}
	if where != "" {
		q += " AND " + where
	}
	var n int
	if err := st.DB.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

const distillableSummary = `## Goal
Tune the auth token rotation.

## Key Learnings:

1. JWT refresh tokens need atomic rotation to avoid races
2. bcrypt cost=12 is the right balance for our server
3. FTS5 queries must be sanitized before MATCH
`

// TestDistillProvenance is 02.1 (Scenario: distilled-layer-carries-resolvable-l0-provenance).
// Every distilled L1/L2/L3 record must carry a link to a RESOLVABLE L0 source
// (a live session row + a non-empty source topic). After a distill of a real
// session, the layers table must be non-empty and every row's source_session
// must resolve to a sessions row.
func TestDistillProvenance(t *testing.T) {
	fx := newDistillFixture(t, "prov", "sess-prov", distillableSummary)
	ctx := context.Background()

	res, err := Distill(ctx, fx.svc, fx.sessionID, DistillOptions{})
	if err != nil {
		t.Fatalf("distill: %v", err)
	}
	if res.L1Count == 0 {
		t.Fatalf("expected L1 atoms from distillable content, got %d (scenario=%d)", res.L1Count, res.ScenarioCount)
	}
	if res.ScenarioCount == 0 {
		t.Errorf("expected an L2 scenario, got 0")
	}
	if res.PersonaDeltaCount == 0 {
		t.Errorf("expected an L3 persona delta, got 0")
	}

	// Every L1 row must link to a resolvable L0 source.
	rows, err := fx.st.DB.Query(`
		SELECT source_session FROM observation_layers
		WHERE project = ? AND layer = 'L1'`, fx.projectID)
	if err != nil {
		t.Fatalf("query layers: %v", err)
	}
	defer rows.Close()
	var sessions []string
	for rows.Next() {
		var ss string
		if err := rows.Scan(&ss); err != nil {
			t.Fatalf("scan layer: %v", err)
		}
		sessions = append(sessions, ss)
	}
	if len(sessions) == 0 {
		t.Fatal("no L1 layer rows written")
	}
	// Each source_session must resolve to a live sessions row AND the link's
	// source topic must be non-empty (a resolvable L0, not an orphan).
	for _, ss := range sessions {
		var n int
		if err := fx.st.DB.QueryRow(
			`SELECT COUNT(*) FROM sessions WHERE id = ? AND project = ?`, ss, fx.projectID,
		).Scan(&n); err != nil || n == 0 {
			t.Fatalf("L1 layer links to unresolvable session %q (orphan layer)", ss)
		}
		var topic sqlNullString
		if err := fx.st.DB.QueryRow(
			`SELECT source_topic FROM observation_layers
			 WHERE project = ? AND layer = 'L1' AND source_session = ? LIMIT 1`,
			fx.projectID, ss,
		).Scan(&topic); err != nil {
			t.Fatalf("read source_topic: %v", err)
		}
		if !topic.Valid || topic.String == "" {
			t.Fatalf("L1 layer for %q has an empty source topic (not a resolvable L0)", ss)
		}
	}
}

// TestDistillUnresolvableSourceNotCreated is 02.1 (Scenario:
// layer-with-unresolvable-source-not-created). Distilling a session id that has
// no sessions row must create NO layer rows — a layer is never orphaned from
// its provenance.
func TestDistillUnresolvableSourceNotCreated(t *testing.T) {
	fx := newDistillFixture(t, "unres", "sess-unres", "")
	ctx := context.Background()

	before := countRows(t, fx.st, fx.projectID, "observation_layers", "")
	res, err := Distill(ctx, fx.svc, "ghost-session-not-a-row", DistillOptions{})
	if err != nil {
		t.Fatalf("distill (unresolvable) should not error: %v", err)
	}
	// The source cannot be resolved, so nothing is created and the result is a
	// no-op.
	if res.Created {
		t.Fatalf("expected no-op for unresolvable source, got created=%v (L1=%d)", res.Created, res.L1Count)
	}
	after := countRows(t, fx.st, fx.projectID, "observation_layers", "")
	if after != before {
		t.Fatalf("orphan layers created: before=%d after=%d", before, after)
	}
}

// TestDistillNoLLMFloor is 02.1 (Scenario: no-llm-floor-produces-provenance-ladder-offline).
// With NO LLM available, the deterministic floor still produces a
// provenance-linked L0→L1→L2→L3 ladder offline.
func TestDistillNoLLMFloor(t *testing.T) {
	fx := newDistillFixture(t, "floor", "sess-floor", distillableSummary)
	ctx := context.Background()

	res, err := Distill(ctx, fx.svc, fx.sessionID, DistillOptions{})
	if err != nil {
		t.Fatalf("offline distill: %v", err)
	}
	if res.L1Count == 0 || res.ScenarioCount == 0 || res.PersonaDeltaCount == 0 {
		t.Fatalf("no-LLM floor must produce L1+L2+L3, got L1=%d L2=%d L3=%d",
			res.L1Count, res.ScenarioCount, res.PersonaDeltaCount)
	}
	if res.LLMUsed {
		t.Errorf("offline floor must not use the LLM (llmUsed=%v)", res.LLMUsed)
	}
	// The produced ladder is provenance-linked: Inspect must return the chain
	// with a resolvable L0.
	chain, err := Inspect(ctx, fx.svc, fx.sessionID)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if chain.L0Session == "" {
		t.Fatalf("inspect returned empty L0 session")
	}
	if len(chain.Atoms) == 0 {
		t.Fatalf("inspect returned no L1 atoms")
	}
}
