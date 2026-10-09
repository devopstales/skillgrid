// Package clm is the Go-side mirror half of the Context Harness (CLM,
// ADR-0027). It renders the non-system turn contents into a 0600 mirror file
// that the Node request path reads and diffs. The system prompt is EXCLUDED —
// it is the prompt-injection surface and is not mirrored.
package clm

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func baseline(t *testing.T, msgs []Message) string {
	t.Helper()
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

func TestRenderMirrorHeader(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "hello", Index: 0}}
	got := RenderMirror(3, "abcnonce", msgs)
	header := "[[LIVE_CONTEXT version=1 revision=3 document=abcnonce baseline=" + baseline(t, msgs) + "]]"
	if !strings.HasPrefix(got, header) {
		t.Fatalf("header mismatch.\ngot:  %q\nwant: %q...", got, header)
	}
}

func TestRenderMirrorOmitsSystemTurns(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "THE SYSTEM PROMPT", Index: 0},
		{Role: "user", Content: "hi", Index: 1},
		{Role: "assistant", Content: "yo", Index: 2},
	}
	got := RenderMirror(0, "nonce0", msgs)
	if strings.Contains(got, "THE SYSTEM PROMPT") {
		t.Fatalf("system turn content must not be mirrored (prompt-injection surface):\n%s", got)
	}
}

func TestRenderMirrorFreshZeroRevision(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "fresh", Index: 0}}
	got := RenderMirror(0, "freshnonce", msgs)
	if !strings.Contains(got, "revision=0") {
		t.Fatalf("fresh revision must be 0:\n%s", got)
	}
}

func TestRenderMirrorPerTurnBlocks(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "sys", Index: 0},
		{Role: "user", Content: "u1", Index: 1},
		{Role: "assistant", Content: "a1", Index: 2},
		{Role: "tool", Content: "t1", Index: 3},
	}
	got := RenderMirror(7, "docnonce", msgs)
	for _, want := range []string{
		"[[CTX_TURN document=docnonce index=1 role=user id=1 protected=false]]",
		"u1",
		"[[CTX_TURN document=docnonce index=2 role=assistant id=2 protected=false]]",
		"a1",
		"[[CTX_TURN document=docnonce index=3 role=tool id=3 protected=false]]",
		"t1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("mirror must contain %q:\n%s", want, got)
		}
	}
	// System turn: no CTX_TURN block, no "role=system".
	if strings.Contains(got, "role=system") {
		t.Fatalf("system turn must be omitted, got:\n%s", got)
	}
}

func TestRenderMirrorBaselineStable12Hex(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "a", Index: 0}, {Role: "assistant", Content: "b", Index: 1}}
	got := RenderMirror(1, "n", msgs)
	wantBase := baseline(t, msgs)
	if !strings.Contains(got, "baseline="+wantBase) {
		t.Fatalf("baseline must be first 12 hex of sha256(non-system contents joined by \\n).\nwant %q\n got:\n%s", wantBase, got)
	}
	// Baseline is stable across renders with different revision/nonce.
	got2 := RenderMirror(2, "different-nonce", msgs)
	if !strings.Contains(got2, "baseline="+wantBase) {
		t.Fatalf("baseline must not depend on revision or nonce")
	}
	if got == got2 {
		t.Fatalf("different revision must produce different mirror")
	}
}

func TestWriteMirror0600AndBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "mirror.md")
	text := RenderMirror(1, "n", []Message{{Role: "user", Content: "x", Index: 0}})
	if err := WriteMirror(path, text); err != nil {
		t.Fatalf("WriteMirror: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat mirror: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("mirror file mode: got %o, want 600", mode)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mirror: %v", err)
	}
	if string(got) != text {
		t.Fatalf("mirror bytes differ:\ngot:  %q\nwant: %q", string(got), text)
	}
}

func TestWriteMirrorCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "mirror.md")
	if err := WriteMirror(path, "text"); err != nil {
		t.Fatalf("WriteMirror must create parent dirs: %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		t.Fatalf("mirror file missing after WriteMirror")
	}
}
