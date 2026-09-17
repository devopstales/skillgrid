package http

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// 2.7 [AFK] /tracker/providers reports the active provider + CLI availability.
func TestPhase2_Providers(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	// File-based provider: connected true (no CLI needed).
	req := httptest.NewRequest(http.MethodGet, "/tracker/providers", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("providers: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if body := rr.Body.String(); !strings.Contains(body, `"provider":"backlogmd"`) || !strings.Contains(body, `"connected":true`) {
		t.Errorf("providers must report backlogmd connected, got %s", body)
	}
	// CLI-backed provider missing its binary → connected:false.
	t.Setenv("PATH", t.TempDir()) // no gh on PATH
	t.Setenv("SKILLGRID_TRACKER", "github")
	req = httptest.NewRequest(http.MethodGet, "/tracker/providers", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("providers github: expected 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); !strings.Contains(body, `"provider":"github"`) || !strings.Contains(body, `"connected":false`) {
		t.Errorf("github missing CLI must report connected:false, got %s", body)
	}
	// Unknown provider → 501.
	req = httptest.NewRequest(http.MethodGet, "/tracker/providers?provider=nope", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("unknown provider must be 501, got %d", rr.Code)
	}
}

// 2.7 [AFK] /tracker/* CRUD mounted; PATCH write-gated; deps + stream mounted.
// "Mounted" = NOT the mux's 404 for an unregistered pattern. We distinguish an
// unmounted route (404 with no body) from a mounted-but-missing-resource 404
// (404 with a JSON error body).
func isMounted(rr *httptest.ResponseRecorder) bool {
	if rr.Code != http.StatusNotFound {
		return true
	}
	// A mounted handler that 404s a missing resource emits JSON.
	return strings.Contains(rr.Body.String(), `"error"`)
}

func TestPhase2_Routes(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("PATH", t.TempDir()) // no CLIs: reads degrade
	t.Chdir(t.TempDir())
	writeBacklogTasks(t, map[string]string{"task-001.md": phase2TaskA})
	for _, target := range []string{
		"/tracker/providers", "/tracker/config", "/tracker/tasks",
		"/tracker/tasks/TASK-001", "/tracker/tasks/TASK-001/deps",
		"/backlog/config", "/backlog/tasks", "/backlog/tasks/TASK-001",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if !isMounted(rr) {
			t.Errorf("%s must be mounted (got unmounted 404)", target)
		}
	}
	// PATCH /tracker/tasks/{id} is mounted (reaches the handler, not the mux 404).
	req := httptest.NewRequest(http.MethodPatch, "/tracker/tasks/TASK-001", strings.NewReader(`{"status":"done"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !isMounted(rr) {
		t.Error("PATCH /tracker/tasks/{id} must be mounted")
	}
}

// 2.4 [RED] PATCH /tracker/tasks/{id} write-gating: 401 without token, 200 with.
func TestPhase2_Patch_Auth(t *testing.T) {
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret") // before newHandler (token read at construction)
	h := newHandler(t)
	// Backlog.md tasks on disk so the file-based adapter has data.
	t.Chdir(t.TempDir())
	writeBacklogTasks(t, map[string]string{"task-001.md": phase2TaskA})
	// Without token → 401.
	req := httptest.NewRequest(http.MethodPatch, "/tracker/tasks/TASK-001", strings.NewReader(`{"status":"ready-for-agent"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("PATCH without token must be 401, got %d", rr.Code)
	}
	// With token → 200 (the `backlog` CLI must exist for the status write).
	mkbinPhase2(t, "backlog", `case "$1 $2" in
  "task edit") exit 0 ;;
  *) exit 1 ;;
esac`)
	req = httptest.NewRequest(http.MethodPatch, "/tracker/tasks/TASK-001", strings.NewReader(`{"status":"ready-for-agent"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("PATCH with token must be 200, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// 2.4 [RED] PATCH invalid status → 400 listing valid statuses.
func TestPhase2_Patch_InvalidStatus(t *testing.T) {
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret") // before newHandler
	h := newHandler(t)
	t.Chdir(t.TempDir())
	writeBacklogTasks(t, map[string]string{"task-001.md": phase2TaskA})
	req := httptest.NewRequest(http.MethodPatch, "/tracker/tasks/TASK-001", strings.NewReader(`{"status":"not-a-real-status"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("PATCH invalid status must be 400, got %d", rr.Code)
	}
	if body := rr.Body.String(); !strings.Contains(body, "valid") {
		t.Errorf("400 must list valid statuses, got %s", body)
	}
}

// 2.4 [RED] GET /tracker/tasks parses Backlog.md frontmatter (file-based).
func TestPhase2_Backlist_HTTP(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("PATH", t.TempDir()) // no CLI: file-based must still work
	t.Chdir(t.TempDir())
	writeBacklogTasks(t, map[string]string{"task-001.md": phase2TaskA, "task-002.md": phase2TaskB})
	req := httptest.NewRequest(http.MethodGet, "/tracker/tasks?provider=backlogmd", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{`"provider":"backlogmd"`, `"TASK-001"`, `"TASK-002"`, `"board"`} {
		if !strings.Contains(body, want) {
			t.Errorf("list must contain %s, got %s", want, body)
		}
	}
	// Unknown id → 404.
	req = httptest.NewRequest(http.MethodGet, "/tracker/tasks/TASK-999", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("unknown id must be 404, got %d", rr.Code)
	}
}

// 2.6 [RED] SSE: a task-file change is emitted live.
func TestPhase2_TrackerStream(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	workdir := t.TempDir()
	t.Chdir(workdir)
	taskDir := filepath.Join(workdir, ".backlog/tasks")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "task-001.md"), []byte(phase2TaskA), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(h)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/tracker/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("content-type must be text/event-stream, got %q", ct)
	}

	scanner := bufio.NewScanner(resp.Body)
	if !waitForEvent(t, scanner, "ready", 3*time.Second) {
		t.Fatalf("did not receive ready event")
	}

	// Trigger a task-file change.
	time.Sleep(150 * time.Millisecond) // let the watcher attach
	if err := os.WriteFile(filepath.Join(taskDir, "task-001.md"), []byte(phase2TaskA+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !waitForEvent(t, scanner, "tasks-changed", 5*time.Second) {
		t.Fatalf("did not receive tasks-changed event after file change")
	}
}

// 2.6 [RED] SSE leak-free: a client that disconnects is cleaned up
// (goroutine count returns to baseline after the requests are cancelled).
func TestPhase2_TrackerStream_NoLeak(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	workdir := t.TempDir()
	t.Chdir(workdir)
	if err := os.MkdirAll(filepath.Join(workdir, ".backlog/tasks"), 0o755); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(h)
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	var clients []context.CancelFunc
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/tracker/stream", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}
		clients = append(clients, cancel)
		resp.Body.Close()
	}
	// Let the stream handlers spin up (each spawns a fan-out goroutine).
	time.Sleep(300 * time.Millisecond)
	mid := runtime.NumGoroutine()

	// Disconnect all clients (cancel context → handler exits → watcher closed).
	for _, cancel := range clients {
		cancel()
	}
	time.Sleep(500 * time.Millisecond)
	after := runtime.NumGoroutine()

	// The streams should have spun up goroutines and cleaned them up on cancel.
	if after > baseline+2 {
		t.Errorf("goroutine leak: baseline=%d mid=%d after=%d", baseline, mid, after)
	}
}

// ── helpers ──

const phase2TaskA = `---
id: TASK-001
title: 'First task'
status: needs-triage
priority: high
type: feature
assignee:
  - "@ana"
labels:
  - ui
dependencies: []
---
First body.
`

const phase2TaskB = `---
id: TASK-002
title: Second task
status: done
priority: low
type: bug
assignee: []
labels: []
dependencies:
  - TASK-001
---
Second body.
`

func writeBacklogTasks(t *testing.T, files map[string]string) {
	t.Helper()
	dir := ".backlog/tasks"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func mkbinPhase2(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/" + name
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	t.Setenv("PATH", dir)
}

func waitForEvent(t *testing.T, scanner *bufio.Scanner, event string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !scanner.Scan() {
			return false
		}
		line := scanner.Text()
		if strings.Contains(line, "event: "+event) {
			return true
		}
	}
	return false
}


