package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/tracker"
)

// trackerStreamClient is one subscribed SSE client. The buffered channel lets
// a slow consumer drop events without blocking the watcher goroutine.
type trackerStreamClient struct {
	events chan string
}

// newTrackerStreamHub wires a fsnotify watcher on .backlog/tasks/ to a set of
// SSE clients. A single watcher goroutine fans out to all clients; each client
// is removed on context cancel (no goroutine leak).
func newTrackerStreamHub(dir string) *trackerStreamHub {
	return &trackerStreamHub{
		dir:      dir,
		clients:  map[*trackerStreamClient]bool{},
		broadcast: make(chan string, 16),
	}
}

type trackerStreamHub struct {
	dir       string
	clients   map[*trackerStreamClient]bool
	broadcast chan string
}

func (h *trackerStreamHub) add(c *trackerStreamClient) {
	h.clients[c] = true
}

func (h *trackerStreamHub) remove(c *trackerStreamClient) {
	delete(h.clients, c)
}

// watchDir returns the directory to watch, creating it if missing so the
// watcher never errors on a fresh repo.
func (h *trackerStreamHub) watchDir() string {
	if h.dir == "" {
		h.dir = ".backlog/tasks"
	}
	_ = os.MkdirAll(h.dir, 0o755)
	return h.dir
}

// handleTrackerStream serves GET /tracker/stream (SSE). It watches
// .backlog/tasks/ and pushes a "tasks-changed" event whenever a task file
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

	// Resolve the tasks dir: prefer the active Backlog.md provider's dir.
	dir := ".backlog/tasks"
	if p, err := resolveTracker(""); err == nil && p.Name() == tracker.ProviderBacklogMD {
		dir = ".backlog/tasks"
	}

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

	watchTarget := dir
	if info, err := os.Stat(watchTarget); err != nil || !info.IsDir() {
		if err2 := os.MkdirAll(watchTarget, 0o755); err2 != nil {
			writeError(w, http.StatusInternalServerError, "tasks dir: "+err2.Error())
			return
		}
	}
	if err := watcher.Add(watchTarget); err != nil {
		writeError(w, http.StatusInternalServerError, "watch add: "+err.Error())
		return
	}

	// Per-client buffered channel: a slow consumer drops events (no blocking).
	client := &trackerStreamClient{events: make(chan string, 32)}
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
				payload, _ := json.Marshal(map[string]string{
					"type":    "tasks-changed",
					"file":    filepath.Base(evt.Name),
					"op":      evt.Op.String(),
					"provider": tracker.ProviderBacklogMD,
				})
				select {
				case client.events <- string(payload):
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
			if _, err := w.Write([]byte("event: tasks-changed\n")); err != nil {
				return
			}
			if _, err := w.Write([]byte("data: " + msg + "\n\n")); err != nil {
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
