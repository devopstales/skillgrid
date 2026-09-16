package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

const distillable = `## Goal
Tune token rotation.

## Key Learnings:

1. JWT refresh tokens need atomic rotation to avoid races
2. bcrypt cost=12 is the right balance for our server
`

// openDistillStore opens a project store, registers a session with a summary,
// and closes the raw handle so the facade can reopen it.
func openDistillStore(t *testing.T, project, sessionID, summary string) string {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	now := "2026-01-01T00:00:00Z"
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, '/tmp', ?, 'active')`, sessionID, project, now); err != nil {
		t.Fatal(err)
	}
	if summary != "" {
		if _, err := st.DB.Exec(`UPDATE sessions SET summary = ? WHERE id = ? AND project = ?`,
			summary, sessionID, project); err != nil {
			t.Fatal(err)
		}
	}
	st.Close()
	return dataDir
}

// TestDistillHook covers 02.4 (Scenario: session-close-distill-l0-to-l1-l2-l3).
// With distillation enabled, SessionEnd runs the pass and the L0 record is
// refined into L1 atoms + L2 scenario + L3 persona delta, each linked to its
// L0 source. It is async (the hook signals completion) and best-effort.
func TestDistillHook(t *testing.T) {
	project := "hook"
	sessionID := "sess-hook"
	dataDir := openDistillStore(t, project, sessionID, distillable)
	svc := service.New(dataDir)
	if err := svc.EnableDistill(); err != nil {
		t.Fatalf("enable distill: %v", err)
	}

	// wg signals the hook ran (the async seam); the method returns the result
	// so the hook's effect is observable here.
	var wg sync.WaitGroup
	out, err := svc.DistillSession(context.Background(), project, sessionID, "", &wg)
	if err != nil {
		t.Fatalf("distill session: %v", err)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("distill hook did not signal completion")
	}
	if out.Err != nil {
		t.Fatalf("distill best-effort error: %v", out.Err)
	}
	if !out.Created {
		t.Fatalf("expected distill to create layers, got %+v", out)
	}
	if out.L1Count == 0 || out.ScenarioCount == 0 || out.PersonaDeltaCount == 0 {
		t.Fatalf("incomplete ladder: L1=%d L2=%d L3=%d", out.L1Count, out.ScenarioCount, out.PersonaDeltaCount)
	}
}

// TestDistillHookDisabled covers 02.4: with distillation NOT enabled (the
// default), SessionEnd is a no-op for the distill pass — no layers are created.
func TestDistillHookDisabled(t *testing.T) {
	project := "hookoff"
	sessionID := "sess-hookoff"
	dataDir := openDistillStore(t, project, sessionID, distillable)
	svc := service.New(dataDir)

	var wg sync.WaitGroup
	out, err := svc.DistillSession(context.Background(), project, sessionID, "", &wg)
	if err != nil {
		t.Fatalf("distill session (disabled): %v", err)
	}
	if out.Created {
		t.Fatalf("distillation is opt-in; disabled must be a no-op, got %+v", out)
	}
	if out.L1Count != 0 || out.ScenarioCount != 0 || out.PersonaDeltaCount != 0 {
		t.Fatalf("disabled pass fabricated layers: L1=%d L2=%d L3=%d",
			out.L1Count, out.ScenarioCount, out.PersonaDeltaCount)
	}
}

// TestDistillHookBestEffort covers 02.4 (opt-in + async + best-effort): a
// distill that fails on a session that no longer exists is captured in the
// result (best-effort) and NEVER breaks the session close — the hook returns
// nil and the session row is still endable.
func TestDistillHookBestEffort(t *testing.T) {
	project := "hookbest"
	sessionID := "sess-hookbest"
	dataDir := openDistillStore(t, project, sessionID, distillable)
	svc := service.New(dataDir)
	if err := svc.EnableDistill(); err != nil {
		t.Fatal(err)
	}

	// Distill a session id that is not a resolvable source: the pass is a
	// no-op (unresolvable), and even a best-effort error would not break close.
	// A non-empty summary exercises the persist path (it errors for the missing
	// session, captured best-effort) before the unresolvable distill no-ops.
	var wg sync.WaitGroup
	out, err := svc.DistillSession(context.Background(), project, "ghost-not-a-session", "## Key Learnings:\n1. a ghost learning to persist\n", &wg)
	if err != nil {
		t.Fatalf("distill hook must not break session close: %v", err)
	}
	if out.Created {
		t.Fatalf("unresolvable source must be a no-op, got %+v", out)
	}
	// The session close itself still succeeds (the hook is best-effort).
	h, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := h.Memory().SessionEnd(context.Background(), sessionID, "done"); err != nil {
		t.Fatalf("session close broken by distill hook: %v", err)
	}
}

// TestAtomCorrectable covers 02.8 (Scenario:
// l1-atom-correctable-with-traceable-provenance). Correcting an L1 atom appends
// a version (step 01) rather than deleting the atom, and the provenance link
// still traces the atom back to its L0 source.
func TestAtomCorrectable(t *testing.T) {
	project := "atom"
	sessionID := "sess-atom"
	dataDir := openDistillStore(t, project, sessionID, distillable)
	svc := service.New(dataDir)
	if err := svc.EnableDistill(); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	out, err := svc.DistillSession(context.Background(), project, sessionID, "", &wg)
	if err != nil {
		t.Fatalf("distill: %v", err)
	}
	if out.L1Count == 0 {
		t.Fatalf("no L1 atom to correct (out=%+v)", out)
	}

	// Find the L1 atom's observation id + its provenance link.
	h, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var atomID int64
	if err := h.Store().DB.QueryRow(`
		SELECT target_id FROM observation_layers
		WHERE project = ? AND layer = 'L1' AND target_kind = 'observation'
		ORDER BY id LIMIT 1`, project).Scan(&atomID); err != nil {
		t.Fatalf("find atom: %v", err)
	}
	if atomID == 0 {
		t.Fatal("no L1 atom id")
	}

	// Correct the atom (append a version).
	beforeRev := 0
	if err := h.Store().DB.QueryRow(`SELECT COALESCE(revision_count,0) FROM observations WHERE id = ?`,
		atomID).Scan(&beforeRev); err != nil {
		t.Fatalf("read rev: %v", err)
	}
	if err := h.Memory().Update(context.Background(), atomID, memory.UpdateInput{Content: "corrected atom content"}); err != nil {
		t.Fatalf("correct atom: %v", err)
	}
	var afterRev int
	if err := h.Store().DB.QueryRow(`SELECT COALESCE(revision_count,0) FROM observations WHERE id = ?`,
		atomID).Scan(&afterRev); err != nil {
		t.Fatalf("read after rev: %v", err)
	}
	if afterRev <= beforeRev {
		t.Fatalf("correction must append a version: before=%d after=%d", beforeRev, afterRev)
	}

	// The atom still exists (not deleted) and the provenance link still traces
	// it back to its L0 source.
	var exists int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM observations WHERE id = ? AND deleted_at IS NULL`,
		atomID).Scan(&exists); err != nil || exists != 1 {
		t.Fatalf("atom must survive correction (exists=%d err=%v)", exists, err)
	}
	var linkSession string
	if err := h.Store().DB.QueryRow(`
		SELECT source_session FROM observation_layers
		WHERE project = ? AND layer = 'L1' AND target_id = ?`, project, atomID).Scan(&linkSession); err != nil {
		t.Fatalf("provenance link must survive correction: %v", err)
	}
	if linkSession != sessionID {
		t.Fatalf("provenance link = %q, want L0 %q", linkSession, sessionID)
	}
}

