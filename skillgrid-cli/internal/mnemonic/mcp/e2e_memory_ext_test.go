package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type rpcReq struct {
	JRPC   string `json:"jsonrpc"`
	ID     any    `json:"id,omitempty"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type rpcRes struct {
	JRPC   string          `json:"jsonrpc"`
	ID     json.RawMessage `json:"id"`
	Err    *rpcError       `json:"error"`
	Result json.RawMessage `json:"result"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (r rpcRes) isError() bool { return r.Err != nil }

// spawnMCPServer starts the skillgrid binary's `mcp` subcommand against the
// given temp data dir and returns a handle that speaks MCP JSON-RPC over
// stdio. It is deliberately simple — no SDK, no pipe-pair acrobatics — so
// the test exercises the *same* wire path a real agent does.
type mcpHandle struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	scanner *bufio.Scanner
	dataDir string
}

func startMCP(t *testing.T, cwd string) *mcpHandle {
	t.Helper()
	bin, err := buildBinary(t)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	dataDir := t.TempDir()

	cmd := exec.Command(bin, "mcp")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"SKILLGRID_MNEMONIC_DATA_DIR="+dataDir,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout: %v", err)
	}
	cmd.Stderr = os.Stderr // surface framing errors when -v
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	h := &mcpHandle{cmd: cmd, stdin: stdin, scanner: scanner, dataDir: dataDir}
	t.Cleanup(func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	return h
}

func buildBinary(t *testing.T) (string, error) {
	t.Helper()
	// Walk up to the skillgrid-cli/ root, then build ./cmd/skillgrid from
	// there.
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 8 && !strings.HasSuffix(dir, "skillgrid-cli"); i++ {
		dir = filepath.Dir(dir)
	}
	if !strings.HasSuffix(dir, "skillgrid-cli") {
		return "", fmt.Errorf("skillgrid-cli root not found from %s", os.Getenv("PWD"))
	}
	bin := filepath.Join(t.TempDir(), "skillgrid-bin-"+t.Name())
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/skillgrid")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build: %v\n%s", err, out)
	}
	return bin, nil
}

// Send writes one JSON-RPC request and (for id'd requests) waits for the
// response with the matching id.
func (h *mcpHandle) Send(haveID bool, id any, method string, params any) (*rpcRes, error) {
	req := rpcReq{JRPC: "2.0", ID: id, Method: method, Params: params}
	line, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := h.stdin.Write(append(line, '\n')); err != nil {
		return nil, err
	}
	if !haveID {
		return nil, nil
	}
	wantID, _ := json.Marshal(id)
	for {
		if !h.scanner.Scan() {
			if err := h.scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
				return nil, err
			}
			return nil, io.EOF
		}
		raw := h.scanner.Bytes()
		var res rpcRes
		if err := json.Unmarshal(raw, &res); err != nil {
			return nil, fmt.Errorf("unmarshal response: %v\n%s", err, raw)
		}
		if string(res.ID) == string(wantID) {
			return &res, nil
		}
	}
}

