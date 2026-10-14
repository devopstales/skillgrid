package observability

// snapshot.go is the durable history snapshot (briefing req 9 /
// @step-13 history-snapshot-durable). It mirrors ctx_stats_history to a
// JSON file under the project directory so an operator can inspect the
// rollups without a store handle, and the file survives a re-index (it is
// written atomically and never touched by the index pipeline).

import (
	"encoding/json"
	"os"
	"path/filepath"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// WriteHistorySnapshot serializes the current ctx_stats_history rollups (via
// StatsHistory) to path and writes it atomically (tmp file + rename in the
// same directory). A nil store or a missing ctx_stats_history table yields an
// all-empty snapshot (ADR-0016 fail-open): the file is still written so a
// fresh project has a valid, matchable artifact. The parent directory is
// created when absent.
func WriteHistorySnapshot(s *ctxstore.Store, path string, granularities []Granularity) error {
	h, err := StatsHistory(s, granularities)
	if err != nil {
		// StatsHistory is fail-open (missing table → empty buckets); a
		// returned error is structural — write an empty snapshot anyway so
		// the file's shape never regresses.
		h = &History{Buckets: make(map[Granularity][]Bucket)}
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	// Atomic write: tmp in the same dir, then rename, so a reader never sees
	// a half-written file (the re-index durability step reads the file).
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
