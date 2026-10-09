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

// baselineDigestOf is the test-side oracle: first 12 hex chars of sha256 over
// the non-system block contents joined by "\n".
func baselineDigestOf(blocks []Block) string {
	var parts []string
	for _, b := range blocks {
		if b.Role == "system" {
			continue
		}
		parts = append(parts, b.Content)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])[:12]
}

func TestRenderMirrorHeader(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 3, Nonce: "abcnonce"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "user", ID: "m1", Content: "hello"})
	got, err := RenderMirror(m)
	if err != nil {
		t.Fatalf("RenderMirror: %v", err)
	}
	wantBase := baselineDigestOf(m.Blocks)
	header := "[[LIVE_CONTEXT version=1 revision=3 document=abcnonce baseline=" + wantBase + "]]"
	if !strings.HasPrefix(got, header) {
		t.Fatalf("header mismatch.\ngot:  %q\nwant: %q...", got, header)
	}
}

func TestRenderMirrorOmitsSystemTurns(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 0, Nonce: "nonce0"}
	m.Blocks = []Block{
		{Index: 0, Role: "system", ID: "s0", Content: "THE SYSTEM PROMPT"},
		{Index: 1, Role: "user", ID: "u1", Content: "hi"},
		{Index: 2, Role: "assistant", ID: "a1", Content: "yo"},
	}
	got, err := RenderMirror(m)
	if err != nil {
		t.Fatalf("RenderMirror: %v", err)
	}
	if strings.Contains(got, "THE SYSTEM PROMPT") {
		t.Fatalf("system turn content must not be mirrored (prompt-injection surface):\n%s", got)
	}
	if strings.Contains(got, "role=system") {
		t.Fatalf("system turn must have no CTX_TURN block:\n%s", got)
	}
}

func TestRenderMirrorFreshZeroRevision(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 0, Nonce: "freshnonce"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "user", ID: "m0", Content: "fresh"})
	got, err := RenderMirror(m)
	if err != nil {
		t.Fatalf("RenderMirror: %v", err)
	}
	if !strings.Contains(got, "revision=0") {
		t.Fatalf("fresh revision must be 0:\n%s", got)
	}
}

func TestRenderMirrorPerTurnBlocks(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 7, Nonce: "docnonce"}
	m.Blocks = []Block{
		{Index: 0, Role: "system", ID: "s0", Content: "sys"},
		{Index: 1, Role: "user", ID: "u1", Content: "u1"},
		{Index: 2, Role: "assistant", ID: "a1", Content: "a1"},
		{Index: 3, Role: "tool", ID: "t1", Content: "t1"},
	}
	got, err := RenderMirror(m)
	if err != nil {
		t.Fatalf("RenderMirror: %v", err)
	}
	for _, want := range []string{
		"[[CTX_TURN document=docnonce index=1 role=user id=u1 protected=false]]",
		"u1",
		"[[CTX_TURN document=docnonce index=2 role=assistant id=a1 protected=false]]",
		"a1",
		"[[CTX_TURN document=docnonce index=3 role=tool id=t1 protected=false]]",
		"t1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("mirror must contain %q:\n%s", want, got)
		}
	}
}

func TestRenderMirrorBaselineStable12Hex(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n"}
	m.Blocks = []Block{
		{Index: 0, Role: "user", ID: "u0", Content: "a"},
		{Index: 1, Role: "assistant", ID: "a1", Content: "b"},
	}
	got, err := RenderMirror(m)
	if err != nil {
		t.Fatalf("RenderMirror: %v", err)
	}
	wantBase := baselineDigestOf(m.Blocks)
	if !strings.Contains(got, "baseline="+wantBase) {
		t.Fatalf("baseline must be first 12 hex of sha256(non-system contents joined by \\n).\nwant %q\n got:\n%s", wantBase, got)
	}
	// Baseline is stable across renders with different revision/nonce.
	m2 := &Mirror{Version: 1, Revision: 2, Nonce: "different-nonce", Blocks: m.Blocks}
	got2, err := RenderMirror(m2)
	if err != nil {
		t.Fatalf("RenderMirror: %v", err)
	}
	if !strings.Contains(got2, "baseline="+wantBase) {
		t.Fatalf("baseline must not depend on revision or nonce")
	}
	if got == got2 {
		t.Fatalf("different revision must produce different mirror")
	}
}

func TestWriteMirror0600AndBytes(t *testing.T) {
	dir := t.TempDir()
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "user", ID: "m0", Content: "x"})
	path, err := WriteMirror(dir, m)
	if err != nil {
		t.Fatalf("WriteMirror: %v", err)
	}
	want := filepath.Join(dir, "clm-mirror.txt")
	if path != want {
		t.Fatalf("WriteMirror path: got %q, want %q", path, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat mirror: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("mirror file mode: got %o, want 600", mode)
	}
	rendered, err := RenderMirror(m)
	if err != nil {
		t.Fatalf("RenderMirror: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mirror: %v", err)
	}
	if string(got) != rendered {
		t.Fatalf("mirror bytes differ:\ngot:  %q\nwant: %q", string(got), rendered)
	}
}

func TestWriteMirrorCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "user", ID: "m0", Content: "x"})
	path, err := WriteMirror(filepath.Join(dir, "a", "b", "c"), m)
	if err != nil {
		t.Fatalf("WriteMirror must create parent dirs: %v", err)
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		t.Fatalf("mirror file missing after WriteMirror")
	}
}
