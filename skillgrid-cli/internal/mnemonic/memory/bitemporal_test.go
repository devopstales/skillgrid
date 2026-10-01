package memory

import (
	"context"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// bitempFixture opens a fresh store + service pair on a t.TempDir() bucket.
// Each test gets an isolated bucket so bi-temporal column state never leaks
// between tests.
// mirroring the ownerFixture idiom (governance_test.go).
type bitempFixture struct {
	st        *store.Store
	svc       *Service
	owner     string
	sessionID string
}

func newBitempFixture(t *testing.T, project string) *bitempFixture {
	t.Helper()
	st, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	svc := New(st, project)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return &bitempFixture{st: st, svc: svc, owner: "bitemp-owner", sessionID: "s1"}
}

// TestSaveStampsBiTemporalColumns covers ADR-0011 stamping: a freshly saved
// observation is valid from creation (valid_at = created_at), has not become
// invalid (invalid_at empty), and is not superseded (superseded_by NULL).
func TestSaveStampsBiTemporalColumns(t *testing.T) {
	fx := newBitempFixture(t, "bitemp-stamp")
	ctx := context.Background()

	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID,
		Type:      "decision",
		Title:     "Bi-temporal stamping",
		Content:   "new rows are valid from creation",
		Owner:     fx.owner,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	obs, err := fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if obs.ValidAt != obs.CreatedAt {
		t.Errorf("valid_at = %q, want created_at %q", obs.ValidAt, obs.CreatedAt)
	}
	if obs.InvalidAt != "" {
		t.Errorf("invalid_at = %q, want empty (still true)", obs.InvalidAt)
	}
	if obs.SupersededBy.Valid {
		t.Errorf("superseded_by = %d, want NULL (not superseded)", obs.SupersededBy.Int64)
	}
}

// TestValidAtTimeWindow covers the "what was true at time T?" query (ADR-0011,
// C1.5): A is valid on [T1, T2), B on [T2, ∞). ValidAtTime(T1) → A only;
// ValidAtTime(T3) (T3 > T2) → B only.
func TestValidAtTimeWindow(t *testing.T) {
	fx := newBitempFixture(t, "bitemp-window")
	ctx := context.Background()

	t1 := "2026-01-01T00:00:00Z"
	t2 := "2026-06-01T00:00:00Z"
	t3 := "2026-12-01T00:00:00Z"

	// A: "windowedalpha" (valid T1..T2). B: "windowedbravo" (valid from T2).
	idA, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID,
		Type:      "learning",
		Title:     "WindowedAlpha",
		Content:   "windowedalpha old fact",
		Owner:     fx.owner,
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}
	idB, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID,
		Type:      "learning",
		Title:     "WindowedBravo",
		Content:   "windowedbravo current fact",
		Owner:     fx.owner,
	})
	if err != nil {
		t.Fatalf("save B: %v", err)
	}
	if _, err := fx.st.DB.Exec(`
		UPDATE observations SET valid_at = ?, invalid_at = ? WHERE id = ?`, t1, t2, idA); err != nil {
		t.Fatalf("seed A window: %v", err)
	}

	if _, err := fx.st.DB.Exec(`
		UPDATE observations SET valid_at = ?, invalid_at = ? WHERE id = ?`, t2, nil, idB); err != nil {
		t.Fatalf("seed B window: %v", err)
	}

	got, err := fx.svc.ValidAtTime(ctx, "windowedalpha windowedbravo", t1)
	if err != nil {
		t.Fatalf("valid_at_time T1: %v", err)
	}
	if len(got) != 1 || got[0].ID != idA {
		t.Errorf("ValidAtTime(T1) = %v, want A only", btIDs(got))
	}

	got, err = fx.svc.ValidAtTime(ctx, "windowedalpha windowedbravo", t3)
	if err != nil {
		t.Fatalf("valid_at_time T3: %v", err)
	}
	if len(got) != 1 || got[0].ID != idB {
		t.Errorf("ValidAtTime(T3) = %v, want B only", btIDs(got))
	}

	// At exactly T2 the window is half-open [valid_at, invalid_at):
	// A (valid [T1,T2)) is out, B (valid [T2,∞)) is in.
	got, err = fx.svc.ValidAtTime(ctx, "windowedalpha windowedbravo", t2)
	if err != nil {
		t.Fatalf("valid_at_time T2: %v", err)
	}
	if len(got) != 1 || got[0].ID != idB {
		t.Errorf("ValidAtTime(T2) = %v, want B only", btIDs(got))
	}
}

