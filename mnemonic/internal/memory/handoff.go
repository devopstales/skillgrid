package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The handoff artifact (014 step 21) is a compact, agent-to-agent context block
// with two halves:
//
//   - a STABLE prefix (hub file summaries + repo file count) that only changes
//     when a hub file (or the file population) changes; and
//   - a DYNAMIC delta (changed file stubs, risk files, recent session events,
//     working set) computed fresh at generation time (not cached).
//
// Detection strategy. The memory package cannot depend on the service layer
// (import cycle), and a fresh CLI process cannot rely on in-process state. We
// therefore persist a handoff cursor into the store's kv_meta table on every
// generation. The delta includes codeindex files whose mtime_ns is newer than
// the stored cursor (i.e. changed since the last handoff) PLUS every hub file
// (so a hub change is always visible even if its mtime predates the cursor).
// This is deterministic, cross-process, and testable without a git dependency.

// handoffKey is the handoff_meta key holding the last-handoff cursor (RFC3339).
const handoffKey = "handoff:cursor"

// hubFilePatterns is the fixed list of "hub" files: small, stable, high-signal
// project metadata (agent config + manifest + entry points). A hub file is any
// codeindex file whose basename or path suffix matches one of these.
var hubFilePatterns = []string{
	"AGENTS.md",
	"CLAUDE.md",
	"go.mod",
	"package.json",
	"main.go",
	"cmd/main.go",
	"internal/main.go",
}

const (
	// hubSummaryLines is the number of leading lines captured per hub file.
	hubSummaryLines = 20
	// stubLines is the number of leading lines captured per changed file stub.
	stubLines = 50
	// recentEventsLimit caps the "recent session events" in the delta.
	recentEventsLimit = 10
)

// Handoff is the agent-to-agent context artifact. Prefix is stable; Delta is
// computed at generation time.
type Handoff struct {
	Prefix      HandoffPrefix `json:"prefix"`
	Delta       HandoffDelta  `json:"delta"`
	GeneratedAt time.Time     `json:"generated_at"`
	ProjectID   string        `json:"project_id"`
}

// HandoffPrefix is the stable half of the handoff. It changes only when a hub
// file changes or the repo file count changes.
type HandoffPrefix struct {
	HubFiles  []HandoffHubFile `json:"hub_files"`
	FileCount int              `json:"file_count"`
}

// HandoffHubFile is a compact summary of one hub file.
type HandoffHubFile struct {
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

// HandoffDelta is the dynamic half of the handoff, computed at generation time.
type HandoffDelta struct {
	ChangedFiles []ChangedFileStub `json:"changed_files"`
	RiskFiles    []string          `json:"risk_files"`
	RecentEvents []RecentEvent     `json:"recent_events"`
	WorkingSet   WorkingSetSummary `json:"working_set"`
}

// ChangedFileStub is a changed file with its leading-line stub.
type ChangedFileStub struct {
	Path string `json:"path"`
	Stub string `json:"stub"`
}

// RecentEvent is a recent session event (from the session_log observations).
type RecentEvent struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

// WorkingSetSummary is a brief summary of the current working state.
type WorkingSetSummary struct {
	ChangedCount int    `json:"changed_count"`
	RiskCount    int    `json:"risk_count"`
	Summary      string `json:"summary"`
}

// GenerateHandoff builds the handoff artifact for the project: a stable prefix
// (hub file summaries + repo file count, derived from the codeindex files table
// and on-disk reads) and a dynamic delta (changed file stubs since the last
// handoff, risk files, recent session events, working set). It records a new
// handoff cursor so the next generation's delta reflects only subsequent
// changes.
func (s *Service) GenerateHandoff(ctx context.Context, repoDir string) (*Handoff, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	cursor, err := s.readHandoffCursor(ctx)
	if err != nil {
		return nil, fmt.Errorf("read handoff cursor: %w", err)
	}

	files, err := s.codeindexFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list codeindex files: %w", err)
	}

	// Build the stable prefix: hub summaries + file count.
	prefix := HandoffPrefix{FileCount: len(files)}
	seen := map[string]bool{}
	for _, f := range files {
		if !isHubFile(f.path) || seen[f.path] {
			continue
		}
		seen[f.path] = true
		summary := readHeadSummary(filepath.Join(repoDir, f.path), hubSummaryLines)
		prefix.HubFiles = append(prefix.HubFiles, HandoffHubFile{Path: f.path, Summary: summary})
	}
	sort.Slice(prefix.HubFiles, func(i, j int) bool { return prefix.HubFiles[i].Path < prefix.HubFiles[j].Path })

	// Build the dynamic delta: changed files since the cursor (+ all hub files
	// so a hub change is always visible), risk files, recent events, working set.
	changed := map[string]bool{}
	if !cursor.IsZero() {
		for _, f := range files {
			if isHubFile(f.path) || fileModifiedSince(filepath.Join(repoDir, f.path), cursor) {
				changed[f.path] = true
			}
		}
	}

	changedPaths := make([]string, 0, len(changed))
	for p := range changed {
		changedPaths = append(changedPaths, p)
	}
	sort.Strings(changedPaths)

	var stubs []ChangedFileStub
	var riskFiles []string
	for _, p := range changedPaths {
		stubs = append(stubs, ChangedFileStub{
			Path: p,
			Stub: readHeadSummary(filepath.Join(repoDir, p), stubLines),
		})
		if isHubFile(p) {
			riskFiles = append(riskFiles, p)
		}
	}

	events, err := s.recentSessionEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("recent session events: %w", err)
	}

	delta := HandoffDelta{
		ChangedFiles: stubs,
		RiskFiles:    riskFiles,
		RecentEvents: events,
		WorkingSet: WorkingSetSummary{
			ChangedCount: len(stubs),
			RiskCount:    len(riskFiles),
			Summary:      fmt.Sprintf("%d changed file(s), %d risk (hub) file(s), %d recent session event(s)", len(stubs), len(riskFiles), len(events)),
		},
	}

	h := &Handoff{
		Prefix:      prefix,
		Delta:       delta,
		GeneratedAt: time.Now().UTC(),
		ProjectID:   s.projectID,
	}

	// Record the new cursor so the NEXT generation reflects only subsequent
	// changes. This runs after the artifact is built (the current delta was
	// computed against the prior cursor).
	if err := s.writeHandoffCursor(ctx, h.GeneratedAt); err != nil {
		return nil, fmt.Errorf("write handoff cursor: %w", err)
	}
	return h, nil
}

