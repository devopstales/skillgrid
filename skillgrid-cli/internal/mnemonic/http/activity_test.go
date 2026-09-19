package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedActivityStore opens the store for proj and seeds a session with several
// observations of different types/sources/tool_names, so the activity
// events/stats/stream endpoints have data to return.
func seedActivityStore(t *testing.T, dataDir string) {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("seed activity store: %v", err)
	}
	defer st.Close()
	seed := `
		INSERT INTO sessions (id, project, directory, started_at, ended_at, summary, status, title)
		VALUES ('act-sess-1', 'http-test', '.', '2026-09-10T09:00:00Z', NULL, '## Goal\nact', 'active', 'Acting session');

		INSERT INTO observations (session_id, type, title, content, project, scope, topic_key, source, tool_name, owner, visibility, status, revision_count, created_at, updated_at, normalized_hash)
		VALUES ('act-sess-1', 'decision', 'Chose SQLite', 'decided sqlite', 'http-test', 'project', 'arch', 'agent', 'mem_save', 'act-sess-1', 'team', 'active', 0, '2026-09-10T09:05:00Z', '2026-09-10T09:05:00Z', 'act-h1');

		INSERT INTO observations (session_id, type, title, content, project, scope, topic_key, source, tool_name, owner, visibility, status, revision_count, created_at, updated_at, normalized_hash)
		VALUES ('act-sess-1', 'bugfix', 'Fixed login', 'bug body', 'http-test', 'project', 'bugs/login', 'agent', 'mem_save', 'act-sess-1', 'team', 'active', 0, '2026-09-10T09:15:00Z', '2026-09-10T09:15:00Z', 'act-h2');

		INSERT INTO observations (session_id, type, title, content, project, scope, topic_key, source, tool_name, owner, visibility, status, revision_count, created_at, updated_at, normalized_hash)
		VALUES ('act-sess-1', 'discovery', 'Found N+1', 'n+1 in list', 'http-test', 'project', 'perf', 'passive', NULL, 'act-sess-1', 'team', 'active', 0, '2026-09-10T09:25:00Z', '2026-09-10T09:25:00Z', 'act-h3');
	`
	if _, err := st.DB.Exec(seed); err != nil {
		t.Fatalf("seed activity: %v", err)
	}
}

// newPhase6ActivityServer builds a server over a seeded activity store.
func newPhase6ActivityServer(t *testing.T) (http.Handler, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedActivityStore(t, dataDir)
	svc := service.New(dataDir)
	return NewServer(svc).Handler(), dataDir
}

func TestPhase6_Activity(t *testing.T) {
	h, _ := newPhase6ActivityServer(t)

	// --- /activity/events?limit=100 (newest first) ---
	rr := doGet(t, h, "/activity/events?project="+proj+"&limit=100")
	if rr.Code != http.StatusOK {
		t.Fatalf("events: status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var em map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &em); err != nil {
		t.Fatalf("events: unmarshal: %v", err)
	}
	evs, _ := em["events"].([]any)
	if len(evs) != 3 {
		t.Fatalf("events: got %d, want 3", len(evs))
	}
	first := evs[0].(map[string]any)
	if first["summary"] != "Found N+1" {
		t.Errorf("events[0].summary = %v, want 'Found N+1' (newest first)", first["summary"])
	}
	if first["type"] != "discovery" {
		t.Errorf("events[0].type = %v, want 'discovery'", first["type"])
	}
	if first["source"] != "passive" {
		t.Errorf("events[0].source = %v, want 'passive'", first["source"])
	}
	second := evs[1].(map[string]any)
	if second["summary"] != "Fixed login" {
		t.Errorf("events[1].summary = %v, want 'Fixed login'", second["summary"])
	}
	if second["actor"] != "mem_save" {
		t.Errorf("events[1].actor = %v, want 'mem_save' (tool_name)", second["actor"])
	}

	// --- limit is honored ---
	rr = doGet(t, h, "/activity/events?project="+proj+"&limit=2")
	var lm map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &lm)
	if got := len(lm["events"].([]any)); got != 2 {
		t.Errorf("events limit=2: got %d, want 2", got)
	}

	// --- /activity/stats ---
	rr = doGet(t, h, "/activity/stats?project="+proj)
	if rr.Code != http.StatusOK {
		t.Fatalf("stats: status = %d, want 200", rr.Code)
	}
	var sm map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &sm); err != nil {
		t.Fatalf("stats: unmarshal: %v", err)
	}
	if total, _ := sm["total"].(float64); int(total) != 3 {
		t.Errorf("stats.total = %v, want 3", sm["total"])
	}
	if byType, _ := sm["byType"].(map[string]any); byType != nil {
		if got, _ := byType["bugfix"].(float64); int(got) != 1 {
			t.Errorf("stats.byType.bugfix = %v, want 1", got)
		}
	} else {
		t.Errorf("stats.byType missing")
	}

	// --- unknown project → 200 with an empty feed (store.Open auto-creates,
	// matching the Phase 5 list convention; 404 is for specific-id lookups) ---
	rr = doGet(t, h, "/activity/events?project=nope")
	if rr.Code != http.StatusOK {
		t.Errorf("unknown project: status = %d, want 200 (empty feed)", rr.Code)
	}
	var ue map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &ue)
	if evs, _ := ue["events"].([]any); len(evs) != 0 {
		t.Errorf("unknown project: events len = %d, want 0", len(evs))
	}
}

