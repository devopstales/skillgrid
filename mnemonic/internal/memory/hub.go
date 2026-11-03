package memory

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Hub file identification, change-impact analysis, and observation risk
// scoring (014, step 23).
//
// A HUB file is a file whose import fan-in is high: 3 or more distinct
// importer files (edges of kind 'imports' pointing at one of its symbols,
// counted by DISTINCT from-file). Changing a hub file risks every file that
// imports it, so the hub analysis feeds the change-impact view and the
// observation risk scores (an observation about a hub file is riskier to
// trust than one about a leaf).
//
// All three concerns are OPT-IN and read-mostly: nothing runs on save by
// default, the hub_score is stamped on symbols only when IdentifyHubFiles is
// called, and the risk_score is stamped on observations only when
// RecomputeRiskScores is called. A file with fewer than MinHubImporters
// (3) is a non-hub; its hub_score and any dependent observation's
// risk_score are 0.

// MinHubImporters is the default importer-fan-in threshold for hub
// identification: a file imported by 3+ distinct files is a hub (014 step
// 23.1). Tunable via the mnemonic.hub.min_importers config key (SetHub).
const MinHubImporters = 3

// HighImpactDependents is the default dependent-file count at which a hub
// change is classified as "high" impact (vs. "medium" for a hub with fewer
// dependents). Tunable via the mnemonic.hub.high_impact_dependents config key.
const HighImpactDependents = 5

// Impact levels, most to least severe.
const (
	ImpactHigh   = "high"
	ImpactMedium = "medium"
	ImpactLow    = "low"
)

// HubFile is one file's import-fan-in profile (014 step 23.1).
type HubFile struct {
	// FilePath is the codeindex files.path of the analyzed file.
	FilePath string
	// ImporterCount is the number of DISTINCT files that import this file
	// (edges kind='imports' targeting one of the file's symbols).
	ImporterCount int
	// IsHub is true when ImporterCount >= the minimum-importers threshold.
	IsHub bool
	// HubScore is ImporterCount / total_files, the hub ratio in [0, 1]
	// (importers / total_files, 014 step 23.1).
	HubScore float64
}

// ImpactResult is one changed file's classification (014 step 23.2).
type ImpactResult struct {
	// FilePath is the changed file as given to AnalyzeImpact.
	FilePath string
	// IsHub is true when the file meets the hub threshold.
	IsHub bool
	// DependentCount is the number of distinct files that import the file
	// (0 for a non-hub: the change has no import fan-out to ripple through).
	DependentCount int
	// ImpactLevel is high (hub, dependents >= HighImpactDependents),
	// medium (hub, fewer dependents), or low (non-hub).
	ImpactLevel string
}

// RiskEntry is one observation's risk row for the `mem graph --risk` view
// (014 step 23.4): the stored risk_score plus the dependent count of the
// file the observation references.
type RiskEntry struct {
	ObservationID  int64   `json:"observation_id"`
	Title          string  `json:"title"`
	FilePath       string  `json:"file_path,omitempty"`
	RiskScore      float64 `json:"risk_score"`
	DependentCount int     `json:"dependent_count"`
}

// RiskOptions tunes the `mem graph --risk` query (014 step 23.4).
type RiskOptions struct {
	// Threshold filters entries to risk_score >= Threshold (default
	// DefaultRiskThreshold = 0.5).
	Threshold float64
	// Limit caps the result set (0 = no cap).
	Limit int
}

// DefaultRiskThreshold is the default minimum risk_score for the
// `mem graph --risk` view (014 step 23.4).
const DefaultRiskThreshold = 0.5

// hubConfig tunes hub identification + impact classification (014 step 23.4:
// "make the threshold configurable"). Zero fields fall back to the
// MinImporters / HighImpactDependents defaults, so a Service built
// directly (unit tests) or a config without the section keeps production
// behavior. Set via SetHub from the mnemonic.hub config section.
type hubConfig struct {
	MinImporters         int
	HighImpactDependents int
}

