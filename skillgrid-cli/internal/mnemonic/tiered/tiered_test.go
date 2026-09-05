package tiered

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func openTierStore(t *testing.T) (*store.Store, *Store, string) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(dir, "tier")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ts := &Store{DB: st.DB, Summarizer: HeuristicSummarizer{}}
	return st, ts, dir
}

func writeL2(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write L2: %v", err)
	}
	return path
}

func TestWriteHookSidecarNonBlocking(t *testing.T) {
	st, ts, dir := openTierStore(t)
	defer st.Close()

	l2 := writeL2(t, dir, "note.md", "# Title\n\nFirst paragraph.\n\nSecond paragraph.\n")
	var wg sync.WaitGroup
	hook := &ContentWriteHook{Store: ts, WaitGroup: &wg}

	start := time.Now()
	hook.AfterContentWrite(context.Background(), "tier", l2, "Title")
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("AfterContentWrite blocked for %s", time.Since(start))
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("tier generation timed out")
	}

	abs, over := SidecarPaths(l2)
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("abstract missing: %v", err)
	}
	if _, err := os.Stat(over); err != nil {
		t.Fatalf("overview missing: %v", err)
	}
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM tiered_contents WHERE full_path = ?`, l2).Scan(&n); err != nil {
		t.Fatalf("sql: %v", err)
	}
	if n != 1 {
		t.Fatalf("tiered_contents rows=%d", n)
	}
}

func TestMigrateTierBackfill(t *testing.T) {
	st, ts, dir := openTierStore(t)
	defer st.Close()
	content := filepath.Join(dir, "content")
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	l2 := writeL2(t, content, "doc.md", "Full detail bytes must stay.\n\nMore.\n")
	before, err := os.ReadFile(l2)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	n, err := MigrateTier(context.Background(), ts, "tier", content)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if n < 1 {
		t.Fatalf("migrated count=%d", n)
	}
	after, err := os.ReadFile(l2)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("L2 bytes changed during migrate")
	}
	if _, err := os.Stat(l2 + ".abstract"); err != nil {
		t.Fatalf("abstract: %v", err)
	}
	if _, err := os.Stat(l2 + ".overview"); err != nil {
		t.Fatalf("overview: %v", err)
	}
}

func TestSummarizerFailPreserve(t *testing.T) {
	st, ts, dir := openTierStore(t)
	defer st.Close()
	ts.Summarizer = FailSummarizer{Err: errors.New("boom")}
	var logged bool
	ts.Logf = func(string, ...any) { logged = true }

	l2 := writeL2(t, dir, "keep.md", "precious L2 content")
	before, _ := os.ReadFile(l2)
	err := ts.GenerateTiers(context.Background(), "tier", l2, "keep")
	if err == nil {
		t.Fatal("expected summarizer error")
	}
	after, _ := os.ReadFile(l2)
	if string(before) != string(after) {
		t.Fatal("L2 changed after summarizer failure")
	}
	if !logged {
		t.Fatal("expected failure log")
	}
	if _, err := os.Stat(l2 + ".abstract"); err == nil {
		t.Fatal("abstract should not exist on summarizer failure")
	}
}