// SaveHandoff generates the handoff and writes it to outDir/handoff.latest.json
// (overwriting any prior artifact). It creates outDir if needed.
func (s *Service) SaveHandoff(ctx context.Context, repoDir, outDir string) error {
	h, err := s.GenerateHandoff(ctx, repoDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create handoff dir: %w", err)
	}
	raw, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal handoff: %w", err)
	}
	p := filepath.Join(outDir, "handoff.latest.json")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		return fmt.Errorf("write handoff: %w", err)
	}
	return nil
}

// HandoffPath returns the default handoff artifact path under outDir.
func HandoffPath(outDir string) string {
	return filepath.Join(outDir, "handoff.latest.json")
}

type handoffFile struct {
	path string
}

// codeindexFiles lists the indexed repo files (path + mtime_ns), excluding the
// session pseudo-files (path "session:<id>") that carry no on-disk content.
func (s *Service) codeindexFiles(ctx context.Context) ([]handoffFile, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT path FROM files WHERE path NOT LIKE 'session:%' ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []handoffFile
	for rows.Next() {
		var f handoffFile
		if err := rows.Scan(&f.path); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// isHubFile reports whether a codeindex path is a hub file (matches the fixed
// hubFilePatterns list by basename or path suffix).
func isHubFile(path string) bool {
	rel := filepath.ToSlash(path)
	for _, pat := range hubFilePatterns {
		if rel == pat || strings.HasSuffix(rel, "/"+pat) {
			return true
		}
	}
	return false
}

// readHeadSummary returns the first n lines of the file at abs, joined with
// newlines. A missing/unreadable file yields "" (best-effort; the handoff must
// never fail on an individual file read).
func readHeadSummary(abs string, n int) string {
	raw, err := os.ReadFile(abs)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// recentSessionEvents returns the last recentEventsLimit session_log
// observations for the project (newest first) — the "recent events".
func (s *Service) recentSessionEvents(ctx context.Context) ([]RecentEvent, error) {
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, title, created_at
		FROM observations
		WHERE project = ? AND type = 'session_log' AND deleted_at IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT ?`,
		s.projectID, recentEventsLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RecentEvent
	for rows.Next() {
		var e RecentEvent
		if err := rows.Scan(&e.ID, &e.Title, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// readHandoffCursor returns the last-handoff cursor, or a zero time when no
// handoff has been generated yet.
func (s *Service) readHandoffCursor(ctx context.Context) (time.Time, error) {
	var ts string
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT value FROM handoff_meta WHERE key = ?`, handoffKey).Scan(&ts)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	if ts == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, nil
	}
	return t, nil
}

// writeHandoffCursor persists the handoff cursor (RFC3339) so the next
// generation's delta reflects only subsequent changes.
func (s *Service) writeHandoffCursor(ctx context.Context, ts time.Time) error {
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO handoff_meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		handoffKey, ts.UTC().Format(time.RFC3339Nano))
	return err
}

// fileModifiedSince reports whether the on-disk file at abs was modified after
// cursor. A missing/unstat-able file is treated as not modified (best-effort;
// the handoff must never fail on a single file). This is the changed-file
// detection: it uses the real filesystem mtime (not the codeindex table, which
// only advances when the codeindex indexer runs), so a handoff generated after
// an agent edits files reflects those edits without a re-index.
func fileModifiedSince(abs string, cursor time.Time) bool {
	info, err := os.Stat(abs)
	if err != nil {
		return false
	}
	return info.ModTime().After(cursor)
}
