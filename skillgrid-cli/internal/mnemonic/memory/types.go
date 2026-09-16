// Package memory — typed memory categories + LLM dedup + async two-phase
// commit (change 014, step 18).
//
// This file holds the step-18 seams:
//
//   - DedupLLM: the injectable LLM seam for the pre-write semantic-dedup check
//     (18.2). It mirrors the step-05 ExtractionLLM pattern — a small interface
//     a backend implements, NO hard-coded HTTP call. Nil / disabled = the
//     deterministic hash-dedup fallback (the default-off behavior).
//   - SessionCommit / the async two-phase commit (18.3): the sync phase
//     (durable write + compression_index increment) returns before the async
//     phase (LLM extraction + dedup + memory_diff.json) runs.
package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DedupLLM is the pluggable LLM seam for the pre-write semantic-dedup check
// (014, step 18.2). It mirrors the step-05 ExtractionLLM seam pattern: a small
// interface a backend implements, with NO CGo / hard-coded HTTP client in this
// package. The method is handed the new observation's content plus a small
// pre-filtered set of candidate existing contents (the cheap SQL pre-filter
// result) and returns whether the new content is a semantic duplicate of any
// candidate, and (when yes) the candidate's observation id so the caller can
// merge into it rather than write a new row. A nil seam (or one that errors)
// leaves dedup to the deterministic hash fallback — the fallback is always
// available.
type DedupLLM interface {
	// Dedup reports whether newContent is a semantic duplicate of any of the
	// candidate contents. duplicateID is the observation id to merge into when
	// duplicate is true (0 when false / unknown).
	Dedup(ctx context.Context, newContent string, candidates []string) (duplicate bool, duplicateID int64, err error)
}

// dedupPreFilterLimit caps how many candidate observations the SQL pre-filter
// hands to the LLM. Keeping the set small bounds the LLM call cost; the
// pre-filter already narrowed to same-project, non-deleted rows.
const dedupPreFilterLimit = 20

// candidateDedupResult is the outcome of the pre-write dedup check.
type candidateDedupResult struct {
	// MergedInto, when > 0, is the id of the existing observation the new save
	// merged into (the duplicate was absorbed; no new row was written).
	MergedInto int64
	// Skipped reports whether the write was skipped because of a duplicate.
	Skipped bool
	// Reason is one of "", "hash", "llm" — which mechanism decided.
	Reason string
}

// dedupPreFilter returns the candidate contents the LLM should compare the
// new content against: the project's non-deleted observations (most recent
// first, capped). This is the cheap SQL pre-filter that stands in for vector
// pre-filtering (no embeddings required); it bounds the LLM input.
func (s *Service) dedupPreFilter(ctx context.Context, limit int) []string {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT content FROM observations
		WHERE project = ? AND deleted_at IS NULL
		ORDER BY created_at DESC, id DESC
		LIMIT ?`,
		s.projectID, limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c sql.NullString
		if err := rows.Scan(&c); err != nil {
			continue
		}
		if c.Valid && strings.TrimSpace(c.String) != "" {
			out = append(out, c.String)
		}
	}
	return out
}

// dedupCandidateID finds the observation id whose content matches one of the
// pre-filter candidates the LLM flagged as a duplicate. When the LLM returns a
// positive id we trust it; otherwise we fall back to the most recent candidate.
func (s *Service) dedupCandidateID(ctx context.Context, llmID int64, candidates []string) int64 {
	if llmID > 0 {
		var n int
		if err := s.store.DB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM observations WHERE id = ? AND project = ? AND deleted_at IS NULL`,
			llmID, s.projectID,
		).Scan(&n); err == nil && n > 0 {
			return llmID
		}
	}
	// Fall back to the most recent candidate row (the head of the pre-filter).
	var id int64
	if err := s.store.DB.QueryRowContext(ctx,
		`SELECT id FROM observations WHERE project = ? AND deleted_at IS NULL ORDER BY created_at DESC, id DESC LIMIT 1`,
		s.projectID,
	).Scan(&id); err == nil {
		return id
	}
	return 0
}

