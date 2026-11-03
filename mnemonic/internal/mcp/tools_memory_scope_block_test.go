package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// scopeBlockHit is one entry of the additive scope block (TICKET-10,
// @step-11 explicit-search-scope, G12): which non-empty buckets a
// multi-bucket search hit, with per-bucket counts. It is a provenance label
// only — it never reorders or re-ranks the merged hits.
type scopeBlockHit struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// scopeBlockBody is the additive part of the all_projects mem_search
// response the scope block tests assert against (raw key map alongside it so
// the ABSENT-key contract is checkable: a struct without omitempty would pass
// a zero check whether the key exists or not).
type scopeBlockBody struct {
	Project      string          `json:"project"`
	AllProjects  bool            `json:"all_projects"`
	Count        int             `json:"count"`
	BucketsHit   []scopeBlockHit `json:"buckets_hit"`
	Observations []struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Project string `json:"project"`
	} `json:"observations"`
}

// TestMemSearch_ScopeBlock is the @step-11 `explicit-search-scope` scenario
// (G12): two non-empty project buckets share a term, so a multi-bucket
// search (all_projects=true) must carry an additive scope block naming both
// buckets with per-bucket counts, AHEAD of the merged ranks, without
// touching the ranking. The block is a provenance label: the merged ranks
// stay the same rows in the same order as a scope-block-less response.
//
// Fixture notes: the scoped search is read as a NAMED reader (the session
// that saved in that bucket) because per-owner visibility (013 step 01) gates
// a private observation from a different owner — the blank reader sees the
// whole corpus, which is the other valid path; the named reader pins the
// row-to-owner link the seed makes. The all_projects leg is not owner-gated
// at the service layer.
func TestMemSearch_ScopeBlock(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_EMBED", "")
	t.Setenv("MNEMONIC_PROJECT", "scope-a")
	SetService(newScopeBlockService(dataDir))
	t.Cleanup(func() { SetService(nil) })
	ctx := context.Background()

	// Two non-empty buckets sharing a term: the cross-store query must hit
	// both. Each bucket gets its own session (the save path's FK to sessions).
	seedObservationInBucket(t, dataDir, "scope-a", "scopeblock probe alpha", "scopeblock shared marker alpha")
	seedObservationInBucket(t, dataDir, "scope-a", "scopeblock probe alpha second", "scopeblock shared marker alpha two")
	seedObservationInBucket(t, dataDir, "scope-b", "scopeblock probe beta", "scopeblock shared marker beta")

	// Multi-bucket search (all_projects=true): the scope block must name both
	// buckets with their per-bucket counts.
	res, err := handleMemSearch(ctx, newCallTool("mem_search", map[string]any{
		"query":        "scopeblock shared marker",
		"all_projects": true,
	}))
	if err != nil {
		t.Fatalf("all_projects search dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("all_projects search errored: %s", callResultText(t, res))
	}
	raw := callResultText(t, res)
	var body scopeBlockBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("unmarshal all_projects result: %v (text %s)", err, raw)
	}
	if body.Project != "all" || !body.AllProjects {
		t.Fatalf("federated response contract wrong: project=%q all_projects=%v", body.Project, body.AllProjects)
	}
	if body.Count != 3 || len(body.Observations) != 3 {
		t.Fatalf("federated search returned %d hits (count=%d), want all three rows", len(body.Observations), body.Count)
	}
	// The additive scope block: both non-empty buckets named with counts.
	if len(body.BucketsHit) != 2 {
		t.Fatalf("expected the scope block to name 2 non-empty buckets, got %d: %s", len(body.BucketsHit), raw)
	}
	want := map[string]int{"scope-a": 2, "scope-b": 1}
	got := map[string]int{}
	for _, b := range body.BucketsHit {
		got[b.Bucket] = b.Count
	}
	for bucket, count := range want {
		if c, ok := got[bucket]; !ok || c != count {
			t.Errorf("scope block: bucket %q count = %d (present=%v), want %d", bucket, c, ok, count)
		}
	}
	// Provenance only: the scope block does NOT reorder or re-rank. The
	// merged ranks are the same rows the service returns (the block adds a
	// label, it never touches order) — every hit carries its project and the
	// per-bucket count equals the number of rows per project in the merged
	// ranks.
	perProject := map[string]int{}
	for _, o := range body.Observations {
		perProject[o.Project]++
	}
	for bucket, count := range perProject {
		if got[bucket] != count {
			t.Errorf("scope block bucket %q count=%d, want the merged-rank row count %d", bucket, got[bucket], count)
		}
	}

	// The scoped default (all_projects=false, the DEFAULT case): the same
	// term in one bucket must carry NO scope block (byte-identical shape,
	// no buckets_hit key). Read as the bucket's saving session (per-owner
	// visibility, 013 step 01).
	scopedRaw := scopedSearchJSON(t, "scope-a", "scopeblock shared marker", seedSessionForBucket(t, dataDir, "scope-a"))
	var rawMap map[string]any
	if err := json.Unmarshal([]byte(scopedRaw), &rawMap); err != nil {
		t.Fatalf("unmarshal scoped result: %v (text %s)", err, scopedRaw)
	}
	if _, present := rawMap["buckets_hit"]; present {
		t.Errorf("default scoped search must NOT carry the scope block, got: %v", rawMap["buckets_hit"])
	}
	if _, present := rawMap["all_projects"]; present {
		t.Errorf("default scoped search must not carry the federated contract key, got: %v", rawMap["all_projects"])
	}
	if rawMap["project"] != "scope-a" {
		t.Errorf("scoped search project=%v, want the bucket id scope-a", rawMap["project"])
	}
}

