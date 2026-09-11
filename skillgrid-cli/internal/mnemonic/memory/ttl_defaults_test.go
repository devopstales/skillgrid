package memory

import (
	"context"
	"testing"
	"time"
)

// TestSaveAutoSetsExpiresAt verifies that Save() stamps an observation with a
// default expiry (~7 days out) when the caller does not supply one, and that an
// explicitly provided expires_at is preserved unchanged (014 step 04, task 04.1).
func TestSaveAutoSetsExpiresAt(t *testing.T) {
	_, svc := newTestStore(t, "ttlautosave")
	sid := newSession(t, svc)
	ctx := context.Background()

	// No explicit expires_at: Save() must auto-stamp ~now + 7d.
	id, err := svc.Save(ctx, SaveInput{
		Title: "auto ttl", Type: "decision", Content: "no explicit expiry",
		SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatalf("save (no expires): %v", err)
	}
	obs, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if obs.ExpiresAt == "" {
		t.Fatal("auto expires_at was not set; expected ~now+7d")
	}
	got, err := time.Parse(time.RFC3339, obs.ExpiresAt)
	if err != nil {
		t.Fatalf("expires_at %q not RFC3339: %v", obs.ExpiresAt, err)
	}
	want := time.Now().UTC().Add(7 * 24 * time.Hour)
	if d := got.Sub(want); d > time.Hour || d < -time.Hour {
		t.Fatalf("auto expires_at %s is not ~7d out (want ~%s, off by %v)", got, want, d)
	}

	// Explicit expires_at must be preserved unchanged.
	explicit := "2030-01-02T03:04:05Z"
	id2, err := svc.Save(ctx, SaveInput{
		Title: "explicit ttl", Type: "decision", Content: "explicit expiry",
		SessionID: sid, Scope: "project", ExpiresAt: explicit,
	})
	if err != nil {
		t.Fatalf("save (explicit expires): %v", err)
	}
	obs2, err := svc.Get(ctx, id2)
	if err != nil {
		t.Fatalf("get explicit: %v", err)
	}
	if obs2.ExpiresAt != explicit {
		t.Fatalf("explicit expires_at not preserved: got %q want %q", obs2.ExpiresAt, explicit)
	}
}

// TestTTLRetireOnlyExpired verifies that TTLRetire soft-deletes ONLY
// observations past their expires_at: an expired row is retired, a row
// expiring in the future is untouched, and a row with no expires_at is
// untouched. It also verifies that search excludes the retired row (014 step
// 04, task 04.2).
func TestTTLRetireOnlyExpired(t *testing.T) {
	_, svc := newTestStore(t, "ttlretire")
	sid := newSession(t, svc)
	ctx := context.Background()

	expiredID, err := svc.Save(ctx, SaveInput{
		Title: "ttl expired", Type: "decision", Content: "already expired",
		SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatalf("save expired: %v", err)
	}
	if err := svc.SetExpiresAt(ctx, expiredID, time.Now().Add(-1*time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("set expired expiry: %v", err)
	}
	futureID, err := svc.Save(ctx, SaveInput{
		Title: "ttl future", Type: "decision", Content: "expires in an hour",
		SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatalf("save future: %v", err)
	}
	if err := svc.SetExpiresAt(ctx, futureID, time.Now().Add(1*time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("set future expiry: %v", err)
	}
	noExpID, err := svc.Save(ctx, SaveInput{
		Title: "ttl noexpiry", Type: "decision", Content: "no expiry set",
		SessionID: sid, Scope: "project",
	})
	if err != nil {
		t.Fatalf("save noexpiry: %v", err)
	}
	// Clear the auto-stamped default so this row genuinely has no expires_at.
	if err := svc.SetExpiresAt(ctx, noExpID, ""); err != nil {
		t.Fatalf("clear noexpiry expiry: %v", err)
	}

	retired, err := svc.TTLRetire(ctx)
	if err != nil {
		t.Fatalf("ttl retire: %v", err)
	}
	if retired != 1 {
		t.Fatalf("TTLRetire retired %d, want exactly 1", retired)
	}

	// Search must exclude the retired row but still return the live ones.
	hits, err := svc.Search(ctx, "ttl", "any", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	seen := map[int64]bool{}
	for _, h := range hits {
		seen[h.ID] = true
	}
	if seen[expiredID] {
		t.Fatal("search returned a retired observation")
	}
	if !seen[futureID] || !seen[noExpID] {
		t.Fatalf("search did not return live observations: future=%v noexpiry=%v", seen[futureID], seen[noExpID])
	}

	// A soft-deleted row is invisible to the read path (Get filters
	// deleted_at IS NULL), confirming the expiry actually soft-deleted it.
	if _, err := svc.Get(ctx, expiredID); err == nil {
		t.Fatal("retired observation is still readable via Get; expected not-found")
	}
	if _, err := svc.Get(ctx, futureID); err != nil {
		t.Fatalf("future observation unreadable after retire: %v", err)
	}
	if _, err := svc.Get(ctx, noExpID); err != nil {
		t.Fatalf("no-expiry observation unreadable after retire: %v", err)
	}
}
