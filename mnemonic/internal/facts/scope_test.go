// Scope routing + two-target category (change 2026-10-24-global-llm-wiki,
// Change A, TICKET-02).
//
// RED notes (the failures that prove each test exercises new code):
//   - TestScopeValues / TestRouteRejectsUnknownScope / TestRouteScopeRouting /
//     TestRouteRejectsWritesToScopeAll: compile error — the facts.Scope type,
//     its constants, and facts.Route do not exist yet.
//   - TestGlobalAddWritesGlobalStoreProjectTrail (BDD: "happy path — global
//     fact add writes to global store with project trail") and
//     TestGlobalAddUserTarget (BDD: "edge — global fact add with target user
//     uses user category"): compile error — facts.Store has no category
//     awareness, and Add takes no trail store, so a fact added through the
//     global store cannot trail into the caller's project store.
//   - TestGlobalAddUnknownSessionFails (BDD: "failure — global fact add with
//     unknown session fails cleanly"): the pre-existing FK rollback covers the
//     same-store case; here the fact must also be absent from the GLOBAL store
//     while the error names the missing project session.
package facts

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// ctx is the shared context for scope tests (every op is synchronous).
var ctx = context.Background()

// TestScopeValues pins the three accepted scope spellings and rejects the
// rest (the MCP layer maps its scope param onto this validation).
func TestScopeValues(t *testing.T) {
	for _, valid := range []Scope{ScopeProject, ScopeGlobal, ScopeAll} {
		if !valid.Valid() {
			t.Errorf("Scope(%q).Valid() = false, want true", valid)
		}
	}
	for _, invalid := range []Scope{"", "memory", "user", "GLOBAL", "All", "global-all"} {
		if invalid.Valid() {
			t.Errorf("Scope(%q).Valid() = true, want false", invalid)
		}
	}
}

// TestRouteScopeRouting (BDD: "happy path — project fact add writes to project
// store (unchanged)"): scope=project routes both stores to the project store
// (the pre-existing behavior, unchanged); scope=global routes the data store
// to __global__ while the trail store stays the caller's project store (C3).
func TestRouteScopeRouting(t *testing.T) {
	dataDir := t.TempDir()
	pst := openTestStore(t) // project store (id "factstest")
	gst, err := store.Open(dataDir, GlobalProjectID)
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	t.Cleanup(func() { gst.Close() })

	t.Run("project routes both stores to the project store", func(t *testing.T) {
		data, trail, err := Route(ScopeProject, pst.DB, pst.DB)
		if err != nil {
			t.Fatalf("Route project: %v", err)
		}
		if data != trail {
			t.Error("scope=project must return the same store for data and trail (pre-change behavior)")
		}
		if data.DB() != pst.DB {
			t.Error("data store is not the project store")
		}
	})

	t.Run("global routes data to __global__, trail to the project store", func(t *testing.T) {
		data, trail, err := Route(ScopeGlobal, gst.DB, pst.DB)
		if err != nil {
			t.Fatalf("Route global: %v", err)
		}
		if data.DB() != gst.DB {
			t.Error("data store is not the global store")
		}
		if trail.DB() != pst.DB {
			t.Error("trail store is not the project store (C3)")
		}
		if data.DB() == trail.DB() {
			t.Error("data and trail must be distinct stores for scope=global")
		}
	})
}

// TestRouteRejectsWritesToScopeAll: scope=all is a read-only scope (fused
// search, Task 3) — a write op must fail cleanly rather than pick a store.
func TestRouteRejectsWritesToScopeAll(t *testing.T) {
	if _, _, err := Route(ScopeAll, nil, nil); err == nil {
		t.Fatal("Route with scope=all should fail for write ops (fused search lands in Task 3)")
	}
}

// TestRouteRejectsUnknownScope is the routing boundary: an unrecognized scope
// (anything outside project|global|all) is a clean error before any store is
// used.
func TestRouteRejectsUnknownScope(t *testing.T) {
	if _, _, err := Route(Scope("bogus"), nil, nil); err == nil {
		t.Fatal("Route with unknown scope should fail")
	}
}

// openTwoStores opens a project store plus the __global__ store in one temp
// data dir (the two-store fixture the C3 tests share).
func openTwoStores(t *testing.T) (project, global *store.Store, dataDir string) {
	t.Helper()
	dataDir = t.TempDir()
	project = openTestStore(t)
	global, err := store.Open(dataDir, GlobalProjectID)
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	t.Cleanup(func() { global.Close() })
	return project, global, dataDir
}