func initSession(t *testing.T, h *mcpHandle, cwd string) string {
	t.Helper()
	init, err := h.Send(true, 1, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "e2e", "version": "0"},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if init.isError() {
		t.Fatalf("initialize error: %s", init.Err.Message)
	}
	if _, err := h.Send(false, json.RawMessage("null"), "notifications/initialized", nil); err != nil {
		t.Fatalf("notifications/initialized: %v", err)
	}

	res, err := h.Send(true, 2, "tools/call", map[string]any{
		"name": "mem_session_start",
		"arguments": map[string]any{
			"directory": cwd,
			"title":     "evt-e2e",
		},
	})
	if err != nil {
		t.Fatalf("session_start rpc: %v", err)
	}
	if res.isError() {
		t.Fatalf("session_start error: %s", res.Err.Message)
	}
	var payload struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(res.Result, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var out struct {
		SessionID string `json:"session_id"`
	}
	if len(payload.Content) == 0 {
		t.Fatalf("empty content")
	}
	if err := json.Unmarshal([]byte(payload.Content[0].Text), &out); err != nil {
		t.Fatalf("unmarshal inner: %v", err)
	}
	return out.SessionID
}

func callTool(t *testing.T, h *mcpHandle, id int, name string, args map[string]any) *rpcRes {
	t.Helper()
	res, err := h.Send(true, id, "tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.isError() {
		t.Fatalf("%s error: %v", name, res.Err)
	}
	return res
}

func toolText(t *testing.T, res *rpcRes) string {
	t.Helper()
	var payload struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(res.Result, &payload); err != nil {
		t.Fatalf("tools/call result: %v\n%s", err, res.Result)
	}
	if len(payload.Content) == 0 {
		t.Fatalf("empty tools/call content: %s", res.Result)
	}
	return payload.Content[0].Text
}

func TestE2EMemoryExtTools(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	h := startMCP(t, cwd)
	sid := initSession(t, h, cwd)

	// anchor + later entries. Sleep 1.1s between saves so the timestamps are
	// strictly ordered (SQLite uses `>` on created_at, which is second-precision).
	anchor := callTool(t, h, 10, "mem_save", map[string]any{
		"title": "evt anchor decision", "type": "decision",
		"content": "**What** anchor for timeline **Why** e2e **Where** n/a **Learned** —",
		"session_id": sid,
	})
	anchorID := extractID(t, anchor)
	time.Sleep(1100 * time.Millisecond)
	_ = callTool(t, h, 11, "mem_save", map[string]any{
		"title": "evt later topic", "type": "pattern",
		"content": "**What** second entry **Why** e2e **Where** n/a **Learned** —",
		"session_id": sid,
	})
	time.Sleep(1100 * time.Millisecond)
	thirdRes := callTool(t, h, 11, "mem_save", map[string]any{
		"title": "evt much later", "type": "learning",
		"content": "**What** third entry **Why** e2e **Where** n/a **Learned** —",
		"session_id": sid,
	})
	thirdID := extractID(t, thirdRes)

	// mem_save_prompt
	promptRes := callTool(t, h, 12, "mem_save_prompt", map[string]any{
		"content":    "please implement the missing memory tools per the engram architecture doc",
		"session_id": sid,
	})
	if text := toolText(t, promptRes); strings.Contains(text, "error") {
		t.Fatalf("mem_save_prompt reported: %s", text)
	}

	// mem_stats (should show >=2 observations)
	stats := callTool(t, h, 13, "mem_stats", nil)
	sres := parseToolJSON(t, stats)
	obsCount, _ := sres["observation_count"].(float64)
	if obsCount < 2 {
		t.Fatalf("expected >=2 observations, got %v", obsCount)
	}

	// mem_current_project
	cpRes := callTool(t, h, 14, "mem_current_project", nil)
	cp := parseToolJSON(t, cpRes)
	if cp["project"] == nil || cp["project"] == "" {
		t.Fatalf("expected non-empty project: %s", cpRes.Result)
	}
	if cp["source"] == nil {
		t.Fatalf("expected source: %s", cpRes.Result)
	}
	projects, _ := cp["projects"].([]any)
	if len(projects) == 0 {
		t.Fatalf("expected at least the current project in projects list: %s", cpRes.Result)
	}

	// mem_doctor
	doc := parseToolJSON(t, callTool(t, h, 15, "mem_doctor", nil))
	schema, _ := doc["schema_version"].(float64)
	if schema < 5 {
		t.Fatalf("expected schema_version >= 5 (including review_cycle), got %v", schema)
	}
	ok, _ := doc["fts_integrity_ok"].(bool)
	if !ok {
		t.Fatalf("expected FTS integrity OK: %s", doc)
	}

	// mem_review list (nothing due yet — all reviews have review_after NULL)
	rev := parseToolJSON(t, callTool(t, h, 16, "mem_review", map[string]any{"action": "list"}))
	count, _ := rev["count"].(float64)
	if count != 0 {
		t.Fatalf("expected no initially-due reviews, got %v: %s", count, rev)
	}

	// mem_review mark_reviewed — advances review_after into the future
	mr := parseToolJSON(t, callTool(t, h, 16, "mem_review", map[string]any{
		"action": "mark_reviewed", "id": anchorID,
	}))
	if mr["marked_reviewed"] != true {
		t.Fatalf("expected marked_reviewed=true: %s", mr)
	}
	ra, _ := mr["review_after"].(string)
	if ra == "" {
		t.Fatalf("expected review_after timestamp: %s", mr)
	}

	// mem_update (change content of anchor)
	upRes := callTool(t, h, 17, "mem_update", map[string]any{
		"id":      anchorID,
		"content": "**What** updated anchor content **Why** e2e **Where** n/a **Learned** —",
	})
	if up := parseToolJSON(t, upRes); up["updated"] != true {
		t.Fatalf("expected updated=true: %s", up)
	}

	// Search finds the updated content
	sr := parseToolJSON(t, callTool(t, h, 18, "mem_search", map[string]any{"query": "updated anchor content"}))
	obs, _ := sr["observations"].([]any)
	if len(obs) == 0 {
		t.Fatalf("expected the updated entry to be findable, got 0: %s", sr)
	}

	// mem_timeline (window 1h)
	tl := parseToolJSON(t, callTool(t, h, 19, "mem_timeline", map[string]any{
		"id": anchorID, "window": "1h", "limit": 5,
	}))
	f, ok := tl["anchor_id"].(float64)
	if !ok || f != float64(anchorID) {
		t.Fatalf("expected anchor_id %d, got %#v", anchorID, tl["anchor_id"])
	}
	before, _ := tl["before"].([]any)
	after, _ := tl["after"].([]any)
	if len(before)+len(after) < 1 {
		t.Fatalf("expected at least one entry in timeline (before or after): %s", tl)
	}

	// mem_judge — record a conflicts_with verdict between anchor and the third
	// entry, then confirm a second judge is an upsert (same id), then clear it
	// with not_conflict.
	j1 := parseToolJSON(t, callTool(t, h, 22, "mem_judge", map[string]any{
		"src_id": anchorID, "dst_id": thirdID, "verdict": "conflicts_with",
		"confidence": 0.8, "reason": "e2e",
	}))
	jid, _ := j1["id"].(float64)
	if jid <= 0 {
		t.Fatalf("expected a relation id from mem_judge: %s", j1)
	}
	if j1["verdict"] == "not_conflict" {
		t.Fatalf("mem_judge should record conflicts_with, not clear: %s", j1)
	}

	// Re-judge the same pair — upsert keeps the same relation id.
	j2 := parseToolJSON(t, callTool(t, h, 23, "mem_judge", map[string]any{
		"src_id": anchorID, "dst_id": thirdID, "verdict": "conflicts_with",
		"reason": "e2e revised",
	}))
	if j2id, _ := j2["id"].(float64); j2id != jid {
		t.Fatalf("expected upsert to keep relation id %v, got %v: %s", jid, j2id, j2)
	}

	// mem_compare — persist a compatible relation in the other direction.
	_ = parseToolJSON(t, callTool(t, h, 24, "mem_compare", map[string]any{
		"src_id": thirdID, "dst_id": anchorID, "relation": "compatible",
	}))

	// not_conflict — clears the conflicts_with link, returns removed=true.
	nc := parseToolJSON(t, callTool(t, h, 25, "mem_judge", map[string]any{
		"src_id": anchorID, "dst_id": thirdID, "verdict": "not_conflict",
	}))
	if nc["removed"] != true {
		t.Fatalf("expected not_conflict to report removed=true: %s", nc)
	}

	// mem_merge_projects — merge the live project into itself (no-op) returns a
	// clean shape, and an identical source/canonical is a no-op.
	projOut := parseToolJSON(t, callTool(t, h, 26, "mem_current_project", nil))
	canon, _ := projOut["project"].(string)
	mergeNoop := parseToolJSON(t, callTool(t, h, 27, "mem_merge_projects", map[string]any{
		"source": canon, "canonical": canon,
	}))
	if mergeNoop["merged"] != false {
		t.Fatalf("expected merged=false for identical source/canonical: %s", mergeNoop)
	}
	// Merge a non-existent source into canonical: alias recorded, 0 rows moved.
	mergeNew := parseToolJSON(t, callTool(t, h, 28, "mem_merge_projects", map[string]any{
		"source": "evt-legacy-alias-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		"canonical": canon,
	}))
	if mergeNew["canonical"] != canon {
		t.Fatalf("expected canonical echoed back: %s", mergeNew)
	}
	if _, ok := mergeNew["rows_moved"]; !ok {
		t.Fatalf("expected rows_moved field: %s", mergeNew)
	}

	// mem_delete (soft)
	del := parseToolJSON(t, callTool(t, h, 20, "mem_delete", map[string]any{"id": anchorID}))
	if del["deleted"] != true || del["hard"] == true {
		t.Fatalf("expected soft delete to succeed: %s", del)
	}

	// After soft-delete, search should not find it anymore.
	srAfter := parseToolJSON(t, callTool(t, h, 21, "mem_search", map[string]any{"query": "updated anchor content"}))
	if obsAfter, _ := srAfter["observations"].([]any); len(obsAfter) != 0 {
		t.Fatalf("expected 0 hits after soft-delete, got %d: %s", len(obsAfter), srAfter)
	}
}

func extractID(t *testing.T, res *rpcRes) int64 {
	t.Helper()
	out := parseToolJSON(t, res)
	id, ok := out["id"].(float64)
	if !ok {
		t.Fatalf("expected int id in %s", out)
	}
	return int64(id)
}

func parseToolJSON(t *testing.T, res *rpcRes) map[string]any {
	t.Helper()
	text := toolText(t, res)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("tool JSON: %v\n%s", err, text)
	}
	return out
}
