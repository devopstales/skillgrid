package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mnemonichttp "github.com/devopstales/skillgrid/mnemonic/internal/http"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// seedLogs records two harness sessions through the real hook writer.
func seedLogs(t *testing.T, st *store.Store, proj string) {
	t.Helper()
	ctx := context.Background()
	svc := memory.New(st, proj)
	svc.SetHooks(memory.HooksConfig{Enabled: true})
	for sid, agent := range map[string]string{"cur-1": "cursor", "oc-1": "opencode"} {
		if _, err := svc.EnsureSession(ctx, sid, proj, "", "", agent); err != nil {
			t.Fatalf("ensure %s: %v", sid, err)
		}
	}
	calls := []memory.HookPayload{
		{SessionID: "cur-1", ToolName: "Read", File: "src/auth/login.go", ResultStatus: "success"},
		{SessionID: "cur-1", ToolName: "mem_search", ResultStatus: "success"},
		{SessionID: "oc-1", ToolName: "bash", Command: "go test ./...", ResultStatus: "error"},
	}
	for _, p := range calls {
		if _, err := svc.RunHook(ctx, memory.HookPostToolUse, p); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func TestLogsWindow_TextAndJSON(t *testing.T) {
	_, proj, st := sessionCLIFixture(t)
	seedLogs(t, st, proj)
	ctx := context.Background()

	var buf bytes.Buffer
	last, err := printLogsWindow(ctx, &buf, st.DB, proj, mnemonichttp.ToolEventFilter{Limit: 100}, false)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines (lifecycle excluded), got %d:\n%s", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "cursor") || !strings.Contains(lines[0], "src/auth/login.go") {
		t.Errorf("oldest first: line 0 = %q", lines[0])
	}
	if !strings.Contains(lines[1], "mem_search [mcp]") {
		t.Errorf("mcp marker missing: %q", lines[1])
	}
	if !strings.Contains(lines[2], "opencode") || !strings.Contains(lines[2], "[error]") {
		t.Errorf("error marker missing: %q", lines[2])
	}
	if last == 0 {
		t.Error("follow cursor = 0, want the newest id")
	}

	buf.Reset()
	if _, err := printLogsWindow(ctx, &buf, st.DB, proj, mnemonichttp.ToolEventFilter{Agent: "opencode", Limit: 100}, true); err != nil {
		t.Fatal(err)
	}
	var e mnemonichttp.ToolEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &e); err != nil {
		t.Fatalf("json line: %v\n%s", err, buf.String())
	}
	if e.Agent != "opencode" || e.Command != "go test ./..." {
		t.Errorf("json event = %+v", e)
	}
}

