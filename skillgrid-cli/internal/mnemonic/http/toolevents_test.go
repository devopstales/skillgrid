package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedToolEvents registers two harness sessions (cursor, opencode) through the
// real tool-call route so the events/stats/list reads see production rows.
func seedToolEvents(t *testing.T, s *Server) {
	t.Helper()
	calls := []struct {
		sid   string
		agent string
		body  map[string]any
	}{
		{"cur-1", "cursor", map[string]any{"tool_name": "Read", "path": "src/auth/login.go"}},
		{"cur-1", "cursor", map[string]any{"tool_name": "Shell", "command": "npm install left-pad"}},
		{"cur-1", "cursor", map[string]any{"tool_name": "mem_search", "content_preview": "query"}},
		{"oc-1", "opencode", map[string]any{"tool_name": "write", "path": "src/auth/token.go"}},
		{"oc-1", "opencode", map[string]any{"tool_name": "bash", "command": "go test ./...", "result_status": "error"}},
	}
	for _, c := range calls {
		c.body["agent"] = c.agent
		if res := postToolCall(t, s, c.sid, c.body); res.Code != http.StatusOK && res.Code != http.StatusCreated {
			t.Fatalf("seed %s %v: status %d: %s", c.sid, c.body, res.Code, res.Body.String())
		}
	}
}

func getToolJSON(t *testing.T, s *Server, target string) map[string]any {
	t.Helper()
	rr := doGet(t, s.Handler(), target)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d: %s", target, rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("GET %s: decode: %v", target, err)
	}
	return out
}

func eventsOf(m map[string]any) []map[string]any {
	raw, _ := m["events"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(map[string]any))
	}
	return out
}

func TestSessionEvents_TimelineNewestFirst(t *testing.T) {
	s, _ := newToolCallsServer(t)
	seedToolEvents(t, s)

	evs := eventsOf(getToolJSON(t, s, "/sessions/cur-1/events?project="+proj))
	if len(evs) != 3 {
		t.Fatalf("cur-1 events = %d, want 3 (lifecycle excluded): %v", len(evs), evs)
	}
	if evs[0]["tool"] != "mem_search" || evs[0]["mcp"] != true {
		t.Errorf("newest event = %v, want mem_search flagged mcp", evs[0])
	}
	if evs[2]["tool"] != "Read" || evs[2]["mcp"] != false || evs[2]["agent"] != "cursor" {
		t.Errorf("oldest event = %v, want built-in Read by cursor", evs[2])
	}
}

func TestEvents_Filters(t *testing.T) {
	s, _ := newToolCallsServer(t)
	seedToolEvents(t, s)

	cases := []struct {
		query string
		want  int
	}{
		{"agent=opencode", 2},
		{"action=command_exec", 2},
		{"file=src/auth/**", 2},
		{"command=npm *", 1},
		{"tool=MEM_SEARCH", 1},
		{"agent=cursor&action=file_read", 1},
		{"session=oc-1", 2},
		{"since=1h", 5},
		{"since=2000-01-01T00:00:00Z&until=2000-01-02T00:00:00Z", 0},
	}
	for _, c := range cases {
		evs := eventsOf(getToolJSON(t, s, "/events?project="+proj+"&"+strings.ReplaceAll(c.query, " ", "%20")))
		if len(evs) != c.want {
			t.Errorf("%s: got %d events, want %d", c.query, len(evs), c.want)
		}
	}
	if rr := doGet(t, s.Handler(), "/events?project="+proj+"&since=yesterday"); rr.Code != http.StatusBadRequest {
		t.Errorf("bad since: status %d, want 400", rr.Code)
	}
}

