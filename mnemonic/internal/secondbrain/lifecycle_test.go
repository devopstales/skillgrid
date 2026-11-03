package secondbrain

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// lifecycle_test.go — TICKET-05 (2026-09-30-mnemonic-second-brain Task 6):
// mem_lifecycle health / dedup / consolidate / archive + the lifecycle_log
// audit trail. RED-first: these tests target symbols in lifecycle.go that do
// not exist yet.

// withHealthProbe installs the injectable hook Health consults (declared in
// lifecycle.go) so a test can force the "computation errors / panics" path
// without a broken store.
func withHealthProbe(t *testing.T, fn func() (HealthReport, error)) {
	t.Helper()
	prev := healthProbe
	healthProbe = fn
	t.Cleanup(func() { healthProbe = prev })
}

// seedOne stores a single observation and returns its id.
func seedOne(t *testing.T, svc *service.Service, project string) int64 {
	t.Helper()
	ensureSession(t, svc, project)
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	id, err := hh.Memory().Save(context.Background(), memory.SaveInput{
		Title: "seed note", Type: "decision", Content: "seed content",
		Scope: "project", SessionID: sessionA,
	})
	if err != nil {
		t.Fatalf("seed save: %v", err)
	}
	return id
}

// seedNearDupePair stores two observations whose content is token-identical,
// embeds both with the near-dupe hash embedder, and returns (canonical,
// duplicate) ids. Token-identical content → identical vectors → cosine 1.0 >
// threshold; distinct titles keep them as separate rows.
func seedNearDupePair(t *testing.T, svc *service.Service, project string) (int64, int64) {
	t.Helper()
	ensureSession(t, svc, project)
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	content := "near dupe payload pair words identical"
	keep, err := hh.Memory().Save(context.Background(), memory.SaveInput{
		Title: "near dupe canonical", Type: "decision",
		Content: content, Scope: "project", SessionID: sessionA,
	})
	if err != nil {
		t.Fatalf("seed canonical: %v", err)
	}
	writeNearDupeEmbedding(t, hh, project, keep, content)
	dup, err := hh.Memory().Save(context.Background(), memory.SaveInput{
		Title: "near dupe copy", Type: "decision",
		Content: content, Scope: "project", SessionID: sessionA,
	})
	if err != nil {
		t.Fatalf("seed duplicate: %v", err)
	}
	writeNearDupeEmbedding(t, hh, project, dup, content)
	return keep, dup
}

// seedNearDupes stores n duplicate pairs (2n token-identical observations) and
// embeds every one with the shared hash embedder so cosine is well-defined.
func seedNearDupes(t *testing.T, svc *service.Service, project string, pairs int) {
	t.Helper()
	ensureSession(t, svc, project)
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	for p := 0; p < pairs; p++ {
		content := fmt.Sprintf("near dupe payload %d identical words", p)
		// One canonical (unique title) + N-1 near-duplicates: same content
		// (identical token hash → cosine ~1.0 > threshold) but a unique title
		// per copy so Save's 24h (title+content+type) dedup keeps them as
		// distinct rows.
		id, err := hh.Memory().Save(context.Background(), memory.SaveInput{
			Title:   fmt.Sprintf("near dupe %d canonical", p),
			Type:    "decision",
			Content: content,
			Scope:   "project", SessionID: sessionA,
		})
		if err != nil {
			t.Fatalf("seed near-dupe save: %v", err)
		}
		writeNearDupeEmbedding(t, hh, project, id, content)
		for c := 1; c < pairs; c++ {
			id, err := hh.Memory().Save(context.Background(), memory.SaveInput{
				Title:   fmt.Sprintf("near dupe %d copy %d", p, c),
				Type:    "decision",
				Content: content,
				Scope:   "project", SessionID: sessionA,
			})
			if err != nil {
				t.Fatalf("seed near-dupe save: %v", err)
			}
			writeNearDupeEmbedding(t, hh, project, id, content)
		}
	}
}