func (c hubConfig) effective() hubConfig {
	if c.MinImporters <= 0 {
		c.MinImporters = MinHubImporters
	}
	if c.HighImpactDependents <= 0 {
		c.HighImpactDependents = HighImpactDependents
	}
	return c
}

// SetHub configures the hub/impact thresholds (014 step 23.4: "make the
// threshold configurable"). Zero fields keep the defaults.
func (s *Service) SetHub(cfg hubConfig) {
	if s == nil {
		return
	}
	s.hubCfg = cfg.effective()
}

// effectiveHubCfg returns the active hub configuration, filling any unset
// field with its default (a Service built directly in unit tests).
func (s *Service) effectiveHubCfg() hubConfig {
	if s == nil {
		return hubConfig{}.effective()
	}
	return s.hubCfg.effective()
}

// importersForFiles counts, per target file, the DISTINCT importing files
// (edges kind='imports', active temporal window: valid_from <= now AND
// (valid_to IS NULL OR valid_to > now) — the step-10 current-state filter).
// An importer is a distinct from-file (the edge's file_id — the file that
// owns the importing symbol), not a distinct edge: multiple symbols or
// repeated import lines in the same importing file count once.
func importersForFiles(ctx context.Context, db *sql.DB, fileIDs []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(fileIDs))
	if len(fileIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(fileIDs))
	for i := range fileIDs {
		placeholders[i] = "?"
	}
	now := time.Now().Unix()
	// Arg order must match the SQL placeholder order:
	//   ? (valid_from)  → now
	//   ? (valid_to)    → now
	//   ? … ? (IN list) → fileIDs
	args := make([]any, 0, len(fileIDs)+2)
	args = append(args, now, now)
	for _, id := range fileIDs {
		args = append(args, id)
	}
	q := `
		SELECT s2.file_id AS to_file, COUNT(DISTINCT e.file_id) AS importers
		FROM edges e
		JOIN symbols s2 ON s2.id = e.to_id
		WHERE e.kind = 'imports'
		  AND e.to_id IS NOT NULL
		  AND e.valid_from <= ?
		  AND (e.valid_to IS NULL OR e.valid_to > ?)
		  AND s2.file_id IN (` + joinPlaceholders(placeholders) + `)
		GROUP BY s2.file_id`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("importers per file: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var toFile int64
		var n int
		if err := rows.Scan(&toFile, &n); err != nil {
			return nil, fmt.Errorf("importers per file scan: %w", err)
		}
		out[toFile] = n
	}
	return out, rows.Err()
}

