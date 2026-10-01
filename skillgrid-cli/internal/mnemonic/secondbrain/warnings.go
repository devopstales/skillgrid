package secondbrain

import (
	"context"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// warnings.go — _health_warnings (2026-09-30-mnemonic-second-brain Task 7):
// the inline, non-breaking lifecycle-warning surface for mem_search / mem_ask.
// ForProject derives HIGH/CRIT-only warnings from the 24h-cached Health report
// (Task 6). It is best-effort and must never break the caller's response: a
// panic or error yields an empty slice (recover), never a thrown panic or a
// returned error, so result ordering and the search contract are untouched.

// HealthWarning is one inline lifecycle warning surfaced in the response's
// _health_warnings array. Only HIGH/CRIT severities are ever emitted (MEDIUM /
// LOW stay in the full mem_lifecycle health report).
type HealthWarning struct {
	Severity string `json:"severity"` // HIGH | CRIT
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
}

// warningCacheTTL bounds how long a computed warnings list is reused. It
// mirrors the 24h health file cache (Health never recomputes within the
// window), so the inline warnings stay in sync with the cached report for
// free — the computation is served from the cache, not a fresh pass.
const warningCacheTTL = 24 * time.Hour

// ForProject returns the HIGH/CRIT lifecycle warnings for a single project,
// derived from the 24h-cached Health report. Any error or panic (including a
// panic inside Health) yields an empty slice — this function must never
// surface a failure to the caller, because its output is an additive meta
// field on a search response. A nil svc or empty projectID is guarded the same
// way Health guards it (returns the same empty report, not an error).
//
// It consumes the cached report (a fresh computation only when the cache is
// cold) and does not otherwise touch the store, so the all-projects path can
// call it cheaply and the unit tests can drive it over a seeded temp store.
func ForProject(svc *service.Service, projectID string) []HealthWarning {
	var out []HealthWarning
	defer func() {
		// Recover any panic (e.g. inside a malformed report) so the caller
		// always gets a usable slice. A recovered panic leaves `out` empty.
		_ = recover()
	}()
	if svc == nil || projectID == "" {
		return []HealthWarning{}
	}
	rep, err := Health(context.Background(), svc, projectID)
	if err != nil || rep.Recommendations == nil {
		return []HealthWarning{}
	}
	for _, r := range rep.Recommendations {
		if r.Severity == "HIGH" || r.Severity == "CRIT" {
			out = append(out, HealthWarning{Severity: r.Severity, Message: r.Message})
		}
	}
	if out == nil {
		out = []HealthWarning{}
	}
	return out
}

// warningCompute is the injectable computation ForProjectOn runs instead of
// the real handle pass (healthComputeReal). The test package's withHealthProbe
// installs a probe here to force the "computation errors / panics" path; a
// nil value is the real path. The 24h file cache is consulted first, so the
// probe only runs on a cache miss.
var warningCompute func(ctx context.Context, mem *memory.Service, projectID string) (HealthReport, error)

// ForProjectOn is the zero-store-open form of ForProject: it derives the
// HIGH/CRIT warnings using the store the caller has ALREADY opened (mem, from
// a ProjectHandle) and never calls svc.Open. That is what keeps the scoped
// mem_search handler inside its single-open contract — ForProject (the svc
// form) opens a store, which would be the second open on the scoped read path.
// The 24h file cache is honored first; a cache miss runs the computation over
// mem and caches the result, so repeated calls (and the mem_lifecycle health
// report) share one computation. Never throws: any error or panic yields [].
func ForProjectOn(ctx context.Context, svc *service.Service, mem *memory.Service, projectID string) []HealthWarning {
	// A recovered panic falls through to the natural []HealthWarning{} return
	// of the guards / early returns — the named result would just be shadowed.
	defer func() {
		_ = recover()
	}()
	if mem == nil || projectID == "" {
		return []HealthWarning{}
	}
	if rep, ok := healthCacheRead(projectID); ok {
		return warningsFromReport(rep)
	}
	var rep HealthReport
	var err error
	if warningCompute != nil {
		rep, err = warningCompute(ctx, mem, projectID)
	} else {
		rep, err = healthComputeReal(ctx, svc, projectID)
	}
	if err != nil {
		return []HealthWarning{}
	}
	if rep.GeneratedAt == "" {
		rep.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	healthCacheWrite(projectID, rep)
	return warningsFromReport(rep)
}

// warningsFromReport filters a health report's recommendations down to the
// HIGH/CRIT severities that are safe to surface inline.
func warningsFromReport(rep HealthReport) []HealthWarning {
	out := []HealthWarning{}
	for _, r := range rep.Recommendations {
		if r.Severity == "HIGH" || r.Severity == "CRIT" {
			out = append(out, HealthWarning{Severity: r.Severity, Message: r.Message})
		}
	}
	return out
}