// writeNearDupeEmbedding stores a deterministic token-hash vector for the
// observation's content: identical content → identical vector (cosine 1.0).
// The dim is small enough that distinct contents stay well under the 0.85
// dedup threshold (a random 64-dim token-hash cosine is ~0.12).
func writeNearDupeEmbedding(t *testing.T, hh *service.ProjectHandle, project string, id int64, content string) {
	t.Helper()
	vec := nearDupeVec(content)
	if err := hh.Memory().SetEmbedding(context.Background(), id, memory.EncodeVector(vec), "near-dupe-hash-v1"); err != nil {
		t.Fatalf("set embedding: %v", err)
	}
}

// nearDupeVec is a deterministic L2-normalized token-hash vector (64 dims,
// no query prefix) — the near-duplicate embedder for the dedup tests.
func nearDupeVec(text string) memory.Vector {
	dim := 64
	vec := make([]float32, dim)
	for _, tok := range strings.Fields(strings.ToLower(text)) {
		fh := fnv.New32a()
		_, _ = fh.Write([]byte(tok))
		vec[int(fh.Sum32()%uint32(dim))]++
	}
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		inv := float32(1 / math.Sqrt(norm))
		for i := range vec {
			vec[i] *= inv
		}
	}
	return memory.Vector{Data: vec}
}

// isArchived reports whether an observation is soft-archived (the 043 pair).
func isArchived(t *testing.T, svc *service.Service, project string, id int64) bool {
	t.Helper()
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	var n int
	if err := hh.Store().DB.QueryRow(
		`SELECT COUNT(*) FROM observations WHERE id = ? AND project = ? AND COALESCE(archived_at,'') != ''`,
		id, project,
	).Scan(&n); err != nil {
		t.Fatalf("archived check: %v", err)
	}
	return n > 0
}

// hasConsolidatedFrom reports whether the canonical observation's metadata
// carries consolidated_from provenance naming the merged duplicate id.
func hasConsolidatedFrom(t *testing.T, svc *service.Service, project string, canonical int64, dup int64) bool {
	t.Helper()
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	var source string
	if err := hh.Store().DB.QueryRow(
		`SELECT COALESCE(source,'') FROM observations WHERE id = ? AND project = ?`,
		canonical, project,
	).Scan(&source); err != nil {
		t.Fatalf("source read: %v", err)
	}
	return strings.Contains(source, "consolidated_from") && strings.Contains(source, fmt.Sprintf("%d", dup))
}

// hasAuditRow reports whether a completed lifecycle_log row exists for the
// action with the given input id and a non-empty completed_at.
func hasAuditRow(t *testing.T, svc *service.Service, project string, action string, status string, id int64) bool {
	t.Helper()
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	var inputIDs, completedAt string
	err = hh.Store().DB.QueryRow(
		`SELECT input_ids, completed_at FROM lifecycle_log
		 WHERE project = ? AND action = ? AND status = ?
		   AND input_ids LIKE '%' || ? || '%'
		 ORDER BY id DESC LIMIT 1`,
		project, action, status, id,
	).Scan(&inputIDs, &completedAt)
	if err != nil {
		t.Fatalf("audit row read: %v", err)
	}
	return inputIDs != "" && completedAt != ""
}

// --- Health ---

func TestLifecycle_HealthReport(t *testing.T) {
	svc := newTestService(t, false)
	// Unique project name so a stale 24h health cache file from a prior run /
	// sibling (shared $TMPDIR/mnemonic-lifecycle-health, keyed by sha256(project))
	// can never be served in place of a fresh computation. Mirrors
	// TestLifecycle_HealthCached, which documents the same isolation.
	project := fmt.Sprintf("lifecycle-health-report-%d", time.Now().UnixNano())
	seedObservations(t, svc, project, 20, "auth")
	defer os.RemoveAll(filepath.Join(os.TempDir(), healthCacheDir))

	rep, err := Health(context.Background(), svc, project)
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if len(rep.CountsByType) == 0 {
		t.Error("expected counts by type")
	}
	if rep.AgeDistribution == nil {
		t.Error("expected age distribution")
	}
	if rep.DuplicateDensity < 0 {
		t.Errorf("expected a duplicate density value, got %v", rep.DuplicateDensity)
	}
	if rep.Recommendations == nil {
		t.Error("expected a (possibly empty) recommendations slice")
	}
}

