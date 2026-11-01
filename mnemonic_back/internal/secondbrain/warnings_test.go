package secondbrain

import (
	"fmt"
	"testing"
	"time"
)

// warnings_test.go — TICKET-06 (2026-09-30-mnemonic-second-brain Task 7):
// _health_warnings — inline, non-breaking. ForProject derives HIGH/CRIT-only
// warnings from the 24h-cached Health report and returns [] on any
// error/panic (recover), so a broken health never breaks mem_search/mem_ask.
// RED-first: these tests target ForProject/HealthWarning, which do not exist
// yet (warnings.go is written in the GREEN step).

// cacheProbeProject returns a unique project id for the 24h health file cache
// (shared across the process's tests) so a stale cache from a prior test run
// can never pre-seed the first ForProject call.
func cacheProbeProject(tag string) string {
	return fmt.Sprintf("warn-%s-%d", tag, time.Now().UnixNano())
}

// TestWarnings_HighOnly locks the filter: a store with high duplicate density
// yields at least one warning, and every warning is HIGH or CRIT. The embedder
// is enabled so healthComputeReal runs the cosine dedup path (density > 0.05
// → the HIGH recommendation), and near-identical seeded content → ~1.0 cosine.
func TestWarnings_HighOnly(t *testing.T) {
	svc := newTestService(t, true)
	project := cacheProbeProject("highonly")
	seedNearDupes(t, svc, project, 6)

	ws := ForProject(svc, project)
	if len(ws) == 0 {
		t.Fatal("expected at least one HIGH/CRIT warning from high duplicate density")
	}
	for _, w := range ws {
		if w.Severity != "HIGH" && w.Severity != "CRIT" {
			t.Errorf("unexpected severity %q (only HIGH/CRIT inline)", w.Severity)
		}
		if w.Message == "" {
			t.Errorf("warning missing message: %+v", w)
		}
	}
}

// TestWarnings_EmptyWhenHealthy locks the happy floor: a store with no high
// duplicate density (no near-dupe embeddings → density 0) yields an empty
// warning slice — a healthy store adds no inline noise.
func TestWarnings_EmptyWhenHealthy(t *testing.T) {
	svc := newTestService(t, false)
	project := cacheProbeProject("healthy")
	seedObservations(t, svc, project, 5, "auth")

	ws := ForProject(svc, project)
	if len(ws) != 0 {
		t.Errorf("expected empty warnings on a healthy store, got %v", ws)
	}
}

// TestWarnings_EmptyOnError locks the never-throws contract: a health
// computation that errors or panics yields [] (recovered), never a thrown
// panic or a nil-that-becomes-an-error.
func TestWarnings_EmptyOnError(t *testing.T) {
	for i, fn := range []func() (HealthReport, error){
		func() (HealthReport, error) { return HealthReport{}, fmt.Errorf("db down") },
		func() (HealthReport, error) { panic("boom") },
	} {
		t.Run(fmt.Sprintf("phase%d", i), func(t *testing.T) {
			svc := newTestService(t, false)
			project := cacheProbeProject(fmt.Sprintf("error-%d", i))
			withHealthProbe(t, fn)
			ws := ForProject(svc, project)
			if ws == nil {
				t.Fatalf("expected a non-nil empty slice on health error, got nil")
			}
			if len(ws) != 0 {
				t.Errorf("expected empty warnings on health error, got %v", ws)
			}
		})
	}
}

// TestWarnings_NilService locks the guard: a nil service or empty project id
// yields [] (matching Health's guard) without touching the store.
func TestWarnings_NilService(t *testing.T) {
	if ws := ForProject(nil, "anything"); len(ws) != 0 {
		t.Errorf("expected empty warnings for nil service, got %v", ws)
	}
	if ws := ForProject(nil, ""); ws == nil {
		t.Error("expected a non-nil empty slice for empty project, got nil")
	}
}