// joinPlaceholders renders a comma-joined "?" placeholder list for an IN
// clause.
func joinPlaceholders(ps []string) string {
	out := ""
	for i, p := range ps {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

// timeUnixNow is the "now" anchor for temporal current-state filters
// (valid_from <= now / valid_to > now). Extracted as a named helper so the
// query construction stays readable; tests seed edges with valid_from = 0,
// so any real now keeps them active.
func timeUnixNow() int64 {
	return time.Now().Unix()
}

// totalIndexedFileCount is the denominator of the hub ratio: every file in
// the codeindex for this store (the files table is per-project-store, so no
// project predicate is needed).
func totalIndexedFileCount(ctx context.Context, db *sql.DB) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM files`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count files: %w", err)
	}
	return n, nil
}

// IdentifyHubFiles identifies the project's hub files (014 step 23.1): it
// counts the distinct importers of every indexed file, flags the files with
// 3+ importers (the MinHubImporters threshold, tunable via SetHub), computes
// hub_score = importers / total_files, and stores the hub_score on each
// symbol row of the file (0 for non-hub files, so a stamped 0 distinguishes
// "analyzed, not a hub" from the NULL "never analyzed"). Returns the hub
// files only (IsHub = true), sorted by importer count descending then path.
//
// The pass is idempotent: re-running it after import edges change re-stamps
// every symbol row to the current ratio.
func (s *Service) IdentifyHubFiles(ctx context.Context) ([]HubFile, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	cfg := s.effectiveHubCfg()
	db := s.store.DB

	var fileIDs []int64
	rows, err := db.QueryContext(ctx, `SELECT id FROM files ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("list files scan: %w", err)
		}
		fileIDs = append(fileIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("list files: %w", err)
	}
	rows.Close()

	total, err := totalIndexedFileCount(ctx, db)
	if err != nil {
		return nil, err
	}
	if total <= 0 {
		// Empty index: no hubs, but still stamp nothing (no symbols).
		return nil, nil
	}
	importers, err := importersForFiles(ctx, db, fileIDs)
	if err != nil {
		return nil, err
	}

	hubs := make([]HubFile, 0)
	for _, fileID := range fileIDs {
		n := importers[fileID]
		var path string
		if err := db.QueryRowContext(ctx, `SELECT path FROM files WHERE id = ?`, fileID).Scan(&path); err != nil {
			return nil, fmt.Errorf("file path %d: %w", fileID, err)
		}
		hf := HubFile{
			FilePath:      path,
			ImporterCount: n,
			IsHub:         n >= cfg.MinImporters,
		}
		if total > 0 {
			hf.HubScore = float64(n) / float64(total)
		}
		score := 0.0
		if hf.IsHub {
			score = hf.HubScore
		}
		// Stamp the file-level ratio on every symbol row of the file so
		// the risk pass (23.3) can join obs → symbol → hub_score in one
		// hop. Best-effort per file: a stamp failure must not abort the
		// whole pass (advisory column).
		if _, serr := db.ExecContext(ctx,
			`UPDATE symbols SET hub_score = ? WHERE file_id = ?`, score, fileID); serr != nil {
			return nil, fmt.Errorf("stamp hub_score for file %s: %w", path, serr)
		}
		if hf.IsHub {
			hubs = append(hubs, hf)
		}
	}
	sort.Slice(hubs, func(a, b int) bool {
		if hubs[a].ImporterCount != hubs[b].ImporterCount {
			return hubs[a].ImporterCount > hubs[b].ImporterCount
		}
		return hubs[a].FilePath < hubs[b].FilePath
	})
	return hubs, nil
}

// AnalyzeImpact classifies a change set (014 step 23.2): it runs the hub
// pass (so the classification reflects the current import graph and re-
// stamps hub_scores), then, for each changed file, reports whether it is a
// hub, how many files depend on it, and the impact level:
//
//	high:   hub with dependents >= HighImpactDependents
//	medium: hub with fewer dependents
//	low:    non-hub (DependentCount 0)
//
// Changed files are matched by files.path (exact) or by suffix (a relative
// path matches an indexed absolute path ending in the same relative tail),
// mirroring graphRefForSource. Results are returned in the input order.
func (s *Service) AnalyzeImpact(ctx context.Context, changedFiles []string) ([]ImpactResult, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	if len(changedFiles) == 0 {
		return nil, nil
	}
	hubs, err := s.IdentifyHubFiles(ctx)
	if err != nil {
		return nil, err
	}
	hubByPath := make(map[string]HubFile, len(hubs))
	for _, h := range hubs {
		hubByPath[h.FilePath] = h
	}
	results := make([]ImpactResult, 0, len(changedFiles))
	for _, cf := range changedFiles {
		r := ImpactResult{FilePath: cf, ImpactLevel: ImpactLow}
		if h, ok := hubByPath[cf]; ok {
			r.IsHub = true
			r.DependentCount = h.ImporterCount
			if h.ImporterCount >= s.effectiveHubCfg().HighImpactDependents {
				r.ImpactLevel = ImpactHigh
			} else {
				r.ImpactLevel = ImpactMedium
			}
			results = append(results, r)
			continue
		}
		// Suffix match: a relative changed path against indexed absolute
		// paths (graphRefForSource semantics).
		if matched, ok := s.matchIndexedPath(ctx, cf); ok {
			if h, isHub := hubByPath[matched]; isHub {
				r.IsHub = true
				r.DependentCount = h.ImporterCount
				if h.ImporterCount >= s.effectiveHubCfg().HighImpactDependents {
					r.ImpactLevel = ImpactHigh
				} else {
					r.ImpactLevel = ImpactMedium
				}
			}
		}
		results = append(results, r)
	}
	return results, nil
}

