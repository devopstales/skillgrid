package layer

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// The LLM distillation pass is cached by the L0 source content-hash. Re-running
// the pass on an unchanged source is a cache hit (no re-distillation, no second
// LLM call); a changed source misses and re-distills. This mirrors the
// process.content-hash label cache: the result is a pure function of the source
// content (modulo the LLM), keyed by that content's hash.

// loadLLMCache returns the cached L2 scenario + L3 persona delta for the given
// source content-hash. ok is false on a miss (no cache row) so the caller runs
// (and re-caches) the LLM pass.
func loadLLMCache(ctx context.Context, svc *memory.Service, sourceHash string) (scenario, delta string, ok bool) {
	row := svc.DB().QueryRowContext(ctx, `
		SELECT scenario, delta FROM distill_llm_cache
		WHERE project = ? AND source_hash = ?`, svc.ProjectID(), sourceHash)
	var s, d sql.NullString
	if err := row.Scan(&s, &d); err != nil {
		return "", "", false
	}
	return s.String, d.String, true
}

// saveLLMCache caches the L2 scenario + L3 persona delta for the source
// content-hash. Idempotent: a re-cache on the same hash overwrites in place.
func saveLLMCache(ctx context.Context, svc *memory.Service, sourceHash, scenario, delta string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = svc.DB().ExecContext(ctx, `
		INSERT INTO distill_llm_cache (project, source_hash, scenario, delta, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(project, source_hash) DO UPDATE SET
			scenario = excluded.scenario,
			delta = excluded.delta`,
		svc.ProjectID(), sourceHash, scenario, delta, now)
}

var _ = fmt.Sprintf