// TestLifecycle_HealthNeverThrows locks the recover contract: a computation
// that panics (or errors) yields an empty report, never a thrown panic/error.
// The project name is unique per phase so the 24h file cache (shared by the
// health probe, not by this test's service) never masks a recomputation.
func TestLifecycle_HealthNeverThrows(t *testing.T) {
	ctx := context.Background()
	for i, fn := range []func() (HealthReport, error){
		func() (HealthReport, error) { panic("boom") },
		func() (HealthReport, error) { return HealthReport{}, fmt.Errorf("db down") },
	} {
		svc := newTestService(t, false)
		seedObservations(t, svc, "test-project", 5, "auth")
		project := fmt.Sprintf("lifecycle-never-throws-%d-%d", time.Now().UnixNano(), i)

		withHealthProbe(t, fn)
		rep, err := Health(ctx, svc, project)
		if err != nil {
			t.Fatalf("phase %d: Health must not throw: %v", i, err)
		}
		if len(rep.Recommendations) != 0 {
			t.Errorf("phase %d: expected empty recommendations on error, got %v", i, rep.Recommendations)
		}
	}
}

// --- healthRecommendations (pure rules, TICKET-02) ---

// now2026 is a pinned reference time so the review rule (review_after <= now)
// is deterministic across runs.
var now2026 = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

// findRec reports whether a fired rule's recommendation is present (matched by
// a stable message fragment) and returns it when it is.
func findRec(t *testing.T, recs []HealthRecommendation, msgFragment string) *HealthRecommendation {
	t.Helper()
	for i := range recs {
		if strings.Contains(recs[i].Message, msgFragment) {
			return &recs[i]
		}
	}
	return nil
}

// TestHealthRecommendations_HighDensity: density > 0.05 fires the HIGH dedup
// rule; all other rules off.
func TestHealthRecommendations_HighDensity(t *testing.T) {
	recs := healthRecommendations(healthInput{
		Total: 10, ByType: map[string]int{"decision": 10},
		Coverage: 1, Density: 0.2,
		Age: map[string]int{"<1d": 10},
	}, now2026)
	r := findRec(t, recs, "run dedup merge")
	if r == nil {
		t.Fatalf("expected HIGH dedup recommendation for density 0.2, got %+v", recs)
	}
	if r.Severity != "HIGH" {
		t.Errorf("dedup severity = %q, want HIGH", r.Severity)
	}
	if len(recs) != 1 {
		t.Errorf("expected exactly 1 recommendation, got %d: %+v", len(recs), recs)
	}
}

// TestHealthRecommendations_EmbeddingsOff is the NO-NAG gate: zero embedding
// coverage with the embedder OFF fires no embeddings recommendation at all.
func TestHealthRecommendations_EmbeddingsOff(t *testing.T) {
	recs := healthRecommendations(healthInput{
		Total: 10, ByType: map[string]int{"decision": 10},
		Coverage: 0, EmbedderActive: false,
		Age: map[string]int{"<1d": 10},
	}, now2026)
	if r := findRec(t, recs, "re-embedding"); r != nil {
		t.Errorf("embedder off must not nag about embeddings: %+v", recs)
	}
	if len(recs) != 0 {
		t.Errorf("expected no recommendations, got %+v", recs)
	}
}