// TestGlobalAddWritesGlobalStoreProjectTrail (BDD: "happy path — global fact
// add writes to global store with project trail"): a fact added with
// scope=global, target=memory lands in __global__.sqlite with
// category='memory' (G1), and its session_events trail lands in the caller's
// project store — NOT in the global store (G2).
func TestGlobalAddWritesGlobalStoreProjectTrail(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	data, trail, err := Route(ScopeGlobal, gst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route global: %v", err)
	}

	const sid = "sess-global-add"
	seedTestSession(t, pst, sid)

	id, err := data.AddWith(ctx, sid, "global operational note: always gofmt before commit", "memory", trail)
	if err != nil {
		t.Fatalf("AddWith global: %v", err)
	}

	// G1: the fact row exists in the global store with category='memory'.
	var content, category string
	if err := gst.DB.QueryRow(
		`SELECT content, category FROM facts WHERE id = ?`, id,
	).Scan(&content, &category); err != nil {
		t.Fatalf("fact missing from the global store: %v", err)
	}
	if content != "global operational note: always gofmt before commit" {
		t.Errorf("fact content = %q", content)
	}
	if category != "memory" {
		t.Errorf("global fact category = %q, want 'memory'", category)
	}

	// G2: the trail row is in the PROJECT store, carrying the target category.
	// (The cross-store trail payload does not carry the fact id: the id is
	// allocated by the global store and the trail row is a session audit
	// entry — see AddWith's cross-store contract.)
	var actionType, payload string
	if err := pst.DB.QueryRow(`
		SELECT action_type, payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_add'`, sid).
		Scan(&actionType, &payload); err != nil {
		t.Fatalf("fact_add trail missing from the project store: %v", err)
	}
	if !strings.Contains(payload, `"category":"memory"`) {
		t.Errorf("trail payload %q does not carry the target category", payload)
	}

	// G2 (negative half): the global store holds NO session_events trail.
	var globalTrails int
	if err := gst.DB.QueryRow(
		`SELECT COUNT(*) FROM session_events WHERE action_type = 'fact_add'`,
	).Scan(&globalTrails); err != nil {
		t.Fatalf("count global trails: %v", err)
	}
	if globalTrails != 0 {
		t.Errorf("global store holds %d fact_add trail rows, want 0 (trail belongs to the project store)", globalTrails)
	}
}

// TestProjectAddWritesProjectStoreOnly (BDD: "happy path — project fact add
// writes to project store (unchanged)"): scope=project leaves the fact — and
// the trail — in the project store, with no row in the global store (G4).
func TestProjectAddWritesProjectStoreOnly(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	data, trail, err := Route(ScopeProject, pst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route project: %v", err)
	}

	const sid = "sess-proj-add"
	seedTestSession(t, pst, sid)

	id, err := data.AddWith(ctx, sid, "project-scoped note stays in the project store", "", trail)
	if err != nil {
		t.Fatalf("AddWith project: %v", err)
	}

	var category string
	if err := pst.DB.QueryRow(
		`SELECT category FROM facts WHERE id = ?`, id,
	).Scan(&category); err != nil {
		t.Fatalf("fact missing from the project store: %v", err)
	}
	// target omitted (blank) → the schema default 'memory'.
	if category != "memory" {
		t.Errorf("project fact category = %q, want the default 'memory'", category)
	}

	// The trail is in the project store (same-store behavior, unchanged).
	var eventsN int
	if err := pst.DB.QueryRow(`
		SELECT COUNT(*) FROM session_events
		WHERE session_id = ? AND action_type = 'fact_add'`, sid).Scan(&eventsN); err != nil {
		t.Fatalf("count project events: %v", err)
	}
	if eventsN != 1 {
		t.Errorf("project store holds %d fact_add events, want 1", eventsN)
	}

	// G4: no fact row in the global store.
	var globalFacts int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&globalFacts); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if globalFacts != 0 {
		t.Errorf("global store holds %d facts, want 0 for a project-scoped add", globalFacts)
	}
}

// TestGlobalAddUserTarget (BDD: "edge — global fact add with target user uses
// user category"): target=user stamps category='user' in the global store (G3).
// ponytail: the per-category char budgets (memory 2200 / user 1375) are
// enforced by the batch ops ticket (TICKET-04) — the category routing that
// makes the right budget applicable is what this ticket owns.
func TestGlobalAddUserTarget(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	data, trail, err := Route(ScopeGlobal, gst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route global: %v", err)
	}

	const sid = "sess-global-user"
	seedTestSession(t, pst, sid)

	id, err := data.AddWith(ctx, sid, "user profile note: prefers concise answers", "user", trail)
	if err != nil {
		t.Fatalf("AddWith user: %v", err)
	}

	var category string
	if err := gst.DB.QueryRow(
		`SELECT category FROM facts WHERE id = ?`, id,
	).Scan(&category); err != nil {
		t.Fatalf("fact missing from the global store: %v", err)
	}
	if category != "user" {
		t.Errorf("global user-target fact category = %q, want 'user'", category)
	}
}

