package kb

// meta.go — kb_rebuild_meta (008 §3): the derived metadata that is NOT the
// FTS index — the backlink in-degree cache + the kb_run_state checkpoint.

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// RebuildMetaResult is the kb_rebuild_meta outcome.
type RebuildMetaResult struct {
	Project  string `json:"project"`
	Pages    int    `json:"pages"`
	RunPhase string `json:"run_phase"`
	RunID    string `json:"run_id,omitempty"`
}

// RebuildMeta rebuilds the derived non-FTS metadata (008 §3 kb_rebuild_meta):
// the backlink in-degree cache (recomputed from the vault's wikilinks) and
// the kb_run_state checkpoint (the row is ensured — an idle/absent
// checkpoint is persisted with the current source fingerprint so the next
// run begins from a known state; an active run is left intact, its
// resumption is Begin's job).
func RebuildMeta(ctx context.Context, st store.Store, vaultRoot, repoRoot, project string) (*RebuildMetaResult, error) {
	res := &RebuildMetaResult{Project: project}
	pages, err := collectPages(vaultRoot)
	if err != nil {
		return nil, err
	}
	res.Pages = len(pages)
	if err := recomputeInDegree(ctx, st, vaultRoot, project); err != nil {
		return nil, err
	}
	fp, err := SourceFingerprint(ctx, st, project, repoRoot)
	if err != nil {
		return nil, err
	}
	rs, err := LoadRun(ctx, st, project)
	if err != nil {
		return nil, err
	}
	if !rs.IsActive() {
		rs = &RunState{
			Project: project, RunID: newRunID(), Mode: "rebuild_meta",
			Phase: PhaseIdle, SourceFingerprint: fp, Processed: map[string]string{},
		}
		if err := saveRun(ctx, st, rs); err != nil {
			return nil, err
		}
	}
	res.RunPhase = rs.Phase
	res.RunID = rs.RunID
	return res, nil
}
