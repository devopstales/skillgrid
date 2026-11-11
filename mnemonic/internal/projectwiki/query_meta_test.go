package projectwiki

import (
	"context"
	"testing"
)

// W003 (QA 2026-11-10): the engine side of spec 005 §6 — a thin result
// carries the mem_lookup fallback hint, and a lagging index degrades
// explicitly instead of silently serving stale rows.

func TestAssessQueryThinOnEmptyIndex(t *testing.T) {
	ctx := context.Background()
	st := openProjectWikiStore(t)

	thin, fallback, degraded, err := AssessQuery(ctx, st, t.TempDir(), "proj-a", 0)
	if err != nil {
		t.Fatalf("assess: %v", err)
	}
	if !thin {
		t.Fatal("0 results on an empty index must be thin (blueprint T1/T4)")
	}
	if fallback != "mem_lookup" {
		t.Fatalf("fallback = %q, want mem_lookup", fallback)
	}
	if degraded != "" {
		t.Fatalf("degraded = %q, want empty for an empty vault", degraded)
	}
}

func TestAssessQueryNotThinWithEnoughHits(t *testing.T) {
	ctx := context.Background()
	st := openProjectWikiStore(t)

	thin, fallback, _, err := AssessQuery(ctx, st, t.TempDir(), "proj-a", ThinThreshold)
	if err != nil {
		t.Fatalf("assess: %v", err)
	}
	if thin || fallback != "" {
		t.Fatalf("thin=%v fallback=%q, want false/empty at threshold", thin, fallback)
	}
}

func TestAssessQueryDegradedWhenIndexLagsVault(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2.")
	st := openProjectWikiStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}

	_, _, degraded, err := AssessQuery(ctx, st, vault, "proj-a", 1)
	if err != nil {
		t.Fatalf("assess fresh: %v", err)
	}
	if degraded != "" {
		t.Fatalf("degraded = %q, want empty right after reindex", degraded)
	}

	// Mutate the file without re-indexing: the index now lags the vault.
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2 with rotation.")
	_, _, degraded, err = AssessQuery(ctx, st, vault, "proj-a", 1)
	if err != nil {
		t.Fatalf("assess stale: %v", err)
	}
	if degraded != "stale_index" {
		t.Fatalf("degraded = %q, want stale_index", degraded)
	}
}
