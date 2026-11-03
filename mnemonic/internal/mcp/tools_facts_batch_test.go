// operations array on the MCP fact_add tool (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-04).
//
// RED notes: these tests call handleFactAdd with an "operations" argument the
// registered tool does not declare yet. Before the change the handler ignores
// it (content is required), so the batch assertions fail: the free+add batch
// is not applied as a unit, and the overflow batch does not return the
// structured current_entries + usage error.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestFactAddOperationsBatchMCP (BDD: "happy path — batch frees space and
// adds new fact", MCP layer): fact_add(scope=global, target=memory) with an
// operations array frees 200 chars (remove) and adds 250 chars (add) at
// 2100/2200 → success, final usage 2150/2200.
func TestFactAddOperationsBatchMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-batch"
	seedFactMCPSession(t, st, project, dataDir, sid)

	// Seed the global store to exactly 2100/2200: filler (1900) + stale (200).
	// "stale entry" is a UNIQUE substring — the other seed uses a different
	// prefix (substring matching, TICKET-04).
	const stale = "stale entry" + " " + "x" // 13 chars; padded below to 200
	stale200 := stale + strings.Repeat("x", 200-len(stale))
	if len(stale200) != 200 {
		t.Fatalf("fixture: stale200 len %d, want 200", len(stale200))
	}
	filler := padMCP(1900, "filler")
	if _, err := seedGlobalFactMCP(t, project, sid, filler); err != nil {
		t.Fatal(err)
	}
	if _, err := seedGlobalFactMCP(t, project, sid, stale200); err != nil {
		t.Fatal(err)
	}

	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": sid,
		"scope":      "global",
		"target":     "memory",
		"operations": []any{
			map[string]any{"action": "remove", "old_text": "stale entry"},
			map[string]any{"action": "add", "content": padMCP(250, "fresh")},
		},
	}))
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if res.IsError {
		t.Fatalf("batch fact_add error: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if got := out["success"]; got != true {
		t.Errorf("batch result success = %v, want true (%s)", got, text)
	}
	if got, _ := out["usage"].(string); got != "2,150/2,200" {
		t.Errorf("batch result usage = %q, want \"2,150/2,200\"", got)
	}

	// The stale fact is soft-deleted, the fresh one is live, in the GLOBAL
	// store — final usage 2150/2200.
	gst, err := openGlobalStoreMCP(t, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	var total int
	rows, err := gst.DB.Query(`SELECT content FROM facts WHERE deleted_at IS NULL`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		total += len(c)
	}
	rows.Close()
	if total != 2150 {
		t.Errorf("final char usage = %d, want 2150", total)
	}
}

// TestFactAddOperationsTrailMCP (review finding, T04): a successful batch
// writes a session_events trail row (action_type=fact_add) to the caller's
// PROJECT store — the same audit-trail role as a single-fact AddWith — with
// a payload summarizing the batch ({"batch": true, "ops": N}). A failed
// batch (overflow) writes no trail: nothing was applied, nothing to audit.
func TestFactAddOperationsTrailMCP(t *testing.T) {
	t.Run("success writes fact_add trail to project store", func(t *testing.T) {
		dataDir, project, st := openFactMCPStore(t)
		sid := "sess-mcp-batch-trail"
		seedFactMCPSession(t, st, project, dataDir, sid)

		res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
			"project":    project,
			"session_id": sid,
			"scope":      "global",
			"target":     "memory",
			"operations": []any{
				map[string]any{"action": "add", "content": "first batch entry"},
				map[string]any{"action": "add", "content": "second batch entry"},
			},
		}))
		if err != nil {
			t.Fatalf("handleFactAdd: %v", err)
		}
		if res.IsError {
			t.Fatalf("batch fact_add error: %s", callResultText(t, res))
		}

		var n int
		if err := st.DB.QueryRow(`
			SELECT COUNT(*) FROM session_events
			WHERE session_id = ? AND action_type = 'fact_add'`, sid).
			Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("fact_add trail rows for session = %d, want 1", n)
		}
		var projectCol, payload string
		if err := st.DB.QueryRow(`
			SELECT project, payload FROM session_events
			WHERE session_id = ? AND action_type = 'fact_add'`, sid).
			Scan(&projectCol, &payload); err != nil {
			t.Fatal(err)
		}
		if projectCol != project {
			t.Errorf("trail row project = %q, want %q", projectCol, project)
		}
		var p map[string]any
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			t.Fatalf("parse trail payload %q: %v", payload, err)
		}
		if p["batch"] != true {
			t.Errorf("trail payload batch = %v, want true", p["batch"])
		}
		if p["ops"] != float64(2) {
			t.Errorf("trail payload ops = %v, want 2", p["ops"])
		}
	})

	t.Run("failed batch writes no trail", func(t *testing.T) {
		dataDir, project, st := openFactMCPStore(t)
		sid := "sess-mcp-batch-trail-overflow"
		seedFactMCPSession(t, st, project, dataDir, sid)

		// Seed the global store to the full 2200 budget. The seed goes
		// straight through the facts API (not the fact_add handler) so it
		// trails NO rows — the assertion below is an absolute count.
		gst, err := openGlobalStoreMCP(t, dataDir)
		if err != nil {
			t.Fatal(err)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := gst.DB.Exec(
			`INSERT INTO facts (content, category, created_at, updated_at)
			 VALUES (?, 'memory', ?, ?)`,
			padMCP(2200, "fill"), now, now); err != nil {
			t.Fatal(err)
		}
		res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
			"project":    project,
			"session_id": sid,
			"scope":      "global",
			"target":     "memory",
			"operations": []any{
				map[string]any{"action": "add", "content": "overflow entry"},
			},
		}))
		if err != nil {
			t.Fatalf("handleFactAdd: %v", err)
		}
		if !res.IsError {
			t.Fatalf("overflow batch must be an error result: %s", callResultText(t, res))
		}
		var n int
		if err := st.DB.QueryRow(`
			SELECT COUNT(*) FROM session_events
			WHERE session_id = ? AND action_type = 'fact_add'`, sid).
			Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("fact_add trail rows after overflow = %d, want 0 (no new trail)", n)
		}
	})
}

