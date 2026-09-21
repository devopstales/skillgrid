package handoff

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteRollup aggregates the hub's snapshots, checkpoints, and handoff refs
// into a markdown rollup written to <repoDir>/.skillgrid/sdd/rollups/<ts>.md.
// It returns the written path. The rollup is a read-only summary (the
// orchestra session-end report equivalent): git change stats + checkpoint
// state + handoff refs.
func WriteRollup(ctx context.Context, h *Hub, repoDir string) (string, error) {
	if h == nil || h.DB == nil {
		return "", fmt.Errorf("handoff hub not initialized")
	}
	snapshots, _ := h.ListSnapshots(ctx, 200)
	checkpoints, _ := h.ListCheckpoints(ctx, 200)
	refs, _ := h.ListHandoffRefs(ctx, 200)

	cutoff := time.Now().Add(-24 * time.Hour)
	inWindow := 0
	for _, s := range snapshots {
		if t, err := time.Parse(time.RFC3339, s.CommittedAt); err == nil && t.After(cutoff) {
			inWindow++
		}
	}

	var b []byte
	b = append(b, []byte("# Rollup: "+time.Now().UTC().Format("2006-01-02 15:04:05 UTC")+"\n\n")...)
	b = append(b, []byte("## Summary\n\n")...)
	b = append(b, []byte(fmt.Sprintf("- **Snapshots total**: %d\n", len(snapshots)))...)
	b = append(b, []byte(fmt.Sprintf("- **Snapshots (last 24h)**: %d\n", inWindow))...)
	b = append(b, []byte(fmt.Sprintf("- **Checkpoints**: %d\n", len(checkpoints)))...)
	b = append(b, []byte(fmt.Sprintf("- **Handoff refs**: %d\n\n", len(refs)))...)

	if len(snapshots) > 0 {
		b = append(b, []byte("## Recent Change Log\n\n")...)
		for i, s := range snapshots {
			if i >= 20 {
				break
			}
			b = append(b, []byte(fmt.Sprintf("- `%s` %s (%s)\n", ShortID(s.Commit), s.Subject, s.Branch))...)
		}
		b = append(b, []byte("\n")...)
	}
	if len(checkpoints) > 0 {
		b = append(b, []byte("## Checkpoints\n\n")...)
		for _, c := range checkpoints {
			b = append(b, []byte(fmt.Sprintf("- `%s` [%s] @ %s (spec: %s)\n", c.Name, c.Status, ShortID(c.Commit), orDash(c.SpecDir)))...)
		}
		b = append(b, []byte("\n")...)
	}
	if len(refs) > 0 {
		b = append(b, []byte("## Handoff Refs\n\n")...)
		for _, r := range refs {
			b = append(b, []byte(fmt.Sprintf("- `%s` (%s) commits %s..%s (spec: %s)\n",
				r.HandoffID, r.HandoffType, orDash(r.FromCommit), orDash(r.ToCommit), orDash(r.SpecDir)))...)
		}
	}

	outDir := filepath.Join(repoDir, ".skillgrid", "sdd", "rollups")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("rollups dir: %w", err)
	}
	p := filepath.Join(outDir, time.Now().UTC().Format("2006-01-02-150405")+".md")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return "", fmt.Errorf("write rollup: %w", err)
	}
	return p, nil
}

// ShortID returns the first 7 chars of a commit (or "-" when empty) for
// compact display in the change log / checkpoints / rollup.
func ShortID(commit string) string {
	if len(commit) >= 7 {
		return commit[:7]
	}
	if commit == "" {
		return "-"
	}
	return commit
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