// TestHealthRecommendations_EmbeddingsOn: zero coverage with the embedder
// ACTIVE fires the MEDIUM embeddings recommendation.
func TestHealthRecommendations_EmbeddingsOn(t *testing.T) {
	recs := healthRecommendations(healthInput{
		Total: 10, ByType: map[string]int{"decision": 10},
		Coverage: 0, EmbedderActive: true,
		Age: map[string]int{"<1d": 10},
	}, now2026)
	r := findRec(t, recs, "re-embedding")
	if r == nil {
		t.Fatalf("expected MEDIUM embeddings recommendation, got %+v", recs)
	}
	if r.Severity != "MEDIUM" {
		t.Errorf("embeddings severity = %q, want MEDIUM", r.Severity)
	}
}

// TestHealthRecommendations_Stale: total > 10 with more than half the rows
// older than 90d fires the MEDIUM stale rule.
func TestHealthRecommendations_Stale(t *testing.T) {
	recs := healthRecommendations(healthInput{
		Total: 12, ByType: map[string]int{"decision": 12},
		Coverage: 1,
		Age:      map[string]int{"<1d": 5, "90-365d": 4, ">1y": 3}, // 7 of 12 > 90d
	}, now2026)
	r := findRec(t, recs, "archive stale")
	if r == nil {
		t.Fatalf("expected MEDIUM stale recommendation, got %+v", recs)
	}
	if r.Severity != "MEDIUM" {
		t.Errorf("stale severity = %q, want MEDIUM", r.Severity)
	}
	if !strings.Contains(r.Message, "7 of 12") {
		t.Errorf("stale message must name the stale fraction: %q", r.Message)
	}
}

// TestHealthRecommendations_AllOff: every rule below threshold yields an
// empty (non-nil) list.
func TestHealthRecommendations_AllOff(t *testing.T) {
	recs := healthRecommendations(healthInput{
		Total: 5, ByType: map[string]int{"decision": 5},
		Embedded: 5, Coverage: 1, Density: 0.0,
		Age:            map[string]int{"<1d": 5},
		EmbedderActive: true,                                 // even the most nag-prone rule stays silent
		Archived:       1,                                    // 20% < 30%
		ReviewAfter:    []string{"2027-01-01T00:00:00Z", ""}, // future + empty
	}, now2026)
	if recs == nil {
		t.Fatal("expected a non-nil empty slice")
	}
	if len(recs) != 0 {
		t.Errorf("expected no recommendations, got %+v", recs)
	}
}

// TestHealthRecommendations_Cleanup: archived over 30% of total fires the
// LOW cleanup rule; at or under 30% it does not.
func TestHealthRecommendations_Cleanup(t *testing.T) {
	recs := healthRecommendations(healthInput{
		Total: 10, ByType: map[string]int{"decision": 10},
		Coverage: 1,
		Age:      map[string]int{"<1d": 10},
		Archived: 4, // 40% > 30%
	}, now2026)
	r := findRec(t, recs, "purge")
	if r == nil {
		t.Fatalf("expected LOW cleanup recommendation at 40%% archived, got %+v", recs)
	}
	if r.Severity != "LOW" {
		t.Errorf("cleanup severity = %q, want LOW", r.Severity)
	}
	// Boundary: exactly 30% is not "over 30%".
	if recs2 := healthRecommendations(healthInput{
		Total: 10, ByType: map[string]int{"decision": 10}, Coverage: 1,
		Age: map[string]int{"<1d": 10}, Archived: 3,
	}, now2026); findRec(t, recs2, "purge") != nil {
		t.Errorf("exactly 30%% archived must not fire cleanup: %+v", recs2)
	}
}