// TestSupersededExcludedFromRecent covers the soft-invalid exclusion on the
// recent-family read paths: a row whose invalid_at is in the past (superseded)
// is absent from Recent, RecentObservations, and RecentWithType, while the
// still-active row remains.
func TestSupersededExcludedFromRecent(t *testing.T) {
	fx := newBitempFixture(t, "bitemp-recent")
	ctx := context.Background()

	idA, err := fx.svc.Save(ctx, SaveInput{
		SessionID:  fx.sessionID,
		Type:       "preference",
		MemoryType: "preferences",
		Title:      "RecentAlpha",
		Content:    "recentfact old preference",
		Owner:      fx.owner,
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}
	idB, err := fx.svc.Save(ctx, SaveInput{
		SessionID:  fx.sessionID,
		Type:       "preference",
		MemoryType: "preferences",
		Title:      "RecentBravo",
		Content:    "recentfact current preference",
		Owner:      fx.owner,
	})
	if err != nil {
		t.Fatalf("save B: %v", err)
	}
	// Supersede A: its validity window closed in the past.
	if _, err := fx.st.DB.Exec(`
		UPDATE observations SET invalid_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), idA); err != nil {
		t.Fatalf("supersede A: %v", err)
	}

	type pathResult struct {
		name string
		obs  []Observation
	}
	pathObs := make([]pathResult, 0, 3)
	obs, err := fx.svc.Recent(ctx, 20)
	pathObs = append(pathObs, pathResult{"Recent", mustObs(t, obs, err)})
	obs, err = fx.svc.RecentObservations(ctx, 20)
	pathObs = append(pathObs, pathResult{"RecentObservations", mustObs(t, obs, err)})
	obs, err = fx.svc.RecentWithType(ctx, "preferences", 20)
	pathObs = append(pathObs, pathResult{"RecentWithType", mustObs(t, obs, err)})
	for _, p := range pathObs {
		if btContainsID(p.obs, idA) {
			t.Errorf("%s returned superseded row %d", p.name, idA)
		}
		if !btContainsID(p.obs, idB) {
			t.Errorf("%s missing active row %d", p.name, idB)
		}
	}
}

// TestSupersededExcludedFromSearch covers the soft-invalid exclusion on every
// live read path: a row whose invalid_at is in the past (superseded) is absent
// from SearchWithScope, SearchOwnerScoped, SearchOwner, and
// AdminCrossOwnerList, while the still-active row remains.
func TestSupersededExcludedFromSearch(t *testing.T) {
	fx := newBitempFixture(t, "bitemp-excl")
	ctx := context.Background()

	idA, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID,
		Type:      "preference",
		Title:     "SupersededAlpha",
		Content:   "supersededfact old preference",
		Owner:     fx.owner,
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}
	idB, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessionID,
		Type:      "preference",
		Title:     "ActiveBravo",
		Content:   "supersededfact current preference",
		Owner:     fx.owner,
	})
	if err != nil {
		t.Fatalf("save B: %v", err)
	}
	// Supersede A: its validity window closed in the past.
	if _, err := fx.st.DB.Exec(`
		UPDATE observations SET invalid_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), idA); err != nil {
		t.Fatalf("supersede A: %v", err)
	}

	type pathResult struct {
		name string
		obs  []Observation
	}
	pathObs := make([]pathResult, 0, 4)
	obs, err := fx.svc.SearchWithScope(ctx, "supersededfact", "any", "", 20)
	pathObs = append(pathObs, pathResult{"SearchWithScope", mustObs(t, obs, err)})
	obs, err = fx.svc.SearchOwnerScoped(ctx, fx.owner, "", "supersededfact", "any", "", 20)
	pathObs = append(pathObs, pathResult{"SearchOwnerScoped", mustObs(t, obs, err)})
	obs, err = fx.svc.SearchOwner(ctx, fx.owner, "supersededfact", "any", 20)
	pathObs = append(pathObs, pathResult{"SearchOwner", mustObs(t, obs, err)})
	obs, err = fx.svc.AdminCrossOwnerList(ctx, fx.owner)
	pathObs = append(pathObs, pathResult{"AdminCrossOwnerList", mustObs(t, obs, err)})
	for _, p := range pathObs {
		if btContainsID(p.obs, idA) {
			t.Errorf("%s returned superseded row %d", p.name, idA)
		}
		if !btContainsID(p.obs, idB) {
			t.Errorf("%s missing active row %d", p.name, idB)
		}
	}
}

func mustObs(t *testing.T, obs []Observation, err error) []Observation {
	t.Helper()
	if err != nil {
		t.Fatalf("read path: %v", err)
	}
	return obs
}

// btIDs extracts the ids of observations (namespaced to avoid colliding with
// idsOf in dream_rollback.go).
func btIDs(obs []Observation) []int64 {
	var out []int64
	for _, o := range obs {
		out = append(out, o.ID)
	}
	return out
}

// btContainsID reports whether id is present in the observations.
func btContainsID(obs []Observation, id int64) bool {
	for _, o := range obs {
		if o.ID == id {
			return true
		}
	}
	return false
}
