package kb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

func seedFact(t *testing.T, st store.Store, project, title, content string) {
	t.Helper()
	if _, err := st.AddObservation(context.Background(), store.AddObservationParams{
		SessionID: "s", Type: "decision", Title: title, Content: content,
		Project: project, Scope: "project", TopicKey: "tk-" + title,
	}); err != nil {
		t.Fatalf("seed fact: %v", err)
	}
}

func TestSourceFingerprintStableAndFactSensitive(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	seedFact(t, st, "proj-a", "a", "first fact")

	fp1, err := SourceFingerprint(ctx, st, "proj-a", "")
	if err != nil {
		t.Fatalf("fp1: %v", err)
	}
	if fp1 == "" || fp1[:7] != "sha256:" {
		t.Fatalf("fp1 format: %q", fp1)
	}
	// Stable on recompute.
	fp1b, err := SourceFingerprint(ctx, st, "proj-a", "")
	if err != nil {
		t.Fatalf("fp1b: %v", err)
	}
	if fp1b != fp1 {
		t.Fatalf("fingerprint not stable: %q vs %q", fp1, fp1b)
	}
	// Sensitive to a new fact.
	seedFact(t, st, "proj-a", "b", "second fact")
	fp2, err := SourceFingerprint(ctx, st, "proj-a", "")
	if err != nil {
		t.Fatalf("fp2: %v", err)
	}
	if fp2 == fp1 {
		t.Fatal("fingerprint must change when a fact is added")
	}
}

func TestSourceFingerprintIgnoresKBOwnWrites(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()

	fp1, err := SourceFingerprint(ctx, st, "proj-a", root)
	if err != nil {
		t.Fatalf("fp1: %v", err)
	}
	// The run's own writes: index/ projections + vault page edits.
	os.MkdirAll(filepath.Join(root, "index"), 0o755)
	if err := os.WriteFile(filepath.Join(root, "index", "last-update.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write projection: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "decisions"), 0o755); err != nil {
		t.Fatalf("mkdir decisions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "decisions", "x.md"), []byte("page"), 0o644); err != nil {
		t.Fatalf("write page: %v", err)
	}
	fp2, err := SourceFingerprint(ctx, st, "proj-a", root)
	if err != nil {
		t.Fatalf("fp2: %v", err)
	}
	if fp2 != fp1 {
		t.Fatalf("fingerprint must not change under the run's own writes: %q vs %q", fp1, fp2)
	}
}

func TestSourceFingerprintSensitiveToGitHead(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()

	realHead := gitHead
	defer func() { gitHead = realHead }()

	gitHead = func(string) string { return "aaaa" }
	fp1, err := SourceFingerprint(ctx, st, "proj-a", root)
	if err != nil {
		t.Fatalf("fp1: %v", err)
	}
	gitHead = func(string) string { return "bbbb" }
	fp2, err := SourceFingerprint(ctx, st, "proj-a", root)
	if err != nil {
		t.Fatalf("fp2: %v", err)
	}
	if fp2 == fp1 {
		t.Fatal("fingerprint must change when the git HEAD changes")
	}
}