// runDedupCheck performs the pre-write dedup (18.2): when the LLM pass is
// enabled and a seam is attached, pre-filter candidates and ask the LLM; on a
// duplicate verdict return the merge target so the caller absorbs the new save
// into it (no new row). When the LLM is disabled, absent, or errors, fall back
// to the deterministic hash dedup (the existing 24h normalized-hash path) —
// the caller then proceeds normally and the hash check does the work.
//
// It returns (mergeInto, reason): mergeInto > 0 means "do not write a new row;
// bump duplicate_count on this id". reason is "llm" when the LLM decided, "hash"
// when the hash fallback path is left to the caller, and "" when no dedup
// mechanism is armed.
func (s *Service) runDedupCheck(ctx context.Context, content string) (int64, string) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, ""
	}
	// LLM pass: OPT-IN. Armed only when enabled AND a seam is attached.
	if s.dedupLLMEnabled && s.dedupLLM != nil {
		candidates := s.dedupPreFilter(ctx, dedupPreFilterLimit)
		if len(candidates) == 0 {
			return 0, ""
		}
		dup, llmID, err := s.dedupLLM.Dedup(ctx, content, candidates)
		if err != nil {
			// Best-effort warning: the hash fallback still runs, so no error is
			// propagated to the caller (mirrors the step-05 LLM-failure
			// fallback).
			fmt.Fprintf(logWriter(), "mnemonic: dedup LLM failed, falling back to hash: %v\n", err)
			return 0, "hash"
		}
		if dup {
			id := s.dedupCandidateID(ctx, llmID, candidates)
			if id > 0 {
				return id, "llm"
			}
			return 0, "llm"
		}
		return 0, "llm"
	}
	// No LLM armed: the caller's existing hash dedup is the fallback.
	return 0, "hash"
}

// SessionCommit is the two-phase session commit entry point (014, step 18.3).
//
//   - SYNC phase (durable, must complete before return): write the session's
//     messages/observations and increment the session's compression_index. The
//     caller's write is durable the moment SessionCommit returns.
//   - ASYNC phase (non-blocking, best-effort): a goroutine runs LLM extraction
//     + dedup over the session content and writes memory_diff.json (in the
//     project's data directory) for auditing. It never blocks the sync path
//     and never fails the commit.
//
// The test seam SetAsyncCommitGate lets a test install a WaitGroup the async
// phase signals, so the test can wait for the async phase without time.Sleep.
func (s *Service) SessionCommit(ctx context.Context, sessionID, summary string) (compressionIndex int64, err error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, fmt.Errorf("memory service not initialized")
	}
	if strings.TrimSpace(sessionID) == "" {
		return 0, fmt.Errorf("session_id is required")
	}

	// ── SYNC phase: durable write + compression_index increment ──────────────
	// The message (the session summary) is written first so the record is
	// durable; then the session's compression_index is incremented (this is the
	// "commit" counter — each commit advances it by one). Both must succeed
	// before SessionCommit returns.
	if strings.TrimSpace(summary) != "" {
		if _, err := s.store.DB.ExecContext(ctx, `
			UPDATE sessions
			SET summary = ?,
			    title = COALESCE(NULLIF(TRIM(title), ''), ?)
			WHERE id = ? AND project = ?`,
			summary, deriveSessionTitle(summary), sessionID, s.projectID,
		); err != nil {
			return 0, fmt.Errorf("write session message: %w", err)
		}
	}
	var n int
	if err := s.store.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE id = ? AND project = ?`,
		sessionID, s.projectID,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("verify session: %w", err)
	}
	if n == 0 {
		return 0, fmt.Errorf("session %s not found", sessionID)
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE sessions SET compression_index = COALESCE(compression_index, 0) + 1
		WHERE id = ? AND project = ?`,
		sessionID, s.projectID,
	)
	if err != nil {
		return 0, fmt.Errorf("increment compression_index: %w", err)
	}
	if _, err := res.RowsAffected(); err != nil {
		return 0, fmt.Errorf("compression_index rows affected: %w", err)
	}
	if err := s.store.DB.QueryRowContext(ctx,
		`SELECT COALESCE(compression_index, 0) FROM sessions WHERE id = ? AND project = ?`,
		sessionID, s.projectID,
	).Scan(&compressionIndex); err != nil {
		return 0, fmt.Errorf("read compression_index: %w", err)
	}

	// ── ASYNC phase: extraction + dedup + memory_diff.json ───────────────────
	// Detached goroutine: it does NOT block the sync return and a failure is
	// swallowed (best-effort audit). The async phase is armed only when the LLM
	// extraction pass is enabled and a seam is attached — otherwise there is
	// nothing to extract, so no diff is written (the default-off behavior: the
	// sync path is unchanged and no async work is spawned).
	if s.extractionLLMEnabled && s.extractionLLM != nil {
		wg := currentAsyncCommitGate()
		go func() {
			s.asyncExtractAndDiff(context.WithoutCancel(ctx), sessionID, summary)
			if wg != nil {
				wg.Done()
			}
		}()
	}
	return compressionIndex, nil
}