// TestMemLayers covers 02.9 (Scenario: mem-layers-inspects-l0-to-l3-chain) at
// the service seam: the Layers method returns the chain with each layer's
// provenance link.
func TestMemLayers(t *testing.T) {
	project := "memlayers"
	sessionID := "sess-memlayers"
	dataDir := openDistillStore(t, project, sessionID, distillable)
	svc := service.New(dataDir)
	if err := svc.EnableDistill(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	if _, err := svc.DistillSession(context.Background(), project, sessionID, "", &wg); err != nil {
		t.Fatalf("distill: %v", err)
	}

	chain, err := svc.Layers(context.Background(), project, sessionID)
	if err != nil {
		t.Fatalf("layers: %v", err)
	}
	if chain.L0Session != sessionID {
		t.Fatalf("L0 session = %q, want %q", chain.L0Session, sessionID)
	}
	if len(chain.Atoms) == 0 || len(chain.Scenarios) == 0 || len(chain.Persona) == 0 {
		t.Fatalf("incomplete chain: atoms=%d scen=%d persona=%d",
			len(chain.Atoms), len(chain.Scenarios), len(chain.Persona))
	}
	for _, l := range append(append(chain.Atoms, chain.Scenarios...), chain.Persona...) {
		if l.SourceSession != sessionID || l.SourceTopic == "" {
			t.Errorf("%s layer provenance broken: session=%q topic=%q", l.Layer, l.SourceSession, l.SourceTopic)
		}
	}
}
