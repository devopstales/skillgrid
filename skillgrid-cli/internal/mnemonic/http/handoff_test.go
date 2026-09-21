package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// newHandoffServer builds a server over a store seeded with one change
// snapshot + one checkpoint (change 015-handoff-hub).
func newHandoffServer(t *testing.T) (http.Handler, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	seed := `
		INSERT INTO change_snapshots
			(project, branch, "commit", commit_short, subject, context_json, author, committed_at, created_at)
		VALUES ('http-test', 'main', 'abc1234def5678', 'abc1234', 'feat: add auth',
			'{"task":"auth","decisions":"jwt","remaining":"tests","tried":"none"}', 'dev', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z');
		INSERT INTO checkpoints
			(project, name, branch, "commit", dirty, status, created_at)
		VALUES ('http-test', 'before-apply-auth', 'main', 'abc1234def5678', 0, 'open', '2026-01-01T00:00:00Z');
	`
	if _, err := st.DB.Exec(seed); err != nil {
		t.Fatalf("seed handoff: %v", err)
	}
	_ = st.Close()
	svc := service.New(dataDir)
	return NewServer(svc).Handler(), dataDir
}

// TestHandoffSnapshotsEndpoint covers GET /handoff/snapshots — the change log
// returns the seeded snapshot with its parsed [skillgrid-context] block. The
// route was renamed from /activity/snapshots (sessions-activity-unification).
func TestHandoffSnapshotsEndpoint(t *testing.T) {
	h, _ := newHandoffServer(t)
	rr := doGet(t, h, "/handoff/snapshots?project="+proj+"&limit=10")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var em map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &em); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	snaps, _ := em["snapshots"].([]any)
	if len(snaps) != 1 {
		t.Fatalf("snapshots len = %d, want 1", len(snaps))
	}
	s0 := snaps[0].(map[string]any)
	if s0["subject"] != "feat: add auth" || s0["commitShort"] != "abc1234" {
		t.Fatalf("snapshot = %v", s0)
	}
	ctx, ok := s0["context"].(map[string]any)
	if !ok || ctx["task"] != "auth" || ctx["decisions"] != "jwt" {
		t.Fatalf("parsed context = %v", s0["context"])
	}
	// The old path is gone (hard rename, no alias): it must no longer resolve to
	// the snapshot handler. The SPA fallback (embed.go) serves the shell HTML
	// for any unmatched GET, so "gone" means it returns HTML, not the JSON
	// snapshot payload — assert the body no longer parses as JSON.
	old := doGet(t, h, "/activity/snapshots?project="+proj+"&limit=10")
	if old.Body.Len() > 0 {
		var probe map[string]any
		if err := json.Unmarshal(old.Body.Bytes(), &probe); err == nil {
			t.Fatalf("old /activity/snapshots still returns JSON after rename: %v", probe)
		}
	}
}

// TestHandoffStatusEndpoint covers GET /handoff/status — the one-call hub view
// returns the latest snapshot + the open checkpoint.
func TestHandoffStatusEndpoint(t *testing.T) {
	h, _ := newHandoffServer(t)
	rr := doGet(t, h, "/handoff/status?project="+proj)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var em map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &em); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	latest, ok := em["latest_snapshot"].(map[string]any)
	if !ok || latest["subject"] != "feat: add auth" {
		t.Fatalf("latest_snapshot = %v", em["latest_snapshot"])
	}
	cps, _ := em["checkpoints"].([]any)
	if len(cps) != 1 {
		t.Fatalf("checkpoints len = %d, want 1", len(cps))
	}
	c0 := cps[0].(map[string]any)
	if c0["name"] != "before-apply-auth" || c0["status"] != "open" {
		t.Fatalf("checkpoint = %v", c0)
	}
}

// TestHandoffSSEEmitsSnapshot covers the extended /activity/stream: after the
// initial ready event, inserting a new change snapshot is emitted as an
// `event: snapshot` SSE frame.
func TestHandoffSSEEmitsSnapshot(t *testing.T) {
	h, dataDir := newHandoffServer(t)

	fw := &flushingWriter{
		ResponseRecorder: httptest.NewRecorder(),
		gotReady:         make(chan struct{}),
		gotActivity:      make(chan struct{}),
		gotSnapshot:      make(chan struct{}),
	}
	req := httptest.NewRequest(http.MethodGet, "/activity/stream?project="+proj, nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(fw, req)
	}()
	defer func() {
		cancel()
		<-done
	}()

	select {
	case <-fw.gotReady:
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive SSE 'ready' within 2s")
	}

	// Insert a new snapshot; the poller should emit it as a snapshot event.
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	_, _ = st.DB.Exec(`
		INSERT INTO change_snapshots
			(project, branch, "commit", commit_short, subject, author, committed_at, created_at)
		VALUES ('http-test', 'main', 'fff00011122233', 'fff0001', 'feat: live snap', 'dev', '2026-01-03T00:00:00Z', '2026-01-03T00:00:00Z')`)
	_ = st.Close()

	select {
	case <-fw.gotSnapshot:
	case <-time.After(5 * time.Second):
		t.Fatalf("did not receive a live snapshot SSE event within 5s")
	}
}
