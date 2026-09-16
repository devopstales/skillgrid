package mcp

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// resetFresh isolates the package-level freshness state for a test (and restores
// it after) by swapping the *freshness pointer.
func resetFresh(t *testing.T) {
	t.Helper()
	old := fresh
	fresh = &freshness{lock: codeindex.NewWriterLock()}
	t.Cleanup(func() { fresh = old })
}

// TestAutoReopen covers 03.3 (Scenario: Reader auto-reopens the new index) on
// the MCP response path: after a publication, a running process opens the
// newly-published index on its next tool call (within the injected ~5s window)
// with no restart; the agent's next query reflects its own just-made edit.
func TestAutoReopen(t *testing.T) {
	resetFresh(t)
	dataDir := t.TempDir()
	_ = os.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	defer os.Unsetenv("SKILLGRID_MNEMONIC_DATA_DIR")

	// A project store with an index file.
	projID := "reopen-proj"
	st, err := store.Open(dataDir, projID)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	live := st.Path()
	if err := os.WriteFile(live, []byte("old-index"), 0o644); err != nil {
		t.Fatalf("seed live: %v", err)
	}

	var reopened int
	svc := service.New(dataDir)
	// Wire freshness with a very short reopen window (injected ~ms, not 5s).
	InitFreshness(svc, projID, ".", codeindex.Config{}, 10*time.Millisecond)

	// Set the tracker's reopen hook to observe the reopen.
	SetReopenHook(func(_ string) { reopened++ })

	// A query before any new publish must NOT reopen.
	_, _ = fresh.preQuery(context.Background(), nil)
	if reopened != 0 {
		t.Fatalf("preQuery reopened before a new index was published")
	}

	// The agent's own edit is published: the live index is now newer.
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(live, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	time.Sleep(15 * time.Millisecond) // let the reopen window elapse

	// The next tool call reopens the new index (no restart).
	_, _ = fresh.preQuery(context.Background(), nil)
	if reopened != 1 {
		t.Fatalf("expected the next tool call to auto-reopen the new index (1 reopen), got %d", reopened)
	}
}

// TestStalenessBanner covers 03.5 (Scenario: Pending referenced file gets a
// staleness banner): during the debounce window, a response referencing a
// still-pending file prepends "⚠️ <file> is pending sync — Read it directly";
// pending files not referenced surface as a small footer; after sync the banner
// clears.
func TestStalenessBanner(t *testing.T) {
	resetFresh(t)

	// A pending source file (mid-debounce-window), injected via the pending
	// source hook so the banner path is testable without a live watcher.
	SetPendingSource(func() map[string]struct{} {
		return map[string]struct{}{"main.go": {}, "helper.ts": {}}
	})

	// 1) A response referencing a pending file → banner names it.
	res, err := JSONResult(map[string]any{"hits": []map[string]any{{"path": "main.go"}}})
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	out := applyFreshness(res, []string{"main.go"})
	text := callResultText(t, out)
	if !contains(text, "⚠️ main.go is pending sync — Read it directly") {
		t.Errorf("expected a staleness banner naming the referenced pending file; got: %s", text)
	}

	// 2) A response NOT referencing the pending files → footer, no banner.
	res2, err := JSONResult(map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	out2 := applyFreshness(res2, nil)
	text2 := callResultText(t, out2)
	if contains(text2, "⚠️ main.go is pending sync") {
		t.Errorf("un-referenced pending file must not get a banner; got: %s", text2)
	}
	if !contains(text2, "pending sync") {
		t.Errorf("expected a footer for un-referenced pending files; got: %s", text2)
	}

	// 3) After sync (pending set cleared) the banner/footer clear.
	SetPendingSource(func() map[string]struct{} { return map[string]struct{}{} })
	res3, err := JSONResult(map[string]any{"hits": []map[string]any{{"path": "main.go"}}})
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	out3 := applyFreshness(res3, []string{"main.go"})
	text3 := callResultText(t, out3)
	if contains(text3, "pending sync") {
		t.Errorf("banner must clear after sync; got: %s", text3)
	}
}

// contains reports whether s contains sub.
func contains(s, sub string) bool { return strings.Contains(s, sub) }
