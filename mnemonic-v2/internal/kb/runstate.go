package kb

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// runstate.go — the resumable run checkpoint (003 §5.1, 011 §3): the
// kb_run_state DB row is AUTHORITATIVE; index/last-update.json is its file
// projection (written for git-diff visibility and human inspection, never read
// back to decide a no-op).
//
// Resume rules (OpenWiki, adapted):
//   - source fingerprint drift → phase resets to planning and the plan is
//     deleted (completed pages stay durable in the baselines); processed
//     survives (hash-keyed — a changed fact is simply not "processed");
//   - no drift → a pending job whose page baseline proves completion for this
//     fingerprint is promoted to complete (the crash-after-page-write window);
//     skipped jobs reset to pending.
//   - clean finish → phase idle + plan cleared + last_full_run_at (full runs);
//     failure → the checkpoint survives intact (never discard resumable work).

// Run phases.
const (
	PhaseIdle       = "idle"
	PhasePlanning   = "planning"
	PhaseGenerating = "generating"
)

// Job statuses (OpenWiki PageJob).
const (
	JobPending  = "pending"
	JobSkipped  = "skipped"
	JobComplete = "complete"
)

// PageJob is one planned page of a generation run.
type PageJob struct {
	RelPath     string `json:"rel_path"`
	Title       string `json:"title"`
	Purpose     string `json:"purpose"`
	Status      string `json:"status"`
	CompletedBy string `json:"completed_by,omitempty"`
}

// RunState is one project's checkpoint.
type RunState struct {
	Project           string            `json:"project"`
	RunID             string            `json:"run_id"`
	Mode              string            `json:"mode"` // init | update | full
	Phase             string            `json:"phase"`
	SourceFingerprint string            `json:"source_fingerprint"`
	Processed         map[string]string `json:"processed"` // sync_id -> fact normalized_hash at processing time
	Plan              []PageJob         `json:"plan"`
	LastFullRunAt     string            `json:"last_full_run_at,omitempty"`
	UpdatedAt         string            `json:"updated_at"`
}

// IsActive reports whether a run is in flight (needs resumption or 409).
func (rs *RunState) IsActive() bool {
	return rs != nil && rs.Phase != PhaseIdle && rs.Phase != ""
}

func newRunID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// LoadRun reads the checkpoint row; a missing row is the idle zero state.
func LoadRun(ctx context.Context, st store.Store, project string) (*RunState, error) {
	rs := &RunState{Project: project, Phase: PhaseIdle, Processed: map[string]string{}}
	rows, err := st.Query(ctx,
		`SELECT run_id, COALESCE(mode,''), phase, source_fingerprint, processed, plan, last_full_run_at
		 FROM kb_run_state WHERE project = ?`, project)
	if err != nil {
		return nil, fmt.Errorf("kb: load run: %w", err)
	}
	if len(rows) == 0 {
		return rs, nil
	}
	rs.RunID = s0(rows[0], 0)
	rs.Mode = s0(rows[0], 1)
	rs.Phase = s0(rows[0], 2)
	rs.SourceFingerprint = s0(rows[0], 3)
	if err := json.Unmarshal([]byte(s0(rows[0], 4)), &rs.Processed); err != nil {
		rs.Processed = map[string]string{}
	}
	if err := json.Unmarshal([]byte(s0(rows[0], 5)), &rs.Plan); err != nil {
		rs.Plan = nil
	}
	rs.LastFullRunAt = s0(rows[0], 6)
	return rs, nil
}

