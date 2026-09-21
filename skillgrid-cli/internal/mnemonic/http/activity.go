package http

import (
	"encoding/json"
	"net/http"
	"time"
)

// activityEvent is one entry in the /activity feed. There is no dedicated
// events table — activity is modeled as observations (agent events) belonging
// to sessions, so this maps the relevant observation columns to the feed shape.
// `actor` is the tool_name (which tool produced the row); `source` is the
// provenance (agent/passive/prompt/compact).
type activityEvent struct {
	ID        int64   `json:"id"`
	TS        string  `json:"ts"`
	Type      string  `json:"type"`
	Source    string  `json:"source"`
	Actor     *string `json:"actor"`
	Severity  string  `json:"severity"`
	Summary   string  `json:"summary"`
	TopicKey  *string `json:"topicKey"`
	SessionID string  `json:"sessionId"`
	Related   []int64 `json:"relatedIds"`
}

// deriveSeverity maps an observation type to a display severity for the feed's
// severity border. There's no severity column, so this is a type heuristic.
func deriveSeverity(otype string) string {
	switch otype {
	case "bugfix", "bug":
		return "high"
	case "decision", "architecture":
		return "medium"
	default:
		return "info"
	}
}

// handleMnemonicActivityEvents serves GET /activity/events?limit=N — the
// newest-first activity feed (observations for the project).
func (s *Server) handleMnemonicActivityEvents(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		// A project that exists but whose DB handle cannot be opened is a
		// server fault, not a 404 — matches mnemonic_graph.go / server.go
		// (review B1: this was StatusNotFound, the others 500).
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()

	limit := queryInt(r, "limit", 100)
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}

	rows, err := h.Store().DB.QueryContext(r.Context(), `
		SELECT o.id, o.created_at, o.type, COALESCE(o.source,'agent'), o.tool_name,
		       o.title, o.topic_key, o.session_id
		FROM observations o
		WHERE o.project = ? AND o.deleted_at IS NULL
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT ?`, projectID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	events := []activityEvent{}
	for rows.Next() {
		var e activityEvent
		var otype string
		if err := rows.Scan(&e.ID, &e.TS, &otype, &e.Source, &e.Actor, &e.Summary, &e.TopicKey, &e.SessionID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		e.Type = otype
		e.Severity = deriveSeverity(otype)
		e.Related = []int64{}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project": projectID,
		"events":  events,
		"limit":   limit,
	})
}

// handleMnemonicActivityStats serves GET /activity/stats — counters for the
// stats bar: total events, per-type breakdown, and active session count.
func (s *Server) handleMnemonicActivityStats(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	db := h.Store().DB

	var total int
	if err := db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL`, projectID,
	).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	byType := map[string]int{}
	rows, err := db.QueryContext(r.Context(),
		`SELECT type, COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL GROUP BY type`, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		byType[typ] = n
	}
	rows.Close()

	var activeSessions int
	if err := db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM sessions WHERE project = ? AND status = 'active'`, projectID,
	).Scan(&activeSessions); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"project":        projectID,
		"total":          total,
		"byType":         byType,
		"activeSessions": activeSessions,
	})
}