// TestPhase6_ActivitySSE verifies the stream emits the initial ready event,
// then a live activity event when a new observation is inserted, and is
// leak-free (goroutine count returns to baseline after disconnect).
func TestPhase6_ActivitySSE(t *testing.T) {
	h, dataDir := newPhase6ActivityServer(t)
	before := runtime.NumGoroutine()

	fw := newFlushingWriter()
	req := httptest.NewRequest(http.MethodGet, "/activity/stream?project="+proj, nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(fw, req)
	}()

	// Initial ready event.
	select {
	case <-fw.gotReady:
	case <-time.After(2 * time.Second):
		cancel()
		<-done
		t.Fatalf("did not receive SSE 'ready' event within 2s")
	}

	// Insert a new observation; the poller should emit it as an activity event.
	st, err := store.Open(dataDir, proj)
	if err != nil {
		cancel()
		<-done
		t.Fatalf("reopen store: %v", err)
	}
	_, _ = st.DB.Exec(`
		INSERT INTO observations (session_id, type, title, content, project, scope, topic_key, source, owner, visibility, status, revision_count, created_at, updated_at, normalized_hash)
		VALUES ('act-sess-1', 'pattern', 'New pattern', 'live', 'http-test', 'project', 'pat', 'agent', 'act-sess-1', 'team', 'active', 0, '2026-09-10T10:00:00Z', '2026-09-10T10:00:00Z', 'act-h4')`)
	_ = st.Close()

	// Wait for the live activity event.
	select {
	case <-fw.gotActivity:
	case <-time.After(5 * time.Second):
		cancel()
		<-done
		t.Fatalf("did not receive a live activity SSE event within 5s")
	}

	// Disconnect and confirm the per-client poller goroutine exits on
	// ctx.Done(). The realistic single-disconnect leak is exactly ONE extra
	// goroutine, so `after > before` is too weak (review A6). Instead, poll
	// until the count returns to baseline within a deadline — the poller
	// goroutine must terminate, not merely be bounded.
	cancel()
	<-done
	deadline := time.Now().Add(2 * time.Second)
	for {
		after := runtime.NumGoroutine()
		if after <= before {
			break
		}
		if time.Now().After(deadline) {
			t.Errorf("goroutine leak: before=%d after=%d (poller did not exit on ctx.Done)", before, after)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// doGet performs a GET and returns the recorder (test helper). It sends
// Accept: application/json — the same header the UI api.ts fetch helpers send —
// so the shellOrJSON-wrapped list routes (plans/activity/git) return JSON.
func doGet(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// flushingWriter is an http.ResponseWriter + http.Flusher that signals when the
// 'ready' and a live activity event have been written.
type flushingWriter struct {
	*httptest.ResponseRecorder
	gotReady    chan struct{}
	gotActivity chan struct{}
}

func newFlushingWriter() *flushingWriter {
	return &flushingWriter{
		ResponseRecorder: httptest.NewRecorder(),
		gotReady:         make(chan struct{}),
		gotActivity:      make(chan struct{}),
	}
}

func (f *flushingWriter) Flush() {}

func (f *flushingWriter) Write(b []byte) (int, error) {
	n, err := f.ResponseRecorder.Write(b)
	if strings.Contains(string(b), "event: ready") {
		closeOnce(&f.gotReady)
	}
	if strings.Contains(string(b), "event: activity") {
		closeOnce(&f.gotActivity)
	}
	return n, err
}

// closeOnce closes ch the first time; later calls are no-ops (channels can only
// be closed once, and Write may be called repeatedly).
func closeOnce(ch *chan struct{}) {
	select {
	case <-*ch:
	default:
		close(*ch)
	}
}

var _ = io.Discard