// TestHealthRecommendations_ReviewDue: any observation with review_after <=
// now fires the LOW review rule; a future review_after does not.
func TestHealthRecommendations_ReviewDue(t *testing.T) {
	recs := healthRecommendations(healthInput{
		Total: 2, ByType: map[string]int{"decision": 2},
		Coverage:    1,
		Age:         map[string]int{"<1d": 2},
		ReviewAfter: []string{"2026-01-01T00:00:00Z", "2027-01-01T00:00:00Z"},
	}, now2026)
	r := findRec(t, recs, "review")
	if r == nil {
		t.Fatalf("expected LOW review recommendation for due review_after, got %+v", recs)
	}
	if r.Severity != "LOW" {
		t.Errorf("review severity = %q, want LOW", r.Severity)
	}
	if recs2 := healthRecommendations(healthInput{
		Total: 1, ByType: map[string]int{"decision": 1}, Coverage: 1,
		Age:         map[string]int{"<1d": 1},
		ReviewAfter: []string{"2027-01-01T00:00:00Z"},
	}, now2026); findRec(t, recs2, "review") != nil {
		t.Errorf("future review_after must not fire: %+v", recs2)
	}
}

// TestHealthRecommendations_StableOrdering: identical inputs fire every rule
// and produce byte-identical output on repeated calls.
func TestHealthRecommendations_StableOrdering(t *testing.T) {
	in := healthInput{
		Total: 20, ByType: map[string]int{"decision": 10, "learning": 10},
		Embedded: 1, Coverage: 0.05, Density: 0.3,
		Age:            map[string]int{"30-90d": 8, "90-365d": 9, ">1y": 3},
		EmbedderActive: true,
		Archived:       8, // 40%
		ReviewAfter:    []string{"", "2026-06-01T00:00:00Z", "2027-01-01T00:00:00Z"},
	}
	first := healthRecommendations(in, now2026)
	if len(first) != 5 {
		t.Fatalf("expected all 5 rules to fire, got %d: %+v", len(first), first)
	}
	// The fixed table order: dedup, embeddings, stale, cleanup, review.
	for i, want := range []string{"HIGH", "MEDIUM", "MEDIUM", "LOW", "LOW"} {
		if first[i].Severity != want {
			t.Errorf("rule %d severity = %q, want %q", i, first[i].Severity, want)
		}
	}
	for call := 0; call < 3; call++ {
		again := healthRecommendations(in, now2026)
		if len(again) != len(first) {
			t.Fatalf("call %d: length drift %d vs %d", call, len(again), len(first))
		}
		for i := range first {
			if again[i] != first[i] {
				t.Fatalf("call %d: rule %d drifted: %+v vs %+v", call, i, again[i], first[i])
			}
		}
	}
}

// TestLifecycle_HealthCached verifies the 24h file cache: the probe (the
// expensive computation) runs once, the second call is served from cache. The
// project name is unique so a stale cache file from a prior test run can never
// pre-seed the first call.
func TestLifecycle_HealthCached(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "test-project", 3, "auth")
	project := fmt.Sprintf("lifecycle-cached-%d", time.Now().UnixNano())

	calls := 0
	withHealthProbe(t, func() (HealthReport, error) {
		calls++
		return HealthReport{Total: 3}, nil
	})
	if _, err := Health(context.Background(), svc, project); err != nil {
		t.Fatalf("first Health: %v", err)
	}
	if _, err := Health(context.Background(), svc, project); err != nil {
		t.Fatalf("second Health: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected the 24h cache to serve the second call (calls=%d, want 1)", calls)
	}
}

// --- Dedup ---

func TestLifecycle_DedupScan(t *testing.T) {
	svc := newTestService(t, true)
	seedNearDupes(t, svc, "test-project", 3)

	clusters, density, err := DedupScan(context.Background(), svc, "test-project", true)
	if err != nil {
		t.Fatalf("DedupScan: %v", err)
	}
	if len(clusters) == 0 {
		t.Error("expected clusters")
	}
	if density <= 0 {
		t.Errorf("expected density > 0, got %v", density)
	}
	if clusters[0].Canonical == 0 {
		t.Error("expected a canonical id in the cluster")
	}
	// dry_run must not mutate: nothing is archived.
	for _, c := range clusters {
		if isArchived(t, svc, "test-project", c.Canonical) {
			t.Error("dry_run scan must not archive the canonical")
		}
	}
}