func saveRun(ctx context.Context, st store.Store, rs *RunState) error {
	rs.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	proc, err := json.Marshal(rs.Processed)
	if err != nil {
		return err
	}
	plan, err := json.Marshal(rs.Plan)
	if err != nil {
		return err
	}
	return st.Exec(ctx,
		`INSERT INTO kb_run_state (project, run_id, mode, phase, source_fingerprint, processed, plan, last_full_run_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(project) DO UPDATE SET
			run_id = excluded.run_id, mode = excluded.mode, phase = excluded.phase,
			source_fingerprint = excluded.source_fingerprint, processed = excluded.processed,
			plan = excluded.plan, last_full_run_at = excluded.last_full_run_at,
			updated_at = excluded.updated_at`,
		rs.Project, rs.RunID, rs.Mode, rs.Phase, rs.SourceFingerprint,
		string(proc), string(plan), nullStr(rs.LastFullRunAt), rs.UpdatedAt)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Begin starts a fresh run or resumes the active one (the rules above).
// fingerprint is the CURRENT source fingerprint; mode is init/update/full.
func Begin(ctx context.Context, st store.Store, root, project, mode, fingerprint string) (*RunState, error) {
	prev, err := LoadRun(ctx, st, project)
	if err != nil {
		return nil, err
	}
	if !prev.IsActive() {
		rs := &RunState{
			Project: project, RunID: newRunID(), Mode: mode, Phase: PhasePlanning,
			SourceFingerprint: fingerprint, Processed: prev.Processed,
		}
		if rs.Processed == nil {
			rs.Processed = map[string]string{}
		}
		if err := saveRun(ctx, st, rs); err != nil {
			return nil, err
		}
		return rs, writeLastUpdateProjection(root, rs, "interrupted")
	}

	// Resume the active run.
	prev.Mode = mode
	if fingerprint != prev.SourceFingerprint {
		// Drift: replan. Completed pages stay durable in the baselines.
		prev.Plan = nil
		prev.Phase = PhasePlanning
		prev.SourceFingerprint = fingerprint
	} else {
		baselines, err := Baselines(ctx, st, project)
		if err != nil {
			return nil, err
		}
		for i := range prev.Plan {
			job := &prev.Plan[i]
			switch job.Status {
			case JobSkipped:
				job.Status = JobPending
			case JobPending:
				if b, ok := baselines[job.RelPath]; ok && b.Current(fingerprint, livePageHash(root, job.RelPath)) {
					job.Status = JobComplete
					job.CompletedBy = b.CompletedBy
				}
			}
		}
	}
	if err := saveRun(ctx, st, prev); err != nil {
		return nil, err
	}
	return prev, writeLastUpdateProjection(root, prev, "interrupted")
}

// SetPlan persists the ordered page-job queue and moves the run to generating.
// Job paths must be pages under a page-type folder (the reserved root pages —
// index.md / log.md / reverse-index.md / KNOWLEDGE_SCHEMA.md — are structural).
func SetPlan(ctx context.Context, st store.Store, root, project string, jobs []PageJob) error {
	rs, err := LoadRun(ctx, st, project)
	if err != nil {
		return err
	}
	if !rs.IsActive() {
		return fmt.Errorf("kb: no active run to plan (phase %q)", rs.Phase)
	}
	for i := range jobs {
		if err := ValidatePagePath(jobs[i].RelPath); err != nil {
			return err
		}
		if jobs[i].Status == "" {
			jobs[i].Status = JobPending
		}
	}
	seen := map[string]bool{}
	for _, j := range jobs {
		if seen[j.RelPath] {
			return fmt.Errorf("kb: duplicate planned page: %s", j.RelPath)
		}
		seen[j.RelPath] = true
	}
	rs.Plan = jobs
	rs.Phase = PhaseGenerating
	if err := saveRun(ctx, st, rs); err != nil {
		return err
	}
	return writeLastUpdateProjection(root, rs, "interrupted")
}

// MarkJob transitions one planned job.
func MarkJob(ctx context.Context, st store.Store, root, project, relPath, status, by string) error {
	rs, err := LoadRun(ctx, st, project)
	if err != nil {
		return err
	}
	found := false
	for i := range rs.Plan {
		if rs.Plan[i].RelPath == relPath {
			rs.Plan[i].Status = status
			if by != "" {
				rs.Plan[i].CompletedBy = by
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("kb: unknown planned page: %s", relPath)
	}
	if err := saveRun(ctx, st, rs); err != nil {
		return err
	}
	return writeLastUpdateProjection(root, rs, "interrupted")
}

// Finish settles the run: clean → idle + plan cleared (+ last_full_run_at for
// full runs); failure → the checkpoint survives intact (phase stays
// generating) so the next run reconstructs and retries.
func Finish(ctx context.Context, st store.Store, root, project string, clean, full bool) error {
	rs, err := LoadRun(ctx, st, project)
	if err != nil {
		return err
	}
	status := "interrupted"
	if clean {
		rs.Phase = PhaseIdle
		rs.Plan = nil
		if full {
			rs.LastFullRunAt = time.Now().UTC().Format(time.RFC3339)
		}
		status = "complete"
	}
	if err := saveRun(ctx, st, rs); err != nil {
		return err
	}
	return writeLastUpdateProjection(root, rs, status)
}

// livePageHash hashes the current bytes of one planned page ("none" when the
// file does not exist).
func livePageHash(root, relPath string) string {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil {
		return ""
	}
	return hashBody(string(b))
}

// writeLastUpdateProjection writes index/last-update.json (atomic tmp+rename).
// Projection only — never read back for decisions (003 §5.1).
func writeLastUpdateProjection(root string, rs *RunState, status string) error {
	p := filepath.Join(root, "index", "last-update.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	doc := map[string]string{
		"updatedAt":         time.Now().UTC().Format(time.RFC3339),
		"runId":             rs.RunID,
		"mode":              rs.Mode,
		"phase":             rs.Phase,
		"sourceFingerprint": rs.SourceFingerprint,
		"status":            status,
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", p, time.Now().UnixNano())
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