// TestGlobalAddUnknownSessionFails (BDD: "failure — global fact add with
// unknown session fails cleanly"): the session lives in the PROJECT store
// (where the trail is written), so an unknown session must roll the whole
// write back — no fact row in the global store (G5) — and the error must name
// the missing session.
func TestGlobalAddUnknownSessionFails(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	data, trail, err := Route(ScopeGlobal, gst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route global: %v", err)
	}

	// A seeded session (must NOT be the one used below), proving the failure
	// is specific to the unknown id, not to an empty project store.
	seedTestSession(t, pst, "sess-known")

	_, err = data.AddWith(ctx, "sess-unknown", "fact that must not land in the global store", "memory", trail)
	if err == nil {
		t.Fatal("AddWith with an unknown project session should fail")
	}
	if !strings.Contains(err.Error(), "sess-unknown") {
		t.Errorf("error %q does not name the missing session", err)
	}

	var globalFacts, projectEvents int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&globalFacts); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if globalFacts != 0 {
		t.Errorf("global store holds %d facts after a failed add, want 0 (no partial write)", globalFacts)
	}
	if err := pst.DB.QueryRow(`
		SELECT COUNT(*) FROM session_events WHERE action_type = 'fact_add'`).Scan(&projectEvents); err != nil {
		t.Fatalf("count project events: %v", err)
	}
	if projectEvents != 0 {
		t.Errorf("project store holds %d fact_add events after a failed add, want 0", projectEvents)
	}
}

// TestAddRejectsUnknownCategory: a target outside memory|user is a clean
// error before any write.
func TestAddRejectsUnknownCategory(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB, "factstest")
	if _, err := s.AddWith(ctx, "sess-any", "content", "world", nil); err == nil {
		t.Fatal("AddWith with an unknown category should fail")
	}
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&n); err != nil {
		t.Fatalf("count facts: %v", err)
	}
	if n != 0 {
		t.Errorf("facts rows = %d, want 0 after a rejected category", n)
	}
}

// TestGlobalAddTrailSequenceIsMonotonic (triangulation of the trail seam):
// two global adds from the same project session must produce trail rows with
// monotonic sequences in the PROJECT store, even though the fact rows live in
// the GLOBAL store — the sequence must track the session's project-store
// events, not the global store's (which never see the session).
func TestGlobalAddTrailSequenceIsMonotonic(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	data, trail, err := Route(ScopeGlobal, gst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route global: %v", err)
	}

	const sid = "sess-seq"
	seedTestSession(t, pst, sid)

	if _, err := data.AddWith(ctx, sid, "first global note", "memory", trail); err != nil {
		t.Fatalf("first AddWith: %v", err)
	}
	if _, err := data.AddWith(ctx, sid, "second global note", "memory", trail); err != nil {
		t.Fatalf("second AddWith: %v", err)
	}

	var seqs []int
	rows, err := pst.DB.Query(`
		SELECT sequence FROM session_events
		WHERE session_id = ? AND action_type = 'fact_add'
		ORDER BY sequence`, sid)
	if err != nil {
		t.Fatalf("query trail sequences: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var seq int
		if err := rows.Scan(&seq); err != nil {
			t.Fatalf("scan sequence: %v", err)
		}
		seqs = append(seqs, seq)
	}
	if len(seqs) != 2 {
		t.Fatalf("trail rows = %d, want 2", len(seqs))
	}
	if seqs[0] >= seqs[1] {
		t.Errorf("trail sequences not monotonic: %v", seqs)
	}
}

// TestGlobalStoreFilePinned (door check, TICKET-01 seam): the __global__
// store opens at exactly <dataDir>/__global__.sqlite (no id mangling).
func TestGlobalStoreFilePinned(t *testing.T) {
	_, gst, dataDir := openTwoStores(t)
	if filepath.Base(gst.Path()) != GlobalProjectID+".sqlite" {
		t.Fatalf("global store file = %q, want __global__.sqlite", filepath.Base(gst.Path()))
	}
	if _, err := gst.DB.Exec(`SELECT 1`); err != nil {
		t.Fatalf("global store unusable: %v", err)
	}
	_ = dataDir
}