func TestEventStats_Aggregates(t *testing.T) {
	s, _ := newToolCallsServer(t)
	seedToolEvents(t, s)

	st := getToolJSON(t, s, "/events/stats?project="+proj+"&since=24h")
	num := func(k string) float64 { v, _ := st[k].(float64); return v }
	if num("total") != 5 || num("sessions") != 2 || num("mcp") != 1 || num("errors") != 1 {
		t.Fatalf("totals = total %v sessions %v mcp %v errors %v, want 5/2/1/1", st["total"], st["sessions"], st["mcp"], st["errors"])
	}
	byAction, _ := st["byAction"].(map[string]any)
	if byAction["command_exec"] != float64(2) || byAction["file_write"] != float64(1) {
		t.Errorf("byAction = %v", byAction)
	}
	agents, _ := st["byAgent"].([]any)
	if len(agents) != 2 {
		t.Fatalf("byAgent = %v, want cursor + opencode", agents)
	}
	first := agents[0].(map[string]any)
	if first["agent"] != "cursor" || first["events"] != float64(3) || first["mcp"] != float64(1) || first["sessions"] != float64(1) {
		t.Errorf("cursor stat = %v", first)
	}
	files, _ := st["topFiles"].([]any)
	if len(files) != 2 {
		t.Errorf("topFiles = %v, want 2 paths", files)
	}
	cmds, _ := st["topCommands"].([]any)
	if len(cmds) != 2 {
		t.Errorf("topCommands = %v, want 2", cmds)
	}

	one := getToolJSON(t, s, "/events/stats?project="+proj+"&agent=opencode")
	if one["total"] != float64(2) || one["sessions"] != float64(1) {
		t.Errorf("agent=opencode stats = total %v sessions %v, want 2/1", one["total"], one["sessions"])
	}
	if rr := doGet(t, s.Handler(), "/events/stats?project="+proj+"&since=bogus"); rr.Code != http.StatusBadRequest {
		t.Errorf("bad since: status %d, want 400", rr.Code)
	}
}

