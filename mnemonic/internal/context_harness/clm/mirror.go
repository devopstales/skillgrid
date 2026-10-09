// Package clm is the Go-side mirror half of the Context Harness (CLM,
// ADR-0027, TICKET-07). It renders the non-system turn contents into a 0600
// mirror file that the Node request path reads and diffs.
//
// The system prompt is EXCLUDED from the mirror — it is the prompt-injection
// surface and is never mirrored. A fresh (empty) session renders revision 0.
// RenderMirror is pure (no I/O); WriteMirror writes the rendered text to disk
// at 0600, creating parent directories as needed.
package clm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Message is one turn of the session context. Role is one of user, assistant,
// tool, or system; system turns are excluded from the mirror. Index is the
// turn's position in the session (0-based). ID is the stable turn identifier
// (defaults to the index). Content is the turn text.
type Message struct {
	Role    string
	Content string
	Index   int
	ID      int
}

// RenderMirror renders the Context Harness mirror for a session revision.
//
// The output is:
//
//	[[LIVE_CONTEXT version=1 revision=N document=<nonce> baseline=<digest>]]
//	[[CTX_TURN document=<nonce> index=I role=R id=<id> protected=false]]
//	<turn content>
//	...
//
// revision is the session's live revision number (0 for a fresh session).
// nonce is the document nonce identifying this prompt revision. baseline is
// the first 12 hex chars of sha256 over the non-system turn contents joined
// by "\n" (stable across revision/nonce changes). System turns are omitted
// entirely (prompt-injection surface). Every non-system turn gets a CTX_TURN
// block followed by its content.
func RenderMirror(revision int, nonce string, msgs []Message) string {
	var b strings.Builder

	baseline := baselineDigest(msgs)
	fmt.Fprintf(&b, "[[LIVE_CONTEXT version=1 revision=%d document=%s baseline=%s]]\n",
		revision, nonce, baseline)

	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		id := m.ID
		if id == 0 {
			id = m.Index
		}
		fmt.Fprintf(&b, "[[CTX_TURN document=%s index=%d role=%s id=%d protected=false]]\n",
			nonce, m.Index, m.Role, id)
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return b.String()
}

// baselineDigest is the first 12 hex chars of sha256 over the non-system turn
// contents joined by "\n". It is stable across revision and nonce changes —
// it captures the *content* of the context, not its framing.
func baselineDigest(msgs []Message) string {
	var parts []string
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		parts = append(parts, m.Content)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])[:12]
}

// WriteMirror writes text to path at 0600 (owner read/write only — the mirror
// can carry turn contents), creating parent directories as needed. The mirror
// is a real file on disk, not an in-memory buffer.
func WriteMirror(path, text string) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("clm: create mirror dir %q: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		return fmt.Errorf("clm: write mirror %q: %w", path, err)
	}
	return nil
}