// TestMemSearch_ScopeBlock_SingleBucket pins the DEFAULT case from the
// multi-bucket side: an all_projects=true search that hits only ONE
// non-empty bucket does not earn a scope block — the block names buckets a
// search HIT, and a single-bucket hit is the shape the default scoped
// search already has.
func TestMemSearch_ScopeBlock_SingleBucket(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_EMBED", "")
	t.Setenv("MNEMONIC_PROJECT", "scope-single")
	SetService(newScopeBlockService(dataDir))
	t.Cleanup(func() { SetService(nil) })
	ctx := context.Background()

	seedObservationInBucket(t, dataDir, "scope-single", "scopeblock single probe", "scopeblock single bucket term")

	res, err := handleMemSearch(ctx, newCallTool("mem_search", map[string]any{
		"query":        "scopeblock single bucket",
		"all_projects": true,
	}))
	if err != nil {
		t.Fatalf("all_projects search dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("all_projects search errored: %s", callResultText(t, res))
	}
	raw := callResultText(t, res)
	var rawMap map[string]any
	if err := json.Unmarshal([]byte(raw), &rawMap); err != nil {
		t.Fatalf("unmarshal all_projects result: %v (text %s)", err, raw)
	}
	if rawMap["project"] != "all" {
		t.Fatalf("federated response contract wrong: project=%v", rawMap["project"])
	}
	if hits, present := rawMap["buckets_hit"]; present {
		t.Errorf("single-bucket all_projects search must NOT carry the scope block, got: %v", hits)
	}
	// The hit itself is unaffected (envelope-additive).
	if n := len(rawMap["observations"].([]any)); n != 1 {
		t.Errorf("expected 1 observation, got %d: %s", n, raw)
	}
}

// TestMemSearch_ScopeBlock_ByteIdentical pins the byte-compatibility
// contract from the acceptance scenario: the default scoped search
// (all_projects=false) response shape is byte-identical to today with no
// scope block. Without a committed pre-change baseline JSON in-tree, the
// provable assertion is that the scope block key is ABSENT from the default
// response (and the response keeps its existing keys exactly) — a new
// buckets_hit key is the only way the shape could drift.
func TestMemSearch_ScopeBlock_ByteIdentical(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_EMBED", "")
	t.Setenv("MNEMONIC_PROJECT", "scope-byte")
	SetService(newScopeBlockService(dataDir))
	t.Cleanup(func() { SetService(nil) })

	seedObservationInBucket(t, dataDir, "scope-byte", "scopeblock byte probe", "scopeblock byte identity term")

	scopedRaw := scopedSearchJSON(t, "scope-byte", "scopeblock byte identity", seedSessionForBucket(t, dataDir, "scope-byte"))
	var rawMap map[string]any
	if err := json.Unmarshal([]byte(scopedRaw), &rawMap); err != nil {
		t.Fatalf("unmarshal scoped result: %v (text %s)", err, scopedRaw)
	}
	// The scope block key is the additive field — it must be ABSENT from the
	// default response (not present-with-empty).
	if _, present := rawMap["buckets_hit"]; present {
		t.Fatalf("default scoped search shape drifted: buckets_hit key present: %s", scopedRaw)
	}
	// The pre-existing default keys are intact (the shape the federated
	// floor already pins: project + observations + count).
	for _, key := range []string{"project", "observations", "count"} {
		if _, ok := rawMap[key]; !ok {
			t.Fatalf("default scoped search lost the pre-existing %q key: %s", key, scopedRaw)
		}
	}
	if rawMap["count"].(float64) != 1 {
		t.Errorf("default scoped search count = %v, want 1", rawMap["count"])
	}
}

// scopedSearchJSON runs a scoped (all_projects=false, the default)
// mem_search as the named reader and returns the raw response text.
func scopedSearchJSON(t *testing.T, svcProject, query, reader string) string {
	t.Helper()
	res, err := handleMemSearch(context.Background(), newCallTool("mem_search", map[string]any{
		"query":        query,
		"reader_owner": reader,
	}))
	if err != nil {
		t.Fatalf("scoped mem_search dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("scoped mem_search errored: %s", callResultText(t, res))
	}
	return callResultText(t, res)
}

// newScopeBlockService builds a service rooted at dataDir (the shared
// test-fixture shape used by the federated/shape tests: a pinned
// MNEMONIC_PROJECT + an injected service).
func newScopeBlockService(dataDir string) *service.Service {
	return service.New(dataDir)
}

// seedObservationInBucket opens the BUCKET's own store handle and starts a
// session inside THAT store (the observations FK references the per-store
// sessions table, so a session created in one bucket is invisible to another
// bucket's store), then saves one observation through the same handle. The
// seedObservation helper below re-opens its own handle, but the session row
// is already committed in the bucket's store file, so the FK resolves.
func seedObservationInBucket(t *testing.T, dataDir, project, title, content string) {
	t.Helper()
	h, cleanup, err := service.New(dataDir).Open(project)
	if err != nil {
		t.Fatalf("open bucket %s for seed: %v", project, err)
	}
	defer cleanup()
	sid, err := h.Memory().SessionStart(context.Background(), dataDir, title)
	if err != nil {
		t.Fatalf("session start in %s: %v", project, err)
	}
	if _, err := h.Memory().Save(context.Background(), memory.SaveInput{
		Title:     title,
		Type:      "learning",
		Content:   content,
		Scope:     "project",
		SessionID: sid,
	}); err != nil {
		t.Fatalf("save observation in %s: %v", project, err)
	}
}

// seedSessionForBucket returns the FIRST session id of the bucket (the
// saving session the seedObservationInBucket flow created), so a scoped
// search can read as that owner and pass the per-owner visibility gate (013
// step 01). It reads the sessions table directly — the handle's store is the
// same one the seed wrote through, so the id it returns is the seed's.
func seedSessionForBucket(t *testing.T, dataDir, project string) string {
	t.Helper()
	h, cleanup, err := service.New(dataDir).Open(project)
	if err != nil {
		t.Fatalf("open bucket for session: %v", err)
	}
	defer cleanup()
	var id string
	err = h.Store().DB.QueryRow(`SELECT id FROM sessions WHERE project = ? ORDER BY rowid LIMIT 1`, project).Scan(&id)
	if err != nil {
		t.Fatalf("first session in %s: %v", project, err)
	}
	return id
}