func TestSessionEvents_IncludesObservations(t *testing.T) {
	s, _ := newToolCallsServer(t)
	seedToolEvents(t, s)

	const obsTitle = "Use JWT for auth"
	rr, _ := do(t, s.Handler(), http.MethodPost, "/observations?project="+proj, map[string]any{
		"session_id": "cur-1",
		"type":       "decision",
		"title":      obsTitle,
		"content":    "Bearer tokens only; no session cookies.",
		"scope":      "project",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /observations: status %d: %s", rr.Code, rr.Body.String())
	}

	body := getToolJSON(t, s, "/sessions/cur-1/events?project="+proj)
	rawObs, ok := body["observations"].([]any)
	if !ok {
		t.Fatalf("observations key missing or wrong type: %v", body["observations"])
	}
	if len(rawObs) != 1 {
		t.Fatalf("observations len = %d, want 1", len(rawObs))
	}
	obs := rawObs[0].(map[string]any)
	if obs["type"] != "decision" || obs["title"] != obsTitle {
		t.Errorf("observation = %v, want type decision title %q", obs, obsTitle)
	}
	if _, ok := obs["id"].(float64); !ok {
		t.Errorf("observation missing id: %v", obs)
	}
	if _, ok := obs["created_at"].(string); !ok || obs["created_at"] == "" {
		t.Errorf("observation missing created_at: %v", obs)
	}
	if _, ok := obs["tokens"].(float64); !ok {
		t.Errorf("observation missing tokens: %v", obs)
	}
	if pinned, ok := obs["pinned"].(bool); !ok {
		t.Errorf("observation missing pinned bool: %v", obs)
	} else if pinned {
		t.Errorf("observation pinned = true, want false")
	}
	if len(eventsOf(body)) != 3 {
		t.Errorf("events unchanged: got %d, want 3", len(eventsOf(body)))
	}
}

func TestMnemonicSessions_ObservationsCount(t *testing.T) {
	s, _ := newToolCallsServer(t)
	seedToolEvents(t, s)

	rr, _ := do(t, s.Handler(), http.MethodPost, "/observations?project="+proj, map[string]any{
		"session_id": "cur-1",
		"type":       "decision",
		"title":      "Count probe",
		"content":    "for sessions list alias",
		"scope":      "project",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /observations: status %d: %s", rr.Code, rr.Body.String())
	}

	raw, _ := getToolJSON(t, s, "/mnemonic/sessions?project="+proj)["sessions"].([]any)
	for _, r := range raw {
		m := r.(map[string]any)
		if m["id"] != "cur-1" {
			continue
		}
		mem, _ := m["memory_count"].(float64)
		obs, ok := m["observations"].(float64)
		if !ok {
			t.Fatalf("cur-1 missing observations count: %v", m)
		}
		if obs != 1 || mem != 1 {
			t.Errorf("cur-1 memory_count=%v observations=%v, want both 1", mem, obs)
		}
		return
	}
	t.Fatal("cur-1 not in sessions list")
}

func TestMnemonicSessions_AgentAndLastTool(t *testing.T) {
	s, _ := newToolCallsServer(t)
	seedToolEvents(t, s)

	raw, _ := getToolJSON(t, s, "/mnemonic/sessions?project="+proj)["sessions"].([]any)
	byID := map[string]map[string]any{}
	for _, r := range raw {
		m := r.(map[string]any)
		byID[m["id"].(string)] = m
	}
	cur := byID["cur-1"]
	if cur == nil || cur["agent"] != "cursor" || cur["last_tool"] != "mem_search" || cur["tool_calls"].(float64) != 3 {
		t.Errorf("cur-1 = %v, want agent cursor, last_tool mem_search, 3 tool calls", cur)
	}
	oc := byID["oc-1"]
	if oc == nil || oc["agent"] != "opencode" || oc["errors"].(float64) != 1 {
		t.Errorf("oc-1 = %v, want agent opencode with 1 error", oc)
	}
	if byID["s1"]["agent"] != "" {
		t.Errorf("seeded mem session agent = %v, want empty", byID["s1"]["agent"])
	}
}

// streamRecorder is a goroutine-safe SSE ResponseWriter whose accumulated body
// can be polled while the handler is still streaming.
type streamRecorder struct {
	mu     sync.Mutex
	header http.Header
	buf    strings.Builder
}

func newPipeWriter() (*streamRecorder, *streamRecorder) {
	r := &streamRecorder{header: http.Header{}}
	return r, r
}

func (r *streamRecorder) Header() http.Header { return r.header }
func (r *streamRecorder) WriteHeader(int)     {}
func (r *streamRecorder) Flush()              {}
func (r *streamRecorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Write(b)
}

func (r *streamRecorder) text() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

func (r *streamRecorder) waitFor(sub string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if strings.Contains(r.text(), sub) {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

// TestActivitySSE_ToolEvent: a tool call recorded after the stream opens is
// emitted as `event: tool`, and a session's first row carries newSession.
func TestActivitySSE_ToolEvent(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	// Make sure the store exists before the stream seeds its high-water marks.
	if st, err := store.Open(dataDir, proj); err == nil {
		st.Close()
	}

	pr, pw := newPipeWriter()
	req := httptest.NewRequest(http.MethodGet, "/activity/stream?project="+proj, nil)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Handler().ServeHTTP(pw, req.WithContext(ctx))
	}()
	if !pr.waitFor("event: ready", 2*time.Second) {
		t.Fatalf("no ready frame")
	}

	if res := postToolCall(t, s, "live-1", map[string]any{"agent": "kilo", "tool_name": "bash", "command": "ls"}); res.Code != http.StatusOK {
		t.Fatalf("post: %d %s", res.Code, res.Body.String())
	}
	if !pr.waitFor(`"newSession":true`, 5*time.Second) {
		t.Fatalf("no newSession tool frame; got %q", pr.text())
	}
	if !pr.waitFor(`"command":"ls"`, 5*time.Second) || !strings.Contains(pr.text(), "event: tool") {
		t.Fatalf("no tool frame for the bash call; got %q", pr.text())
	}
	if !strings.Contains(pr.text(), `"agent":"kilo"`) {
		t.Errorf("tool frame missing agent kilo: %q", pr.text())
	}
	cancel()
	<-done
}