// TestFactAddOperationsOverflowMCP (BDD: "edge — overflow returns structured
// error with current entries", MCP layer): a batch that overflows at
// 2150/2200 returns a structured error carrying current_entries and the
// usage string — the tool result is an error naming them (not a bare "char
// limit" message), and no fact row is created.
func TestFactAddOperationsOverflowMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-batch-overflow"
	seedFactMCPSession(t, st, project, dataDir, sid)

	base := padMCP(2150, "base")
	if _, err := seedGlobalFactMCP(t, project, sid, base); err != nil {
		t.Fatal(err)
	}

	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": sid,
		"scope":      "global",
		"target":     "memory",
		"operations": []any{
			map[string]any{"action": "add", "content": padMCP(100, "new")},
		},
	}))
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if !res.IsError {
		t.Fatalf("overflow batch must be an error result: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	if !strings.Contains(text, "2,150/2,200") {
		t.Errorf("error %q does not carry the usage string", text)
	}
	if !strings.Contains(text, "current_entries") {
		t.Errorf("error %q does not name current_entries", text)
	}
	if !strings.Contains(text, "base") {
		// padMCP(2150, "base") starts with "base" — the error string
		// carries the full current entry content (no truncation).
		t.Errorf("error does not carry the current entry content (got %d chars)", len(text))
	}

	gst, err := openGlobalStoreMCP(t, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("live facts after overflow = %d, want 1 (no ops applied)", n)
	}
}

// TestFactAddOperationsMalformedMCP: a PRESENT but malformed operations
// argument (an object, not an array) is a clean error at the batch boundary —
// not a silent fall-through to the content path.
func TestFactAddOperationsMalformedMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-batch-malformed"
	seedFactMCPSession(t, st, project, dataDir, sid)

	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": sid,
		"scope":      "global",
		"target":     "memory",
		"content":    "a fact",
		"operations": map[string]any{"action": "add"},
	}))
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if !res.IsError {
		t.Fatalf("malformed operations must error: %s", callResultText(t, res))
	}
	if !strings.Contains(callResultText(t, res), "operations") {
		t.Errorf("error %q does not name the operations argument", callResultText(t, res))
	}
}

// padMCP builds a deterministic n-byte content string (the MCP fixtures need
// exact char counts for the 2200 budget). The result ALWAYS starts with the
// prefix verbatim (no truncation) so a substring match on the prefix works.
func padMCP(n int, prefix string) string {
	if len(prefix) >= n {
		return prefix[:n]
	}
	return prefix + strings.Repeat("x", n-len(prefix))
}

// seedGlobalFactMCP adds one global-scope fact through the real fact_add
// handler and returns its id.
func seedGlobalFactMCP(t *testing.T, project, sid, content string) (int64, error) {
	t.Helper()
	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": sid,
		"scope":      "global",
		"target":     "memory",
		"content":    content,
	}))
	if err != nil {
		return 0, err
	}
	if res.IsError {
		return 0, errors.New("seed fact_add: " + callResultText(t, res))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		return 0, err
	}
	id, _ := out["fact_id"].(float64)
	return int64(id), nil
}

// openGlobalStoreMCP opens the global store in dataDir for assertions.
func openGlobalStoreMCP(t *testing.T, dataDir string) (*store.Store, error) {
	t.Helper()
	gst, err := store.Open(dataDir, "global")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { gst.Close() })
	return gst, nil
}