func TestSessionsList(t *testing.T) {
	_, proj, st := sessionCLIFixture(t)
	seedLogs(t, st, proj)
	rows, err := listSessionRows(context.Background(), st.DB, proj, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	byID := map[string]sessionRow{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	if r := byID["cur-1"]; r.Agent != "cursor" || r.ToolCalls != 2 || r.LastTool != "mem_search" {
		t.Errorf("cur-1 = %+v", r)
	}
	only, _ := listSessionRows(context.Background(), st.DB, proj, "opencode", 50)
	if len(only) != 1 || only[0].ID != "oc-1" {
		t.Errorf("agent filter = %+v", only)
	}
	var buf bytes.Buffer
	writeSessionRows(&buf, rows)
	if !strings.Contains(buf.String(), "cur-1") || !strings.Contains(buf.String(), "n/a") {
		t.Errorf("table:\n%s", buf.String())
	}
}

func TestStatsCLI(t *testing.T) {
	_, proj, st := sessionCLIFixture(t)
	seedLogs(t, st, proj)
	cost := 1.25
	if _, err := memory.New(st, proj).RecordSessionUsage(context.Background(), "cur-1", memory.UsageReport{
		Model: "claude-sonnet-4-5", Input: 1000, Output: 100, Cost: &cost,
	}); err != nil {
		t.Fatal(err)
	}
	s, err := mnemonichttp.ComputeEventStats(context.Background(), st.DB, proj, "24h", "")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	writeStats(&buf, s)
	out := buf.String()
	for _, want := range []string{"3 tool calls", "2 sessions", "1 MCP", "cursor", "opencode", "src/auth/login.go", "mem_search [mcp]", "claude-sonnet-4-5", "$1.25"} {
		if !strings.Contains(out, want) {
			t.Errorf("stats output missing %q:\n%s", want, out)
		}
	}
}

func TestFollowSSE_FiltersToolFrames(t *testing.T) {
	frame := func(e mnemonichttp.ToolEvent) string {
		raw, _ := json.Marshal(e)
		return "event: tool\ndata: " + string(raw) + "\n\n"
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project") != "p1" {
			http.Error(w, "project", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: ready\ndata: {}\n\n")
		fmt.Fprint(w, frame(mnemonichttp.ToolEvent{ID: 1, Action: "session_start", SessionID: "s", Agent: "cursor"}))
		fmt.Fprint(w, frame(mnemonichttp.ToolEvent{ID: 2, Action: "file_read", Tool: "Read", Path: "src/a.go", SessionID: "s", Agent: "cursor", TS: time.Now().UTC().Format(time.RFC3339)}))
		fmt.Fprint(w, frame(mnemonichttp.ToolEvent{ID: 3, Action: "command_exec", Tool: "bash", Command: "ls", SessionID: "o", Agent: "opencode"}))
		fmt.Fprint(w, "event: activity\ndata: {\"id\":9}\n\n")
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := followSSE(context.Background(), &buf, srv.URL, "p1", mnemonichttp.ToolEventFilter{Agent: "cursor", File: "src/**"}, false)
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Errorf("err = %v, want stream closed", err)
	}
	out := strings.TrimSpace(buf.String())
	if strings.Count(out, "\n") != 0 || !strings.Contains(out, "src/a.go") {
		t.Errorf("want only the cursor src/ read, got:\n%s", out)
	}

	if err := followSSE(context.Background(), &buf, "http://127.0.0.1:1", "p1", mnemonichttp.ToolEventFilter{}, false); err == nil ||
		!strings.Contains(err.Error(), "not reachable") {
		t.Errorf("unreachable server err = %v", err)
	}
}

func TestFollowStore_TailsNewRows(t *testing.T) {
	_, proj, st := sessionCLIFixture(t)
	seedLogs(t, st, proj)
	var cursor int64
	if err := st.DB.QueryRow(`SELECT MAX(id) FROM session_events`).Scan(&cursor); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	var buf bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- followStore(ctx, &buf, st.DB, proj, mnemonichttp.ToolEventFilter{Action: "command_exec", AfterID: cursor}, false, 50*time.Millisecond)
	}()
	time.Sleep(150 * time.Millisecond)
	svc := memory.New(st, proj)
	svc.SetHooks(memory.HooksConfig{Enabled: true})
	if _, err := svc.RunHook(context.Background(), memory.HookPostToolUse, memory.HookPayload{
		SessionID: "oc-1", ToolName: "bash", Command: "make lint", ResultStatus: "success",
	}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	if strings.Count(out, "\n") != 0 || !strings.Contains(out, "make lint") {
		t.Errorf("follow must print only the call inserted after it started:\n%s", out)
	}
}

func TestGlobRegexp(t *testing.T) {
	cases := []struct {
		glob, v string
		want    bool
	}{
		{"src/**", "src/a/b.go", true},
		{"*.go", "x/y.go", true},
		{"npm *", "npm install", true},
		{"npm *", "pnpm install", false},
		{"a?c", "abc", true},
		{"a.c", "abc", false},
	}
	for _, c := range cases {
		if got := globRegexp(c.glob).MatchString(c.v); got != c.want {
			t.Errorf("%q ~ %q = %v, want %v", c.glob, c.v, got, c.want)
		}
	}
}
