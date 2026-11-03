// Package clm is the Go-side mirror half of the Context Harness (CLM,
// ADR-0027). It renders the non-system turn contents into a 0600 mirror file
// that the Node request path reads and diffs.
//
// The system prompt is EXCLUDED from the mirror — it is the prompt-injection
// surface and is never mirrored. A fresh (empty) session renders revision 0.
// RenderMirror is pure (no I/O); WriteMirror writes the rendered text to a
// 0600 file, creating parent directories as needed.
package clm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Block is one message in the mirror. Index is the turn's 0-based position in
// the session; ID is the stable turn identifier; Role is one of user,
// assistant, tool, or system (system blocks are excluded from the render);
// Protected marks a turn the overflow guard must never withhold; Content is
// the turn text.
type Block struct {
	Index     int
	Role      string
	ID        string
	Protected bool
	Content   string
}

// Mirror is the renderable context: framing (version, revision, nonce,
// baseline) plus the per-turn blocks.
type Mirror struct {
	Version  int
	Revision int
	Nonce    string
	Baseline string
	Blocks   []Block
}

// RenderMirror renders the mirror text (header + CTX_TURN blocks).
//
// The output is:
//
//	[[LIVE_CONTEXT version=1 revision=N document=<nonce> baseline=<digest>]]
//	[[CTX_TURN document=<nonce> index=I role=R id=<id> protected=false]]
//	<turn content>
//	...
//
// baseline is the first 12 hex chars of sha256 over the non-system block
// contents joined by "\n"; it captures the context *content*, stable across
// revision and nonce changes. System blocks are omitted entirely
// (prompt-injection surface).
func RenderMirror(m *Mirror) (string, error) {
	if m == nil {
		return "", fmt.Errorf("clm: render nil mirror")
	}
	baseline := baselineDigest(m.Blocks)
	var b strings.Builder
	fmt.Fprintf(&b, "[[LIVE_CONTEXT version=%d revision=%d document=%s baseline=%s]]\n",
		m.Version, m.Revision, m.Nonce, baseline)
	for _, blk := range m.Blocks {
		if blk.Role == "system" {
			continue
		}
		protected := "false"
		if blk.Protected {
			protected = "true"
		}
		fmt.Fprintf(&b, "[[CTX_TURN document=%s index=%d role=%s id=%s protected=%s]]\n%s\n",
			m.Nonce, blk.Index, blk.Role, blk.ID, protected, blk.Content)
	}
	return b.String(), nil
}

// baselineDigest is the first 12 hex chars of sha256 over the non-system block
// contents joined by "\n".
func baselineDigest(blocks []Block) string {
	var parts []string
	for _, blk := range blocks {
		if blk.Role == "system" {
			continue
		}
		parts = append(parts, blk.Content)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])[:12]
}

// WriteMirror writes the rendered mirror to <dir>/clm-mirror.txt at 0600
// (owner read/write only — the mirror can carry turn contents), creating
// parent directories as needed, and returns the written path.
func WriteMirror(dir string, m *Mirror) (string, error) {
	content, err := RenderMirror(m)
	if err != nil {
		return "", err
	}
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("clm: create mirror dir %q: %w", dir, err)
		}
	}
	path := filepath.Join(dir, "clm-mirror.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("clm: write mirror %q: %w", path, err)
	}
	return path, nil
}
