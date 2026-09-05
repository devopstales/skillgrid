package service_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/tiered"
)

func TestMnemonicCommitLongTerm(t *testing.T) {
	dataDir := t.TempDir()
	project := "commitproj"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()

	svc := service.New(dataDir)
	var wg sync.WaitGroup
	out, err := svc.MnemonicCommit(context.Background(), project, service.MnemonicCommitInput{
		Title:          "Lesson",
		LessonsLearned: "Always write tests first.",
		SourceLink:     "https://example.test/src",
	}, &wg)
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if out.MemoryID == 0 || out.FullPath == "" {
		t.Fatalf("bad result: %+v", out)
	}
	body, err := os.ReadFile(out.FullPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "Always write tests first." {
		t.Fatalf("L2=%q", body)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("tiers timed out")
	}
	if _, err := os.Stat(out.FullPath + ".abstract"); err != nil {
		t.Fatalf("abstract: %v", err)
	}
	st2, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	var link string
	if err := st2.DB.QueryRow(`SELECT source_link FROM long_term_memories WHERE id=?`, out.MemoryID).Scan(&link); err != nil {
		t.Fatal(err)
	}
	if link != "https://example.test/src" {
		t.Fatalf("source_link=%q", link)
	}
}

func TestNoAutoCommitSessionEnd(t *testing.T) {
	dataDir := t.TempDir()
	project := "noauto"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', ?, '/tmp', ?, 'active')`, project, now); err != nil {
		t.Fatal(err)
	}
	st.Close()

	svc := service.New(dataDir)
	before, err := svc.CountLongTermMemories(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SessionEnd(context.Background(), project, "s1", "unsaved lessons"); err != nil {
		t.Fatalf("session end: %v", err)
	}
	after, err := svc.CountLongTermMemories(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("session end wrote LTM: before=%d after=%d", before, after)
	}
}

func TestMissingSourcesPartial(t *testing.T) {
	dataDir := t.TempDir()
	project := "miss"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	svc := service.New(dataDir)
	_, err = svc.MnemonicCommit(context.Background(), project, service.MnemonicCommitInput{}, nil)
	if err == nil || !errors.Is(err, service.ErrMissingCommitSources) {
		t.Fatalf("want missing sources, got %v", err)
	}
	n, err := svc.CountLongTermMemories(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("partial row written n=%d", n)
	}
	entries, _ := os.ReadDir(filepath.Join(dataDir, project, "ltm"))
	if len(entries) != 0 {
		t.Fatalf("partial files: %v", entries)
	}
}

func TestCommitAsyncNoAwaitTier(t *testing.T) {
	dataDir := t.TempDir()
	project := "async"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()

	svc := service.New(dataDir)
	start := time.Now()
	out, err := svc.MnemonicCommit(context.Background(), project, service.MnemonicCommitInput{
		Title:          "Async",
		LessonsLearned: "body",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("commit blocked waiting for tiers: %s", time.Since(start))
	}
	if _, err := os.Stat(out.FullPath); err != nil {
		t.Fatal("L2 missing")
	}
	time.Sleep(300 * time.Millisecond)
	body, _ := os.ReadFile(out.FullPath)
	if string(body) != "body" {
		t.Fatal("L2 changed")
	}
	ts := &tiered.Store{
		DB:         mustOpenDB(t, dataDir, project).DB,
		Summarizer: tiered.FailSummarizer{Err: errors.New("slow fail")},
	}
	_ = ts.GenerateTiers(context.Background(), project, out.FullPath, "Async")
	body2, _ := os.ReadFile(out.FullPath)
	if string(body2) != "body" {
		t.Fatal("L2 undone after tier failure")
	}
}

func mustOpenDB(t *testing.T, dataDir, project string) *store.Store {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}