// matchIndexedPath resolves a changed file path against the codeindex files
// table: exact path first, then a suffix match (the changed path is a tail of
// the indexed path). Returns the indexed path.
func (s *Service) matchIndexedPath(ctx context.Context, path string) (string, bool) {
	var p string
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT path FROM files WHERE path = ? OR path LIKE '%' || ? LIMIT 1`, path, path).Scan(&p)
	if err != nil {
		return "", false
	}
	return p, true
}

// RecomputeRiskScores stamps risk_score on every live observation of this
// project (014 step 23.3): risk_score is the hub_score of the file the
// observation references (observations.graph_ref → symbols.file_id → the
// file's stamped hub_score), 0 for observations with no graph_ref or whose
// referenced file is not a hub. It first runs the hub pass so the risk is
// computed against the current import graph; re-running it after import
// edges change re-stamps the scores. Best-effort per observation: a risk
// stamp is advisory and never blocks anything.
func (s *Service) RecomputeRiskScores(ctx context.Context) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return fmt.Errorf("memory service not initialized")
	}
	if _, err := s.IdentifyHubFiles(ctx); err != nil {
		return err
	}
	_, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations
		SET risk_score = COALESCE((
			SELECT COALESCE(s.hub_score, 0)
			FROM symbols s
			WHERE s.id = observations.graph_ref
		), 0)
		WHERE project = ? AND deleted_at IS NULL`, s.projectID)
	if err != nil {
		return fmt.Errorf("recompute risk scores: %w", err)
	}
	return nil
}

// RiskReport lists the project's highest-risk observations for the
// `mem graph --risk` view (014 step 23.4): observations ordered by
// risk_score DESC, filtered to risk_score >= Threshold (default
// DefaultRiskThreshold), each annotated with the referenced file's path and
// dependent count.
func (s *Service) RiskReport(ctx context.Context, opts RiskOptions) ([]RiskEntry, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	if opts.Threshold < 0 {
		opts.Threshold = DefaultRiskThreshold
	}
	q := `
		SELECT o.id, o.title, COALESCE(o.risk_score, 0),
			COALESCE(f.path, ''),
			COALESCE((SELECT COUNT(DISTINCT e2.file_id)
				FROM edges e2
				WHERE e2.kind = 'imports' AND e2.to_id = o.graph_ref
				  AND e2.valid_from <= ? AND (e2.valid_to IS NULL OR e2.valid_to > ?)), 0)
		FROM observations o
		LEFT JOIN symbols sy ON sy.id = o.graph_ref
		LEFT JOIN files f ON f.id = sy.file_id
		WHERE o.project = ? AND o.deleted_at IS NULL AND o.risk_score >= ?
		ORDER BY o.risk_score DESC, o.id`
	now := time.Now().Unix()
	args := []any{now, now, s.projectID, opts.Threshold}
	if opts.Limit > 0 {
		q += " LIMIT " + strconv.Itoa(opts.Limit)
	}
	rows, err := s.store.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("risk report: %w", err)
	}
	defer rows.Close()
	out := make([]RiskEntry, 0)
	for rows.Next() {
		var e RiskEntry
		if err := rows.Scan(&e.ObservationID, &e.Title, &e.RiskScore, &e.FilePath, &e.DependentCount); err != nil {
			return nil, fmt.Errorf("risk report scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