// TestLifecycle_DedupDegradesHash locks the no-embedder floor: with no embedder
// the scan clusters by content hash and is marked degraded.
func TestLifecycle_DedupDegradesHash(t *testing.T) {
	svc := newTestService(t, false)
	// Two pairs sharing one normalized_hash. Save dedups identical
	// (title+content+type) triples within 24h, so we insert rows directly via
	// SQL to guarantee 4 distinct rows with 2 pairs of matching hashes.
	ensureSession(t, svc, "test-project")
	hh, cleanup, err := svc.Open("test-project")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	db := hh.Store().DB
	now := time.Now().UTC().Format(time.RFC3339)
	hashA := "aarduphashpair0000000000000000"
	hashB := "bbdduphashpair0000000000000000"
	rows := []struct {
		id    int
		hash  string
		title string
	}{
		{1, hashA, "hash dupe A"},
		{2, hashA, "hash dupe A"},
		{3, hashB, "hash dupe B"},
		{4, hashB, "hash dupe B"},
	}
	for _, r := range rows {
		if _, err := db.Exec(`
			INSERT INTO observations (id, project, title, content, type, scope,
				normalized_hash, session_id, owner, visibility, status, created_at, updated_at)
			VALUES (?, 'test-project', ?, 'hash content', 'decision', 'project',
				?, ?, 'test-owner', 'private', 'active', ?, ?)`,
			r.id, r.title, r.hash, sessionA, now, now,
		); err != nil {
			t.Fatalf("insert row %d: %v", r.id, err)
		}
	}

	clusters, _, err := DedupScan(context.Background(), svc, "test-project", true)
	if err != nil {
		t.Fatalf("DedupScan: %v", err)
	}
	if len(clusters) == 0 {
		t.Fatal("expected hash-based clusters")
	}
	if !clusters[0].Degraded {
		t.Error("expected degraded=true with no embedder")
	}
}

func TestLifecycle_DedupMergeProvenance(t *testing.T) {
	svc := newTestService(t, true)
	keep, dup := seedNearDupePair(t, svc, "test-project")

	if err := DedupMerge(context.Background(), svc, "test-project", keep); err != nil {
		t.Fatalf("DedupMerge: %v", err)
	}
	if !isArchived(t, svc, "test-project", dup) {
		t.Error("duplicate should be soft-archived")
	}
	if isArchived(t, svc, "test-project", keep) {
		t.Error("canonical should not be archived")
	}
	if !hasConsolidatedFrom(t, svc, "test-project", keep, dup) {
		t.Error("canonical should have consolidated_from with the duplicate")
	}
}

// --- Consolidate (LLM fail-open to a deterministic provenance note) ---

func TestLifecycle_ConsolidateFailOpen(t *testing.T) {
	svc := newTestService(t, false)
	a := seedOne(t, svc, "test-project")
	b := seedOne(t, svc, "test-project")

	// No LLM attached → fail open to the deterministic provenance note.
	newID, err := Consolidate(context.Background(), svc, "test-project", []int64{a, b}, "Merged auth notes")
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if newID == 0 || newID == a || newID == b {
		t.Errorf("expected a NEW consolidated id, got %d (sources %d %d)", newID, a, b)
	}
	// Sources are soft-archived.
	if !isArchived(t, svc, "test-project", a) || !isArchived(t, svc, "test-project", b) {
		t.Error("sources should be soft-archived")
	}
	// The new observation carries consolidated_from provenance.
	if !hasConsolidatedFrom(t, svc, "test-project", newID, a) || !hasConsolidatedFrom(t, svc, "test-project", newID, b) {
		t.Error("new observation should carry consolidated_from with both source ids")
	}
}

