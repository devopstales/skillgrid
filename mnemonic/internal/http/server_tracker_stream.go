package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/devopstales/skillgrid/mnemonic/internal/http/tracker"
)

// trackerStreamClient is one subscribed SSE client. The buffered channel lets
// a slow consumer drop events without blocking the watcher goroutine.
type trackerStreamClient struct {
	events chan sseEvent
}

// Watched dirs. Single source of truth, derived from the SAME layout the
// backlog reader uses (milestones = sibling of the tasks dir) so the watcher
// and the reader can never point at different places.
const defaultBacklogTasksDir = ".backlog/tasks"

// Derived exactly as backlogAdapter.milestonesDir() derives it, so the SSE
// watcher and the reader always agree on the milestones location.
var defaultBacklogMilestonesDir = filepath.Join(filepath.Dir(defaultBacklogTasksDir), "milestones")

// sseEvent is one SSE message: the event type (used for the `event:` line) and
// the JSON payload (the `data:` line).
type sseEvent struct {
	event   string
	payload string
}

// handleTrackerStream serves GET /tracker/stream (SSE). It watches
// .backlog/tasks/ and .backlog/milestones/ and pushes a "tasks-changed" event
// on task-file changes and a "milestones-changed" event on milestone-file
// changes, so the Kanban board live-updates without a manual reload.
//
// The watcher is per-request (simplest correct lifecycle): each client runs
// its own fsnotify watcher, removed on disconnect. For the single-operator
// dashboard this is cheap; a shared hub can replace it later without changing
// the wire format.
func (s *Server) handleTrackerStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	// The SSE stream only exists for the file-based Backlog.md provider; its
	// dirs default to the constants above. (A ?provider= override selects a
	// remote provider that has no local dirs to watch — nothing to stream.)
	tasksDir := defaultBacklogTasksDir
	milestonesDir := defaultBacklogMilestonesDir

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Initial event so the client knows the stream is alive.
	_, _ = w.Write([]byte("event: ready\n\n"))
	flusher.Flush()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "watcher: "+err.Error())
		return
	}
	defer watcher.Close()

	// Watch the tasks dir (always present — create if missing).
	if info, err := os.Stat(tasksDir); err != nil || !info.IsDir() {
		if err2 := os.MkdirAll(tasksDir, 0o755); err2 != nil {
			writeError(w, http.StatusInternalServerError, "tasks dir: "+err2.Error())
			return
		}
	}
	if err := watcher.Add(tasksDir); err != nil {
		writeError(w, http.StatusInternalServerError, "watch add: "+err.Error())
		return
	}

	// Watch the milestones dir (optional — skip if missing, no error).
	if info, err := os.Stat(milestonesDir); err == nil && info.IsDir() {
		_ = watcher.Add(milestonesDir) // non-fatal: the tasks stream still works
	}

	// Per-client buffered channel: a slow consumer drops events (no blocking).
	client := &trackerStreamClient{events: make(chan sseEvent, 32)}
	ctx := r.Context()

	// Fan-out goroutine: watcher → client channel.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !isTaskChange(evt) {
					continue
				}
				// Determine event type by which watched dir the path is under.
				eventType := "tasks-changed"
				if strings.HasPrefix(evt.Name, milestonesDir) {
					eventType = "milestones-changed"
				}
				payload, _ := json.Marshal(map[string]string{
					"type":     eventType,
					"file":     filepath.Base(evt.Name),
					"op":       evt.Op.String(),
					"provider": tracker.ProviderBacklogMD,
				})
				select {
				case client.events <- sseEvent{event: eventType, payload: string(payload)}:
				default: // slow consumer: drop, never block
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	// Write loop: client channel → SSE. Exits on context cancel.
	// Heartbeat keeps proxies from timing out idle streams.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-client.events:
			if _, err := w.Write([]byte("event: " + msg.event + "\n")); err != nil {
				return
			}
			if _, err := w.Write([]byte("data: " + msg.payload + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := w.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// isTaskChange reports whether an fsnotify event is a meaningful task change
// (write/create/remove/rename of a .md file).
func isTaskChange(evt fsnotify.Event) bool {
	if evt.Op&fsnotify.Chmod != 0 &&
		evt.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}
	return strings.HasSuffix(strings.ToLower(evt.Name), ".md")
}
