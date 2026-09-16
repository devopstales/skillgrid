// Package relay owns the Session Relay: the SQL writes for
// session_handoffs / session_archives and the .skillgrid/.cleave/ filesystem
// bundle (PROGRESS / KNOWLEDGE / NEXT_PROMPT). MCP and CLI are thin callers of
// the same Handoff / Resume interfaces (change 006, step 02).
package relay

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cleaveDir is the repo-relative cleave bundle directory under .skillgrid/.
// It is gitignored by default (like the 003 L0 scratch).
const cleaveDir = ".skillgrid/.cleave"

// The three cleave bundle files. PROGRESS (what was done), KNOWLEDGE
// (decisions / learnings), and NEXT_PROMPT (the prompt that seeds the next
// session). NEXT_PROMPT is the resume target: session_resume returns its
// content verbatim (never invented).
const (
	FileProgress   = "PROGRESS.md"
	FileKnowledge  = "KNOWLEDGE.md"
	FileNextPrompt = "NEXT_PROMPT.md"
)

// L0Dir is the soft-optional 003 tiered-storage session workspace under
// .skillgrid/workspace/sessions/{id}/. It is SOFT: a handoff works without it
// (the bundle degrades gracefully) and never errors on its absence. When
// present, the cleave files are mirrored into it so the L0 tree stays
// coherent.
const L0Dir = ".skillgrid/workspace/sessions"

// Bundle is the structured cleave bundle content written at handoff time.
type Bundle struct {
	Progress   string
	Knowledge  string
	NextPrompt string
	// SourceSession is the sessions.id this handoff was taken from (recorded
	// on the session_handoffs row). Optional.
	SourceSession string
	// ContextSummary is an optional short context note recorded on the row.
	ContextSummary string
}

// BundleDir returns the absolute cleave directory for projectRoot.
func BundleDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".skillgrid", ".cleave")
}

// WriteBundle writes the three cleave files under projectRoot/.skillgrid/.cleave/.
// It returns the three absolute paths. A write failure removes the partially
// written files so a failed handoff leaves no half-bundle (fail closed).
// Soft-optional L0: if projectRoot/.skillgrid/workspace/sessions/{sessionID}/
// exists, the bundle is mirrored there too (best-effort; an L0 mirror failure
// does not fail the handoff).
func WriteBundle(projectRoot string, sessionID string, h Bundle) ([]string, error) {
	dir := BundleDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("relay: create cleave dir: %w", err)
	}

	files := []struct {
		name string
		body string
	}{
		{FileProgress, h.Progress},
		{FileKnowledge, h.Knowledge},
		{FileNextPrompt, h.NextPrompt},
	}
	paths := make([]string, 0, len(files))
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		if err := os.WriteFile(p, []byte(f.body), 0o644); err != nil {
			// Fail closed: roll back every file written so far so the failed
			// handoff leaves no orphan cleave files behind.
			for _, done := range paths {
				_ = os.Remove(done)
			}
			return nil, fmt.Errorf("relay: write %s: %w", f.name, err)
		}
		paths = append(paths, p)
	}

	// Soft-optional L0 mirror: fold the bundle into the 003 session workspace
	// when it is present. Absent L0 is a no-op (degrade, don't error).
	mirrorL0(projectRoot, sessionID, files)
	return paths, nil
}

// mirrorL0 copies the cleave bundle into .skillgrid/workspace/sessions/{id}/
// when that directory exists. Best-effort: any error is swallowed because L0
// is soft-optional — the handoff succeeds even if the mirror fails.
func mirrorL0(projectRoot, sessionID string, files []struct {
	name string
	body string
}) {
	id := strings.TrimSpace(sessionID)
	if id == "" || strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return
	}
	l0Dir := filepath.Join(projectRoot, L0Dir, id)
	st, err := os.Stat(l0Dir)
	if err != nil || !st.IsDir() {
		return // no L0 tree — degrade gracefully
	}
	if err := os.MkdirAll(l0Dir, 0o755); err != nil {
		return
	}
	for _, f := range files {
		_ = os.WriteFile(filepath.Join(l0Dir, f.name), []byte(f.body), 0o644)
	}
}

// ReadBundle reads the three cleave files for projectRoot. A missing
// .cleave/ directory or any missing file returns an error (fail closed) —
// session_resume must not invent prompt content from an absent bundle.
func ReadBundle(projectRoot string) (h Bundle, err error) {
	dir := BundleDir(projectRoot)
	read := func(name string) (string, error) {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("relay: missing cleave file %s (no .cleave/ bundle)", name)
			}
			return "", fmt.Errorf("relay: read %s: %w", name, err)
		}
		return string(b), nil
	}
	h.Progress, err = read(FileProgress)
	if err != nil {
		return Bundle{}, err
	}
	h.Knowledge, err = read(FileKnowledge)
	if err != nil {
		return Bundle{}, err
	}
	h.NextPrompt, err = read(FileNextPrompt)
	if err != nil {
		return Bundle{}, err
	}
	return h, nil
}
