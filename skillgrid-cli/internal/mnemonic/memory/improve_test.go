package memory

import (
	"context"

	"slices"
	"testing"
	"time"
)

// newImproveFixture opens a store and service, enabling the opt-in improve()
// feedback loop (014 step 08) with a known config, and a session for saves.
func newImproveFixture(t *testing.T, project string) *ownerFixture {
	t.Helper()
	fx := newOwnerFixture(t, project)
	fx.svc.SetImprove(ImproveConfig{
		Enabled:     true,
		Threshold:   5,
		MaxUsage:    50,
		BoostRate:   0.1,
		DecayRate:   0.25,
		Cooldown:    0,
		Now:         func() time.Time { return time.Now().UTC() },
	})
	return fx
}

// seedImproveObs inserts an observation directly with a controlled
// retrieval_usage and created_at (the FTS trigger keeps search in sync).
// createdOffset is how far in the past created_at sits relative to now.
func seedImproveObs(t *testing.T, fx *ownerFixture, title, body, owner string, usage int, createdOffset time.Duration) int64 {
	t.Helper()
	now := time.Now().UTC()
	fmtS := func(d time.Time) string { return d.UTC().Format(time.RFC3339) }
	created := fmtS(now.Add(-createdOffset))
	updated := fmtS(now)
	expires := fmtS(now.Add(7 * 24 * time.Hour))
	res, err := fx.st.DB.Exec(`
		INSERT INTO observations (
			session_id, type, title, content, project, scope,
			normalized_hash, created_at, updated_at, source,
			owner, visibility, status, retrieval_usage, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fx.sessionID, "decision", title, body, fx.svc.ProjectID(), "",
		"", created, updated, "manual", owner, "private", "active", usage, expires,
	)
	if err != nil {
		t.Fatalf("seed observation %q: %v", title, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seed last insert id %q: %v", title, err)
	}
	return id
}

// TestImproveBoostsHighUsageObservations covers @step-08 (Scenario:
// high-usage-observations-are-boosted): with improve() enabled, an
// observation with retrieval_usage=100 outranks one with retrieval_usage=10,
// which outranks one with retrieval_usage=0 (below threshold).
func TestImproveBoostsHighUsageObservations(t *testing.T) {
	fx := newImproveFixture(t, "improve-boost")
	ctx := context.Background()

	// The 0-usage observation matches the most query tokens (title token +
	// its distinctive "shared" body token), so the raw SQL bm25 order is
	// [0-usage, 100-usage, 10-usage] — the opposite of the boosted order.
	seedImproveObs(t, fx, "improvealpha highusage asset", "improvealpha filler body one", fx.ownerA, 100, 1*time.Hour)
	seedImproveObs(t, fx, "improvebeta midusage asset", "improvebeta filler body two", fx.ownerA, 10, 1*time.Hour)
	seedImproveObs(t, fx, "improveceta lowusage asset", "improveceta shared filler body three", fx.ownerA, 0, 1*time.Hour)

	// Capture the raw SQL ranking first (improve off): this is the
	// pre-improve order the boost must correct.
	rawCfg := fx.svc.ImproveConfig()
	fx.svc.SetImprove(ImproveConfig{Enabled: false})
	raw, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent",
		"improvealpha improvebeta improveceta shared", "any", "", 10)
	if err != nil {
		t.Fatalf("search (raw): %v", err)
	}
	fx.svc.SetImprove(rawCfg)
	if len(raw) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(raw))
	}
	rawUsage := make([]int, len(raw))
	for i, h := range raw {
		rawUsage[i] = h.RetrievalUsage
	}
	if slices.Equal(rawUsage, []int{100, 10, 0}) {
		t.Fatalf("test setup broken: raw SQL order is already usage-desc (%v); "+
			"pick tokens/usage so the pre-improve order differs from the boosted order", rawUsage)
	}

	// Now with improve() enabled the same search must be re-ranked by
	// retrieval usage: 100 > 10 > 0.
	hits, err := fx.svc.SearchOwnerScoped(ctx, fx.ownerA, "agent",
		"improvealpha improvebeta improveceta shared", "any", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}
	gotOrder := make([]int, len(hits))
	for i, h := range hits {
		gotOrder[i] = h.RetrievalUsage
	}
	// The raw-capture search above bumped each hit by one (governance
	// retrieval-usage), so the boosted values are +1: 101 > 11 > 1.
	want := []int{101, 11, 1}
	if !slices.Equal(gotOrder, want) {
		t.Fatalf("improve() must re-rank by retrieval usage desc: raw SQL order %v, got %v, want %v",
			rawUsage, gotOrder, want)
	}
}