func TestLifecycle_ConsolidateLLM(t *testing.T) {
	svc := newTestService(t, false)
	a := seedOne(t, svc, "test-project")
	b := seedOne(t, svc, "test-project")

	setAskLLM(t, &fakeLLM{fn: func(_ context.Context, _, _ string) (string, error) {
		return "Combined: we decided to use SQLite [obs:1].", nil
	}})
	newID, err := Consolidate(context.Background(), svc, "test-project", []int64{a, b}, "Merged")
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if newID == 0 {
		t.Fatal("expected a new consolidated id")
	}
	hh, cleanup, err := svc.Open("test-project")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	o, err := hh.Memory().Get(context.Background(), newID)
	if err != nil {
		t.Fatalf("get new: %v", err)
	}
	if !strings.Contains(o.Content, "Combined:") {
		t.Errorf("expected LLM prose in the consolidated content, got %q", o.Content)
	}
}

// --- Archive / restore / list / stale ---

func TestLifecycle_ArchiveRestore(t *testing.T) {
	svc := newTestService(t, false)
	id := seedOne(t, svc, "test-project")

	res, err := Archive(context.Background(), svc, "test-project", "archive", []int64{id}, "stale", 90)
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if res == nil || len(res.Affected) == 0 {
		t.Errorf("expected affected ids, got %+v", res)
	}
	if !isArchived(t, svc, "test-project", id) {
		t.Error("observation should be archived")
	}
	// archive_reason is recorded.
	hh, cleanup, oerr := svc.Open("test-project")
	if oerr != nil {
		t.Fatalf("open: %v", oerr)
	}
	defer cleanup()
	var reason string
	if err := hh.Store().DB.QueryRow(
		`SELECT COALESCE(archive_reason,'') FROM observations WHERE id = ?`, id,
	).Scan(&reason); err != nil {
		t.Fatalf("reason read: %v", err)
	}
	if reason != "stale" {
		t.Errorf("archive_reason = %q, want stale", reason)
	}

	if _, err := Archive(context.Background(), svc, "test-project", "restore", []int64{id}, "", 90); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if isArchived(t, svc, "test-project", id) {
		t.Error("observation should be restored")
	}
}

// TestLifecycle_ArchiveListEmpty locks "failed reads return an empty list": a
// project with no archived observations yields an empty list, not an error.
func TestLifecycle_ArchiveListEmpty(t *testing.T) {
	svc := newTestService(t, false)
	res, err := Archive(context.Background(), svc, "test-project", "list", nil, "", 90)
	if err != nil {
		t.Fatalf("list on empty project: %v", err)
	}
	if res == nil || res.Affected != nil {
		t.Errorf("expected a nil/empty affected list, got %+v", res)
	}
}

// TestLifecycle_ArchiveStale returns only observations older than staleDays.
func TestLifecycle_ArchiveStale(t *testing.T) {
	svc := newTestService(t, false)
	ensureSession(t, svc, "test-project")
	hh, cleanup, err := svc.Open("test-project")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Five old (created 100d ago) and five recent observations.
	oldStamp := time.Now().UTC().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	recent := time.Now().UTC().Format(time.RFC3339)
	ids := make([]int64, 0, 10)
	for i := 0; i < 5; i++ {
		id, err := hh.Memory().Save(context.Background(), memory.SaveInput{
			Title: fmt.Sprintf("old %d", i), Type: "decision",
			Content: "old content", Scope: "project", SessionID: sessionA,
		})
		if err != nil {
			t.Fatalf("save old: %v", err)
		}
		ids = append(ids, id)
		if _, err := hh.Store().DB.Exec(`UPDATE observations SET created_at = ? WHERE id = ?`, oldStamp, id); err != nil {
			t.Fatalf("backdate: %v", err)
		}
	}
	for i := 0; i < 5; i++ {
		id, err := hh.Memory().Save(context.Background(), memory.SaveInput{
			Title: fmt.Sprintf("recent %d", i), Type: "decision",
			Content: "recent content", Scope: "project", SessionID: sessionA,
		})
		if err != nil {
			t.Fatalf("save recent: %v", err)
		}
		if _, err := hh.Store().DB.Exec(`UPDATE observations SET created_at = ? WHERE id = ?`, recent, id); err != nil {
			t.Fatalf("stamp recent: %v", err)
		}
	}
	cleanup()

	res, err := Archive(context.Background(), svc, "test-project", "stale", nil, "", 90)
	if err != nil {
		t.Fatalf("stale: %v", err)
	}
	if len(res.Affected) != 5 {
		t.Fatalf("expected 5 stale observations, got %d", len(res.Affected))
	}
}

