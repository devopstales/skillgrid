package kb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// manifest.go — the per-page baseline manifest (003 §5.2, 011 §3):
// kb_page_baselines (DB, authoritative) + index/.page-manifest.json (file
// projection, 003 §5.2 shape: rel_path → {sourceFingerprint, pageVersion,
// completedBy, completedRunId}).
//
// A page earns a baseline ONLY when its markdown and its pin-check are
// mutually consistent (the caller passes the hash of the exact bytes that
// passed). A page whose baseline fingerprint no longer matches the live source
// is re-derived; one that still matches is a no-op candidate (the 4-layer
// no-op proof, layer 3).

// Baseline is one page's durable correctness entry.
type Baseline struct {
	RelPath           string `json:"rel_path"`
	SourceFingerprint string `json:"source_fingerprint"`
	PageVersion       string `json:"page_version"`
	CompletedBy       string `json:"completed_by"`
	CompletedRunID    string `json:"completed_run_id"`
	UpdatedAt         string `json:"updated_at"`
}

// Current reports whether the baseline still proves this page: the source
// fingerprint matches AND the verified bytes still match the live file.
func (b Baseline) Current(sourceFingerprint, livePageHash string) bool {
	return b.SourceFingerprint == sourceFingerprint && b.PageVersion == livePageHash
}

// RecordBaseline upserts one page's baseline and rewrites the projection.
// Call only after a proven, pin-checked page write.
func RecordBaseline(ctx context.Context, st store.Store, root, project, relPath, sourceFingerprint, pageVersion, completedRunID, by string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if err := st.Exec(ctx,
		`INSERT INTO kb_page_baselines (project, rel_path, source_fingerprint, page_version, completed_by, completed_run_id, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(project, rel_path) DO UPDATE SET
			source_fingerprint = excluded.source_fingerprint, page_version = excluded.page_version,
			completed_by = excluded.completed_by, completed_run_id = excluded.completed_run_id,
			updated_at = excluded.updated_at`,
		project, relPath, sourceFingerprint, pageVersion, by, completedRunID, now); err != nil {
		return fmt.Errorf("kb: record baseline %s: %w", relPath, err)
	}
	all, err := Baselines(ctx, st, project)
	if err != nil {
		return err
	}
	return WriteManifestProjection(root, all)
}

// Baselines reads the whole manifest for one project.
func Baselines(ctx context.Context, st store.Store, project string) (map[string]Baseline, error) {
	rows, err := st.Query(ctx,
		`SELECT rel_path, source_fingerprint, page_version, completed_by, completed_run_id, updated_at
		 FROM kb_page_baselines WHERE project = ? ORDER BY rel_path`, project)
	if err != nil {
		return nil, fmt.Errorf("kb: baselines: %w", err)
	}
	out := make(map[string]Baseline, len(rows))
	for _, r := range rows {
		out[s0(r, 0)] = Baseline{
			RelPath: s0(r, 0), SourceFingerprint: s0(r, 1), PageVersion: s0(r, 2),
			CompletedBy: s0(r, 3), CompletedRunID: s0(r, 4), UpdatedAt: s0(r, 5),
		}
	}
	return out, nil
}

// WriteManifestProjection writes index/.page-manifest.json (sorted keys,
// 003 §5.2 shape; atomic tmp+rename).
func WriteManifestProjection(root string, baselines map[string]Baseline) error {
	p := filepath.Join(root, "index", ".page-manifest.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	doc := make(map[string]map[string]string, len(baselines))
	for rel, b := range baselines {
		doc[rel] = map[string]string{
			"sourceFingerprint": b.SourceFingerprint,
			"pageVersion":       b.PageVersion,
			"completedBy":       b.CompletedBy,
			"completedRunId":    b.CompletedRunID,
		}
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(doc))
	for k := range doc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	_ = keys // json.MarshalIndent already sorts map keys
	tmp := fmt.Sprintf("%s.%d.tmp", p, time.Now().UnixNano())
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