// handleMnemonicActivityStream serves GET /activity/stream (SSE). It polls the
// observations table for new rows and emits an `event: activity` SSE message
// for each, so the Activity feed live-updates. Copy of the tracker-stream
// pattern (server_tracker_stream.go) but the producer is a DB poller instead of
// an fsnotify watcher. Leak-free: the poller goroutine exits on ctx.Done().
func (s *Server) handleMnemonicActivityStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		// Before the SSE headers are written, so this is a clean JSON error
		// (not a body into text/event-stream). 500, not 404 (review B1).
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Initial event so the client knows the stream is alive.
	_, _ = w.Write([]byte("event: ready\n\n"))
	flusher.Flush()

	ctx := r.Context()
	client := &activityStreamClient{events: make(chan streamEvent, 64)}

	// Poller: track the newest observation id AND the newest change_snapshot
	// id (Handoff Hub, change 015). Each tick emits rows newer than the mark,
	// tagged with the SSE event type (activity | snapshot).
	go func() {
		db := h.Store().DB
		// Seed the high-water marks with the current newest ids (no replay of
		// history on connect — the client already has it from /events and
		// /activity/snapshots). If the observations seed errors, treat the
		// stream as closed rather than replaying the whole table from id>0.
		// The snapshot seed is best-effort (the table may predate the hub).
		var newest int64
		if err := db.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(id), 0) FROM observations WHERE project = ? AND deleted_at IS NULL`, projectID,
		).Scan(&newest); err != nil {
			return
		}
		var newestSnap int64
		_ = db.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(id), 0) FROM change_snapshots WHERE project = ?`, projectID,
		).Scan(&newestSnap)

		ticker := time.NewTicker(800 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Activity (observations) rows.
				rows, err := db.QueryContext(ctx, `
					SELECT id, created_at, type, COALESCE(source,'agent'), tool_name, title, session_id
					FROM observations
					WHERE project = ? AND deleted_at IS NULL AND id > ?
					ORDER BY id ASC
					LIMIT 100`, projectID, newest)
				if err != nil {
					return
				}
				type evRow struct {
					id       int64
					ts       string
					otype    string
					source   string
					toolName *string
					title    string
					session  string
				}
				batch := []evRow{}
				for rows.Next() {
					var er evRow
					if err := rows.Scan(&er.id, &er.ts, &er.otype, &er.source, &er.toolName, &er.title, &er.session); err != nil {
						rows.Close()
						return
					}
					if er.id > newest {
						newest = er.id
					}
					batch = append(batch, er)
				}
				rows.Close()
				for _, er := range batch {
					payload, _ := json.Marshal(map[string]any{
						"id":        er.id,
						"ts":        er.ts,
						"type":      er.otype,
						"source":    er.source,
						"actor":     er.toolName,
						"severity":  deriveSeverity(er.otype),
						"summary":   er.title,
						"sessionId": er.session,
					})
					select {
					case client.events <- streamEvent{kind: "activity", data: string(payload)}:
					default: // slow consumer: drop, never block
					}
				}

				// Change snapshots (Handoff Hub). Best-effort: an error here
				// (e.g. the table is absent on a very old store) just skips
				// this tick rather than killing the activity stream.
				if srows, serr := db.QueryContext(ctx, `
					SELECT id, branch, "commit", COALESCE(commit_short,''), COALESCE(subject,''),
					       COALESCE(author,''), committed_at
					FROM change_snapshots
					WHERE project = ? AND id > ?
					ORDER BY id ASC
					LIMIT 100`, projectID, newestSnap); serr == nil {
					type snapRow struct {
						id     int64
						branch string
						commit string
						short  string
						subj   string
						author string
						at     string
					}
					var sbatch []snapRow
					for srows.Next() {
						var sr snapRow
						if err := srows.Scan(&sr.id, &sr.branch, &sr.commit, &sr.short, &sr.subj, &sr.author, &sr.at); err != nil {
							break
						}
						if sr.id > newestSnap {
							newestSnap = sr.id
						}
						sbatch = append(sbatch, sr)
					}
					srows.Close()
					for _, sr := range sbatch {
						payload, _ := json.Marshal(map[string]any{
							"commit":       sr.commit,
							"commitShort":  sr.short,
							"subject":      sr.subj,
							"branch":       sr.branch,
							"author":       sr.author,
							"committedAt":  sr.at,
						})
						select {
						case client.events <- streamEvent{kind: "snapshot", data: string(payload)}:
						default:
						}
					}
				}
			}
		}
	}()

	// Write loop: client channel → SSE. Exits on context cancel.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-client.events:
			if _, err := w.Write([]byte("event: " + ev.kind + "\n")); err != nil {
				return
			}
			if _, err := w.Write([]byte("data: " + ev.data + "\n\n")); err != nil {
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

// activityStreamClient is one subscribed SSE client; the buffered channel lets
// a slow consumer drop events without blocking the poller.
type activityStreamClient struct {
	events chan streamEvent
}

// streamEvent is one SSE frame: the event kind (activity | snapshot) plus the
// JSON data payload. The kind maps to the SSE `event:` line so clients can
// multiplex on event type.
type streamEvent struct {
	kind string
	data string
}