// --- Growth timeline ---

// seedGrowthRow stores one observation backdated to the given RFC3339 stamp.
// When softDelete is true the row's deleted_at is set after insert.
func seedGrowthRow(t *testing.T, svc *service.Service, project string, createdAt string, softDelete bool) int64 {
	t.Helper()
	ensureSession(t, svc, project)
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	id, err := hh.Memory().Save(context.Background(), memory.SaveInput{
		Title: "growth row", Type: "decision", Content: "growth content",
		Scope: "project", SessionID: sessionA,
	})
	if err != nil {
		t.Fatalf("growth save: %v", err)
	}
	if _, err := hh.Store().DB.Exec(`UPDATE observations SET created_at = ? WHERE id = ?`, createdAt, id); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	if softDelete {
		if _, err := hh.Store().DB.Exec(
			`UPDATE observations SET deleted_at = ? WHERE id = ?`,
			time.Now().UTC().Format(time.RFC3339), id,
		); err != nil {
			t.Fatalf("soft delete: %v", err)
		}
	}
	return id
}

func TestGrowthTimeline_Monthly(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, false)
	project := "growth-monthly"

	seedGrowthRow(t, svc, project, "2026-01-15T00:00:00Z", false) // January (a1)
	seedGrowthRow(t, svc, project, "2026-02-10T00:00:00Z", false) // February (a2)
	seedGrowthRow(t, svc, project, "2026-03-05T00:00:00Z", false) // March (a3)
	seedGrowthRow(t, svc, project, "2026-03-20T00:00:00Z", true)  // March, soft-deleted (excluded)

	out, err := GrowthTimeline(ctx, svc, project, "month")
	if err != nil {
		t.Fatalf("GrowthTimeline: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("expected 3 periods, got %d: %+v", len(out), out)
	}
	// January: a1 added, a1 total.
	if out[0].Added != 1 || out[0].Total != 1 {
		t.Errorf("January = %+v, want {Added:1 Total:1}", out[0])
	}
	// February: a2 added, a1+a2 total (cumulative).
	if out[1].Added != 1 || out[1].Total != 2 {
		t.Errorf("February = %+v, want {Added:1 Total:2}", out[1])
	}
	// March: a3 added (soft-deleted row excluded), a1+a2+a3 total.
	if out[2].Added != 1 || out[2].Total != 3 {
		t.Errorf("March = %+v, want {Added:1 Total:3}", out[2])
	}

	// Empty store: an empty (non-nil) slice, no error.
	emptySvc := newTestService(t, false)
	empty, err := GrowthTimeline(ctx, emptySvc, "growth-empty", "month")
	if err != nil {
		t.Fatalf("GrowthTimeline empty: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Errorf("expected non-nil empty slice on empty store, got %#v", empty)
	}
}

// --- Audit log ---

func TestLifecycle_AuditRow(t *testing.T) {
	svc := newTestService(t, false)
	id := seedOne(t, svc, "test-project")

	if _, err := Archive(context.Background(), svc, "test-project", "archive", []int64{id}, "stale", 90); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if !hasAuditRow(t, svc, "test-project", "archive", "completed", id) {
		t.Error("expected a completed lifecycle_log row for the archive")
	}
}
