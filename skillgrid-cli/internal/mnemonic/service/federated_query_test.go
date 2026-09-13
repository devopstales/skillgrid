package service

import (
	"context"
	"math"
	"sort"
	"strconv"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// federatedQueryTestConfig is the weights used by the federated-query tests:
// the default 0.5/0.5 (change 014, step 16).
var federatedQueryTestConfig = FederatedConfig{RankWeight: 0.5, ImportanceWeight: 0.5}

// setFederatedForTest pins the federated merge weights on svc for a test and
// restores the prior state on cleanup.
func setFederatedForTest(t *testing.T, svc *Service, cfg FederatedConfig) {
	t.Helper()
	svc.federatedMu.Lock()
	prev := svc.federatedWeights
	svc.federatedWeights = cfg
	svc.federatedMu.Unlock()
	t.Cleanup(func() {
		svc.federatedMu.Lock()
		svc.federatedWeights = prev
		svc.federatedMu.Unlock()
	})
}

// seedFederatedObs seeds one observation into projectID with a raw importance
// score (the value written to the importance_score column by the test seam)
// and returns its ID. The title is what the FTS query matches; the content
// carries distinct filler so bm25 ties break deterministically.
func seedFederatedObs(t *testing.T, svc *Service, projectID, title, content string, rawScore float64) int64 {
	t.Helper()
	h, cleanup, err := svc.openProject(projectID, ".")
	if err != nil {
		t.Fatalf("open %s: %v", projectID, err)
	}
	defer cleanup()
	prev := h.memory.TestRawImportance
	h.memory.TestRawImportance = rawScore
	sid, err := h.memory.SessionStart(context.Background(), t.TempDir(), "federated")
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	id, err := h.memory.Save(context.Background(), memory.SaveInput{
		Title: title, Type: "decision", Content: content, SessionID: sid, Scope: "project",
	})
	h.memory.TestRawImportance = prev
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	return id
}

// TestFederatedQueryImportanceRanking covers 16.1: the federated merge ranks
// by a composite score combining the cross-store rank and the per-observation
// importance score. With three stores whose observations have varying
// importance, the observation with the highest composite score must rank
// first — and a deeper hit with high importance must outrank a #1 hit with
// low importance (something a pure rank-merge would not do).
func TestFederatedQueryImportanceRanking(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	const query = "federated importance marker"

	// store-a: obs a1 (high importance) is the #2 hit in store-a; a2 (low)
	// repeats the query token twice so it leads store-a's raw bm25 ranking.
	seedFederatedObs(t, svc, "fed-a", query+" one", query+" filler body one", 9.0)
	seedFederatedObs(t, svc, "fed-a", query+" two", query+" "+query+" filler body two", 1.0)
	// store-b: b1 medium importance, single hit.
	seedFederatedObs(t, svc, "fed-b", query+" three", query+" filler body three", 5.0)
	// store-c: c1 low importance, single hit.
	seedFederatedObs(t, svc, "fed-c", query+" four", query+" filler body four", 2.0)

	// Setup check: within store-a the low-importance doc leads the raw SQL
	// ranking (more token matches), so a rank-only merge would put it ahead
	// of a1.
	scoped, err := svc.SearchObservationsScoped(context.Background(), "fed-a", query, "any", "", 10)
	if err != nil {
		t.Fatalf("scoped store-a: %v", err)
	}
	if len(scoped) != 2 || scoped[0].Title != query+" two" {
		t.Fatalf("setup: store-a must have 2 hits led by the repeated-token doc, got %d", len(scoped))
	}

	// The federated merge must compute (maxImportance=9, 0.5/0.5 weights):
	//   b1: 0.5*(1/1) + 0.5*(5/9) = 0.778  <- highest composite, ranks first
	//   a1: 0.5*(1/2) + 0.5*(9/9) = 0.750  (rank 1 in store-a, imp 9)
	//   c1: 0.5*(1/1) + 0.5*(2/9) = 0.611  (rank 0 in store-c, imp 2)
	//   a2: 0.5*(1/1) + 0.5*(1/9) = 0.556  (rank 0 in store-a, imp 1)
	// The discriminating assertion: a1 (rank 1, imp 9) must outrank c1
	// (rank 0, imp 2). A pure cross-store rank merge (step 03) would rank c1
	// above a1; only the importance term flips it.
	setFederatedForTest(t, svc, federatedQueryTestConfig)
	hits, err := svc.SearchObservationsAll(context.Background(), query, "any", "", 20)
	if err != nil {
		t.Fatalf("federated search: %v", err)
	}
	if len(hits) != 4 {
		t.Fatalf("want 4 hits, got %d", len(hits))
	}
	if hits[0].Title != query+" three" {
		t.Fatalf("highest composite (store-b) must rank first, got order %v", titles(hits))
	}
	iA1, iC1 := indexTitle(hits, query+" one"), indexTitle(hits, query+" four")
	if iA1 < 0 || iC1 < 0 {
		t.Fatalf("both a1 and c1 must be present, got %v", titles(hits))
	}
	if iA1 > iC1 {
		t.Fatalf("composite merge must rank a1 (imp 9, rank 1) above c1 (imp 2, rank 0); got order %v", titles(hits))
	}
	// a2 (lowest composite, 0.556) must rank last.
	if iA2 := indexTitle(hits, query+" two"); iA2 != len(hits)-1 {
		t.Fatalf("lowest composite a2 must rank last, got order %v", titles(hits))
	}
}

// TestFederatedQueryDedup covers 16.2: the federated merge maintains a seen
// map keyed by observation identity (ID + project — each store has its own
// autoincrement ID space, so the project disambiguates the ID); when the same
// observation is emitted into the merge more than once, it appears exactly
// once and the higher-composite copy is kept.
func TestFederatedQueryDedup(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	const query = "federated dedup marker"

	// Seed distinct content in each store so their autoincrement ID spaces
	// stay independent (a store's first save is ID 1 in that store). Each
	// store gets TWO matching observations so the scoped search returns >=2.
	seedFederatedObs(t, svc, "dedup-d1", query+" one", query+" seed body one", 0.5)
	seedFederatedObs(t, svc, "dedup-d1", query+" extra", query+" seed body extra", 0.5)
	seedFederatedObs(t, svc, "dedup-d2", query+" two", query+" seed body two", 0.5)
	seedFederatedObs(t, svc, "dedup-d2", query+" extra", query+" seed body extra2", 0.5)

	// Both stores now have the same first-observation ID (1) with different
	// content — the merge must keep BOTH (distinct observations, ID space is
	// per-store; the project in the key prevents a false cross-store collapse
	// — the step-03 contract, unchanged).
	hits, err := svc.SearchObservationsAll(context.Background(), query, "any", "", 20)
	if err != nil {
		t.Fatalf("federated search (baseline): %v", err)
	}
	if indexTitle(hits, query+" one") < 0 || indexTitle(hits, query+" two") < 0 {
		t.Fatalf("distinct same-ID observations from different stores must both be kept, got %v", titles(hits))
	}

	// Now create a genuine duplicate: the same observation (same ID 1, same
	// project dedup-d1) emitted into the merge twice. The store-level 24h
	// dedup makes a re-save return the same ID, so the scoped search returns
	// one row — but the merge's seen map is the dedup under test: feed the
	// same storeRanked twice through mergeRanked and assert one survivor,
	// with the higher-composite copy kept.
	h, cleanup, err := svc.openProject("dedup-d1", ".")
	if err != nil {
		t.Fatalf("open dedup-d1: %v", err)
	}
	defer cleanup()
	res, err := h.memory.SearchWithScope(context.Background(), query, "any", "", 20)
	if err != nil {
		t.Fatalf("scoped dedup-d1: %v", err)
	}
	if len(res) < 2 {
		t.Fatalf("setup: dedup-d1 must have >=2 hits, got %d", len(res))
	}
	ranked := rankHits(res)
	// Emit the same store's ranked results twice (the duplicate-emit case the
	// seen map guards against): the first copy at rank i, the duplicate at
	// rank i+2 (a WORSE in-store rank → lower composite).
	dup := make([]storeRanked, 0, len(ranked)*2)
	for i, r := range ranked {
		dup = append(dup, r)
		dup = append(dup, storeRanked{obs: r.obs, rank: i + 2})
	}
	merged := mergeRanked(dup, 20, federatedQueryTestConfig)
	// Each observation must survive exactly once (the duplicate emit
	// collapsed).
	seenCount := map[string]int{}
	for _, o := range merged {
		seenCount[strconv.FormatInt(o.ID, 10)+"/"+o.Project]++
	}
	for k, n := range seenCount {
		if n != 1 {
			t.Fatalf("observation %s must appear exactly once after dedup, got %d", k, n)
		}
	}
	// The kept copy must be the higher-composite one: recompute the composite
	// ordering of the ORIGINAL ranks — the merged output must match it
	// exactly. If the degraded rank+2 duplicates had won, the ordering (and
	// thus the survivor set's order) would differ.
	maxImp := 0.0
	imps := make([]float64, len(ranked))
	for i, r := range ranked {
		imps[i] = observedImportanceOf(r.obs)
		if imps[i] > maxImp {
			maxImp = imps[i]
		}
	}
	type want struct {
		id  int64
		prj string
	}
	wantOrder := make([]want, len(ranked))
	pos := make([]int, len(ranked))
	for i := range ranked {
		pos[i] = i
	}
	sort.SliceStable(pos, func(a, b int) bool {
		ca := federatedComposite(ranked[pos[a]].rank, imps[pos[a]], maxImp, federatedQueryTestConfig)
		cb := federatedComposite(ranked[pos[b]].rank, imps[pos[b]], maxImp, federatedQueryTestConfig)
		if ca != cb {
			return ca > cb
		}
		return ranked[pos[a]].obs.UpdatedAt > ranked[pos[b]].obs.UpdatedAt
	})
	for i, p := range pos {
		wantOrder[i] = want{id: ranked[p].obs.ID, prj: ranked[p].obs.Project}
	}
	if len(merged) != len(wantOrder) {
		t.Fatalf("merged count=%d, want %d", len(merged), len(wantOrder))
	}
	for i, o := range merged {
		if o.ID != wantOrder[i].id || o.Project != wantOrder[i].prj {
			t.Fatalf("position %d: kept obs %d/%s, want %d/%s — dedup must keep the higher-composite copy", i, o.ID, o.Project, wantOrder[i].id, wantOrder[i].prj)
		}
	}
}

// TestFederatedQueryRespectsProjectImportance covers 16.3: each store's
// observations carry their LOCAL (per-project) importance score into the
// federated merge, and the composite balances FTS rank against it. Store A's
// observation has much higher importance (9.0) than store B's (1.0); even
// though store B's observation leads its own store's FTS ranking, the
// per-project importance applied before the merge must carry store A's
// observation to the top of the federated result.
func TestFederatedQueryRespectsProjectImportance(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	const query = "federated project importance"

	// Store A: a1 importance 9.0, a single hit (rank 0).
	seedFederatedObs(t, svc, "fedpi-a", query+" alpha", query+" filler body alpha", 9.0)
	// Store B: b1 importance 1.0 but a strong FTS hit (repeated token).
	seedFederatedObs(t, svc, "fedpi-b", query+" beta", query+" "+query+" filler body beta", 1.0)

	setFederatedForTest(t, svc, federatedQueryTestConfig)
	hits, err := svc.SearchObservationsAll(context.Background(), query, "any", "", 20)
	if err != nil {
		t.Fatalf("federated search: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("want 2 hits, got %d", len(hits))
	}
	iA, iB := indexTitle(hits, query+" alpha"), indexTitle(hits, query+" beta")
	if iA < 0 || iB < 0 {
		t.Fatalf("both observations must be present, got %v", titles(hits))
	}
	// Per-project importance (step 13) is applied before the merge: store A's
	// importance (9.0) dominates at the 0.5/0.5 weights, so A ranks first
	// despite store B's stronger FTS signal.
	if iA > iB {
		t.Fatalf("per-project importance must be applied before the merge: store A (imp 9.0) must rank above store B (imp 1.0, stronger FTS); got order %v", titles(hits))
	}

	// Sanity on the composite values themselves: the formula must produce
	// exactly the documented numbers for this fixture (both rank 0,
	// maxImportance 9):
	//   A: 0.5*(1/1) + 0.5*(9/9) = 1.000
	//   B: 0.5*(1/1) + 0.5*(1/9) = 0.5556
	ca := federatedComposite(0, 9.0, 9.0, federatedQueryTestConfig)
	cb := federatedComposite(0, 1.0, 9.0, federatedQueryTestConfig)
	if math.Abs(ca-1.0) > 1e-9 {
		t.Fatalf("composite for store A must be 1.0 (0.5*1 + 0.5*9/9), got %v", ca)
	}
	if math.Abs(cb-(0.5+0.5/9.0)) > 1e-9 {
		t.Fatalf("composite for store B must be ≈0.5556 (0.5*1 + 0.5*1/9), got %v", cb)
	}
	if !(ca > cb) {
		t.Fatalf("store A composite (%v) must exceed store B (%v)", ca, cb)
	}
}

// titles returns the observation titles of a result set (for assertions).
func titles(obs []memory.Observation) []string {
	out := make([]string, len(obs))
	for i, o := range obs {
		out[i] = o.Title
	}
	return out
}

func containsTitle(obs []memory.Observation, title string) bool {
	return indexTitle(obs, title) >= 0
}

func indexTitle(obs []memory.Observation, title string) int {
	for i, o := range obs {
		if o.Title == title {
			return i
		}
	}
	return -1
}