// asyncExtractAndDiff is the async half of SessionCommit (18.3): run the LLM
// extraction over the session summary, dedup the extracted items against the
// existing store, and write memory_diff.json to the project's data directory
// recording what was extracted and what was deduped. Every step is
// best-effort — an error is logged to the diff file's "error" field, never
// propagated.
func (s *Service) asyncExtractAndDiff(ctx context.Context, sessionID, summary string) {
	diff := memoryDiff{
		SessionID:   sessionID,
		Project:     s.projectID,
		Compression: "async",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	if items, err := ExtractWithLLM(ctx, summary, s.extractionLLM); err != nil {
		diff.Error = err.Error()
	} else {
		for _, item := range items {
			title, typ := shapePassiveItem(item)
			if title == "" || !IsValidType(typ) {
				continue
			}
			entry := memoryDiffEntry{
				Title: truncateTitle(title, 120),
				Type:  typ,
			}
			// Dedup: is this extracted learning already in the store?
			dup, id, dErr := s.isExtractedDuplicate(ctx, item.Text, title, typ)
			if dErr != nil {
				entry.Duplicate = false
			} else {
				entry.Duplicate = dup
				entry.MergedInto = id
			}
			diff.Extracted = append(diff.Extracted, entry)
		}
	}
	s.writeMemoryDiff(diff)
}

// isExtractedDuplicate checks (via the LLM seam when available, else by exact
// content) whether an extracted learning is already stored. It returns the
// merged-into id when duplicate.
func (s *Service) isExtractedDuplicate(ctx context.Context, text, title, typ string) (bool, int64, error) {
	if s.dedupLLMEnabled && s.dedupLLM != nil {
		candidates := s.dedupPreFilter(ctx, dedupPreFilterLimit)
		if len(candidates) == 0 {
			return false, 0, nil
		}
		dup, id, err := s.dedupLLM.Dedup(ctx, text, candidates)
		if err != nil {
			return false, 0, err
		}
		if dup {
			return true, s.dedupCandidateID(ctx, id, candidates), nil
		}
		return false, 0, nil
	}
	// Hash fallback: exact normalized-hash match within the project.
	hash := normalizedHash(title, text, typ)
	var id int64
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT id FROM observations
		WHERE project = ? AND normalized_hash = ? AND deleted_at IS NULL
		ORDER BY id DESC LIMIT 1`,
		s.projectID, hash,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, id, nil
}

// writeMemoryDiff writes memory_diff.json to the project's data directory
// (<dataDir>/<project>/memory_diff.json) for auditing. It is best-effort: a
// write failure is logged to stderr, never returned.
func (s *Service) writeMemoryDiff(diff memoryDiff) {
	dataDir := ""
	if s != nil && s.store != nil && s.store.DB != nil {
		dataDir = filepath.Dir(s.store.Path())
	}
	if dataDir == "" || dataDir == "." {
		return
	}
	dir := filepath.Join(dataDir, s.projectID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(logWriter(), "mnemonic: memory_diff mkdir: %v\n", err)
		return
	}
	b, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		fmt.Fprintf(logWriter(), "mnemonic: memory_diff marshal: %v\n", err)
		return
	}
	path := filepath.Join(dir, "memory_diff.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fmt.Fprintf(logWriter(), "mnemonic: memory_diff write: %v\n", err)
		return
	}
}

// memoryDiff is the audit record written by the async commit phase (18.3):
// which items the LLM extracted, which were deduped, and any extraction error.
type memoryDiff struct {
	SessionID   string            `json:"session_id"`
	Project     string            `json:"project"`
	Compression string            `json:"compression"`
	Timestamp   string            `json:"timestamp"`
	Extracted   []memoryDiffEntry `json:"extracted,omitempty"`
	Error       string            `json:"error,omitempty"`
}

type memoryDiffEntry struct {
	Title      string `json:"title"`
	Type       string `json:"type"`
	Duplicate  bool   `json:"duplicate"`
	MergedInto int64  `json:"merged_into,omitempty"`
}

// asyncCommitGateMu guards asyncCommitGate. The gate is a test seam: a
// non-nil WaitGroup the async phase signals when it completes, so a test can
// wait for the async phase deterministically (no time.Sleep).
var (
	asyncCommitGateMu sync.Mutex
	asyncCommitGate   *sync.WaitGroup
)

// SetAsyncCommitGate installs a WaitGroup the async commit phase signals (one
// Done per spawned goroutine). A nil gate disables the seam (production). It is
// a package-level seam (not on *Service) so a test can arm it without a
// Service field; it is process-global and must only be used in tests.
func SetAsyncCommitGate(g *sync.WaitGroup) {
	asyncCommitGateMu.Lock()
	defer asyncCommitGateMu.Unlock()
	asyncCommitGate = g
}

// currentAsyncCommitGate returns the installed gate (nil in production).
func currentAsyncCommitGate() *sync.WaitGroup {
	asyncCommitGateMu.Lock()
	defer asyncCommitGateMu.Unlock()
	return asyncCommitGate
}
